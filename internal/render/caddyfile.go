package render

import (
	"fmt"
	"sort"
	"strings"
	"text/template"

	"caddy-configurator/internal/config"
	"caddy-configurator/internal/constants"
)

// ServiceName returns the compose/upstream service name for a site.
func ServiceName(s config.Site) string {
	if s.Backend != nil {
		return constants.ServiceNameFor(s.ID, s.Backend.Service)
	}
	return s.ID
}

// UpstreamFor derives the proxy target <service>:<internalPort>.
func UpstreamFor(s config.Site) string {
	return fmt.Sprintf("%s:%d", ServiceName(s), s.Backend.InternalPort)
}

// FastCGIFor derives the php target <service>:9000.
func FastCGIFor(s config.Site) string {
	return fmt.Sprintf("%s:%d", ServiceName(s), constants.PHPFastCGIPort)
}

// sortedDomains returns a sorted copy of the site domains.
func sortedDomains(s config.Site) []string {
	out := append([]string(nil), s.Domains...)
	sort.Strings(out)
	return out
}

// orderedSites returns sites ordered by first domain (tie: id), with domains sorted.
func orderedSites(sites []config.Site) []config.Site {
	out := append([]config.Site(nil), sites...)
	for i := range out {
		out[i].Domains = sortedDomains(out[i])
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].Domains[0], out[j].Domains[0]
		if a != b {
			return a < b
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// siteAddress renders the address line: http:// prefix per domain when TLS is off.
func siteAddress(s config.Site) string {
	domains := sortedDomains(s)
	if s.TLS == config.TLSOff {
		for i := range domains {
			domains[i] = "http://" + domains[i]
		}
	}
	return strings.Join(domains, ", ")
}

var caddyfileTmpl = template.Must(template.New("caddyfile").Parse(`{{if .Email}}{
	email {{.Email}}
}

{{end}}{{range $i, $s := .Sites}}{{if $i}}
{{end}}{{$s.Address}} {
{{$s.Body}}
}
{{end}}`))

var presetTmpl = template.Must(template.New("preset").Parse(`{{define "static"}}	root * {{.Root}}
	file_server{{end}}{{define "spa"}}	root * {{.Root}}
	try_files {path} {path}/ /index.html
	file_server{{end}}{{define "proxy"}}	reverse_proxy {{.Upstream}}{{end}}{{define "php"}}	root * {{.Root}}
	php_fastcgi {{.FastCGI}}
	file_server{{end}}`))

type caddySite struct {
	Address string
	Body    string
}

// RenderCaddyfile renders the whole project deterministically.
func RenderCaddyfile(p *config.Project) string {
	var buf strings.Builder
	type data struct {
		Email string
		Sites []caddySite
	}
	d := data{Email: strings.TrimSpace(p.Meta.Global.Email)}
	for _, s := range orderedSites(p.Sites) {
		var body strings.Builder
		tmplData := map[string]string{
			"Root":     s.Root,
			"Upstream": "",
			"FastCGI":  "",
		}
		if s.Preset == config.PresetProxy {
			tmplData["Upstream"] = UpstreamFor(s)
		}
		if s.Preset == config.PresetPHP {
			tmplData["FastCGI"] = FastCGIFor(s)
		}
		if err := presetTmpl.ExecuteTemplate(&body, s.Preset, tmplData); err != nil {
			panic("render: unknown preset " + s.Preset)
		}
		if s.Log {
			body.WriteString("\n\tlog")
		}
		d.Sites = append(d.Sites, caddySite{Address: siteAddress(s), Body: body.String()})
	}
	if err := caddyfileTmpl.Execute(&buf, d); err != nil {
		panic("render: " + err.Error())
	}
	return buf.String()
}

// RenderSiteSnippet renders a single site block (for UI preview).
func RenderSiteSnippet(s config.Site) string {
	p := &config.Project{Sites: []config.Site{s}}
	out := RenderCaddyfile(p)
	return strings.TrimSuffix(out, "\n")
}
