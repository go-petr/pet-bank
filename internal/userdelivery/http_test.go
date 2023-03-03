// Package userdelivery manages delivery layer of users.
package userdelivery

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/userservice"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/go-petr/pet-bank/pkg/web"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/golang/mock/gomock"
)

var (
	testConfig configpkg.Config
)

func TestMain(m *testing.M) {
	testConfig = configpkg.Config{
		TokenSymmetricKey:   randompkg.String(32),
		AccessTokenDuration: time.Minute,
	}

	gin.SetMode(gin.ReleaseMode)
	os.Exit(m.Run())
}

func TestCreate(t *testing.T) {
	user := domain.User{
		Username:       randompkg.Owner(),
		HashedPassword: randompkg.String(10),
		FullName:       randompkg.Owner(),
		Email:          randompkg.Email(),
	}

	type requestBody struct {
		Username string `json:"username"`
		Password string `json:"password"`
		FullName string `json:"fullname"`
		Email    string `json:"email"`
	}

	testCases := []struct {
		name           string
		requestBody    requestBody
		buildStubs     func(userService *MockService, sessionMaker *MockSessionMaker)
		wantStatusCode int
		wantError      string
		checkData      func(reqBody requestBody, resp web.Response)
	}{
		{
			name: "OK",
			requestBody: requestBody{
				Username: user.Username,
				Password: user.HashedPassword,
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				createdUser := userservice.NewUserWihtoutPassword(user)
				createdUser.CreatedAt = time.Now().Truncate(time.Second)

				userService.EXPECT().
					Create(
						gomock.Any(),
						gomock.Eq(user.Username),
						gomock.Eq(user.HashedPassword),
						gomock.Eq(user.FullName),
						gomock.Eq(user.Email),
					).
					Times(1).
					Return(createdUser, nil)

				arg := domain.CreateSessionParams{
					Username: user.Username,
				}

				createdSession := domain.Session{
					RefreshToken: "RefreshToken",
					ExpiresAt:    time.Now().Add(time.Hour),
				}

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return("accessToken", time.Now().Add(time.Hour), createdSession, nil)
			},
			wantStatusCode: http.StatusCreated,
			checkData: func(reqBody requestBody, resp web.Response) {
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
					Username:  reqBody.Username,
					FullName:  reqBody.FullName,
					Email:     reqBody.Email,
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
			requestBody: requestBody{
				Username: "user&%",
				Password: user.HashedPassword,
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Username accepts only alphanumeric characters",
		},
		{
			name: "ShortPassword",
			requestBody: requestBody{
				Username: user.Username,
				Password: "xyz",
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Password must be at least 6 characters long",
		},
		{
			name: "InvalidEmail",
			requestBody: requestBody{
				Username: user.Username,
				Password: user.HashedPassword,
				FullName: user.FullName,
				Email:    "user%email.com",
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Email must contain a valid email",
		},
		{
			name: "UniqueViolationUsername",
			requestBody: requestBody{
				Username: user.Username,
				Password: user.HashedPassword,
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(user.Username),
						gomock.Eq(user.HashedPassword),
						gomock.Eq(user.FullName),
						gomock.Eq(user.Email)).
					Times(1).
					Return(domain.UserWihtoutPassword{}, domain.ErrUsernameAlreadyExists)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusConflict,
			wantError:      domain.ErrUsernameAlreadyExists.Error(),
		},
		{
			name: "UniqueViolationEmail",
			requestBody: requestBody{
				Username: user.Username,
				Password: user.HashedPassword,
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(user.Username),
						gomock.Eq(user.HashedPassword),
						gomock.Eq(user.FullName),
						gomock.Eq(user.Email)).
					Times(1).
					Return(domain.UserWihtoutPassword{}, domain.ErrEmailALreadyExists)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusConflict,
			wantError:      domain.ErrEmailALreadyExists.Error(),
		},
		{
			name: "CreateInternalError",
			requestBody: requestBody{
				Username: user.Username,
				Password: user.HashedPassword,
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(user.Username),
						gomock.Eq(user.HashedPassword),
						gomock.Eq(user.FullName),
						gomock.Eq(user.Email)).
					Times(1).
					Return(domain.UserWihtoutPassword{}, errorspkg.ErrInternal)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      errorspkg.ErrInternal.Error(),
		},
		{
			name: "CreateSessionInternalError",
			requestBody: requestBody{
				Username: user.Username,
				Password: user.HashedPassword,
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(user.Username),
						gomock.Eq(user.HashedPassword),
						gomock.Eq(user.FullName),
						gomock.Eq(user.Email)).
					Times(1).
					Return(domain.UserWihtoutPassword{}, nil)

				arg := domain.CreateSessionParams{
					Username: user.Username,
				}

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return("", time.Now(), domain.Session{}, errorspkg.ErrInternal)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      errorspkg.ErrInternal.Error(),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Initialize mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sessionMaker := NewMockSessionMaker(ctrl)
			userService := NewMockService(ctrl)
			userHandler := NewHandler(userService, sessionMaker)

			server := gin.New()
			url := "/users"
			server.POST(url, userHandler.Create)

			tc.buildStubs(userService, sessionMaker)

			// Send request
			body, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Encoding request body error: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)

			// Test response
			if got := recorder.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
			}

			resp := web.Response{
				Data: &struct {
					User domain.UserWihtoutPassword `json:"user,omitempty"`
				}{},
			}

			if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
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

func TestLoginAPI(t *testing.T) {
	user := domain.User{
		Username:       randompkg.Owner(),
		HashedPassword: randompkg.String(10),
		FullName:       randompkg.Owner(),
		Email:          randompkg.Email(),
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sessionMaker := NewMockSessionMaker(ctrl)
	userService := NewMockService(ctrl)
	userHandler := NewHandler(userService, sessionMaker)
	server := gin.New()
	url := "/users/login"
	server.POST(url, userHandler.Login)

	type requestBody struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	testCases := []struct {
		name           string
		requestBody    requestBody
		buildStubs     func(userService *MockService, sessionMaker *MockSessionMaker)
		wantStatusCode int
		wantError      string
		checkData      func(resp web.Response)
	}{
		{
			name: "OK",
			requestBody: requestBody{
				Username: user.Username,
				Password: user.HashedPassword,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(user.Username), gomock.Eq(user.HashedPassword)).
					Times(1).
					Return(userservice.NewUserWihtoutPassword(user), nil)

				arg := domain.CreateSessionParams{
					Username: user.Username,
				}

				createdSession := domain.Session{
					RefreshToken: "RefreshToken",
					ExpiresAt:    time.Now().Add(time.Hour),
				}

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return("accessToken", time.Now().Add(time.Hour), createdSession, nil)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(resp web.Response) {
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
			name: "InvalidUsernameRequest",
			requestBody: requestBody{
				Username: "invalid-%user#1",
				Password: user.HashedPassword,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Username accepts only alphanumeric characters",
		},
		{
			name: "ShortPasswordRequest",
			requestBody: requestBody{
				Username: user.Username,
				Password: "xyz",
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Password must be at least 6 characters long",
		},

		{
			name: "UserNotFound",
			requestBody: requestBody{
				Username: "NotFound",
				Password: user.HashedPassword,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return(domain.UserWihtoutPassword{}, domain.ErrUserNotFound)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      domain.ErrUserNotFound.Error(),
		},
		{
			name: "IncorrectPassword",
			requestBody: requestBody{
				Username: user.Username,
				Password: "incorrect",
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(user.Username), gomock.Eq("incorrect")).
					Times(1).
					Return(domain.UserWihtoutPassword{}, domain.ErrWrongPassword)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      domain.ErrWrongPassword.Error(),
		},
		{
			name: "CheckPasswordInternalError",
			requestBody: requestBody{
				Username: user.Username,
				Password: user.HashedPassword,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(user.Username), gomock.Eq(user.HashedPassword)).
					Times(1).
					Return(domain.UserWihtoutPassword{}, errorspkg.ErrInternal)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      errorspkg.ErrInternal.Error(),
		},
		{
			name: "CreateSessionInternalError",
			requestBody: requestBody{
				Username: user.Username,
				Password: user.HashedPassword,
			},
			buildStubs: func(userService *MockService, sessionMaker *MockSessionMaker) {
				userService.EXPECT().
					CheckPassword(gomock.Any(), gomock.Eq(user.Username), gomock.Eq(user.HashedPassword)).
					Times(1).
					Return(userservice.NewUserWihtoutPassword(user), nil)

				sessionMaker.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(1).
					Return("", time.Now(), domain.Session{}, errorspkg.ErrInternal)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      errorspkg.ErrInternal.Error(),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Initialize mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sessionMaker := NewMockSessionMaker(ctrl)
			userService := NewMockService(ctrl)
			userHandler := NewHandler(userService, sessionMaker)

			server := gin.New()
			url := "/users/login"
			server.POST(url, userHandler.Login)

			tc.buildStubs(userService, sessionMaker)

			// Send request
			body, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Encoding request body error: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)

			// Test response
			if got := recorder.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
			}

			resp := web.Response{
				Data: &struct {
					User domain.UserWihtoutPassword `json:"user,omitempty"`
				}{},
			}

			if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
				t.Errorf("Decoding response body error: %v", err)
			}

			if tc.wantStatusCode != http.StatusOK {
				if resp.Error != tc.wantError {
					t.Errorf(`resp.Error = %q, want %q`, resp.Error, tc.wantError)
				}
			} else {
				tc.checkData(resp)
			}
		})
	}
}
