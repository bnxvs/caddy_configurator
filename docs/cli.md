# CLI usage and release build

Every `$ caddy-configurator ...` command below is executed verbatim by
`internal/cli/doctest_test.go` (task 7.4 verification). Keep examples
runnable from an empty directory.

## Create a project

```sh
$ caddy-configurator init ./demo
$ caddy-configurator validate ./demo
```

`init` refuses to overwrite an existing `caddy.project.yaml`.
`validate` checks the schema and, when a `caddy` binary is on PATH,
the rendered Caddyfile.

## Render artifacts headlessly

```sh
$ caddy-configurator generate ./demo
$ caddy-configurator validate ./demo
```

`generate` writes `generated/Caddyfile` and `generated/compose.yaml`
atomically (nothing is written when validation fails) and pipes the
Caddyfile through `caddy fmt` when available.

## Serve the local UI

```sh
$ caddy-configurator serve ./demo --port 18080
```

Serves one project on loopback only. Flags: `--port` (default 8080),
`--bind` (default 127.0.0.1), `--dev-frontend <vite-url>` to proxy
frontend assets to a dev server while the API stays local.

Run the generated stack from the project root:

```sh
$ docker compose --project-directory ./demo -f ./demo/generated/compose.yaml up
```

## Release build (single binary)

Node is needed only to build the frontend once; the Go binary embeds it:

```sh
$ npm --prefix web run build
$ go build -o caddy-configurator ./cmd/caddy-configurator
```

The resulting binary needs no Node at runtime. `caddy fmt`/`validate`
integration activates automatically when a `caddy` binary is on PATH.
