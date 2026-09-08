package idp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	googleProvider     = "google"
	googleAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL     = "https://oauth2.googleapis.com/token"
	googleUserInfoURL  = "https://www.googleapis.com/oauth2/v3/userinfo"
	googleScope        = "openid email profile"
)

// Mount attaches Google OAuth routes to r. Only registered when a
// google-client-id and secret are configured; otherwise these routes stay
// off and the /oauth/authorize form renders without a Google button.
func (h *Handler) Mount(r chi.Router) {
	if h.Cfg.GoogleClientID == "" || h.Cfg.GoogleClientSecret == "" {
		return
	}
	r.Get("/auth/google", h.googleStart)
	r.Get("/auth/google/callback", h.googleCallback)
}

// GoogleEnabled reports whether the handler was configured with Google
// credentials — used by the /oauth/authorize page to decide whether to show
// the button.
func (h *Handler) GoogleEnabled() bool {
	return h.Cfg.GoogleClientID != "" && h.Cfg.GoogleClientSecret != ""
}

// googleStart initiates a Google OAuth flow.
//
// Two entry points feed this endpoint:
//
//   - The web app "Sign in with Google" button: query has `return_to=<path>`.
//   - The /oauth/authorize page: query carries the pending downstream
//     OAuth request (client_id, redirect_uri, code_challenge, ...).
//
// Both cases persist their state into sso_login_intents keyed by the random
// state we send to Google.
func (h *Handler) googleStart(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var pending *PendingAuthzRequest
	if q.Get("client_id") != "" {
		pending = &PendingAuthzRequest{
			ClientID:            q.Get("client_id"),
			RedirectURI:         q.Get("redirect_uri"),
			CodeChallenge:       q.Get("code_challenge"),
			CodeChallengeMethod: q.Get("code_challenge_method"),
			DownstreamState:     q.Get("state"),
			Scope:               q.Get("scope"),
		}
	}
	returnTo := q.Get("return_to")
	invite := q.Get("invite")

	state, err := h.StartIntent(r.Context(), googleProvider, returnTo, pending, invite)
	if err != nil {
		http.Error(w, "could not start login: "+err.Error(), http.StatusInternalServerError)
		return
	}

	params := url.Values{
		"client_id":     {h.Cfg.GoogleClientID},
		"redirect_uri":  {h.Cfg.Issuer + "/auth/google/callback"},
		"response_type": {"code"},
		"scope":         {googleScope},
		"state":         {state},
		"access_type":   {"online"},
		"prompt":        {"select_account"},
	}
	http.Redirect(w, r, googleAuthorizeURL+"?"+params.Encode(), http.StatusFound)
}

// googleCallback handles the return leg of the flow: exchanges the code,
// resolves the identity to a tenant, and finishes either as a downstream
// authorize response or as a browser redirect carrying our access token.
func (h *Handler) googleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errParam := q.Get("error"); errParam != "" {
		http.Error(w, "google denied the login: "+errParam, http.StatusForbidden)
		return
	}
	state := q.Get("state")
	code := q.Get("code")
	if state == "" || code == "" {
		http.Error(w, "callback missing state or code", http.StatusBadRequest)
		return
	}

	intent, err := h.ConsumeIntent(r.Context(), state)
	if err != nil {
		http.Error(w, errIntentExpired.Error(), http.StatusBadRequest)
		return
	}

	// Exchange the code for tokens.
	tokenBody, err := postForm(r.Context(), googleTokenURL, url.Values{
		"code":          {code},
		"client_id":     {h.Cfg.GoogleClientID},
		"client_secret": {h.Cfg.GoogleClientSecret},
		"redirect_uri":  {h.Cfg.Issuer + "/auth/google/callback"},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		http.Error(w, "token exchange failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	var tokResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.Unmarshal(tokenBody, &tokResp); err != nil || tokResp.AccessToken == "" {
		http.Error(w, "malformed token response from google", http.StatusBadGateway)
		return
	}

	// Fetch the profile.
	info, err := fetchGoogleUserInfo(r.Context(), tokResp.AccessToken)
	if err != nil {
		http.Error(w, "userinfo fetch failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	inviteCode := ""
	if intent.InviteCode != nil {
		inviteCode = *intent.InviteCode
	}
	tenantID, err := h.ResolveTenant(r.Context(), info, inviteCode)
	if err != nil {
		if errors.Is(err, ErrInviteRequired) || errors.Is(err, ErrInviteInvalid) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, "identity resolution failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Finish depending on whether a downstream OAuth request is riding along.
	if pending, ok := pendingFromIntent(intent); ok {
		redirect, err := h.finishAuthzCode(r.Context(), tenantID, pending)
		if err != nil {
			http.Error(w, "could not finalize authorize: "+err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}

	// Web app case: send the raw access token back in the URL fragment so
	// the SPA JS can pull it into localStorage. The fragment is never sent
	// to any server, which is the intended property for an implicit-style
	// callback.
	raw, err := h.finishWebLogin(r.Context(), tenantID)
	if err != nil {
		http.Error(w, "could not mint token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	returnTo := "/"
	if intent.ReturnTo != nil && strings.HasPrefix(*intent.ReturnTo, "/") {
		returnTo = *intent.ReturnTo
	}
	http.Redirect(w, r, returnTo+"#access_token="+raw, http.StatusFound)
}

// -----------------------------------------------------------------------------
// low-level http helpers
// -----------------------------------------------------------------------------

func postForm(ctx context.Context, endpoint string, values url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("upstream " + endpoint + " returned " + resp.Status + ": " + string(body))
	}
	return body, nil
}

func fetchGoogleUserInfo(ctx context.Context, accessToken string) (UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return UserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return UserInfo{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UserInfo{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return UserInfo{}, fmt.Errorf("userinfo returned %s: %s", resp.Status, body)
	}
	var raw struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return UserInfo{}, err
	}
	if raw.Sub == "" || raw.Email == "" {
		return UserInfo{}, errors.New("userinfo missing sub or email")
	}
	return UserInfo{
		Provider:    googleProvider,
		Subject:     raw.Sub,
		Email:       raw.Email,
		DisplayName: raw.Name,
	}, nil
}
