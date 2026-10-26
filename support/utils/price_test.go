package utils

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/assert"
)

// relativeError returns how far the price p is from want, as a fraction of want
func relativeError(p xdr.Price, want float64) float64 {
	got, _ := new(big.Rat).SetFrac64(int64(p.N), int64(p.D)).Float64()
	return math.Abs(got-want) / want
}

func TestPriceFromFloat(t *testing.T) {
	testCases := []struct {
		name  string
		price float64
		want  xdr.Price
	}{
		// a BTC-quoted price, which used to lose precision when rounded to 7 decimal places
		{name: "1/83000", price: 1.0 / 83000, want: xdr.Price{N: 1, D: 83000}},
		{name: "1/83000 + 0.1%", price: 1.001 / 83000, want: xdr.Price{N: 1001, D: 83000000}},
		{name: "1/83000 + 0.2%", price: 1.002 / 83000, want: xdr.Price{N: 501, D: 41500000}},
		{name: "tiny", price: 0.0000032, want: xdr.Price{N: 1, D: 312500}},
		{name: "tiny + 0.1%", price: 0.0000032032, want: xdr.Price{N: 1001, D: 312500000}},
		{name: "decimal", price: 0.1001, want: xdr.Price{N: 1001, D: 10000}},
		{name: "whole number", price: 2, want: xdr.Price{N: 2, D: 1}},
		{name: "one", price: 1, want: xdr.Price{N: 1, D: 1}},
		{name: "BTC price in USD", price: 83000, want: xdr.Price{N: 83000, D: 1}},
		{name: "smallest price", price: 1.0 / math.MaxInt32, want: xdr.Price{N: 1, D: math.MaxInt32}},
		{name: "largest price", price: math.MaxInt32, want: xdr.Price{N: math.MaxInt32, D: 1}},
		// the closest fraction within the int32 limit is a semiconvergent, about twice as close as the last
		// convergent that fits (270565557/735473437)
		{name: "1/e", price: 1 / math.E, want: xdr.Price{N: 681609404, D: 1852806457}},
	}

	for _, k := range testCases {
		t.Run(k.name, func(t *testing.T) {
			got, e := PriceFromFloat(k.price)
			if assert.NoError(t, e) {
				assert.Equal(t, k.want, got)
			}
		})
	}
}

// prices that cannot be written as a short fraction still come out within a tiny relative error, at any magnitude
func TestPriceFromFloat_Precision(t *testing.T) {
	for _, price := range []float64{math.Pi, 1 / math.Pi, 312456.7890123, 1.0 / 83123.456, 0.0000032123, 1.2345e-9, 1.2345e8} {
		got, e := PriceFromFloat(price)
		if assert.NoError(t, e, "%v", price) {
			assert.Less(t, relativeError(got, price), 1e-8, "%v approximated as %d/%d", price, got.N, got.D)
		}
	}
}

// a strategy's levels are a small percentage apart, and each one must stay a distinct price on the ledger
func TestPriceFromFloat_AdjacentLevelsStayDistinct(t *testing.T) {
	for _, midPrice := range []float64{1.0 / 83000, 0.0000032, 1e-8, 0.1, 83000} {
		t.Run(fmt.Sprintf("%v", midPrice), func(t *testing.T) {
			var previous *big.Rat
			for level := 0; level <= 10; level++ {
				price := midPrice * (1 + 0.001*float64(level))
				got, e := PriceFromFloat(price)
				if !assert.NoError(t, e) {
					return
				}
				assert.Less(t, relativeError(got, price), 1e-6, "level %d", level)

				current := new(big.Rat).SetFrac64(int64(got.N), int64(got.D))
				if previous != nil {
					assert.Equal(t, 1, current.Cmp(previous), "level %d (%d/%d) is not above the level below it", level, got.N, got.D)
				}
				previous = current
			}
		})
	}
}

func TestPriceFromFloat_Unrepresentable(t *testing.T) {
	for _, price := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1), 1e-10, 0.99 / math.MaxInt32, 3e9} {
		_, e := PriceFromFloat(price)
		assert.Error(t, e, "%v", price)
	}
}
