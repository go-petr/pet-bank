package transfer

import (
	"errors"
	"time"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/entry"
)

var (
	ErrCurrencyMismatch    = errors.New("accounts currency mismatch")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrNegativeAmount      = errors.New("negative amount")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidOwner        = errors.New("unauthorized owner")
)

type Transfer struct {
	ID            int64 `json:"id"`
	FromAccountID int32 `json:"from_account_id"`
	ToAccountID   int32 `json:"to_account_id"`
	// must be positive
	Amount    string    `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTransferParams struct {
	FromAccountID int32  `json:"from_account_id"`
	ToAccountID   int32  `json:"to_account_id"`
	Amount        string `json:"amount"`
}

type ListTransfersParams struct {
	FromAccountID int32 `json:"from_account_id"`
	ToAccountID   int32 `json:"to_account_id"`
	Limit         int32 `json:"limit"`
	Offset        int32 `json:"offset"`
}

// TransferTxResult is the result of the transfer transaction
type TransferTxResult struct {
	Transfer    Transfer       `json:"transfer"`
	FromAccount domain.Account `json:"fromAccount"`
	ToAccount   domain.Account `json:"toAccount"`
	FromEntry   entry.Entry    `json:"fromEntry"`
	ToEntry     entry.Entry    `json:"toEntry"`
}
