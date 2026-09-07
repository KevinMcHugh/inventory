// Package oauth implements the minimum MCP OAuth 2.1 surface required for
// Claude (and other MCP clients) to authenticate against the inventory MCP
// endpoint.
//
// The flow: a client dynamically registers (POST /oauth/register), sends its
// user through /oauth/authorize, gets a code back on its redirect_uri, and
// exchanges the code + PKCE verifier at /oauth/token for a bearer access
// token. Middleware accepts that access token in addition to the raw
// inv_... api keys.
//
// Approving an authorize request means pasting an existing inv_... api key
// on the /oauth/authorize form. The tenant scope of the resulting access
// token is the tenant that owns the pasted key.
package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

const (
	// AccessTokenTTL is how long an issued access token is valid.
	AccessTokenTTL = 24 * time.Hour
	// AuthCodeTTL is how long an authorization code is valid before the
	// client must exchange it.
	AuthCodeTTL = 10 * time.Minute

	AccessTokenPrefix = "inv_at_"
	AuthCodePrefix    = "inv_ac_"
	ClientSecretPrefix = "inv_cs_"
)

// Handler bundles the state the OAuth endpoints need.
type Handler struct {
	// Issuer is the canonical base URL of this server, e.g.
	// "https://inventory-b2mxg.sprites.app". Used in discovery docs and
	// resource identifiers.
	Issuer string
	Q      dbgen.Querier
}

// Mount attaches every OAuth endpoint onto r.
func (h *Handler) Mount(r chi.Router) {
	r.Get("/.well-known/oauth-protected-resource", h.protectedResourceMetadata)
	r.Get("/.well-known/oauth-authorization-server", h.authorizationServerMetadata)
	r.Post("/oauth/register", h.register)
	r.Get("/oauth/authorize", h.authorizeGET)
	r.Post("/oauth/authorize", h.authorizePOST)
	r.Post("/oauth/token", h.token)
}

// -----------------------------------------------------------------------------
// Random tokens and hashing
// -----------------------------------------------------------------------------

// randToken returns a raw token with the given prefix and 32 random bytes
// encoded as URL-safe base64.
func randToken(prefix string) (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// hashToken is the canonical SHA-256 hex used everywhere in this package.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// -----------------------------------------------------------------------------
// PKCE
// -----------------------------------------------------------------------------

// verifyPKCE returns true if hash(codeVerifier, method) == codeChallenge.
// Only S256 is supported; anything else fails.
func verifyPKCE(codeVerifier, codeChallenge, method string) bool {
	if method != "S256" {
		return false
	}
	sum := sha256.Sum256([]byte(codeVerifier))
	got := base64.RawURLEncoding.EncodeToString(sum[:])
	return got == codeChallenge
}

// -----------------------------------------------------------------------------
// Small HTTP helpers
// -----------------------------------------------------------------------------

// writeJSONError writes an OAuth-style error response as defined by RFC 6749.
func writeJSONError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `","error_description":"` + description + `"}`))
}

// stringsIncludes reports whether needle is in haystack.
func stringsIncludes(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// trimBearer strips a leading "Bearer " (case-insensitive) from an
// Authorization header value.
func trimBearer(h string) string {
	if len(h) < 7 {
		return ""
	}
	if !strings.EqualFold(h[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}
