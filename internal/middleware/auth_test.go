package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	tokenMaker, err := tokenpkg.NewPasetoMaker(randompkg.String(32))
	require.NoError(t, err)

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "NoAuthorization",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InvalidAuthorizationHeader",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := AddAuthorization(request, tokenMaker, "", "user", time.Minute)
				require.NoError(t, err)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "UnsupportedAuthorization",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := AddAuthorization(request, tokenMaker, "unsupported", "user", time.Minute)
				require.NoError(t, err)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "ExpiredToken",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := AddAuthorization(request, tokenMaker, AuthTypeBearer, "user", -time.Minute)
				require.NoError(t, err)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker tokenpkg.Maker) {
				err := AddAuthorization(request, tokenMaker, AuthTypeBearer, "user", time.Minute)
				require.NoError(t, err)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.ReleaseMode)
			server := gin.New()

			authPath := "/auth"
			server.GET(
				authPath,
				AuthMiddleware(tokenMaker),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, tokenMaker)
			server.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
