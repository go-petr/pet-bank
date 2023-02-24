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

type data struct {
	Account domain.Account `json:"account"`
}
type response struct {
	Data data `json:"data,omitempty"`
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
		var (
			ve     validator.ValidationErrors
			errMsg string
		)

		if errors.As(err, &ve) {
			field := ve[0]
			errMsg = field.Field() + web.GetErrorMsg(field)
		}

		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, web.Response{Error: errMsg})

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

	res := response{
		Data: data{createdAccount},
	}

	gctx.JSON(http.StatusOK, res)
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
		var (
			ve     validator.ValidationErrors
			errMsg string
		)

		if errors.As(err, &ve) {
			field := ve[0]
			errMsg = field.Field() + web.GetErrorMsg(field)
		}

		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, web.Response{Error: errMsg})

		return
	}

	acc, err := h.service.Get(ctx, req.ID)
	if err != nil {
		if err == domain.ErrAccountNotFound {
			gctx.JSON(http.StatusNotFound, web.Error(err))
			return
		}

		gctx.JSON(http.StatusInternalServerError, web.Error(errorspkg.ErrInternal))

		return
	}

	authPayload := gctx.MustGet(middleware.AuthPayloadKey).(*tokenpkg.Payload)
	if acc.Owner != authPayload.Username {
		l.Warn().Err(err).Send()
		gctx.JSON(http.StatusUnauthorized, web.Error(domain.ErrAccountOwnerMismatch))

		return
	}

	res := response{
		Data: data{acc},
	}

	gctx.JSON(http.StatusOK, res)
}

type listRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

type dataAccounts struct {
	Accounts []domain.Account `json:"accounts"`
}
type responseAccounts struct {
	Data dataAccounts `json:"data,omitempty"`
}

// List handles http request to list accounts.
func (h *Handler) List(gctx *gin.Context) {
	ctx := gctx.Request.Context()
	l := zerolog.Ctx(gctx)

	var req listRequest
	if err := gctx.ShouldBindJSON(&req); err != nil {
		var (
			ve     validator.ValidationErrors
			errMsg string
		)

		if errors.As(err, &ve) {
			field := ve[0]
			errMsg = field.Field() + web.GetErrorMsg(field)
		}

		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, web.Response{Error: errMsg})

		return
	}

	authPayload := gctx.MustGet(middleware.AuthPayloadKey).(*tokenpkg.Payload)

	accounts, err := h.service.List(ctx, authPayload.Username, req.PageSize, req.PageID)
	if err != nil {

		gctx.JSON(http.StatusInternalServerError, web.Error(errorspkg.ErrInternal))

		return
	}

	res := responseAccounts{
		Data: dataAccounts{accounts},
	}

	gctx.JSON(http.StatusOK, res)
}
