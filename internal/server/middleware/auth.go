// Package middleware carries request-scoping middleware for the HTTP server.
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/KevinMcHugh/inventory/internal/auth"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	"github.com/KevinMcHugh/inventory/internal/oauth"
)

// Auth returns middleware that resolves an Authorization: Bearer <token>
// header to a tenant and stores it on the request context.
//
// Two token flavors are accepted:
//
//   - Raw api keys (prefix "inv_"), looked up in api_keys.
//   - OAuth access tokens (prefix "inv_at_"), looked up in oauth_tokens.
//
// The prefix determines which lookup runs; both hash the raw token with
// SHA-256 before hitting the DB, so plaintext never touches storage.
//
// Paths listed in skipAuth and OAuth discovery/authorization endpoints are
// passed through untouched. Discovery is public per RFC 8414 / 9728, and
// /oauth/authorize + /oauth/token do their own credential validation.
//
// On 401, the middleware advertises the resource-metadata URL via a
// WWW-Authenticate header so MCP clients can start the OAuth flow.
func Auth(q dbgen.Querier, issuer string) func(http.Handler) http.Handler {
	skipExact := map[string]bool{
		"/health":                                   true,
		"/.well-known/oauth-protected-resource":     true,
		"/.well-known/oauth-authorization-server":   true,
		"/oauth/register":                           true,
		"/oauth/authorize":                          true,
		"/oauth/token":                              true,
	}
	wwwAuth := `Bearer resource_metadata="` + issuer + `/.well-known/oauth-protected-resource"`

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skipExact[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			token, err := bearerFromHeader(r.Header.Get("Authorization"))
			if err != nil {
				w.Header().Set("WWW-Authenticate", wwwAuth)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			tenantID, apiKeyID, err := resolve(r, q, token)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					w.Header().Set("WWW-Authenticate", wwwAuth)
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, "auth lookup failed", http.StatusInternalServerError)
				return
			}

			ctx := auth.WithTenant(r.Context(), tenantID, apiKeyID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// resolve looks token up in the right table and returns (tenant, apiKey|"").
// api_keys.id is set on raw-key requests; empty for OAuth access tokens.
func resolve(r *http.Request, q dbgen.Querier, token string) (string, string, error) {
	// Access-token prefix is more specific than plain "inv_", so check first.
	if strings.HasPrefix(token, oauth.AccessTokenPrefix) {
		row, err := q.GetOAuthTokenByHash(r.Context(), sha256Hex(token))
		if err != nil {
			return "", "", err
		}
		return row.TenantID, "", nil
	}
	row, err := q.GetAPIKeyByHash(r.Context(), auth.Hash(token))
	if err != nil {
		return "", "", err
	}
	// Best-effort last-used stamp; failures should not break the request.
	_ = q.TouchAPIKey(r.Context(), row.ID)
	return row.TenantID, row.ID, nil
}

// sha256Hex mirrors auth.Hash but is duplicated here so this file does not
// need to know about internal/oauth's private helper set.
func sha256Hex(raw string) string {
	return auth.Hash(raw)
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
