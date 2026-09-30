package config

import (
	"fmt"
	"regexp"
	"strings"
)

// Preset values supported by the v1 schema.
const (
	PresetStatic = "static"
	PresetSPA    = "spa"
	PresetProxy  = "proxy"
	PresetPHP    = "php"
)

// TLS modes supported by the v1 schema.
const (
	TLSAuto = "auto"
	TLSOff  = "off"
)

// ProjectMeta is the parsed content of caddy.project.yaml.
type ProjectMeta struct {
	Version int           `yaml:"version" json:"version"`
	Name    string        `yaml:"name" json:"name"`
	Global  GlobalOptions `yaml:"global" json:"global"`
}

// GlobalOptions holds project-wide settings.
type GlobalOptions struct {
	Email string `yaml:"email,omitempty" json:"email"`
}

// Backend describes a ready container image backing a proxy/php site.
type Backend struct {
	Service      string            `yaml:"service,omitempty" json:"service,omitempty"`
	Image        string            `yaml:"image" json:"image"`
	InternalPort int               `yaml:"internalPort,omitempty" json:"internalPort,omitempty"`
	Env          map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

// Site is the parsed content of one sites/<id>.yaml file.
// ID comes from the file name, not the YAML body.
type Site struct {
	ID      string   `yaml:"-" json:"id"`
	Version int      `yaml:"version" json:"version"`
	Domains []string `yaml:"domains" json:"domains"`
	Preset  string   `yaml:"preset" json:"preset"`
	TLS     string   `yaml:"tls" json:"tls"`
	Log     bool     `yaml:"log" json:"log"`
	Root    string   `yaml:"root,omitempty" json:"root,omitempty"`
	Backend *Backend `yaml:"backend,omitempty" json:"backend,omitempty"`
}

// FileName returns the on-disk file name for the site.
func (s Site) FileName() string {
	return s.ID + ".yaml"
}

// Project is a fully loaded configurator project.
type Project struct {
	Root     string
	Meta     ProjectMeta
	Sites    []Site // sorted by ID
	Warnings []Warning
}

// FieldError is one per-file/per-field problem: "<file>: <field>: <reason>".
type FieldError struct {
	File   string `json:"file"`
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func (e FieldError) Error() string {
	return fmt.Sprintf("%s: %s: %s", e.File, e.Field, e.Reason)
}

// ValidationErrors is a set of field problems blocking an operation.
type ValidationErrors []FieldError

func (e ValidationErrors) Error() string {
	parts := make([]string, 0, len(e))
	for _, fe := range e {
		parts = append(parts, fe.Error())
	}
	return strings.Join(parts, "; ")
}

// Warning is a non-blocking notice (e.g. unknown future field, ignored).
type Warning struct {
	File   string `json:"file"`
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

var (
	siteIDPattern   = regexp.MustCompile(`^[a-z0-9_.-]+$`)
	servicePattern  = regexp.MustCompile(`^[a-z0-9_.-]+$`)
	hostnamePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*$`)
)

// ValidSiteID reports whether id is safe to map to sites/<id>.yaml.
func ValidSiteID(id string) bool {
	return id != "" && siteIDPattern.MatchString(id)
}
