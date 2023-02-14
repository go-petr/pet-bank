package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrBlockedSession indicates that the session is blocked.
	ErrBlockedSession = errors.New("blocked session")
	// ErrMismatchedRefreshToken indicates mismatch between the given token and the session token.
	ErrMismatchedRefreshToken = errors.New("mismatched session token")
	// ErrInvalidUser indicates that the session is not related to the given domain.
	ErrInvalidUser = errors.New("incorrect session user")
	// ErrExpiredSession indicates that the expired session.
	ErrExpiredSession = errors.New("expired session")
	// ErrSessionNotFound indicates that the session is not found.
	ErrSessionNotFound = errors.New("Session not found")
)

// Session holds session data for particular domain.
type Session struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	RefreshToken string    `json:"refresh_token"`
	UserAgent    string    `json:"user_agent"`
	ClientIP     string    `json:"client_ip"`
	IsBlocked    bool      `json:"is_blocked"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateSessionParams holds data nedeed for Session creation.
type CreateSessionParams struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	RefreshToken string    `json:"refresh_token"`
	UserAgent    string    `json:"user_agent"`
	ClientIP     string    `json:"client_ip"`
	IsBlocked    bool      `json:"is_blocked"`
	ExpiresAt    time.Time `json:"expires_at"`
}
