package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every fenced ```yaml project / ```yaml site:<id> block in docs/schema.md
// must load and validate. This keeps documentation and implementation
// from drifting apart (task 2.4 verification).
func TestDocsSchemaExamples(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "schema.md"))
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile("(?m)^```yaml (project|site:[a-z0-9_.-]+)\\s*$")
	matches := re.FindAllStringSubmatchIndex(string(doc), -1)
	if len(matches) == 0 {
		t.Fatal("no yaml examples found in docs/schema.md")
	}
	files := map[string]string{}
	for i, m := range matches {
		kind := string(doc[m[2]:m[3]])
		start := m[1]
		end := len(doc)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		body := string(doc[start:end])
		if idx := strings.Index(body, "\n```"); idx >= 0 {
			body = body[:idx]
		}
		switch {
		case kind == "project":
			files["caddy.project.yaml"] = body
		case strings.HasPrefix(kind, "site:"):
			id := strings.TrimPrefix(kind, "site:")
			files["sites/"+id+".yaml"] = body
		}
	}
	root := writeProject(t, files)
	p, err := LoadProject(root)
	if err != nil {
		t.Fatalf("docs examples do not validate: %v", err)
	}
	if len(p.Sites) != 4 {
		t.Fatalf("want 4 doc sites, got %d", len(p.Sites))
	}
}
