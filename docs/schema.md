# Schema v1 — file layout and site reference

A project root looks like this:

```text
my-caddy/
  caddy.project.yaml      # project metadata (source, committed)
  sites/<id>.yaml         # one site per file (source, committed)
  generated/Caddyfile     # derived artifact (committed, never hand-edited)
  generated/compose.yaml  # derived artifact (committed, never hand-edited)
  data/<id>/              # host-side site data (see volume convention)
```

Every example below is tested: `internal/config/doctest_test.go` extracts
each fenced block and requires it to load and validate.

## caddy.project.yaml

```yaml project
version: 1
name: demo
global:
  email: admin@example.com
```

- `version` is required and must be `1`. Newer majors are rejected;
  unknown fields produce warnings (forward tolerance for v2).
- `global.email` is optional; when set it lands in the Caddy global block.

## sites/<id>.yaml — common fields

Every site carries `version: 1`, `domains` (1+, valid hostnames),
`preset` (one of `static`, `spa`, `proxy`, `php`), `tls` (`auto` or `off`),
and `log` (boolean). The file name (without `.yaml`) is the site id and
must match `[a-z0-9_.-]`.

## static — plain files

```yaml site:web
version: 1
domains: [example.com, www.example.com]
preset: static
tls: auto
log: true
root: /var/www/html
```

## spa — single-page app with index fallback

```yaml site:app
version: 1
domains: [app.example.com]
preset: spa
tls: auto
log: true
root: /var/www/app
```

## proxy — reverse proxy to an image backend

`upstream` is derived as `<service>:<internalPort>`; there is no upstream
field. `service` defaults to the site id.

```yaml site:api
version: 1
domains: [api.example.com]
preset: proxy
tls: auto
log: true
backend:
  service: api
  image: node:20-alpine
  internalPort: 3000
  env:
    NODE_ENV: production
```

## php — FastCGI to a PHP-FPM image backend

The FastCGI target is derived as `<service>:9000`; `internalPort` must not
be set for `php`.

```yaml site:blog
version: 1
domains: [blog.example.com]
preset: php
tls: off
log: true
root: /var/www/blog
backend:
  image: php:8.3-fpm
```

## Conventions (MVP, hardcoded)

- Proxy upstreams: `<service>:<internalPort>`; php FastCGI: `<service>:9000`.
- Container path of `root` is exactly the `root` value; the host path is
  always `./data/<id>` (relative to the project root; run compose with
  `--project-directory .` when using `-f generated/compose.yaml`).
- Proxy backends mount no code volumes (code is baked into the image).
- PHP code is mounted into both the caddy service (read-only) and the
  php service at the same container path.
- Reserved for v2 (currently warned-and-ignored): `upstreamOverride`,
  `volumeSource`, `backend.command`, `backend.build.*`, `global.docker.*`.
