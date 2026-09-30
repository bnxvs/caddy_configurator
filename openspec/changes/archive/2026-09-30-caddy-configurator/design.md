# Design

## Context

Greenfield project: repo contains only `openspec/` scaffolding, no code and no existing specs. See `proposal.md` for motivation. Constraints fixed during exploration: single project root per run (P1), files on disk as source of truth (no hidden DB), git-friendly deterministic output, Go backend + React SPA in one binary, image-only Docker backends, hard conventions in MVP with `version: 1` forward tolerance for later overrides.

## Goals / Non-Goals

**Goals:**
- One Go binary: CLI (`init/generate/validate/serve`) and web UI share the same core (config load/save/validate + render + caddy exec).
- Source/artifact split: `sites/*.yaml` are hand- and UI-editable sources; `generated/` files are pure function output.
- Fast UI loop: in-memory preview per keystroke-cost (explicit fetch, no autosave-to-disk), explicit generate writes.
- Safe coexistence with manual editing: atomic writes + changed-on-disk detection, never silent overwrite.

**Non-Goals:**
- No auth/multi-user, no multi-project registry, no git automation (files only).
- No build-context backends, no arbitrary compose escape hatches, no dev/prod compose variants.
- No auto-merge of concurrent edits; no remote deployment or `caddy reload` on servers.

## Decisions

### 1. Core packages shared by CLI and web
`core/config` (YAML load/save, strict validation, version gate), `core/render` (Caddyfile + compose rendering, pure functions over the model), `core/caddy` (exec `caddy fmt/validate` when present). `cli` (cobra) and `web` (net/http handlers) are thin shells over core. Rationale: guarantees CLI and UI can never diverge in output; enables `generate` in CI. Alternative (logic duplicated in handlers) rejected: two truths, drift.

### 2. JSON model in API, YAML on disk
API speaks a JSON site model; the server translates to/from YAML deterministically (sorted keys, stable field order). Rationale: React forms bind to JSON naturally; YAML stays canonical on disk. Alternative (API shuttles raw YAML text) rejected: pushes parsing/validation into the frontend.

### 3. Per-preset template partials
One small partial per preset (`static`, `spa`, `proxy`, `php`) plus shared partials (site address line, `tls`, `log`), composed in sorted site order via `text/template`. Compose rendering mirrors this: `caddy` service partial + one backend partial per site with a backend. Rationale: adding a preset or a future optional field touches one partial, not a forest of if-else. Alternative (single giant template) rejected: unmaintainable past two presets.

### 4. Strict decode + warn-on-unknown
YAML decoded strictly against the v1 schema for known fields (type/port/hostname checks), but unknown mapping keys produce warnings, not errors; `version > 1` major is a hard error. Rationale: MVP stays strict while reserving the extension path (`upstreamOverride`, `build.*`, `global.docker.*`) without breaking old projects. Alternative (fully lenient) rejected: typos would silently pass.

### 5. Preview pure, generate atomic
`preview` calls `render` only. `generate` renders, optionally pipes through `caddy fmt`, then writes each artifact via temp-file + rename. Rationale: preview can never dirty git status; crashes leave previous valid files. Alternative (autosave on edit) rejected: every keystroke would churn `git status`.

### 6. Change detection via mtime + optional watcher
`status` compares in-memory generation input hashes / file mtimes against last known state; `fsnotify` where reliable, polling fallback. UI shows banner, never auto-merges. Rationale: cross-platform reliability for MVP without merge semantics. Alternative (full three-way merge) rejected: out of MVP scope.

### 7. SPA: React+TS (vite) embedded via `embed.FS`, dev proxy mode
`web/` builds to `dist/`, embedded into the binary; `serve --dev` proxies asset requests to the vite dev server while API stays local. Rationale: single-binary distribution with no Node at runtime; dev keeps hot reload. Alternative (server-rendered Go templates) was considered and rejected after the explicit SPA decision: richer preview/diff UX later.

### 8. P1 security posture
Root fixed at startup; `:id` restricted to `[a-z0-9-_.]+`; all file access joined and re-validated under root; default bind `127.0.0.1`. Rationale: even single-user localhost UI must not become a file-read/write gadget. Alternative (open file picker / multi-root) rejected with P1.

### 9. Compose D2 image-only with derived values
`upstream`/`fastcgi` derived (`<service>:<internalPort>`, `<service>:9000`); host path derived (`./data/<id>`); proxy mounts no code; php shares one host dir into caddy (ro) and php (rw); caddy `depends_on` all backends; pinned-in-code `caddy:2-alpine`, ports 80/443, `unless-stopped`, default network. Rationale: minimal fields, no duplication, fully derivable from the v1 schema. Alternative (user-editable upstream/volumes now) rejected for MVP; reserved as v2 optionals.

## Risks / Trade-offs

- [Risk] Caddy syntax drift (new directives) → Mitigation: templates own the syntax; `caddy fmt/validate` passthrough catches breakage when binary present.
- [Risk] `root` duality confusion (host `./data/<id>` vs container `root`) → Mitigation: convention documented in UI labels ("container path" vs auto host mapping); v2 adds `volumeSource` override.
- [Risk] fsnotify flakiness on some OSes → Mitigation: mtime/hash comparison as ground truth, watcher only a hint; polling fallback.
- [Risk] Image tags floating (`node:20-alpine` moves) → Mitigation: MVP keeps user-written tags verbatim (user owns pinning); v2 may add digest/recorded-tag warnings.
- [Risk] SPA/Go type drift → Mitigation: small endpoint surface (5 groups); shared error shape `{file, field, reason}`; contract tests in tasks.
- [Trade-off] Single binary + embedded SPA needs Node at build time → accepted: runtime stays dependency-free, CI builds frontend once.

## Migration Plan

Greenfield: no migration. Rollout: `go build` single binary → `init` a project → `serve` locally → commit sources + generated files. Rollback: generated files are disposable (re-render from sources); sources are plain YAML under git. Schema v1→v2 later: additive optional fields only, old files keep loading.

## Open Questions

- Exact generated compose filename: `generated/compose.yaml` vs `generated/docker-compose.yaml` (deferrable: single constant, no spec impact).
- Whether to pin the default Caddy image tag with minor version (`caddy:2.8-alpine`) or floating `caddy:2-alpine` (deferrable: one constant + docs line).
- Hand-written TS types vs generated client from Go structs/OpenAPI (deferrable: does not change endpoint behavior).
- `service` defaulting rule detail: strict `= id` vs editable service name field (deferrable within v1: propose strict default, add editable in v2).
