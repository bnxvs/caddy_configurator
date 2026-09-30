// Package project orchestrates load → render → write for preview and
// generate flows shared by the CLI and the web UI.
package project

import (
	"io"
	"os"
	"path/filepath"

	"caddy-configurator/internal/caddy"
	"caddy-configurator/internal/config"
	"caddy-configurator/internal/constants"
	"caddy-configurator/internal/render"
)

// Preview is the in-memory rendering of both artifacts.
type Preview struct {
	Caddyfile string
	Compose   string
	Warnings  []config.Warning
}

// PreviewProject loads and renders without touching the disk.
func PreviewProject(root string) (Preview, error) {
	p, err := config.LoadProject(root)
	if err != nil {
		return Preview{}, err
	}
	return Preview{
		Caddyfile: render.RenderCaddyfile(p),
		Compose:   render.RenderCompose(p),
		Warnings:  p.Warnings,
	}, nil
}

// GenerateResult reports written artifact paths and formatting.
type GenerateResult struct {
	CaddyfilePath string
	ComposePath   string
	Formatted     bool
	Warnings      []config.Warning
}

// GenerateProject loads, renders, formats, and atomically writes both
// artifacts. On validation failure nothing is written.
func GenerateProject(root string) (*GenerateResult, error) {
	p, err := config.LoadProject(root)
	if err != nil {
		return nil, err
	}
	caddyfile, formatted := caddy.FormatCaddyfile(render.RenderCaddyfile(p))
	compose := render.RenderCompose(p)

	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	genDir := filepath.Join(abs, constants.GeneratedDir)
	if err := os.MkdirAll(genDir, 0o755); err != nil {
		return nil, err
	}
	caddyPath := filepath.Join(genDir, constants.CaddyfileName)
	composePath := filepath.Join(genDir, constants.ComposeFileName)
	if err := config.AtomicWrite(caddyPath, 0o644, writeString(caddyfile)); err != nil {
		return nil, err
	}
	if err := config.AtomicWrite(composePath, 0o644, writeString(compose)); err != nil {
		return nil, err
	}
	return &GenerateResult{
		CaddyfilePath: filepath.Join(constants.GeneratedDir, constants.CaddyfileName),
		ComposePath:   filepath.Join(constants.GeneratedDir, constants.ComposeFileName),
		Formatted:     formatted,
		Warnings:      p.Warnings,
	}, nil
}

func writeString(s string) func(w io.Writer) error {
	return func(w io.Writer) error {
		_, err := w.Write([]byte(s))
		return err
	}
}
