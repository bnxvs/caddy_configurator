package caddy

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// execTimeout bounds calls to the external binary.
const execTimeout = 30 * time.Second

// Found reports whether a `caddy` binary is available in PATH.
func Found() bool {
	_, err := exec.LookPath("caddy")
	return err == nil
}

// FormatCaddyfile pipes input through `caddy fmt --overwrite` on a temp
// file when the binary is available. It returns the formatted output and
// whether formatting was applied; any failure falls back to the input.
func FormatCaddyfile(input string) (string, bool) {
	bin, err := exec.LookPath("caddy")
	if err != nil {
		return input, false
	}
	dir, err := os.MkdirTemp("", "caddy-fmt-")
	if err != nil {
		return input, false
	}
	defer os.RemoveAll(dir)
	tmp := filepath.Join(dir, "Caddyfile")
	if err := os.WriteFile(tmp, []byte(input), 0o600); err != nil {
		return input, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "fmt", "--overwrite", tmp)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return input, false
	}
	out, err := os.ReadFile(tmp)
	if err != nil {
		return input, false
	}
	return string(out), true
}

// ValidateResult is the outcome of `caddy validate`.
type ValidateResult struct {
	Available bool
	OK        bool
	Output    string
}

// ValidateCaddyfile checks a rendered Caddyfile with `caddy validate
// --adapter caddyfile` when the binary exists.
func ValidateCaddyfile(input string) ValidateResult {
	bin, err := exec.LookPath("caddy")
	if err != nil {
		return ValidateResult{Available: false}
	}
	dir, err := os.MkdirTemp("", "caddy-validate-")
	if err != nil {
		return ValidateResult{Available: true, Output: err.Error()}
	}
	defer os.RemoveAll(dir)
	tmp := filepath.Join(dir, "Caddyfile")
	if err := os.WriteFile(tmp, []byte(input), 0o600); err != nil {
		return ValidateResult{Available: true, Output: err.Error()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "validate", "--config", tmp, "--adapter", "caddyfile")
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	runErr := cmd.Run()
	return ValidateResult{Available: true, OK: runErr == nil, Output: combined.String()}
}
