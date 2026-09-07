// Package mcp exposes the inventory datastore over the Model Context Protocol
// so Claude (or any MCP client) can list and manipulate models.
//
// Tools declare typed input and output structs; jsonschema is inferred from
// struct tags. Each tool captures the dbgen.Querier in a closure — the same
// store the HTTP endpoints use, so both interfaces see identical data.
//
// Output structs embed the view-model types from the endpoint packages
// (tenants.ViewModel, kinds.ViewModel, etc.). Those types own the M→VM
// conversion; MCP tools call the exported ToViewModel helpers rather than
// maintaining a parallel view layer.
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
