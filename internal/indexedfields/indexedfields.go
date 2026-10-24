// Package indexedfields turns a model body into rows for the indexed_fields
// table: a derived, per-field search index that lets Postgres filter and
// sort on JSONB body properties without scanning the whole body.
//
// Which properties get indexed, and as what type, is declared on the
// kind_version's schema (kindschema.Field.Indexed). Extract does the
// schema-driven extraction; Rebuild writes the result for one model, and is
// shared by the model write path and the reindex CLI command.
package indexedfields

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/xid"

	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	"github.com/KevinMcHugh/inventory/internal/kindschema"
)

// Store is the narrow surface Rebuild needs to replace a model's rows.
type Store interface {
	DeleteIndexedFieldsByModel(ctx context.Context, arg dbgen.DeleteIndexedFieldsByModelParams) error
	CreateIndexedField(ctx context.Context, arg dbgen.CreateIndexedFieldParams) (dbgen.IndexedField, error)
}

// Rebuild replaces a model's indexed_fields rows with ones extracted from
// body under schema s. A failure to clear the old rows stops early; a
// failure on one field does not stop the others, and all such failures come
// back joined.
func Rebuild(ctx context.Context, store Store, s *kindschema.Schema, tenantID, kindID, modelID string, body map[string]any) error {
	if err := store.DeleteIndexedFieldsByModel(ctx, dbgen.DeleteIndexedFieldsByModelParams{
		TenantID: tenantID,
		ModelID:  modelID,
	}); err != nil {
		return fmt.Errorf("clear indexed fields: %w", err)
	}
	var errs []error
	for _, row := range Extract(s, tenantID, kindID, modelID, body) {
		if _, err := store.CreateIndexedField(ctx, row); err != nil {
			errs = append(errs, fmt.Errorf("create indexed field %q: %w", row.FieldKey, err))
		}
	}
	return errors.Join(errs...)
}

// Extract computes one indexed_fields row per schema field marked Indexed
// that has a present, well-typed value in body. Fields not marked Indexed,
// fields with no value, and values that do not match their declared type
// are skipped rather than erroring.
func Extract(s *kindschema.Schema, tenantID, kindID, modelID string, body map[string]any) []dbgen.CreateIndexedFieldParams {
	if s == nil {
		return nil
	}
	var rows []dbgen.CreateIndexedFieldParams
	for _, f := range s.Fields {
		if !f.Indexed {
			continue
		}
		v, ok := body[f.Key]
		if !ok || v == nil {
			continue
		}
		row := dbgen.CreateIndexedFieldParams{
			ID:       xid.New().String(),
			TenantID: tenantID,
			KindID:   kindID,
			ModelID:  modelID,
			FieldKey: f.Key,
		}
		if !fillValue(&row, f, v) {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

// fillValue sets the one typed column matching f.Type on row. It reports
// whether v matched the declared type closely enough to index.
func fillValue(row *dbgen.CreateIndexedFieldParams, f kindschema.Field, v any) bool {
	switch f.Type {
	case kindschema.TypeText, kindschema.TypeURL, kindschema.TypeEnum:
		s, ok := v.(string)
		if !ok || s == "" {
			return false
		}
		row.StringValue = &s
		return true

	case kindschema.TypeNumber:
		n, ok := kindschema.Numeric(v)
		if !ok {
			return false
		}
		row.NumericValue = &n
		return true

	case kindschema.TypeInteger:
		n, ok := kindschema.Numeric(v)
		if !ok {
			return false
		}
		i := int64(n)
		row.IntValue = &i
		return true

	case kindschema.TypeBoolean:
		b, ok := v.(bool)
		if !ok {
			return false
		}
		row.BoolValue = &b
		return true

	case kindschema.TypeDate:
		s, ok := v.(string)
		if !ok {
			return false
		}
		t, ok := kindschema.ParseDate(s)
		if !ok {
			return false
		}
		row.DateValue = pgtype.Timestamptz{Time: t, Valid: true}
		return true

	default:
		// TypeTags (and any future array-shaped type) does not fit a single
		// scalar column -- not indexed yet.
		return false
	}
}
