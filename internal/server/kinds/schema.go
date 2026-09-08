package kinds

import (
	"context"

	apigen "github.com/KevinMcHugh/inventory/internal/api/gen"
	"github.com/KevinMcHugh/inventory/internal/auth"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	"github.com/KevinMcHugh/inventory/internal/kindschema"
)

// -----------------------------------------------------------------------------
// Get Schema
// -----------------------------------------------------------------------------

// GetSchemaStore is the minimum surface the schema endpoint needs.
type GetSchemaStore interface {
	GetKind(ctx context.Context, arg dbgen.GetKindParams) (dbgen.Kind, error)
	GetLatestKindVersion(ctx context.Context, kindID string) (dbgen.KindVersion, error)
}

type GetSchemaEndpoint struct{ Store GetSchemaStore }

func (e GetSchemaEndpoint) Interact(ctx context.Context, req apigen.GetKindSchemaRequestObject) (*kindschema.Schema, error) {
	tenantID, err := auth.TenantID(ctx)
	if err != nil {
		return nil, err
	}
	// Confirm the kind belongs to the caller's tenant before returning any
	// schema — this both scopes the read and produces the 404 the API contract
	// promises for unknown kinds.
	if _, err := e.Store.GetKind(ctx, dbgen.GetKindParams{
		ID:       req.KindId,
		TenantID: tenantID,
	}); err != nil {
		return nil, err
	}
	return kindschema.LoadLatest(ctx, e.Store, req.KindId)
}

func (e GetSchemaEndpoint) Build(s *kindschema.Schema) *kindschema.Schema { return s }

func (e GetSchemaEndpoint) Render(s *kindschema.Schema) apigen.GetKindSchemaResponseObject {
	return apigen.GetKindSchema200JSONResponse(toAPISchema(s))
}

// -----------------------------------------------------------------------------
// View conversion
// -----------------------------------------------------------------------------

// toAPISchema converts kindschema.Schema (domain) to apigen.Schema (wire).
// Both types are structurally the same; the copy is here so we do not leak
// domain types into HTTP-shaped fields.
func toAPISchema(s *kindschema.Schema) apigen.Schema {
	if s == nil {
		return apigen.Schema{Fields: []apigen.Field{}}
	}
	out := apigen.Schema{Fields: make([]apigen.Field, len(s.Fields))}
	if s.Version != 0 {
		v := s.Version
		out.Version = &v
	}
	for i, f := range s.Fields {
		out.Fields[i] = toAPIField(f)
	}
	return out
}

func toAPIField(f kindschema.Field) apigen.Field {
	af := apigen.Field{
		Key:  f.Key,
		Type: apigen.FieldType(f.Type),
	}
	if f.Label != "" {
		l := f.Label
		af.Label = &l
	}
	if f.Pinned {
		p := true
		af.Pinned = &p
	}
	if len(f.Values) > 0 {
		v := f.Values
		af.Values = &v
	}
	if f.Min != nil {
		v := float32(*f.Min)
		af.Min = &v
	}
	if f.Max != nil {
		v := float32(*f.Max)
		af.Max = &v
	}
	if f.Unit != "" {
		u := f.Unit
		af.Unit = &u
	}
	return af
}
