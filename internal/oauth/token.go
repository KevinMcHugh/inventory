package oauth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/xid"

	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

// token handles POST /oauth/token, the authorization_code grant with PKCE.
func (h *Handler) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "bad form encoding")
		return
	}
	if got := r.Form.Get("grant_type"); got != "authorization_code" {
		writeJSONError(w, http.StatusBadRequest, "unsupported_grant_type", "only authorization_code is supported")
		return
	}
	code := r.Form.Get("code")
	verifier := r.Form.Get("code_verifier")
	clientID := r.Form.Get("client_id")
	redirectURI := r.Form.Get("redirect_uri")
	if code == "" || verifier == "" || clientID == "" || redirectURI == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "code, code_verifier, client_id, redirect_uri all required")
		return
	}

	row, err := h.Q.ConsumeOAuthCode(r.Context(), hashToken(code))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusBadRequest, "invalid_grant", "code is unknown, expired, or already used")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server_error", "code lookup failed")
		return
	}

	if derefOr(row.ClientID, "") != clientID {
		writeJSONError(w, http.StatusBadRequest, "invalid_grant", "code was issued for a different client")
		return
	}
	if row.RedirectUri != redirectURI {
		writeJSONError(w, http.StatusBadRequest, "invalid_grant", "redirect_uri does not match the one used at /authorize")
		return
	}
	if !verifyPKCE(verifier, row.CodeChallenge, row.CodeChallengeMethod) {
		writeJSONError(w, http.StatusBadRequest, "invalid_grant", "PKCE verifier does not match code_challenge")
		return
	}

	rawToken, err := randToken(AccessTokenPrefix)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "token generation failed")
		return
	}
	expires := time.Now().Add(AccessTokenTTL)
	if _, err := h.Q.CreateOAuthToken(r.Context(), dbgen.CreateOAuthTokenParams{
		ID:         xid.New().String(),
		TokenHash:  hashToken(rawToken),
		ClientID:   row.ClientID,
		TenantID:   row.TenantID,
		Scope:      row.Scope,
		ExpiresAt:  timestamptzFrom(expires),
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "token persist failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(tokenResponse{
		AccessToken: rawToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(AccessTokenTTL.Seconds()),
		Scope:       derefOr(row.Scope, ""),
	})
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
}
