package oauth

import (
	"context"
	"time"

	"github.com/rs/xid"

	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

// MintAccessToken issues a fresh inv_at_ token for the given tenant. The
// clientID argument may be empty ("" or the sentinel "web") when the token
// is minted for a browser session that did not go through a downstream
// OAuth client. The returned string is the raw token; only its hash is
// persisted.
func (h *Handler) MintAccessToken(
	ctx context.Context,
	tenantID string,
	clientID string,
	ttl time.Duration,
) (string, error) {
	if ttl <= 0 {
		ttl = AccessTokenTTL
	}
	raw, err := randToken(AccessTokenPrefix)
	if err != nil {
		return "", err
	}
	var client *string
	if clientID != "" && clientID != "web" {
		c := clientID
		client = &c
	}
	if _, err := h.Q.CreateOAuthToken(ctx, dbgen.CreateOAuthTokenParams{
		ID:        xid.New().String(),
		TokenHash: hashToken(raw),
		ClientID:  client,
		TenantID:  tenantID,
		ExpiresAt: timestamptzFrom(time.Now().Add(ttl)),
	}); err != nil {
		return "", err
	}
	return raw, nil
}

// MintAuthzCodeParams is the input for MintAuthzCode.
type MintAuthzCodeParams struct {
	ClientID            string
	TenantID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	TTL                 time.Duration
}

// MintAuthzCode issues an inv_ac_ authorization code bound to the given
// client + tenant + PKCE challenge. Used by the SSO handler to complete a
// pending downstream authorize request without going through the paste-a-key
// form.
func (h *Handler) MintAuthzCode(
	ctx context.Context,
	p MintAuthzCodeParams,
) (string, error) {
	if p.TTL <= 0 {
		p.TTL = AuthCodeTTL
	}
	raw, err := randToken(AuthCodePrefix)
	if err != nil {
		return "", err
	}
	if _, err := h.Q.CreateOAuthCode(ctx, dbgen.CreateOAuthCodeParams{
		CodeHash:            hashToken(raw),
		ClientID:            &p.ClientID,
		TenantID:            p.TenantID,
		RedirectUri:         p.RedirectURI,
		CodeChallenge:       p.CodeChallenge,
		CodeChallengeMethod: p.CodeChallengeMethod,
		Scope:               nullable(p.Scope),
		ExpiresAt:           timestamptzFrom(time.Now().Add(p.TTL)),
	}); err != nil {
		return "", err
	}
	return raw, nil
}
