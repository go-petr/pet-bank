package repo

import (
	"context"

	"github.com/go-petr/pet-bank/internal/entry"
	"github.com/go-petr/pet-bank/pkg/apperrors"
	"github.com/go-petr/pet-bank/pkg/database"
	"github.com/rs/zerolog"
)

type EntryRepo struct {
	db database.SQLInterface
}

func NewEntryRepo(db database.SQLInterface) *EntryRepo {
	return &EntryRepo{db: db}
}

const createEntry = `
INSERT INTO
    entries (account_id, amount)
VALUES
    ($1, $2)
RETURNING id, account_id, amount, created_at
`

func (r *EntryRepo) CreateEntry(ctx context.Context, arg entry.CreateEntryParams) (entry.Entry, error) {

	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, createEntry, arg.AccountID, arg.Amount)

	var e entry.Entry

	err := row.Scan(
		&e.ID,
		&e.AccountID,
		&e.Amount,
		&e.CreatedAt,
	)

	if err != nil {
		l.Error().Err(err).Send()
		return e, apperrors.ErrInternal
	}

	return e, nil
}

const getEntry = `
SELECT id, account_id, amount, created_at FROM entries
WHERE id = $1 LIMIT 1
`

func (r *EntryRepo) GetEntry(ctx context.Context, id int64) (entry.Entry, error) {

	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, getEntry, id)

	var e entry.Entry

	err := row.Scan(
		&e.ID,
		&e.AccountID,
		&e.Amount,
		&e.CreatedAt,
	)

	if err != nil {
		l.Error().Err(err).Send()
		return e, apperrors.ErrInternal
	}

	return e, nil
}

const listEntries = `
SELECT id, account_id, amount, created_at FROM entries
WHERE account_id = $1
LIMIT $2 OFFSET $3
`

func (r *EntryRepo) ListEntries(ctx context.Context, arg entry.ListEntriesParams) ([]entry.Entry, error) {

	l := zerolog.Ctx(ctx)

	rows, err := r.db.QueryContext(ctx, listEntries, arg.AccountID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []entry.Entry{}

	for rows.Next() {
		var e entry.Entry
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
		return items, apperrors.ErrInternal
	}
	if err := rows.Err(); err != nil {
		l.Error().Err(err).Send()
		return items, apperrors.ErrInternal
	}
	return items, nil
}
