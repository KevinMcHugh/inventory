package kinds

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/xid"

	apigen "github.com/KevinMcHugh/inventory/internal/api/gen"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

// -----------------------------------------------------------------------------
// Kind view-model
// -----------------------------------------------------------------------------

type ViewModel struct {
	ID          string
	TenantID    string
	Name        string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func toViewModel(k dbgen.Kind) ViewModel {
	return ViewModel{
		ID:          k.ID,
		TenantID:    k.TenantID,
		Name:        k.Name,
		Description: k.Description,
		CreatedAt:   k.CreatedAt.Time,
		UpdatedAt:   k.UpdatedAt.Time,
	}
}

func toAPI(vm ViewModel) apigen.Kind {
	return apigen.Kind{
		Id:          vm.ID,
		TenantId:    vm.TenantID,
		Name:        vm.Name,
		Description: vm.Description,
		CreatedAt:   vm.CreatedAt,
		UpdatedAt:   vm.UpdatedAt,
	}
}

// -----------------------------------------------------------------------------
// List Kinds
// -----------------------------------------------------------------------------

type ListStore interface {
	ListKindsByTenant(ctx context.Context, tenantID string) ([]dbgen.Kind, error)
}

type ListEndpoint struct{ Store ListStore }

func (e ListEndpoint) Interact(ctx context.Context, req apigen.ListKindsRequestObject) ([]dbgen.Kind, error) {
	return e.Store.ListKindsByTenant(ctx, req.TenantId)
}

func (e ListEndpoint) Build(ks []dbgen.Kind) []ViewModel {
	out := make([]ViewModel, len(ks))
	for i, k := range ks {
		out[i] = toViewModel(k)
	}
	return out
}

func (e ListEndpoint) Render(vms []ViewModel) apigen.ListKindsResponseObject {
	resp := make(apigen.ListKinds200JSONResponse, len(vms))
	for i, vm := range vms {
		resp[i] = toAPI(vm)
	}
	return resp
}

// -----------------------------------------------------------------------------
// Get Kind
// -----------------------------------------------------------------------------

type GetStore interface {
	GetKind(ctx context.Context, arg dbgen.GetKindParams) (dbgen.Kind, error)
}

type GetEndpoint struct{ Store GetStore }

func (e GetEndpoint) Interact(ctx context.Context, req apigen.GetKindRequestObject) (dbgen.Kind, error) {
	return e.Store.GetKind(ctx, dbgen.GetKindParams{ID: req.KindId, TenantID: req.TenantId})
}

func (e GetEndpoint) Build(k dbgen.Kind) ViewModel { return toViewModel(k) }

func (e GetEndpoint) Render(vm ViewModel) apigen.GetKindResponseObject {
	return apigen.GetKind200JSONResponse(toAPI(vm))
}

// -----------------------------------------------------------------------------
// Create Kind
// -----------------------------------------------------------------------------

type CreateStore interface {
	CreateKind(ctx context.Context, arg dbgen.CreateKindParams) (dbgen.Kind, error)
}

type CreateEndpoint struct{ Store CreateStore }

func (e CreateEndpoint) Interact(ctx context.Context, req apigen.CreateKindRequestObject) (dbgen.Kind, error) {
	return e.Store.CreateKind(ctx, dbgen.CreateKindParams{
		ID:          xid.New().String(),
		TenantID:    req.TenantId,
		Name:        req.Body.Name,
		Description: req.Body.Description,
	})
}

func (e CreateEndpoint) Build(k dbgen.Kind) ViewModel { return toViewModel(k) }

func (e CreateEndpoint) Render(vm ViewModel) apigen.CreateKindResponseObject {
	return apigen.CreateKind201JSONResponse(toAPI(vm))
}

// -----------------------------------------------------------------------------
// Update Kind
// -----------------------------------------------------------------------------

type UpdateStore interface {
	UpdateKind(ctx context.Context, arg dbgen.UpdateKindParams) (dbgen.Kind, error)
}

type UpdateEndpoint struct{ Store UpdateStore }

func (e UpdateEndpoint) Interact(ctx context.Context, req apigen.UpdateKindRequestObject) (dbgen.Kind, error) {
	return e.Store.UpdateKind(ctx, dbgen.UpdateKindParams{
		ID:          req.KindId,
		TenantID:    req.TenantId,
		Name:        req.Body.Name,
		Description: req.Body.Description,
	})
}

func (e UpdateEndpoint) Build(k dbgen.Kind) ViewModel { return toViewModel(k) }

func (e UpdateEndpoint) Render(vm ViewModel) apigen.UpdateKindResponseObject {
	return apigen.UpdateKind200JSONResponse(toAPI(vm))
}

// -----------------------------------------------------------------------------
// Delete Kind
// -----------------------------------------------------------------------------

type DeleteStore interface {
	DeleteKind(ctx context.Context, arg dbgen.DeleteKindParams) error
}

type DeleteEndpoint struct{ Store DeleteStore }

func (e DeleteEndpoint) Interact(ctx context.Context, req apigen.DeleteKindRequestObject) (bool, error) {
	return true, e.Store.DeleteKind(ctx, dbgen.DeleteKindParams{ID: req.KindId, TenantID: req.TenantId})
}

func (e DeleteEndpoint) Build(_ bool) bool { return true }

func (e DeleteEndpoint) Render(_ bool) apigen.DeleteKindResponseObject {
	return apigen.DeleteKind204Response{}
}

// -----------------------------------------------------------------------------
// KindVersion view-model
// -----------------------------------------------------------------------------

type VersionViewModel struct {
	ID        string
	KindID    string
	Schema    map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

func toVersionViewModel(v dbgen.KindVersion) VersionViewModel {
	var schema map[string]any
	if len(v.Schema) > 0 {
		_ = json.Unmarshal(v.Schema, &schema)
	}
	return VersionViewModel{
		ID:        v.ID,
		KindID:    v.KindID,
		Schema:    schema,
		CreatedAt: v.CreatedAt.Time,
		UpdatedAt: v.UpdatedAt.Time,
	}
}

func toVersionAPI(vm VersionViewModel) apigen.KindVersion {
	return apigen.KindVersion{
		Id:        vm.ID,
		KindId:    vm.KindID,
		Schema:    vm.Schema,
		CreatedAt: vm.CreatedAt,
		UpdatedAt: vm.UpdatedAt,
	}
}

// -----------------------------------------------------------------------------
// List KindVersions
// -----------------------------------------------------------------------------

type ListVersionsStore interface {
	ListKindVersionsByKind(ctx context.Context, kindID string) ([]dbgen.KindVersion, error)
}

type ListVersionsEndpoint struct{ Store ListVersionsStore }

func (e ListVersionsEndpoint) Interact(ctx context.Context, req apigen.ListKindVersionsRequestObject) ([]dbgen.KindVersion, error) {
	return e.Store.ListKindVersionsByKind(ctx, req.KindId)
}

func (e ListVersionsEndpoint) Build(vs []dbgen.KindVersion) []VersionViewModel {
	out := make([]VersionViewModel, len(vs))
	for i, v := range vs {
		out[i] = toVersionViewModel(v)
	}
	return out
}

func (e ListVersionsEndpoint) Render(vms []VersionViewModel) apigen.ListKindVersionsResponseObject {
	resp := make(apigen.ListKindVersions200JSONResponse, len(vms))
	for i, vm := range vms {
		resp[i] = toVersionAPI(vm)
	}
	return resp
}

// -----------------------------------------------------------------------------
// Create KindVersion
// -----------------------------------------------------------------------------

type CreateVersionStore interface {
	CreateKindVersion(ctx context.Context, arg dbgen.CreateKindVersionParams) (dbgen.KindVersion, error)
}

type CreateVersionEndpoint struct{ Store CreateVersionStore }

func (e CreateVersionEndpoint) Interact(ctx context.Context, req apigen.CreateKindVersionRequestObject) (dbgen.KindVersion, error) {
	schemaBytes, err := json.Marshal(req.Body.Schema)
	if err != nil {
		return dbgen.KindVersion{}, err
	}
	return e.Store.CreateKindVersion(ctx, dbgen.CreateKindVersionParams{
		ID:     xid.New().String(),
		KindID: req.KindId,
		Schema: schemaBytes,
	})
}

func (e CreateVersionEndpoint) Build(v dbgen.KindVersion) VersionViewModel {
	return toVersionViewModel(v)
}

func (e CreateVersionEndpoint) Render(vm VersionViewModel) apigen.CreateKindVersionResponseObject {
	return apigen.CreateKindVersion201JSONResponse(toVersionAPI(vm))
}
