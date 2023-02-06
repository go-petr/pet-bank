package session

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrBlockedSession         = errors.New("blocked session")
	ErrMismatchedRefreshToken = errors.New("mismatched session token")
	ErrInvalidUser            = errors.New("incorrect session user")
	ErrExpiredSession         = errors.New("expired session")
	ErrSessionNotFound        = errors.New("Session not found")
)

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

type CreateSessionParams struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	RefreshToken string    `json:"refresh_token"`
	UserAgent    string    `json:"user_agent"`
	ClientIP     string    `json:"client_ip"`
	IsBlocked    bool      `json:"is_blocked"`
	ExpiresAt    time.Time `json:"expires_at"`
}
