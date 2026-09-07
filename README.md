# inventory

A general-purpose datastore for AI agents, exposed over REST and the [Model Context Protocol](https://modelcontextprotocol.io).

- Multi-tenant from day one
- Flexible schema — `Kinds` + versioned `KindVersions` let new data types be added without redeploying
- Core primitive: a `Model` — a schematized JSON blob keyed by `(tenant, kind, slug)`
- Claude reaches it over MCP at `/mcp`

## Stack

Go · chi · Postgres · sqlc · oapi-codegen · dbmate · Vite · React

## Quick start

Prereqs: Go 1.23+, Postgres, [dbmate](https://github.com/amacneil/dbmate) (`brew install dbmate`), Node 18+ for the web UI.

```sh
export DATABASE_URL=postgres://localhost/inventory?sslmode=disable
cd web && npm install && cd ..
make dev
```

`make dev` runs migrations, then starts the API on `:8080` and the Vite dev server on `:5173`. Ctrl+C stops both.

Everything except `/health` requires `Authorization: Bearer <key>`. Mint the first tenant + key with:

```sh
make bootstrap NAME="Kev"
```

MCP is mounted at `http://localhost:8080/mcp`. REST is at `http://localhost:8080/tenants/...`.

## Repo layout

```
api/                    OpenAPI 3 spec — source of truth for the HTTP surface
cmd/server/             the Go binary (API + MCP in one process)
db/migrations/          dbmate SQL migrations
db/queries/             sqlc SQL queries
internal/api/gen/       oapi-codegen output — DO NOT EDIT
internal/db/gen/        sqlc output — DO NOT EDIT
internal/server/        HTTP handlers built on the Endpoint MVVM pattern
internal/mcp/           MCP tools (read + write)
web/                    Vite + React + TypeScript UI
```

## Docs

- [Architecture](docs/architecture.md) — packages, request flow, code-gen boundaries
- [Adding a resource](docs/adding-a-resource.md) — end-to-end recipe
- [MCP](docs/mcp.md) — transport, tools, adding new ones
- [CLAUDE.md](CLAUDE.md) — orientation for Claude Code sessions

## Make targets

| target         | what it does                                                  |
| -------------- | ------------------------------------------------------------- |
| `make dev`               | migrate, then run API + web dev server together               |
| `make bootstrap NAME=…`  | create a tenant and its first api key (prints raw key once)   |
| `make generate`          | regenerate sqlc + oapi-codegen (after any spec or SQL change) |
| `make migrate`           | `dbmate up`                                                   |
| `make run`               | just the API server                                           |
| `make build`             | compile the API binary to `bin/server`                        |

## Deploy target

fly.io.
