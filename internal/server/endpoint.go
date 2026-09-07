package server

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Endpoint is the MVVM contract for a single HTTP operation.
//
//   - Req is the parsed input (usually a *RequestObject from oapi-codegen).
//   - M  is the domain model returned by a data-store operation.
//   - VM is the view-model — a pure transformation of M for presentation.
//   - Res is the typed response value (usually a *ResponseObject from oapi-codegen).
//
// Concrete endpoints embed their own narrow store interface so each handler
// declares exactly the persistence surface it needs.
type Endpoint[Req, M, VM, Res any] interface {
	Interact(ctx context.Context, req Req) (M, error)
	Build(m M) VM
	Render(vm VM) Res
}

// Run drives an Endpoint end-to-end. Handlers call this from the strict server
// wrapper to keep call sites uniform.
func Run[Req, M, VM, Res any](ctx context.Context, e Endpoint[Req, M, VM, Res], req Req) (Res, error) {
	m, err := e.Interact(ctx, req)
	if err != nil {
		var zero Res
		return zero, err
	}
	return e.Render(e.Build(m)), nil
}

// ErrNotFound signals that a requested resource does not exist. Handlers map
// this (and pgx.ErrNoRows) to a 404 response.
var ErrNotFound = errors.New("not found")

// IsNotFound reports whether err represents a missing row.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, pgx.ErrNoRows)
}
