package service

import (
	"context"
	"fmt"
	reflect "reflect"
	"strings"
	"testing"

	"github.com/go-petr/pet-bank/internal/domain"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/passpkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func randomUser(t *testing.T) (domain.User, string) {
	password := randompkg.String(10)

	hashedPassword, err := passpkg.Hash(password)
	require.NoError(t, err)

	user := domain.User{
		Username:       randompkg.Owner(),
		HashedPassword: hashedPassword,
		FullName:       randompkg.Owner(),
		Email:          randompkg.Email(),
	}

	return user, password
}

type eqCreateUserParamsMathcer struct {
	arg      domain.CreateUserParams
	password string
}

func (e eqCreateUserParamsMathcer) Matches(x interface{}) bool {
	arg, ok := x.(domain.CreateUserParams)
	if !ok {
		return false
	}

	err := passpkg.Check(e.password, arg.HashedPassword)
	if err != nil {
		return false
	}

	e.arg.HashedPassword = arg.HashedPassword
	
	return reflect.DeepEqual(e.arg, arg)
}

func (e eqCreateUserParamsMathcer) String() string {
	return fmt.Sprintf("mathces arg %v and password %v", e.arg, e.password)
}

func EqCreateUserParams(arg domain.CreateUserParams, password string) gomock.Matcher {
	return eqCreateUserParamsMathcer{arg, password}
}

func TestCreateUser(t *testing.T) {
	testUser, testPassword := randomUser(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := NewMockuserRepoInterface(ctrl)
	userService := NewUserService(userRepo)

	testCases := []struct {
		name  string
		input struct {
			Username string
			Password string
			Fullname string
			Email    string
		}
		buildStubs    func(userRepo *MockuserRepoInterface)
		checkResponse func(response domain.UserWihtoutPassword, err error)
	}{
		{
			name: "HashPasswordErr",
			input: struct {
				Username string
				Password string
				Fullname string
				Email    string
			}{
				testUser.Username,
				strings.Repeat("long", 100),
				testUser.FullName,
				testUser.Email,
			},
			buildStubs: func(userRepo *MockuserRepoInterface) {
				userRepo.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(response domain.UserWihtoutPassword, err error) {
				require.Equal(t, domain.UserWihtoutPassword{}, response)
				require.NotEmpty(t, err)
			},
		},
		{
			name: "CreateUserRepoErr",
			input: struct {
				Username string
				Password string
				Fullname string
				Email    string
			}{
				testUser.Username,
				testPassword,
				testUser.FullName,
				testUser.Email,
			},
			buildStubs: func(userRepo *MockuserRepoInterface) {
				userRepo.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(
						domain.CreateUserParams{
							Username:       testUser.Username,
							HashedPassword: testUser.HashedPassword,
							FullName:       testUser.FullName,
							Email:          testUser.Email,
						}, testPassword)).
					Times(1).
					Return(domain.User{}, errorspkg.ErrInternal)
			},
			checkResponse: func(response domain.UserWihtoutPassword, err error) {
				require.Equal(t, domain.UserWihtoutPassword{}, response)
				require.NotEmpty(t, err)
			},
		},
		{
			name: "OK",
			input: struct {
				Username string
				Password string
				Fullname string
				Email    string
			}{
				testUser.Username,
				testPassword,
				testUser.FullName,
				testUser.Email,
			},
			buildStubs: func(userRepo *MockuserRepoInterface) {
				userRepo.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(
						domain.CreateUserParams{
							Username:       testUser.Username,
							HashedPassword: testUser.HashedPassword,
							FullName:       testUser.FullName,
							Email:          testUser.Email,
						}, testPassword)).
					Times(1).
					Return(testUser, nil)
			},
			checkResponse: func(response domain.UserWihtoutPassword, err error) {
				require.NoError(t, err)

				require.Equal(t, testUser.Username, response.Username)
				require.Equal(t, testUser.FullName, response.FullName)
				require.Equal(t, testUser.Email, response.Email)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs(userRepo)

			response, err := userService.CreateUser(context.Background(),
				tc.input.Username,
				tc.input.Password,
				tc.input.Fullname,
				tc.input.Email,
			)

			tc.checkResponse(response, err)
		})
	}
}

func TestCheckPassword(t *testing.T) {
	testUser, testPassword := randomUser(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := NewMockuserRepoInterface(ctrl)
	userService := NewUserService(userRepo)

	testCases := []struct {
		name          string
		username      string
		password      string
		buildStubs    func(userRepo *MockuserRepoInterface)
		checkResponse func(response domain.UserWihtoutPassword, err error)
	}{
		{
			name:     "GetUserError",
			username: testUser.Username,
			password: testPassword,
			buildStubs: func(userRepo *MockuserRepoInterface) {
				userRepo.EXPECT().
					GetUser(gomock.Any(), testUser.Username).
					Times(1).
					Return(domain.User{}, domain.ErrUsernameAlreadyExists)
			},
			checkResponse: func(response domain.UserWihtoutPassword, err error) {
				require.Equal(t, domain.UserWihtoutPassword{}, response)
				require.EqualError(t, domain.ErrUsernameAlreadyExists, err.Error())
			},
		},

		{
			name:     "WrongPassword",
			username: testUser.Username,
			password: "wrong",
			buildStubs: func(userRepo *MockuserRepoInterface) {
				userRepo.EXPECT().
					GetUser(gomock.Any(), testUser.Username).
					Times(1).
					Return(testUser, nil)
			},
			checkResponse: func(response domain.UserWihtoutPassword, err error) {
				require.Equal(t, domain.UserWihtoutPassword{}, response)
				require.EqualError(t, domain.ErrWrongPassword, err.Error())
			},
		},

		{
			name:     "OK",
			username: testUser.Username,
			password: testPassword,
			buildStubs: func(userRepo *MockuserRepoInterface) {
				userRepo.EXPECT().
					GetUser(gomock.Any(), testUser.Username).
					Times(1).
					Return(testUser, nil)
			},
			checkResponse: func(response domain.UserWihtoutPassword, err error) {
				require.Equal(t, NewUserWihtoutPassword(testUser), response)
				require.NoError(t, err)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs(userRepo)

			response, err := userService.CheckPassword(context.Background(),
				tc.username,
				tc.password,
			)

			tc.checkResponse(response, err)
		})
	}
}
