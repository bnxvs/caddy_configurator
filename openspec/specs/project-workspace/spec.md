# Project Workspace Specification

## Purpose

Defines how a configurator project is laid out on disk and how a single project folder is opened, versioned, and kept consistent when edited both from the UI and by hand.

## Requirements

### Requirement: Project directory layout
The system SHALL use a fixed on-disk layout for a project root: `caddy.project.yaml` for project metadata, `sites/<id>.yaml` for one site each, `generated/Caddyfile` and `generated/compose.yaml` for derived artifacts, and `data/<id>/` for host-side site data.

#### Scenario: Fresh project layout is recognized
- **WHEN** the server is started with a root containing `caddy.project.yaml` and a `sites/` directory
- **THEN** the project loads without errors and lists every `sites/<id>.yaml` file as a site

#### Scenario: Missing project marker is reported
- **WHEN** the server is started with a root that has no `caddy.project.yaml`
- **THEN** startup fails with an error stating the folder is not a configurator project

### Requirement: Single-folder session model
The system SHALL serve exactly one project root per server run, and all site operations SHALL be scoped to that root with no ability to address paths outside it.

#### Scenario: Site id cannot escape the project
- **WHEN** a request references a site id containing path separators or parent traversal (e.g. `../x`, `a/b`)
- **THEN** the request is rejected with a client error and no file is read or written

#### Scenario: All operations stay inside the root
- **WHEN** any site is created, read, updated, or deleted
- **THEN** only files under `<root>/sites/` are touched and nothing outside the root is modified

### Requirement: Schema versioning with forward tolerance
Project and site files SHALL carry `version: 1`, and the system SHALL reject files with a newer major version while warning (not failing) on unknown optional fields.

#### Scenario: Unknown future field warns
- **WHEN** a site file contains an unknown optional field from a newer schema
- **THEN** the site still loads and a warning naming the field is surfaced

#### Scenario: Newer major version is rejected
- **WHEN** a file declares a major version newer than the supported one
- **THEN** loading fails with an error naming the file and the unsupported version

### Requirement: Atomic writes
The system SHALL persist every file write atomically so a crash or concurrent write never leaves a half-written YAML file.

#### Scenario: Interrupted save leaves previous version intact
- **WHEN** a save operation fails partway through writing
- **THEN** the previous complete version of the file remains on disk and valid

### Requirement: External change detection
The system SHALL detect when site files change on disk outside the UI session and report which files changed instead of silently overwriting them.

#### Scenario: Manual edit is surfaced
- **WHEN** a `sites/*.yaml` file is modified externally while the UI is open
- **THEN** the status endpoint reports that file as changed-on-disk so the UI can prompt for reload

### Requirement: Generated artifacts are committable
The system SHALL treat `generated/Caddyfile` and `generated/compose.yaml` as committed build artifacts reproducible byte-for-byte from the same sources.

#### Scenario: Regeneration is deterministic
- **WHEN** generation runs twice with unchanged sources
- **THEN** both output files are byte-identical between runs
