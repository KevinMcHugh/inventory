package idp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// InvitePrefix is the human-visible label on raw invite codes.
const InvitePrefix = "inv_iv_"

// GenerateInvite returns a new raw invite code. Callers show it to a human
// once and never persist it — only HashInvite(rawCode) is stored server-side.
func GenerateInvite() (string, error) {
	var b [20]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding.EncodeToString(b[:])
	return InvitePrefix + enc, nil
}

// HashInvite returns the hex-encoded SHA-256 of a raw invite code — the value
// stored in invites.code_hash.
func HashInvite(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
