package userservice

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
	"github.com/google/go-cmp/cmp"
)

func randomUser(t *testing.T) (domain.User, string) {
	password := randompkg.String(10)

	hashedPassword, err := passpkg.Hash(password)
	if err != nil {
		t.Fatalf("passpkg.Hash(%v) failed: %v", password, err)
	}

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

func TestCreate(t *testing.T) {
	t.Parallel()

	user, password := randomUser(t)

	type input struct {
		Username string
		Password string
		Fullname string
		Email    string
	}

	testCases := []struct {
		name          string
		input         input
		buildStubs    func(userRepo *MockRepo)
		checkResponse func(t *testing.T, got domain.UserWihtoutPassword)
		wantError     error
	}{
		{
			name: "OK",
			input: input{
				user.Username,
				password,
				user.FullName,
				user.Email,
			},
			buildStubs: func(userRepo *MockRepo) {
				userRepo.EXPECT().
					Create(gomock.Any(), EqCreateUserParams(
						domain.CreateUserParams{
							Username:       user.Username,
							HashedPassword: user.HashedPassword,
							FullName:       user.FullName,
							Email:          user.Email,
						}, password)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(t *testing.T, got domain.UserWihtoutPassword) {
				want := NewUserWihtoutPassword(user)

				if !cmp.Equal(got, want) {
					t.Errorf("domain.UserWihtoutPassword = %+v, want %+v", got, want)
				}
			},
		},
		{
			name: "HashPasswordErr",
			input: input{
				user.Username,
				strings.Repeat("long", 100),
				user.FullName,
				user.Email,
			},
			buildStubs: func(userRepo *MockRepo) {
				userRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantError: errorspkg.ErrInternal,
		},
		{
			name: "CreateUserRepoErr",
			input: input{
				user.Username,
				password,
				user.FullName,
				user.Email,
			},
			buildStubs: func(userRepo *MockRepo) {
				userRepo.EXPECT().
					Create(gomock.Any(), EqCreateUserParams(
						domain.CreateUserParams{
							Username:       user.Username,
							HashedPassword: user.HashedPassword,
							FullName:       user.FullName,
							Email:          user.Email,
						}, password)).
					Times(1).
					Return(domain.User{}, errorspkg.ErrInternal)
			},
			wantError: errorspkg.ErrInternal,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userRepo := NewMockRepo(ctrl)
			userService := New(userRepo)

			tc.buildStubs(userRepo)

			got, err := userService.Create(context.Background(),
				tc.input.Username,
				tc.input.Password,
				tc.input.Fullname,
				tc.input.Email,
			)
			if err != nil {
				if err == tc.wantError {
					return
				}

				t.Fatalf("userService.Create(context.Background(), %v, %v, %v, %v) got error %v, want %v",
					tc.input.Username, tc.input.Password, tc.input.Fullname, tc.input.Email, err, tc.wantError)
			}

			tc.checkResponse(t, got)
		})
	}
}

func TestCheckPassword(t *testing.T) {
	t.Parallel()

	user, password := randomUser(t)

	testCases := []struct {
		name          string
		username      string
		password      string
		buildStubs    func(userRepo *MockRepo)
		checkResponse func(t *testing.T, got domain.UserWihtoutPassword)
		wantError     error
	}{
		{
			name:     "OK",
			username: user.Username,
			password: password,
			buildStubs: func(userRepo *MockRepo) {
				userRepo.EXPECT().
					Get(gomock.Any(), user.Username).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(t *testing.T, got domain.UserWihtoutPassword) {
				want := NewUserWihtoutPassword(user)

				if !cmp.Equal(got, want) {
					t.Errorf("domain.UserWihtoutPassword = %+v, want %+v", got, want)
				}
			},
		},
		{
			name:     "GetUserError",
			username: user.Username,
			password: password,
			buildStubs: func(userRepo *MockRepo) {
				userRepo.EXPECT().
					Get(gomock.Any(), user.Username).
					Times(1).
					Return(domain.User{}, domain.ErrUsernameAlreadyExists)
			},
			wantError: domain.ErrUsernameAlreadyExists,
		},
		{
			name:     "WrongPassword",
			username: user.Username,
			password: "wrong",
			buildStubs: func(userRepo *MockRepo) {
				userRepo.EXPECT().
					Get(gomock.Any(), user.Username).
					Times(1).
					Return(user, nil)
			},
			wantError: domain.ErrWrongPassword,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userRepo := NewMockRepo(ctrl)
			userService := New(userRepo)

			tc.buildStubs(userRepo)

			got, err := userService.CheckPassword(context.Background(),
				tc.username,
				tc.password,
			)
			if err != nil {
				if err == tc.wantError {
					return
				}

				t.Fatalf("userService.CheckPassword(context.Background(), %v, %v) got error %v, want %v",
					tc.username, tc.password, err, tc.wantError)
			}

			tc.checkResponse(t, got)
		})
	}
}
