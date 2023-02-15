// Package sessionrepo manages repository layer of entries.
package sessionrepo

import (
	"context"
	"database/sql"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

// RepoPGS facilitates session repository layer logic.
type RepoPGS struct {
	db *sql.DB
}

// NewRepoPGS returns account RepoPGS.
func NewRepoPGS(db *sql.DB) *RepoPGS {
	return &RepoPGS{
		db: db,
	}
}

const createQuery = `
INSERT INTO sessions (
	id,
	username,
	refresh_token,
	user_agent,
	client_ip,
	is_blocked,
	expires_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7
	) RETURNING id, username, refresh_token, user_agent, client_ip, is_blocked, expires_at, created_at;
`

// Create creates the session and then returns it.
func (r *RepoPGS) Create(ctx context.Context, arg domain.CreateSessionParams) (domain.Session, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, createQuery,
		arg.ID,
		arg.Username,
		arg.RefreshToken,
		arg.UserAgent,
		arg.ClientIP,
		arg.IsBlocked,
		arg.ExpiresAt,
	)

	var s domain.Session

	err := row.Scan(
		&s.ID,
		&s.Username,
		&s.RefreshToken,
		&s.UserAgent,
		&s.ClientIP,
		&s.IsBlocked,
		&s.ExpiresAt,
		&s.CreatedAt,
	)

	if err != nil {
		l.Error().Err(err).Send()

		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Constraint == "sessions_username_fkey" {
				return s, domain.ErrUserNotFound
			}
		}

		return s, errorspkg.ErrInternal
	}

	return s, nil
}

const getSession = `
SELECT 
	id,
	username,
	refresh_token,
	user_agent,
	client_ip,
	is_blocked,
	expires_at,
	created_at
FROM sessions
WHERE id = $1
`

// Get returns session with the given id.
func (r *RepoPGS) Get(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, getSession, id)

	var s domain.Session

	err := row.Scan(
		&s.ID,
		&s.Username,
		&s.RefreshToken,
		&s.UserAgent,
		&s.ClientIP,
		&s.IsBlocked,
		&s.ExpiresAt,
		&s.CreatedAt,
	)

	if err != nil {
		l.Error().Err(err).Send()
	}

	return s, nil
}
