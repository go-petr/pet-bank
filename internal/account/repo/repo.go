package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-petr/pet-bank/internal/account"
	"github.com/lib/pq"
)

type accountRepo struct {
	db *sql.DB
}

func NewAccountRepo(db *sql.DB) *accountRepo {
	return &accountRepo{
		db: db,
	}
}

const addAccountBalance = `
UPDATE accounts
SET balance = balance + $1
WHERE id = $2
RETURNING id, owner, balance, currency, created_at
`

func (r *accountRepo) AddAccountBalance(ctx context.Context, arg account.AddAccountBalanceParams) (account.Account, error) {

	row := r.db.QueryRowContext(ctx, addAccountBalance, arg.Amount, arg.ID)

	var a account.Account

	err := row.Scan(
		&a.ID,
		&a.Owner,
		&a.Balance,
		&a.Currency,
		&a.CreatedAt,
	)
	return a, err
}

const createAccount = `
INSERT INTO 
    accounts (owner, balance, currency)
VALUES
    ($1, $2, $3)
RETURNING id, owner, balance, currency, created_at
`

func (r *accountRepo) CreateAccount(ctx context.Context, arg account.CreateAccountParams) (account.Account, error) {

	row := r.db.QueryRowContext(ctx, createAccount, arg.Owner, arg.Balance, arg.Currency)

	var a account.Account

	err := row.Scan(
		&a.ID,
		&a.Owner,
		&a.Balance,
		&a.Currency,
		&a.CreatedAt,
	)

	if pqErr, ok := err.(*pq.Error); ok {
		fmt.Println(err)
		switch pqErr.Constraint {
		case "accounts_owner_fkey":
			return a, account.ErrNoOwnerExists
		case "accounts_owner_currency_idx":
			return a, account.ErrCurrencyAlreadyExists
		}
	}

	return a, err
}

const deleteAccount = `
DELETE FROM accounts
WHERE id = $1
`

func (r *accountRepo) DeleteAccount(ctx context.Context, id int32) error {
	_, err := r.db.ExecContext(ctx, deleteAccount, id)
	return err
}

const getAccount = `
SELECT id, owner, balance, currency, created_at FROM accounts
WHERE id = $1
`

func (r *accountRepo) GetAccount(ctx context.Context, id int32) (account.Account, error) {

	row := r.db.QueryRowContext(ctx, getAccount, id)

	var a account.Account

	err := row.Scan(
		&a.ID,
		&a.Owner,
		&a.Balance,
		&a.Currency,
		&a.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return a, account.ErrAccountNotFound
	}
	return a, err
}

const listAccounts = `
SELECT id, owner, balance, currency, created_at FROM accounts
WHERE owner = $1
ORDER BY id
LIMIT $2 OFFSET $3
`

func (r *accountRepo) ListAccounts(ctx context.Context, arg account.ListAccountsParams) ([]account.Account, error) {

	rows, err := r.db.QueryContext(ctx, listAccounts, arg.Owner, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	items := []account.Account{}

	for rows.Next() {
		var a account.Account
		if err := rows.Scan(
			&a.ID,
			&a.Owner,
			&a.Balance,
			&a.Currency,
			&a.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, a)
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
