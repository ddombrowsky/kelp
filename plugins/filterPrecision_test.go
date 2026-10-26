package plugins

import (
	"testing"

	hProtocol "github.com/stellar/go-stellar-sdk/protocols/horizon"
	"github.com/stellar/go-stellar-sdk/txnbuild"
	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/kelp/support/utils"
)

// a sell offer of USD for BTC at 1/83000 BTC per USD
func makeUSDBTCSellOp() *txnbuild.ManageSellOffer {
	return &txnbuild.ManageSellOffer{
		Selling: utils.Asset2Asset(testUSD),
		Buying:  utils.Asset2Asset(testBTC),
		Amount:  "100.0000000",
		Price:   xdr.Price{N: 1, D: 83000},
	}
}

func applyFilter(t *testing.T, filter SubmitFilter, op *txnbuild.ManageSellOffer) []txnbuild.Operation {
	ops, e := filter.Apply([]txnbuild.Operation{op}, []hProtocol.Offer{}, []hProtocol.Offer{})
	require.NoError(t, e)
	return ops
}

// the filters compare against the exact price: 1/83000 = 0.0000120481928..., which rounds to 0.0000120 at 7 decimals
func TestMinPriceFilter_SmallPriceIsNotRounded(t *testing.T) {
	below := 0.0000120481
	filter, e := MakeFilterMinPrice(testUSD, testBTC, &MinPriceFilterConfig{MinPrice: &below})
	require.NoError(t, e)
	assert.Len(t, applyFilter(t, filter, makeUSDBTCSellOp()), 1, "price is above the minimum, so the offer is kept")

	above := 0.0000120482
	filter, e = MakeFilterMinPrice(testUSD, testBTC, &MinPriceFilterConfig{MinPrice: &above})
	require.NoError(t, e)
	assert.Len(t, applyFilter(t, filter, makeUSDBTCSellOp()), 0, "price is below the minimum, so the offer is dropped")
}

func TestMaxPriceFilter_SmallPriceIsNotRounded(t *testing.T) {
	above := 0.0000120482
	filter, e := MakeFilterMaxPrice(testUSD, testBTC, &MaxPriceFilterConfig{MaxPrice: &above})
	require.NoError(t, e)
	assert.Len(t, applyFilter(t, filter, makeUSDBTCSellOp()), 1, "price is below the maximum, so the offer is kept")

	below := 0.0000120481
	filter, e = MakeFilterMaxPrice(testUSD, testBTC, &MaxPriceFilterConfig{MaxPrice: &below})
	require.NoError(t, e)
	assert.Len(t, applyFilter(t, filter, makeUSDBTCSellOp()), 0, "price is above the maximum, so the offer is dropped")
}

// a buy offer priced below 5e-8 reads as 0 when rounded to 7 decimals, which made its base volume 0 and got it dropped
func TestOrderConstraintsFilter_TinyPricedBuyOfferIsKept(t *testing.T) {
	tiny := hProtocol.Asset{Type: "credit_alphanum4", Code: "TINY", Issuer: testUSD.Issuer}
	filter := MakeFilterOrderConstraints(sdexOrderConstraints, testBTC, tiny)

	// buying BTC (base) with 1,000,000 TINY (quote) at 3e-9 BTC per TINY is a base volume of 0.003 BTC
	op := &txnbuild.ManageSellOffer{
		Selling: utils.Asset2Asset(tiny),
		Buying:  utils.Asset2Asset(testBTC),
		Amount:  "1000000.0000000",
		Price:   xdr.Price{N: 3, D: 1000000000},
	}
	ops := applyFilter(t, filter, op)
	if assert.Len(t, ops, 1) {
		assert.Same(t, op, ops[0])
	}
}
