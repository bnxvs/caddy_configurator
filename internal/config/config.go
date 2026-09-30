// Package config loads, validates and saves v1 project and site YAML files.
//
// Layout: caddy.project.yaml + sites/<id>.yaml are sources,
// generated/Caddyfile + generated/compose.yaml are derived artifacts.
package config
