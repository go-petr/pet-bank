package repo

import (
	"context"
	"database/sql"

	"github.com/go-petr/pet-bank/internal/account"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/database"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

type AccountRepo struct {
	db database.SQLInterface
}

func NewAccountRepo(db database.SQLInterface) *AccountRepo {
	return &AccountRepo{
		db: db,
	}
}

const addAccountBalance = `
UPDATE accounts
SET balance = balance + $1
WHERE id = $2
RETURNING id, owner, balance, currency, created_at
`

func (r *AccountRepo) AddAccountBalance(ctx context.Context, arg account.AddAccountBalanceParams) (account.Account, error) {

	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, addAccountBalance, arg.Amount, arg.ID)

	var a account.Account

	err := row.Scan(
		&a.ID,
		&a.Owner,
		&a.Balance,
		&a.Currency,
		&a.CreatedAt,
	)

	if err != nil {

		l.Error().Err(err).Send()

		return a, errorspkg.ErrInternal
	}

	return a, nil
}

const createAccount = `
INSERT INTO 
    accounts (owner, balance, currency)
VALUES
    ($1, $2, $3)
RETURNING id, owner, balance, currency, created_at
`

func (r *AccountRepo) CreateAccount(ctx context.Context, arg account.CreateAccountParams) (account.Account, error) {

	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, createAccount, arg.Owner, arg.Balance, arg.Currency)

	var a account.Account

	err := row.Scan(
		&a.ID,
		&a.Owner,
		&a.Balance,
		&a.Currency,
		&a.CreatedAt,
	)

	if err != nil {

		l.Error().Err(err).Send()

		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Constraint {
			case "accounts_owner_fkey":
				return a, account.ErrNoOwnerExists
			case "accounts_owner_currency_idx":
				return a, account.ErrCurrencyAlreadyExists
			}
		}

		return a, errorspkg.ErrInternal
	}

	return a, nil
}

const deleteAccount = `
DELETE FROM accounts
WHERE id = $1
`

func (r *AccountRepo) DeleteAccount(ctx context.Context, id int32) error {
	_, err := r.db.ExecContext(ctx, deleteAccount, id)
	return err
}

const getAccount = `
SELECT id, owner, balance, currency, created_at FROM accounts
WHERE id = $1
`

func (r *AccountRepo) GetAccount(ctx context.Context, id int32) (account.Account, error) {

	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, getAccount, id)

	var a account.Account

	err := row.Scan(
		&a.ID,
		&a.Owner,
		&a.Balance,
		&a.Currency,
		&a.CreatedAt,
	)

	if err != nil {

		l.Error().Err(err).Send()

		if err == sql.ErrNoRows {
			return a, account.ErrAccountNotFound
		}

		return a, errorspkg.ErrInternal
	}

	return a, nil
}

const listAccounts = `
SELECT id, owner, balance, currency, created_at FROM accounts
WHERE owner = $1
ORDER BY id
LIMIT $2 OFFSET $3
`

func (r *AccountRepo) ListAccounts(ctx context.Context, arg account.ListAccountsParams) ([]account.Account, error) {

	l := zerolog.Ctx(ctx)

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
		l.Error().Err(err).Send()
		return nil, errorspkg.ErrInternal
	}

	if err := rows.Err(); err != nil {
		l.Error().Err(err).Send()
		return nil, errorspkg.ErrInternal
	}

	return items, nil
}
