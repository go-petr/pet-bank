package user

import (
	"errors"
	"time"
)

var (
	ErrUsernameAlreadyExists = errors.New("Username already exists")
	ErrEmailALreadyExists    = errors.New("Email already exists")
	ErrUserNotFound          = errors.New("User not found")
	ErrWrongPassword         = errors.New("Wrong password")
	ErrInternal              = errors.New("internal")
)

type User struct {
	Username          string    `json:"username"`
	HashedPassword    string    `json:"hashed_password"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	PasswordChangedAt time.Time `json:"password_changed_at,omitempty"`
	CreatedAt         time.Time `json:"created_at,omitempty"`
}

type CreateUserParams struct {
	Username       string `json:"username"`
	HashedPassword string `json:"hashed_password"`
	FullName       string `json:"full_name"`
	Email          string `json:"email"`
}

type UserWihtoutPassword struct {
	Username          string    `json:"username"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
	CreatedAt         time.Time `json:"created_at"`
}
