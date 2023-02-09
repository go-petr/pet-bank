package delivery

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/internal/transfer"
	"github.com/go-petr/pet-bank/pkg/apperrors"
	"github.com/go-petr/pet-bank/pkg/jsonresponse"
	"github.com/go-petr/pet-bank/pkg/token"
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

func (h *transferHandler) CreateTransfer(gctx *gin.Context) {

	ctx := gctx.Request.Context()
	l := zerolog.Ctx(ctx)

	var req transferRequest
	if err := gctx.ShouldBindJSON(&req); err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))
		return
	}

	authPayload := gctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	arg := transfer.CreateTransferParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	result, err := h.service.TransferTx(ctx, authPayload.Username, arg)
	if err != nil {
		l.Info().Err(err).Send()
		switch err {
		case transfer.ErrInvalidOwner:
			gctx.JSON(http.StatusUnauthorized, jsonresponse.Error(err))
			return
		case transfer.ErrInvalidAmount,
			transfer.ErrNegativeAmount,
			transfer.ErrInsufficientBalance,
			transfer.ErrCurrencyMismatch:
			gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))
			return
		}

		gctx.JSON(http.StatusInternalServerError, jsonresponse.Error(apperrors.ErrInternal))
		return
	}

	res := transferResponse{
		Data: struct {
			Transfer transfer.TransferTxResult "json:\"transfer\""
		}{
			result,
		},
	}

	gctx.JSON(http.StatusOK, res)
}
