package accountdelivery

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/integrationtest/helpers"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/pkg/currencypkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/go-petr/pet-bank/pkg/web"
	"github.com/golang/mock/gomock"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestCreate(t *testing.T) {
	username := randompkg.Owner()
	account := helpers.RandomAccount(username)
	tokenSymmetricKey := randompkg.String(32)

	tokenMaker, err := tokenpkg.NewPasetoMaker(tokenSymmetricKey)
	if err != nil {
		t.Fatalf("tokenpkg.NewPasetoMaker(%v) returned error: %v", tokenSymmetricKey, err)
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("currency", currencypkg.ValidCurrency); err != nil {
			t.Fatalf("tokenpkg.NewPasetoMaker(%v) returned error: %v", tokenSymmetricKey, err)
		}
	}

	authType := middleware.AuthTypeBearer
	duration := time.Minute

	type requestBody struct {
		Currency string `json:"currency"`
	}

	testCases := []struct {
		name           string
		requestBody    requestBody
		setupAuth      func(t *testing.T, r *http.Request) error
		buildStubs     func(accountService *MockService)
		wantStatusCode int
		wantError      string
		checkData      func(req requestBody, data any)
	}{
		{
			name: "OK",
			requestBody: requestBody{
				Currency: account.Currency,
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(), gomock.Eq(account.Owner), gomock.Eq(account.Currency)).
					Times(1).
					Return(account, nil)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(req requestBody, data any) {
				got, ok := data.(*struct {
					Account domain.Account `json:"account"`
				})
				if !ok {
					t.Errorf(`res.Data=%v, failed type conversion`, data)
				}

				want := account

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, got.Account, compareCreatedAt); diff != "" {
					t.Errorf("res.Data mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "NoAuthorization",
			requestBody: requestBody{
				Currency: account.Currency,
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return nil
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      middleware.ErrAuthHeaderNotFound.Error(),
		},
		{
			name: "InvalidCurrency",
			requestBody: requestBody{
				Currency: "RUB",
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Currency is not supported",
		},
		{
			name: "ErrOwnerNotFound",
			requestBody: requestBody{
				Currency: account.Currency,
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(account.Owner),
						gomock.Eq(account.Currency)).
					Times(1).
					Return(domain.Account{}, domain.ErrOwnerNotFound)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      domain.ErrOwnerNotFound.Error(),
		},
		{
			name: "ErrCurrencyAlreadyExists",
			requestBody: requestBody{
				Currency: account.Currency,
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(account.Owner),
						gomock.Eq(account.Currency)).
					Times(1).
					Return(domain.Account{}, domain.ErrCurrencyAlreadyExists)
			},
			wantStatusCode: http.StatusConflict,
			wantError:      domain.ErrCurrencyAlreadyExists.Error(),
		},
		{
			name: "InternalServerError",
			requestBody: requestBody{
				Currency: account.Currency,
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Create(gomock.Any(),
						gomock.Eq(account.Owner),
						gomock.Eq(account.Currency)).
					Times(1).
					Return(domain.Account{}, errorspkg.ErrInternal)
			},
			checkData: func(req requestBody, data any) {
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
			accountService := NewMockService(ctrl)
			accountHandler := NewHandler(accountService)

			server := gin.New()
			server.Use(middleware.AuthMiddleware(tokenMaker))
			server.POST("/accounts", accountHandler.Create)

			tc.buildStubs(accountService)

			// Send request
			body, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Encoding request body error: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			if err = tc.setupAuth(t, req); err != nil {
				t.Fatalf("tc.setupAuth(t, %+v) returned error: %v", req, err)
			}

			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)

			// Test response
			if got := recorder.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
			}

			res := web.Response{
				Data: &struct {
					Account domain.Account `json:"account"`
				}{},
			}

			if err := json.NewDecoder(recorder.Body).Decode(&res); err != nil {
				t.Errorf("Decoding response body error: %v", err)
			}

			if tc.wantStatusCode != http.StatusOK {
				if res.Error != tc.wantError {
					t.Errorf(`resp.Error=%q, want %q`, res.Error, tc.wantError)
				}
			} else {
				tc.checkData(tc.requestBody, res.Data)
			}
		})
	}
}

func TestGet(t *testing.T) {
	username := randompkg.Owner()
	account := helpers.RandomAccount(username)
	username2 := randompkg.Owner()
	account2 := helpers.RandomAccount(username2)
	tokenSymmetricKey := randompkg.String(32)

	tokenMaker, err := tokenpkg.NewPasetoMaker(tokenSymmetricKey)
	if err != nil {
		t.Fatalf("tokenpkg.NewPasetoMaker(%v) returned error: %v", tokenSymmetricKey, err)
	}

	authType := middleware.AuthTypeBearer
	duration := time.Minute

	testCases := []struct {
		name           string
		accountID      int32
		setupAuth      func(t *testing.T, r *http.Request) error
		buildStubs     func(accountService *MockService)
		wantStatusCode int
		wantError      string
		checkData      func(data any)
	}{
		{
			name:      "OK",
			accountID: account.ID,
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(data any) {
				got, ok := data.(*struct {
					Account domain.Account `json:"account"`
				})
				if !ok {
					t.Errorf(`res.Data=%v, failed type conversion`, data)
				}

				want := account

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, got.Account, compareCreatedAt); diff != "" {
					t.Errorf("res.Data mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:      "NoAuthorization",
			accountID: account.ID,
			setupAuth: func(t *testing.T, r *http.Request) error {
				return nil
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      middleware.ErrAuthHeaderNotFound.Error(),
		},
		{
			name:      "InvalidID",
			accountID: -1,
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "ID must be at least 1 characters long",
		},
		{
			name:      "ErrAccountNotFound",
			accountID: account.ID,
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(domain.Account{}, domain.ErrAccountNotFound)
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      domain.ErrAccountNotFound.Error(),
		},
		{
			name:      "InternalError",
			accountID: account.ID,
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(domain.Account{}, sql.ErrConnDone)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      errorspkg.ErrInternal.Error(),
		},
		{
			name:      "UnauthorizedUser",
			accountID: account2.ID,
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService) {
				accountService.EXPECT().
					Get(gomock.Any(), gomock.Eq(account2.ID)).
					Times(1).
					Return(account2, nil)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      domain.ErrAccountOwnerMismatch.Error(),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Initialize mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			accountService := NewMockService(ctrl)
			accountHandler := NewHandler(accountService)

			server := gin.New()
			server.Use(middleware.AuthMiddleware(tokenMaker))
			server.GET("/accounts/:id", accountHandler.Get)

			tc.buildStubs(accountService)

			// Send request
			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			if err = tc.setupAuth(t, req); err != nil {
				t.Fatalf("tc.setupAuth(t, %+v) returned error: %v", req, err)
			}

			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)

			// Test response
			if got := recorder.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
			}

			res := web.Response{
				Data: &struct {
					Account domain.Account `json:"account"`
				}{},
			}

			if err := json.NewDecoder(recorder.Body).Decode(&res); err != nil {
				t.Fatalf("Decoding response body error: %v", err)
			}

			if tc.wantStatusCode != http.StatusOK {
				if res.Error != tc.wantError {
					t.Errorf(`resp.Error=%q, want %q`, res.Error, tc.wantError)
				}
			} else {
				tc.checkData(res.Data)
			}
		})
	}
}

func TestList(t *testing.T) {
	username := randompkg.Owner()
	tokenSymmetricKey := randompkg.String(32)

	tokenMaker, err := tokenpkg.NewPasetoMaker(tokenSymmetricKey)
	if err != nil {
		t.Fatalf("tokenpkg.NewPasetoMaker(%v) returned error: %v", tokenSymmetricKey, err)
	}

	authType := middleware.AuthTypeBearer
	duration := time.Minute

	// n specifies number of account in DB
	n := 10
	accounts := make([]domain.Account, n)

	for i := 0; i < n; i++ {
		accounts[i] = helpers.RandomAccount(username)
	}

	testCases := []struct {
		name           string
		pageID         int32
		pageSize       int32
		setupAuth      func(t *testing.T, r *http.Request) error
		buildStubs     func(accountService *MockService, pageID, pageSize int32)
		wantStatusCode int
		checkData      func(data any)
		wantError      string
	}{
		{
			name:     "OK",
			pageID:   1,
			pageSize: 10,
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService, pageID, pageSize int32) {
				accountService.EXPECT().
					List(context.Background(), username, pageID, pageSize).
					Times(1).
					Return(accounts, nil)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(data any) {
				got := data.(*struct {
					Accounts []domain.Account `json:"accounts"`
				})

				want := accounts

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, got.Accounts, compareCreatedAt); diff != "" {
					t.Errorf("res.Data mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:     "NoAuthorization",
			pageID:   1,
			pageSize: 5,
			setupAuth: func(t *testing.T, r *http.Request) error {
				return nil
			},
			buildStubs: func(accountService *MockService, pageID, pageSize int32) {
				accountService.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      middleware.ErrAuthHeaderNotFound.Error(),
		},
		{
			name:     "InvalidPageID",
			pageID:   0,
			pageSize: 5,
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService, pageID, pageSize int32) {
				accountService.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "PageID field is required",
		},
		{
			name:     "ExceededPageSize",
			pageID:   1,
			pageSize: 500,
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService, pageID, pageSize int32) {
				accountService.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "PageSize must be less than 100",
		},
		{
			name:     "InternalError",
			pageID:   1,
			pageSize: 5,
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username, duration)
			},
			buildStubs: func(accountService *MockService, pageID, pageSize int32) {
				accountService.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return([]domain.Account{}, errorspkg.ErrInternal)
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
			accountService := NewMockService(ctrl)
			accountHandler := NewHandler(accountService)

			server := gin.New()
			server.Use(middleware.AuthMiddleware(tokenMaker))
			server.GET("/accounts", accountHandler.List)

			tc.buildStubs(accountService, tc.pageID, tc.pageSize)

			// Send request
			url := fmt.Sprintf("/accounts?page_id=%v&page_size=%v", tc.pageID, tc.pageSize)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			if err = tc.setupAuth(t, req); err != nil {
				t.Fatalf("tc.setupAuth(t, %+v) returned error: %v", req, err)
			}

			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)

			// Test response
			if got := recorder.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
			}

			res := web.Response{
				Data: &struct {
					Accounts []domain.Account `json:"accounts"`
				}{},
			}

			if err := json.NewDecoder(recorder.Body).Decode(&res); err != nil {
				t.Fatalf("Decoding response body error: %v", err)
			}

			if tc.wantStatusCode != http.StatusOK {
				if res.Error != tc.wantError {
					t.Errorf(`resp.Error=%q, want %q`, res.Error, tc.wantError)
				}
			} else {
				tc.checkData(res.Data)
			}
		})
	}
}
