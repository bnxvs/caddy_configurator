package config

import (
	"strings"
	"testing"
)

func checkErr(t *testing.T, file string, site Site, wantField, wantSub string) {
	t.Helper()
	errs := ValidateSite(file, site)
	for _, e := range errs {
		if e.File == file && e.Field == wantField && strings.Contains(e.Reason, wantSub) {
			return
		}
	}
	t.Fatalf("ValidateSite(%+v) = %v, want field %q containing %q", site, errs, wantField, wantSub)
}

func validSite() Site {
	return Site{
		ID: "web", Version: 1,
		Domains: []string{"example.com"}, Preset: PresetStatic,
		TLS: TLSAuto, Log: true, Root: "/var/www/html",
	}
}

func TestValidationRejections(t *testing.T) {
	proxy := func() Site {
		return Site{
			ID: "api", Version: 1, Domains: []string{"api.example.com"},
			Preset: PresetProxy, TLS: TLSAuto, Log: true,
			Backend: &Backend{Service: "api", Image: "node:20-alpine", InternalPort: 3000},
		}
	}
	cases := []struct {
		name   string
		mutate func(*Site)
		field  string
		sub    string
	}{
		{"empty domains", func(s *Site) { s.Domains = nil }, "domains", "at least one"},
		{"bad hostname", func(s *Site) { s.Domains = []string{"not a host!"} }, "domains[0]", "invalid hostname"},
		{"empty domain", func(s *Site) { s.Domains = []string{""} }, "domains[0]", "must not be empty"},
		{"bad tls", func(s *Site) { s.TLS = "strict" }, "tls", "auto or off"},
		{"bad preset", func(s *Site) { s.Preset = "wordpress" }, "preset", "unknown preset"},
		{"static no root", func(s *Site) { s.Root = "" }, "root", "required"},
		{"relative root", func(s *Site) { s.Root = "var/www" }, "root", "absolute"},
		{"backend on static", func(s *Site) { s.Backend = &Backend{Image: "x"} }, "backend", "not used"},
		{"root on proxy", func(s *Site) {
			*s = proxy()
			s.Root = "/x"
		}, "root", "not used"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := validSite()
			c.mutate(&s)
			checkErr(t, "sites/web.yaml", s, c.field, c.sub)
		})
	}

	t.Run("proxy without backend", func(t *testing.T) {
		s := proxy()
		s.Backend = nil
		checkErr(t, "sites/api.yaml", s, "backend", "required")
	})
	t.Run("proxy backend without image", func(t *testing.T) {
		s := proxy()
		s.Backend.Image = ""
		checkErr(t, "sites/api.yaml", s, "backend.image", "must not be empty")
	})
	t.Run("proxy backend without port", func(t *testing.T) {
		s := proxy()
		s.Backend.InternalPort = 0
		checkErr(t, "sites/api.yaml", s, "backend.internalPort", "required")
	})
	t.Run("proxy port out of range", func(t *testing.T) {
		s := proxy()
		s.Backend.InternalPort = 99999
		checkErr(t, "sites/api.yaml", s, "backend.internalPort", "required")
	})
	t.Run("bad service name", func(t *testing.T) {
		s := proxy()
		s.Backend.Service = "NOPE"
		checkErr(t, "sites/api.yaml", s, "backend.service", "invalid service name")
	})
	t.Run("php without backend", func(t *testing.T) {
		s := Site{ID: "blog", Version: 1, Domains: []string{"blog.example.com"},
			Preset: PresetPHP, TLS: TLSAuto, Log: true, Root: "/var/www/blog"}
		checkErr(t, "sites/blog.yaml", s, "backend", "required")
	})
	t.Run("php with internalPort", func(t *testing.T) {
		s := Site{ID: "blog", Version: 1, Domains: []string{"blog.example.com"},
			Preset: PresetPHP, TLS: TLSAuto, Log: true, Root: "/var/www/blog",
			Backend: &Backend{Image: "php:8.3-fpm", InternalPort: 9000}}
		checkErr(t, "sites/blog.yaml", s, "backend.internalPort", "only valid for preset proxy")
	})
	t.Run("empty env key", func(t *testing.T) {
		s := proxy()
		s.Backend.Env = map[string]string{"": "x"}
		checkErr(t, "sites/api.yaml", s, "backend.env", "must not be empty")
	})
}

func TestImageBackendAccepted(t *testing.T) {
	s := Site{
		ID: "api", Version: 1, Domains: []string{"api.example.com"},
		Preset: PresetProxy, TLS: TLSAuto, Log: true,
		Backend: &Backend{
			Service: "api", Image: "node:20-alpine", InternalPort: 3000,
			Env: map[string]string{"NODE_ENV": "production"},
		},
	}
	if errs := ValidateSite("sites/api.yaml", s); len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
}

func TestErrorFormat(t *testing.T) {
	e := FieldError{File: "sites/api.yaml", Field: "backend.internalPort", Reason: "bad port"}
	if e.Error() != "sites/api.yaml: backend.internalPort: bad port" {
		t.Fatalf("format = %q", e.Error())
	}
}
