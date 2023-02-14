package repo

import (
	"context"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/dbpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/rs/zerolog"
)

type EntryRepo struct {
	db dbpkg.SQLInterface
}

func NewEntryRepo(db dbpkg.SQLInterface) *EntryRepo {
	return &EntryRepo{db: db}
}

const createEntry = `
INSERT INTO
    entries (account_id, amount)
VALUES
    ($1, $2)
RETURNING id, account_id, amount, created_at
`

func (r *EntryRepo) CreateEntry(ctx context.Context, amount string, account int32) (domain.Entry, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, createEntry, account, amount)

	var e domain.Entry

	err := row.Scan(
		&e.ID,
		&e.AccountID,
		&e.Amount,
		&e.CreatedAt,
	)

	if err != nil {
		l.Error().Err(err).Send()
		return e, errorspkg.ErrInternal
	}

	return e, nil
}

const getEntry = `
SELECT id, account_id, amount, created_at FROM entries
WHERE id = $1 LIMIT 1
`

func (r *EntryRepo) GetEntry(ctx context.Context, id int64) (domain.Entry, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, getEntry, id)

	var e domain.Entry

	err := row.Scan(
		&e.ID,
		&e.AccountID,
		&e.Amount,
		&e.CreatedAt,
	)

	if err != nil {
		l.Error().Err(err).Send()
		return e, errorspkg.ErrInternal
	}

	return e, nil
}

const listEntries = `
SELECT id, account_id, amount, created_at FROM entries
WHERE account_id = $1
LIMIT $2 OFFSET $3
`

func (r *EntryRepo) ListEntries(ctx context.Context, accountID int32, limit, offset int32) ([]domain.Entry, error) {
	l := zerolog.Ctx(ctx)

	rows, err := r.db.QueryContext(ctx, listEntries, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Entry{}

	for rows.Next() {
		var e domain.Entry
		if err := rows.Scan(
			&e.ID,
			&e.AccountID,
			&e.Amount,
			&e.CreatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, e)
	}

	if err := rows.Close(); err != nil {
		l.Error().Err(err).Send()
		return items, errorspkg.ErrInternal
	}

	if err := rows.Err(); err != nil {
		l.Error().Err(err).Send()
		return items, errorspkg.ErrInternal
	}

	return items, nil
}