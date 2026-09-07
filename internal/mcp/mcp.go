// Package mcp exposes the inventory datastore over the Model Context Protocol
// so Claude (or any MCP client) can list and manipulate models.
//
// Tools declare typed input and output structs; jsonschema is inferred from
// struct tags. Each tool captures the dbgen.Querier in a closure — the same
// store the HTTP endpoints use, so both interfaces see identical data.
package mcp

import (
	"encoding/json"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

// NewServer builds an MCP server with all inventory tools registered.
func NewServer(q dbgen.Querier) *mcpsdk.Server {
	s := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "inventory",
		Version: "0.1.0",
	}, nil)
	registerReadTools(s, q)
	registerWriteTools(s, q)
	return s
}

// -----------------------------------------------------------------------------
// Shared views
// -----------------------------------------------------------------------------

type TenantView struct {
	ID        string    `json:"id" jsonschema:"tenant xid"`
	Name      string    `json:"name" jsonschema:"tenant display name"`
	CreatedAt time.Time `json:"createdAt"`
}

type KindView struct {
	ID          string  `json:"id" jsonschema:"kind xid"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type KindVersionView struct {
	ID     string         `json:"id" jsonschema:"kind version xid"`
	KindID string         `json:"kindId"`
	Schema map[string]any `json:"schema"`
}

type ModelView struct {
	ID            string         `json:"id"`
	Slug          string         `json:"slug"`
	KindVersionID string         `json:"kindVersionId"`
	Body          map[string]any `json:"body"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

func toTenantView(t dbgen.Tenant) TenantView {
	return TenantView{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt.Time}
}

func toKindView(k dbgen.Kind) KindView {
	return KindView{ID: k.ID, Name: k.Name, Description: k.Description}
}

func toKindVersionView(v dbgen.KindVersion) KindVersionView {
	var schema map[string]any
	if len(v.Schema) > 0 {
		_ = json.Unmarshal(v.Schema, &schema)
	}
	return KindVersionView{ID: v.ID, KindID: v.KindID, Schema: schema}
}

func toModelView(m dbgen.Model) ModelView {
	var body map[string]any
	if len(m.Body) > 0 {
		_ = json.Unmarshal(m.Body, &body)
	}
	return ModelView{
		ID:            m.ID,
		Slug:          m.Slug,
		KindVersionID: m.KindVersionID,
		Body:          body,
		CreatedAt:     m.CreatedAt.Time,
		UpdatedAt:     m.UpdatedAt.Time,
	}
}
