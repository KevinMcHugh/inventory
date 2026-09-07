// Package mcp exposes the inventory datastore over the Model Context Protocol
// so that Claude (or any MCP client) can list and manipulate models.
//
// Tools declare typed input and output structs; jsonschema is inferred from
// struct tags. Each tool captures the dbgen.Querier in a closure — the same
// store the HTTP endpoints use, so both interfaces see identical data.
package mcp

import (
	"context"
	"encoding/json"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	dbgen "github.com/kevinmchugh/inventory/internal/db/gen"
)

// NewServer builds an MCP server with all inventory tools registered.
func NewServer(q dbgen.Querier) *mcpsdk.Server {
	s := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "inventory",
		Version: "0.1.0",
	}, nil)
	registerReadTools(s, q)
	return s
}

// -----------------------------------------------------------------------------
// list_tenants
// -----------------------------------------------------------------------------

type ListTenantsInput struct{}

type TenantView struct {
	ID        string    `json:"id" jsonschema:"tenant xid"`
	Name      string    `json:"name" jsonschema:"tenant display name"`
	CreatedAt time.Time `json:"createdAt"`
}

type ListTenantsOutput struct {
	Tenants []TenantView `json:"tenants"`
}

// -----------------------------------------------------------------------------
// list_kinds
// -----------------------------------------------------------------------------

type ListKindsInput struct {
	TenantID string `json:"tenantId" jsonschema:"tenant xid"`
}

type KindView struct {
	ID          string  `json:"id" jsonschema:"kind xid"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type ListKindsOutput struct {
	Kinds []KindView `json:"kinds"`
}

// -----------------------------------------------------------------------------
// list_models
// -----------------------------------------------------------------------------

type ListModelsInput struct {
	TenantID string `json:"tenantId" jsonschema:"tenant xid"`
	KindID   string `json:"kindId" jsonschema:"kind xid"`
}

type ModelView struct {
	ID            string         `json:"id"`
	Slug          string         `json:"slug"`
	KindVersionID string         `json:"kindVersionId"`
	Body          map[string]any `json:"body"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

type ListModelsOutput struct {
	Models []ModelView `json:"models"`
}

// -----------------------------------------------------------------------------
// get_model
// -----------------------------------------------------------------------------

type GetModelInput struct {
	TenantID string `json:"tenantId" jsonschema:"tenant xid"`
	KindID   string `json:"kindId" jsonschema:"kind xid"`
	Slug     string `json:"slug" jsonschema:"human-readable model slug"`
}

type GetModelOutput struct {
	Model ModelView `json:"model"`
}

// -----------------------------------------------------------------------------
// registration
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
			out.Tenants[i] = TenantView{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt.Time}
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
			out.Kinds[i] = KindView{ID: k.ID, Name: k.Name, Description: k.Description}
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

func toModelView(m dbgen.Model) ModelView {
	var body map[string]any
	if len(m.Body) > 0 {
		_ = json.Unmarshal(m.Body, &body)
	}
	return ModelView{
		ID:            m.ID,
		Slug:          m.Slug,
		KindVersionID: m.KindVersionID,
		Body:          body,
		CreatedAt:     m.CreatedAt.Time,
		UpdatedAt:     m.UpdatedAt.Time,
	}
}
