// Package randompkg provides functionality gor generating random applications common items.
package randompkg

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/shopspring/decimal"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// Intn is a shortcut for generating a random integer between 0 and max using crypto/rand.
func Intn(max int) int64 {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}

	return nBig.Int64()
}

// Float64 is a shortcut for generating a random float between 0 and 1 using crypto/rand.
func Float64() float64 {
	return float64(Intn(1<<32)) / (1 << 32)
}

// IntBetween generates a random integer between min and max.
func IntBetween(min, max int) int32 {
	return int32(Intn(max+min)) - int32(min)
}

// FloatBetween generates a random decimal number between min and max rounded to 4 decimals.
func FloatBetween(min, max float64) float64 {
	numInRange := min + Float64()*(max-min)
	return math.Floor(numInRange*10_000) / 10_000
}

// String generates a random string of length n.
func String(n int) string {
	var sb strings.Builder

	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[Intn(k)]

		_ = sb.WriteByte(c) // The returned err is always nil.
	}

	return sb.String()
}

// Owner generates a random owner name.
func Owner() string {
	return String(6)
}

// MoneyAmountBetween generates a random amount of money between min and max rounded to 4 decimals.
func MoneyAmountBetween(min, max float64) string {
	return decimal.NewFromFloat(FloatBetween(min, max)).String()
}

// Currency generates a random currency code.
func Currency() string {
	currencies := []string{"USD", "EUR", "RMB"}
	return currencies[Intn(len(currencies))]
}

// Email generates a random email.
func Email() string {
	return fmt.Sprintf("%s@email.com", String(10))
}
