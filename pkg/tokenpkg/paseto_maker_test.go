package tokenpkg

import (
	"testing"
	"time"

	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestPasetoMaker(t *testing.T) {
	t.Parallel()

	secretKey := randompkg.String(32)

	maker, err := NewPasetoMaker(secretKey)
	if err != nil {
		t.Fatalf("NewPasetoMaker(%v) returned error: %v", secretKey, err)
	}

	username := randompkg.Owner()
	duration := time.Minute

	token, payload, err := maker.CreateToken(username, duration)
	if err != nil {
		t.Errorf("maker.CreateToken(%v, %v) returned error: %v", username, duration, err)
	}

	_, err = maker.VerifyToken(token)
	if err != nil {
		t.Errorf("maker.VerifyToken(%v) returned error: %v", token, err)
	}

	want := &Payload{
		Username:  username,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}

	ignore := cmpopts.IgnoreFields(Payload{}, "ID")
	delta := cmpopts.EquateApproxTime(time.Minute)

	if diff := cmp.Diff(payload, want, ignore, delta); diff != "" {
		t.Errorf("maker.CreateToken(%v, %v) returned unexpected diff: %v", username, duration, diff)
	}
}

func TestExpiredPasetoToken(t *testing.T) {
	t.Parallel()

	secretKey := randompkg.String(32)

	maker, err := NewPasetoMaker(randompkg.String(32))
	if err != nil {
		t.Fatalf("NewPasetoMaker(%v) returned error: %v", secretKey, err)
	}

	username := randompkg.Owner()
	duration := -time.Minute

	token, _, err := maker.CreateToken(username, duration)
	if err != nil {
		t.Errorf("maker.CreateToken(%v, %v) returned error: %v", username, duration, err)
	}

	_, err = maker.VerifyToken(token)
	if err != ErrExpiredToken {
		t.Errorf("maker.VerifyToken(%v) returned unexpected error: %v", token, err)
	}
}
