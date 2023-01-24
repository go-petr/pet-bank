package delivery

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/account"
	"github.com/go-petr/pet-bank/internal/middleware"

	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/go-petr/pet-bank/pkg/util"
)

//go:generate mockgen -source handlers.go -destination handlers_mock.go -package delivery
type accountServiceInterface interface {
	CreateAccount(ctx context.Context, owner, currency string) (account.Account, error)
	GetAccount(ctx context.Context, id int32) (account.Account, error)
	ListAccounts(ctx context.Context, owner string, pageSize, pageID int32) ([]account.Account, error)
}

type accountHandler struct {
	service accountServiceInterface
}

func NewAccountHandler(as accountServiceInterface) accountHandler {
	return accountHandler{service: as}
}

type createAccountRequest struct {
	Currency string `json:"currency" binding:"required,currency"`
}

func (h *accountHandler) CreateAccount(ctx *gin.Context) {

	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrResponse{Error: err.Error()})
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	createdAccount, err := h.service.CreateAccount(ctx, authPayload.Username, req.Currency)
	if err != nil {
		switch err {
		case account.ErrNoOwnerExists:
			ctx.JSON(http.StatusBadRequest, util.ErrResponse{Error: err.Error()})
			return
		case account.ErrCurrencyAlreadyExists:
			ctx.JSON(http.StatusConflict, util.ErrResponse{Error: err.Error()})
			return
		}

		ctx.JSON(http.StatusInternalServerError, util.ErrResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, createdAccount)
}

type getAccountRequest struct {
	ID int32 `uri:"id" binding:"required,min=1"`
}

func (h *accountHandler) GetAccount(ctx *gin.Context) {
	var req getAccountRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrResponse{Error: err.Error()})
		return
	}

	acc, err := h.service.GetAccount(ctx, req.ID)
	if err != nil {
		if err == account.ErrAccountNotFound {
			ctx.JSON(http.StatusNotFound, util.ErrResponse{Error: err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrResponse{Error: err.Error()})
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	if acc.Owner != authPayload.Username {
		err := errors.New("account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, util.ErrResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, acc)
}

type listAccountsRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *accountHandler) ListAccounts(ctx *gin.Context) {
	var req listAccountsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrResponse{Error: err.Error()})
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	accounts, err := h.service.ListAccounts(ctx, authPayload.Username, req.PageSize, req.PageID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, accounts)
}
