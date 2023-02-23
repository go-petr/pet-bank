//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/test"
	"github.com/go-petr/pet-bank/pkg/dbpkg/integrationtest"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/go-petr/pet-bank/pkg/web"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCreateUserAPI(t *testing.T) {
	defer func() {
		integrationtest.Flush(t, server.DB)
	}()

	user := test.SeedUser(t, server.DB)

	var (
		username = "firstuser"
		password = "qwerty"
		fullname = "Foo Boo"
		email    = "foo@boo.email"
	)

	testCases := []struct {
		name           string
		requestBody    gin.H
		wantStatusCode int
		wantError      string
		checkData      func(reqBody gin.H, resp web.Response)
	}{
		{
			name: "OK",
			requestBody: gin.H{
				"username": username,
				"password": password,
				"fullname": fullname,
				"email":    email,
			},
			wantStatusCode: http.StatusOK,
			checkData: func(reqBody gin.H, resp web.Response) {
				if resp.AccessToken == "" {
					t.Error(`resp.AccessToken="", want not empty`)
				}
				if resp.AccessTokenExpiresAt.IsZero() {
					t.Error(`resp.AccessTokenExpiresAt is zero, want non zero`)
				}
				if resp.RefreshToken == "" {
					t.Error(`resp.RefreshToken="", want not empty`)
				}
				if resp.RefreshTokenExpiresAt.IsZero() {
					t.Error(`resp.RefreshTokenExpiresAt is zero, want non zero`)
				}
				if resp.Error != "" {
					t.Errorf(`resp.Error=%q, want ""`, resp.Error)
				}

				gotData, ok := resp.Data.(*struct {
					User domain.UserWihtoutPassword `json:"user,omitempty"`
				})
				if !ok {
					t.Errorf(`resp.Data=%v, failed type conversion`, resp.Data)
				}

				want := domain.UserWihtoutPassword{
					Username:  reqBody["username"].(string),
					FullName:  reqBody["fullname"].(string),
					Email:     reqBody["email"].(string),
					CreatedAt: time.Now().Truncate(time.Second),
				}

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, gotData.User, compareCreatedAt); diff != "" {
					t.Errorf("resp.Data mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "InvalidUsername",
			requestBody: gin.H{
				"username": "user&%",
				"password": password,
				"fullname": fullname,
				"email":    email,
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Username accepts only alphanumeric characters",
		},
		{
			name: "ShortPassword",
			requestBody: gin.H{
				"username": username,
				"password": "short",
				"fullname": fullname,
				"email":    email,
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Password must be at least 6 characters long",
		},
		{
			name: "MissingFullName",
			requestBody: gin.H{
				"username": username,
				"password": password,
				"fullname": "",
				"email":    email,
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "FullName field is required",
		},
		{
			name: "InvalidEmail",
			requestBody: gin.H{
				"username": username,
				"password": password,
				"fullname": fullname,
				"email":    "user%email.com",
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Email must contain a valid email",
		},
		{
			name: "UniqueViolationUsername",
			requestBody: gin.H{
				"username": user.Username,
				"password": user.HashedPassword,
				"fullname": user.FullName,
				"email":    user.Email,
			},
			wantStatusCode: http.StatusConflict,
			wantError:      domain.ErrUsernameAlreadyExists.Error(),
		},
		{
			name: "UniqueViolationEmail",
			requestBody: gin.H{
				"username": username + "2",
				"password": password,
				"fullname": fullname + "2",
				"email":    user.Email,
			},
			wantStatusCode: http.StatusConflict,
			wantError:      domain.ErrEmailALreadyExists.Error(),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {

			// Send request
			body, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Encoding request body error: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			// Test response
			if got := w.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
			}

			resp := web.Response{
				Data: &struct {
					User domain.UserWihtoutPassword `json:"user,omitempty"`
				}{},
			}

			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Errorf("Decoding response body error: %v", err)
			}

			if tc.wantStatusCode != http.StatusOK {
				if resp.Error != tc.wantError {
					t.Errorf(`resp.Error=%q, want %q`, resp.Error, tc.wantError)
				}
			} else {
				tc.checkData(tc.requestBody, resp)
			}
		})
	}
}

func TestLoginUserAPI(t *testing.T) {
	defer func() {
		integrationtest.Flush(t, server.DB)
	}()

	password := randompkg.String(10)
	user := test.SeedUserWith(t, server.DB, password)

	testCases := []struct {
		name           string
		requestBody    gin.H
		wantStatusCode int
		wantError      string
		checkData      func(reqBody gin.H, resp web.Response)
	}{
		{
			name: "OK",
			requestBody: gin.H{
				"username": user.Username,
				"password": password,
			},
			wantStatusCode: http.StatusOK,
			checkData: func(reqBody gin.H, resp web.Response) {
				if resp.AccessToken == "" {
					t.Error(`resp.AccessToken="", want not empty`)
				}
				if resp.AccessTokenExpiresAt.IsZero() {
					t.Error(`resp.AccessTokenExpiresAt is zero, want non zero`)
				}
				if resp.RefreshToken == "" {
					t.Error(`resp.RefreshToken="", want not empty`)
				}
				if resp.RefreshTokenExpiresAt.IsZero() {
					t.Error(`resp.RefreshTokenExpiresAt is zero, want non zero`)
				}
				if resp.Error != "" {
					t.Errorf(`resp.Error=%q, want ""`, resp.Error)
				}

				gotData, ok := resp.Data.(*struct {
					User domain.UserWihtoutPassword `json:"user,omitempty"`
				})
				if !ok {
					t.Errorf(`resp.Data=%v, failed type conversion`, resp.Data)
				}

				want := domain.UserWihtoutPassword{
					Username:  user.Username,
					FullName:  user.FullName,
					Email:     user.Email,
					CreatedAt: user.CreatedAt,
				}

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, gotData.User, compareCreatedAt); diff != "" {
					t.Errorf("resp.Data mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "InvalidUsername",
			requestBody: gin.H{
				"username": "user&%",
				"password": password,
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Username accepts only alphanumeric characters",
		},
		{
			name: "ShortPassword",
			requestBody: gin.H{
				"username": user.Username,
				"password": "short",
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Password must be at least 6 characters long",
		},
		{
			name: "ErrUserNotFound",
			requestBody: gin.H{
				"username": "ErrUserNotFound",
				"password": user.HashedPassword,
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      domain.ErrUserNotFound.Error(),
		},
		{
			name: "ErrWrongPassword",
			requestBody: gin.H{
				"username": user.Username,
				"password": "wrongPass",
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      domain.ErrWrongPassword.Error(),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {

			// Send request
			body, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Encoding request body error: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, "/users/login", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			// Test response
			if got := w.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
			}

			resp := web.Response{
				Data: &struct {
					User domain.UserWihtoutPassword `json:"user,omitempty"`
				}{},
			}

			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Errorf("Decoding response body error: %v", err)
			}

			if tc.wantStatusCode != http.StatusOK {
				if resp.Error != tc.wantError {
					t.Errorf(`resp.Error=%q, want %q`, resp.Error, tc.wantError)
				}
			} else {
				tc.checkData(tc.requestBody, resp)
			}
		})
	}
}
