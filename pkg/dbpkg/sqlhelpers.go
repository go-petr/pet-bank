// Package dbpkg provides helpers to make db initialization and testing easier.
package dbpkg

import (
	"context"
	"database/sql"
)

// Setup sets up connection with database.
func Setup(driver, source string) (*sql.DB, error) {
	db, err := sql.Open(driver, source)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// SQLInterface provides necessary db methods to perform queries.
type SQLInterface interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}
