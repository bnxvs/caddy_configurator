package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"caddy-configurator/internal/caddy"
	"caddy-configurator/internal/config"
	"caddy-configurator/internal/constants"
	"caddy-configurator/internal/project"
	"caddy-configurator/internal/render"
)

// APIVersion is the versioned-in-path API prefix.
const APIVersion = "/api/v1"

// Server is a single-project (P1) API + frontend server.
type Server struct {
	root     string
	mux      *http.ServeMux
	mu       sync.Mutex
	snapshot map[string]string // rel source path -> sha256 hex
}

// New loads the project at root, snapshots source state, and wires routes.
// devFrontend proxies frontend assets to a vite dev server when non-empty.
func New(root, devFrontend string) (*Server, error) {
	if _, err := config.LoadProject(root); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	frontend, err := NewFrontendHandler(devFrontend)
	if err != nil {
		return nil, err
	}
	s := &Server{root: abs, mux: http.NewServeMux(), snapshot: map[string]string{}}
	s.routes(frontend)
	s.refreshSnapshotLocked()
	return s, nil
}

// Handler returns the full HTTP handler (API + frontend fallback).
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes(frontend http.Handler) {
	s.mux.HandleFunc("GET "+APIVersion+"/project", s.handleProject)
	s.mux.HandleFunc("PUT "+APIVersion+"/project", s.handlePutProject)
	s.mux.HandleFunc("GET "+APIVersion+"/sites", s.handleSites)
	s.mux.HandleFunc("POST "+APIVersion+"/sites", s.handleCreateSite)
	s.mux.HandleFunc("GET "+APIVersion+"/sites/{id}", s.handleGetSite)
	s.mux.HandleFunc("GET "+APIVersion+"/sites/{id}/snippet", s.handleSnippet)
	s.mux.HandleFunc("PUT "+APIVersion+"/sites/{id}", s.handlePutSite)
	s.mux.HandleFunc("DELETE "+APIVersion+"/sites/{id}", s.handleDeleteSite)
	s.mux.HandleFunc("GET "+APIVersion+"/preview", s.handlePreview)
	s.mux.HandleFunc("POST "+APIVersion+"/generate", s.handleGenerate)
	s.mux.HandleFunc("GET "+APIVersion+"/validate", s.handleValidate)
	s.mux.HandleFunc("GET "+APIVersion+"/status", s.handleStatus)
	s.mux.HandleFunc("POST "+APIVersion+"/status/refresh", s.handleRefresh)
	s.mux.Handle("/", frontend)
}

// ResolveAddr returns the listen address, defaulting to loopback.
func ResolveAddr(bind string, port int) string {
	if bind == "" {
		bind = constants.DefaultBind
	}
	if port == 0 {
		port = constants.DefaultPort
	}
	return bind + ":" + strconv.Itoa(port)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErrors(w http.ResponseWriter, errs config.ValidationErrors) {
	writeJSON(w, http.StatusBadRequest, map[string]any{"errors": errs})
}

func (s *Server) load() (*config.Project, error) {
	return config.LoadProject(s.root)
}

type siteSummary struct {
	ID      string   `json:"id"`
	Domains []string `json:"domains"`
	Preset  string   `json:"preset"`
}

func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	p, err := s.load()
	if err != nil {
		writeErrors(w, toFieldErrors(err))
		return
	}
	summaries := make([]siteSummary, 0, len(p.Sites))
	for _, site := range p.Sites {
		summaries = append(summaries, siteSummary{ID: site.ID, Domains: site.Domains, Preset: site.Preset})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name": p.Meta.Name, "root": p.Root, "version": p.Meta.Version,
		"email": p.Meta.Global.Email, "sites": summaries,
		"caddyFound": caddy.Found(),
	})
}

func (s *Server) handlePutProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  *string `json:"name"`
		Email *string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON: " + err.Error()})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		writeErrors(w, toFieldErrors(err))
		return
	}
	meta := p.Meta
	if body.Name != nil {
		meta.Name = *body.Name
	}
	if body.Email != nil {
		meta.Global.Email = *body.Email
	}
	if err := config.SaveProjectMeta(s.root, meta); err != nil {
		writeErrors(w, toFieldErrors(err))
		return
	}
	s.refreshSnapshotLocked()
	writeJSON(w, http.StatusOK, map[string]any{
		"name": meta.Name, "email": meta.Global.Email, "version": meta.Version,
	})
}

func (s *Server) handleSnippet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !config.ValidSiteID(id) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("invalid site id %q", id)})
		return
	}
	p, err := s.load()
	if err != nil {
		writeErrors(w, toFieldErrors(err))
		return
	}
	for _, site := range p.Sites {
		if site.ID == id {
			writeJSON(w, http.StatusOK, map[string]any{"snippet": render.RenderSiteSnippet(site)})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]any{"error": "site not found"})
}

func (s *Server) handleSites(w http.ResponseWriter, r *http.Request) {
	p, err := s.load()
	if err != nil {
		writeErrors(w, toFieldErrors(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sites": p.Sites})
}

func (s *Server) handleGetSite(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !config.ValidSiteID(id) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("invalid site id %q", id)})
		return
	}
	p, err := s.load()
	if err != nil {
		writeErrors(w, toFieldErrors(err))
		return
	}
	for _, site := range p.Sites {
		if site.ID == id {
			writeJSON(w, http.StatusOK, site)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]any{"error": "site not found"})
}

func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	var site config.Site
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON: " + err.Error()})
		return
	}
	if !config.ValidSiteID(site.ID) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("invalid site id %q", site.ID)})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Stat(filepath.Join(s.root, constants.SitesDir, site.ID+constants.SiteFileSuffix)); err == nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "site already exists"})
		return
	}
	file := filepath.Join(constants.SitesDir, site.ID+constants.SiteFileSuffix)
	if errs := config.ValidateSite(file, site); len(errs) > 0 {
		writeErrors(w, errs)
		return
	}
	if err := config.SaveSite(s.root, site); err != nil {
		writeErrors(w, toFieldErrors(err))
		return
	}
	s.refreshSnapshotLocked()
	writeJSON(w, http.StatusCreated, site)
}

func (s *Server) handlePutSite(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !config.ValidSiteID(id) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("invalid site id %q", id)})
		return
	}
	var site config.Site
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON: " + err.Error()})
		return
	}
	site.ID = id // path wins; body id (if any) is ignored
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Stat(filepath.Join(s.root, constants.SitesDir, id+constants.SiteFileSuffix)); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "site not found"})
		return
	}
	file := filepath.Join(constants.SitesDir, id+constants.SiteFileSuffix)
	if errs := config.ValidateSite(file, site); len(errs) > 0 {
		writeErrors(w, errs)
		return
	}
	if err := config.SaveSite(s.root, site); err != nil {
		writeErrors(w, toFieldErrors(err))
		return
	}
	s.refreshSnapshotLocked()
	writeJSON(w, http.StatusOK, site)
}

func (s *Server) handleDeleteSite(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !config.ValidSiteID(id) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("invalid site id %q", id)})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := config.DeleteSite(s.root, id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "site not found"})
		return
	}
	s.refreshSnapshotLocked()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	pv, err := project.PreviewProject(s.root)
	if err != nil {
		writeErrors(w, toFieldErrors(err))
		return
	}
	formatted := false
	pv.Caddyfile, formatted = caddy.FormatCaddyfile(pv.Caddyfile)
	if pv.Warnings == nil {
		pv.Warnings = []config.Warning{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"caddyfile": pv.Caddyfile, "compose": pv.Compose,
		"formatted": formatted, "warnings": pv.Warnings,
	})
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := project.GenerateProject(s.root)
	if err != nil {
		writeErrors(w, toFieldErrors(err))
		return
	}
	s.refreshSnapshotLocked()
	writeJSON(w, http.StatusOK, map[string]any{
		"caddyfilePath": res.CaddyfilePath, "composePath": res.ComposePath,
		"formatted": res.Formatted, "warnings": nonNilWarnings(res.Warnings),
	})
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	p, err := s.load()
	if err != nil {
		writeErrors(w, toFieldErrors(err))
		return
	}
	caddyOut := caddy.ValidateCaddyfile(render.RenderCaddyfile(p))
	if len(p.Sites) == 0 {
		caddyOut = caddy.ValidateResult{Available: caddy.Found(), OK: true, Output: "no sites to validate"}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"errors": []config.FieldError{},
		"caddy": map[string]any{
			"available": caddyOut.Available, "ok": caddyOut.OK, "output": caddyOut.Output,
		},
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"changedOnDisk":  s.changedLocked(),
		"caddyFound":     caddy.Found(),
		"generatedStale": s.generatedStaleLocked(),
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshSnapshotLocked()
	writeJSON(w, http.StatusOK, map[string]any{"changedOnDisk": []string{}})
}

// snapshot helpers (caller holds s.mu).

func (s *Server) sourceFilesLocked() []string {
	var out []string
	out = append(out, constants.ProjectFileName)
	entries, err := os.ReadDir(filepath.Join(s.root, constants.SitesDir))
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, filepath.Join(constants.SitesDir, e.Name()))
		}
	}
	return out
}

func hashFile(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), true
}

func (s *Server) refreshSnapshotLocked() {
	s.snapshot = map[string]string{}
	for _, rel := range s.sourceFilesLocked() {
		if h, ok := hashFile(filepath.Join(s.root, filepath.FromSlash(rel))); ok {
			s.snapshot[rel] = h
		}
	}
}

func (s *Server) changedLocked() []string {
	out := []string{}
	seen := map[string]bool{}
	for _, rel := range s.sourceFilesLocked() {
		seen[rel] = true
		h, ok := hashFile(filepath.Join(s.root, filepath.FromSlash(rel)))
		if !ok || s.snapshot[rel] != h {
			out = append(out, rel)
		}
	}
	for rel := range s.snapshot {
		if !seen[rel] {
			out = append(out, rel) // deleted on disk
		}
	}
	sort.Strings(out)
	return out
}

func (s *Server) generatedStaleLocked() bool {
	p, err := config.LoadProject(s.root)
	if err != nil {
		return true
	}
	caddyfile := render.RenderCaddyfile(p)
	caddyfile, _ = caddy.FormatCaddyfile(caddyfile) // match generate path
	compose := render.RenderCompose(p)
	for rel, want := range map[string]string{
		filepath.Join(constants.GeneratedDir, constants.CaddyfileName):   caddyfile,
		filepath.Join(constants.GeneratedDir, constants.ComposeFileName): compose,
	} {
		got, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(rel)))
		if err != nil || string(got) != want {
			return true
		}
	}
	return false
}

func nonNilWarnings(w []config.Warning) []config.Warning {
	if w == nil {
		return []config.Warning{}
	}
	return w
}

func toFieldErrors(err error) config.ValidationErrors {
	if verrs, ok := err.(config.ValidationErrors); ok {
		return verrs
	}
	return config.ValidationErrors{{File: "", Field: "", Reason: err.Error()}}
}
