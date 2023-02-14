package domain

import "time"

// Entry holds balance change data for an account.
type Entry struct {
	ID        int64
	AccountID int32
	Amount    string // can be negative or positive
	CreatedAt time.Time
}
