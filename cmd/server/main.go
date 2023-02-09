package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"

	"github.com/go-petr/pet-bank/pkg/appconfig"
	"github.com/go-petr/pet-bank/pkg/token"
	_ "github.com/lib/pq"

	ah "github.com/go-petr/pet-bank/internal/account/delivery"
	ar "github.com/go-petr/pet-bank/internal/account/repo"
	as "github.com/go-petr/pet-bank/internal/account/service"
	sh "github.com/go-petr/pet-bank/internal/session/delivery"
	sr "github.com/go-petr/pet-bank/internal/session/repo"
	ss "github.com/go-petr/pet-bank/internal/session/service"

	"github.com/go-petr/pet-bank/internal/middleware"
	th "github.com/go-petr/pet-bank/internal/transfer/delivery"
	tr "github.com/go-petr/pet-bank/internal/transfer/repo"
	ts "github.com/go-petr/pet-bank/internal/transfer/service"
	uh "github.com/go-petr/pet-bank/internal/user/delivery"
	ur "github.com/go-petr/pet-bank/internal/user/repo"
	us "github.com/go-petr/pet-bank/internal/user/service"
)

func main() {

	config, err := appconfig.Load("./configs")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db:")
	}

	logger := middleware.GetLogger(config)

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot connect to db:")
	}

	userRepo := ur.NewUserRepo(conn)
	accountRepo := ar.NewAccountRepo(conn)
	transferRepo := tr.NewTransferRepo(conn)
	sessionRepo := sr.NewSessionRepo(conn)

	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot create token maker:")
	}

	userService := us.NewUserService(userRepo)
	accountService := as.NewAccountService(accountRepo)
	transferService := ts.NewTransferService(transferRepo, accountService)
	sessionService, err := ss.NewSessionService(sessionRepo, config, tokenMaker)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot initialize session service:")
	}

	userHandler := uh.NewUserHandler(userService, sessionService)
	accountHandler := ah.NewAccountHandler(accountService)
	transferHandler := th.NewTransferHandler(transferService)
	sessionHandler := sh.NewSessionHandler(sessionService)

	gin.SetMode(gin.ReleaseMode)
	server := gin.New()

	server.Use(middleware.RequestLogger(logger))
	server.Use(gin.Recovery())

	server.POST("/users", userHandler.CreateUser)
	server.POST("/users/login", userHandler.LoginUser)
	server.POST("/sessions", sessionHandler.RenewAccessToken)

	authRoutes := server.Group("/").Use(middleware.AuthMiddleware(sessionService.TokenMaker))

	authRoutes.POST("/accounts", accountHandler.CreateAccount)
	authRoutes.GET("/accounts/:id", accountHandler.GetAccount)
	authRoutes.GET("/accounts", accountHandler.ListAccounts)

	authRoutes.POST("/transfers", transferHandler.CreateTransfer)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", ah.ValidCurrency)
	}

	err = server.Run(config.ServerAddress)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot start server:")
	}
}
