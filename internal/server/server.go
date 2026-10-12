package server

import (
	"context"
	"errors"

	apigen "github.com/KevinMcHugh/inventory/internal/api/gen"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	"github.com/KevinMcHugh/inventory/internal/kindschema"
	"github.com/KevinMcHugh/inventory/internal/server/kinds"
	"github.com/KevinMcHugh/inventory/internal/server/models"
	"github.com/KevinMcHugh/inventory/internal/server/tenants"
)

// Server implements apigen.StrictServerInterface by composing per-resource
// endpoints. Each endpoint holds a narrow store interface; the sqlc-generated
// *dbgen.Queries satisfies all of them.
type Server struct {
	q dbgen.Querier
}

func New(q dbgen.Querier) *Server {
	return &Server{q: q}
}

// -----------------------------------------------------------------------------
// Health
// -----------------------------------------------------------------------------

func (s *Server) GetHealth(_ context.Context, _ apigen.GetHealthRequestObject) (apigen.GetHealthResponseObject, error) {
	return apigen.GetHealth200JSONResponse{Ok: true}, nil
}

// -----------------------------------------------------------------------------
// Tenant (self)
// -----------------------------------------------------------------------------

func (s *Server) GetTenant(ctx context.Context, req apigen.GetTenantRequestObject) (apigen.GetTenantResponseObject, error) {
	return Run(ctx, tenants.GetEndpoint{Store: s.q}, req)
}

func (s *Server) UpdateTenant(ctx context.Context, req apigen.UpdateTenantRequestObject) (apigen.UpdateTenantResponseObject, error) {
	if req.Body == nil {
		return nil, errors.New("body required")
	}
	return Run(ctx, tenants.UpdateEndpoint{Store: s.q}, req)
}

// -----------------------------------------------------------------------------
// Kinds
// -----------------------------------------------------------------------------

func (s *Server) ListKinds(ctx context.Context, req apigen.ListKindsRequestObject) (apigen.ListKindsResponseObject, error) {
	return Run(ctx, kinds.ListEndpoint{Store: s.q}, req)
}

func (s *Server) GetKind(ctx context.Context, req apigen.GetKindRequestObject) (apigen.GetKindResponseObject, error) {
	resp, err := Run(ctx, kinds.GetEndpoint{Store: s.q}, req)
	if IsNotFound(err) {
		return apigen.GetKind404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse{Message: "kind not found"}}, nil
	}
	return resp, err
}

func (s *Server) CreateKind(ctx context.Context, req apigen.CreateKindRequestObject) (apigen.CreateKindResponseObject, error) {
	if req.Body == nil {
		return nil, errors.New("body required")
	}
	return Run(ctx, kinds.CreateEndpoint{Store: s.q}, req)
}

func (s *Server) UpdateKind(ctx context.Context, req apigen.UpdateKindRequestObject) (apigen.UpdateKindResponseObject, error) {
	if req.Body == nil {
		return nil, errors.New("body required")
	}
	return Run(ctx, kinds.UpdateEndpoint{Store: s.q}, req)
}

func (s *Server) DeleteKind(ctx context.Context, req apigen.DeleteKindRequestObject) (apigen.DeleteKindResponseObject, error) {
	return Run(ctx, kinds.DeleteEndpoint{Store: s.q}, req)
}

func (s *Server) ListKindVersions(ctx context.Context, req apigen.ListKindVersionsRequestObject) (apigen.ListKindVersionsResponseObject, error) {
	return Run(ctx, kinds.ListVersionsEndpoint{Store: s.q}, req)
}

func (s *Server) GetKindSchema(ctx context.Context, req apigen.GetKindSchemaRequestObject) (apigen.GetKindSchemaResponseObject, error) {
	resp, err := Run(ctx, kinds.GetSchemaEndpoint{Store: s.q}, req)
	if IsNotFound(err) {
		return apigen.GetKindSchema404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse{Message: "kind not found"}}, nil
	}
	return resp, err
}

func (s *Server) CreateKindVersion(ctx context.Context, req apigen.CreateKindVersionRequestObject) (apigen.CreateKindVersionResponseObject, error) {
	if req.Body == nil {
		return nil, errors.New("body required")
	}
	return Run(ctx, kinds.CreateVersionEndpoint{Store: s.q}, req)
}

// -----------------------------------------------------------------------------
// Models
// -----------------------------------------------------------------------------

func (s *Server) ListModels(ctx context.Context, req apigen.ListModelsRequestObject) (apigen.ListModelsResponseObject, error) {
	return Run(ctx, models.ListEndpoint{Store: s.q}, req)
}

func (s *Server) GetModel(ctx context.Context, req apigen.GetModelRequestObject) (apigen.GetModelResponseObject, error) {
	resp, err := Run(ctx, models.GetEndpoint{Store: s.q}, req)
	if IsNotFound(err) {
		return apigen.GetModel404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse{Message: "model not found"}}, nil
	}
	return resp, err
}

func (s *Server) CreateModel(ctx context.Context, req apigen.CreateModelRequestObject) (apigen.CreateModelResponseObject, error) {
	if req.Body == nil {
		return nil, errors.New("body required")
	}
	resp, err := Run(ctx, models.CreateEndpoint{Store: s.q}, req)
	if verrs, ok := asValidationErrors(err); ok {
		return apigen.CreateModel400JSONResponse{BadRequestJSONResponse: validationErrorResponse(verrs)}, nil
	}
	return resp, err
}

func (s *Server) UpdateModel(ctx context.Context, req apigen.UpdateModelRequestObject) (apigen.UpdateModelResponseObject, error) {
	if req.Body == nil {
		return nil, errors.New("body required")
	}
	resp, err := Run(ctx, models.UpdateEndpoint{Store: s.q}, req)
	if verrs, ok := asValidationErrors(err); ok {
		return apigen.UpdateModel400JSONResponse{BadRequestJSONResponse: validationErrorResponse(verrs)}, nil
	}
	return resp, err
}

func (s *Server) DeleteModel(ctx context.Context, req apigen.DeleteModelRequestObject) (apigen.DeleteModelResponseObject, error) {
	return Run(ctx, models.DeleteEndpoint{Store: s.q}, req)
}

func (s *Server) SearchModels(ctx context.Context, req apigen.SearchModelsRequestObject) (apigen.SearchModelsResponseObject, error) {
	resp, err := Run(ctx, models.SearchEndpoint{Store: s.q}, req)
	if verrs, ok := asValidationErrors(err); ok {
		return apigen.SearchModels400JSONResponse{BadRequestJSONResponse: validationErrorResponse(verrs)}, nil
	}
	return resp, err
}

// Compile-time assertion.
var _ apigen.StrictServerInterface = (*Server)(nil)

// asValidationErrors unwraps err into kindschema.ValidationErrors if it is
// one, so callers can produce a structured 400 response instead of a 500.
func asValidationErrors(err error) (kindschema.ValidationErrors, bool) {
	if err == nil {
		return nil, false
	}
	var verrs kindschema.ValidationErrors
	if errors.As(err, &verrs) {
		return verrs, true
	}
	return nil, false
}

// validationErrorResponse builds a BadRequestJSONResponse populated with
// per-field errors from a ValidationErrors list.
func validationErrorResponse(verrs kindschema.ValidationErrors) apigen.BadRequestJSONResponse {
	fields := make([]apigen.FieldError, len(verrs))
	for i, e := range verrs {
		fields[i] = apigen.FieldError{Field: e.Field, Message: e.Message}
	}
	code := "validation_failed"
	return apigen.BadRequestJSONResponse{
		Message: "one or more fields did not match the kind schema",
		Code:    &code,
		Fields:  &fields,
	}
}
