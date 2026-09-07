package mcp

import (
	"context"
	"encoding/json"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/xid"

	dbgen "github.com/kevinmchugh/inventory/internal/db/gen"
)

// -----------------------------------------------------------------------------
// Input / Output types
// -----------------------------------------------------------------------------

type CreateTenantInput struct {
	Name string `json:"name" jsonschema:"tenant display name"`
}

type CreateTenantOutput struct {
	Tenant TenantView `json:"tenant"`
}

type CreateKindInput struct {
	TenantID    string  `json:"tenantId" jsonschema:"tenant xid"`
	Name        string  `json:"name" jsonschema:"kind name (e.g. \"meal\", \"garment\")"`
	Description *string `json:"description,omitempty" jsonschema:"optional description"`
}

type CreateKindOutput struct {
	Kind KindView `json:"kind"`
}

type CreateKindVersionInput struct {
	KindID string         `json:"kindId" jsonschema:"kind xid"`
	Schema map[string]any `json:"schema" jsonschema:"JSON schema describing the body of models of this kind"`
}

type CreateKindVersionOutput struct {
	Version KindVersionView `json:"version"`
}

type CreateModelInput struct {
	TenantID      string         `json:"tenantId" jsonschema:"tenant xid"`
	KindID        string         `json:"kindId" jsonschema:"kind xid"`
	Slug          string         `json:"slug" jsonschema:"URL-safe slug unique within (tenant, kind)"`
	Body          map[string]any `json:"body" jsonschema:"the model payload — must conform to the resolved KindVersion schema"`
	KindVersionID *string        `json:"kindVersionId,omitempty" jsonschema:"pin to a specific KindVersion; omitted means latest"`
}

type CreateModelOutput struct {
	Model ModelView `json:"model"`
}

type UpdateModelInput struct {
	TenantID      string         `json:"tenantId" jsonschema:"tenant xid"`
	KindID        string         `json:"kindId" jsonschema:"kind xid"`
	Slug          string         `json:"slug"`
	Body          map[string]any `json:"body"`
	KindVersionID *string        `json:"kindVersionId,omitempty" jsonschema:"pin to a specific KindVersion; omitted means latest"`
}

type UpdateModelOutput struct {
	Model ModelView `json:"model"`
}

type DeleteModelInput struct {
	TenantID string `json:"tenantId"`
	KindID   string `json:"kindId"`
	Slug     string `json:"slug"`
}

type DeleteModelOutput struct {
	Deleted bool `json:"deleted"`
}

// -----------------------------------------------------------------------------
// Registration
// -----------------------------------------------------------------------------

func registerWriteTools(s *mcpsdk.Server, q dbgen.Querier) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_tenant",
		Description: "Create a new tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateTenantInput) (*mcpsdk.CallToolResult, CreateTenantOutput, error) {
		t, err := q.CreateTenant(ctx, dbgen.CreateTenantParams{
			ID:   xid.New().String(),
			Name: in.Name,
		})
		if err != nil {
			return nil, CreateTenantOutput{}, err
		}
		return nil, CreateTenantOutput{Tenant: toTenantView(t)}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_kind",
		Description: "Create a new kind (data type) under a tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateKindInput) (*mcpsdk.CallToolResult, CreateKindOutput, error) {
		k, err := q.CreateKind(ctx, dbgen.CreateKindParams{
			ID:          xid.New().String(),
			TenantID:    in.TenantID,
			Name:        in.Name,
			Description: in.Description,
		})
		if err != nil {
			return nil, CreateKindOutput{}, err
		}
		return nil, CreateKindOutput{Kind: toKindView(k)}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_kind_version",
		Description: "Register a new schema version for a kind. Later models can pin to it, or default to the latest.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateKindVersionInput) (*mcpsdk.CallToolResult, CreateKindVersionOutput, error) {
		schemaBytes, err := json.Marshal(in.Schema)
		if err != nil {
			return nil, CreateKindVersionOutput{}, err
		}
		v, err := q.CreateKindVersion(ctx, dbgen.CreateKindVersionParams{
			ID:     xid.New().String(),
			KindID: in.KindID,
			Schema: schemaBytes,
		})
		if err != nil {
			return nil, CreateKindVersionOutput{}, err
		}
		return nil, CreateKindVersionOutput{Version: toKindVersionView(v)}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_model",
		Description: "Store a new model. If kindVersionId is omitted, the latest version of the kind is used.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateModelInput) (*mcpsdk.CallToolResult, CreateModelOutput, error) {
		versionID, err := resolveVersion(ctx, q, in.KindID, in.KindVersionID)
		if err != nil {
			return nil, CreateModelOutput{}, err
		}
		bodyBytes, err := json.Marshal(in.Body)
		if err != nil {
			return nil, CreateModelOutput{}, err
		}
		m, err := q.CreateModel(ctx, dbgen.CreateModelParams{
			ID:            xid.New().String(),
			TenantID:      in.TenantID,
			KindID:        in.KindID,
			KindVersionID: versionID,
			Slug:          in.Slug,
			Body:          bodyBytes,
		})
		if err != nil {
			return nil, CreateModelOutput{}, err
		}
		return nil, CreateModelOutput{Model: toModelView(m)}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "update_model",
		Description: "Replace a model body by tenant, kind, and slug.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in UpdateModelInput) (*mcpsdk.CallToolResult, UpdateModelOutput, error) {
		versionID, err := resolveVersion(ctx, q, in.KindID, in.KindVersionID)
		if err != nil {
			return nil, UpdateModelOutput{}, err
		}
		bodyBytes, err := json.Marshal(in.Body)
		if err != nil {
			return nil, UpdateModelOutput{}, err
		}
		m, err := q.UpdateModelBySlug(ctx, dbgen.UpdateModelBySlugParams{
			TenantID:      in.TenantID,
			KindID:        in.KindID,
			Slug:          in.Slug,
			Body:          bodyBytes,
			KindVersionID: versionID,
		})
		if err != nil {
			return nil, UpdateModelOutput{}, err
		}
		return nil, UpdateModelOutput{Model: toModelView(m)}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "delete_model",
		Description: "Soft-delete a model by tenant, kind, and slug.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteModelInput) (*mcpsdk.CallToolResult, DeleteModelOutput, error) {
		err := q.DeleteModelBySlug(ctx, dbgen.DeleteModelBySlugParams{
			TenantID: in.TenantID,
			KindID:   in.KindID,
			Slug:     in.Slug,
		})
		if err != nil {
			return nil, DeleteModelOutput{}, err
		}
		return nil, DeleteModelOutput{Deleted: true}, nil
	})
}

func resolveVersion(ctx context.Context, q dbgen.Querier, kindID string, provided *string) (string, error) {
	if provided != nil && *provided != "" {
		return *provided, nil
	}
	v, err := q.GetLatestKindVersion(ctx, kindID)
	if err != nil {
		return "", err
	}
	return v.ID, nil
}
