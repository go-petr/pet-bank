// Package accountrepo manages repository layer of accounts.
package accountrepo

import (
	"context"
	"database/sql"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/dbpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"

	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

// RepoPGS facilitates account repository layer logic.
type RepoPGS struct {
	db dbpkg.SQLInterface
}

// NewRepoPGS returns account RepoPGS.
func NewRepoPGS(db dbpkg.SQLInterface) *RepoPGS {
	return &RepoPGS{
		db: db,
	}
}

const addBalanceQuery = `
UPDATE accounts
SET balance = balance + $1
WHERE id = $2
RETURNING id, owner, balance, currency, created_at
`

// AddBalance changes the account's balance and returns the changed account.
func (r *RepoPGS) AddBalance(ctx context.Context, amount string, id int32) (domain.Account, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, addBalanceQuery, amount, id)

	var a domain.Account

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
			return a, domain.ErrAccountNotFound
		}

		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Constraint == "accounts_balance_check" {
				return a, domain.ErrInsufficientBalance
			}
		}

		return a, errorspkg.ErrInternal
	}

	return a, nil
}

const createQuery = `
INSERT INTO 
    accounts (owner, balance, currency)
VALUES
    ($1, $2, $3)
RETURNING id, owner, balance, currency, created_at
`

// Create creates the account and then returns it.
func (r *RepoPGS) Create(ctx context.Context, owner, balance, currency string) (domain.Account, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, createQuery, owner, balance, currency)

	var a domain.Account

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
				return a, domain.ErrOwnerNotFound
			case "accounts_owner_currency_key":
				return a, domain.ErrCurrencyAlreadyExists
			}
		}

		return a, errorspkg.ErrInternal
	}

	return a, nil
}

const deleteQuery = `
DELETE FROM accounts
WHERE id = $1
`

// Delete removes the account with the given id.
func (r *RepoPGS) Delete(ctx context.Context, id int32) error {
	_, err := r.db.ExecContext(ctx, deleteQuery, id)
	return err
}

const getQuery = `
SELECT 
	id, owner, balance, currency, created_at 
FROM accounts
WHERE id = $1
`

// Get returns the account with the given id.
func (r *RepoPGS) Get(ctx context.Context, id int32) (domain.Account, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, getQuery, id)

	var a domain.Account

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
			return a, domain.ErrAccountNotFound
		}

		return a, errorspkg.ErrInternal
	}

	return a, nil
}

const listAccounts = `
SELECT 
	id, owner, balance, currency, created_at 
FROM accounts
WHERE owner = $1
ORDER BY id
LIMIT $2 OFFSET $3
`

// List returns the specified number of accounts for the given user.
func (r *RepoPGS) List(ctx context.Context, owner string, limit, offset int32) ([]domain.Account, error) {
	l := zerolog.Ctx(ctx)

	rows, err := r.db.QueryContext(ctx, listAccounts, owner, limit, offset)
	if err != nil {
		l.Error().Err(err).Send()
		return nil, errorspkg.ErrInternal
	}
	defer rows.Close()

	items := []domain.Account{}

	for rows.Next() {
		var a domain.Account
		if err := rows.Scan(&a.ID, &a.Owner, &a.Balance, &a.Currency, &a.CreatedAt); err != nil {
			l.Error().Err(err).Send()
			return nil, errorspkg.ErrInternal
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
