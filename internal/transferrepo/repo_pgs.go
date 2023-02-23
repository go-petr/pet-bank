// Package transferrepo manages repository layer of transfers.
package transferrepo

import (
	"context"
	"database/sql"

	"github.com/go-petr/pet-bank/internal/accountrepo"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/entryrepo"
	"github.com/go-petr/pet-bank/pkg/dbpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

// RepoPGS facilitates transfer repository layer logic.
type RepoPGS struct {
	db   dbpkg.SQLInterface
	conn *sql.DB
}

// NewTxRepoPGS returns account RepoPGS.
func NewTxRepoPGS(db dbpkg.SQLInterface) *RepoPGS {
	return &RepoPGS{
		db: db,
	}
}

// NewRepoPGS returns account RepoPGS wiht connection to start transactions.
func NewRepoPGS(db *sql.DB) *RepoPGS {
	return &RepoPGS{
		db:   db,
		conn: db,
	}
}

const createQuery = `
INSERT INTO
    transfers (from_account_id, to_account_id, amount)
VALUES
    ($1, $2, $3)
RETURNING id, from_account_id, to_account_id, amount, created_at
`

// Create creates the transfer and then returns it.
func (r *RepoPGS) Create(ctx context.Context, arg domain.CreateTransferParams) (domain.Transfer, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, createQuery, arg.FromAccountID, arg.ToAccountID, arg.Amount)

	var t domain.Transfer
	err := row.Scan(
		&t.ID,
		&t.FromAccountID,
		&t.ToAccountID,
		&t.Amount,
		&t.CreatedAt,
	)

	if err != nil {
		l.Error().Err(err).Msgf("Create(ctx context.Context, %+v)", arg)

		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Constraint {
			case "transfers_from_account_id_fkey":
				return t, domain.ErrAccountNotFound
			case "transfers_to_account_id_fkey":
				return t, domain.ErrAccountNotFound
			case "transfers_amount_check":
				return t, domain.ErrInvalidAmount
			}
		}

		return t, errorspkg.ErrInternal
	}

	return t, nil
}

const getQuery = `
SELECT 
	id, from_account_id, to_account_id, amount, created_at 
FROM transfers
WHERE id = $1
`

// Get returns the transfer with the given id.
func (r *RepoPGS) Get(ctx context.Context, id int64) (domain.Transfer, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, getQuery, id)

	var t domain.Transfer

	err := row.Scan(
		&t.ID,
		&t.FromAccountID,
		&t.ToAccountID,
		&t.Amount,
		&t.CreatedAt,
	)

	if err != nil {
		l.Error().Err(err).Send()

		if err == sql.ErrNoRows {
			return t, domain.ErrTransferNotFound
		}

		return t, errorspkg.ErrInternal
	}

	return t, nil
}

const listTransfers = `
SELECT 
	id, from_account_id, to_account_id, amount, created_at 
FROM transfers
WHERE 
    from_account_id = $1 OR to_account_id = $2
ORDER BY id
LIMIT $3 OFFSET $4
`

// List returns the transfers betweem the specified accounts.
func (r *RepoPGS) List(ctx context.Context, arg domain.ListTransfersParams) ([]domain.Transfer, error) {
	l := zerolog.Ctx(ctx)

	rows, err := r.db.QueryContext(ctx, listTransfers,
		arg.FromAccountID,
		arg.ToAccountID,
		arg.Limit,
		arg.Offset,
	)
	if err != nil {
		l.Error().Err(err).Send()
		return nil, errorspkg.ErrInternal
	}
	defer rows.Close()

	items := []domain.Transfer{}

	for rows.Next() {
		var t domain.Transfer
		if err := rows.Scan(
			&t.ID,
			&t.FromAccountID,
			&t.ToAccountID,
			&t.Amount,
			&t.CreatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, t)
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

// Transfer performs a money between two accounts.
//
// It creates a transfer record, add account entries, and update accounts' balance
// within a single dbpkg transaction.
func (r *RepoPGS) Transfer(ctx context.Context, arg domain.CreateTransferParams) (domain.TransferTxResult, error) {
	l := zerolog.Ctx(ctx)

	var result domain.TransferTxResult

	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			l.Error().Err(err).Send()
		}
	}()

	entryRepo := entryrepo.NewRepoPGS(tx)
	accountRepo := accountrepo.NewRepoPGS(tx)

	result.Transfer, err = r.Create(ctx, arg)
	if err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}

	result.FromEntry, err = entryRepo.Create(ctx, "-"+arg.Amount, arg.FromAccountID)
	if err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}

	result.ToEntry, err = entryRepo.Create(ctx, arg.Amount, arg.ToAccountID)
	if err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}

	var fromAccount, toAccount domain.Account
	// To avoid deadlocks execute statements in consistent id order
	if arg.FromAccountID < arg.ToAccountID {
		argAddBalance := addBalanceParams{
			account1ID: arg.FromAccountID,
			amount1:    "-" + arg.Amount,
			account2ID: arg.ToAccountID,
			amount2:    arg.Amount,
		}

		fromAccount, toAccount, err = addBalances(ctx, accountRepo, argAddBalance)
	} else {
		argAddBalance := addBalanceParams{
			account1ID: arg.ToAccountID,
			amount1:    arg.Amount,
			account2ID: arg.FromAccountID,
			amount2:    "-" + arg.Amount,
		}

		toAccount, fromAccount, err = addBalances(ctx, accountRepo, argAddBalance)
	}

	if err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}

	result.FromAccount, result.ToAccount = fromAccount, toAccount

	if err := tx.Commit(); err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}

	return result, nil
}

type addBalanceParams struct {
	account1ID int32
	amount1    string
	account2ID int32
	amount2    string
}

func addBalances(ctx context.Context, r *accountrepo.RepoPGS, arg addBalanceParams) (domain.Account, domain.Account, error) {
	account1, err := r.AddBalance(ctx, arg.amount1, arg.account1ID)
	if err != nil {
		return domain.Account{}, domain.Account{}, err
	}

	account2, err := r.AddBalance(ctx, arg.amount2, arg.account2ID)
	if err != nil {
		return domain.Account{}, domain.Account{}, err
	}

	return account1, account2, nil
}
