package domain

import (
	"errors"
	"time"
)

var (
	// ErrUsernameAlreadyExists indicates the the user with the given username already exists.
	ErrUsernameAlreadyExists = errors.New("username already exists")
	// ErrEmailALreadyExists indicates the the user with the given email already exists.
	ErrEmailALreadyExists = errors.New("email already exists")
	// ErrUserNotFound indicates the the user is not found.
	ErrUserNotFound = errors.New("user not found")
	// ErrWrongPassword indicates the wrong password for the given domain.
	ErrWrongPassword = errors.New("wrong password")
)

// User holds user data.
type User struct {
	Username          string    `json:"username"`
	HashedPassword    string    `json:"hashed_password"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	PasswordChangedAt time.Time `json:"password_changed_at,omitempty"`
	CreatedAt         time.Time `json:"created_at,omitempty"`
}

// CreateUserParams is the input data to create a user.
type CreateUserParams struct {
	Username       string `json:"username"`
	HashedPassword string `json:"hashed_password"`
	FullName       string `json:"full_name"`
	Email          string `json:"email"`
}

// UserWihtoutPassword is User data excluding password data.
type UserWihtoutPassword struct {
	Username  string    `json:"username"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
