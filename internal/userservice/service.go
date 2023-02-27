// Package userservice manages business logic layer of users.
package userservice

import (
	"context"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/passpkg"
	"github.com/rs/zerolog"
)

// Repo provides data access layer interface needed by user service layer.
//
//go:generate mockgen -source service.go -destination service_mock.go -package userservice
type Repo interface {
	Create(ctx context.Context, arg domain.CreateUserParams) (domain.User, error)
	Get(ctx context.Context, username string) (domain.User, error)
}

// Service facilitates user service layer logic.
type Service struct {
	repo Repo
}

// New return user service struct to manage user bussines logic.
func New(ur Repo) *Service {
	return &Service{
		repo: ur,
	}
}

// NewUserWihtoutPassword returns user with removed sensitive data.
func NewUserWihtoutPassword(u domain.User) domain.UserWihtoutPassword {
	return domain.UserWihtoutPassword{
		Username:  u.Username,
		FullName:  u.FullName,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

// Create creates and returns user.
func (s *Service) Create(ctx context.Context, username, password, fullname, email string) (domain.UserWihtoutPassword, error) {
	l := zerolog.Ctx(ctx)

	var result domain.UserWihtoutPassword

	hashedPassword, err := passpkg.Hash(password)
	if err != nil {
		l.Error().Err(err).Send()
		return result, errorspkg.ErrInternal
	}

	arg := domain.CreateUserParams{
		Username:       username,
		HashedPassword: hashedPassword,
		FullName:       fullname,
		Email:          email,
	}

	gotUser, err := s.repo.Create(ctx, arg)
	if err != nil {
		return result, err
	}

	result = NewUserWihtoutPassword(gotUser)

	return result, nil
}

// CheckPassword checks if the password is valid for the given username.
func (s *Service) CheckPassword(ctx context.Context, username, pass string) (domain.UserWihtoutPassword, error) {
	l := zerolog.Ctx(ctx)

	var response domain.UserWihtoutPassword

	gotUser, err := s.repo.Get(ctx, username)
	if err != nil {
		return response, err
	}

	err = passpkg.Check(pass, gotUser.HashedPassword)
	if err != nil {
		l.Warn().Err(err).Send()
		return response, domain.ErrWrongPassword
	}

	response = NewUserWihtoutPassword(gotUser)

	return response, nil
}
