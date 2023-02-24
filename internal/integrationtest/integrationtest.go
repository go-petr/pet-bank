// Package integrationtest provides db helpers used in integration tests.
package integrationtest

import (
	"database/sql"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/cmd/httpserver"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/dbpkg"
	"github.com/rs/zerolog"
)

// SetupServer returns test server that cleans up database after each integration test.
func SetupServer(t *testing.T) *httpserver.Server {
	config, err := configpkg.Load("../../configs")
	if err != nil {
		t.Fatalf(`configpkg.Load("../../configs") returned error: %v`, err)
	}

	zerolog.SetGlobalLevel(zerolog.FatalLevel)

	logger := middleware.CreateLogger(config)

	db := SetupDB(t, config.DBDriver, config.DBSource)

	gin.SetMode(gin.ReleaseMode)

	server, err := httpserver.New(db, logger, config)
	if err != nil {
		t.Fatalf(`httpserver.New(db, logger, config) returned error: %v`, err)
	}

	return server
}

// Flush flushes all db tables without droping.
func Flush(t *testing.T, db *sql.DB) {
	t.Helper()

	var tables string

	const query = `
	SELECT string_agg(table_name, ', ')
	FROM information_schema.tables 
	WHERE table_schema='public';`

	row := db.QueryRow(query)

	err := row.Scan(&tables)
	if err != nil {
		t.Fatalf("db cleanup failed. err: %v", err)
	}

	if _, err := db.Exec(`TRUNCATE TABLE ` + tables + " CASCADE"); err != nil {
		t.Fatalf("db cleanup failed. err: %v", err)
	}
}

// SetupDB sets up connection with database for testing and then cleans it.
func SetupDB(t *testing.T, driver, source string) *sql.DB {
	t.Helper()

	db, err := dbpkg.Setup(driver, source)
	if err != nil {
		t.Fatalf("db initialization failed. err: %v", err)
	}

	t.Cleanup(func() {
		Flush(t, db)

		if err := db.Close(); err != nil {
			t.Fatalf("db cleanup failed. err: %v", err)
		}
	})

	return db
}

// SetupTX sets up a database transaction to be used in tests.
//
// Once the tests are done it will rollback the transaction.
func SetupTX(t *testing.T, driver, source string) *sql.Tx {
	t.Helper()

	db, err := dbpkg.Setup(driver, source)
	if err != nil {
		t.Fatalf("db initialization failed. err: %v", err)
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

// SetupSchema sets up a database schema to be used in tests.
//
// It creates a new schema with the t.Name().
// Once the test is complete, it will drop the created schema and close the db connection.
// func SetupSchema(t *testing.T, driver, source string) *sql.DB {
// 	t.Helper()

// 	db, err := Setup(driver, source)
// 	if err != nil {
// 		t.Fatalf("db initialization failed. err: %v", err)
// 	}

// 	schemaName := strings.ToLower(t.Name())

// 	if _, err = db.Exec("CREATE SCHEMA IF NOT EXISTS " + schemaName); err != nil {
// 		t.Fatalf("schema creation failed. err: %v", err)
// 	}

// 	// Run migrations
// 	migrationSrc := "file://../../configs/db/migration"
// 	m, err := migrate.New(migrationSrc, source+"&search_path="+schemaName)

// 	if err != nil {
// 		t.Fatalf(`migrate.New(%v, %v) returned  error: %v`, migrationSrc, source, err)
// 	}

// 	// Migrate all the way up
// 	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
// 		t.Fatalf(`m.Up() returned  error: %v`, err)
// 	}

// 	t.Cleanup(func() {
// 		_, err := db.Exec("DROP SCHEMA " + schemaName + " CASCADE")
// 		if err != nil {
// 			t.Fatalf("db cleanup failed. err: %v", err)
// 		}
// 		_ = db.Close()
// 	})

// 	// use schema
// 	_, err = db.Exec("SET search_path TO " + schemaName)
// 	if err != nil {
// 		t.Fatalf("error while switching to schema. err: %v", err)
// 	}

// 	return db
// }
