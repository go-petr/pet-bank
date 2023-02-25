// Package sessiondelivery manages delivery layer of sessions.
package sessiondelivery

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

// Service provides service layer interface needed by session delivery layer.
//
//go:generate mockgen -source http.go -destination http_mock.go -package sessiondelivery
type Service interface {
	RenewAccessToken(ctx context.Context, refreshToken string) (string, time.Time, error)
}

// Handler facilitates session delivery layer logic.
type Handler struct {
	service Service
}

// NewHandler returns session handler.
func NewHandler(ss Service) *Handler {
	return &Handler{
		service: ss,
	}
}

type renewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RenewAccessToken handles http request to renew access token.
func (h *Handler) RenewAccessToken(gctx *gin.Context) {
	ctx := gctx.Request.Context()
	l := zerolog.Ctx(ctx)

	var req renewAccessTokenRequest
	if err := gctx.ShouldBindJSON(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			gctx.JSON(http.StatusBadRequest, web.Response{Error: web.GetErrorMsg(ve)})

			return
		}

		l.Error().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, web.Error(errorspkg.ErrInternal))

		return
	}

	accessToken, accessTokenExpiresAt, err := h.service.RenewAccessToken(ctx, req.RefreshToken)
	if err != nil {
		switch err {
		case
			domain.ErrSessionNotFound:
			gctx.JSON(http.StatusUnauthorized, web.Error(err))
			return
		case
			domain.ErrExpiredToken,
			domain.ErrBlockedSession,
			domain.ErrInvalidUser,
			domain.ErrMismatchedRefreshToken,
			domain.ErrExpiredSession:
			gctx.JSON(http.StatusForbidden, web.Error(err))
			return
		}

		l.Info().Err(err).Send()
		gctx.JSON(http.StatusInternalServerError, web.Error(errorspkg.ErrInternal))

		return
	}

	res := web.Response{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessTokenExpiresAt,
	}
	gctx.JSON(http.StatusOK, res)
}
