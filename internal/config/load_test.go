package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const projectFile = `version: 1
name: demo
global:
  email: admin@example.com
`

func TestLoadValidProjectAllPresets(t *testing.T) {
	root := writeProject(t, map[string]string{
		"caddy.project.yaml": projectFile,
		"sites/web.yaml": `version: 1
domains: [example.com, www.example.com]
preset: static
tls: auto
log: true
root: /var/www/html
`,
		"sites/app.yaml": `version: 1
domains: [app.example.com]
preset: spa
tls: auto
log: true
root: /var/www/app
`,
		"sites/api.yaml": `version: 1
domains: [api.example.com]
preset: proxy
tls: auto
log: true
backend:
  service: api
  image: node:20-alpine
  internalPort: 3000
  env: {NODE_ENV: production}
`,
		"sites/blog.yaml": `version: 1
domains: [blog.example.com]
preset: php
tls: off
log: false
root: /var/www/blog
backend:
  image: php:8.3-fpm
`,
	})
	p, err := LoadProject(root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if len(p.Sites) != 4 {
		t.Fatalf("got %d sites, want 4", len(p.Sites))
	}
	if p.Sites[0].ID != "api" || p.Sites[3].ID != "web" {
		t.Fatalf("sites not sorted by id: %v", []string{p.Sites[0].ID, p.Sites[3].ID})
	}
	if p.Meta.Global.Email != "admin@example.com" {
		t.Fatalf("email = %q", p.Meta.Global.Email)
	}
	api := p.Sites[0]
	if api.Backend.Image != "node:20-alpine" || api.Backend.InternalPort != 3000 {
		t.Fatalf("api backend = %+v", api.Backend)
	}
}

func TestMissingProjectFile(t *testing.T) {
	root := writeProject(t, map[string]string{"sites/a.yaml": "version: 1\n"})
	_, err := LoadProject(root)
	if err == nil || !strings.Contains(err.Error(), "not a configurator project") {
		t.Fatalf("err = %v, want 'not a configurator project'", err)
	}
}

func TestUnsupportedProjectVersion(t *testing.T) {
	root := writeProject(t, map[string]string{
		"caddy.project.yaml": "version: 2\nname: demo\n",
	})
	_, err := LoadProject(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported version 2") {
		t.Fatalf("err = %v, want unsupported version", err)
	}
}

func TestUnsupportedSiteVersion(t *testing.T) {
	root := writeProject(t, map[string]string{
		"caddy.project.yaml": projectFile,
		"sites/a.yaml":       "version: 2\ndomains: [a.example.com]\npreset: static\ntls: auto\nlog: true\nroot: /x\n",
	})
	_, err := LoadProject(root)
	if err == nil || !strings.Contains(err.Error(), "sites/a.yaml") || !strings.Contains(err.Error(), "unsupported version 2") {
		t.Fatalf("err = %v, want file + unsupported version", err)
	}
}

func TestMissingVersionRejected(t *testing.T) {
	root := writeProject(t, map[string]string{
		"caddy.project.yaml": "name: demo\n",
	})
	_, err := LoadProject(root)
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("err = %v, want missing version error", err)
	}
}

func TestUnknownFieldsWarn(t *testing.T) {
	root := writeProject(t, map[string]string{
		"caddy.project.yaml": projectFile,
		"sites/api.yaml": `version: 1
domains: [api.example.com]
preset: proxy
tls: auto
log: true
upstream: ignored-by-v1
backend:
  image: node:20-alpine
  internalPort: 3000
  build: ./api
`,
	})
	p, err := LoadProject(root)
	if err != nil {
		t.Fatalf("LoadProject should tolerate unknown fields: %v", err)
	}
	var gotUpstream, gotBuild bool
	for _, w := range p.Warnings {
		if w.Field == "upstream" {
			gotUpstream = true
		}
		if w.Field == "backend.build" {
			gotBuild = true
		}
	}
	if !gotUpstream || !gotBuild {
		t.Fatalf("warnings = %+v, want upstream + backend.build", p.Warnings)
	}
}

func TestInvalidSiteIDRejected(t *testing.T) {
	// Sanity: ValidSiteID itself.
	for id, want := range map[string]bool{"api": true, "a.b_c-d": true, "../x": false, "a/b": false, "": false, "UP": false, "a b": false} {
		if ValidSiteID(id) != want {
			t.Fatalf("ValidSiteID(%q) = %v, want %v", id, !want, want)
		}
	}
	dir := writeProject(t, map[string]string{
		"caddy.project.yaml":  projectFile,
		"sites/ok.yaml":       "version: 1\ndomains: [a.example.com]\npreset: static\ntls: auto\nlog: true\nroot: /x\n",
		"sites/BAD NAME.yaml": "version: 1\n",
	})
	_, err := LoadProject(dir)
	if err == nil || !strings.Contains(err.Error(), "sites/BAD NAME.yaml") {
		t.Fatalf("err = %v, want invalid id error naming the file", err)
	}
}
