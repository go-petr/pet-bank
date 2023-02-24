//go:build integration

package tests

import (
	"log"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/cmd/httpserver"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/dbpkg"
	"github.com/rs/zerolog"
)

var server *httpserver.Server

// TestMain calls testMain and passes the returned exit code to os.Exit(). The reason
// that TestMain is basically a wrapper around testMain is because os.Exit() does not
// respect deferred functions, so this configuration allows for a deferred function.
func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

// testMain returns an integer denoting an exit code to be returned and used in
// TestMain. The exit code 0 denotes success, all other codes denote failure.
func testMain(m *testing.M) int {
	config, err := configpkg.Load("../../../configs")
	if err != nil {
		log.Println("cannot load config:", err)
		return 1
	}

	zerolog.SetGlobalLevel(zerolog.FatalLevel)
	logger := middleware.CreateLogger(config)

	conn, err := dbpkg.Setup(config.DBDriver, config.DBSource)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot setup database")
	}

	gin.SetMode(gin.ReleaseMode)
	server, err = httpserver.New(conn, logger, config)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot start server")
	}

	return m.Run()
}
