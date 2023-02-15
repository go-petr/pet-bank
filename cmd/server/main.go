package main

import (
	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	_ "github.com/lib/pq"

	"github.com/go-petr/pet-bank/internal/accountdelivery"
	"github.com/go-petr/pet-bank/internal/accountrepo"
	"github.com/go-petr/pet-bank/internal/accountservice"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/internal/sessiondelivery"
	"github.com/go-petr/pet-bank/internal/sessionrepo"
	"github.com/go-petr/pet-bank/internal/sessionservice"
	"github.com/go-petr/pet-bank/internal/transferdelivery"
	"github.com/go-petr/pet-bank/internal/transferrepo"
	"github.com/go-petr/pet-bank/internal/transferservice"
	"github.com/go-petr/pet-bank/internal/userdelivery"
	"github.com/go-petr/pet-bank/internal/userrepo"
	"github.com/go-petr/pet-bank/internal/userservice"
)

func main() {
	config, err := configpkg.Load("./configs")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	logger := middleware.CreateLogger(config)

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot connect to db")
	}

	server, err := createServer(conn, logger, config)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot create server")
	}

	err = server.Run(config.ServerAddress)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot start server")
	}
}

func createServer(conn *sql.DB, logger zerolog.Logger, config configpkg.Config) (*gin.Engine, error) {
	userRepo := userrepo.NewRepoPGS(conn)
	accountRepo := accountrepo.NewRepoPGS(conn)
	transferRepo := transferrepo.NewRepoPGS(conn)
	sessionRepo := sessionrepo.NewRepoPGS(conn)

	tokenMaker, err := tokenpkg.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, errors.New("cannot create token maker")
	}

	userService := userservice.New(userRepo)
	accountService := accountservice.New(accountRepo)
	transferService := transferservice.New(transferRepo, accountService)
	sessionService, err := sessionservice.New(sessionRepo, config, tokenMaker)

	if err != nil {
		return nil, errors.New("cannot initialize session service")
	}

	userHandler := userdelivery.NewHandler(userService, sessionService)
	accountHandler := accountdelivery.NewHandler(accountService)
	transferHandler := transferdelivery.NewHandler(transferService)
	sessionHandler := sessiondelivery.NewHandler(sessionService)

	gin.SetMode(gin.ReleaseMode)
	server := gin.New()

	server.Use(middleware.RequestLogger(logger))
	server.Use(gin.Recovery())

	server.POST("/users", userHandler.Create)
	server.POST("/users/login", userHandler.Login)
	server.POST("/sessions", sessionHandler.RenewAccessToken)

	authRoutes := server.Group("/").Use(middleware.AuthMiddleware(sessionService.TokenMaker))

	authRoutes.POST("/accounts", accountHandler.Create)
	authRoutes.GET("/accounts/:id", accountHandler.Get)
	authRoutes.GET("/accounts", accountHandler.List)

	authRoutes.POST("/transfers", transferHandler.Create)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		err := v.RegisterValidation("currency", accountdelivery.ValidCurrency)
		if err != nil {
			return nil, errors.New("cannot register currency validator")
		}
	}

	return server, nil
}
