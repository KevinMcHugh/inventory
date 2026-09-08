package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	apigen "github.com/KevinMcHugh/inventory/internal/api/gen"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	"github.com/KevinMcHugh/inventory/internal/kindschema"
	"github.com/KevinMcHugh/inventory/internal/server"
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
//
// Each tool builds the same Endpoint the HTTP handler for the equivalent
// operation uses, drives it through server.BuildViewModel (Interact + Build,
// the Store-backed domain logic and M->VM transform), and renders the
// resulting view-model into its own Output struct in place of Render.
// -----------------------------------------------------------------------------

func registerReadTools(s *mcpsdk.Server, q dbgen.Querier) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "whoami",
		Description: "Return the tenant this MCP session is authenticated as.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ WhoamiInput) (*mcpsdk.CallToolResult, WhoamiOutput, error) {
		vm, err := server.BuildViewModel(ctx, tenants.GetEndpoint{Store: q}, apigen.GetTenantRequestObject{})
		if err != nil {
			return nil, WhoamiOutput{}, err
		}
		return nil, WhoamiOutput{Tenant: vm}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_kinds",
		Description: "List all kinds (data types) for the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ ListKindsInput) (*mcpsdk.CallToolResult, ListKindsOutput, error) {
		vm, err := server.BuildViewModel(ctx, kinds.ListEndpoint{Store: q}, apigen.ListKindsRequestObject{})
		if err != nil {
			return nil, ListKindsOutput{}, err
		}
		return nil, ListKindsOutput{Kinds: vm}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_kind_versions",
		Description: "List all schema versions for a kind, most recent first.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListKindVersionsInput) (*mcpsdk.CallToolResult, ListKindVersionsOutput, error) {
		vm, err := server.BuildViewModel(ctx, kinds.ListVersionsEndpoint{Store: q}, apigen.ListKindVersionsRequestObject{KindId: in.KindID})
		if err != nil {
			return nil, ListKindVersionsOutput{}, err
		}
		return nil, ListKindVersionsOutput{Versions: vm}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_models",
		Description: "List all models for a given kind within the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListModelsInput) (*mcpsdk.CallToolResult, ListModelsOutput, error) {
		vm, err := server.BuildViewModel(ctx, models.ListEndpoint{Store: q}, apigen.ListModelsRequestObject{KindId: in.KindID})
		if err != nil {
			return nil, ListModelsOutput{}, err
		}
		return nil, ListModelsOutput{Models: vm}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_model",
		Description: "Fetch a single model by kind and slug within the current tenant.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetModelInput) (*mcpsdk.CallToolResult, GetModelOutput, error) {
		vm, err := server.BuildViewModel(ctx, models.GetEndpoint{Store: q}, apigen.GetModelRequestObject{
			KindId: in.KindID,
			Slug:   in.Slug,
		})
		if err != nil {
			return nil, GetModelOutput{}, err
		}
		return nil, GetModelOutput{Model: vm}, nil
	})

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_kind_schema",
		Description: "Return the latest schema authored for a kind (empty if none). The schema drives column order, labels, per-field types, and filter widgets on the web UI.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetKindSchemaInput) (*mcpsdk.CallToolResult, GetKindSchemaOutput, error) {
		vm, err := server.BuildViewModel(ctx, kinds.GetSchemaEndpoint{Store: q}, apigen.GetKindSchemaRequestObject{KindId: in.KindID})
		if err != nil {
			return nil, GetKindSchemaOutput{}, err
		}
		return nil, GetKindSchemaOutput{Schema: *vm}, nil
	})
}
