package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	apigen "github.com/KevinMcHugh/inventory/internal/api/gen"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	"github.com/KevinMcHugh/inventory/internal/server"
	"github.com/KevinMcHugh/inventory/internal/server/kinds"
	"github.com/KevinMcHugh/inventory/internal/server/models"
)

// -----------------------------------------------------------------------------
// Input / Output types
// -----------------------------------------------------------------------------

type CreateKindInput struct {
	Name        string  `json:"name" jsonschema:"kind name (e.g. meal, garment)"`
	Description *string `json:"description,omitempty" jsonschema:"optional description"`
}

type CreateKindOutput struct {
	Kind kinds.ViewModel `json:"kind"`
}

type CreateKindVersionInput struct {
	KindID string         `json:"kindId" jsonschema:"kind xid"`
	Schema map[string]any `json:"schema" jsonschema:"JSON schema for models of this kind"`
}

type CreateKindVersionOutput struct {
	Version kinds.VersionViewModel `json:"version"`
}

type CreateModelInput struct {
	KindID        string         `json:"kindId" jsonschema:"kind xid"`
	Slug          string         `json:"slug" jsonschema:"URL-safe slug unique within kind"`
	Body          map[string]any `json:"body" jsonschema:"the model payload"`
	KindVersionID *string        `json:"kindVersionId,omitempty" jsonschema:"pin to a specific KindVersion; omitted means latest"`
}

type CreateModelOutput struct {
	Model models.ViewModel `json:"model"`
}

type UpdateModelInput struct {
	KindID        string         `json:"kindId"`
	Slug          string         `json:"slug"`
	Body          map[string]any `json:"body"`
	KindVersionID *string        `json:"kindVersionId,omitempty" jsonschema:"pin to a specific KindVersion; omitted means latest"`
}

type UpdateModelOutput struct {
	Model models.ViewModel `json:"model"`
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
//
// Each tool builds the same Endpoint the HTTP handler for the equivalent
// operation uses, drives it through server.BuildViewModel (Interact + Build,
// the Store-backed domain logic and M->VM transform), and renders the
// resulting view-model into its own Output struct in place of Render.
// -----------------------------------------------------------------------------

func registerWriteTools(s *mcpsdk.Server, q dbgen.Querier) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_kind",
		Description: "Create a new kind (data type) under the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateKindInput) (*mcpsdk.CallToolResult, CreateKindOutput, error) {
		vm, err := server.BuildViewModel(ctx, kinds.CreateEndpoint{Store: q}, apigen.CreateKindRequestObject{
			Body: &apigen.CreateKindJSONRequestBody{
				Name:        in.Name,
				Description: in.Description,
			},
		})
		if err != nil {
			return nil, CreateKindOutput{}, err
		}
		return nil, CreateKindOutput{Kind: vm}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_kind_version",
		Description: "Register a new schema version for a kind. Later models can pin to it, or default to the latest.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateKindVersionInput) (*mcpsdk.CallToolResult, CreateKindVersionOutput, error) {
		vm, err := server.BuildViewModel(ctx, kinds.CreateVersionEndpoint{Store: q}, apigen.CreateKindVersionRequestObject{
			KindId: in.KindID,
			Body: &apigen.CreateKindVersionJSONRequestBody{
				Schema: in.Schema,
			},
		})
		if err != nil {
			return nil, CreateKindVersionOutput{}, err
		}
		return nil, CreateKindVersionOutput{Version: vm}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_model",
		Description: "Store a new model. If kindVersionId is omitted, the latest version of the kind is used.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateModelInput) (*mcpsdk.CallToolResult, CreateModelOutput, error) {
		vm, err := server.BuildViewModel(ctx, models.CreateEndpoint{Store: q}, apigen.CreateModelRequestObject{
			KindId: in.KindID,
			Body: &apigen.CreateModelJSONRequestBody{
				Slug:          in.Slug,
				Body:          in.Body,
				KindVersionId: in.KindVersionID,
			},
		})
		if err != nil {
			return nil, CreateModelOutput{}, err
		}
		return nil, CreateModelOutput{Model: vm}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "update_model",
		Description: "Replace a model body by kind and slug within the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in UpdateModelInput) (*mcpsdk.CallToolResult, UpdateModelOutput, error) {
		vm, err := server.BuildViewModel(ctx, models.UpdateEndpoint{Store: q}, apigen.UpdateModelRequestObject{
			KindId: in.KindID,
			Slug:   in.Slug,
			Body: &apigen.UpdateModelJSONRequestBody{
				Body:          in.Body,
				KindVersionId: in.KindVersionID,
			},
		})
		if err != nil {
			return nil, UpdateModelOutput{}, err
		}
		return nil, UpdateModelOutput{Model: vm}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "delete_model",
		Description: "Soft-delete a model by kind and slug within the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteModelInput) (*mcpsdk.CallToolResult, DeleteModelOutput, error) {
		deleted, err := server.BuildViewModel(ctx, models.DeleteEndpoint{Store: q}, apigen.DeleteModelRequestObject{
			KindId: in.KindID,
			Slug:   in.Slug,
		})
		if err != nil {
			return nil, DeleteModelOutput{}, err
		}
		return nil, DeleteModelOutput{Deleted: deleted}, nil
	})
}
