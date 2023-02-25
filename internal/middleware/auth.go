package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/go-petr/pet-bank/pkg/web"
)

var (
	// AuthHeaderKey is the key for authorization header.
	AuthHeaderKey = "authorization"
	// AuthTypeBearer is the type of athorization token.
	AuthTypeBearer = "bearer"
	// AuthPayloadKey is the key for authorization payload.
	AuthPayloadKey = "authorization_payload"
	// ErrAuthHeaderNotFound indicates that an authorization header is not found.
	ErrAuthHeaderNotFound = errors.New("authorization header is not found")
	// ErrBadAuthHeaderFormat indicates invalid authorization header format.
	ErrBadAuthHeaderFormat = errors.New("invalid authorization header format")
	// ErrUnsupportedAuthType indicates unsupported authorization type.
	ErrUnsupportedAuthType = errors.New("unsupported authorization type")
)

// AddAuthorization sets authorization token to the given request.
func AddAuthorization(r *http.Request, tm tokenpkg.Maker, authType string, username string, d time.Duration) error {
	token, _, err := tm.CreateToken(username, d)
	if err != nil {
		return err
	}

	authorizationHeader := fmt.Sprintf("%s %s", authType, token)
	r.Header.Set(AuthHeaderKey, authorizationHeader)

	return nil
}

// AuthMiddleware verifies request authorization token.
func AuthMiddleware(tokenMaker tokenpkg.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(AuthHeaderKey)
		if len(authorizationHeader) == 0 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, web.Error(ErrAuthHeaderNotFound))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, web.Error(ErrBadAuthHeaderFormat))
			return
		}

		authType := strings.ToLower(fields[0])
		if authType != AuthTypeBearer {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, web.Error(ErrUnsupportedAuthType))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)

		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, web.Error(err))
			return
		}

		ctx.Set(AuthPayloadKey, payload)
		ctx.Next()
	}
}
