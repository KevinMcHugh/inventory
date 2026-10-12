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

// mintedTokenPair is the raw access + refresh token strings returned by the
// token endpoint. Only their hashes are persisted.
type mintedTokenPair struct {
	AccessToken  string
	RefreshToken string
}

// mintTokenPair issues a fresh access token bound to a fresh refresh token,
// both tied to the same oauth_tokens row so refreshing can revoke the old
// row and rotate to a new pair. Used by the /oauth/token authorization_code
// and refresh_token grants.
func (h *Handler) mintTokenPair(ctx context.Context, clientID *string, tenantID, scope string) (mintedTokenPair, error) {
	rawAccess, err := randToken(AccessTokenPrefix)
	if err != nil {
		return mintedTokenPair{}, err
	}
	rawRefresh, err := randToken(RefreshTokenPrefix)
	if err != nil {
		return mintedTokenPair{}, err
	}
	refreshHash := hashToken(rawRefresh)
	if _, err := h.Q.CreateOAuthToken(ctx, dbgen.CreateOAuthTokenParams{
		ID:               xid.New().String(),
		TokenHash:        hashToken(rawAccess),
		ClientID:         clientID,
		TenantID:         tenantID,
		Scope:            nullable(scope),
		ExpiresAt:        timestamptzFrom(time.Now().Add(AccessTokenTTL)),
		RefreshTokenHash: &refreshHash,
		RefreshExpiresAt: timestamptzFrom(time.Now().Add(RefreshTokenTTL)),
	}); err != nil {
		return mintedTokenPair{}, err
	}
	return mintedTokenPair{AccessToken: rawAccess, RefreshToken: rawRefresh}, nil
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
