# Artifact Generation Specification

## Purpose

Defines how the two derived artifacts — the Caddyfile and the compose file — are rendered deterministically, previewed without side effects, and written to disk on demand.

## Requirements

### Requirement: Deterministic Caddyfile rendering
The system SHALL render all sites into a single `Caddyfile` with an optional global options block (ACME email) followed by one site block per site, ordered by the site's first domain, with domains inside a block sorted.

#### Scenario: Ordering is stable
- **WHEN** generation runs over the same set of sites in any input order
- **THEN** the resulting `Caddyfile` bytes are identical

#### Scenario: Site blocks match presets
- **WHEN** the project contains one site of each preset
- **THEN** the `Caddyfile` contains a `file_server` block, an SPA fallback to `index.html`, a `reverse_proxy` line, and a `php_fastcgi` line respectively

### Requirement: Deterministic compose rendering
The system SHALL render a single compose file with a `caddy` service (Caddy image, ports 80/443, Caddyfile mount, site data mounts, persistent `caddy_data`/`caddy_config` volumes) plus one service per image-only backend, ordered by service name.

#### Scenario: Static site needs no backend service
- **WHEN** compose is generated for a project with only `static` sites
- **THEN** the compose file contains only the `caddy` service and no backend services

#### Scenario: Proxy and PHP backends appear
- **WHEN** compose is generated for `proxy` (image node:20-alpine) and `php` (image php:8.3-fpm) sites
- **THEN** the compose file contains the corresponding backend services with their images and env, and caddy `depends_on` them

### Requirement: Preview without side effects
Preview operations SHALL render both artifacts in memory and MUST NOT write any file to disk.

#### Scenario: Preview leaves disk untouched
- **WHEN** preview is requested after editing a site
- **THEN** the response contains the full rendered `Caddyfile` and compose content while no file mtime on disk changes

### Requirement: Explicit generate writes artifacts
The generate operation SHALL write `generated/Caddyfile` and the compose file atomically and report their paths plus whether Caddy formatting was applied.

#### Scenario: Generate writes both files
- **WHEN** generate is invoked on a valid project
- **THEN** both artifact files exist on disk with the previewed content and the response reports their paths

#### Scenario: Generate on invalid project writes nothing
- **WHEN** generate is invoked while any site fails validation
- **THEN** no artifact file is modified and per-file errors are returned

### Requirement: Caddy formatting integration
When a `caddy` binary is available in PATH, generation SHALL pipe the rendered Caddyfile through `caddy fmt`; when absent, generation SHALL succeed with raw output and report formatting as skipped.

#### Scenario: Missing caddy binary still generates
- **WHEN** no `caddy` binary is found
- **THEN** generation succeeds and reports formatting skipped rather than failing

### Requirement: Optional Caddy validation
When a `caddy` binary is available, the system SHALL offer a validate operation that checks the rendered Caddyfile and returns its diagnostics; when absent, validate SHALL report itself unavailable.

#### Scenario: Invalid output is caught when possible
- **WHEN** validate runs with a `caddy` binary present and the rendered file is invalid
- **THEN** the diagnostics from the binary are returned to the caller
