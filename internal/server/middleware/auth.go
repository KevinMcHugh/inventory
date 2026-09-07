// Package middleware carries request-scoping middleware for the HTTP server.
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/KevinMcHugh/inventory/internal/auth"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

// Auth returns middleware that resolves an Authorization: Bearer <key> header
// to a tenant and stores it on the request context.
//
// Unauthenticated (skipAuth) prefixes are passed through untouched — currently
// just /health and /mcp discovery. Everything else 401s if the token is
// missing or unknown.
//
// On matched routes with a {tenantId} URL parameter, the value is compared to
// the authenticated tenant and a mismatch returns 403.
func Auth(q dbgen.Querier) func(http.Handler) http.Handler {
	skipAuth := []string{"/health"}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range skipAuth {
				if r.URL.Path == p {
					next.ServeHTTP(w, r)
					return
				}
			}

			token, err := bearerFromHeader(r.Header.Get("Authorization"))
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			row, err := q.GetAPIKeyByHash(r.Context(), auth.Hash(token))
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, "auth lookup failed", http.StatusInternalServerError)
				return
			}

			if urlTenant := chi.URLParam(r, "tenantId"); urlTenant != "" && urlTenant != row.TenantID {
				http.Error(w, "tenant mismatch", http.StatusForbidden)
				return
			}

			// Best-effort last-used stamp; failures should not break the request.
			_ = q.TouchAPIKey(r.Context(), row.ID)

			ctx := auth.WithTenant(r.Context(), row.TenantID, row.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerFromHeader(h string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return "", errors.New("missing bearer")
	}
	token := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if token == "" {
		return "", errors.New("empty bearer")
	}
	return token, nil
}
