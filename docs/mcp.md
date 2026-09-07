# MCP

The inventory exposes its datastore over the [Model Context Protocol](https://modelcontextprotocol.io) so Claude (and any other MCP client) can read and write models.

## Transport

Streamable HTTP, mounted at `/mcp` on the same chi router as the REST API. Wired in `cmd/server/main.go`:

```go
mcpHandler := mcpsdk.NewStreamableHTTPHandler(
    func(*http.Request) *mcpsdk.Server { return mcpServer },
    nil,
)
r.Handle("/mcp", mcpHandler)
r.Handle("/mcp/*", mcpHandler)
```

Running one server means MCP and REST share the same pgx pool, tenant model, and logging.

## SDK

[`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) — the official Go SDK from the MCP maintainers.

## Where things live

- `internal/mcp/mcp.go` — server construction, shared `View` types, view converters.
- `internal/mcp/read.go` — read-only tools (list, get).
- `internal/mcp/write.go` — mutating tools (create, update, delete).

The split is by mutation semantics, not resource. A future auth layer can gate `write.go` more tightly than `read.go`.

## Tool contract

Each tool is a Go function of the form:

```go
func(ctx context.Context, req *mcpsdk.CallToolRequest, in Input) (
    *mcpsdk.CallToolResult, Output, error,
)
```

Both `Input` and `Output` are plain Go structs. JSONSchema is inferred from struct tags:

```go
type CreateKindInput struct {
    TenantID    string  `json:"tenantId" jsonschema:"tenant xid"`
    Name        string  `json:"name" jsonschema:"kind name"`
    Description *string `json:"description,omitempty" jsonschema:"optional description"`
}
```

The `jsonschema` tag becomes the field description Claude sees. Pointer types and `omitempty` mark fields optional.

## Registered tools

All tools operate within the tenant resolved from the caller's API key. `tenantId` is never a tool argument — see [Auth](#auth) below.

Read (in `read.go`):
- `whoami` — returns the caller's tenant
- `list_kinds`
- `list_kind_versions` — `(kindId)`
- `list_models` — `(kindId)`
- `get_model` — `(kindId, slug)`

Write (in `write.go`):
- `create_kind` — `(name, description?)`
- `create_kind_version` — `(kindId, schema)`
- `create_model` — `(kindId, slug, body, kindVersionId?)`
- `update_model` — `(kindId, slug, body, kindVersionId?)`
- `delete_model` — `(kindId, slug)`

Creating a *tenant* is deliberately not an MCP tool — it is an admin operation done via `./server bootstrap` (see [Auth](#auth)).

When `kindVersionId` is omitted on model create/update, the tool resolves the latest version for the kind so Claude does not need to enumerate versions first.

## Adding a tool

1. Add Input/Output structs in the appropriate file (`read.go` for queries, `write.go` for mutations).
2. Register the tool inside `registerReadTools` or `registerWriteTools` with `mcpsdk.AddTool`.
3. The handler calls `dbgen.Querier` directly and converts rows via the shared view helpers in `mcp.go`.
4. If the tool returns a domain entity that other tools also return, extend the shared `*View` structs rather than adding a parallel one.

## Testing locally

- Connect Claude Desktop to `http://localhost:8080/mcp` via its MCP settings.
- Or use [`mcp inspector`](https://github.com/modelcontextprotocol/inspector) pointed at the same URL.
- Or write a Go client with the SDK's `Client` type and call `session.CallTool(...)`.

## Auth

Every request to `/mcp` (and to the REST API) requires `Authorization: Bearer <key>`.

Keys are minted by the bootstrap CLI, which is also the only way to create a new tenant:

```sh
make bootstrap NAME="Kev"
# tenant_id: c8xyz...
# api_key:   inv_5b2q...       (save now — never shown again)
```

Under the hood: `internal/server/middleware/auth.go` parses the bearer token, SHA-256s it, looks it up in `api_keys.key_hash`, and calls `auth.WithTenant(ctx, tenantID, apiKeyID)`. The chi router's context propagates down into MCP tool handlers, so tools call `auth.TenantID(ctx)` to know who they are serving.

Notes:

- Keys are never stored in plaintext. `key_hash` is the only column with the secret material.
- The bearer prefix is `inv_`. Rotation is a matter of inserting a new row and soft-deleting the old one (a `rotate` subcommand can land later).
- REST routes with a `{tenantId}` path segment additionally require that segment to match the authenticated tenant, or the middleware returns 403.
- `/health` is the only unauthenticated endpoint.

Configuring Claude Desktop or `claude.ai/code` to reach the server: point at `http://localhost:8080/mcp` (or your deployed URL) and paste the raw key from `make bootstrap` into the connector's auth field.
