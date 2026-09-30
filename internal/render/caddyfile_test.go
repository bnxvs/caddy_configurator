package render

import (
	"os"
	"path/filepath"
	"testing"

	"caddy-configurator/internal/config"
)

func fixtureProject() *config.Project {
	return &config.Project{
		Meta: config.ProjectMeta{
			Version: 1, Name: "demo",
			Global: config.GlobalOptions{Email: "admin@example.com"},
		},
		Sites: []config.Site{
			{ID: "blog", Version: 1, Domains: []string{"blog.example.com"},
				Preset: "php", TLS: "off", Log: true, Root: "/var/www/blog",
				Backend: &config.Backend{Image: "php:8.3-fpm"}},
			{ID: "web", Version: 1, Domains: []string{"www.example.com", "example.com"},
				Preset: "static", TLS: "auto", Log: true, Root: "/var/www/html"},
			{ID: "api", Version: 1, Domains: []string{"api.example.com"},
				Preset: "proxy", TLS: "auto", Log: false,
				Backend: &config.Backend{Service: "api", Image: "node:20-alpine", InternalPort: 3000,
					Env: map[string]string{"NODE_ENV": "production"}}},
			{ID: "app", Version: 1, Domains: []string{"app.example.com"},
				Preset: "spa", TLS: "auto", Log: true, Root: "/var/www/app"},
		},
	}
}

func TestCaddyfileGolden(t *testing.T) {
	got := RenderCaddyfile(fixtureProject())
	want, err := os.ReadFile(filepath.Join("testdata", "caddyfile", "basic.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("Caddyfile mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestCaddyfileOrderingStable(t *testing.T) {
	a := RenderCaddyfile(fixtureProject())
	rev := fixtureProject()
	for i, j := 0, len(rev.Sites)-1; i < j; i, j = i+1, j-1 {
		rev.Sites[i], rev.Sites[j] = rev.Sites[j], rev.Sites[i]
	}
	b := RenderCaddyfile(rev)
	if a != b {
		t.Fatalf("order-dependent output:\n%s\nvs\n%s", a, b)
	}
}
