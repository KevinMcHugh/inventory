package oauth

import (
	"encoding/json"
	"net/http"
)

// protectedResourceMetadata is served at /.well-known/oauth-protected-resource
// per RFC 9728 (draft). It advertises which authorization server(s) guard
// this resource.
func (h *Handler) protectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	doc := map[string]any{
		"resource":                 h.Issuer + "/mcp",
		"authorization_servers":    []string{h.Issuer},
		"bearer_methods_supported": []string{"header"},
		"scopes_supported":         []string{"mcp"},
	}
	writeJSON(w, doc)
}

// authorizationServerMetadata is served at /.well-known/oauth-authorization-server
// per RFC 8414. It advertises the endpoints and capabilities of this AS.
func (h *Handler) authorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	doc := map[string]any{
		"issuer":                                h.Issuer,
		"authorization_endpoint":                h.Issuer + "/oauth/authorize",
		"token_endpoint":                        h.Issuer + "/oauth/token",
		"registration_endpoint":                 h.Issuer + "/oauth/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_post"},
		"scopes_supported":                      []string{"mcp"},
	}
	writeJSON(w, doc)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
