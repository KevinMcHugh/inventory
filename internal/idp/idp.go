// Package idp integrates external identity providers (Google today; others
// slot in beside it) into the inventory. On the way in, a user starts an SSO
// flow at /auth/<provider>. We hand them off to the provider, receive their
// email + stable subject on the callback, look them up in user_identities,
// mint an inv_at_ access token bound to their tenant, and either:
//
//  - complete a pending downstream OAuth 2.1 authorize request (Claude case)
//    by redirecting to the client's redirect_uri with our authz code, or
//  - redirect the browser to a return_to path with the raw access token in
//    the URL fragment (web app case).
//
// Access model: first-login-creates-tenant. If we have never seen this
// (provider, subject) before we create a fresh tenant on the spot and link
// the identity to it.
package idp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/xid"

	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

// IntentTTL bounds how long an SSO login intent survives between the start
// and callback halves of the flow. Ten minutes leaves headroom for a slow
// consent screen.
const IntentTTL = 10 * time.Minute

// Config carries the wiring one Handler needs. Issuer is the canonical base
// URL of this server, used for building the OAuth redirect URIs we register
// with the IDP.
type Config struct {
	Issuer            string
	GoogleClientID    string
	GoogleClientSecret string
}

// Handler is the http-facing surface. It reads/writes the DB through Q and
// mints access tokens via TokenMinter to keep this package independent of
// internal/oauth's helpers.
type Handler struct {
	Cfg         Config
	Q           dbgen.Querier
	TokenMinter TokenMinter
}

// TokenMinter issues an inv_at_ access token and returns the raw value. The
// oauth package supplies the concrete implementation; keeping it as an
// interface here avoids an import cycle.
type TokenMinter interface {
	MintAccessToken(ctx context.Context, tenantID, clientID string, ttl time.Duration) (string, error)
	MintAuthzCode(ctx context.Context, params AuthzCodeParams) (string, error)
}

// AuthzCodeParams captures everything MintAuthzCode needs to reproduce the
// pending downstream OAuth authorize request that started the SSO flow.
type AuthzCodeParams struct {
	ClientID            string
	TenantID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	TTL                 time.Duration
}

// -----------------------------------------------------------------------------
// Provider identity resolution (create-or-find + tenant)
// -----------------------------------------------------------------------------

// UserInfo is the subset of provider profile fields we need. subject is the
// provider's stable id, email/name are for display.
type UserInfo struct {
	Provider    string
	Subject     string
	Email       string
	DisplayName string
}

// ResolveTenant looks up (or, first login, creates + links) the tenant that
// owns this identity. Returns the tenant id ready to be embedded in an
// access token. Idempotent: repeat calls for the same (provider, subject)
// resolve to the same tenant.
//
// For unknown identities, a valid invite code MUST be provided. The invite
// either names a tenant to link into or leaves it unspecified, in which
// case a fresh tenant is created for the new identity.
func (h *Handler) ResolveTenant(ctx context.Context, info UserInfo, inviteCode string) (string, error) {
	// Fast path: known identity.
	row, err := h.Q.GetUserIdentityByProviderSubject(ctx, dbgen.GetUserIdentityByProviderSubjectParams{
		Provider: info.Provider,
		Subject:  info.Subject,
	})
	if err == nil {
		_ = h.Q.TouchUserIdentity(ctx, dbgen.TouchUserIdentityParams{
			ID:          row.ID,
			Email:       info.Email,
			DisplayName: nullableString(info.DisplayName),
		})
		return row.TenantID, nil
	}

	// Unknown identity: require + claim an invite.
	if inviteCode == "" {
		return "", ErrInviteRequired
	}
	invite, err := h.Q.ClaimInviteByHash(ctx, HashInvite(inviteCode))
	if err != nil {
		return "", ErrInviteInvalid
	}

	// Determine target tenant: prefer invite.tenant_id, else create fresh.
	tenantID := ""
	if invite.TenantID != nil {
		tenantID = *invite.TenantID
	} else {
		tenantName := info.DisplayName
		if tenantName == "" {
			tenantName = info.Email
		}
		tenant, err := h.Q.CreateTenant(ctx, dbgen.CreateTenantParams{
			ID:   xid.New().String(),
			Name: tenantName,
		})
		if err != nil {
			return "", err
		}
		tenantID = tenant.ID
	}

	identity, err := h.Q.CreateUserIdentity(ctx, dbgen.CreateUserIdentityParams{
		ID:          xid.New().String(),
		TenantID:    tenantID,
		Provider:    info.Provider,
		Subject:     info.Subject,
		Email:       info.Email,
		DisplayName: nullableString(info.DisplayName),
	})
	if err != nil {
		return "", err
	}
	// Fill in the audit trail on the invite row.
	usedBy := identity.ID
	_ = h.Q.SetInviteUsedBy(ctx, dbgen.SetInviteUsedByParams{
		ID:                invite.ID,
		UsedByIdentityID: &usedBy,
	})
	return tenantID, nil
}

// ErrInviteRequired is returned when a Google login lands with an unknown
// (provider, subject) and no invite was carried through the flow.
var ErrInviteRequired = errors.New("sso: invite required for first login")

// ErrInviteInvalid is returned when the presented invite code is unknown,
// already used, expired, or revoked.
var ErrInviteInvalid = errors.New("sso: invite is invalid or already used")

// -----------------------------------------------------------------------------
// Intent state store (shared across providers)
// -----------------------------------------------------------------------------

// PendingAuthzRequest carries the downstream OAuth authorize request we
// interrupted with an SSO handoff. If ClientID is empty, this SSO flow was
// started directly by the web app.
type PendingAuthzRequest struct {
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	DownstreamState     string
	Scope               string
}

// StartIntent inserts a fresh intent row and returns the state token to
// send to the IDP. inviteCode may be empty; it is only consulted when the
// callback lands with an unknown identity.
func (h *Handler) StartIntent(
	ctx context.Context,
	provider string,
	returnTo string,
	pending *PendingAuthzRequest,
	inviteCode string,
) (string, error) {
	state, err := randomState()
	if err != nil {
		return "", err
	}
	arg := dbgen.CreateSSOLoginIntentParams{
		State:      state,
		Provider:   provider,
		ExpiresAt:  pgTs(time.Now().Add(IntentTTL)),
		InviteCode: nullableString(inviteCode),
	}
	if returnTo != "" {
		arg.ReturnTo = &returnTo
	}
	if pending != nil {
		arg.ClientID = &pending.ClientID
		arg.RedirectUri = &pending.RedirectURI
		arg.CodeChallenge = &pending.CodeChallenge
		arg.CodeChallengeMethod = &pending.CodeChallengeMethod
		arg.DownstreamState = nullableString(pending.DownstreamState)
		arg.Scope = nullableString(pending.Scope)
	}
	if _, err := h.Q.CreateSSOLoginIntent(ctx, arg); err != nil {
		return "", err
	}
	return state, nil
}

// ConsumeIntent looks up (and removes) the intent that started this flow.
func (h *Handler) ConsumeIntent(ctx context.Context, state string) (dbgen.SsoLoginIntent, error) {
	return h.Q.ConsumeSSOLoginIntent(ctx, state)
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

func randomState() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// buildRedirect appends query params to a base URL. Used both for provider
// authorize URLs and for the final client redirect after a successful login.
func buildRedirect(base string, params url.Values) string {
	sep := "?"
	if u, err := url.Parse(base); err == nil && u.RawQuery != "" {
		sep = "&"
	}
	return base + sep + params.Encode()
}

// pendingFromIntent recovers the downstream authorize request (or reports
// there was not one).
func pendingFromIntent(row dbgen.SsoLoginIntent) (PendingAuthzRequest, bool) {
	if row.ClientID == nil || row.RedirectUri == nil {
		return PendingAuthzRequest{}, false
	}
	return PendingAuthzRequest{
		ClientID:            *row.ClientID,
		RedirectURI:         *row.RedirectUri,
		CodeChallenge:       derefOr(row.CodeChallenge, ""),
		CodeChallengeMethod: derefOr(row.CodeChallengeMethod, ""),
		DownstreamState:     derefOr(row.DownstreamState, ""),
		Scope:               derefOr(row.Scope, ""),
	}, true
}

func derefOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}

// finishWebLogin mints a fresh access token bound to tenantID and returns
// the raw string for embedding in a URL fragment.
func (h *Handler) finishWebLogin(ctx context.Context, tenantID string) (string, error) {
	return h.TokenMinter.MintAccessToken(ctx, tenantID, "web", 24*time.Hour)
}

// finishAuthzCode reissues a pending OAuth 2.1 authorize request as if the
// user had approved: mints our authz code and returns the redirect URL for
// the downstream client.
func (h *Handler) finishAuthzCode(ctx context.Context, tenantID string, pending PendingAuthzRequest) (string, error) {
	rawCode, err := h.TokenMinter.MintAuthzCode(ctx, AuthzCodeParams{
		ClientID:            pending.ClientID,
		TenantID:            tenantID,
		RedirectURI:         pending.RedirectURI,
		CodeChallenge:       pending.CodeChallenge,
		CodeChallengeMethod: pending.CodeChallengeMethod,
		Scope:               pending.Scope,
		TTL:                 10 * time.Minute,
	})
	if err != nil {
		return "", err
	}
	q := url.Values{"code": {rawCode}}
	if pending.DownstreamState != "" {
		q.Set("state", pending.DownstreamState)
	}
	return buildRedirect(pending.RedirectURI, q), nil
}

// -----------------------------------------------------------------------------
// sanity errors
// -----------------------------------------------------------------------------

var errIntentExpired = errors.New("sso: login intent expired or not found")

func pgTs(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
