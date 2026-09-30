package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"caddy-configurator/internal/project"
	"caddy-configurator/internal/server"
)

// Full flow: init, one site per preset via the API, preview, generate,
// re-generate byte-identical, clean inventory (task 8.1 verification).
func TestEndToEndFlow(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if err := InitProject(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	srv, err := server.New(root, "")
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	call := func(method, path, body string) *httptest.ResponseRecorder {
		var rdr *strings.Reader
		if body != "" {
			rdr = strings.NewReader(body)
		} else {
			rdr = strings.NewReader("")
		}
		var req *http.Request
		if body == "" {
			req = httptest.NewRequest(method, path, nil)
		} else {
			req = httptest.NewRequest(method, path, rdr)
		}
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	sites := []string{
		`{"id":"web","version":1,"domains":["example.com"],"preset":"static","tls":"auto","log":true,"root":"/var/www/html"}`,
		`{"id":"app","version":1,"domains":["app.example.com"],"preset":"spa","tls":"auto","log":true,"root":"/var/www/app"}`,
		`{"id":"api","version":1,"domains":["api.example.com"],"preset":"proxy","tls":"auto","log":true,"backend":{"image":"node:20-alpine","internalPort":3000}}`,
		`{"id":"blog","version":1,"domains":["blog.example.com"],"preset":"php","tls":"auto","log":true,"root":"/var/www/blog","backend":{"image":"php:8.3-fpm"}}`,
	}
	for _, body := range sites {
		if rec := call("POST", "/api/v1/sites", body); rec.Code != 201 {
			t.Fatalf("create: code=%d body=%s", rec.Code, rec.Body.String())
		}
	}

	rec := call("GET", "/api/v1/preview", "")
	if rec.Code != 200 {
		t.Fatalf("preview: %s", rec.Body.String())
	}
	var pv map[string]any
	json.Unmarshal(rec.Body.Bytes(), &pv)
	for _, marker := range []string{"file_server", "try_files", "reverse_proxy api:3000", "php_fastcgi blog:9000"} {
		if !strings.Contains(pv["caddyfile"].(string), marker) {
			t.Fatalf("preview caddyfile lacks %q", marker)
		}
	}

	if rec := call("POST", "/api/v1/generate", ""); rec.Code != 200 {
		t.Fatalf("generate: %s", rec.Body.String())
	}
	read := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return string(data)
	}
	firstCaddy, firstCompose := read("generated/Caddyfile"), read("generated/compose.yaml")

	if _, err := project.GenerateProject(root); err != nil {
		t.Fatalf("re-generate: %v", err)
	}
	if second := read("generated/Caddyfile"); second != firstCaddy {
		t.Fatal("Caddyfile changed between generates")
	}
	if second := read("generated/compose.yaml"); second != firstCompose {
		t.Fatal("compose changed between generates")
	}

	// Inventory: exactly the expected files, no temp leftovers.
	var files []string
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(root, p)
			files = append(files, rel)
		}
		return nil
	})
	want := map[string]bool{
		"caddy.project.yaml":     true,
		"sites/web.yaml":         true,
		"sites/app.yaml":         true,
		"sites/api.yaml":         true,
		"sites/blog.yaml":        true,
		"generated/Caddyfile":    true,
		"generated/compose.yaml": true,
	}
	if len(files) != len(want) {
		t.Fatalf("inventory = %v", files)
	}
	for _, f := range files {
		if !want[filepath.ToSlash(f)] {
			t.Fatalf("unexpected file %q (temp leftover?)", f)
		}
	}
}
