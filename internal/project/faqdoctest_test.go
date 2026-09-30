package project

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every example in FAQ.md is verified: site YAML blocks must load and
// validate, and every caddyfile/compose fragment must appear verbatim in
// the generated artifacts. This keeps the FAQ from going stale
// (the ```sh docker block is documentation only and is skipped).
func TestFaqExamples(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "FAQ.md"))
	if err != nil {
		t.Fatal(err)
	}
	fence := regexp.MustCompile("(?m)^```(yaml|caddyfile)\\s*$")
	siteRe := regexp.MustCompile("(?s)`sites/([a-z0-9_.-]+)\\.yaml`:\\s*\n```yaml\\n(.*?)```")
	matches := fence.FindAllStringSubmatchIndex(string(doc), -1)
	if len(matches) == 0 {
		t.Fatal("no fenced examples found in FAQ.md")
	}
	type block struct {
		lang string
		body string
	}
	type named struct {
		id   string
		body string
	}
	var blocks []block
	for i, m := range matches {
		lang := string(doc[m[2]:m[3]])
		start := m[1]
		end := len(doc)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		body := string(doc[start:end])
		if idx := strings.Index(body, "\n```"); idx >= 0 {
			body = body[:idx]
		}
		body = strings.Trim(body, "\n")
		blocks = append(blocks, block{lang, body})
	}

	var sites []named
	var caddyFrags, composeFrags []string
	for _, b := range blocks {
		switch {
		case b.lang == "yaml" && !(strings.HasPrefix(b.body, "version: 1") && strings.Contains(b.body, "\npreset:")):
			composeFrags = append(composeFrags, b.body)
		case b.lang == "caddyfile":
			caddyFrags = append(caddyFrags, b.body)
		}
	}
	// Site ids come from the `sites/<id>.yaml`: captions so derived values
	// (service names, upstreams, volume paths) render exactly as documented.
	for _, m := range siteRe.FindAllStringSubmatch(string(doc), -1) {
		sites = append(sites, named{id: m[1], body: strings.Trim(m[2], "\n")})
	}
	if len(sites) == 0 {
		t.Fatal("no site examples found in FAQ.md")
	}
	if len(caddyFrags) == 0 || len(composeFrags) == 0 {
		t.Fatal("no output fragments found in FAQ.md")
	}

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
	write("caddy.project.yaml", "version: 1\nname: faqcheck\n")
	for _, s := range sites {
		write("sites/"+s.id+".yaml", s.body)
	}
	if _, err := GenerateProject(root); err != nil {
		t.Fatalf("FAQ site examples do not validate: %v", err)
	}
	caddyData, err := os.ReadFile(filepath.Join(root, "generated", "Caddyfile"))
	if err != nil {
		t.Fatal(err)
	}
	composeData, err := os.ReadFile(filepath.Join(root, "generated", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, frag := range caddyFrags {
		if !strings.Contains(string(caddyData), strings.Trim(frag, "\n")) {
			t.Fatalf("FAQ caddyfile fragment not in generated Caddyfile:\n%s", frag)
		}
	}
	for _, frag := range composeFrags {
		if !strings.Contains(string(composeData), strings.Trim(frag, "\n")) {
			t.Fatalf("FAQ compose fragment not in generated compose.yaml:\n%s", frag)
		}
	}
}
