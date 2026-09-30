package cli

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Every `$ caddy-configurator ...` line in docs/cli.md runs verbatim here
// (task 7.4 verification). Only that prefix is collected; npm/go/docker
// lines are documentation, not CLI contract.
func TestDocsCliCommands(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "cli.md"))
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^\$ (caddy-configurator .+?)\s*$`)
	matches := re.FindAllStringSubmatch(string(doc), -1)
	if len(matches) == 0 {
		t.Fatal("no CLI examples found in docs/cli.md")
	}

	bin := filepath.Join(t.TempDir(), "caddy-configurator")
	build := exec.Command("go", "build", "-o", bin, "./cmd/caddy-configurator")
	build.Dir = filepath.Join("..", "..")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	workdir := t.TempDir()
	for _, m := range matches {
		args := strings.Fields(m[1])[1:] // drop argv[0]
		if args[0] == "serve" {
			runServeExample(t, bin, workdir, args[1:])
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Dir = workdir
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			t.Fatalf("%s: %v\n%s", m[1], err, out)
		}
	}
}

func runServeExample(t *testing.T, bin, workdir string, args []string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, append([]string{"serve"}, args...)...)
	cmd.Dir = workdir
	cmd.Stdout = nil
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	// Wait for the listener: poll the documented port.
	deadline := time.Now().Add(15 * time.Second)
	for {
		res, err := http.Get("http://127.0.0.1:18080/api/v1/project")
		if err == nil {
			res.Body.Close()
			if res.StatusCode == 200 {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("serve %v never became ready (pipe: %v)", args, out)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
