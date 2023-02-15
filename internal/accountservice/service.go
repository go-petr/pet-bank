// Package accountservice manages business logic layer of accounts.
package accountservice

import (
	"context"

	"github.com/go-petr/pet-bank/internal/domain"
)

// Repo provides data access layer interface needed by account service layer.
type Repo interface {
	Create(ctx context.Context, owner, balance, currency string) (domain.Account, error)
	Get(ctx context.Context, id int32) (domain.Account, error)
	List(ctx context.Context, owner string, limit, offset int32) ([]domain.Account, error)
}

// Service facilitates account service layer logic.
type Service struct {
	repo Repo
}

// New returns account service struct to manage account bussines logic.
func New(ar Repo) *Service {
	return &Service{repo: ar}
}

// Create creates and returns account for the given owner and currency.
func (s *Service) Create(ctx context.Context, owner, currency string) (domain.Account, error) {
	account, err := s.repo.Create(ctx, owner, "0", currency)
	if err != nil {
		return account, err
	}

	return account, nil
}

// Get returns account for the given account ID.
func (s *Service) Get(ctx context.Context, id int32) (domain.Account, error) {
	account, err := s.repo.Get(ctx, id)
	if err != nil {
		return account, err
	}

	return account, nil
}

// List returns accounts that are owned by the given user.
func (s *Service) List(ctx context.Context, owner string, pageSize, pageID int32) ([]domain.Account, error) {
	limit := pageSize
	offset := (pageID - 1) * pageSize

	accounts, err := s.repo.List(ctx, owner, limit, offset)
	if err != nil {
		return nil, err
	}

	return accounts, err
}
