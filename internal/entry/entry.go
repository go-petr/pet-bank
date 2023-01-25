package entry

import "time"

type Entry struct {
	ID        int64
	AccountID int32
	// can be negative or positive
	Amount    string
	CreatedAt time.Time
}

type CreateEntryParams struct {
	AccountID int32  `json:"account_id"`
	Amount    string `json:"amount"`
}

type ListEntriesParams struct {
	AccountID int32 `json:"account_id"`
	Limit     int32 `json:"limit"`
	Offset    int32 `json:"offset"`
}
