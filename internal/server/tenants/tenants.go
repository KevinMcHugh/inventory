package tenants

import (
	"context"
	"time"

	"github.com/rs/xid"

	apigen "github.com/kevinmchugh/inventory/internal/api/gen"
	dbgen "github.com/kevinmchugh/inventory/internal/db/gen"
)

// ViewModel is the presentation form of a tenant.
type ViewModel struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func toViewModel(t dbgen.Tenant) ViewModel {
	return ViewModel{
		ID:        t.ID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt.Time,
		UpdatedAt: t.UpdatedAt.Time,
	}
}

func toAPI(vm ViewModel) apigen.Tenant {
	return apigen.Tenant{
		Id:        vm.ID,
		Name:      vm.Name,
		CreatedAt: vm.CreatedAt,
		UpdatedAt: vm.UpdatedAt,
	}
}

// -----------------------------------------------------------------------------
// List
// -----------------------------------------------------------------------------

type ListStore interface {
	ListTenants(ctx context.Context) ([]dbgen.Tenant, error)
}

type ListEndpoint struct{ Store ListStore }

func (e ListEndpoint) Interact(ctx context.Context, _ apigen.ListTenantsRequestObject) ([]dbgen.Tenant, error) {
	return e.Store.ListTenants(ctx)
}

func (e ListEndpoint) Build(ts []dbgen.Tenant) []ViewModel {
	out := make([]ViewModel, len(ts))
	for i, t := range ts {
		out[i] = toViewModel(t)
	}
	return out
}

func (e ListEndpoint) Render(vms []ViewModel) apigen.ListTenantsResponseObject {
	resp := make(apigen.ListTenants200JSONResponse, len(vms))
	for i, vm := range vms {
		resp[i] = toAPI(vm)
	}
	return resp
}

// -----------------------------------------------------------------------------
// Get
// -----------------------------------------------------------------------------

type GetStore interface {
	GetTenant(ctx context.Context, id string) (dbgen.Tenant, error)
}

type GetEndpoint struct{ Store GetStore }

func (e GetEndpoint) Interact(ctx context.Context, req apigen.GetTenantRequestObject) (dbgen.Tenant, error) {
	return e.Store.GetTenant(ctx, req.TenantId)
}

func (e GetEndpoint) Build(t dbgen.Tenant) ViewModel { return toViewModel(t) }

func (e GetEndpoint) Render(vm ViewModel) apigen.GetTenantResponseObject {
	return apigen.GetTenant200JSONResponse(toAPI(vm))
}

// -----------------------------------------------------------------------------
// Create
// -----------------------------------------------------------------------------

type CreateStore interface {
	CreateTenant(ctx context.Context, arg dbgen.CreateTenantParams) (dbgen.Tenant, error)
}

type CreateEndpoint struct{ Store CreateStore }

func (e CreateEndpoint) Interact(ctx context.Context, req apigen.CreateTenantRequestObject) (dbgen.Tenant, error) {
	return e.Store.CreateTenant(ctx, dbgen.CreateTenantParams{
		ID:   xid.New().String(),
		Name: req.Body.Name,
	})
}

func (e CreateEndpoint) Build(t dbgen.Tenant) ViewModel { return toViewModel(t) }

func (e CreateEndpoint) Render(vm ViewModel) apigen.CreateTenantResponseObject {
	return apigen.CreateTenant201JSONResponse(toAPI(vm))
}

// -----------------------------------------------------------------------------
// Update
// -----------------------------------------------------------------------------

type UpdateStore interface {
	UpdateTenant(ctx context.Context, arg dbgen.UpdateTenantParams) (dbgen.Tenant, error)
}

type UpdateEndpoint struct{ Store UpdateStore }

func (e UpdateEndpoint) Interact(ctx context.Context, req apigen.UpdateTenantRequestObject) (dbgen.Tenant, error) {
	return e.Store.UpdateTenant(ctx, dbgen.UpdateTenantParams{
		ID:   req.TenantId,
		Name: req.Body.Name,
	})
}

func (e UpdateEndpoint) Build(t dbgen.Tenant) ViewModel { return toViewModel(t) }

func (e UpdateEndpoint) Render(vm ViewModel) apigen.UpdateTenantResponseObject {
	return apigen.UpdateTenant200JSONResponse(toAPI(vm))
}

// -----------------------------------------------------------------------------
// Delete
// -----------------------------------------------------------------------------

type DeleteStore interface {
	DeleteTenant(ctx context.Context, id string) error
}

type DeleteEndpoint struct{ Store DeleteStore }

// Delete is a special case — there is no domain model, only success/failure.
// We satisfy Endpoint by carrying a bool through Build/Render.
func (e DeleteEndpoint) Interact(ctx context.Context, req apigen.DeleteTenantRequestObject) (bool, error) {
	return true, e.Store.DeleteTenant(ctx, req.TenantId)
}

func (e DeleteEndpoint) Build(_ bool) bool { return true }

func (e DeleteEndpoint) Render(_ bool) apigen.DeleteTenantResponseObject {
	return apigen.DeleteTenant204Response{}
}
