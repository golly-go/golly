package utils

import (
	"math"
	"math/rand"
	"time"

	"github.com/shopspring/decimal"
)

func IntDefault(a, b int) int {
	if a == 0 {
		return b
	}
	return a
}

// IntMax
func IntMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func IntMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Int64Max(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

func Int64Min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

func RandUint64() uint64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Uint64()
}

// CentsToDollars converts cents to dollars
func CentsToDollars(amount int) float64 {
	return float64(amount) / 100.0

}

// DollarsToCents converts dollars to cents
func DollarsToCents(amount float64) int {
	return int(BankRoundCurrency(amount * 100))
}

// BankRoundCurrency runs bank rounding with percision of 2 for cents
func BankRoundCurrency(amt float64) float64 {
	a, _ := decimal.NewFromFloat(amt).RoundBank(2).Float64()
	return a
}

func Round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(Round(num*output)) / output
}
