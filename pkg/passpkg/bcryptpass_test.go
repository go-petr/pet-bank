package passpkg

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestPassword(t *testing.T) {
	password := "abcdefghijklmnopqrstuvwxyz"
	hashedPassword1, err := Hash(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword1)

	err = Check(password, hashedPassword1)
	require.NoError(t, err)

	wrongPassword := "abc"
	err = Check(wrongPassword, hashedPassword1)
	require.EqualError(t, err, bcrypt.ErrMismatchedHashAndPassword.Error())

	// Test for random salt generation
	hashedPassword2, err := Hash(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword1)
	require.NotEqual(t, hashedPassword1, hashedPassword2)
}
