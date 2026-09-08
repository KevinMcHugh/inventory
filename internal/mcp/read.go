package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/KevinMcHugh/inventory/internal/auth"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	"github.com/KevinMcHugh/inventory/internal/kindschema"
	"github.com/KevinMcHugh/inventory/internal/server/kinds"
	"github.com/KevinMcHugh/inventory/internal/server/models"
	"github.com/KevinMcHugh/inventory/internal/server/tenants"
)

// -----------------------------------------------------------------------------
// Input / Output types
// -----------------------------------------------------------------------------

type WhoamiInput struct{}

type WhoamiOutput struct {
	Tenant tenants.ViewModel `json:"tenant"`
}

type ListKindsInput struct{}

type ListKindsOutput struct {
	Kinds []kinds.ViewModel `json:"kinds"`
}

type ListKindVersionsInput struct {
	KindID string `json:"kindId" jsonschema:"kind xid"`
}

type ListKindVersionsOutput struct {
	Versions []kinds.VersionViewModel `json:"versions"`
}

type ListModelsInput struct {
	KindID string `json:"kindId" jsonschema:"kind xid"`
}

type ListModelsOutput struct {
	Models []models.ViewModel `json:"models"`
}

type GetModelInput struct {
	KindID string `json:"kindId" jsonschema:"kind xid"`
	Slug   string `json:"slug" jsonschema:"human-readable model slug"`
}

type GetModelOutput struct {
	Model models.ViewModel `json:"model"`
}

type GetKindSchemaInput struct {
	KindID string `json:"kindId" jsonschema:"kind xid"`
}

type GetKindSchemaOutput struct {
	Schema kindschema.Schema `json:"schema"`
}

// -----------------------------------------------------------------------------
// Registration
// -----------------------------------------------------------------------------

func registerReadTools(s *mcpsdk.Server, q dbgen.Querier) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "whoami",
		Description: "Return the tenant this MCP session is authenticated as.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ WhoamiInput) (*mcpsdk.CallToolResult, WhoamiOutput, error) {
		tenantID, err := auth.TenantID(ctx)
		if err != nil {
			return nil, WhoamiOutput{}, err
		}
		t, err := q.GetTenant(ctx, tenantID)
		if err != nil {
			return nil, WhoamiOutput{}, err
		}
		return nil, WhoamiOutput{Tenant: tenants.ToViewModel(t)}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_kinds",
		Description: "List all kinds (data types) for the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ ListKindsInput) (*mcpsdk.CallToolResult, ListKindsOutput, error) {
		tenantID, err := auth.TenantID(ctx)
		if err != nil {
			return nil, ListKindsOutput{}, err
		}
		ks, err := q.ListKindsByTenant(ctx, tenantID)
		if err != nil {
			return nil, ListKindsOutput{}, err
		}
		out := ListKindsOutput{Kinds: make([]kinds.ViewModel, len(ks))}
		for i, k := range ks {
			out.Kinds[i] = kinds.ToViewModel(k)
		}
		return nil, out, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_kind_versions",
		Description: "List all schema versions for a kind, most recent first.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListKindVersionsInput) (*mcpsdk.CallToolResult, ListKindVersionsOutput, error) {
		vs, err := q.ListKindVersionsByKind(ctx, in.KindID)
		if err != nil {
			return nil, ListKindVersionsOutput{}, err
		}
		out := ListKindVersionsOutput{Versions: make([]kinds.VersionViewModel, len(vs))}
		for i, v := range vs {
			out.Versions[i] = kinds.ToVersionViewModel(v)
		}
		return nil, out, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_models",
		Description: "List all models for a given kind within the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListModelsInput) (*mcpsdk.CallToolResult, ListModelsOutput, error) {
		tenantID, err := auth.TenantID(ctx)
		if err != nil {
			return nil, ListModelsOutput{}, err
		}
		ms, err := q.ListModelsByKind(ctx, dbgen.ListModelsByKindParams{
			TenantID: tenantID,
			KindID:   in.KindID,
		})
		if err != nil {
			return nil, ListModelsOutput{}, err
		}
		out := ListModelsOutput{Models: make([]models.ViewModel, len(ms))}
		for i, m := range ms {
			out.Models[i] = models.ToViewModel(m)
		}
		return nil, out, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_model",
		Description: "Fetch a single model by kind and slug within the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetModelInput) (*mcpsdk.CallToolResult, GetModelOutput, error) {
		tenantID, err := auth.TenantID(ctx)
		if err != nil {
			return nil, GetModelOutput{}, err
		}
		m, err := q.GetModelBySlug(ctx, dbgen.GetModelBySlugParams{
			TenantID: tenantID,
			KindID:   in.KindID,
			Slug:     in.Slug,
		})
		if err != nil {
			return nil, GetModelOutput{}, err
		}
		return nil, GetModelOutput{Model: models.ToViewModel(m)}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_kind_schema",
		Description: "Return the latest schema authored for a kind (empty if none). The schema drives column order, labels, per-field types, and filter widgets on the web UI.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetKindSchemaInput) (*mcpsdk.CallToolResult, GetKindSchemaOutput, error) {
		tenantID, err := auth.TenantID(ctx)
		if err != nil {
			return nil, GetKindSchemaOutput{}, err
		}
		// Scope the read to the caller's tenant.
		if _, err := q.GetKind(ctx, dbgen.GetKindParams{ID: in.KindID, TenantID: tenantID}); err != nil {
			return nil, GetKindSchemaOutput{}, err
		}
		s, err := kindschema.LoadLatest(ctx, q, in.KindID)
		if err != nil {
			return nil, GetKindSchemaOutput{}, err
		}
		return nil, GetKindSchemaOutput{Schema: *s}, nil
	})
}
