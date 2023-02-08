package delivery

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/pkg/util"
	"github.com/rs/zerolog"
)

//go:generate mockgen -source handlers.go -destination handlers_mock.go -package delivery
type SessionServiceInterface interface {
	RenewAccessToken(ctx context.Context, refreshToken string) (string, time.Time, error)
}

type SessionHandler struct {
	service SessionServiceInterface
}

func NewSessionHandler(ss SessionServiceInterface) *SessionHandler {
	return &SessionHandler{
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

func (h *SessionHandler) RenewAccessToken(gctx *gin.Context) {

	ctx := gctx.Request.Context()
	l := zerolog.Ctx(ctx)

	var req renewAccessTokenRequest
	if err := gctx.ShouldBindJSON(&req); err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, util.ErrResponse{Error: err.Error()})
		return
	}

	accessToken, accessTokenExpiresAt, err := h.service.RenewAccessToken(ctx, req.RefreshToken)
	if err != nil {
		gctx.JSON(http.StatusInternalServerError, util.ErrResponse{Error: err.Error()})
		return
	}

	rsp := renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessTokenExpiresAt,
	}
	gctx.JSON(http.StatusOK, rsp)
}
