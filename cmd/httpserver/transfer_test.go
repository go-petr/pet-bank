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
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/pkg/currencypkg"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/go-petr/pet-bank/pkg/web"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCreateTranferAPI(t *testing.T) {
	server := integrationtest.SetupServer(t)

	user1 := helpers.SeedUser(t, server.DB)
	user2 := helpers.SeedUser(t, server.DB)
	account1 := helpers.SeedAccountWith1000USDBalance(t, server.DB, user1.Username)
	account2 := helpers.SeedAccountWith1000USDBalance(t, server.DB, user2.Username)
	account3 := helpers.SeedAccountWith1000Balance(t, server.DB, user2.Username, currencypkg.EUR)
	amount := "100"

	tokenMaker, err := tokenpkg.NewPasetoMaker(server.Config.TokenSymmetricKey)
	if err != nil {
		t.Fatalf("tokenpkg.NewPasetoMaker(%v) returned error: %v", server.Config.TokenSymmetricKey, err)
	}

	authType := middleware.AuthTypeBearer
	duration := server.Config.AccessTokenDuration

	type requestBody struct {
		FromAccountID int32  `json:"from_account_id" binding:"required,min=1"`
		ToAccountID   int32  `json:"to_account_id" binding:"required,min=1"`
		Amount        string `json:"amount" binding:"required"`
	}

	testCases := []struct {
		name           string
		requestBody    requestBody
		setupAuth      func(t *testing.T, r *http.Request) error
		wantStatusCode int
		checkData      func(req requestBody, data any)
		wantError      string
	}{
		{
			name: "OK",
			requestBody: requestBody{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, user1.Username, duration)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(req requestBody, data any) {
				got, ok := data.(*struct {
					Transfer domain.TransferTxResult `json:"transfer"`
				})
				if !ok {
					t.Errorf(`res.Data=%#v, failed type conversion`, data)
				}

				want := domain.TransferTxResult{
					Transfer: domain.Transfer{
						FromAccountID: req.FromAccountID,
						ToAccountID:   req.ToAccountID,
						Amount:        req.Amount,
						CreatedAt:     time.Now().UTC().Truncate(time.Second),
					},
					FromAccount: domain.Account{
						Owner:     account1.Owner,
						Balance:   "900",
						Currency:  account1.Currency,
						CreatedAt: account1.CreatedAt,
					},
					ToAccount: domain.Account{
						Owner:     account2.Owner,
						Balance:   "1100",
						Currency:  account2.Currency,
						CreatedAt: account2.CreatedAt,
					},
					FromEntry: domain.Entry{
						AccountID: account1.ID,
						Amount:    "-" + amount,
						CreatedAt: time.Now().UTC().Truncate(time.Second),
					},
					ToEntry: domain.Entry{
						AccountID: account2.ID,
						Amount:    amount,
						CreatedAt: time.Now().UTC().Truncate(time.Second),
					},
				}

				ignoreAccountID := cmpopts.IgnoreFields(domain.Account{}, "ID")
				ignoreTransferID := cmpopts.IgnoreFields(domain.Transfer{}, "ID")
				ignoreEntryID := cmpopts.IgnoreFields(domain.Entry{}, "ID")

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, got.Transfer, ignoreTransferID, ignoreAccountID, ignoreEntryID, compareCreatedAt); diff != "" {
					t.Errorf("res.Data mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "RequiredFromAccountID",
			requestBody: requestBody{
				FromAccountID: 0,
				ToAccountID:   account2.ID,
				Amount:        amount,
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, user1.Username, duration)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "FromAccountID field is required",
		},
		{
			name: "RequiredToAccountID",
			requestBody: requestBody{
				FromAccountID: account1.ID,
				ToAccountID:   0,
				Amount:        amount,
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, user1.Username, duration)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "ToAccountID field is required",
		},
		{
			name: "RequiredAmount",
			requestBody: requestBody{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        "",
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, user1.Username, duration)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Amount field is required",
		},
		{
			name: "UnauthorizedOwner",
			requestBody: requestBody{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, user2.Username, duration)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      domain.ErrInvalidOwner.Error(),
		},
		{
			name: "ErrCurrencyMismatch",
			requestBody: requestBody{
				FromAccountID: account1.ID,
				ToAccountID:   account3.ID,
				Amount:        amount,
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, user1.Username, duration)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      domain.ErrCurrencyMismatch.Error(),
		},
		{
			name: "NoAuthorization",
			requestBody: requestBody{
				FromAccountID: account1.ID,
				ToAccountID:   account3.ID,
				Amount:        amount,
			},
			setupAuth: func(t *testing.T, r *http.Request) error {
				return nil
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      middleware.ErrAuthHeaderNotFound.Error(),
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

			req, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(body))
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
					Transfer domain.TransferTxResult `json:"transfer"`
				}{},
			}

			if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
				t.Errorf("Decoding response body error: %v", err)
			}

			if tc.wantStatusCode != http.StatusOK {
				if res.Error != tc.wantError {
					t.Errorf(`res.Error=%q, want %q`, res.Error, tc.wantError)
				}
			} else {
				tc.checkData(tc.requestBody, res.Data)
			}
		})
	}
}
