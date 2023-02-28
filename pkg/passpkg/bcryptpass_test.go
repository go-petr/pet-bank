package passpkg

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPassword(t *testing.T) {
	password := "abcdefghijklmnopqrstuvwxyz"

	// OK
	hashedPassword1, err := Hash(password)
	if err != nil {
		t.Errorf("Hash(%v) returned unexpected error: %v", password, err)
	}

	if err := Check(password, hashedPassword1); err != nil {
		t.Errorf("Check(%v, %v) returned unexpected error: %v", password, hashedPassword1, err)
	}

	// WrongPassword
	wrongPassword := "abc"

	if err := Check(wrongPassword, hashedPassword1); err != bcrypt.ErrMismatchedHashAndPassword {
		t.Errorf("Check(%v, %v), returned unexpected error: %v", wrongPassword, hashedPassword1, err)
	}

	// LongPassword
	longPassword := strings.Repeat("abc", 100)
	want := "failed to hash password: bcrypt: password length exceeds 72 bytes"

	hashedPassword1, err = Hash(longPassword)
	if err.Error() != want {
		t.Errorf("Hash(%v) returned unexpected error: %v", password, err)
	}

	// RandomSaltGeneration
	hashedPassword2, err := Hash(password)
	if err != nil {
		t.Errorf("Hash(%v) returned error: %v", password, err)
	}

	if hashedPassword1 == hashedPassword2 {
		t.Error("hashedPassword1 == hashedPassword2, want unequal")
	}
}
