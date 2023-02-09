package repo

import (
	"context"
	"database/sql"

	"github.com/go-petr/pet-bank/internal/session"
	"github.com/go-petr/pet-bank/internal/user"
	"github.com/go-petr/pet-bank/pkg/apperrors"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{
		db: db,
	}
}

const createSession = `
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

func (r *SessionRepo) CreateSession(ctx context.Context, arg session.CreateSessionParams) (session.Session, error) {

	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, createSession,
		arg.ID,
		arg.Username,
		arg.RefreshToken,
		arg.UserAgent,
		arg.ClientIP,
		arg.IsBlocked,
		arg.ExpiresAt,
	)

	var s session.Session

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
			switch pqErr.Constraint {
			case "sessions_username_fkey":
				return s, user.ErrUserNotFound
			}
		}

		return s, apperrors.ErrInternal
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

func (r *SessionRepo) GetSession(ctx context.Context, id uuid.UUID) (session.Session, error) {

	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, getSession, id)

	var s session.Session

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
