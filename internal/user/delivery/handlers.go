package delivery

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/user"
	"github.com/go-petr/pet-bank/pkg/token"
	"github.com/go-petr/pet-bank/pkg/util"
)

//go:generate mockgen -source handlers.go -destination handlers_mock.go -package delivery
type userServiceInterface interface {
	CreateUser(ctx context.Context, username, password, fullname, email string) (user.UserWihtoutPassword, error)
	CheckPassword(ctx context.Context, username, password string) (user.UserWihtoutPassword, error)
}

type userHandler struct {
	service       userServiceInterface
	tokenMaker    token.Maker
	tokenDuration time.Duration
}

func NewUserHandler(us userServiceInterface, tm token.Maker, td time.Duration) *userHandler {
	return &userHandler{
		service:       us,
		tokenMaker:    tm,
		tokenDuration: td,
	}
}

type userResponse struct {
	Token string `json:"token,omitempty"`
	Data  struct {
		User user.UserWihtoutPassword `json:"user,omitempty"`
	} `json:"data,omitempty"`
}

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"fullname" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

func (h *userHandler) CreateUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrResponse{Error: err.Error()})
		return
	}

	createdUser, err := h.service.CreateUser(ctx, req.Username, req.Password, req.FullName, req.Email)
	if err != nil {
		switch err {
		case user.ErrUserNotFound:
			ctx.JSON(http.StatusBadRequest, util.ErrResponse{Error: err.Error()})
			return
		case user.ErrUsernameAlreadyExists:
			ctx.JSON(http.StatusConflict, util.ErrResponse{Error: err.Error()})
			return
		case user.ErrEmailALreadyExists:
			ctx.JSON(http.StatusConflict, util.ErrResponse{Error: err.Error()})
			return
		}

		ctx.JSON(http.StatusInternalServerError, util.ErrResponse{Error: err.Error()})
		return
	}

	accessToken, err := h.tokenMaker.CreateToken(req.Username, h.tokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrResponse{Error: err.Error()})
		return
	}

	res := userResponse{
		Token: accessToken,
		Data: struct {
			User user.UserWihtoutPassword "json:\"user,omitempty\""
		}{
			User: createdUser,
		},
	}

	ctx.JSON(http.StatusOK, res)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
}

func (h *userHandler) LoginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrResponse{Error: err.Error()})
		return
	}

	userWihtoutPassword, err := h.service.CheckPassword(ctx, req.Username, req.Password)
	if err != nil {
		switch err {
		case user.ErrUserNotFound:
			ctx.JSON(http.StatusNotFound, util.ErrResponse{Error: err.Error()})
			return
		case user.ErrWrongPassword:
			ctx.JSON(http.StatusUnauthorized, util.ErrResponse{Error: err.Error()})
			return
		}

		ctx.JSON(http.StatusInternalServerError, util.ErrResponse{Error: err.Error()})
		return
	}

	accessToken, err := h.tokenMaker.CreateToken(req.Username, h.tokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrResponse{Error: err.Error()})
		return
	}

	res := userResponse{
		Token: accessToken,
		Data: struct {
			User user.UserWihtoutPassword "json:\"user,omitempty\""
		}{
			User: userWihtoutPassword,
		},
	}

	ctx.JSON(http.StatusOK, res)
}
