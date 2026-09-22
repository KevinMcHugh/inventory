package oauth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// token handles POST /oauth/token: the authorization_code grant with PKCE,
// and the refresh_token grant for renewing an access token without
// re-running /oauth/authorize.
func (h *Handler) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "bad form encoding")
		return
	}
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		h.tokenFromCode(w, r)
	case "refresh_token":
		h.tokenFromRefresh(w, r)
	default:
		writeJSONError(w, http.StatusBadRequest, "unsupported_grant_type", "only authorization_code and refresh_token are supported")
	}
}

// tokenFromCode handles the authorization_code grant.
func (h *Handler) tokenFromCode(w http.ResponseWriter, r *http.Request) {
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

	pair, err := h.mintTokenPair(r.Context(), row.ClientID, row.TenantID, derefOr(row.Scope, ""))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "token persist failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(tokenResponse{
		AccessToken:  pair.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(AccessTokenTTL.Seconds()),
		RefreshToken: pair.RefreshToken,
		Scope:        derefOr(row.Scope, ""),
	})
}

// tokenFromRefresh handles the refresh_token grant. Refresh tokens are
// single-use: the row backing the presented token is revoked and a fresh
// access + refresh pair is minted, bound to the same client and tenant.
func (h *Handler) tokenFromRefresh(w http.ResponseWriter, r *http.Request) {
	rawRefresh := r.Form.Get("refresh_token")
	if rawRefresh == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}

	refreshHash := hashToken(rawRefresh)
	row, err := h.Q.GetOAuthTokenByRefreshHash(r.Context(), &refreshHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusBadRequest, "invalid_grant", "refresh token is unknown, expired, or already used")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server_error", "refresh token lookup failed")
		return
	}

	if clientID := r.Form.Get("client_id"); clientID != "" && derefOr(row.ClientID, "") != clientID {
		writeJSONError(w, http.StatusBadRequest, "invalid_grant", "refresh token was issued to a different client")
		return
	}

	if err := h.Q.RevokeOAuthToken(r.Context(), row.ID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "refresh token revoke failed")
		return
	}

	pair, err := h.mintTokenPair(r.Context(), row.ClientID, row.TenantID, derefOr(row.Scope, ""))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "token persist failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(tokenResponse{
		AccessToken:  pair.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(AccessTokenTTL.Seconds()),
		RefreshToken: pair.RefreshToken,
		Scope:        derefOr(row.Scope, ""),
	})
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}
