package delivery

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/user"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/jsonresponse"
	"github.com/rs/zerolog"
)

//go:generate mockgen -source handlers.go -destination handlers_mock.go -package delivery
type userServiceInterface interface {
	CreateUser(ctx context.Context, username, password, fullname, email string) (user.UserWihtoutPassword, error)
	CheckPassword(ctx context.Context, username, password string) (user.UserWihtoutPassword, error)
}

type SessionMakerInterface interface {
	Create(ctx context.Context, arg domain.CreateSessionParams) (string, time.Time, domain.Session, error)
}

type userHandler struct {
	service  userServiceInterface
	sessions SessionMakerInterface
}

func NewUserHandler(us userServiceInterface, sm SessionMakerInterface) *userHandler {
	return &userHandler{
		service:  us,
		sessions: sm,
	}
}

type userResponse struct {
	AccessToken           string    `json:"token,omitempty"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token,omitempty"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	Data                  struct {
		User user.UserWihtoutPassword `json:"user,omitempty"`
	} `json:"data,omitempty"`
}

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"fullname" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

func (h *userHandler) CreateUser(gctx *gin.Context) {

	ctx := gctx.Request.Context()
	l := zerolog.Ctx(ctx)

	var req createUserRequest
	if err := gctx.ShouldBindJSON(&req); err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))
		return
	}

	createdUser, err := h.service.CreateUser(ctx, req.Username, req.Password, req.FullName, req.Email)
	if err != nil {
		switch err {
		case user.ErrUsernameAlreadyExists:
			gctx.JSON(http.StatusConflict, jsonresponse.Error(err))
			return
		case user.ErrEmailALreadyExists:
			gctx.JSON(http.StatusConflict, jsonresponse.Error(err))
			return
		}

		gctx.JSON(http.StatusInternalServerError, jsonresponse.Error(errorspkg.ErrInternal))
		return
	}

	arg := domain.CreateSessionParams{
		Username:  req.Username,
		UserAgent: gctx.Request.UserAgent(),
		ClientIP:  gctx.ClientIP(),
	}

	accessToken, accessTokenExpiresAt, session, err := h.sessions.Create(ctx, arg)
	if err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusInternalServerError, jsonresponse.Error(errorspkg.ErrInternal))
		return
	}

	res := userResponse{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt,
		RefreshToken:          session.RefreshToken,
		RefreshTokenExpiresAt: session.ExpiresAt,
		Data: struct {
			User user.UserWihtoutPassword "json:\"user,omitempty\""
		}{
			User: createdUser,
		},
	}

	gctx.JSON(http.StatusOK, res)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
}

func (h *userHandler) LoginUser(gctx *gin.Context) {

	ctx := gctx.Request.Context()
	l := zerolog.Ctx(ctx)

	var req loginUserRequest
	if err := gctx.ShouldBindJSON(&req); err != nil {
		l.Info().Err(err).Send()
		gctx.JSON(http.StatusBadRequest, jsonresponse.Error(err))
		return
	}

	userWihtoutPassword, err := h.service.CheckPassword(ctx, req.Username, req.Password)
	if err != nil {
		switch err {
		case user.ErrUserNotFound:
			gctx.JSON(http.StatusNotFound, jsonresponse.Error(err))
			return
		case user.ErrWrongPassword:
			gctx.JSON(http.StatusUnauthorized, jsonresponse.Error(err))
			return
		}

		gctx.JSON(http.StatusInternalServerError, jsonresponse.Error(errorspkg.ErrInternal))
		return
	}

	arg := domain.CreateSessionParams{
		Username:  req.Username,
		UserAgent: gctx.Request.UserAgent(),
		ClientIP:  gctx.ClientIP(),
	}

	accessToken, accessTokenExpiresAt, session, err := h.sessions.Create(ctx, arg)
	if err != nil {
		l.Warn().Err(err).Send()
		gctx.JSON(http.StatusInternalServerError, jsonresponse.Error(errorspkg.ErrInternal))
		return
	}

	res := userResponse{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt,
		RefreshToken:          session.RefreshToken,
		RefreshTokenExpiresAt: session.ExpiresAt,
		Data: struct {
			User user.UserWihtoutPassword "json:\"user,omitempty\""
		}{
			User: userWihtoutPassword,
		},
	}

	gctx.JSON(http.StatusOK, res)
}
