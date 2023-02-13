package delivery

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/account"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/rs/zerolog"

	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/jsonresponse"
	"github.com/go-petr/pet-bank/pkg/token"
)

//go:generate mockgen -source handlers.go -destination handlers_mock.go -package delivery
type AccountServiceInterface interface {
	CreateAccount(ctx context.Context, owner, currency string) (account.Account, error)
	GetAccount(ctx context.Context, id int32) (account.Account, error)
	ListAccounts(ctx context.Context, owner string, pageSize, pageID int32) ([]account.Account, error)
}

type accountHandler struct {
	service AccountServiceInterface
}

func NewAccountHandler(as AccountServiceInterface) accountHandler {
	return accountHandler{service: as}
}

type createAccountRequest struct {
	Currency string `json:"currency" binding:"required,currency"`
}

func (h *accountHandler) CreateAccount(gctx *gin.Context) {

	ctx := gctx.Request.Context()
	l := zerolog.Ctx(gctx)

	var req createAccountRequest
	if err := gctx.ShouldBindJSON(&req); err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))
		return
	}

	authPayload := gctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	createdAccount, err := h.service.CreateAccount(ctx, authPayload.Username, req.Currency)
	if err != nil {
		switch err {
		case account.ErrNoOwnerExists:
			gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))
			return
		case account.ErrCurrencyAlreadyExists:
			gctx.JSON(http.StatusConflict, jsonresponse.Error(err))
			return
		}

		gctx.JSON(http.StatusInternalServerError, jsonresponse.Error(errorspkg.ErrInternal))
		return
	}

	gctx.JSON(http.StatusOK, createdAccount)
}

type getAccountRequest struct {
	ID int32 `uri:"id" binding:"required,min=1"`
}

func (h *accountHandler) GetAccount(gctx *gin.Context) {

	ctx := gctx.Request.Context()
	l := zerolog.Ctx(gctx)

	var req getAccountRequest
	if err := gctx.ShouldBindUri(&req); err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))
		return
	}

	acc, err := h.service.GetAccount(ctx, req.ID)
	if err != nil {
		if err == account.ErrAccountNotFound {
			gctx.JSON(http.StatusNotFound, jsonresponse.Error(err))
			return
		}
		gctx.JSON(http.StatusInternalServerError, jsonresponse.Error(errorspkg.ErrInternal))
		return
	}

	authPayload := gctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	if acc.Owner != authPayload.Username {
		l.Warn().Err(err).Send()
		err := errors.New("account doesn't belong to the authenticated user")
		gctx.JSON(http.StatusUnauthorized, jsonresponse.Error(err))
		return
	}

	gctx.JSON(http.StatusOK, acc)
}

type listAccountsRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *accountHandler) ListAccounts(gctx *gin.Context) {

	ctx := gctx.Request.Context()
	l := zerolog.Ctx(gctx)

	var req listAccountsRequest
	if err := gctx.ShouldBindQuery(&req); err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))
		return
	}

	authPayload := gctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	accounts, err := h.service.ListAccounts(ctx, authPayload.Username, req.PageSize, req.PageID)
	if err != nil {
		gctx.JSON(http.StatusInternalServerError, jsonresponse.Error(errorspkg.ErrInternal))
		return
	}

	gctx.JSON(http.StatusOK, accounts)
}
