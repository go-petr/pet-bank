// Package httpserver manages server creation and api routing.
package httpserver

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

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
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/currencypkg"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
)

// Server holds db connection, handlers router and configuration.
type Server struct {
	DB     *sql.DB
	Engine *gin.Engine
	Config configpkg.Config
}

// ServeHTTP implements the http.Handler interface for the Server type.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Engine.ServeHTTP(w, r)
}

// New creates Server type with instantiated domains and routes.
func New(conn *sql.DB, logger zerolog.Logger, config configpkg.Config) (*Server, error) {
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
	engine := gin.New()

	engine.Use(middleware.RequestLogger(logger))
	engine.Use(gin.Recovery())

	engine.POST("/users", userHandler.Create)
	engine.POST("/users/login", userHandler.Login)
	engine.POST("/sessions", sessionHandler.RenewAccessToken)

	authRoutes := engine.Group("/").Use(middleware.AuthMiddleware(sessionService.TokenMaker))

	authRoutes.POST("/accounts", accountHandler.Create)
	authRoutes.GET("/accounts/:id", accountHandler.Get)
	authRoutes.GET("/accounts", accountHandler.List)

	authRoutes.POST("/transfers", transferHandler.Create)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		err := v.RegisterValidation("currency", currencypkg.ValidCurrency)
		if err != nil {
			return nil, errors.New("cannot register currency validator")
		}
	}

	server := &Server{
		DB:     conn,
		Engine: engine,
		Config: config,
	}

	return server, nil
}
