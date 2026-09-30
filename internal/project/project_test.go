package project

import (
	"os"
	"path/filepath"
	"testing"
)

func fixtureRoot(t *testing.T) string {
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

func mtimes(t *testing.T, root string) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(root, p)
			out[rel] = info.ModTime().UnixNano()
		}
		return nil
	})
	return out
}

func TestPreviewPure(t *testing.T) {
	root := fixtureRoot(t)
	before := mtimes(t, root)
	pv, err := PreviewProject(root)
	if err != nil {
		t.Fatalf("PreviewProject: %v", err)
	}
	if pv.Caddyfile == "" || pv.Compose == "" {
		t.Fatal("empty preview")
	}
	after := mtimes(t, root)
	if len(before) != len(after) {
		t.Fatalf("preview created files: %v -> %v", before, after)
	}
	for f, m := range before {
		if after[f] != m {
			t.Fatalf("preview modified %s", f)
		}
	}
}

func TestGenerateWritesBoth(t *testing.T) {
	root := fixtureRoot(t)
	res, err := GenerateProject(root)
	if err != nil {
		t.Fatalf("GenerateProject: %v", err)
	}
	if res.CaddyfilePath != "generated/Caddyfile" || res.ComposePath != "generated/compose.yaml" {
		t.Fatalf("paths = %+v", res)
	}
	for _, rel := range []string{"generated/Caddyfile", "generated/compose.yaml"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil || len(data) == 0 {
			t.Fatalf("%s missing/empty: %v", rel, err)
		}
	}
	pv, _ := PreviewProject(root)
	caddyData, _ := os.ReadFile(filepath.Join(root, "generated", "Caddyfile"))
	composeData, _ := os.ReadFile(filepath.Join(root, "generated", "compose.yaml"))
	if string(caddyData) != pv.Caddyfile || string(composeData) != pv.Compose {
		t.Fatal("generated files differ from preview")
	}
}

func TestGenerateInvalidWritesNothing(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "sites"), 0o755)
	os.WriteFile(filepath.Join(root, "caddy.project.yaml"), []byte("version: 1\nname: demo\n"), 0o644)
	os.WriteFile(filepath.Join(root, "sites", "bad.yaml"), []byte("version: 1\npreset: static\ntls: auto\nlog: true\n"), 0o644)
	if _, err := GenerateProject(root); err == nil {
		t.Fatal("GenerateProject accepted invalid project")
	}
	if _, err := os.Stat(filepath.Join(root, "generated")); !os.IsNotExist(err) {
		t.Fatal("generated/ created for invalid project")
	}
}
