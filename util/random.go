package util

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

// RandomInt generates a random integer between min and max
func RandomInt(min, max int32) int32 {
	return min + rand.Int31n(max-min+1)
}

// RandomFloat generates a random decimal number between min and max rounded.
func RandomFloat(min, max float64) float64 {
	return math.Floor((min+rand.Float64()*(max-min))*10_000) / 10_000
}

// RandomString generates a random string of length n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// RandomOwner generates a random owner name
func RandomOwner() string {
	return RandomString(6)
}

// RandomMoneyAmount generates a random amount of money between 1,000 and 10,000
func RandomMoneyAmountBetween(min, max float64) string {
	return decimal.NewFromFloat(RandomFloat(min, max)).String()
}

// RandomCurrency generates a random currency code
func RandomCurrency() string {
	currencies := []string{USD, EUR, RMB}
	return currencies[rand.Intn(len(currencies))]
}

// RandomEmail generates a random email
func RandomEmail() string {
	return fmt.Sprintf("%s@email.com", RandomString(10))
}
