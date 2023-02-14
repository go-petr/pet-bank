package service

import (
	"context"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/passpkg"
	"github.com/rs/zerolog"
)

//go:generate mockgen -source service.go -destination service_mock.go -package service
type userRepoInterface interface {
	CreateUser(ctx context.Context, arg domain.CreateUserParams) (domain.User, error)
	GetUser(ctx context.Context, username string) (domain.User, error)
}

type userService struct {
	repo userRepoInterface
}

func NewUserService(ur userRepoInterface) *userService {
	return &userService{
		repo: ur,
	}
}

func NewUserWihtoutPassword(u domain.User) domain.UserWihtoutPassword {
	return domain.UserWihtoutPassword{
		Username:  u.Username,
		FullName:  u.FullName,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

func (s *userService) CreateUser(ctx context.Context, username, password, fullname, email string) (domain.UserWihtoutPassword, error) {
	l := zerolog.Ctx(ctx)

	var response domain.UserWihtoutPassword

	hashedPassword, err := passpkg.Hash(password)
	if err != nil {
		l.Error().Err(err).Send()
		return response, errorspkg.ErrInternal
	}

	arg := domain.CreateUserParams{
		Username:       username,
		HashedPassword: hashedPassword,
		FullName:       fullname,
		Email:          email,
	}

	gotUser, err := s.repo.CreateUser(ctx, arg)
	if err != nil {
		return response, err
	}

	response = NewUserWihtoutPassword(gotUser)

	return response, nil
}
func (s *userService) CheckPassword(ctx context.Context, username, pass string) (domain.UserWihtoutPassword, error) {
	l := zerolog.Ctx(ctx)

	var response domain.UserWihtoutPassword

	gotUser, err := s.repo.GetUser(ctx, username)
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
