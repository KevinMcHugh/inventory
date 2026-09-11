// Package kindschema is the domain model for the JSONB payload stored on
// kind_versions.schema. It defines the shape the frontend and MCP tools
// expect, plus a LoadLatest helper that fetches and parses the current
// schema for a given kind.
//
// Kind versions are immutable snapshots — a schema document is one row in
// kind_versions, and bumping the schema means inserting a new row. This
// package does not itself touch the database except through the Querier
// passed to LoadLatest.
package kindschema

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

// VersionLoader is the narrow store surface LoadLatest needs. Both
// *dbgen.Queries and per-endpoint store interfaces that declare this method
// satisfy it — no need to pass the full Querier through.
type VersionLoader interface {
	GetLatestKindVersion(ctx context.Context, kindID string) (dbgen.KindVersion, error)
}

// VersionByIDLoader is the narrow store surface Load needs — one method to
// fetch a KindVersion by its own id.
type VersionByIDLoader interface {
	GetKindVersion(ctx context.Context, id string) (dbgen.KindVersion, error)
}

// FieldType enumerates the value shapes the frontend knows how to render
// and filter. Unknown types fall back to Text on the render side.
type FieldType string

const (
	TypeText    FieldType = "text"
	TypeNumber  FieldType = "number"
	TypeInteger FieldType = "integer"
	TypeBoolean FieldType = "boolean"
	TypeDate    FieldType = "date"
	TypeEnum    FieldType = "enum"
	TypeURL     FieldType = "url"
	TypeTags    FieldType = "tags"
)

// Field describes one property of a model body.
type Field struct {
	Key      string    `json:"key"`
	Label    string    `json:"label,omitempty"`
	Type     FieldType `json:"type"`
	Pinned   bool      `json:"pinned,omitempty"`
	Required bool      `json:"required,omitempty"`
	Indexed  bool      `json:"indexed,omitempty"`

	// enum
	Values []string `json:"values,omitempty"`

	// number / integer
	Min  *float64 `json:"min,omitempty"`
	Max  *float64 `json:"max,omitempty"`
	Unit string   `json:"unit,omitempty"`
}

// Schema is the parsed shape of kind_versions.schema. Array order in Fields
// drives column order in the UI.
type Schema struct {
	Version int     `json:"version"`
	Fields  []Field `json:"fields"`
}

// LoadLatest returns the parsed schema of the most recent KindVersion for a
// kind. When the kind has no versions yet (fresh kind, never had a schema
// authored), it returns a zero-value *Schema so callers do not need to
// nil-check.
func LoadLatest(ctx context.Context, q VersionLoader, kindID string) (*Schema, error) {
	v, err := q.GetLatestKindVersion(ctx, kindID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &Schema{}, nil
		}
		return nil, err
	}
	return Parse(v.Schema)
}

// Load returns the parsed schema of a specific KindVersion by id. Used when
// a model write pins to a version explicitly and we want to validate the
// body against that version's rules rather than the latest.
func Load(ctx context.Context, q VersionByIDLoader, versionID string) (*Schema, error) {
	v, err := q.GetKindVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	return Parse(v.Schema)
}

// Parse unmarshals raw JSONB bytes into a Schema. Empty bytes yield an empty
// (non-nil) Schema so callers do not have to nil-check.
func Parse(raw []byte) (*Schema, error) {
	s := &Schema{}
	if len(raw) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(raw, s); err != nil {
		return nil, err
	}
	return s, nil
}
