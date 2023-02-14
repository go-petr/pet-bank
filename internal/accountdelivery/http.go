// Package accountdelivery manages delivery layer of accounts.
package accountdelivery

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/rs/zerolog"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/jsonresponse"
	"github.com/go-petr/pet-bank/pkg/token"
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
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))

		return
	}

	authPayload := gctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	createdAccount, err := h.service.Create(ctx, authPayload.Username, req.Currency)
	if err != nil {
		switch err {
		case domain.ErrOwnerNotFound:
			gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))
			return
		case domain.ErrCurrencyAlreadyExists:
			gctx.JSON(http.StatusConflict, jsonresponse.Error(err))
			return
		}

		gctx.JSON(http.StatusInternalServerError, jsonresponse.Error(errorspkg.ErrInternal))

		return
	}

	gctx.JSON(http.StatusOK, createdAccount)
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
		gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))

		return
	}

	acc, err := h.service.Get(ctx, req.ID)
	if err != nil {
		if err == domain.ErrAccountNotFound {
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
		gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))

		return
	}

	authPayload := gctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	accounts, err := h.service.List(ctx, authPayload.Username, req.PageSize, req.PageID)
	if err != nil {
		gctx.JSON(http.StatusInternalServerError, jsonresponse.Error(errorspkg.ErrInternal))
		return
	}

	gctx.JSON(http.StatusOK, accounts)
}