package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/KevinMcHugh/inventory/internal/auth"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

// -----------------------------------------------------------------------------
// Input / Output types
// -----------------------------------------------------------------------------

type WhoamiInput struct{}

type WhoamiOutput struct {
	Tenant TenantView `json:"tenant"`
}

type ListKindsInput struct{}

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
	KindID string `json:"kindId" jsonschema:"kind xid"`
}

type ListModelsOutput struct {
	Models []ModelView `json:"models"`
}

type GetModelInput struct {
	KindID string `json:"kindId" jsonschema:"kind xid"`
	Slug   string `json:"slug" jsonschema:"human-readable model slug"`
}

type GetModelOutput struct {
	Model ModelView `json:"model"`
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
		return nil, WhoamiOutput{Tenant: toTenantView(t)}, nil
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
		out := ListModelsOutput{Models: make([]ModelView, len(ms))}
		for i, m := range ms {
			out.Models[i] = toModelView(m)
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
		return nil, GetModelOutput{Model: toModelView(m)}, nil
	})
}
