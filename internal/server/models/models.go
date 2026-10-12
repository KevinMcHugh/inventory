package models

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/rs/xid"

	apigen "github.com/KevinMcHugh/inventory/internal/api/gen"
	"github.com/KevinMcHugh/inventory/internal/auth"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	"github.com/KevinMcHugh/inventory/internal/indexedfields"
	"github.com/KevinMcHugh/inventory/internal/kindschema"
)

// -----------------------------------------------------------------------------
// View-model
// -----------------------------------------------------------------------------

type ViewModel struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenantId"`
	KindID        string         `json:"kindId"`
	KindVersionID string         `json:"kindVersionId"`
	Slug          string         `json:"slug"`
	Body          map[string]any `json:"body"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

func ToViewModel(m dbgen.Model) ViewModel {
	var body map[string]any
	if len(m.Body) > 0 {
		_ = json.Unmarshal(m.Body, &body)
	}
	return ViewModel{
		ID:            m.ID,
		TenantID:      m.TenantID,
		KindID:        m.KindID,
		KindVersionID: m.KindVersionID,
		Slug:          m.Slug,
		Body:          body,
		CreatedAt:     m.CreatedAt.Time,
		UpdatedAt:     m.UpdatedAt.Time,
	}
}

func toAPI(vm ViewModel) apigen.Model {
	return apigen.Model{
		Id:            vm.ID,
		TenantId:      vm.TenantID,
		KindId:        vm.KindID,
		KindVersionId: vm.KindVersionID,
		Slug:          vm.Slug,
		Body:          vm.Body,
		CreatedAt:     vm.CreatedAt,
		UpdatedAt:     vm.UpdatedAt,
	}
}

// -----------------------------------------------------------------------------
// List Models
// -----------------------------------------------------------------------------

type ListStore interface {
	ListModelsByKind(ctx context.Context, arg dbgen.ListModelsByKindParams) ([]dbgen.Model, error)
}

type ListEndpoint struct{ Store ListStore }

func (e ListEndpoint) Interact(ctx context.Context, req apigen.ListModelsRequestObject) ([]dbgen.Model, error) {
	tenantID, err := auth.TenantID(ctx)
	if err != nil {
		return nil, err
	}
	return e.Store.ListModelsByKind(ctx, dbgen.ListModelsByKindParams{
		TenantID: tenantID,
		KindID:   req.KindId,
	})
}

func (e ListEndpoint) Build(ms []dbgen.Model) []ViewModel {
	out := make([]ViewModel, len(ms))
	for i, m := range ms {
		out[i] = ToViewModel(m)
	}
	return out
}

func (e ListEndpoint) Render(vms []ViewModel) apigen.ListModelsResponseObject {
	resp := make(apigen.ListModels200JSONResponse, len(vms))
	for i, vm := range vms {
		resp[i] = toAPI(vm)
	}
	return resp
}

// -----------------------------------------------------------------------------
// Get Model
// -----------------------------------------------------------------------------

type GetStore interface {
	GetModelBySlug(ctx context.Context, arg dbgen.GetModelBySlugParams) (dbgen.Model, error)
}

type GetEndpoint struct{ Store GetStore }

func (e GetEndpoint) Interact(ctx context.Context, req apigen.GetModelRequestObject) (dbgen.Model, error) {
	tenantID, err := auth.TenantID(ctx)
	if err != nil {
		return dbgen.Model{}, err
	}
	return e.Store.GetModelBySlug(ctx, dbgen.GetModelBySlugParams{
		TenantID: tenantID,
		KindID:   req.KindId,
		Slug:     req.Slug,
	})
}

func (e GetEndpoint) Build(m dbgen.Model) ViewModel { return ToViewModel(m) }

func (e GetEndpoint) Render(vm ViewModel) apigen.GetModelResponseObject {
	return apigen.GetModel200JSONResponse(toAPI(vm))
}

// -----------------------------------------------------------------------------
// Create Model
// -----------------------------------------------------------------------------

type CreateStore interface {
	CreateModel(ctx context.Context, arg dbgen.CreateModelParams) (dbgen.Model, error)
	GetLatestKindVersion(ctx context.Context, kindID string) (dbgen.KindVersion, error)
	GetKindVersion(ctx context.Context, id string) (dbgen.KindVersion, error)
	DeleteIndexedFieldsByModel(ctx context.Context, arg dbgen.DeleteIndexedFieldsByModelParams) error
	CreateIndexedField(ctx context.Context, arg dbgen.CreateIndexedFieldParams) (dbgen.IndexedField, error)
}

type CreateEndpoint struct{ Store CreateStore }

func (e CreateEndpoint) Interact(ctx context.Context, req apigen.CreateModelRequestObject) (dbgen.Model, error) {
	tenantID, err := auth.TenantID(ctx)
	if err != nil {
		return dbgen.Model{}, err
	}
	versionID, err := resolveVersion(ctx, e.Store, req.KindId, req.Body.KindVersionId)
	if err != nil {
		return dbgen.Model{}, err
	}
	schema := loadSchema(ctx, e.Store, versionID)
	if errs := kindschema.Validate(schema, req.Body.Body); len(errs) > 0 {
		return dbgen.Model{}, errs
	}
	bodyBytes, err := json.Marshal(req.Body.Body)
	if err != nil {
		return dbgen.Model{}, err
	}
	m, err := e.Store.CreateModel(ctx, dbgen.CreateModelParams{
		ID:            xid.New().String(),
		TenantID:      tenantID,
		KindID:        req.KindId,
		KindVersionID: versionID,
		Slug:          req.Body.Slug,
		Body:          bodyBytes,
	})
	if err != nil {
		return dbgen.Model{}, err
	}
	reindex(ctx, e.Store, schema, tenantID, req.KindId, m.ID, req.Body.Body)
	return m, nil
}

func (e CreateEndpoint) Build(m dbgen.Model) ViewModel { return ToViewModel(m) }

func (e CreateEndpoint) Render(vm ViewModel) apigen.CreateModelResponseObject {
	return apigen.CreateModel201JSONResponse(toAPI(vm))
}

// -----------------------------------------------------------------------------
// Update Model
// -----------------------------------------------------------------------------

type UpdateStore interface {
	UpdateModelBySlug(ctx context.Context, arg dbgen.UpdateModelBySlugParams) (dbgen.Model, error)
	GetLatestKindVersion(ctx context.Context, kindID string) (dbgen.KindVersion, error)
	GetKindVersion(ctx context.Context, id string) (dbgen.KindVersion, error)
	DeleteIndexedFieldsByModel(ctx context.Context, arg dbgen.DeleteIndexedFieldsByModelParams) error
	CreateIndexedField(ctx context.Context, arg dbgen.CreateIndexedFieldParams) (dbgen.IndexedField, error)
}

type UpdateEndpoint struct{ Store UpdateStore }

func (e UpdateEndpoint) Interact(ctx context.Context, req apigen.UpdateModelRequestObject) (dbgen.Model, error) {
	tenantID, err := auth.TenantID(ctx)
	if err != nil {
		return dbgen.Model{}, err
	}
	versionID, err := resolveVersion(ctx, e.Store, req.KindId, req.Body.KindVersionId)
	if err != nil {
		return dbgen.Model{}, err
	}
	schema := loadSchema(ctx, e.Store, versionID)
	if errs := kindschema.Validate(schema, req.Body.Body); len(errs) > 0 {
		return dbgen.Model{}, errs
	}
	bodyBytes, err := json.Marshal(req.Body.Body)
	if err != nil {
		return dbgen.Model{}, err
	}
	m, err := e.Store.UpdateModelBySlug(ctx, dbgen.UpdateModelBySlugParams{
		TenantID:      tenantID,
		KindID:        req.KindId,
		Slug:          req.Slug,
		Body:          bodyBytes,
		KindVersionID: versionID,
	})
	if err != nil {
		return dbgen.Model{}, err
	}
	reindex(ctx, e.Store, schema, tenantID, req.KindId, m.ID, req.Body.Body)
	return m, nil
}

func (e UpdateEndpoint) Build(m dbgen.Model) ViewModel { return ToViewModel(m) }

func (e UpdateEndpoint) Render(vm ViewModel) apigen.UpdateModelResponseObject {
	return apigen.UpdateModel200JSONResponse(toAPI(vm))
}

// -----------------------------------------------------------------------------
// Delete Model
// -----------------------------------------------------------------------------

type DeleteStore interface {
	DeleteModelBySlug(ctx context.Context, arg dbgen.DeleteModelBySlugParams) error
	DeleteIndexedFieldsByModelSlug(ctx context.Context, arg dbgen.DeleteIndexedFieldsByModelSlugParams) error
}

type DeleteEndpoint struct{ Store DeleteStore }

func (e DeleteEndpoint) Interact(ctx context.Context, req apigen.DeleteModelRequestObject) (bool, error) {
	tenantID, err := auth.TenantID(ctx)
	if err != nil {
		return false, err
	}
	// Best-effort: indexed_fields is a derived index, so a failure here
	// should not block the model delete itself.
	if err := e.Store.DeleteIndexedFieldsByModelSlug(ctx, dbgen.DeleteIndexedFieldsByModelSlugParams{
		TenantID: tenantID,
		KindID:   req.KindId,
		Slug:     req.Slug,
	}); err != nil {
		slog.Error("delete indexed fields", "kindId", req.KindId, "slug", req.Slug, "err", err)
	}
	return true, e.Store.DeleteModelBySlug(ctx, dbgen.DeleteModelBySlugParams{
		TenantID: tenantID,
		KindID:   req.KindId,
		Slug:     req.Slug,
	})
}

func (e DeleteEndpoint) Build(_ bool) bool { return true }

func (e DeleteEndpoint) Render(_ bool) apigen.DeleteModelResponseObject {
	return apigen.DeleteModel204Response{}
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

type versionResolver interface {
	GetLatestKindVersion(ctx context.Context, kindID string) (dbgen.KindVersion, error)
}

func resolveVersion(ctx context.Context, s versionResolver, kindID string, provided *string) (string, error) {
	if provided != nil && *provided != "" {
		return *provided, nil
	}
	v, err := s.GetLatestKindVersion(ctx, kindID)
	if err != nil {
		return "", err
	}
	return v.ID, nil
}

// loadSchema loads the schema pinned by versionID. A missing/broken schema
// should not block writes, so it comes back as an empty (non-nil) Schema
// rather than an error -- kindschema.Validate treats that as no rules, and
// Extract simply finds no indexed fields to compute.
func loadSchema(ctx context.Context, q kindschema.VersionByIDLoader, versionID string) *kindschema.Schema {
	s, err := kindschema.Load(ctx, q, versionID)
	if err != nil {
		return &kindschema.Schema{}
	}
	return s
}

// indexStore is the narrow surface reindex needs to refresh a model's
// indexed_fields rows.
type indexStore interface {
	DeleteIndexedFieldsByModel(ctx context.Context, arg dbgen.DeleteIndexedFieldsByModelParams) error
	CreateIndexedField(ctx context.Context, arg dbgen.CreateIndexedFieldParams) (dbgen.IndexedField, error)
}

// reindex refreshes a model's indexed_fields rows to match its current body.
// It runs after the model write has already succeeded and is best-effort:
// indexed_fields is a derived search index, not the model's source of truth,
// so a failure here is logged rather than surfaced as a write failure.
func reindex(ctx context.Context, store indexStore, schema *kindschema.Schema, tenantID, kindID, modelID string, body map[string]any) {
	if err := store.DeleteIndexedFieldsByModel(ctx, dbgen.DeleteIndexedFieldsByModelParams{
		TenantID: tenantID,
		ModelID:  modelID,
	}); err != nil {
		slog.Error("clear indexed fields", "modelId", modelID, "err", err)
		return
	}
	for _, row := range indexedfields.Extract(schema, tenantID, kindID, modelID, body) {
		if _, err := store.CreateIndexedField(ctx, row); err != nil {
			slog.Error("create indexed field", "modelId", modelID, "field", row.FieldKey, "err", err)
		}
	}
}
