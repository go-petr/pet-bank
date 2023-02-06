package delivery

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/pkg/util"
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

func (h *SessionHandler) RenewAccessToken(ctx *gin.Context) {
	var req renewAccessTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrResponse{Error: err.Error()})
		return
	}

	accessToken, accessTokenExpiresAt, err := h.service.RenewAccessToken(ctx, req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrResponse{Error: err.Error()})
		return
	}

	rsp := renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessTokenExpiresAt,
	}
	ctx.JSON(http.StatusOK, rsp)
}
