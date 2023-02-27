package transferservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/internal/accountdelivery"
	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/currencypkg"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

func randomAccount(id int32, balance, currency string) domain.Account {
	return domain.Account{
		ID:        id,
		Owner:     randompkg.Owner(),
		Balance:   balance,
		Currency:  currency,
		CreatedAt: time.Now().Truncate(time.Second).UTC(),
	}
}

func TestTransfer(t *testing.T) {
	accountUSD1 := randomAccount(1, "1000", currencypkg.USD)
	accountUSD2 := randomAccount(2, "1000", currencypkg.USD)
	accountEUR3 := randomAccount(1, "1000", currencypkg.EUR)
	amount := "100"

	want := domain.TransferTxResult{
		Transfer: domain.Transfer{
			FromAccountID: accountUSD1.ID,
			ToAccountID:   accountUSD2.ID,
			Amount:        amount,
		},
		FromAccount: accountUSD1,
		ToAccount:   accountUSD2,
		FromEntry:   domain.Entry{AccountID: accountUSD1.ID, Amount: "-" + amount},
		ToEntry:     domain.Entry{AccountID: accountUSD2.ID, Amount: amount},
	}

	type input struct {
		fromUsername string
		arg          domain.CreateTransferParams
	}

	testCases := []struct {
		name          string
		input         input
		buildStubs    func(repo *MockRepo, accountService *accountdelivery.MockService)
		checkResponse func(t *testing.T, res domain.TransferTxResult)
		wantError     string
	}{
		{
			name: "OK",
			input: input{
				fromUsername: accountUSD1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: accountUSD1.ID,
					ToAccountID:   accountUSD2.ID,
					Amount:        amount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).
					Times(1).
					Return(want, nil)

				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountUSD1.ID)).
					Times(1).
					Return(domain.Account{
						ID:       accountUSD1.ID,
						Owner:    accountUSD1.Owner,
						Balance:  accountUSD1.Balance,
						Currency: accountUSD1.Currency,
					}, nil)

				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountUSD2.ID)).
					Times(1).
					Return(domain.Account{
						Currency: accountUSD2.Currency,
					}, nil)
			},
			checkResponse: func(t *testing.T, got domain.TransferTxResult) {
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("Response returned unexpected diff: %s", diff)
				}
			},
		},
		{
			name: "ErrInvalidAmount",
			input: input{
				fromUsername: accountUSD1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: accountUSD1.ID,
					ToAccountID:   accountUSD2.ID,
					Amount:        "!@#$",
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)
			},
			wantError: domain.ErrInvalidAmount.Error(),
		},
		{
			name: "ErrNegativeAmount",
			input: input{
				fromUsername: accountUSD1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: accountUSD1.ID,
					ToAccountID:   accountUSD2.ID,
					Amount:        "-100",
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)
				accountService.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)
			},
			wantError: domain.ErrNegativeAmount.Error(),
		},
		{
			name: "AccountServiceInternalError",
			input: input{
				fromUsername: accountUSD1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: accountUSD1.ID,
					ToAccountID:   accountUSD2.ID,
					Amount:        amount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)

				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountUSD1.ID)).
					Times(1).
					Return(domain.Account{}, errorspkg.ErrInternal)
			},
			wantError: errorspkg.ErrInternal.Error(),
		},
		{
			name: "ErrInvalidOwner",
			input: input{
				fromUsername: accountUSD1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: accountUSD2.ID,
					ToAccountID:   accountUSD1.ID,
					Amount:        amount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)

				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountUSD2.ID)).
					Times(1).
					Return(domain.Account{
						Owner: accountUSD2.Owner,
					}, nil)
			},
			wantError: domain.ErrInvalidOwner.Error(),
		},
		{
			name: "FromAccountServiceInternalError",
			input: input{
				fromUsername: accountUSD1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: accountUSD1.ID,
					ToAccountID:   accountUSD2.ID,
					Amount:        amount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)

				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountUSD1.ID)).
					Times(1).
					Return(domain.Account{
						Owner:   accountUSD1.Owner,
						Balance: "invalid",
					}, nil)
			},
			wantError: fmt.Errorf("can't convert %s to decimal", "invalid").Error(),
		},
		{
			name: "ErrInsufficientBalance",
			input: input{
				fromUsername: accountUSD1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: accountUSD1.ID,
					ToAccountID:   accountUSD2.ID,
					Amount:        "10000",
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)

				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountUSD1.ID)).
					Times(1).
					Return(domain.Account{
						ID:       accountUSD1.ID,
						Owner:    accountUSD1.Owner,
						Balance:  accountUSD1.Balance,
						Currency: accountUSD1.Currency,
					}, nil)
			},
			wantError: domain.ErrInsufficientBalance.Error(),
		},
		{
			name: "ToAccountServiceInternalError",
			input: input{
				fromUsername: accountUSD1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: accountUSD1.ID,
					ToAccountID:   accountUSD2.ID,
					Amount:        amount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)

				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountUSD1.ID)).
					Times(1).
					Return(domain.Account{
						ID:       accountUSD1.ID,
						Owner:    accountUSD1.Owner,
						Balance:  accountUSD1.Balance,
						Currency: accountUSD1.Currency,
					}, nil)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountUSD2.ID)).
					Times(1).
					Return(domain.Account{}, errorspkg.ErrInternal)
			},
			wantError: errorspkg.ErrInternal.Error(),
		},
		{
			name: "ErrCurrencyMismatch",
			input: input{
				fromUsername: accountUSD1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: accountUSD1.ID,
					ToAccountID:   accountEUR3.ID,
					Amount:        amount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).Times(0)

				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountUSD1.ID)).
					Times(1).
					Return(domain.Account{
						ID:       accountUSD1.ID,
						Owner:    accountUSD1.Owner,
						Balance:  accountUSD1.Balance,
						Currency: accountUSD1.Currency,
					}, nil)
				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountEUR3.ID)).
					Times(1).
					Return(domain.Account{
						Currency: accountEUR3.Currency,
					}, nil)
			},
			wantError: domain.ErrCurrencyMismatch.Error(),
		},
		{
			name: "RepoInternalError",
			input: input{
				fromUsername: accountUSD1.Owner,
				arg: domain.CreateTransferParams{
					FromAccountID: accountUSD1.ID,
					ToAccountID:   accountUSD2.ID,
					Amount:        amount,
				},
			},
			buildStubs: func(repo *MockRepo, accountService *accountdelivery.MockService) {
				repo.EXPECT().Transfer(gomock.Any(), gomock.Any()).
					Times(1).
					Return(domain.TransferTxResult{}, errorspkg.ErrInternal)

				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountUSD1.ID)).
					Times(1).
					Return(domain.Account{
						ID:       accountUSD1.ID,
						Owner:    accountUSD1.Owner,
						Balance:  accountUSD1.Balance,
						Currency: accountUSD1.Currency,
					}, nil)

				accountService.EXPECT().Get(gomock.Any(), gomock.Eq(accountUSD2.ID)).
					Times(1).
					Return(domain.Account{
						Currency: accountUSD2.Currency,
					}, nil)
			},
			wantError: errorspkg.ErrInternal.Error(),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tranferRepo := NewMockRepo(ctrl)
			accountService := accountdelivery.NewMockService(ctrl)
			transferService := New(tranferRepo, accountService)

			tc.buildStubs(tranferRepo, accountService)

			got, err := transferService.Transfer(context.Background(), tc.input.fromUsername, tc.input.arg)
			if err != nil {
				if err.Error() == tc.wantError {
					return
				}

				t.Fatalf("transferService.Transfer(context.Background(), %v, %+v) got error: %v, want: %v",
					tc.input.fromUsername, tc.input.arg, err, tc.wantError)
			}

			tc.checkResponse(t, got)
		})
	}
}
