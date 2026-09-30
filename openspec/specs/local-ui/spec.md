# Local UI Specification

## Purpose

Defines the localhost web interface that lets users manage sites through forms and previews without editing YAML or learning Caddy syntax directly.

## Requirements

### Requirement: Localhost-only serving
The web server SHALL bind to loopback (`127.0.0.1`) by default and SHALL NOT listen on all interfaces unless explicitly overridden, with no login required for MVP.

#### Scenario: Default bind is loopback
- **WHEN** the server starts with default options
- **THEN** it listens on `127.0.0.1` and is not reachable via external interfaces

### Requirement: Project overview
The UI SHALL show the project name, root path, Caddy binary presence, site list (domains + preset each), and actions for full preview and generate.

#### Scenario: Sites are listed
- **WHEN** the UI loads a project with three sites
- **THEN** all three appear with their domains and presets

### Requirement: Preset-driven site editor
The site editor SHALL render one form per preset showing only that preset's fields plus the common fields, and SHALL show the per-site rendered Caddy snippet with field-level validation errors next to the form.

#### Scenario: Proxy form hides derived fields
- **WHEN** a user opens a `proxy` site for editing
- **THEN** the form shows image, port, and env inputs but no upstream input, while the snippet shows the derived `reverse_proxy` target

#### Scenario: Field error is shown inline
- **WHEN** saving a site with an invalid field
- **THEN** the save is rejected, the YAML file is untouched, and the error appears next to the offending input

### Requirement: Site CRUD without data loss
Creating, renaming (via id), updating, and deleting sites SHALL map 1:1 to `sites/<id>.yaml` files, and a failed save SHALL never truncate the existing file.

#### Scenario: Failed save preserves file
- **WHEN** an update fails validation
- **THEN** the on-disk YAML still contains the previous valid content

#### Scenario: Delete removes one file
- **WHEN** a user deletes a site
- **THEN** exactly that site's YAML file is removed and other sites are untouched

### Requirement: Global settings editing
The UI SHALL allow editing the project-level global options (ACME email) from a dedicated section.

#### Scenario: Email is saved
- **WHEN** a user sets the ACME email and saves
- **THEN** `caddy.project.yaml` contains the new value and the next preview includes it in global options

### Requirement: Disk-change banner data
The UI SHALL surface the changed-on-disk file list from the status endpoint as a reload prompt instead of auto-merging.

#### Scenario: External edit prompts reload
- **WHEN** the status endpoint reports a site file as changed on disk
- **THEN** the UI shows a banner naming the file with a reload action and does not overwrite the user's open form

### Requirement: JSON API contract
The server SHALL expose versioned-in-path JSON endpoints for project, site CRUD, in-memory preview, explicit generate, validate, and status, with errors in per-file/per-field form.

#### Scenario: Preview endpoint is side-effect free
- **WHEN** the preview endpoint is called repeatedly
- **THEN** each response contains the full rendered artifacts and no file on disk changes

#### Scenario: Generate endpoint writes
- **WHEN** the generate endpoint is called on a valid project
- **THEN** both artifact files are written and the response reports their paths
