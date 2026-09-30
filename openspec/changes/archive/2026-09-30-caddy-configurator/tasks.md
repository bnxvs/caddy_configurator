# Tasks

## 1. Scaffolding

- [x] 1.1 Create Go module, cobra CLI skeleton (`init/generate/validate/serve`), core/web package layout, and verify `go build ./...` succeeds
- [x] 1.2 Scaffold React+TS frontend via vite under `web/`, wire `embed.FS` dist serving with dev-proxy mode flag, and verify `npm run build` output is served by the Go server locally
- [x] 1.3 Fix open-question constants (generated compose filename, default Caddy image tag, default service-name rule) behind named constants and verify they are referenced from a single place each

## 2. Core config and validation

- [x] 2.1 Implement v1 YAML schema (project + four presets + common fields + image-only backend) with strict decode, `version` gate, and warn-on-unknown fields, and verify unit tests cover valid files for all presets
- [x] 2.2 Implement per-file/per-field validation (domains, tls enum, required preset fields, backend image/port) with `{file, field, reason}` errors, and verify unit tests cover each rejection scenario in the site-config spec
- [x] 2.3 Implement atomic file writes (temp + rename) for site and project saves, and verify a simulated mid-write failure leaves the previous valid file intact
- [x] 2.4 Document the v1 file layout and schema in `docs/schema.md` and verify every example in it loads and validates as written

## 3. Artifact rendering

- [x] 3.1 Implement Caddyfile renderer (global block + per-preset partials, sorted sites/domains) and verify golden-file tests for one site per preset plus multi-domain ordering
- [x] 3.2 Implement compose renderer (caddy service + image-only backends, shared php volume, depends_on, sorted services) and verify golden-file tests for static-only and proxy+php projects
- [x] 3.3 Implement in-memory preview (no disk writes) and explicit generate (atomic writes of both artifacts with path report), and verify preview leaves mtimes untouched while generate writes both files
- [x] 3.4 Add determinism test (render twice, byte-identical) and verify it passes in CI

## 4. Caddy binary integration

- [x] 4.1 Pipe rendered Caddyfile through `caddy fmt` when the binary is in PATH, report formatting applied/skipped, and verify tests cover both present and absent binary via a fake PATH
- [x] 4.2 Add validate path (schema + `caddy validate` diagnostics when available, unavailable report otherwise) and verify tests cover invalid output surfaced and missing-binary behavior

## 5. HTTP API and disk awareness

- [x] 5.1 Implement JSON endpoints (project, site CRUD, preview, generate, validate, status) with the shared error shape, and verify contract tests for each endpoint including side-effect-free preview
- [x] 5.2 Enforce site-id sanitization (`[a-z0-9-_.]+`, root-joined paths) and loopback default bind, and verify traversal ids are rejected and nothing outside root is touched
- [x] 5.3 Implement changed-on-disk detection (mtime/hash vs last known, watcher hint with polling fallback) in status, and verify an external edit is reported without overwriting open state

## 6. React UI

- [x] 6.1 Build project overview (name, root, caddy presence, site list, preview/generate actions) and verify it renders from the project endpoint against a fixture project
- [x] 6.2 Build preset-driven site editor (per-preset fields only, no upstream/fastcgi inputs, per-site snippet, inline field errors, failed save preserves file) and verify for all four presets
- [x] 6.3 Build full preview view, global email section, and disk-change reload banner, and verify preview matches API output and banner appears on reported external change

## 7. CLI commands

- [x] 7.1 Implement `init` (skeleton creation, refuse existing project) and verify it creates a valid project and fails closed on clash
- [x] 7.2 Implement headless `generate` and `validate` (exit codes, no writes on failure) and verify both on valid and invalid fixture projects
- [x] 7.3 Implement `serve` (single root, configurable loopback port, embedded assets + dev-proxy mode) and verify UI+API respond and dev mode proxies assets
- [x] 7.4 Document CLI usage and single-binary release build in `docs/cli.md` and verify every documented command runs as written

## 8. Integration and release

- [x] 8.1 Run end-to-end flow (init, add one site per preset via API, preview, generate, re-generate byte-identical, git diff review) and verify clean deterministic diff
- [x] 8.2 Verify single-binary release artifact runs `generate` and `serve` on a machine image without Node/toolchain (caddy optional)
