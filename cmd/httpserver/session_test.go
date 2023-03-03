//go:build integration

package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/integrationtest"
	"github.com/go-petr/pet-bank/internal/integrationtest/helpers"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/go-petr/pet-bank/pkg/web"
)

func TestRenewAccessTokenAPI(t *testing.T) {
	server := integrationtest.SetupServer(t)

	tokenMaker, err := tokenpkg.NewPasetoMaker(server.Config.TokenSymmetricKey)
	if err != nil {
		t.Fatalf("tokenpkg.NewPasetoMaker(%v) returned error: %v",
			server.Config.TokenSymmetricKey, err)
	}

	duration := server.Config.RefreshTokenDuration

	type requestBody struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	testCases := []struct {
		name           string
		requestBody    func(t *testing.T) requestBody
		wantStatusCode int
		checkData      func(t *testing.T, res web.Response)
		wantError      string
	}{
		{
			name: "OK",
			requestBody: func(t *testing.T) requestBody {
				user := helpers.SeedUser(t, server.DB)

				refreshToken, payload, err := tokenMaker.CreateToken(user.Username, duration)
				if err != nil {
					t.Fatalf("tokenMaker.CreateToken(%v, %v) returned error: %v",
						user.Username, duration, err)
				}

				arg := domain.CreateSessionParams{
					ID:           payload.ID,
					Username:     user.Username,
					RefreshToken: refreshToken,
					UserAgent:    "Mozilla/5.0",
					ClientIP:     "123.123.123.123",
					IsBlocked:    false,
					ExpiresAt:    payload.ExpiredAt,
				}
				helpers.SeedSession(t, server.DB, arg)

				return requestBody{
					RefreshToken: refreshToken,
				}
			},
			wantStatusCode: http.StatusCreated,
			checkData: func(t *testing.T, got web.Response) {
				t.Helper()

				_, err := tokenMaker.VerifyToken(got.AccessToken)
				if err != nil {
					t.Errorf("tokenMaker.VerifyToken(got.AccessToken) returned error: %v", err)
				}
			},
		},
		{
			name: "ErrExpiredToken",
			requestBody: func(t *testing.T) requestBody {
				user := helpers.SeedUser(t, server.DB)

				refreshToken, _, err := tokenMaker.CreateToken(user.Username, time.Nanosecond)
				if err != nil {
					t.Fatalf("tokenMaker.CreateToken(%v, %v) returned error: %v",
						user.Username, duration, err)
				}

				return requestBody{
					RefreshToken: refreshToken,
				}
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      tokenpkg.ErrExpiredToken.Error(),
		},
		{
			name: "ErrSessionNotFound",
			requestBody: func(t *testing.T) requestBody {
				user := helpers.SeedUser(t, server.DB)

				refreshToken, _, err := tokenMaker.CreateToken(user.Username, duration)
				if err != nil {
					t.Fatalf("tokenMaker.CreateToken(%v, %v) returned error: %v",
						user.Username, duration, err)
				}

				return requestBody{
					RefreshToken: refreshToken,
				}
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      domain.ErrSessionNotFound.Error(),
		},
		{
			name: "ErrBlockedSession",
			requestBody: func(t *testing.T) requestBody {
				user := helpers.SeedUser(t, server.DB)

				refreshToken, payload, err := tokenMaker.CreateToken(user.Username, duration)
				if err != nil {
					t.Fatalf("tokenMaker.CreateToken(%v, %v) returned error: %v",
						user.Username, duration, err)
				}

				arg := domain.CreateSessionParams{
					ID:           payload.ID,
					Username:     user.Username,
					RefreshToken: refreshToken,
					UserAgent:    "Mozilla/5.0",
					ClientIP:     "123.123.123.123",
					IsBlocked:    true,
					ExpiresAt:    payload.ExpiredAt,
				}
				helpers.SeedSession(t, server.DB, arg)

				return requestBody{
					RefreshToken: refreshToken,
				}
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      domain.ErrBlockedSession.Error(),
		},
		{
			name: "ErrInvalidUser",
			requestBody: func(t *testing.T) requestBody {
				user := helpers.SeedUser(t, server.DB)

				refreshToken, payload, err := tokenMaker.CreateToken(user.Username, duration)
				if err != nil {
					t.Fatalf("tokenMaker.CreateToken(%v, %v) returned error: %v",
						user.Username, duration, err)
				}

				user2 := helpers.SeedUser(t, server.DB)
				arg := domain.CreateSessionParams{
					ID:           payload.ID,
					Username:     user2.Username,
					RefreshToken: refreshToken,
					UserAgent:    "Mozilla/5.0",
					ClientIP:     "123.123.123.123",
					IsBlocked:    false,
					ExpiresAt:    payload.ExpiredAt,
				}
				helpers.SeedSession(t, server.DB, arg)

				return requestBody{
					RefreshToken: refreshToken,
				}
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      domain.ErrInvalidUser.Error(),
		},
		{
			name: "ErrMismatchedRefreshToken",
			requestBody: func(t *testing.T) requestBody {
				user := helpers.SeedUser(t, server.DB)

				refreshToken, payload, err := tokenMaker.CreateToken(user.Username, duration)
				if err != nil {
					t.Fatalf("tokenMaker.CreateToken(%v, %v) returned error: %v",
						user.Username, duration, err)
				}

				refreshToken1, _, err := tokenMaker.CreateToken(user.Username, duration)
				if err != nil {
					t.Fatalf("tokenMaker.CreateToken(%v, %v) returned error: %v",
						user.Username, duration, err)
				}

				arg := domain.CreateSessionParams{
					ID:           payload.ID,
					Username:     user.Username,
					RefreshToken: refreshToken1,
					UserAgent:    "Mozilla/5.0",
					ClientIP:     "123.123.123.123",
					IsBlocked:    false,
					ExpiresAt:    payload.ExpiredAt,
				}
				helpers.SeedSession(t, server.DB, arg)

				return requestBody{
					RefreshToken: refreshToken,
				}
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      domain.ErrMismatchedRefreshToken.Error(),
		},
		{
			name: "ErrExpiredSession",
			requestBody: func(t *testing.T) requestBody {
				user := helpers.SeedUser(t, server.DB)

				refreshToken, payload, err := tokenMaker.CreateToken(user.Username, duration)
				if err != nil {
					t.Fatalf("tokenMaker.CreateToken(%v, %v) returned error: %v",
						user.Username, duration, err)
				}

				arg := domain.CreateSessionParams{
					ID:           payload.ID,
					Username:     user.Username,
					RefreshToken: refreshToken,
					UserAgent:    "Mozilla/5.0",
					ClientIP:     "123.123.123.123",
					IsBlocked:    false,
					ExpiresAt:    payload.ExpiredAt.Add(-72 * time.Hour),
				}
				helpers.SeedSession(t, server.DB, arg)

				return requestBody{
					RefreshToken: refreshToken,
				}
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      domain.ErrExpiredSession.Error(),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Send request
			body, err := json.Marshal(tc.requestBody(t))
			if err != nil {
				t.Fatalf("Encoding request body error: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			// Test response
			if got := w.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
			}

			res := web.Response{}

			if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
				t.Errorf("Decoding response body error: %v", err)
			}

			if tc.wantStatusCode != http.StatusOK {
				if res.Error != tc.wantError {
					t.Errorf(`res.Error=%q, want %q`, res.Error, tc.wantError)
				}
			} else {
				tc.checkData(t, res)
			}
		})
	}
}
