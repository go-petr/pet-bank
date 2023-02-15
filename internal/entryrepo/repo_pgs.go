// Package entryrepo manages repository layer of entries.
package entryrepo

import (
	"context"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/dbpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/rs/zerolog"
)

// RepoPGS facilitates entry repository layer logic.
type RepoPGS struct {
	db dbpkg.SQLInterface
}

// NewRepoPGS returns account RepoPGS.
func NewRepoPGS(db dbpkg.SQLInterface) *RepoPGS {
	return &RepoPGS{db: db}
}

const createQuery = `
INSERT INTO
    entries (account_id, amount)
VALUES
    ($1, $2)
RETURNING id, account_id, amount, created_at
`

// Create creates the entry and then returns it.
func (r *RepoPGS) Create(ctx context.Context, amount string, account int32) (domain.Entry, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, createQuery, account, amount)

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

const getQuery = `
SELECT id, account_id, amount, created_at FROM entries
WHERE id = $1 LIMIT 1
`

// Get returns the entry with the given id.
func (r *RepoPGS) Get(ctx context.Context, id int64) (domain.Entry, error) {
	l := zerolog.Ctx(ctx)

	row := r.db.QueryRowContext(ctx, getQuery, id)

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

const listQuery = `
SELECT id, account_id, amount, created_at FROM entries
WHERE account_id = $1
LIMIT $2 OFFSET $3
`

// List returns the specified number of entries for the given accountID.
func (r *RepoPGS) List(ctx context.Context, accountID int32, limit, offset int32) ([]domain.Entry, error) {
	l := zerolog.Ctx(ctx)

	rows, err := r.db.QueryContext(ctx, listQuery, accountID, limit, offset)
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
		return nil, errorspkg.ErrInternal
	}

	if err := rows.Err(); err != nil {
		l.Error().Err(err).Send()
		return nil, errorspkg.ErrInternal
	}

	return items, nil
}
