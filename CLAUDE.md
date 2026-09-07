# CLAUDE.md

Orientation for Claude Code sessions in this repo. See @README.md for the pitch.

## Never edit

- `internal/api/gen/**` — regenerate with `make generate` after editing `api/openapi.yaml`.
- `internal/db/gen/**` — regenerate with `make generate` after editing `db/queries/*.sql` or a migration.

If you want to change something in a generated file, edit the source (OpenAPI spec, SQL query, or migration) and regenerate. If the generator itself needs tweaking, edit `sqlc.yaml` or `oapi-codegen.yaml`.

## Core invariants

- **IDs are xids.** All IDs are `github.com/rs/xid` values stored as `CHAR(20)`. Create with `xid.New().String()`.
- **Soft delete.** Every table has `created_at`, `updated_at`, `deleted_at`. Every query filters `deleted_at IS NULL`. Deletes set `deleted_at = NOW()`.
- **Tenant scoping.** Everything is under a tenant. `models` has a partial unique index on `(tenant_id, kind_id, slug) WHERE deleted_at IS NULL`.
- **Model bodies flow as `map[string]any` at API/MCP boundaries and `[]byte` (JSONB) in postgres.** The view-model layer marshals both directions.

## The Endpoint pattern

HTTP handlers implement `Endpoint[Req, M, VM, Res]` (see `internal/server/endpoint.go`):

- `Interact(ctx, req) (M, error)` — do the DB work via a narrow Store interface that the endpoint declares itself.
- `Build(M) VM` — pure transformation of the domain model to a view model.
- `Render(VM) Res` — produce a typed oapi-codegen response object.

Each endpoint declares its own `Store` interface with only the querier methods it needs (minimum surface area). `*dbgen.Queries` satisfies all of them because sqlc generates every method. The composed `internal/server.Server` implements `apigen.StrictServerInterface` by delegating each operation through `server.Run(ctx, endpoint, req)`.

## Adding a resource

See @docs/adding-a-resource.md for the full walkthrough. Skeleton:

1. Add path + component schemas to `api/openapi.yaml`.
2. Add SQL to `db/queries/<resource>.sql`. If it needs new tables, add a dbmate migration first (`dbmate new <name>`).
3. `make generate` — regenerates both sqlc and oapi-codegen output.
4. Create `internal/server/<resource>/<resource>.go` with Endpoint structs following the pattern.
5. Wire the methods into `internal/server/server.go`.
6. If Claude should see it, add matching MCP tools in `internal/mcp/read.go` or `write.go`.

## MCP

Mounted at `/mcp` inside the same chi router as the REST API. Tools live in `internal/mcp/read.go` and `write.go`; shared view types in `mcp.go`. See @docs/mcp.md.

## Migrations

`db/migrations/YYYYMMDDHHMMSS_description.sql` in dbmate format (`-- migrate:up` / `-- migrate:down`). Create with `dbmate new <name>`. Never edit an applied migration in place — add a new one.

## Commands

- `make dev` — migrate, then API :8080 + web :5173 together; Ctrl+C stops both
- `make generate` — regenerate sqlc + oapi-codegen (run after any spec or SQL change)
- `make migrate` — dbmate up
- `make run` — just the API
- `make build` — compile the API binary

## Commit style

- One-line subject, present tense, lowercase after the type prefix (`feat:`, `fix:`, `chore:`).
- Body explains *why*, not *what* the diff shows.
- Attribution footers are set by the session guidance in the environment.
- **Avoid apostrophes in commit messages** when composing with the `git commit -m "$(cat <<'EOF' ...)"` heredoc pattern — the outer `$(...)` breaks on unbalanced single quotes. Say "do not" instead of "don't".

## Push cadence

Push after each self-contained commit. The repo lives at `git@github.com:KevinMcHugh/inventory.git`.
