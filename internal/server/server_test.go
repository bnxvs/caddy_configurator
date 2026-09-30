package server

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("caddy.project.yaml", "version: 1\nname: demo\nglobal:\n  email: admin@example.com\n")
	write("sites/web.yaml", "version: 1\ndomains: [example.com]\npreset: static\ntls: auto\nlog: true\nroot: /var/www/html\n")
	write("sites/api.yaml", "version: 1\ndomains: [api.example.com]\npreset: proxy\ntls: auto\nlog: true\nbackend:\n  image: node:20-alpine\n  internalPort: 3000\n")
	return root
}

func testServer(t *testing.T) (*Server, string) {
	t.Helper()
	root := testRoot(t)
	s, err := New(root, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s, root
}

func doReq(t *testing.T, s *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func TestProjectEndpoint(t *testing.T) {
	s, _ := testServer(t)
	rec := doReq(t, s, "GET", "/api/v1/project", "")
	if rec.Code != 200 {
		t.Fatalf("code = %d body = %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got["name"] != "demo" || got["email"] != "admin@example.com" {
		t.Fatalf("body = %s", rec.Body.String())
	}
	sites := got["sites"].([]any)
	if len(sites) != 2 {
		t.Fatalf("sites = %v", sites)
	}
	if _, ok := got["caddyFound"]; !ok {
		t.Fatal("missing caddyFound")
	}
}

func TestSiteCRUD(t *testing.T) {
	s, root := testServer(t)

	// Create.
	rec := doReq(t, s, "POST", "/api/v1/sites", `{"id":"blog","version":1,"domains":["blog.example.com"],"preset":"php","tls":"auto","log":true,"root":"/var/www/blog","backend":{"image":"php:8.3-fpm"}}`)
	if rec.Code != 201 {
		t.Fatalf("create code = %d body = %s", rec.Code, rec.Body.String())
	}
	// Duplicate.
	rec = doReq(t, s, "POST", "/api/v1/sites", `{"id":"blog","version":1,"domains":["blog.example.com"],"preset":"php","tls":"auto","log":true,"root":"/var/www/blog","backend":{"image":"php:8.3-fpm"}}`)
	if rec.Code != 409 {
		t.Fatalf("duplicate code = %d", rec.Code)
	}
	// Get.
	rec = doReq(t, s, "GET", "/api/v1/sites/blog", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "php:8.3-fpm") {
		t.Fatalf("get code = %d body = %s", rec.Code, rec.Body.String())
	}
	// Get missing.
	if rec := doReq(t, s, "GET", "/api/v1/sites/nope", ""); rec.Code != 404 {
		t.Fatalf("missing code = %d", rec.Code)
	}
	// Invalid update leaves file untouched.
	before, _ := os.ReadFile(filepath.Join(root, "sites", "blog.yaml"))
	rec = doReq(t, s, "PUT", "/api/v1/sites/blog", `{"version":1,"domains":[],"preset":"php","tls":"auto","log":true,"root":"/var/www/blog","backend":{"image":"php:8.3-fpm"}}`)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "domains") {
		t.Fatalf("invalid update code = %d body = %s", rec.Code, rec.Body.String())
	}
	after, _ := os.ReadFile(filepath.Join(root, "sites", "blog.yaml"))
	if string(before) != string(after) {
		t.Fatal("file changed on rejected update")
	}
	// Valid update.
	rec = doReq(t, s, "PUT", "/api/v1/sites/blog", `{"version":1,"domains":["blog.example.com","www.blog.example.com"],"preset":"php","tls":"auto","log":true,"root":"/var/www/blog","backend":{"image":"php:8.3-fpm"}}`)
	if rec.Code != 200 {
		t.Fatalf("update code = %d body = %s", rec.Code, rec.Body.String())
	}
	// Delete removes exactly one file.
	if rec := doReq(t, s, "DELETE", "/api/v1/sites/blog", ""); rec.Code != 204 {
		t.Fatalf("delete code = %d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(root, "sites", "blog.yaml")); !os.IsNotExist(err) {
		t.Fatal("blog.yaml still exists")
	}
	if _, err := os.Stat(filepath.Join(root, "sites", "web.yaml")); err != nil {
		t.Fatal("web.yaml touched by blog delete")
	}
	if rec := doReq(t, s, "DELETE", "/api/v1/sites/blog", ""); rec.Code != 404 {
		t.Fatalf("re-delete code = %d", rec.Code)
	}
}

func TestInvalidIDRejectedNoEscape(t *testing.T) {
	s, root := testServer(t)
	// ".." is path-cleaned by the mux itself (307, handler never runs);
	// other bad ids must be rejected 400 by the handler.
	if rec := doReq(t, s, "PUT", "/api/v1/sites/..", `{"version":1}`); rec.Code != 307 {
		t.Fatalf("id '..': code = %d, want 307 redirect", rec.Code)
	}
	for _, bad := range []string{"BAD!", "a%20b"} {
		rec := doReq(t, s, "PUT", "/api/v1/sites/"+bad, `{"version":1}`)
		if rec.Code != 400 && rec.Code != 404 {
			t.Fatalf("id %q: code = %d, want 4xx", bad, rec.Code)
		}
	}
	// Nothing escaped the project root: only expected entries exist.
	var names []string
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(root, p)
			names = append(names, rel)
		}
		return nil
	})
	if len(names) != 3 {
		t.Fatalf("files = %v, want only the 3 initial files", names)
	}
}

func TestPreviewEndpointPure(t *testing.T) {
	s, root := testServer(t)
	before := map[string]int64{}
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(root, p)
			before[rel] = info.ModTime().UnixNano()
		}
		return nil
	})
	rec := doReq(t, s, "GET", "/api/v1/preview", "")
	if rec.Code != 200 {
		t.Fatalf("code = %d body = %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	json.Unmarshal(rec.Body.Bytes(), &got)
	if !strings.Contains(got["caddyfile"].(string), "reverse_proxy api:3000") {
		t.Fatalf("caddyfile = %s", got["caddyfile"])
	}
	if !strings.Contains(got["compose"].(string), "services:") {
		t.Fatalf("compose = %s", got["compose"])
	}
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(root, p)
			if m, ok := before[rel]; !ok || m != info.ModTime().UnixNano() {
				t.Errorf("preview touched %s", rel)
			}
		}
		return nil
	})
}

func TestGenerateEndpoint(t *testing.T) {
	s, root := testServer(t)
	rec := doReq(t, s, "POST", "/api/v1/generate", "")
	if rec.Code != 200 {
		t.Fatalf("code = %d body = %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got["caddyfilePath"] != "generated/Caddyfile" || got["composePath"] != "generated/compose.yaml" {
		t.Fatalf("body = %s", rec.Body.String())
	}
	for _, rel := range []string{"generated/Caddyfile", "generated/compose.yaml"} {
		if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil || st.Size() == 0 {
			t.Fatalf("%s missing/empty", rel)
		}
	}
}

func TestValidateEndpointShape(t *testing.T) {
	s, _ := testServer(t)
	rec := doReq(t, s, "GET", "/api/v1/validate", "")
	if rec.Code != 200 {
		t.Fatalf("code = %d", rec.Code)
	}
	var got map[string]any
	json.Unmarshal(rec.Body.Bytes(), &got)
	if _, ok := got["errors"]; !ok {
		t.Fatal("missing errors")
	}
	caddy := got["caddy"].(map[string]any)
	if _, ok := caddy["available"]; !ok {
		t.Fatal("missing caddy.available")
	}
}

func TestChangedOnDiskFlow(t *testing.T) {
	s, root := testServer(t)
	rec := doReq(t, s, "GET", "/api/v1/status", "")
	if !strings.Contains(rec.Body.String(), `"changedOnDisk":[]`) {
		t.Fatalf("initial status = %s", rec.Body.String())
	}
	// External edit.
	p := filepath.Join(root, "sites", "web.yaml")
	data, _ := os.ReadFile(p)
	os.WriteFile(p, append(data, []byte("# touched\n")...), 0o644)

	rec = doReq(t, s, "GET", "/api/v1/status", "")
	if !strings.Contains(rec.Body.String(), "sites/web.yaml") {
		t.Fatalf("status after external edit = %s", rec.Body.String())
	}
	// UI reload acknowledgement clears it.
	rec = doReq(t, s, "POST", "/api/v1/status/refresh", "")
	if rec.Code != 200 {
		t.Fatalf("refresh code = %d", rec.Code)
	}
	rec = doReq(t, s, "GET", "/api/v1/status", "")
	if !strings.Contains(rec.Body.String(), `"changedOnDisk":[]`) {
		t.Fatalf("status after refresh = %s", rec.Body.String())
	}
	// And the external content survived (no silent overwrite).
	after, _ := os.ReadFile(p)
	if !strings.Contains(string(after), "# touched") {
		t.Fatal("external edit was overwritten")
	}
}

func TestProjectEmailUpdate(t *testing.T) {
	s, root := testServer(t)
	rec := doReq(t, s, "PUT", "/api/v1/project", `{"email":"ops@example.com"}`)
	if rec.Code != 200 {
		t.Fatalf("code = %d body = %s", rec.Code, rec.Body.String())
	}
	data, _ := os.ReadFile(filepath.Join(root, "caddy.project.yaml"))
	if !strings.Contains(string(data), "ops@example.com") {
		t.Fatalf("project file = %s", data)
	}
	rec = doReq(t, s, "GET", "/api/v1/preview", "")
	if !strings.Contains(rec.Body.String(), "ops@example.com") {
		t.Fatalf("preview lacks new email: %s", rec.Body.String())
	}
	// Empty name is rejected.
	if rec := doReq(t, s, "PUT", "/api/v1/project", `{"name":""}`); rec.Code != 400 {
		t.Fatalf("empty name code = %d", rec.Code)
	}
}

func TestSnippetEndpoint(t *testing.T) {
	s, _ := testServer(t)
	rec := doReq(t, s, "GET", "/api/v1/sites/api/snippet", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "reverse_proxy api:3000") {
		t.Fatalf("code = %d body = %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(t, s, "GET", "/api/v1/sites/nope/snippet", ""); rec.Code != 404 {
		t.Fatalf("missing code = %d", rec.Code)
	}
}

func TestResolveAddr(t *testing.T) {
	if got := ResolveAddr("", 0); got != "127.0.0.1:8080" {
		t.Fatalf("defaults = %q", got)
	}
	if got := ResolveAddr("0.0.0.0", 9090); got != "0.0.0.0:9090" {
		t.Fatalf("explicit = %q", got)
	}
}
