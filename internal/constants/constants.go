// Package constants holds the single-definition project-wide constants
// (file names, conventions, defaults) chosen during planning. Every other
// package MUST reference these instead of duplicating literals.
package constants

const (
	// SchemaVersion is the current caddy.project.yaml / sites/*.yaml schema.
	SchemaVersion = 1

	// On-disk layout.
	ProjectFileName = "caddy.project.yaml"
	SitesDir        = "sites"
	GeneratedDir    = "generated"
	DataDir         = "data"

	// Generated artifact file names (inside GeneratedDir).
	// Open question resolved: compose.yaml (not docker-compose.yaml).
	CaddyfileName   = "Caddyfile"
	ComposeFileName = "compose.yaml"

	// Site file suffix.
	SiteFileSuffix = ".yaml"

	// DefaultCaddyImage is the image used for the caddy service.
	// Open question resolved: floating 2-alpine tag; users pin backend
	// images themselves via their site files.
	DefaultCaddyImage = "caddy:2-alpine"

	// DefaultPort is the default loopback port for `serve`.
	DefaultPort = 8080

	// DefaultBind is the default bind address for `serve` (loopback only).
	DefaultBind = "127.0.0.1"

	// PHPFastCGIPort is the port PHP-FPM listens on inside compose networks.
	PHPFastCGIPort = 9000

	// CaddyDataVolume and CaddyConfigVolume persist caddy state.
	CaddyDataVolume   = "caddy_data"
	CaddyConfigVolume = "caddy_config"
)

// ServiceNameFor returns the compose service name for a site: the explicit
// backend service when set, otherwise the site id.
// Open question resolved: strict default (= id), editable in v2.
func ServiceNameFor(siteID, backendService string) string {
	if backendService != "" {
		return backendService
	}
	return siteID
}
