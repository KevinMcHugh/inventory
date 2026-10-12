package models

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/KevinMcHugh/inventory/internal/api/gen"
	"github.com/KevinMcHugh/inventory/internal/auth"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	"github.com/KevinMcHugh/inventory/internal/kindschema"
)

const (
	defaultSearchLimit = 50
	maxSearchLimit     = 200
)

// SearchStore is the store surface SearchEndpoint needs: one query per
// indexed_fields value type (plus a sort-only pair with no filter join),
// and a way to load the kind's current schema to resolve a field's type.
type SearchStore interface {
	GetLatestKindVersion(ctx context.Context, kindID string) (dbgen.KindVersion, error)
	SortModelsByFieldAsc(ctx context.Context, arg dbgen.SortModelsByFieldAscParams) ([]dbgen.Model, error)
	SortModelsByFieldDesc(ctx context.Context, arg dbgen.SortModelsByFieldDescParams) ([]dbgen.Model, error)
	SearchModelsByStringFieldAsc(ctx context.Context, arg dbgen.SearchModelsByStringFieldAscParams) ([]dbgen.Model, error)
	SearchModelsByStringFieldDesc(ctx context.Context, arg dbgen.SearchModelsByStringFieldDescParams) ([]dbgen.Model, error)
	SearchModelsByNumericFieldAsc(ctx context.Context, arg dbgen.SearchModelsByNumericFieldAscParams) ([]dbgen.Model, error)
	SearchModelsByNumericFieldDesc(ctx context.Context, arg dbgen.SearchModelsByNumericFieldDescParams) ([]dbgen.Model, error)
	SearchModelsByIntFieldAsc(ctx context.Context, arg dbgen.SearchModelsByIntFieldAscParams) ([]dbgen.Model, error)
	SearchModelsByIntFieldDesc(ctx context.Context, arg dbgen.SearchModelsByIntFieldDescParams) ([]dbgen.Model, error)
	SearchModelsByBoolFieldAsc(ctx context.Context, arg dbgen.SearchModelsByBoolFieldAscParams) ([]dbgen.Model, error)
	SearchModelsByBoolFieldDesc(ctx context.Context, arg dbgen.SearchModelsByBoolFieldDescParams) ([]dbgen.Model, error)
	SearchModelsByDateFieldAsc(ctx context.Context, arg dbgen.SearchModelsByDateFieldAscParams) ([]dbgen.Model, error)
	SearchModelsByDateFieldDesc(ctx context.Context, arg dbgen.SearchModelsByDateFieldDescParams) ([]dbgen.Model, error)
}

type SearchEndpoint struct{ Store SearchStore }

func (e SearchEndpoint) Interact(ctx context.Context, req apigen.SearchModelsRequestObject) ([]dbgen.Model, error) {
	tenantID, err := auth.TenantID(ctx)
	if err != nil {
		return nil, err
	}

	p := req.Params
	limit, offset := searchPage(p)
	desc := p.SortOrder != nil && *p.SortOrder == apigen.Desc
	sortField := strOr(p.SortField, "")

	filterField := strOr(p.FilterField, "")
	if filterField == "" {
		// No filter: sort (or plain browse, when sortField is also empty --
		// an empty field_key never matches an indexed_fields row, so every
		// query here falls back to the m.created_at DESC tiebreaker, same
		// order ListModels already uses).
		common := dbgen.SortModelsByFieldAscParams{
			TenantID: tenantID, KindID: req.KindId, SortField: sortField,
			LimitCount: limit, OffsetCount: offset,
		}
		if desc {
			return e.Store.SortModelsByFieldDesc(ctx, dbgen.SortModelsByFieldDescParams(common))
		}
		return e.Store.SortModelsByFieldAsc(ctx, common)
	}
	if sortField == "" {
		sortField = filterField
	}

	schema, err := kindschema.LoadLatest(ctx, e.Store, req.KindId)
	if err != nil {
		return nil, err
	}
	field, ok := findField(schema, filterField)
	if !ok || !field.Indexed {
		return nil, kindschema.ValidationErrors{{Field: "filterField", Message: "not an indexed field on this kind"}}
	}

	return e.searchTyped(ctx, field, tenantID, req.KindId, filterField, sortField, p, limit, offset, desc)
}

func (e SearchEndpoint) Build(ms []dbgen.Model) []ViewModel {
	out := make([]ViewModel, len(ms))
	for i, m := range ms {
		out[i] = ToViewModel(m)
	}
	return out
}

func (e SearchEndpoint) Render(vms []ViewModel) apigen.SearchModelsResponseObject {
	resp := make(apigen.SearchModels200JSONResponse, len(vms))
	for i, vm := range vms {
		resp[i] = toAPI(vm)
	}
	return resp
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

func findField(s *kindschema.Schema, key string) (kindschema.Field, bool) {
	if s == nil {
		return kindschema.Field{}, false
	}
	for _, f := range s.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return kindschema.Field{}, false
}

func strOr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

func searchPage(p apigen.SearchModelsParams) (limit, offset int32) {
	limit = defaultSearchLimit
	if p.Limit != nil {
		limit = int32(*p.Limit)
		if limit < 1 {
			limit = 1
		}
		if limit > maxSearchLimit {
			limit = maxSearchLimit
		}
	}
	if p.Offset != nil && *p.Offset > 0 {
		offset = int32(*p.Offset)
	}
	return limit, offset
}

// nonEmpty treats an omitted or blank query param as "not provided" --
// oapi-codegen cannot tell "?eq=" apart from "?eq=" with no value, and
// neither is a meaningful filter value.
func nonEmpty(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

// searchTyped resolves which of the five indexed_fields value types the
// filter field uses and dispatches to the matching sqlc query, parsing the
// raw string query params into that type along the way. Any parse failure
// or operator not valid for the field's type comes back as a
// kindschema.ValidationErrors, which server.go renders as 400.
func (e SearchEndpoint) searchTyped(
	ctx context.Context,
	field kindschema.Field,
	tenantID, kindID, filterField, sortField string,
	p apigen.SearchModelsParams,
	limit, offset int32,
	desc bool,
) ([]dbgen.Model, error) {
	switch field.Type {
	case kindschema.TypeText, kindschema.TypeURL, kindschema.TypeEnum:
		if err := rejectOps(
			disallowedOp{"lt", nonEmpty(p.Lt)}, disallowedOp{"lte", nonEmpty(p.Lte)},
			disallowedOp{"gt", nonEmpty(p.Gt)}, disallowedOp{"gte", nonEmpty(p.Gte)},
		); err != nil {
			return nil, err
		}
		args := dbgen.SearchModelsByStringFieldAscParams{
			TenantID: tenantID, KindID: kindID, FilterField: filterField, SortField: sortField,
			Eq: nonEmpty(p.Eq), Ne: nonEmpty(p.Ne), Contains: nonEmpty(p.Contains),
			LimitCount: limit, OffsetCount: offset,
		}
		if desc {
			return e.Store.SearchModelsByStringFieldDesc(ctx, dbgen.SearchModelsByStringFieldDescParams(args))
		}
		return e.Store.SearchModelsByStringFieldAsc(ctx, args)

	case kindschema.TypeNumber:
		if err := rejectOps(disallowedOp{"contains", nonEmpty(p.Contains)}); err != nil {
			return nil, err
		}
		eq, ne, lt, lte, gt, gte, err := parseFloatOps(p)
		if err != nil {
			return nil, err
		}
		args := dbgen.SearchModelsByNumericFieldAscParams{
			TenantID: tenantID, KindID: kindID, FilterField: filterField, SortField: sortField,
			Eq: eq, Ne: ne, Lt: lt, Lte: lte, Gt: gt, Gte: gte,
			LimitCount: limit, OffsetCount: offset,
		}
		if desc {
			return e.Store.SearchModelsByNumericFieldDesc(ctx, dbgen.SearchModelsByNumericFieldDescParams(args))
		}
		return e.Store.SearchModelsByNumericFieldAsc(ctx, args)

	case kindschema.TypeInteger:
		if err := rejectOps(disallowedOp{"contains", nonEmpty(p.Contains)}); err != nil {
			return nil, err
		}
		eq, ne, lt, lte, gt, gte, err := parseIntOps(p)
		if err != nil {
			return nil, err
		}
		args := dbgen.SearchModelsByIntFieldAscParams{
			TenantID: tenantID, KindID: kindID, FilterField: filterField, SortField: sortField,
			Eq: eq, Ne: ne, Lt: lt, Lte: lte, Gt: gt, Gte: gte,
			LimitCount: limit, OffsetCount: offset,
		}
		if desc {
			return e.Store.SearchModelsByIntFieldDesc(ctx, dbgen.SearchModelsByIntFieldDescParams(args))
		}
		return e.Store.SearchModelsByIntFieldAsc(ctx, args)

	case kindschema.TypeBoolean:
		if err := rejectOps(
			disallowedOp{"lt", nonEmpty(p.Lt)}, disallowedOp{"lte", nonEmpty(p.Lte)},
			disallowedOp{"gt", nonEmpty(p.Gt)}, disallowedOp{"gte", nonEmpty(p.Gte)},
			disallowedOp{"contains", nonEmpty(p.Contains)},
		); err != nil {
			return nil, err
		}
		eq, ne, err := parseBoolOps(p)
		if err != nil {
			return nil, err
		}
		args := dbgen.SearchModelsByBoolFieldAscParams{
			TenantID: tenantID, KindID: kindID, FilterField: filterField, SortField: sortField,
			Eq: eq, Ne: ne,
			LimitCount: limit, OffsetCount: offset,
		}
		if desc {
			return e.Store.SearchModelsByBoolFieldDesc(ctx, dbgen.SearchModelsByBoolFieldDescParams(args))
		}
		return e.Store.SearchModelsByBoolFieldAsc(ctx, args)

	case kindschema.TypeDate:
		if err := rejectOps(disallowedOp{"contains", nonEmpty(p.Contains)}); err != nil {
			return nil, err
		}
		eq, ne, lt, lte, gt, gte, err := parseDateOps(p)
		if err != nil {
			return nil, err
		}
		args := dbgen.SearchModelsByDateFieldAscParams{
			TenantID: tenantID, KindID: kindID, FilterField: filterField, SortField: sortField,
			Eq: eq, Ne: ne, Lt: lt, Lte: lte, Gt: gt, Gte: gte,
			LimitCount: limit, OffsetCount: offset,
		}
		if desc {
			return e.Store.SearchModelsByDateFieldDesc(ctx, dbgen.SearchModelsByDateFieldDescParams(args))
		}
		return e.Store.SearchModelsByDateFieldAsc(ctx, args)

	default:
		return nil, kindschema.ValidationErrors{{Field: "filterField", Message: "fields of type " + string(field.Type) + " are not searchable"}}
	}
}

// disallowedOp names a query param that was set but is not valid for the
// filter field's type.
type disallowedOp struct {
	name string
	val  *string
}

// rejectOps returns a ValidationErrors naming every op whose val is
// non-nil -- used to flag operators the filter field's type does not
// support (e.g. "contains" on a number field).
func rejectOps(ops ...disallowedOp) kindschema.ValidationErrors {
	var errs kindschema.ValidationErrors
	for _, o := range ops {
		if o.val != nil {
			errs = append(errs, kindschema.ValidationError{Field: o.name, Message: "not valid for this field's type"})
		}
	}
	return errs
}

func parseFloatOps(p apigen.SearchModelsParams) (eq, ne, lt, lte, gt, gte *float64, err error) {
	fields := []struct {
		name string
		raw  *string
		out  **float64
	}{
		{"eq", nonEmpty(p.Eq), &eq}, {"ne", nonEmpty(p.Ne), &ne},
		{"lt", nonEmpty(p.Lt), &lt}, {"lte", nonEmpty(p.Lte), &lte},
		{"gt", nonEmpty(p.Gt), &gt}, {"gte", nonEmpty(p.Gte), &gte},
	}
	var errs kindschema.ValidationErrors
	for _, f := range fields {
		if f.raw == nil {
			continue
		}
		v, perr := strconv.ParseFloat(*f.raw, 64)
		if perr != nil {
			errs = append(errs, kindschema.ValidationError{Field: f.name, Message: "not a number"})
			continue
		}
		*f.out = &v
	}
	if len(errs) > 0 {
		return nil, nil, nil, nil, nil, nil, errs
	}
	return eq, ne, lt, lte, gt, gte, nil
}

func parseIntOps(p apigen.SearchModelsParams) (eq, ne, lt, lte, gt, gte *int64, err error) {
	fields := []struct {
		name string
		raw  *string
		out  **int64
	}{
		{"eq", nonEmpty(p.Eq), &eq}, {"ne", nonEmpty(p.Ne), &ne},
		{"lt", nonEmpty(p.Lt), &lt}, {"lte", nonEmpty(p.Lte), &lte},
		{"gt", nonEmpty(p.Gt), &gt}, {"gte", nonEmpty(p.Gte), &gte},
	}
	var errs kindschema.ValidationErrors
	for _, f := range fields {
		if f.raw == nil {
			continue
		}
		v, perr := strconv.ParseInt(*f.raw, 10, 64)
		if perr != nil {
			errs = append(errs, kindschema.ValidationError{Field: f.name, Message: "not a whole number"})
			continue
		}
		*f.out = &v
	}
	if len(errs) > 0 {
		return nil, nil, nil, nil, nil, nil, errs
	}
	return eq, ne, lt, lte, gt, gte, nil
}

func parseBoolOps(p apigen.SearchModelsParams) (eq, ne *bool, err error) {
	fields := []struct {
		name string
		raw  *string
		out  **bool
	}{
		{"eq", nonEmpty(p.Eq), &eq}, {"ne", nonEmpty(p.Ne), &ne},
	}
	var errs kindschema.ValidationErrors
	for _, f := range fields {
		if f.raw == nil {
			continue
		}
		v, perr := strconv.ParseBool(*f.raw)
		if perr != nil {
			errs = append(errs, kindschema.ValidationError{Field: f.name, Message: "expected true or false"})
			continue
		}
		*f.out = &v
	}
	if len(errs) > 0 {
		return nil, nil, errs
	}
	return eq, ne, nil
}

func parseDateOps(p apigen.SearchModelsParams) (eq, ne, lt, lte, gt, gte pgtype.Timestamptz, err error) {
	fields := []struct {
		name string
		raw  *string
		out  *pgtype.Timestamptz
	}{
		{"eq", nonEmpty(p.Eq), &eq}, {"ne", nonEmpty(p.Ne), &ne},
		{"lt", nonEmpty(p.Lt), &lt}, {"lte", nonEmpty(p.Lte), &lte},
		{"gt", nonEmpty(p.Gt), &gt}, {"gte", nonEmpty(p.Gte), &gte},
	}
	var errs kindschema.ValidationErrors
	for _, f := range fields {
		if f.raw == nil {
			continue
		}
		t, ok := kindschema.ParseDate(*f.raw)
		if !ok {
			errs = append(errs, kindschema.ValidationError{Field: f.name, Message: "expected an ISO date string"})
			continue
		}
		*f.out = pgtype.Timestamptz{Time: t, Valid: true}
	}
	if len(errs) > 0 {
		return pgtype.Timestamptz{}, pgtype.Timestamptz{}, pgtype.Timestamptz{}, pgtype.Timestamptz{}, pgtype.Timestamptz{}, pgtype.Timestamptz{}, errs
	}
	return eq, ne, lt, lte, gt, gte, nil
}
