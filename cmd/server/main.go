package main

import (
	"database/sql"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/go-petr/pet-bank/pkg/util"
	_ "github.com/lib/pq"

	// ah "github.com/go-petr/pet-bank/internal/account/delivery"
	// ar "github.com/go-petr/pet-bank/internal/account/repo"
	// as "github.com/go-petr/pet-bank/internal/account/service"

	uh "github.com/go-petr/pet-bank/internal/user/delivery"
	ur "github.com/go-petr/pet-bank/internal/user/repo"
	us "github.com/go-petr/pet-bank/internal/user/service"
)

func main() {

	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	server := NewServer(config, conn)

	// Start rung the HTTP server on a specific address.
	err = server.Run(config.ServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}

func NewServer(config util.Config, db *sql.DB) *gin.Engine {

	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		log.Fatal("cannot create token maker:", err)
	}

	userRepo := ur.NewUserRepo(db)
	// accountRepo := ar.NewAccountRepo(db)

	userService := us.NewUserService(userRepo)
	// accountService := as.NewAccountService(accountRepo)

	userHandler := uh.NewUserHandler(userService, tokenMaker, config.AccessTokenDuration)
	// accountHandler := ah.NewAccountHandler(accountService)

	server := gin.Default()
	server.POST("/users", userHandler.CreateUser)
	server.POST("/users/login", userHandler.LoginUser)

	// authRoutes := server.Group("/").Use(middleware.AuthMiddleware(tokenMaker))

	// authRoutes.POST("/accounts", accountHandler.CreateAccount)
	// authRoutes.GET("/accounts/:id", accountHandler.GetAccount)
	// authRoutes.GET("/accounts", accountHandler.ListAccounts)

	// authRoutes.POST("/transfers", server.createTransfer)

	// if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
	// 	v.RegisterValidation("currency", ah.ValidCurrency)
	// }

	return server
}
