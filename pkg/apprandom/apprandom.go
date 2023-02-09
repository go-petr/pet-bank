// Package apprandom provides functionality gor generating random applications common items.
package apprandom

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// IntBetween generates a random integer between min and max.
func IntBetween(min, max int32) int32 {
	return min + rand.Int31n(max-min+1)
}

// FloatBetween generates a random decimal number between min and max rounded.
func FloatBetween(min, max float64) float64 {
	return math.Floor((min+rand.Float64()*(max-min))*10_000) / 10_000
}

// String generates a random string of length n.
func String(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// Owner generates a random owner name.
func Owner() string {
	return String(6)
}

// MoneyAmountBetween generates a random amount of money between 1,000 and 10,000.
func MoneyAmountBetween(min, max float64) string {
	return decimal.NewFromFloat(FloatBetween(min, max)).String()
}

// Currency generates a random currency code.
func Currency() string {
	currencies := []string{"USD", "EUR", "RMB"}
	return currencies[rand.Intn(len(currencies))]
}

// Email generates a random email.
func Email() string {
	return fmt.Sprintf("%s@email.com", String(10))
}
