package render

import (
	"os"
	"path/filepath"
	"testing"

	"caddy-configurator/internal/config"
)

func staticOnlyProject() *config.Project {
	return &config.Project{
		Meta: config.ProjectMeta{Version: 1, Name: "demo"},
		Sites: []config.Site{
			{ID: "web", Version: 1, Domains: []string{"example.com"},
				Preset: "static", TLS: "auto", Log: true, Root: "/var/www/html"},
		},
	}
}

func TestComposeGoldenStaticOnly(t *testing.T) {
	got := RenderCompose(staticOnlyProject())
	want, err := os.ReadFile(filepath.Join("testdata", "compose", "static-only.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("compose mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestComposeGoldenProxyPHP(t *testing.T) {
	got := RenderCompose(fixtureProject())
	want, err := os.ReadFile(filepath.Join("testdata", "compose", "proxy-php.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("compose mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestComposeOrderingStable(t *testing.T) {
	a := RenderCompose(fixtureProject())
	rev := fixtureProject()
	for i, j := 0, len(rev.Sites)-1; i < j; i, j = i+1, j-1 {
		rev.Sites[i], rev.Sites[j] = rev.Sites[j], rev.Sites[i]
	}
	if b := RenderCompose(rev); a != b {
		t.Fatalf("order-dependent output:\n%s\nvs\n%s", a, b)
	}
}
