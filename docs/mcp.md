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

Read (in `read.go`):
- `list_tenants`
- `list_kinds` — `(tenantId)`
- `list_kind_versions` — `(kindId)`
- `list_models` — `(tenantId, kindId)`
- `get_model` — `(tenantId, kindId, slug)`

Write (in `write.go`):
- `create_tenant` — `(name)`
- `create_kind` — `(tenantId, name, description?)`
- `create_kind_version` — `(kindId, schema)`
- `create_model` — `(tenantId, kindId, slug, body, kindVersionId?)`
- `update_model` — `(tenantId, kindId, slug, body, kindVersionId?)`
- `delete_model` — `(tenantId, kindId, slug)`

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

**Not yet implemented.** All requests to `/mcp` currently reach every tool with no credential check, and every tool takes `tenantId` as an explicit argument. Planned next step:

- Bearer token in `Authorization` header → looked up in an `api_keys` table → injected into `context.Context` as a resolved tenant id.
- Tools drop the `tenantId` argument and read it from context.
- The `getServer` callback returns a per-tenant server so Claude only ever sees its own tenant scope.

See the next commit adding auth for concrete wiring.
