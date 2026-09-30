package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"caddy-configurator/internal/constants"
)

// Known top-level keys, used to turn unknown future fields into warnings
// instead of hard errors (forward tolerance for schema v2).
var (
	knownProjectKeys = map[string]bool{"version": true, "name": true, "global": true}
	knownGlobalKeys  = map[string]bool{"email": true}
	knownSiteKeys    = map[string]bool{
		"version": true, "domains": true, "preset": true, "tls": true,
		"root": true, "log": true, "backend": true,
	}
	knownBackendKeys = map[string]bool{
		"service": true, "image": true, "internalPort": true, "env": true,
	}
)

// unknownKeys returns sorted keys of m that are not in known.
func unknownKeys(m map[string]any, known map[string]bool) []string {
	var out []string
	for k := range m {
		if !known[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func asMap(data []byte) (map[string]any, error) {
	var m map[string]any
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// LoadProject reads and validates a whole project rooted at root.
func LoadProject(root string) (*Project, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	metaPath := filepath.Join(abs, constants.ProjectFileName)
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("not a configurator project: cannot read %s: %w", constants.ProjectFileName, err)
	}
	meta, metaWarnings, merr := parseProjectMeta(metaPath, metaData)
	if len(merr) > 0 {
		return nil, merr
	}

	sitesDir := filepath.Join(abs, constants.SitesDir)
	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s/: %w", constants.SitesDir, err)
	}

	p := &Project{Root: abs, Meta: meta, Warnings: metaWarnings}
	var errs ValidationErrors
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), constants.SiteFileSuffix) {
			continue
		}
		id := strings.TrimSuffix(e.Name(), constants.SiteFileSuffix)
		rel := filepath.Join(constants.SitesDir, e.Name())
		if !ValidSiteID(id) {
			errs = append(errs, FieldError{File: rel, Field: "id", Reason: fmt.Sprintf("invalid site id %q: use [a-z0-9_.-]", id)})
			continue
		}
		data, err := os.ReadFile(filepath.Join(sitesDir, e.Name()))
		if err != nil {
			errs = append(errs, FieldError{File: rel, Field: "", Reason: "cannot read file: " + err.Error()})
			continue
		}
		site, warnings, verrs := parseSite(id, rel, data)
		p.Warnings = append(p.Warnings, warnings...)
		if len(verrs) > 0 {
			errs = append(errs, verrs...)
			continue
		}
		p.Sites = append(p.Sites, site)
	}
	sort.Slice(p.Sites, func(i, j int) bool { return p.Sites[i].ID < p.Sites[j].ID })
	if len(errs) > 0 {
		return nil, errs
	}
	return p, nil
}

// parseProjectMeta decodes the project file with version gate + warnings.
func parseProjectMeta(file string, data []byte) (ProjectMeta, []Warning, ValidationErrors) {
	var meta ProjectMeta
	var warnings []Warning
	m, err := asMap(data)
	if err != nil {
		return meta, nil, ValidationErrors{{File: file, Field: "", Reason: "invalid YAML: " + err.Error()}}
	}
	for _, k := range unknownKeys(m, knownProjectKeys) {
		warnings = append(warnings, Warning{File: file, Field: k, Reason: "unknown field, ignored"})
	}
	if g, ok := m["global"].(map[string]any); ok {
		for _, k := range unknownKeys(g, knownGlobalKeys) {
			warnings = append(warnings, Warning{File: file, Field: "global." + k, Reason: "unknown field, ignored"})
		}
	}
	ver, ok := m["version"]
	if !ok {
		return meta, warnings, ValidationErrors{{File: file, Field: "version", Reason: "missing version: expected version: 1"}}
	}
	verNum, ok := ver.(int)
	if !ok {
		return meta, warnings, ValidationErrors{{File: file, Field: "version", Reason: "version must be an integer"}}
	}
	if verNum != constants.SchemaVersion {
		return meta, warnings, ValidationErrors{{File: file, Field: "version", Reason: fmt.Sprintf("unsupported version %d: this tool supports version %d", verNum, constants.SchemaVersion)}}
	}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return meta, warnings, ValidationErrors{{File: file, Field: "", Reason: "invalid YAML: " + err.Error()}}
	}
	var errs ValidationErrors
	if strings.TrimSpace(meta.Name) == "" {
		errs = append(errs, FieldError{File: file, Field: "name", Reason: "project name must not be empty"})
	}
	return meta, warnings, errs
}

// parseSite decodes one site file with version gate, warnings, and full validation.
func parseSite(id, file string, data []byte) (Site, []Warning, ValidationErrors) {
	var site Site
	var warnings []Warning
	m, err := asMap(data)
	if err != nil {
		return site, nil, ValidationErrors{{File: file, Field: "", Reason: "invalid YAML: " + err.Error()}}
	}
	for _, k := range unknownKeys(m, knownSiteKeys) {
		warnings = append(warnings, Warning{File: file, Field: k, Reason: "unknown field, ignored"})
	}
	if b, ok := m["backend"].(map[string]any); ok {
		for _, k := range unknownKeys(b, knownBackendKeys) {
			warnings = append(warnings, Warning{File: file, Field: "backend." + k, Reason: "unknown field, ignored"})
		}
	}
	ver, ok := m["version"]
	if !ok {
		return site, warnings, ValidationErrors{{File: file, Field: "version", Reason: "missing version: expected version: 1"}}
	}
	verNum, ok := ver.(int)
	if !ok {
		return site, warnings, ValidationErrors{{File: file, Field: "version", Reason: "version must be an integer"}}
	}
	if verNum != constants.SchemaVersion {
		return site, warnings, ValidationErrors{{File: file, Field: "version", Reason: fmt.Sprintf("unsupported version %d: this tool supports version %d", verNum, constants.SchemaVersion)}}
	}
	if err := yaml.Unmarshal(data, &site); err != nil {
		return site, warnings, ValidationErrors{{File: file, Field: "", Reason: "invalid YAML: " + err.Error()}}
	}
	site.ID = id
	return site, warnings, ValidateSite(file, site)
}
