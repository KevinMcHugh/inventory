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

type SearchModelsInput struct {
	KindID      string  `json:"kindId" jsonschema:"kind xid"`
	FilterField *string `json:"filterField,omitempty" jsonschema:"indexed field key to filter on; omit to skip filtering"`
	Eq          *string `json:"eq,omitempty" jsonschema:"equals"`
	Ne          *string `json:"ne,omitempty" jsonschema:"not equal to"`
	Lt          *string `json:"lt,omitempty" jsonschema:"less than (number/integer/date fields)"`
	Lte         *string `json:"lte,omitempty" jsonschema:"less than or equal to (number/integer/date fields)"`
	Gt          *string `json:"gt,omitempty" jsonschema:"greater than (number/integer/date fields)"`
	Gte         *string `json:"gte,omitempty" jsonschema:"greater than or equal to (number/integer/date fields); combine with lte for a range"`
	Contains    *string `json:"contains,omitempty" jsonschema:"substring match; text/url/enum fields only"`
	SortField   *string `json:"sortField,omitempty" jsonschema:"indexed field key to sort by; defaults to filterField when omitted"`
	SortOrder   *string `json:"sortOrder,omitempty" jsonschema:"asc or desc, default asc"`
	Limit       *int    `json:"limit,omitempty" jsonschema:"max results, default 50, max 200"`
	Offset      *int    `json:"offset,omitempty" jsonschema:"results to skip, default 0"`
}

type SearchModelsOutput struct {
	Models []models.ViewModel `json:"models"`
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
		Name:        "search_models",
		Description: "Filter and/or sort models of a kind by an indexed field. filterField and sortField must be marked indexed on the kind's latest schema (see get_kind_schema); sortField defaults to filterField. Combine gte and lte for a range.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SearchModelsInput) (*mcpsdk.CallToolResult, SearchModelsOutput, error) {
		var sortOrder *apigen.SearchModelsParamsSortOrder
		if in.SortOrder != nil {
			v := apigen.SearchModelsParamsSortOrder(*in.SortOrder)
			sortOrder = &v
		}
		vm, err := server.BuildViewModel(ctx, models.SearchEndpoint{Store: q}, apigen.SearchModelsRequestObject{
			KindId: in.KindID,
			Params: apigen.SearchModelsParams{
				FilterField: in.FilterField,
				Eq:          in.Eq,
				Ne:          in.Ne,
				Lt:          in.Lt,
				Lte:         in.Lte,
				Gt:          in.Gt,
				Gte:         in.Gte,
				Contains:    in.Contains,
				SortField:   in.SortField,
				SortOrder:   sortOrder,
				Limit:       in.Limit,
				Offset:      in.Offset,
			},
		})
		if err != nil {
			return nil, SearchModelsOutput{}, err
		}
		return nil, SearchModelsOutput{Models: vm}, nil
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
