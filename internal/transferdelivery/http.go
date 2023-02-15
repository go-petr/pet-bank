// Package transferdelivery manages delivery layer of transfers.
package transferdelivery

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/jsonresponse"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
)

// Service provides service layer interface needed by transfer delivery layer.
//
//go:generate mockgen -source http.go -destination http_mock.go -package transferdelivery
type Service interface {
	Transfer(ctx context.Context, fromUsername string, arg domain.CreateTransferParams) (domain.TransferTxResult, error)
}

// Handler facilitates transfer delivery layer logic.
type Handler struct {
	service Service
}

// NewHandler returns transfer handler.
func NewHandler(ts Service) *Handler {
	return &Handler{
		service: ts,
	}
}

type request struct {
	FromAccountID int32  `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int32  `json:"to_account_id" binding:"required,min=1"`
	Amount        string `json:"amount" binding:"required"`
}

type data struct {
	Transfer domain.TransferTxResult `json:"transfer"`
}

type response struct {
	Data data `json:"data,omitempty"`
}

// Create handles http request to create a transfer between two accounts.
func (h *Handler) Create(gctx *gin.Context) {
	ctx := gctx.Request.Context()
	l := zerolog.Ctx(ctx)

	var req request
	if err := gctx.ShouldBindJSON(&req); err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))

		return
	}

	authPayload := gctx.MustGet(middleware.AuthPayloadKey).(*tokenpkg.Payload)

	arg := domain.CreateTransferParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	result, err := h.service.Transfer(ctx, authPayload.Username, arg)
	if err != nil {
		l.Info().Err(err).Send()

		switch err {
		case
			domain.ErrInvalidOwner:
			gctx.JSON(http.StatusUnauthorized, jsonresponse.Error(err))

			return
		case
			domain.ErrInvalidAmount,
			domain.ErrNegativeAmount,
			domain.ErrInsufficientBalance,
			domain.ErrCurrencyMismatch:
			gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))

			return
		}

		gctx.JSON(http.StatusInternalServerError, jsonresponse.Error(errorspkg.ErrInternal))

		return
	}

	res := response{
		Data: data{result},
	}

	gctx.JSON(http.StatusOK, res)
}
