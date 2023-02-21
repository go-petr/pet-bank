// Package dbpkg provides helpers to make db initialization and testing easier.
package dbpkg

import (
	"context"
	"database/sql"
	"testing"
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

// SetupTX sets up a database transaction to be used in tests.
// Once the tests are done it will rollback the transaction.
func SetupTX(t *testing.T, driver, source string) *sql.Tx {
	t.Helper()

	db, err := sql.Open(driver, source)
	if err != nil {
		t.Fatalf("Database open connection failed: %v", err)
	}

	if err = db.Ping(); err != nil {
		t.Fatalf("db.Ping() failed: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("db.Begin() failed: %v", err)
	}

	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil {
			t.Fatalf("tx.Rollback() failed: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() failed: %v", err)
		}
	})

	return tx
}

// SQLInterface provides neccessary db methods to perform queries.
type SQLInterface interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}
