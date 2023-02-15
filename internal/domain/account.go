// Package domain provides defenitions of all entities.
package domain

import (
	"errors"
	"time"
)

var (
	// ErrAccountNotFound indicates that the account is not found.
	ErrAccountNotFound = errors.New("account not found")
	// ErrCurrencyAlreadyExists indicates that the account with the given currency already exists.
	ErrCurrencyAlreadyExists = errors.New("account currency already exists")
	// ErrOwnerNotFound indicates that the owner for the account is not found.
	ErrOwnerNotFound = errors.New("owner not found")
)

// Account holds user balance data for specific currency.
type Account struct {
	ID        int32     `json:"id"`
	Owner     string    `json:"owner"`
	Balance   string    `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}
