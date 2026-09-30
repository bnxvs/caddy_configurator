package config

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteFailureKeepsOriginal(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "site.yaml")
	if err := os.WriteFile(target, []byte("original: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	errBoom := errors.New("boom mid-write")
	err := AtomicWrite(target, 0o644, func(w io.Writer) error {
		w.Write([]byte("partial..."))
		return errBoom
	})
	if err != errBoom {
		t.Fatalf("err = %v, want boom", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "original: true\n" {
		t.Fatalf("target = %q, want original intact", got)
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, ".tmp-*"))
	if len(leftovers) != 0 {
		t.Fatalf("temp leftovers: %v", leftovers)
	}
}

func TestSaveSiteRoundTrip(t *testing.T) {
	root := writeProject(t, map[string]string{
		"caddy.project.yaml": projectFile,
		"sites/.keep":        "",
	})
	s := validSite()
	s.ID = "web"
	if err := SaveSite(root, s); err != nil {
		t.Fatalf("SaveSite: %v", err)
	}
	p, err := LoadProject(root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if len(p.Sites) != 1 || p.Sites[0].Root != "/var/www/html" {
		t.Fatalf("sites = %+v", p.Sites)
	}
}

func TestSaveInvalidSiteUntouched(t *testing.T) {
	root := writeProject(t, map[string]string{
		"caddy.project.yaml": projectFile,
		"sites/web.yaml":     "version: 1\ndomains: [example.com]\npreset: static\ntls: auto\nlog: true\nroot: /var/www/html\n",
	})
	before, _ := os.ReadFile(filepath.Join(root, "sites", "web.yaml"))
	s := validSite()
	s.ID = "web"
	s.Root = "" // invalid
	if err := SaveSite(root, s); err == nil {
		t.Fatal("SaveSite accepted invalid site")
	}
	after, _ := os.ReadFile(filepath.Join(root, "sites", "web.yaml"))
	if string(before) != string(after) {
		t.Fatalf("file changed on failed save:\n%s\nvs\n%s", before, after)
	}
}

func TestSaveSiteBadIDWritesNothing(t *testing.T) {
	root := writeProject(t, map[string]string{
		"caddy.project.yaml": projectFile,
	})
	s := validSite()
	s.ID = "../evil"
	if err := SaveSite(root, s); err == nil {
		t.Fatal("SaveSite accepted traversal id")
	}
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.Name() != "caddy.project.yaml" && e.Name() != "sites" {
			t.Fatalf("unexpected file created: %s", e.Name())
		}
	}
}
