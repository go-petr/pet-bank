package repo

import (
	"context"
	"database/sql"

	"github.com/go-petr/pet-bank/internal/account"
	ar "github.com/go-petr/pet-bank/internal/account/repo"
	"github.com/go-petr/pet-bank/internal/entry"
	er "github.com/go-petr/pet-bank/internal/entry/repo"
	"github.com/go-petr/pet-bank/internal/transfer"
)

type transferRepo struct {
	db *sql.DB
}

func NewTransferRepo(db *sql.DB) *transferRepo {
	return &transferRepo{
		db: db,
	}
}

const createTransfer = `
INSERT INTO
    transfers (from_account_id, to_account_id, amount)
VALUES
    ($1, $2, $3)
RETURNING id, from_account_id, to_account_id, amount, created_at
`

func (r *transferRepo) CreateTransfer(ctx context.Context, arg transfer.CreateTransferParams) (transfer.Transfer, error) {

	row := r.db.QueryRowContext(ctx, createTransfer, arg.FromAccountID, arg.ToAccountID, arg.Amount)

	var t transfer.Transfer
	err := row.Scan(
		&t.ID,
		&t.FromAccountID,
		&t.ToAccountID,
		&t.Amount,
		&t.CreatedAt,
	)
	return t, err
}

const getTransfer = `
SELECT id, from_account_id, to_account_id, amount, created_at FROM transfers
WHERE id = $1 LIMIT 1
`

func (r *transferRepo) GetTransfer(ctx context.Context, id int64) (transfer.Transfer, error) {

	row := r.db.QueryRowContext(ctx, getTransfer, id)

	var t transfer.Transfer

	err := row.Scan(
		&t.ID,
		&t.FromAccountID,
		&t.ToAccountID,
		&t.Amount,
		&t.CreatedAt,
	)
	return t, err
}

const listTransfers = `
SELECT id, from_account_id, to_account_id, amount, created_at FROM transfers
WHERE 
    from_account_id = $1 OR
    to_account_id = $2
ORDER BY id
LIMIT $3 OFFSET $4
`

func (r *transferRepo) ListTransfers(ctx context.Context, arg transfer.ListTransfersParams) ([]transfer.Transfer, error) {

	rows, err := r.db.QueryContext(ctx, listTransfers,
		arg.FromAccountID,
		arg.ToAccountID,
		arg.Limit,
		arg.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []transfer.Transfer{}

	for rows.Next() {
		var t transfer.Transfer
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
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// TransferTx performs a money transfer from one account to the other.
// It creates a transfer record, add account entries, and update accounts' balance within a single database transaction
func (r *transferRepo) TransferTx(ctx context.Context, arg transfer.CreateTransferParams) (transfer.TransferTxResult, error) {

	var (
		result transfer.TransferTxResult
		empty  transfer.TransferTxResult
	)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()

	entryTxRepo := er.NewEntryRepo(tx)
	accountTxRepo := ar.NewAccountRepo(tx)

	result.Transfer, err = r.CreateTransfer(ctx, arg)
	if err != nil {
		return empty, err
	}

	result.FromEntry, err = entryTxRepo.CreateEntry(ctx, entry.CreateEntryParams{
		AccountID: arg.FromAccountID,
		Amount:    "-" + arg.Amount,
	})
	if err != nil {
		return empty, err
	}

	result.ToEntry, err = entryTxRepo.CreateEntry(ctx, entry.CreateEntryParams{
		AccountID: arg.ToAccountID,
		Amount:    arg.Amount,
	})
	if err != nil {
		return empty, err
	}

	// To avoid deadlocks execute statements in consistent id order
	if arg.FromAccountID < arg.ToAccountID {
		result.FromAccount, result.ToAccount, err = addBalances(ctx, accountTxRepo, arg.FromAccountID, "-"+arg.Amount, arg.ToAccountID, arg.Amount)
	} else {
		result.ToAccount, result.FromAccount, err = addBalances(ctx, accountTxRepo, arg.ToAccountID, arg.Amount, arg.FromAccountID, "-"+arg.Amount)
	}
	if err != nil {
		return empty, err
	}

	tx.Commit()

	return result, nil
}

func addBalances(
	ctx context.Context, r *ar.AccountRepo,
	account1ID int32, amount1 string,
	account2ID int32, amount2 string) (account.Account, account.Account, error) {

	account1, err := r.AddAccountBalance(ctx, account.AddAccountBalanceParams{
		ID:     account1ID,
		Amount: amount1,
	})
	if err != nil {
		return account.Account{}, account.Account{}, err
	}

	account2, err := r.AddAccountBalance(ctx, account.AddAccountBalanceParams{
		ID:     account2ID,
		Amount: amount2,
	})
	if err != nil {
		return account.Account{}, account.Account{}, err
	}

	return account1, account2, nil
}
