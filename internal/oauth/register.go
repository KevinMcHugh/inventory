package oauth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/xid"

	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

// register handles POST /oauth/register — dynamic client registration
// (RFC 7591). The client sends its metadata; we mint a client_id + optional
// client_secret and echo back a full registration response.
//
// Public clients (no client_secret) are the common case for MCP.
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_client_metadata", "malformed JSON body")
		return
	}
	if len(req.RedirectURIs) == 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_client_metadata", "redirect_uris is required")
		return
	}
	for _, u := range req.RedirectURIs {
		if u == "" {
			writeJSONError(w, http.StatusBadRequest, "invalid_redirect_uri", "empty redirect_uri")
			return
		}
	}

	clientID := xid.New().String()

	// This tiny AS treats every registration as a public client — no secret.
	// If a client explicitly asks for token_endpoint_auth_method other than
	// "none", we still return "none" and let it use PKCE.

	redirectJSON, err := json.Marshal(req.RedirectURIs)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "marshal redirect_uris")
		return
	}

	if _, err := h.Q.CreateOAuthClient(r.Context(), dbgen.CreateOAuthClientParams{
		ID:               clientID,
		ClientSecretHash: nil,
		RedirectUris:     redirectJSON,
		ClientName:       nullable(req.ClientName),
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "persist client")
		return
	}

	resp := registrationResponse{
		ClientID:                clientID,
		ClientName:              req.ClientName,
		RedirectURIs:            req.RedirectURIs,
		TokenEndpointAuthMethod: "none",
		GrantTypes:              []string{"authorization_code"},
		ResponseTypes:           []string{"code"},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

type registrationRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	Scope                   string   `json:"scope,omitempty"`
}

type registrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
}

// loadClientRedirectURIs returns the client's registered redirect_uris.
func (h *Handler) loadClientRedirectURIs(r *http.Request, clientID string) ([]string, error) {
	client, err := h.Q.GetOAuthClient(r.Context(), clientID)
	if err != nil {
		return nil, err
	}
	var out []string
	if err := json.Unmarshal(client.RedirectUris, &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, errors.New("client has no registered redirect_uris")
	}
	return out, nil
}
