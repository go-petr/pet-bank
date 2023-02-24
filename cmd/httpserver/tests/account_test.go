//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
					ID:        1,
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
