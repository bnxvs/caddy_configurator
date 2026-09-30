package render

import (
	"fmt"
	"sort"
	"strings"
	"text/template"

	"caddy-configurator/internal/config"
	"caddy-configurator/internal/constants"
)

// hostDataDir returns the compose host path for a site's data.
func hostDataDir(id string) string {
	return "./" + constants.DataDir + "/" + id
}

type envVar struct {
	Key   string
	Value string
}

type composeBackend struct {
	Name    string
	Image   string
	Env     []envVar
	Volumes []string
	Restart string
}

type composeDataVolume struct {
	Host      string
	Container string
	ReadOnly  bool
}

type composeData struct {
	CaddyImage   string
	Volumes      []composeDataVolume
	DependsOn    []string
	Backends     []composeBackend
	Restart      string
	DataVolume   string
	ConfigVolume string
}

var composeTmpl = template.Must(template.New("compose").Parse(`services:
  caddy:
    image: {{.CaddyImage}}
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./{{.GeneratedDir}}/{{.CaddyfileName}}:/etc/caddy/Caddyfile:ro{{range .Volumes}}
      - {{.Host}}:{{.Container}}{{if .ReadOnly}}:ro{{end}}{{end}}
      - {{.DataVolume}}:/data
      - {{.ConfigVolume}}:/config{{if .DependsOn}}
    depends_on:{{range .DependsOn}}
      - {{.}}{{end}}{{end}}
    restart: {{.Restart}}{{range .Backends}}
  {{.Name}}:
    image: {{.Image}}{{if .Env}}
    environment:{{range .Env}}
      {{.Key}}: {{.Value}}{{end}}{{end}}{{if .Volumes}}
    volumes:{{range .Volumes}}
      - {{.}}{{end}}{{end}}
    restart: {{.Restart}}{{end}}
volumes:
  {{.DataVolume}}:
  {{.ConfigVolume}}:
`))

// RenderCompose renders the compose file deterministically.
func RenderCompose(p *config.Project) string {
	d := composeData{
		CaddyImage:   constants.DefaultCaddyImage,
		Restart:      "unless-stopped",
		DataVolume:   constants.CaddyDataVolume,
		ConfigVolume: constants.CaddyConfigVolume,
	}
	// Caddy data mounts for file-serving presets, ordered by site id.
	for _, s := range orderedSites(p.Sites) {
		switch s.Preset {
		case config.PresetStatic, config.PresetSPA, config.PresetPHP:
			d.Volumes = append(d.Volumes, composeDataVolume{
				Host: hostDataDir(s.ID), Container: s.Root, ReadOnly: true,
			})
		}
	}
	// Backend services, ordered by name.
	byName := map[string]composeBackend{}
	for _, s := range orderedSites(p.Sites) {
		if s.Backend == nil {
			continue
		}
		name := ServiceName(s)
		b := composeBackend{Name: name, Image: s.Backend.Image, Restart: "unless-stopped"}
		for _, k := range sortedKeys(s.Backend.Env) {
			b.Env = append(b.Env, envVar{Key: k, Value: s.Backend.Env[k]})
		}
		if s.Preset == config.PresetPHP {
			b.Volumes = append(b.Volumes, fmt.Sprintf("%s:%s", hostDataDir(s.ID), s.Root))
		}
		byName[name] = b
	}
	for name := range byName {
		d.Backends = append(d.Backends, byName[name])
		d.DependsOn = append(d.DependsOn, name)
	}
	sort.Slice(d.Backends, func(i, j int) bool { return d.Backends[i].Name < d.Backends[j].Name })
	sort.Strings(d.DependsOn)
	var buf strings.Builder
	tmplData := map[string]any{
		"CaddyImage":    d.CaddyImage,
		"GeneratedDir":  constants.GeneratedDir,
		"CaddyfileName": constants.CaddyfileName,
		"Volumes":       d.Volumes,
		"DataVolume":    d.DataVolume,
		"ConfigVolume":  d.ConfigVolume,
		"DependsOn":     d.DependsOn,
		"Backends":      d.Backends,
		"Restart":       d.Restart,
	}
	if err := composeTmpl.Execute(&buf, tmplData); err != nil {
		panic("render: " + err.Error())
	}
	return buf.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
