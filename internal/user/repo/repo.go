package repo

import (
	"context"
	"database/sql"

	"github.com/go-petr/pet-bank/internal/user"
	"github.com/lib/pq"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

const createUser = `
INSERT INTO users (
    username,
    hashed_password,
    full_name,
    email
) VALUES (
    $1, $2, $3, $4
) RETURNING username, hashed_password, full_name, email, password_changed_at, created_at
`

func (r *UserRepo) CreateUser(ctx context.Context, arg user.CreateUserParams) (user.User, error) {

	row := r.db.QueryRowContext(ctx, createUser,
		arg.Username,
		arg.HashedPassword,
		arg.FullName,
		arg.Email,
	)

	var u user.User

	err := row.Scan(
		&u.Username,
		&u.HashedPassword,
		&u.FullName,
		&u.Email,
		&u.PasswordChangedAt,
		&u.CreatedAt,
	)

	if pqErr, ok := err.(*pq.Error); ok {
		if pqErr.Code.Name() == "unique_violation" {
			switch pqErr.Constraint {
			case "users_pkey":
				return u, user.ErrUsernameAlreadyExists
			case "users_email_key":
				return u, user.ErrEmailALreadyExists
			}
		}
	}

	return u, err
}

const getUser = `
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

func (r *UserRepo) GetUser(ctx context.Context, username string) (user.User, error) {

	row := r.db.QueryRowContext(ctx, getUser, username)

	var u user.User

	err := row.Scan(
		&u.Username,
		&u.HashedPassword,
		&u.FullName,
		&u.Email,
		&u.PasswordChangedAt,
		&u.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return u, user.ErrUserNotFound
	}
	
	return u, err
}