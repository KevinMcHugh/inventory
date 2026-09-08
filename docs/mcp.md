# MCP

The inventory exposes its datastore over the [Model Context Protocol](https://modelcontextprotocol.io) so Claude (and any other MCP client) can read and write models.

## Transport

Streamable HTTP, mounted at `/mcp/rpc` on the same chi router as the REST API. Wired in `cmd/server/main.go`:

```go
mcpHandler := mcpsdk.NewStreamableHTTPHandler(
    func(*http.Request) *mcpsdk.Server { return mcpServer },
    nil,
)
r.Handle("/mcp/rpc", mcpHandler)
r.Handle("/mcp/rpc/*", mcpHandler)
```

The subpath is deliberate: fly.io's edge / sprite proxy intercepts the bare `/mcp` path and hangs POSTs against it before they reach the app. `/mcp/rpc` routes cleanly.

Running one server means MCP and REST share the same pgx pool, tenant model, and logging.

## SDK

[`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) — the official Go SDK from the MCP maintainers.

## Where things live

- `internal/mcp/mcp.go` — server construction only. Delegates tool registration to `read.go` and `write.go`.
- `internal/mcp/read.go` — read-only tools (list, get, whoami).
- `internal/mcp/write.go` — mutating tools (create, update, delete).

The split is by mutation semantics, not resource. A future auth layer can gate `write.go` more tightly than `read.go`.

### Shared view types

MCP does not maintain a parallel view layer. Tool Output structs embed the endpoint packages' view-model types directly — `tenants.ViewModel`, `kinds.ViewModel`, `kinds.VersionViewModel`, `models.ViewModel` — and use the exported `ToViewModel` / `ToVersionViewModel` converters. Add a new field to a view-model in `internal/server/<resource>/` and it appears in MCP responses on the next request; there is no second place to update.

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

- Connect Claude Desktop to `http://localhost:8080/mcp/rpc` via its MCP settings.
- Or use [`mcp inspector`](https://github.com/modelcontextprotocol/inspector) pointed at the same URL.
- Or write a Go client with the SDK's `Client` type and call `session.CallTool(...)`.

## Auth

Every request to `/mcp/rpc` (and to the REST API) requires `Authorization: Bearer <token>`.

Two token flavors are accepted:

- **Raw api keys** with prefix `inv_` — minted by the bootstrap and `keys` subcommands. Best for curl, scripts, and Claude Desktop where you can paste a header.
- **OAuth access tokens** with prefix `inv_at_` — issued by the built-in OAuth 2.1 authorization server. Required for Claude.ai custom connectors, which only speak OAuth.

### Minting the first api key

The bootstrap CLI is the only way to create a tenant and its first key:

```sh
make bootstrap NAME="Kev"
# tenant_id: c8xyz...
# api_key:   inv_5b2q...       (save now — never shown again)
```

Under the hood: `internal/server/middleware/auth.go` parses the bearer token, SHA-256s it, looks it up in `api_keys.key_hash` (or `oauth_tokens.token_hash` for `inv_at_` tokens), and calls `auth.WithTenant(ctx, tenantID, apiKeyID)`. The chi router's context propagates down into MCP tool handlers, so tools call `auth.TenantID(ctx)` to know who they are serving.

### OAuth 2.1 flow

Implemented in `internal/oauth/`. Endpoints:

- `GET /.well-known/oauth-protected-resource` — RFC 9728 resource-server metadata; points at this issuer as the AS.
- `GET /.well-known/oauth-authorization-server` — RFC 8414 authorization-server metadata.
- `POST /oauth/register` — RFC 7591 dynamic client registration (public clients, PKCE-only).
- `GET /oauth/authorize` — a small HTML form asking the user to paste an `inv_` key.
- `POST /oauth/authorize` — validates the pasted key, mints a code, redirects to `redirect_uri`.
- `POST /oauth/token` — PKCE-validated code → access token.

The `WWW-Authenticate: Bearer resource_metadata=…` header on 401 tells conforming clients where discovery lives. `PUBLIC_URL` env var supplies the issuer origin so those URLs point at the right host.

### Configuring Claude

**claude.ai custom connector:**
- URL: `https://<host>/mcp/rpc`
- Authentication: **Always required**
- OAuth client: **No client ID — register one automatically** (DCR)
- On first connect, Claude bounces the user through `/oauth/authorize` where they paste an `inv_` key. That approval issues an `inv_at_` access token bound to the caller's tenant, good for 24 hours.

**Claude Desktop (raw key path):**
- URL: `https://<host>/mcp/rpc`
- Header: `Authorization: Bearer inv_...` — the plain api key from bootstrap.

### Key management

- Keys are never stored in plaintext. `key_hash` (api keys) and `token_hash` (OAuth) are the only columns with secret material.
- Rotate an api key: `./server keys rotate --key-id <xid>` (or `make keys-rotate KEY_ID=…`). Old key stops working immediately, new raw key printed once.
- List keys: `./server keys list --tenant <xid>`. Mint more: `./server keys create --tenant <xid> --name <label>`.
- OAuth access tokens expire on their own (24h). No separate revocation CLI yet — planned.
- `/health` and the OAuth discovery + flow endpoints are the only unauthenticated paths.
