package account

import (
	"errors"
	"time"
)

var (
	ErrAccountNotFound       = errors.New("Account not found")
	ErrCurrencyAlreadyExists = errors.New("Account currency already exists")
	ErrNoOwnerExists         = errors.New("Owner does not exists")
	ErrOwnerAlreadyExists    = errors.New("Owner already exists")
	ErrInternal              = errors.New("internal")
)

type Account struct {
	ID        int32     `json:"id"`
	Owner     string    `json:"owner"`
	Balance   string    `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

type AddAccountBalanceParams struct {
	Amount string
	ID     int32
}

type CreateAccountParams struct {
	Owner    string
	Balance  string
	Currency string
}

type ListAccountsParams struct {
	Owner  string
	Limit  int32
	Offset int32
}

type UpdateAccountParams struct {
	ID      int32
	Balance string
}
