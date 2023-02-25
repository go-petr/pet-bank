package domain

import (
	"errors"
	"time"
)

var (
	// ErrEntryNotFound indicates that the entry is not found.
	ErrEntryNotFound = errors.New("entry not found")
)

// Entry holds balance change data for an account.
type Entry struct {
	ID        int64     `json:"id"`
	AccountID int32     `json:"account_id"`
	Amount    string    // can be negative or positive `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}
