package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/go-petr/pet-bank/pkg/web"
)

func TestAuthMiddleware(t *testing.T) {
	tokenSymmetricKey := randompkg.String(32)

	tokenMaker, err := tokenpkg.NewPasetoMaker(tokenSymmetricKey)
	if err != nil {
		t.Fatalf("tokenpkg.NewPasetoMaker(%v) returned error: %v", tokenSymmetricKey, err)
	}

	testCases := []struct {
		name           string
		setupAuth      func(t *testing.T, r *http.Request) error
		wantStatusCode int
		wantError      string
		checkResponse  func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "NoAuthorization",
			setupAuth: func(t *testing.T, r *http.Request) error {
				return nil
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      ErrAuthHeaderNotFound.Error(),
		},
		{
			name: "InvalidAuthorizationHeader",
			setupAuth: func(t *testing.T, r *http.Request) error {
				return AddAuthorization(r, tokenMaker, "", "user", time.Minute)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      ErrBadAuthHeaderFormat.Error(),
		},
		{
			name: "UnsupportedAuthorization",
			setupAuth: func(t *testing.T, r *http.Request) error {
				return AddAuthorization(r, tokenMaker, "unsupported", "user", time.Minute)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      ErrUnsupportedAuthType.Error(),
		},
		{
			name: "ExpiredToken",
			setupAuth: func(t *testing.T, r *http.Request) error {
				return AddAuthorization(r, tokenMaker, AuthTypeBearer, "user", -time.Minute)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      tokenpkg.ErrExpiredToken.Error(),
		},
		{
			name: "OK",
			setupAuth: func(t *testing.T, r *http.Request) error {
				return AddAuthorization(r, tokenMaker, AuthTypeBearer, "user", time.Minute)
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.ReleaseMode)
			server := gin.New()

			authPath := "/auth"
			handler := func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{})
			}
			server.GET(authPath, AuthMiddleware(tokenMaker), handler)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			if err != nil {
				t.Fatalf("tokenpkg.NewPasetoMaker(%v) returned error: %v", tokenSymmetricKey, err)
			}

			if err = tc.setupAuth(t, request); err != nil {
				t.Fatalf("tc.setupAuth(t, %v) returned error: %v", request, err)
			}

			server.ServeHTTP(recorder, request)

			if recorder.Code != tc.wantStatusCode {
				t.Errorf("recorder.Code = %v, tc.wantStatusCode = %v, want equal",
					recorder.Code, tc.wantStatusCode)
			}

			got := web.Response{}
			if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
				t.Fatalf("Decoding response body error: %v", err)
			}

			if got.Error != tc.wantError {
				t.Errorf("got.Error = %v, tc.wantError = %v, want equal", got.Error, tc.wantError)
			}
		})
	}
}
