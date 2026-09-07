package oauth

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}

func timestamptzFrom(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
