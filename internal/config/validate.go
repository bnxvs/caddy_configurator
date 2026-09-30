package config

import (
	"fmt"
	"strings"
)

// ValidateSite checks a decoded site against the v1 rules and returns all
// problems in {file, field, reason} form. An empty result means valid.
func ValidateSite(file string, s Site) ValidationErrors {
	var errs ValidationErrors
	add := func(field, reason string) {
		errs = append(errs, FieldError{File: file, Field: field, Reason: reason})
	}

	// Common fields.
	if len(s.Domains) == 0 {
		add("domains", "at least one domain is required")
	}
	for i, d := range s.Domains {
		field := fmt.Sprintf("domains[%d]", i)
		if strings.TrimSpace(d) == "" {
			add(field, "domain must not be empty")
			continue
		}
		if !hostnamePattern.MatchString(d) || len(d) > 253 {
			add(field, fmt.Sprintf("invalid hostname %q", d))
		}
	}
	switch s.Preset {
	case PresetStatic, PresetSPA, PresetProxy, PresetPHP:
	default:
		add("preset", fmt.Sprintf("unknown preset %q: expected one of static, spa, proxy, php", s.Preset))
	}
	switch s.TLS {
	case TLSAuto, TLSOff:
	default:
		add("tls", fmt.Sprintf("invalid tls mode %q: expected auto or off", s.TLS))
	}

	// Preset-specific fields.
	switch s.Preset {
	case PresetStatic, PresetSPA, PresetPHP:
		if strings.TrimSpace(s.Root) == "" {
			add("root", fmt.Sprintf("root is required for preset %q (container path, e.g. /var/www/html)", s.Preset))
		} else if !strings.HasPrefix(s.Root, "/") {
			add("root", fmt.Sprintf("root %q must be an absolute container path", s.Root))
		}
	case PresetProxy:
		if s.Root != "" {
			add("root", "root is not used for preset proxy: content comes from the backend image")
		}
	}

	// Backend rules.
	switch s.Preset {
	case PresetProxy:
		if s.Backend == nil {
			add("backend", "backend is required for preset proxy")
		} else {
			validateBackend(file, s.Backend, true, add)
		}
	case PresetPHP:
		if s.Backend == nil {
			add("backend", "backend is required for preset php")
		} else {
			validateBackend(file, s.Backend, false, add)
		}
	case PresetStatic, PresetSPA:
		if s.Backend != nil {
			add("backend", fmt.Sprintf("backend is not used for preset %q: static content is served from root", s.Preset))
		}
	}
	return errs
}

func validateBackend(file string, b *Backend, needPort bool, add func(field, reason string)) {
	if b.Service != "" && !servicePattern.MatchString(b.Service) {
		add("backend.service", fmt.Sprintf("invalid service name %q: use [a-z0-9_.-]", b.Service))
	}
	if strings.TrimSpace(b.Image) == "" {
		add("backend.image", "backend image must not be empty (e.g. node:20-alpine)")
	} else if strings.ContainsAny(b.Image, " \t\n") {
		add("backend.image", fmt.Sprintf("backend image %q must not contain whitespace", b.Image))
	}
	if needPort {
		if b.InternalPort < 1 || b.InternalPort > 65535 {
			add("backend.internalPort", "internalPort is required for preset proxy (1-65535)")
		}
	} else if b.InternalPort != 0 {
		add("backend.internalPort", "internalPort is only valid for preset proxy: php always uses 9000")
	}
	for k := range b.Env {
		if strings.TrimSpace(k) == "" {
			add("backend.env", "env var name must not be empty")
		}
	}
}
