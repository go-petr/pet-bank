// Package bankapi provides the API to mange users, accounts and money transfers.
package main

import (
	"database/sql"

	"github.com/rs/zerolog/log"

	"github.com/go-petr/pet-bank/cmd/httpserver"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/pkg/configpkg"

	_ "github.com/lib/pq"
)

func main() {
	config, err := configpkg.Load("./configs")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	logger := middleware.CreateLogger(config)

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot open database")
	}

	if err = conn.Ping(); err != nil {
		logger.Fatal().Err(err).Msg("cannot connect to database")
	}

	server, err := httpserver.New(conn, logger, config)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot create server")
	}

	logger.Info().Msg("BANK API SERVER HAS STARTED")

	err = server.Engine.Run(config.ServerAddress)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot start server")
	}
}
