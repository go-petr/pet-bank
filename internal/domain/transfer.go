package domain

import (
	"errors"
	"time"
)

var (
	// ErrCurrencyMismatch indicates that transfer accounts have different currencies.
	ErrCurrencyMismatch = errors.New("accounts currency mismatch")
	// ErrInvalidAmount indicates invalid amount.
	ErrInvalidAmount = errors.New("invalid amount")
	// ErrNegativeAmount indicates negative amount.
	ErrNegativeAmount = errors.New("negative amount")
	// ErrInsufficientBalance indicates that the account does not have sufficient balance.
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrInvalidOwner indicates that the user is unauthorized to transfer money from the account.
	ErrInvalidOwner = errors.New("unauthorized owner")
)

// Transfer holds transfer data between two accounts.
type Transfer struct {
	ID            int64     `json:"id"`
	FromAccountID int32     `json:"from_account_id"`
	ToAccountID   int32     `json:"to_account_id"`
	Amount        string    `json:"amount"` // must be positive
	CreatedAt     time.Time `json:"created_at"`
}

// CreateTransferParams is the input data for the transfer transaction.
type CreateTransferParams struct {
	FromAccountID int32  `json:"from_account_id"`
	ToAccountID   int32  `json:"to_account_id"`
	Amount        string `json:"amount"`
}

// ListTransfersParams is the input data to get transfers between two accounts.
type ListTransfersParams struct {
	FromAccountID int32 `json:"from_account_id"`
	ToAccountID   int32 `json:"to_account_id"`
	Limit         int32 `json:"limit"`
	Offset        int32 `json:"offset"`
}

// TransferTxResult is the result of the transfer transaction.
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"fromAccount"`
	ToAccount   Account  `json:"toAccount"`
	FromEntry   Entry    `json:"fromEntry"`
	ToEntry     Entry    `json:"toEntry"`
}
