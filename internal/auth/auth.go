// Package auth carries request-scoped identity through context and provides
// the canonical hashing and key-encoding routines used by both the middleware
// and the bootstrap CLI.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
)

// KeyPrefix is the human-visible label on raw API keys.
const KeyPrefix = "inv_"

// GenerateKey returns a new raw API key. The caller shows it to a human once
// and never persists it — only Hash(rawKey) is stored server-side.
func GenerateKey() (string, error) {
	var b [20]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// 20 bytes → 32 base32 chars, lowercase, no padding.
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	return KeyPrefix + enc, nil
}

// Hash returns the hex-encoded SHA-256 of a raw key; the value stored in
// api_keys.key_hash.
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// -----------------------------------------------------------------------------
// Context
// -----------------------------------------------------------------------------

type ctxKey int

const (
	tenantIDKey ctxKey = iota
	apiKeyIDKey
)

// WithTenant returns a copy of ctx carrying the given tenant and api-key ids.
func WithTenant(ctx context.Context, tenantID, apiKeyID string) context.Context {
	ctx = context.WithValue(ctx, tenantIDKey, tenantID)
	ctx = context.WithValue(ctx, apiKeyIDKey, apiKeyID)
	return ctx
}

// TenantID returns the tenant the current request was authenticated as.
// It returns ErrNoTenant if no tenant is in context — callers reaching this
// path without going through auth middleware is a programming bug.
func TenantID(ctx context.Context) (string, error) {
	v, ok := ctx.Value(tenantIDKey).(string)
	if !ok || v == "" {
		return "", ErrNoTenant
	}
	return v, nil
}

// APIKeyID returns the id of the api key used to authenticate this request,
// or the empty string if none is present.
func APIKeyID(ctx context.Context) string {
	v, _ := ctx.Value(apiKeyIDKey).(string)
	return v
}

// ErrNoTenant is returned when a handler that requires auth is reached without
// a tenant in context.
var ErrNoTenant = errors.New("no tenant in context")
