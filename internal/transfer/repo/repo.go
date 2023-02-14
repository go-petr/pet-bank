package repo

import (
	"context"
	"database/sql"

	ar "github.com/go-petr/pet-bank/internal/account/repo"
	"github.com/go-petr/pet-bank/internal/domain"
	er "github.com/go-petr/pet-bank/internal/entry/repo"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/rs/zerolog"
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

func (r *transferRepo) CreateTransfer(ctx context.Context, arg domain.CreateTransferParams) (domain.Transfer, error) {

	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, createTransfer, arg.FromAccountID, arg.ToAccountID, arg.Amount)

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
		return t, errorspkg.ErrInternal
	}

	return t, nil
}

const getTransfer = `
SELECT id, from_account_id, to_account_id, amount, created_at FROM transfers
WHERE id = $1 LIMIT 1
`

func (r *transferRepo) GetTransfer(ctx context.Context, id int64) (domain.Transfer, error) {

	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, getTransfer, id)

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
		return t, errorspkg.ErrInternal
	}

	return t, nil
}

const listTransfers = `
SELECT id, from_account_id, to_account_id, amount, created_at FROM transfers
WHERE 
    from_account_id = $1 OR
    to_account_id = $2
ORDER BY id
LIMIT $3 OFFSET $4
`

func (r *transferRepo) ListTransfers(ctx context.Context, arg domain.ListTransfersParams) ([]domain.Transfer, error) {

	l := zerolog.Ctx(ctx)

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

// TransferTx performs a money transfer from one account to the other.
// It creates a transfer record, add account entries, and update accounts' balance within a single dbpkg transaction
func (r *transferRepo) TransferTx(ctx context.Context, arg domain.CreateTransferParams) (domain.TransferTxResult, error) {

	l := zerolog.Ctx(ctx)

	var (
		result domain.TransferTxResult
	)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}
	defer tx.Rollback()

	entryTxRepo := er.NewEntryRepo(tx)
	accountTxRepo := ar.NewAccountRepo(tx)

	result.Transfer, err = r.CreateTransfer(ctx, arg)
	if err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}

	result.FromEntry, err = entryTxRepo.CreateEntry(ctx, "-"+arg.Amount, arg.FromAccountID)
	if err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}

	result.ToEntry, err = entryTxRepo.CreateEntry(ctx, arg.Amount, arg.ToAccountID)
	if err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}

	// To avoid deadlocks execute statements in consistent id order
	if arg.FromAccountID < arg.ToAccountID {
		result.FromAccount, result.ToAccount, err = addBalances(ctx, accountTxRepo, arg.FromAccountID, "-"+arg.Amount, arg.ToAccountID, arg.Amount)
	} else {
		result.ToAccount, result.FromAccount, err = addBalances(ctx, accountTxRepo, arg.ToAccountID, arg.Amount, arg.FromAccountID, "-"+arg.Amount)
	}
	if err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}

	tx.Commit()

	return result, nil
}

func addBalances(
	ctx context.Context, r *ar.AccountRepo,
	account1ID int32, amount1 string,
	account2ID int32, amount2 string) (domain.Account, domain.Account, error) {

	account1, err := r.AddAccountBalance(ctx, amount1, account1ID)
	if err != nil {
		return domain.Account{}, domain.Account{}, err
	}

	account2, err := r.AddAccountBalance(ctx, amount2, account2ID)
	if err != nil {
		return domain.Account{}, domain.Account{}, err
	}

	return account1, account2, nil
}
