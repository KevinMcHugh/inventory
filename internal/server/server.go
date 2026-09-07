package server

import (
	"context"
	"errors"

	apigen "github.com/kevinmchugh/inventory/internal/api/gen"
	dbgen "github.com/kevinmchugh/inventory/internal/db/gen"
	"github.com/kevinmchugh/inventory/internal/server/tenants"
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
// Tenants
// -----------------------------------------------------------------------------

func (s *Server) ListTenants(ctx context.Context, req apigen.ListTenantsRequestObject) (apigen.ListTenantsResponseObject, error) {
	return Run(ctx, tenants.ListEndpoint{Store: s.q}, req)
}

func (s *Server) GetTenant(ctx context.Context, req apigen.GetTenantRequestObject) (apigen.GetTenantResponseObject, error) {
	resp, err := Run(ctx, tenants.GetEndpoint{Store: s.q}, req)
	if IsNotFound(err) {
		return apigen.GetTenant404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse{Message: "tenant not found"}}, nil
	}
	return resp, err
}

func (s *Server) CreateTenant(ctx context.Context, req apigen.CreateTenantRequestObject) (apigen.CreateTenantResponseObject, error) {
	if req.Body == nil {
		return apigen.CreateTenant400JSONResponse{BadRequestJSONResponse: apigen.BadRequestJSONResponse{Message: "body required"}}, nil
	}
	return Run(ctx, tenants.CreateEndpoint{Store: s.q}, req)
}

func (s *Server) UpdateTenant(ctx context.Context, req apigen.UpdateTenantRequestObject) (apigen.UpdateTenantResponseObject, error) {
	if req.Body == nil {
		return nil, errors.New("body required")
	}
	resp, err := Run(ctx, tenants.UpdateEndpoint{Store: s.q}, req)
	if IsNotFound(err) {
		return apigen.UpdateTenant404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse{Message: "tenant not found"}}, nil
	}
	return resp, err
}

func (s *Server) DeleteTenant(ctx context.Context, req apigen.DeleteTenantRequestObject) (apigen.DeleteTenantResponseObject, error) {
	return Run(ctx, tenants.DeleteEndpoint{Store: s.q}, req)
}

// -----------------------------------------------------------------------------
// Kinds — TODO(next commit): replace stubs with real endpoints.
// -----------------------------------------------------------------------------

func (s *Server) ListKinds(context.Context, apigen.ListKindsRequestObject) (apigen.ListKindsResponseObject, error) {
	return nil, errNotImplemented
}
func (s *Server) CreateKind(context.Context, apigen.CreateKindRequestObject) (apigen.CreateKindResponseObject, error) {
	return nil, errNotImplemented
}
func (s *Server) GetKind(context.Context, apigen.GetKindRequestObject) (apigen.GetKindResponseObject, error) {
	return nil, errNotImplemented
}
func (s *Server) UpdateKind(context.Context, apigen.UpdateKindRequestObject) (apigen.UpdateKindResponseObject, error) {
	return nil, errNotImplemented
}
func (s *Server) DeleteKind(context.Context, apigen.DeleteKindRequestObject) (apigen.DeleteKindResponseObject, error) {
	return nil, errNotImplemented
}
func (s *Server) ListKindVersions(context.Context, apigen.ListKindVersionsRequestObject) (apigen.ListKindVersionsResponseObject, error) {
	return nil, errNotImplemented
}
func (s *Server) CreateKindVersion(context.Context, apigen.CreateKindVersionRequestObject) (apigen.CreateKindVersionResponseObject, error) {
	return nil, errNotImplemented
}

// -----------------------------------------------------------------------------
// Models — TODO(next commit): replace stubs with real endpoints.
// -----------------------------------------------------------------------------

func (s *Server) ListModels(context.Context, apigen.ListModelsRequestObject) (apigen.ListModelsResponseObject, error) {
	return nil, errNotImplemented
}
func (s *Server) CreateModel(context.Context, apigen.CreateModelRequestObject) (apigen.CreateModelResponseObject, error) {
	return nil, errNotImplemented
}
func (s *Server) GetModel(context.Context, apigen.GetModelRequestObject) (apigen.GetModelResponseObject, error) {
	return nil, errNotImplemented
}
func (s *Server) UpdateModel(context.Context, apigen.UpdateModelRequestObject) (apigen.UpdateModelResponseObject, error) {
	return nil, errNotImplemented
}
func (s *Server) DeleteModel(context.Context, apigen.DeleteModelRequestObject) (apigen.DeleteModelResponseObject, error) {
	return nil, errNotImplemented
}

var errNotImplemented = errors.New("not implemented")

// Compile-time assertion.
var _ apigen.StrictServerInterface = (*Server)(nil)
