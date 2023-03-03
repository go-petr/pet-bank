package transferdelivery

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/internal/integrationtest/helpers"
	"github.com/go-petr/pet-bank/internal/middleware"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/go-petr/pet-bank/pkg/web"
	"github.com/golang/mock/gomock"
)

func TestCreate(t *testing.T) {
	username1 := randompkg.Owner()
	username2 := randompkg.Owner()
	account1 := helpers.RandomAccount(username1)
	account2 := helpers.RandomAccount(username2)
	amount := "100"
	symmetricKey := randompkg.String(32)

	tokenMaker, err := tokenpkg.NewPasetoMaker(symmetricKey)
	if err != nil {
		t.Fatalf("tokenpkg.NewPasetoMaker(%v) returned error: %v", symmetricKey, err)
	}

	authType := middleware.AuthTypeBearer
	duration := time.Minute

	type requestBody struct {
		FromAccountID int32  `json:"from_account_id" binding:"required,min=1"`
		ToAccountID   int32  `json:"to_account_id" binding:"required,min=1"`
		Amount        string `json:"amount" binding:"required"`
	}

	want := domain.TransferTxResult{
		Transfer: domain.Transfer{
			FromAccountID: account1.ID,
			ToAccountID:   account2.ID,
			Amount:        amount,
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

	testCases := []struct {
		name           string
		requestBody    requestBody
		setupAuth      func(r *http.Request) error
		buildStubs     func(transferService *MockService)
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
			setupAuth: func(r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username1, duration)
			},
			buildStubs: func(transferService *MockService) {
				arg := domain.CreateTransferParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					Transfer(gomock.Any(), gomock.Eq(username1), gomock.Eq(arg)).
					Times(1).
					Return(want, nil)
			},
			wantStatusCode: http.StatusCreated,
			checkData: func(req requestBody, data any) {
				got, ok := data.(*struct {
					Transfer domain.TransferTxResult `json:"transfer"`
				})
				if !ok {
					t.Errorf(`res.Data=%#v, failed type conversion`, data)
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
			name: "NoAuthorization",
			requestBody: requestBody{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			},
			setupAuth: func(r *http.Request) error {
				return nil
			},
			buildStubs: func(transferService *MockService) {
				transferService.EXPECT().Transfer(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      middleware.ErrAuthHeaderNotFound.Error(),
		},
		{
			name: "RequiredFromAccountID",
			requestBody: requestBody{
				FromAccountID: 0,
				ToAccountID:   account2.ID,
				Amount:        amount,
			},
			setupAuth: func(r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username1, duration)
			},
			buildStubs: func(transferService *MockService) {
				transferService.EXPECT().Transfer(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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
			setupAuth: func(r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username1, duration)
			},
			buildStubs: func(transferService *MockService) {
				transferService.EXPECT().Transfer(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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
			setupAuth: func(r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username1, duration)
			},
			buildStubs: func(transferService *MockService) {
				transferService.EXPECT().Transfer(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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
			setupAuth: func(r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username2, duration)
			},
			buildStubs: func(transferService *MockService) {
				arg := domain.CreateTransferParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					Transfer(gomock.Any(), gomock.Eq(username2), gomock.Eq(arg)).
					Times(1).
					Return(domain.TransferTxResult{}, domain.ErrInvalidOwner)
			},
			wantStatusCode: http.StatusUnauthorized,
			wantError:      domain.ErrInvalidOwner.Error(),
		},
		{
			name: "ErrCurrencyMismatch",
			requestBody: requestBody{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			},
			setupAuth: func(r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username1, duration)
			},
			buildStubs: func(transferService *MockService) {
				arg := domain.CreateTransferParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					Transfer(gomock.Any(), gomock.Eq(username1), gomock.Eq(arg)).
					Times(1).
					Return(domain.TransferTxResult{}, domain.ErrCurrencyMismatch)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      domain.ErrCurrencyMismatch.Error(),
		},
		{
			name: "InvalidTransferInternalError",
			requestBody: requestBody{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			},
			setupAuth: func(r *http.Request) error {
				return middleware.AddAuthorization(r, tokenMaker, authType, username1, duration)
			},
			buildStubs: func(transferService *MockService) {
				arg := domain.CreateTransferParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				}

				transferService.EXPECT().
					Transfer(gomock.Any(), gomock.Eq(username1), gomock.Eq(arg)).
					Times(1).
					Return(domain.TransferTxResult{}, errorspkg.ErrInternal)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      errorspkg.ErrInternal.Error(),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Set up mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			transferService := NewMockService(ctrl)
			transferHandler := NewHandler(transferService)

			gin.SetMode(gin.ReleaseMode)
			server := gin.New()
			url := "/transfers"

			server.Use(middleware.AuthMiddleware(tokenMaker))
			server.POST(url, transferHandler.Create)

			tc.buildStubs(transferService)

			// Send request
			body, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Encoding request body error: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			if err = tc.setupAuth(req); err != nil {
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
