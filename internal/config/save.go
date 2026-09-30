package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"caddy-configurator/internal/constants"
)

// AtomicWrite writes via a temp file in the same directory followed by a
// rename, so a failure partway never leaves a half-written target behind
// and no temp leftovers survive.
func AtomicWrite(path string, perm os.FileMode, write func(w io.Writer) error) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	failed := true
	defer func() {
		if failed {
			os.Remove(name)
		}
	}()
	if err := write(tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, perm); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	failed = false
	return nil
}

// sitePath joins root/sites/<id>.yaml and guarantees containment in root.
func sitePath(root, id string) (string, error) {
	if !ValidSiteID(id) {
		return "", ValidationErrors{{
			File:   constants.SitesDir + "/" + id + constants.SiteFileSuffix,
			Field:  "id",
			Reason: fmt.Sprintf("invalid site id %q: use [a-z0-9_.-]", id),
		}}
	}
	return filepath.Join(root, constants.SitesDir, id+constants.SiteFileSuffix), nil
}

// SaveSite validates and atomically persists one site. On validation
// failure nothing on disk is touched.
func SaveSite(root string, s Site) error {
	p, err := sitePath(root, s.ID)
	if err != nil {
		return err
	}
	file := filepath.Join(constants.SitesDir, s.ID+constants.SiteFileSuffix)
	if errs := ValidateSite(file, s); len(errs) > 0 {
		return errs
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return AtomicWrite(p, 0o644, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}

// DeleteSite removes exactly one site file.
func DeleteSite(root, id string) error {
	p, err := sitePath(root, id)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		return fmt.Errorf("%s: cannot delete: %w", filepath.Join(constants.SitesDir, id+constants.SiteFileSuffix), err)
	}
	return nil
}

// SaveProjectMeta validates and atomically persists caddy.project.yaml.
func SaveProjectMeta(root string, meta ProjectMeta) error {
	file := constants.ProjectFileName
	if meta.Version != constants.SchemaVersion {
		return ValidationErrors{{File: file, Field: "version", Reason: fmt.Sprintf("unsupported version %d", meta.Version)}}
	}
	if meta.Name == "" {
		return ValidationErrors{{File: file, Field: "name", Reason: "project name must not be empty"}}
	}
	data, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	return AtomicWrite(filepath.Join(root, file), 0o644, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}
