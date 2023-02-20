// Package userdelivery manages delivery layer of users.
package userdelivery

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/web"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

// Service provides service layer interface needed by user delivery layer.
//
//go:generate mockgen -source http.go -destination http_mock.go -package userdelivery
type Service interface {
	Create(ctx context.Context, username, password, fullname, email string) (domain.UserWihtoutPassword, error)
	CheckPassword(ctx context.Context, username, password string) (domain.UserWihtoutPassword, error)
}

// SessionMaker facilitates session creation.
//
//go:generate mockgen -source http.go -destination http_mock.go -package userdelivery
type SessionMaker interface {
	Create(ctx context.Context, arg domain.CreateSessionParams) (string, time.Time, domain.Session, error)
}

// Handler facilitates user delivery layer logic.
type Handler struct {
	service      Service
	sessionMaker SessionMaker
}

// NewHandler returns user handler.
func NewHandler(us Service, sm SessionMaker) *Handler {
	return &Handler{
		service:      us,
		sessionMaker: sm,
	}
}

type data struct {
	User domain.UserWihtoutPassword `json:"user,omitempty"`
}
type response struct {
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	Data                  data      `json:"data,omitempty"`
}

type createRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"fullname" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

// Create handles http request to create user.
func (h *Handler) Create(gctx *gin.Context) {
	ctx := gctx.Request.Context()
	l := zerolog.Ctx(ctx)

	var req createRequest
	if err := gctx.ShouldBindJSON(&req); err != nil {
		var (
			ve     validator.ValidationErrors
			errMsg string
		)

		if errors.As(err, &ve) {
			field := ve[0]
			errMsg = field.Field() + web.GetErrorMsg(field)
		}

		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, web.Response{Error: errMsg})

		return
	}

	createdUser, err := h.service.Create(ctx, req.Username, req.Password, req.FullName, req.Email)
	if err != nil {
		switch err {
		case domain.ErrUsernameAlreadyExists:
			gctx.JSON(http.StatusConflict, web.Error(err))
			return
		case domain.ErrEmailALreadyExists:
			gctx.JSON(http.StatusConflict, web.Error(err))
			return
		}

		gctx.JSON(http.StatusInternalServerError, web.Error(errorspkg.ErrInternal))

		return
	}

	arg := domain.CreateSessionParams{
		Username:  req.Username,
		UserAgent: gctx.Request.UserAgent(),
		ClientIP:  gctx.ClientIP(),
	}

	accessToken, accessTokenExpiresAt, session, err := h.sessionMaker.Create(ctx, arg)
	if err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusInternalServerError, web.Error(errorspkg.ErrInternal))

		return
	}

	res := response{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt,
		RefreshToken:          session.RefreshToken,
		RefreshTokenExpiresAt: session.ExpiresAt,
		Data:                  data{User: createdUser},
	}

	gctx.JSON(http.StatusOK, res)
}

type loginRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
}

// Login handlek http login request and returns user and session data.
func (h *Handler) Login(gctx *gin.Context) {
	ctx := gctx.Request.Context()
	l := zerolog.Ctx(ctx)

	var req loginRequest
	if err := gctx.ShouldBindJSON(&req); err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, web.Error(err))

		return
	}

	userWihtoutPassword, err := h.service.CheckPassword(ctx, req.Username, req.Password)
	if err != nil {
		switch err {
		case domain.ErrUserNotFound:
			gctx.JSON(http.StatusNotFound, web.Error(err))
			return
		case domain.ErrWrongPassword:
			gctx.JSON(http.StatusUnauthorized, web.Error(err))
			return
		}

		gctx.JSON(http.StatusInternalServerError, web.Error(errorspkg.ErrInternal))

		return
	}

	arg := domain.CreateSessionParams{
		Username:  req.Username,
		UserAgent: gctx.Request.UserAgent(),
		ClientIP:  gctx.ClientIP(),
	}

	accessToken, accessTokenExpiresAt, session, err := h.sessionMaker.Create(ctx, arg)
	if err != nil {
		l.Warn().Err(err).Send()
		gctx.JSON(http.StatusInternalServerError, web.Error(errorspkg.ErrInternal))

		return
	}

	res := response{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt,
		RefreshToken:          session.RefreshToken,
		RefreshTokenExpiresAt: session.ExpiresAt,
		Data:                  data{User: userWihtoutPassword},
	}

	gctx.JSON(http.StatusOK, res)
}
