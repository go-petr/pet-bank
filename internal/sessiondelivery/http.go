// Package sessiondelivery manages delivery layer of sessions.
package sessiondelivery

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/pkg/web"
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

type renewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

// RenewAccessToken handles http request to renew access token.
func (h *Handler) RenewAccessToken(gctx *gin.Context) {
	ctx := gctx.Request.Context()
	l := zerolog.Ctx(ctx)

	var req renewAccessTokenRequest
	if err := gctx.ShouldBindJSON(&req); err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, web.Error(err))

		return
	}

	accessToken, accessTokenExpiresAt, err := h.service.RenewAccessToken(ctx, req.RefreshToken)
	if err != nil {
		gctx.JSON(http.StatusInternalServerError, web.Error(err))

		return
	}

	rsp := renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessTokenExpiresAt,
	}
	gctx.JSON(http.StatusOK, rsp)
}
