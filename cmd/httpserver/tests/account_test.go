//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/internal/test"
	"github.com/go-petr/pet-bank/pkg/currencypkg"
	"github.com/go-petr/pet-bank/pkg/dbpkg/integrationtest"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/go-petr/pet-bank/pkg/web"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
)

func TestCreateAccountAPI(t *testing.T) {
	defer integrationtest.Flush(t, server.DB)

	user := test.SeedUser(t, server.DB)
	test.SeedAccountWith1000USDBalance(t, server.DB, user.Username)
	tokenMaker, err := tokenpkg.NewPasetoMaker(server.Config.TokenSymmetricKey)
	require.NoError(t, err)

	type requestBody struct {
		Currency string `json:"currency"`
	}

	testCases := []struct {
		name           string
		requestBody    requestBody
		setupAuth      func(t *testing.T, r *http.Request) error
		wantStatusCode int
		checkData      func(req requestBody, res web.Response)
		wantError      string
	}{
		{
			name:        "OK",
			requestBody: requestBody{Currency: currencypkg.EUR},
			setupAuth: func(t *testing.T, r *http.Request) error {
				authType := middleware.AuthTypeBearer
				d := server.Config.AccessTokenDuration
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, d)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(req requestBody, res web.Response) {
				gotData, ok := res.Data.(*struct {
					Account domain.Account `json:"account"`
				})
				if !ok {
					t.Errorf(`res.Data=%v, failed type conversion`, res.Data)
				}

				want := domain.Account{
					Owner:     user.Username,
					Balance:   "0",
					Currency:  currencypkg.EUR,
					CreatedAt: time.Now().UTC().Truncate(time.Second),
				}

				ignoreFields := cmpopts.IgnoreFields(domain.Account{}, "ID")
				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, gotData.Account, ignoreFields, compareCreatedAt); diff != "" {
					t.Errorf("res.Data mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:        "NoAuthorization",
			requestBody: requestBody{Currency: currencypkg.USD},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return nil
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      middleware.ErrAuthHeaderNotFound.Error(),
		},
		{
			name:        "InvalidCurrency",
			requestBody: requestBody{Currency: "FAIL"},
			setupAuth: func(t *testing.T, r *http.Request) error {
				authType := middleware.AuthTypeBearer
				d := server.Config.AccessTokenDuration
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, d)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Currency is not supported",
		},
		{
			name:        "ErrOwnerNotFound",
			requestBody: requestBody{Currency: currencypkg.EUR},
			setupAuth: func(t *testing.T, r *http.Request) error {
				authType := middleware.AuthTypeBearer
				d := server.Config.AccessTokenDuration
				return middleware.AddAuthorization(r, tokenMaker, authType, "username", d)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      domain.ErrOwnerNotFound.Error(),
		},
		{
			name:        "ErrCurrencyAlreadyExists",
			requestBody: requestBody{Currency: currencypkg.USD},
			setupAuth: func(t *testing.T, r *http.Request) error {
				authType := middleware.AuthTypeBearer
				d := server.Config.AccessTokenDuration
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, d)
			},
			wantStatusCode: http.StatusConflict,
			wantError:      domain.ErrCurrencyAlreadyExists.Error(),
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

			req, err := http.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			if err = tc.setupAuth(t, req); err != nil {
				t.Fatalf("tc.setupAuth(t, %+v) returned error: %v", req, err)
			}

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			// Test response
			if got := w.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
			}

			res := web.Response{
				Data: &struct {
					Account domain.Account `json:"account"`
				}{},
			}

			if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
				t.Errorf("Decoding response body error: %v", err)
			}

			if tc.wantStatusCode != http.StatusOK {
				if res.Error != tc.wantError {
					t.Errorf(`resp.Error=%q, want %q`, res.Error, tc.wantError)
				}
			} else {
				tc.checkData(tc.requestBody, res)
			}
		})
	}
}

func TestGetAccountAPI(t *testing.T) {
	defer integrationtest.Flush(t, server.DB)

	user := test.SeedUser(t, server.DB)
	account := test.SeedAccountWith1000USDBalance(t, server.DB, user.Username)
	user2 := test.SeedUser(t, server.DB)
	account2 := test.SeedAccountWith1000USDBalance(t, server.DB, user2.Username)
	tokenMaker, err := tokenpkg.NewPasetoMaker(server.Config.TokenSymmetricKey)
	require.NoError(t, err)

	type requestBody struct {
		ID int32 `json:"id"`
	}

	testCases := []struct {
		name           string
		accountID      int32
		setupAuth      func(t *testing.T, r *http.Request) error
		wantStatusCode int
		checkData      func(res web.Response)
		wantError      string
	}{
		{
			name:      "OK",
			accountID: account.ID,
			setupAuth: func(t *testing.T, r *http.Request) error {
				authType := middleware.AuthTypeBearer
				d := server.Config.AccessTokenDuration
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, d)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(res web.Response) {
				gotData, ok := res.Data.(*struct {
					Account domain.Account `json:"account"`
				})
				if !ok {
					t.Errorf(`res.Data=%v, failed type conversion`, res.Data)
				}

				want := account

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, gotData.Account, compareCreatedAt); diff != "" {
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
			wantStatusCode: http.StatusUnauthorized,
			wantError:      middleware.ErrAuthHeaderNotFound.Error(),
		},
		{
			name:      "InvalidID",
			accountID: -1,
			setupAuth: func(t *testing.T, r *http.Request) error {
				authType := middleware.AuthTypeBearer
				d := server.Config.AccessTokenDuration
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, d)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "ID must be at least 1 characters long",
		},
		{
			name:      "ErrAccountNotFound",
			accountID: 1200000000,
			setupAuth: func(t *testing.T, r *http.Request) error {
				authType := middleware.AuthTypeBearer
				d := server.Config.AccessTokenDuration
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, d)
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      domain.ErrAccountNotFound.Error(),
		},
		{
			name:      "ErrAccountOwnerMismatch",
			accountID: account2.ID,
			setupAuth: func(t *testing.T, r *http.Request) error {
				authType := middleware.AuthTypeBearer
				d := server.Config.AccessTokenDuration
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, d)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      domain.ErrAccountOwnerMismatch.Error(),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {

			// Send request
			req, err := http.NewRequest(http.MethodGet, "/accounts/"+strconv.Itoa(int(tc.accountID)), nil)
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			if err = tc.setupAuth(t, req); err != nil {
				t.Fatalf("tc.setupAuth(t, %+v) returned error: %v", req, err)
			}

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			// Test response
			if got := w.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
			}

			res := web.Response{
				Data: &struct {
					Account domain.Account `json:"account"`
				}{},
			}

			if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
				t.Errorf("Decoding response body error: %v", err)
			}

			if tc.wantStatusCode != http.StatusOK {
				if res.Error != tc.wantError {
					t.Errorf(`resp.Error=%q, want %q`, res.Error, tc.wantError)
				}
			} else {
				tc.checkData(res)
			}
		})
	}
}

func TestListAccountAPI(t *testing.T) {
	defer integrationtest.Flush(t, server.DB)

	user := test.SeedUser(t, server.DB)
	accounts := test.SeedAllCurrenciesAccountsWith1000Balance(t, server.DB, user.Username)
	tokenMaker, err := tokenpkg.NewPasetoMaker(server.Config.TokenSymmetricKey)
	if err != nil {
		t.Fatalf("tokenpkg.NewPasetoMaker(%v) returned error: %v", server.Config.TokenSymmetricKey, err)
	}

	type requestBody struct {
		PageID   int32 `form:"page_id" binding:"required,min=1"`
		PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
	}

	authType := middleware.AuthTypeBearer
	duration := server.Config.AccessTokenDuration

	testCases := []struct {
		name           string
		requestBody    requestBody
		setupAuth      func(t *testing.T, r *http.Request) error
		wantStatusCode int
		checkData      func(res web.Response)
		wantError      string
	}{
		{
			name:        "OK",
			requestBody: requestBody{PageID: 1, PageSize: 5},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, duration)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(res web.Response) {
				gotData, ok := res.Data.(*struct {
					Accounts []domain.Account `json:"accounts"`
				})
				if !ok {
					t.Errorf(`res.Data=%v, failed type conversion`, res.Data)
				}

				want := accounts

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, gotData.Accounts, compareCreatedAt); diff != "" {
					t.Errorf("res.Data mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:        "PageSize:2",
			requestBody: requestBody{PageID: 1, PageSize: 2},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, duration)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(res web.Response) {
				gotData, ok := res.Data.(*struct {
					Accounts []domain.Account `json:"accounts"`
				})
				if !ok {
					t.Errorf(`res.Data=%v, failed type conversion`, res.Data)
				}

				want := accounts[:2]

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, gotData.Accounts, compareCreatedAt); diff != "" {
					t.Errorf("res.Data mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:        "PageID:2PageSize:2",
			requestBody: requestBody{PageID: 2, PageSize: 2},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, duration)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(res web.Response) {
				gotData, ok := res.Data.(*struct {
					Accounts []domain.Account `json:"accounts"`
				})
				if !ok {
					t.Errorf(`res.Data=%v, failed type conversion`, res.Data)
				}

				want := accounts[2:3]

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, gotData.Accounts, compareCreatedAt); diff != "" {
					t.Errorf("res.Data mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:        "NoAuthorization",
			requestBody: requestBody{PageID: 1, PageSize: 5},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return nil
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      middleware.ErrAuthHeaderNotFound.Error(),
		},
		{
			name:        "InvalidPageID",
			requestBody: requestBody{PageID: 0, PageSize: 5},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, duration)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "PageID field is required",
		},
		{
			name:        "ExceededPageSize",
			requestBody: requestBody{PageID: 1, PageSize: 500},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, user.Username, duration)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "PageSize must be less than 100",
		},
		{
			name:        "NoAccountsForGivenOwner",
			requestBody: requestBody{PageID: 1, PageSize: 50},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, "user.Username", duration)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(res web.Response) {
				gotData, ok := res.Data.(*struct {
					Accounts []domain.Account `json:"accounts"`
				})
				if !ok {
					t.Errorf(`res.Data=%v, failed type conversion`, res.Data)
				}

				want := []domain.Account{}

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, gotData.Accounts, compareCreatedAt); diff != "" {
					t.Errorf("res.Data mismatch (-want +got):\n%s", diff)
				}
			},
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

			req, err := http.NewRequest(http.MethodGet, "/accounts", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			if err = tc.setupAuth(t, req); err != nil {
				t.Fatalf("tc.setupAuth(t, %+v) returned error: %v", req, err)
			}

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			// Test response
			res := web.Response{
				Data: &struct {
					Accounts []domain.Account `json:"accounts"`
				}{},
			}

			if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
				t.Errorf("Decoding response body error: %v", err)
			}

			if got := w.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
				t.Errorf(`res.Error=%q, want %q`, res.Error, tc.wantError)
			}

			if tc.wantStatusCode != http.StatusOK {
				if res.Error != tc.wantError {
					t.Errorf(`resp.Error=%q, want %q`, res.Error, tc.wantError)
				}
			} else {
				tc.checkData(res)
			}
		})
	}
}
