// Package mcp exposes the inventory datastore over the Model Context Protocol
// so Claude (or any MCP client) can list and manipulate models.
//
// Tools declare typed input and output structs; jsonschema is inferred from
// struct tags. Each tool builds the same Endpoint the equivalent HTTP handler
// uses (from internal/server/<resource>) and drives it with
// server.BuildViewModel, which runs Interact (the Store-backed domain logic,
// including tenant scoping via auth.TenantID) and Build (the pure M→VM
// transform) — the same MVVM flow as the REST API, stopping short of Render
// since MCP output shapes aren't apigen response objects. Tools wrap the
// resulting view-model in their own Output struct instead.
package mcp

import (
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
