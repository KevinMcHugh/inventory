# Adding a resource, end to end

A worked recipe for adding a new CRUD resource. Uses `widget` as the example.

## 1. Add the schema

Create a migration:

```sh
dbmate new create_widgets
```

Edit `db/migrations/<timestamp>_create_widgets.sql`:

```sql
-- migrate:up

CREATE TABLE widgets (
    id         CHAR(20)    PRIMARY KEY,
    tenant_id  CHAR(20)    NOT NULL REFERENCES tenants(id),
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- migrate:down

DROP TABLE widgets;
```

Every table gets `created_at`, `updated_at`, `deleted_at`. IDs are `CHAR(20)` xids.

## 2. Add SQL queries

Create `db/queries/widgets.sql`:

```sql
-- name: CreateWidget :one
INSERT INTO widgets (id, tenant_id, name) VALUES ($1, $2, $3) RETURNING *;

-- name: GetWidget :one
SELECT * FROM widgets WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL;

-- name: ListWidgets :many
SELECT * FROM widgets WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC;

-- name: UpdateWidget :one
UPDATE widgets SET name = $3, updated_at = NOW()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL RETURNING *;

-- name: DeleteWidget :exec
UPDATE widgets SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL;
```

All reads and writes must filter `deleted_at IS NULL`.

## 3. Add the OpenAPI paths

Edit `api/openapi.yaml`. Add the paths under `paths:` and the schemas under `components/schemas`:

```yaml
paths:
  /widgets:
    get:
      operationId: listWidgets
      responses: { "200": { ... } }
    post:
      operationId: createWidget
      requestBody: { ... }
      responses: { "201": { ... } }
  # ... plus /widgets/{widgetId} for get/put/delete
components:
  schemas:
    Widget: { ... }
    WidgetCreate: { ... }
    WidgetUpdate: { ... }
```

Follow the shape of the existing `Kind` and `Tenant` resources. The tenant is derived from the api key by middleware — never take a `tenantId` URL parameter.

## 4. Regenerate

```sh
make generate
```

This runs sqlc and oapi-codegen. New types appear under `internal/db/gen/` and `internal/api/gen/`.

## 5. Write the endpoints

Create `internal/server/widgets/widgets.go`. Follow the pattern in `internal/server/tenants/tenants.go`:

```go
package widgets

type ViewModel struct { /* domain-level fields */ }

func ToViewModel(w dbgen.Widget) ViewModel { ... }
func toAPI(vm ViewModel) apigen.Widget { ... }

// One block per operation: Store interface + Endpoint struct + Interact/Build/Render.

type ListStore interface {
    ListWidgetsByTenant(ctx context.Context, tenantID string) ([]dbgen.Widget, error)
}

type ListEndpoint struct{ Store ListStore }

func (e ListEndpoint) Interact(ctx context.Context, _ apigen.ListWidgetsRequestObject) ([]dbgen.Widget, error) {
    tenantID, err := auth.TenantID(ctx)
    if err != nil {
        return nil, err
    }
    return e.Store.ListWidgetsByTenant(ctx, tenantID)
}
func (e ListEndpoint) Build(ws []dbgen.Widget) []ViewModel { ... }
func (e ListEndpoint) Render(vms []ViewModel) apigen.ListWidgetsResponseObject { ... }
```

Repeat for Get, Create, Update, Delete. Each endpoint declares its own narrow Store interface, and pulls the current tenant from context via `auth.TenantID(ctx)`. Export `ToViewModel` (capital T) so MCP tools can reuse it — see @docs/architecture.md.

## 6. Wire into the composed server

Edit `internal/server/server.go`. Add an import for `widgets`, then add methods:

```go
func (s *Server) ListWidgets(ctx context.Context, req apigen.ListWidgetsRequestObject) (apigen.ListWidgetsResponseObject, error) {
    return Run(ctx, widgets.ListEndpoint{Store: s.q}, req)
}

func (s *Server) GetWidget(ctx context.Context, req apigen.GetWidgetRequestObject) (apigen.GetWidgetResponseObject, error) {
    resp, err := Run(ctx, widgets.GetEndpoint{Store: s.q}, req)
    if IsNotFound(err) {
        return apigen.GetWidget404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse{Message: "widget not found"}}, nil
    }
    return resp, err
}
// ... etc
```

The compile-time assertion `var _ apigen.StrictServerInterface = (*Server)(nil)` at the bottom of `server.go` will complain until every method is implemented.

## 7. Add MCP tools

If Claude should be able to see or write widgets, add tools in `internal/mcp/read.go` and/or `write.go`. Follow the pattern already there — declare typed Input/Output structs with `jsonschema` tags, then register via `mcpsdk.AddTool`.

## 8. Verify

```sh
make dev
```

Then curl the new endpoints. For MCP, connect a client (Claude Desktop, `mcp inspector`, or the MCP client SDK) to `http://localhost:8080/mcp` and call the new tools.

## Common pitfalls

- Forgetting to regenerate after editing openapi.yaml or a SQL query — the compiler will not find the new types.
- Adding a query that scans deleted rows (missing `AND deleted_at IS NULL`).
- Manually editing a file in `internal/db/gen/` or `internal/api/gen/` — those get blown away by `make generate`. Change the source instead.
- Naming a slug parameter or ID field in a way that conflicts with an existing one — oapi-codegen will silently coalesce them.
