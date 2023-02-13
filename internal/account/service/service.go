package service

import (
	"context"

	"github.com/go-petr/pet-bank/internal/domain"
)

type accountRepoInterface interface {
	CreateAccount(ctx context.Context, owner, balance, currency string) (domain.Account, error)
	GetAccount(ctx context.Context, id int32) (domain.Account, error)
	ListAccounts(ctx context.Context, owner string, limit, offset int32) ([]domain.Account, error)
}

type accountService struct {
	repo accountRepoInterface
}

func NewAccountService(ar accountRepoInterface) *accountService {
	return &accountService{repo: ar}
}

func (s *accountService) CreateAccount(ctx context.Context, owner, currency string) (domain.Account, error) {

	account, err := s.repo.CreateAccount(ctx, owner, "0", currency)
	if err != nil {
		return account, err
	}

	return account, nil
}

func (s *accountService) GetAccount(ctx context.Context, id int32) (domain.Account, error) {

	account, err := s.repo.GetAccount(ctx, id)
	if err != nil {
		return account, err
	}

	return account, nil
}

func (s *accountService) ListAccounts(ctx context.Context, owner string, pageSize, pageID int32) ([]domain.Account, error) {

	limit := pageSize
	offset := (pageID - 1) * pageSize

	accounts, err := s.repo.ListAccounts(ctx, owner, limit, offset)
	if err != nil {
		return nil, err
	}

	return accounts, err
}
