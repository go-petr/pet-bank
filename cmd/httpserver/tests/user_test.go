//go:build integration

package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/userrepo"
	"github.com/go-petr/pet-bank/pkg/web"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func SeedUser(db *sql.DB) (domain.User, error) {

	row := db.QueryRowContext(context.Background(), userrepo.CreateQuery,
		"SeedUser",
		"arg.HashedPassword",
		"SeedUser.FullName",
		"SeedUser@Email.com",
	)

	var u domain.User

	err := row.Scan(
		&u.Username,
		&u.HashedPassword,
		&u.FullName,
		&u.Email,
		&u.PasswordChangedAt,
		&u.CreatedAt,
	)

	if err != nil {
		return u, err
	}

	return u, nil
}

func TestCreateUserAPI(t *testing.T) {
	defer func() {
		if err := DeleteUsers(server.DB); err != nil {
			t.Errorf("Clearing database error: %v", err)
		}
	}()

	seededUser, err := SeedUser(server.DB)
	if err != nil {
		t.Fatalf("error seeding users: %v", err)
	}

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
			wantError:      "",
			checkData: func(reqBody gin.H, resp web.Response) {
				if resp.AccessToken == "" {
					t.Error(`resp.AccessToken="", want not empty`)
				}
				if resp.AccessTokenExpiresAt == "" {
					t.Error(`resp.AccessTokenExpiresAt="", want not empty`)
				}
				if resp.RefreshToken == "" {
					t.Error(`resp.RefreshToken="", want not empty`)
				}
				if resp.RefreshTokenExpiresAt == "" {
					t.Error(`resp.RefreshTokenExpiresAt="", want not empty`)
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

				wantData := domain.UserWihtoutPassword{
					Username: reqBody["username"].(string),
					FullName: reqBody["fullname"].(string),
					Email:    reqBody["email"].(string),
				}

				ignoreCreatedAt := cmpopts.IgnoreFields(domain.UserWihtoutPassword{}, "CreatedAt")
				if diff := cmp.Diff(wantData, gotData.User, ignoreCreatedAt); diff != "" {
					t.Errorf("resp.Data mismatch (-want +got):\n%s", diff)
				}

				delta := cmpopts.EquateApproxTime(time.Minute)
				currentTime := time.Now()
				if !cmp.Equal(gotData.User.CreatedAt, currentTime, delta) {
					t.Errorf("gotData.User.CreatedAt=%v, want %v +- minute", gotData.User.CreatedAt, currentTime)
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
				"username": seededUser.Username,
				"password": seededUser.HashedPassword,
				"fullname": seededUser.FullName,
				"email":    seededUser.Email,
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
				"email":    seededUser.Email,
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
