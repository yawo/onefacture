# Contributing to onefacture

Thanks for your interest in helping democratise French e-invoicing. This guide
covers what you need to get a local dev environment running and how we work.

## Prerequisites

- Go 1.23+
- Python 3.12+ (for the validation sidecar)
- Docker + Docker Compose
- `golangci-lint` and `golang-migrate` if you want to run linting/migrations outside Docker

## Local dev

```bash
git clone https://github.com/yawo/onefacture.git
cd onefacture
make dev          # spins up postgres + redis + sidecar + api
make migrate-up   # apply DB migrations (uses ONEFACTURE_DB_DSN or default)
```

The API lives on `http://localhost:8080`, the Scalar docs on
`http://localhost:8080/docs`, and the Python sidecar on `:8081`.

## Tests

```bash
make test               # unit tests + coverage
make test-integration   # integration tests (require docker)
make lint               # golangci-lint
```

The CI gate is **60 %** unit-test coverage today (climbing to 80 % as adapters
get wired up).

## Repo layout

```
cmd/api              Gateway entrypoint
cmd/genopenapi       Print the OpenAPI spec to stdout
internal/
  config             Env-loaded runtime config
  core/invoice       Unified domain model (EN 16931)
  core/facturx       CII XML + PDF/A-3 packaging
  validation         Validation pipeline + sidecar client
  storage            Postgres repos + migrations
  events             Redis Streams event bus
  webhooks           Outbound webhook deliverer
  workers            Background pollers
  adapters           PAAdapter interface + concrete adapters
  gateway            HTTP router, middleware, routes, OpenAPI
sidecar              Python (FastAPI + lxml) validation service
deploy               Docker, Helm, compose
docs/specs           Official XSDs / Schematron / DGFiP YAML
sdks                 SDK scaffolding (python, typescript)
```

## Coding rules

See `AGENTS.md` § 7 for the full list. The most important ones:

- Wrap errors with context: `fmt.Errorf("operation: %w", err)`.
- All env vars are prefixed `ONEFACTURE_`.
- Errors over the wire follow RFC 7807 (see `internal/gateway/problem`).
- Multi-tenant data must be scoped by `organization_id`.
- Lifecycle transitions go through `invoice.Transition` so the state machine
  rules are enforced consistently.

## Adding a PA adapter

1. Create `internal/adapters/<name>/<name>.go` implementing `adapters.PAAdapter`.
2. Register it in `internal/adapters/registry/registry.go`.
3. Map PA-specific lifecycle codes to canonical `invoice.Status` values in
   `Submit`, `GetStatus`, and `Webhook`.
4. Add an integration test gated by `//go:build integration` and an env var
   carrying sandbox credentials.

## Reporting issues / bug bounty

Security issues: please email `security@onefacture.io` rather than opening a
public issue.
