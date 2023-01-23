package service

import (
	"context"

	"github.com/go-petr/pet-bank/internal/user"
	"github.com/go-petr/pet-bank/pkg/util"
)

//go:generate mockgen -source service.go -destination service_mock.go -package service
type userRepo interface {
	CreateUser(ctx context.Context, arg user.CreateUserParams) (user.User, error)
	GetUser(ctx context.Context, username string) (user.User, error)
}

type userService struct {
	repo userRepo
}

func NewUserService(ur userRepo) *userService {
	return &userService{
		repo: ur,
	}
}

func NewUserWihtoutPassword(u user.User) user.UserWihtoutPassword {
	return user.UserWihtoutPassword{
		Username:          u.Username,
		FullName:          u.FullName,
		Email:             u.Email,
		PasswordChangedAt: u.PasswordChangedAt,
		CreatedAt:         u.CreatedAt,
	}
}

func (s *userService) CreateUser(ctx context.Context, username, password, fullname, email string) (user.UserWihtoutPassword, error) {

	var response user.UserWihtoutPassword

	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		return response, err
	}

	arg := user.CreateUserParams{
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
func (s *userService) CheckPassword(ctx context.Context, username, pass string) (user.UserWihtoutPassword, error) {

	var response user.UserWihtoutPassword

	gotUser, err := s.repo.GetUser(ctx, username)
	if err != nil {
		return response, err
	}

	err = util.CheckPassword(pass, gotUser.HashedPassword)
	if err != nil {
		return response, user.ErrWrongPassword
	}

	response = NewUserWihtoutPassword(gotUser)
	return response, nil
}
