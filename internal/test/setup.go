package test

// import (
// 	"testing"

// 	"github.com/gin-gonic/gin"
// 	"github.com/go-petr/pet-bank/cmd/httpserver"
// 	"github.com/go-petr/pet-bank/internal/middleware"
// 	"github.com/go-petr/pet-bank/pkg/configpkg"
// 	"github.com/go-petr/pet-bank/pkg/dbpkg/integrationtest"
// 	"github.com/rs/zerolog"
// )

// // SetupServer returns test server that cleans up database after each integration test.
// func SetupServer(t *testing.T) *httpserver.Server {
// 	config, err := configpkg.Load("../../../configs")
// 	if err != nil {
// 		t.Fatalf(`configpkg.Load("../../../configs") returned error: %v`, err)
// 	}

// 	zerolog.SetGlobalLevel(zerolog.FatalLevel)
// 	logger := middleware.CreateLogger(config)

// 	db := integrationtest.SetupDB(t, config.DBDriver, config.DBSource)

// 	gin.SetMode(gin.ReleaseMode)
// 	server, err := httpserver.New(db, logger, config)
// 	if err != nil {
// 		t.Fatalf(`httpserver.New(db, logger, config) returned error: %v`, err)
// 	}

// 	return server
// }
