# Site Config Specification

## Purpose

Defines the site presets, shared fields, and image-only backend model that let users describe static, SPA, proxy, and PHP sites without writing Caddy syntax by hand.

## Requirements

### Requirement: Site presets
The system SHALL support exactly four MVP presets: `static` (plain files), `spa` (single-page fallback to index), `proxy` (reverse proxy for node/python/go backends), and `php` (FastCGI to PHP-FPM).

#### Scenario: Each preset is selectable
- **WHEN** a user creates a site with preset `static`, `spa`, `proxy`, or `php`
- **THEN** the site is accepted and rendered with that preset's handler block

#### Scenario: Unknown preset is rejected
- **WHEN** a site file declares a preset outside the four supported values
- **THEN** validation fails with an error naming the site and the invalid preset

### Requirement: Common site fields
Every site SHALL declare `domains` (one or more), `tls` (`auto` or `off`), and `log` (on or off), with `domains` non-empty and each entry a syntactically valid hostname.

#### Scenario: Valid common fields pass
- **WHEN** a site declares two domains, `tls: auto`, and logging on
- **THEN** validation passes

#### Scenario: Empty domains fail
- **WHEN** a site declares an empty `domains` list
- **THEN** validation fails with an error pointing at the `domains` field

#### Scenario: Invalid TLS mode fails
- **WHEN** a site declares `tls` with a value other than `auto` or `off`
- **THEN** validation fails with an error naming the allowed values

### Requirement: Preset-specific fields
The system SHALL require `root` for `static`, `spa`, and `php`; `upstream` MUST NOT be a user field for `proxy` and `fastcgi` MUST NOT be a user field for `php` — both are derived from the backend. `proxy` requires a backend with `image` and `internalPort`; `php` requires a backend with `image`.

#### Scenario: Static without root fails
- **WHEN** a `static` site omits `root`
- **THEN** validation fails with an error naming the missing `root` field

#### Scenario: Proxy backend is required
- **WHEN** a `proxy` site omits its backend block
- **THEN** validation fails with an error stating the backend is required

### Requirement: Image-only backends
Backends in MVP SHALL be described by a ready container image plus environment only: `service` name, `image` reference, and for `proxy` an `internalPort`; optional `env` map. Build contexts, custom commands, and extra volumes/ports SHALL NOT be part of the MVP schema.

#### Scenario: Image backend is accepted
- **WHEN** a `proxy` site declares `backend: {service: api, image: node:20-alpine, internalPort: 3000, env: {NODE_ENV: production}}`
- **THEN** validation passes

#### Scenario: Build field is rejected or warned
- **WHEN** a site file contains a `build` key under backend
- **THEN** the loader either rejects it as unsupported or surfaces it as an unknown-field warning and ignores it

### Requirement: Hard addressing conventions
The system SHALL derive `proxy` upstreams as `<service>:<internalPort>` and `php` FastCGI targets as `<service>:9000`, where `service` defaults to the site file id when not given.

#### Scenario: Proxy upstream is derived
- **WHEN** a `proxy` site has backend `service: api` and `internalPort: 3000`
- **THEN** the generated Caddy block reverse-proxies to `api:3000` with no manual upstream field

#### Scenario: PHP FastCGI target is derived
- **WHEN** a `php` site has backend `service: php`
- **THEN** the generated Caddy block uses `php:9000` as the FastCGI address

### Requirement: Volume convention
The container path of a site's `root` SHALL be exactly the configured `root` value, the host path SHALL be `./data/<id>`, proxy backends SHALL mount no code volumes, and `php` code SHALL be mounted into both the caddy service (read-only) and the php service.

#### Scenario: PHP shares code with caddy
- **WHEN** compose is generated for a `php` site with `root: /var/www/blog`
- **THEN** both the caddy service (read-only) and the php service mount host `./data/<id>` at `/var/www/blog`

### Requirement: Human-readable validation errors
Validation failures SHALL be reported per file and per field in the form `<file>: <field>: <reason>` so the UI can show them next to the offending input.

#### Scenario: Error points at file and field
- **WHEN** `sites/api.yaml` has an invalid port
- **THEN** the error names `sites/api.yaml`, the field, and the reason
