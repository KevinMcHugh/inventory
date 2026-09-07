package mcp

import (
	"context"
	"encoding/json"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/xid"

	"github.com/kevinmchugh/inventory/internal/auth"
	dbgen "github.com/kevinmchugh/inventory/internal/db/gen"
)

// -----------------------------------------------------------------------------
// Input / Output types
// -----------------------------------------------------------------------------

type CreateKindInput struct {
	Name        string  `json:"name" jsonschema:"kind name (e.g. meal, garment)"`
	Description *string `json:"description,omitempty" jsonschema:"optional description"`
}

type CreateKindOutput struct {
	Kind KindView `json:"kind"`
}

type CreateKindVersionInput struct {
	KindID string         `json:"kindId" jsonschema:"kind xid"`
	Schema map[string]any `json:"schema" jsonschema:"JSON schema for models of this kind"`
}

type CreateKindVersionOutput struct {
	Version KindVersionView `json:"version"`
}

type CreateModelInput struct {
	KindID        string         `json:"kindId" jsonschema:"kind xid"`
	Slug          string         `json:"slug" jsonschema:"URL-safe slug unique within kind"`
	Body          map[string]any `json:"body" jsonschema:"the model payload"`
	KindVersionID *string        `json:"kindVersionId,omitempty" jsonschema:"pin to a specific KindVersion; omitted means latest"`
}

type CreateModelOutput struct {
	Model ModelView `json:"model"`
}

type UpdateModelInput struct {
	KindID        string         `json:"kindId"`
	Slug          string         `json:"slug"`
	Body          map[string]any `json:"body"`
	KindVersionID *string        `json:"kindVersionId,omitempty" jsonschema:"pin to a specific KindVersion; omitted means latest"`
}

type UpdateModelOutput struct {
	Model ModelView `json:"model"`
}

type DeleteModelInput struct {
	KindID string `json:"kindId"`
	Slug   string `json:"slug"`
}

type DeleteModelOutput struct {
	Deleted bool `json:"deleted"`
}

// -----------------------------------------------------------------------------
// Registration
// -----------------------------------------------------------------------------

func registerWriteTools(s *mcpsdk.Server, q dbgen.Querier) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_kind",
		Description: "Create a new kind (data type) under the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateKindInput) (*mcpsdk.CallToolResult, CreateKindOutput, error) {
		tenantID, err := auth.TenantID(ctx)
		if err != nil {
			return nil, CreateKindOutput{}, err
		}
		k, err := q.CreateKind(ctx, dbgen.CreateKindParams{
			ID:          xid.New().String(),
			TenantID:    tenantID,
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
		tenantID, err := auth.TenantID(ctx)
		if err != nil {
			return nil, CreateModelOutput{}, err
		}
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
			TenantID:      tenantID,
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
		Description: "Replace a model body by kind and slug within the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in UpdateModelInput) (*mcpsdk.CallToolResult, UpdateModelOutput, error) {
		tenantID, err := auth.TenantID(ctx)
		if err != nil {
			return nil, UpdateModelOutput{}, err
		}
		versionID, err := resolveVersion(ctx, q, in.KindID, in.KindVersionID)
		if err != nil {
			return nil, UpdateModelOutput{}, err
		}
		bodyBytes, err := json.Marshal(in.Body)
		if err != nil {
			return nil, UpdateModelOutput{}, err
		}
		m, err := q.UpdateModelBySlug(ctx, dbgen.UpdateModelBySlugParams{
			TenantID:      tenantID,
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
		Description: "Soft-delete a model by kind and slug within the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteModelInput) (*mcpsdk.CallToolResult, DeleteModelOutput, error) {
		tenantID, err := auth.TenantID(ctx)
		if err != nil {
			return nil, DeleteModelOutput{}, err
		}
		err = q.DeleteModelBySlug(ctx, dbgen.DeleteModelBySlugParams{
			TenantID: tenantID,
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
