# CLI Specification

## Purpose

Defines the command-line surface that scaffolds projects, regenerates artifacts without the UI, validates sources, and serves the web interface from a single binary.

## Requirements

### Requirement: Project scaffolding via init
The `init` command SHALL create a minimal valid project skeleton (project file, empty sites directory, gitignore-friendly structure) at the given path.

#### Scenario: Init creates skeleton
- **WHEN** `init ./new-project` runs on an empty directory
- **THEN** the directory contains a valid `caddy.project.yaml` and an empty `sites/` directory

#### Scenario: Init refuses non-empty clash
- **WHEN** `init` targets a directory that already contains a project file
- **THEN** it fails with an error stating the project already exists and writes nothing

### Requirement: Headless generate
The `generate` command SHALL render and write both artifacts from disk sources with no UI, exiting non-zero and writing nothing when validation fails.

#### Scenario: CI regeneration works
- **WHEN** `generate ./my-caddy` runs on a valid project
- **THEN** both artifact files are written and the exit code is zero

#### Scenario: Invalid project fails closed
- **WHEN** `generate` runs on a project with an invalid site
- **THEN** the exit code is non-zero, nothing is written, and errors name file and field

### Requirement: Validate command
The `validate` command SHALL check schema validity of all sources and, when a `caddy` binary is present, the rendered Caddyfile, reporting per-file diagnostics.

#### Scenario: Valid project passes
- **WHEN** `validate` runs on a valid project
- **THEN** the exit code is zero and a success summary is printed

### Requirement: Serve command
The `serve` command SHALL start the web UI for exactly one project root on a configurable loopback port, serving the API and the embedded frontend.

#### Scenario: Serve opens one project
- **WHEN** `serve ./my-caddy --port 8080` runs
- **THEN** the UI and API for that project respond on `127.0.0.1:8080`

### Requirement: Dev frontend mode
The `serve` command SHALL support a dev mode that proxies frontend asset requests to a local dev server instead of serving embedded assets.

#### Scenario: Dev mode proxies assets
- **WHEN** serve runs in dev mode with a dev server URL configured
- **THEN** frontend asset requests are proxied there while API requests are still served locally

### Requirement: Single-binary distribution
Releases SHALL be distributable as a single executable containing the backend and the production frontend assets, requiring only an optional `caddy` binary for formatting/validation extras.

#### Scenario: No runtime dependencies
- **WHEN** the released binary runs `generate` on a machine without Node or a frontend toolchain
- **THEN** generation succeeds with formatting applied only if `caddy` is present
