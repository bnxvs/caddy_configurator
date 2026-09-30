package project

import (
	"os"
	"path/filepath"
	"testing"
)

// Rendering the same sources twice (in memory and on disk) must produce
// byte-identical artifacts, keeping git history clean (task 3.4).
func TestDeterministicArtifacts(t *testing.T) {
	root := fixtureRoot(t)
	a, err := PreviewProject(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := PreviewProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if a.Caddyfile != b.Caddyfile || a.Compose != b.Compose {
		t.Fatal("in-memory renders differ between runs")
	}
	if _, err := GenerateProject(root); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(root, "generated", "Caddyfile"))
	if err != nil {
		t.Fatal(err)
	}
	firstCompose, err := os.ReadFile(filepath.Join(root, "generated", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateProject(root); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(root, "generated", "Caddyfile"))
	secondCompose, _ := os.ReadFile(filepath.Join(root, "generated", "compose.yaml"))
	if string(first) != string(second) || string(firstCompose) != string(secondCompose) {
		t.Fatal("on-disk artifacts differ between generate runs")
	}
}
