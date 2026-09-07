package tenants

import (
	"context"
	"time"

	apigen "github.com/KevinMcHugh/inventory/internal/api/gen"
	"github.com/KevinMcHugh/inventory/internal/auth"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

// ViewModel is the presentation form of a tenant.
type ViewModel struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func ToViewModel(t dbgen.Tenant) ViewModel {
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
// Get self
// -----------------------------------------------------------------------------

type GetStore interface {
	GetTenant(ctx context.Context, id string) (dbgen.Tenant, error)
}

type GetEndpoint struct{ Store GetStore }

func (e GetEndpoint) Interact(ctx context.Context, _ apigen.GetTenantRequestObject) (dbgen.Tenant, error) {
	tenantID, err := auth.TenantID(ctx)
	if err != nil {
		return dbgen.Tenant{}, err
	}
	return e.Store.GetTenant(ctx, tenantID)
}

func (e GetEndpoint) Build(t dbgen.Tenant) ViewModel { return ToViewModel(t) }

func (e GetEndpoint) Render(vm ViewModel) apigen.GetTenantResponseObject {
	return apigen.GetTenant200JSONResponse(toAPI(vm))
}

// -----------------------------------------------------------------------------
// Update self
// -----------------------------------------------------------------------------

type UpdateStore interface {
	UpdateTenant(ctx context.Context, arg dbgen.UpdateTenantParams) (dbgen.Tenant, error)
}

type UpdateEndpoint struct{ Store UpdateStore }

func (e UpdateEndpoint) Interact(ctx context.Context, req apigen.UpdateTenantRequestObject) (dbgen.Tenant, error) {
	tenantID, err := auth.TenantID(ctx)
	if err != nil {
		return dbgen.Tenant{}, err
	}
	return e.Store.UpdateTenant(ctx, dbgen.UpdateTenantParams{
		ID:   tenantID,
		Name: req.Body.Name,
	})
}

func (e UpdateEndpoint) Build(t dbgen.Tenant) ViewModel { return ToViewModel(t) }

func (e UpdateEndpoint) Render(vm ViewModel) apigen.UpdateTenantResponseObject {
	return apigen.UpdateTenant200JSONResponse(toAPI(vm))
}
