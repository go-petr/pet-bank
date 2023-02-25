// Package userrepo manages repository layer of users.
package userrepo

import (
	"context"
	"database/sql"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/dbpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

// RepoPGS facilitates user repository layer logic.
type RepoPGS struct {
	db dbpkg.SQLInterface
}

// NewRepoPGS returns account RepoPGS.
func NewRepoPGS(db dbpkg.SQLInterface) *RepoPGS {
	return &RepoPGS{
		db: db,
	}
}

// CreateQuery inserts into users table.
const CreateQuery = `
INSERT INTO users (
    username,
    hashed_password,
    full_name,
    email
) VALUES (
    $1, $2, $3, $4
) RETURNING username, hashed_password, full_name, email, password_changed_at, created_at
`

// Create creates the user and then returns it.
func (r *RepoPGS) Create(ctx context.Context, arg domain.CreateUserParams) (domain.User, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, CreateQuery,
		arg.Username,
		arg.HashedPassword,
		arg.FullName,
		arg.Email,
	)

	var u domain.User

	err := row.Scan(
		&u.Username,
		&u.HashedPassword,
		&u.FullName,
		&u.Email,
		&u.PasswordChangedAt,
		&u.CreatedAt,
	)

	if err != nil {
		l.Error().Err(err).Send()

		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "unique_violation" {
				switch pqErr.Constraint {
				case "users_pkey":
					return u, domain.ErrUsernameAlreadyExists
				case "users_email_key":
					return u, domain.ErrEmailALreadyExists
				}
			}
		}

		return u, err
	}

	return u, nil
}

const getQuery = `
SELECT 
	username, 
	hashed_password, 
	full_name, 
	email, 
	password_changed_at, 
	created_at 
FROM users
WHERE username = $1
`

// Get returns the user with the given username.
func (r *RepoPGS) Get(ctx context.Context, username string) (domain.User, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, getQuery, username)

	var u domain.User

	err := row.Scan(
		&u.Username,
		&u.HashedPassword,
		&u.FullName,
		&u.Email,
		&u.PasswordChangedAt,
		&u.CreatedAt,
	)

	if err != nil {
		l.Error().Err(err).Send()

		if err == sql.ErrNoRows {
			return u, domain.ErrUserNotFound
		}

		return u, errorspkg.ErrInternal
	}

	return u, nil
}
