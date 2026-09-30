package cli

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// The release artifact must build and run with no Node toolchain on PATH
// (task 8.2 verification): CGO off, module cache only, node/npm unresolvable.
func TestSingleBinaryWithoutNode(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("path-stripping test supports darwin/linux")
	}
	binDir := t.TempDir()
	goReal, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	if err := os.Symlink(goReal, filepath.Join(binDir, "go")); err != nil {
		t.Fatal(err)
	}
	stripped := []string{
		"PATH=" + binDir + ":/usr/bin:/bin",
		"HOME=" + os.Getenv("HOME"),
		"GOCACHE=" + os.Getenv("GOCACHE"),
		"GOPATH=" + os.Getenv("GOPATH"),
		"GOFLAGS=-mod=mod",
		"GOPROXY=off",
		"CGO_ENABLED=0",
	}
	// Precondition: node and npm are really gone here.
	for _, tool := range []string{"node", "npm"} {
		check := exec.Command("sh", "-c", "command -v "+tool)
		check.Env = stripped
		if err := check.Run(); err == nil {
			t.Fatalf("%s resolvable in stripped env: not a valid no-Node check", tool)
		}
	}

	bin := filepath.Join(t.TempDir(), "caddy-configurator")
	build := exec.Command("go", "build", "-o", bin, "./cmd/caddy-configurator")
	build.Dir = filepath.Join("..", "..")
	build.Env = stripped
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("stripped go build: %v\n%s", err, out)
	}

	root := fixtureCLIProject(t)
	gen := exec.Command(bin, "generate", root)
	gen.Env = stripped
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("stripped generate: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(root, "generated", "Caddyfile")); err != nil {
		t.Fatalf("no Caddyfile after stripped generate: %v", err)
	}

	serve := exec.Command(bin, "serve", root, "--port", "18081")
	serve.Env = stripped
	if err := serve.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = serve.Process.Kill() }()
	deadline := time.Now().Add(15 * time.Second)
	for {
		res, err := http.Get("http://127.0.0.1:18081/api/v1/project")
		if err == nil {
			res.Body.Close()
			if res.StatusCode == 200 {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("stripped serve never became ready")
		}
		time.Sleep(200 * time.Millisecond)
	}
}
