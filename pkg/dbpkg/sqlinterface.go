// Package dbpkg provides dbpkg support functionality.
package dbpkg

import (
	"context"
	"database/sql"
)

// SQLInterface provides neccessary db methods to perform transactions and queries.
type SQLInterface interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}
