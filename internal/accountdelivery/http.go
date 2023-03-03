// Package accountdelivery manages delivery layer of accounts.
package accountdelivery

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/web"

	"github.com/go-petr/pet-bank/pkg/tokenpkg"
)

// Service provides service layer interface needed by account delivery layer.
//
//go:generate mockgen -source http.go -destination http_mock.go -package accountdelivery
type Service interface {
	Create(ctx context.Context, owner, currency string) (domain.Account, error)
	Get(ctx context.Context, id int32) (domain.Account, error)
	List(ctx context.Context, owner string, pageSize, pageID int32) ([]domain.Account, error)
}

// Handler facilitates account delivery layer logic.
type Handler struct {
	service Service
}

// NewHandler returns account handler.
func NewHandler(as Service) Handler {
	return Handler{service: as}
}

type createRequest struct {
	Currency string `json:"currency" binding:"required,currency"`
}

// Create handles http request to create account.
func (h *Handler) Create(gctx *gin.Context) {
	ctx := gctx.Request.Context()
	l := zerolog.Ctx(gctx)

	var req createRequest
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

	authPayload := gctx.MustGet(middleware.AuthPayloadKey).(*tokenpkg.Payload)

	createdAccount, err := h.service.Create(ctx, authPayload.Username, req.Currency)
	if err != nil {
		switch err {
		case domain.ErrOwnerNotFound:
			gctx.JSON(http.StatusBadRequest, web.Error(err))
			return
		case domain.ErrCurrencyAlreadyExists:
			gctx.JSON(http.StatusConflict, web.Error(err))
			return
		}

		gctx.JSON(http.StatusInternalServerError, web.Error(errorspkg.ErrInternal))

		return
	}

	res := web.Response{
		Data: &struct {
			Account domain.Account `json:"account"`
		}{
			Account: createdAccount,
		},
	}

	gctx.JSON(http.StatusCreated, res)
}

type getRequest struct {
	ID int32 `uri:"id" binding:"required,min=1"`
}

// Get handles http request to get account.
func (h *Handler) Get(gctx *gin.Context) {
	ctx := gctx.Request.Context()
	l := zerolog.Ctx(gctx)

	var req getRequest
	if err := gctx.ShouldBindUri(&req); err != nil {
		l.Info().Err(err).Send()

		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			gctx.JSON(http.StatusBadRequest, web.Response{Error: web.GetErrorMsg(ve)})

			return
		}

		gctx.JSON(http.StatusBadRequest, web.Error(err))

		return
	}

	account, err := h.service.Get(ctx, req.ID)
	if err != nil {
		if err == domain.ErrAccountNotFound {
			gctx.JSON(http.StatusNotFound, web.Error(err))
			return
		}

		gctx.JSON(http.StatusInternalServerError, web.Error(errorspkg.ErrInternal))

		return
	}

	authPayload := gctx.MustGet(middleware.AuthPayloadKey).(*tokenpkg.Payload)
	if account.Owner != authPayload.Username {
		l.Warn().Err(err).Send()
		gctx.JSON(http.StatusUnauthorized, web.Error(domain.ErrAccountOwnerMismatch))

		return
	}

	res := web.Response{
		Data: &struct {
			Account domain.Account `json:"account"`
		}{
			Account: account,
		},
	}

	gctx.JSON(http.StatusOK, res)
}

type listRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

// List handles http request to list accounts.
func (h *Handler) List(gctx *gin.Context) {
	ctx := gctx.Request.Context()
	l := zerolog.Ctx(gctx)

	var req listRequest
	if err := gctx.ShouldBindQuery(&req); err != nil {
		l.Info().Err(err).Send()

		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			gctx.JSON(http.StatusBadRequest, web.Response{Error: web.GetErrorMsg(ve)})

			return
		}

		gctx.JSON(http.StatusBadRequest, web.Error(err))

		return
	}

	authPayload := gctx.MustGet(middleware.AuthPayloadKey).(*tokenpkg.Payload)

	accounts, err := h.service.List(ctx, authPayload.Username, req.PageID, req.PageSize)
	if err != nil {
		gctx.JSON(http.StatusInternalServerError, web.Error(errorspkg.ErrInternal))

		return
	}

	res := web.Response{
		Data: &struct {
			Accounts []domain.Account `json:"accounts"`
		}{
			Accounts: accounts,
		},
	}

	gctx.JSON(http.StatusOK, res)
}
