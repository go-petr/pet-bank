package tokenpkg

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNewJWTMaker(t *testing.T) {
	t.Parallel()

	// OK
	secretKey := strings.Repeat("x", 32)

	_, err := NewJWTMaker(secretKey)
	if err != nil {
		t.Errorf("NewJWTMaker(%v) returned error: %v", secretKey, err)
	}

	// shortKeyError
	shortKey := strings.Repeat("x", 30)

	got, err := NewJWTMaker(shortKey)
	if err.Error() != fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize).Error() {
		t.Errorf("NewJWTMaker(%v) returned unexpected error: %v", secretKey, err)
	}

	if got != nil {
		t.Errorf("JWTMaker = %+v, want nil", got)
	}
}

func TestJWTMaker(t *testing.T) {
	t.Parallel()

	secretKey := randompkg.String(32)

	maker, err := NewJWTMaker(secretKey)
	if err != nil {
		t.Fatalf("NewJWTMaker(%v) returned error: %v", secretKey, err)
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

func TestExpiredJWTToken(t *testing.T) {
	t.Parallel()

	secretKey := randompkg.String(32)

	maker, err := NewJWTMaker(secretKey)
	if err != nil {
		t.Fatalf("NewJWTMaker(%v) returned error: %v", secretKey, err)
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

func TestInvalidJWTTokenAlgNone(t *testing.T) {
	t.Parallel()

	username := randompkg.Owner()
	duration := time.Minute

	payload, err := NewPayload(username, duration)
	if err != nil {
		t.Errorf("NewPayload(%v, %v) returned error: %v", username, duration, err)
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodNone, payload)

	token, err := jwtToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Errorf("jwtToken.SignedString(%v) returned error: %v", jwt.UnsafeAllowNoneSignatureType, err)
	}

	secretKey := randompkg.String(32)

	maker, err := NewJWTMaker(secretKey)
	if err != nil {
		t.Fatalf("NewJWTMaker(%v) returned error: %v", secretKey, err)
	}

	_, err = maker.VerifyToken(token)
	if err != ErrInvalidToken {
		t.Errorf("maker.VerifyToken(%v) returned error: %v", token, err)
	}
}
