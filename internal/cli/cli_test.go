package cli

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"caddy-configurator/internal/config"
	"caddy-configurator/internal/server"
)

func TestInitCreatesSkeleton(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new-project")
	if err := InitProject(dir, ""); err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	p, err := config.LoadProject(dir)
	if err != nil {
		t.Fatalf("skeleton invalid: %v", err)
	}
	if p.Meta.Name != "new-project" || len(p.Sites) != 0 {
		t.Fatalf("meta = %+v", p.Meta)
	}
	// Second init fails closed.
	before, _ := os.ReadFile(filepath.Join(dir, "caddy.project.yaml"))
	if err := InitProject(dir, "other"); err == nil {
		t.Fatal("second init succeeded, want refusal")
	}
	after, _ := os.ReadFile(filepath.Join(dir, "caddy.project.yaml"))
	if string(before) != string(after) {
		t.Fatal("second init modified the project file")
	}
}

func fixtureCLIProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("caddy.project.yaml", "version: 1\nname: demo\n")
	mustWrite("sites/web.yaml", "version: 1\ndomains: [example.com]\npreset: static\ntls: auto\nlog: true\nroot: /var/www/html\n")
	return root
}

func TestValidateCommand(t *testing.T) {
	if err := ValidateProject(fixtureCLIProject(t)); err != nil {
		t.Fatalf("ValidateProject: %v", err)
	}
	bad := fixtureCLIProject(t)
	os.WriteFile(filepath.Join(bad, "sites", "bad.yaml"),
		[]byte("version: 1\npreset: proxy\ntls: auto\nlog: true\n"), 0o644)
	if err := ValidateProject(bad); err == nil {
		t.Fatal("ValidateProject accepted invalid project")
	}
}

func TestServeResponds(t *testing.T) {
	root := fixtureCLIProject(t)
	srv, err := server.New(root, "")
	if err != nil {
		t.Fatal(err)
	}
	// Ephemeral port: ResolveAddr maps 0 to the default, and sharing a
	// fixed port across tests lets the HTTP transport reuse keep-alive
	// connections to a previous test's server.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go http.Serve(ln, srv.Handler())
	base := "http://" + ln.Addr().String()

	for _, path := range []string{"/", "/api/v1/project"} {
		res, err := http.Get(base + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("GET %s = %d", path, res.StatusCode)
		}
		if path == "/" && !strings.Contains(string(body), `id="root"`) {
			t.Fatal("index is not the SPA")
		}
	}
	if addr := ln.Addr().String(); !isLoopback(addr) {
		t.Fatalf("listening on %s, want loopback", addr)
	}
}

func TestServeDevProxy(t *testing.T) {
	root := fixtureCLIProject(t)
	vite := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "vite-dev")
	})
	lnVite, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lnVite.Close()
	go http.Serve(lnVite, vite)

	srv, err := server.New(root, "http://"+lnVite.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go http.Serve(ln, srv.Handler())

	res, err := http.Get("http://" + ln.Addr().String() + "/src/main.tsx")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(body) != "vite-dev" {
		t.Fatalf("body = %q, want proxied vite-dev", body)
	}
	// API still served locally.
	res, err = http.Get("http://" + ln.Addr().String() + "/api/v1/project")
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("api in dev mode: %v %v", res, err)
	}
	res.Body.Close()
}

func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	return host == "127.0.0.1" || host == "::1"
}
