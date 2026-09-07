package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	dbgen "github.com/kevinmchugh/inventory/internal/db/gen"
)

// -----------------------------------------------------------------------------
// Input / Output types
// -----------------------------------------------------------------------------

type ListTenantsInput struct{}

type ListTenantsOutput struct {
	Tenants []TenantView `json:"tenants"`
}

type ListKindsInput struct {
	TenantID string `json:"tenantId" jsonschema:"tenant xid"`
}

type ListKindsOutput struct {
	Kinds []KindView `json:"kinds"`
}

type ListKindVersionsInput struct {
	KindID string `json:"kindId" jsonschema:"kind xid"`
}

type ListKindVersionsOutput struct {
	Versions []KindVersionView `json:"versions"`
}

type ListModelsInput struct {
	TenantID string `json:"tenantId" jsonschema:"tenant xid"`
	KindID   string `json:"kindId" jsonschema:"kind xid"`
}

type ListModelsOutput struct {
	Models []ModelView `json:"models"`
}

type GetModelInput struct {
	TenantID string `json:"tenantId" jsonschema:"tenant xid"`
	KindID   string `json:"kindId" jsonschema:"kind xid"`
	Slug     string `json:"slug" jsonschema:"human-readable model slug"`
}

type GetModelOutput struct {
	Model ModelView `json:"model"`
}

// -----------------------------------------------------------------------------
// Registration
// -----------------------------------------------------------------------------

func registerReadTools(s *mcpsdk.Server, q dbgen.Querier) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_tenants",
		Description: "List all tenants in the inventory.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ ListTenantsInput) (*mcpsdk.CallToolResult, ListTenantsOutput, error) {
		ts, err := q.ListTenants(ctx)
		if err != nil {
			return nil, ListTenantsOutput{}, err
		}
		out := ListTenantsOutput{Tenants: make([]TenantView, len(ts))}
		for i, t := range ts {
			out.Tenants[i] = toTenantView(t)
		}
		return nil, out, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_kinds",
		Description: "List all kinds (data types) for a tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListKindsInput) (*mcpsdk.CallToolResult, ListKindsOutput, error) {
		ks, err := q.ListKindsByTenant(ctx, in.TenantID)
		if err != nil {
			return nil, ListKindsOutput{}, err
		}
		out := ListKindsOutput{Kinds: make([]KindView, len(ks))}
		for i, k := range ks {
			out.Kinds[i] = toKindView(k)
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
		out := ListKindVersionsOutput{Versions: make([]KindVersionView, len(vs))}
		for i, v := range vs {
			out.Versions[i] = toKindVersionView(v)
		}
		return nil, out, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_models",
		Description: "List all models for a given kind.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListModelsInput) (*mcpsdk.CallToolResult, ListModelsOutput, error) {
		ms, err := q.ListModelsByKind(ctx, dbgen.ListModelsByKindParams{
			TenantID: in.TenantID,
			KindID:   in.KindID,
		})
		if err != nil {
			return nil, ListModelsOutput{}, err
		}
		out := ListModelsOutput{Models: make([]ModelView, len(ms))}
		for i, m := range ms {
			out.Models[i] = toModelView(m)
		}
		return nil, out, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_model",
		Description: "Fetch a single model by tenant, kind, and slug.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetModelInput) (*mcpsdk.CallToolResult, GetModelOutput, error) {
		m, err := q.GetModelBySlug(ctx, dbgen.GetModelBySlugParams{
			TenantID: in.TenantID,
			KindID:   in.KindID,
			Slug:     in.Slug,
		})
		if err != nil {
			return nil, GetModelOutput{}, err
		}
		return nil, GetModelOutput{Model: toModelView(m)}, nil
	})
}
