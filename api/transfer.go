package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/go-petr/pet-bank/db/sqlc"
	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/shopspring/decimal"
)

type transferRequest struct {
	FromAccountID int32  `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int32  `json:"to_account_id" binding:"required,min=1"`
	Amount        string `json:"amount" binding:"required"`
	Currency      string `json:"currency" binding:"required,currency"`
}

func (server *Server) validTransferRequest(ctx *gin.Context, FromAccountID, ToAccountID int32, amount, currrency string) (*db.Account, bool) {

	amountDecimal, err := decimal.NewFromString(amount)
	if err != nil {
		err := fmt.Errorf("ivalid transfer amount")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return nil, false
	}

	if amountDecimal.LessThanOrEqual(decimal.Zero) {
		err := fmt.Errorf("ivalid transfer amount")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return nil, false
	}

	FromAccount, err := server.store.GetAccount(ctx, FromAccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return nil, false
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	}

	currentFromAccountBalance, err := decimal.NewFromString(FromAccount.Balance)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return nil, false
	}

	if currentFromAccountBalance.LessThan(amountDecimal) {
		err := fmt.Errorf("not enough balance")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return nil, false
	}

	if FromAccount.Currency != currrency {
		err := fmt.Errorf("account [%d] currency mismatch: %s vs %s", FromAccount.ID, FromAccount.Currency, currrency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return nil, false
	}

	ToAccount, err := server.store.GetAccount(ctx, ToAccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return nil, false
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return nil, false
	}

	if ToAccount.Currency != currrency {
		err := fmt.Errorf("account [%d] currency mismatch: %s vs %s", ToAccount.ID, ToAccount.Currency, currrency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return nil, false
	}

	return &FromAccount, true
}

func (server *Server) createTransfer(ctx *gin.Context) {
	var req transferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	fromAccount, valid := server.validTransferRequest(ctx, req.FromAccountID, req.ToAccountID, req.Amount, req.Currency)
	if !valid {
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if fromAccount.Owner != authPayload.Username {
		err := errors.New("from account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	arg := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, result)
}
