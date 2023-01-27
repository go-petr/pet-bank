package delivery

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/internal/transfer"
	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/go-petr/pet-bank/pkg/util"
)

//go:generate mockgen -source handlers.go -destination handlers_mock.go -package delivery
type transferServiceInterface interface {
	TransferTx(ctx context.Context, fromUsername string, arg transfer.CreateTransferParams) (transfer.TransferTxResult, error)
}

type transferHandler struct {
	service transferServiceInterface
}

func NewTransferHandler(ts transferServiceInterface) *transferHandler {
	return &transferHandler{
		service: ts,
	}
}

type transferRequest struct {
	FromAccountID int32  `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int32  `json:"to_account_id" binding:"required,min=1"`
	Amount        string `json:"amount" binding:"required"`
}

type transferResponse struct {
	Data struct {
		Transfer transfer.TransferTxResult `json:"transfer"`
	} `json:"data,omitempty"`
}

func (h *transferHandler) CreateTransfer(ctx *gin.Context) {
	var req transferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrResponse{Error: err.Error()})
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	arg := transfer.CreateTransferParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	result, err := h.service.TransferTx(ctx, authPayload.Username, arg)
	if err != nil {
		switch err {
		case transfer.ErrInvalidOwner:
			ctx.JSON(http.StatusUnauthorized, util.ErrResponse{Error: err.Error()})
			return
		case transfer.ErrInvalidAmount,
			transfer.ErrNegativeAmount,
			transfer.ErrInsufficientBalance,
			transfer.ErrCurrencyMismatch:
			ctx.JSON(http.StatusBadRequest, util.ErrResponse{Error: err.Error()})
			return
		}

		ctx.JSON(http.StatusInternalServerError, util.ErrResponse{Error: err.Error()})
		return
	}

	res := transferResponse{
		Data: struct {
			Transfer transfer.TransferTxResult "json:\"transfer\""
		}{
			result,
		},
	}

	ctx.JSON(http.StatusOK, res)
}
