package service

import (
	"context"

	"github.com/go-petr/pet-bank/internal/account"
)

type accountRepoInterface interface {
	CreateAccount(ctx context.Context, arg account.CreateAccountParams) (account.Account, error)
	GetAccount(ctx context.Context, id int32) (account.Account, error)
	ListAccounts(ctx context.Context, arg account.ListAccountsParams) ([]account.Account, error)
}

type accountService struct {
	repo accountRepoInterface
}

func NewAccountService(ar accountRepoInterface) *accountService {
	return &accountService{repo: ar}
}

func (s *accountService) CreateAccount(ctx context.Context, owner, currency string) (account.Account, error) {

	arg := account.CreateAccountParams{
		Owner:    owner,
		Currency: currency,
		Balance:  "0",
	}

	account, err := s.repo.CreateAccount(ctx, arg)
	if err != nil {
		return account, err
	}

	return account, nil
}

func (s *accountService) GetAccount(ctx context.Context, id int32) (account.Account, error) {

	account, err := s.repo.GetAccount(ctx, id)
	if err != nil {
		return account, err
	}

	return account, nil
}

func (s *accountService) ListAccounts(ctx context.Context, owner string, pageSize, pageID int32) ([]account.Account, error) {

	arg := account.ListAccountsParams{
		Owner:  owner,
		Limit:  pageSize,
		Offset: (pageID - 1) * pageSize,
	}

	accounts, err := s.repo.ListAccounts(ctx, arg)
	if err != nil {
		return nil, err
	}

	return accounts, err
}
