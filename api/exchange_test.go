package api

import (
	"testing"

	"github.com/stellar/go-stellar-sdk/price"
	"github.com/stellar/go-stellar-sdk/txnbuild"
	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCoupon = txnbuild.CreditAsset{Code: "COUPON", Issuer: "GBMMZMK2DC4FFP4CAI6KCVNCQ7WLO5A7DQU7EC7WGHRDQBZB763X4OQI"}
var testBTC = txnbuild.CreditAsset{Code: "BTC", Issuer: "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"}

const testSourceAccount = "GDUKMGUGDZQK6YHYA5Z6AY2G4XDSZPSZ3SW5UN3ARVMO6QSRDWP5YLEX"

// The expected values are base64 XDR operations produced by the conversion through the deprecated "build" package of
// the old stellar/go SDK (github.com/stellar/go v0.0.0-20211007183021-ea18bbab9344), which NormalizeOfferOps replaces.
// wantSubmitted is the result of one conversion, applied to every submission (prune ops, deleting all offers).
// wantUpdate is the result of the two conversions applied to the strategy's update ops (in the trader, then on submit).
var normalizeTestCases = []struct {
	name          string
	op            *txnbuild.ManageSellOffer
	wantSubmitted string
	wantUpdate    string
}{
	{
		name:          "create xlm for coupon",
		op:            &txnbuild.ManageSellOffer{Selling: txnbuild.NativeAsset{}, Buying: testCoupon, Amount: "100.0000000", Price: price.MustParse("0.1001000")},
		wantSubmitted: "AAAAAAAAAAMAAAAAAAAAAkNPVVBPTgAAAAAAAAAAAABYzLFaGLhSv4ICPKFVoofst3QfHCnyC/Yx4jgHIf+3fgAAAAA7msoAAAAD6QAAJxAAAAAAAAAAAA==",
		wantUpdate:    "AAAAAAAAAAMAAAAAAAAAAkNPVVBPTgAAAAAAAAAAAABYzLFaGLhSv4ICPKFVoofst3QfHCnyC/Yx4jgHIf+3fgAAAAA7msoAAAAD6QAAJxAAAAAAAAAAAA==",
	}, {
		name:          "modify coupon for xlm",
		op:            &txnbuild.ManageSellOffer{Selling: testCoupon, Buying: txnbuild.NativeAsset{}, Amount: "1000.0000000", Price: price.MustParse("9.9900100"), OfferID: 12345},
		wantSubmitted: "AAAAAAAAAAMAAAACQ09VUE9OAAAAAAAAAAAAAFjMsVoYuFK/ggI8oVWih+y3dB8cKfIL9jHiOAch/7d+AAAAAAAAAAJUC+QAAA8+WQABhqAAAAAAAAAwOQ==",
		wantUpdate:    "AAAAAAAAAAMAAAACQ09VUE9OAAAAAAAAAAAAAFjMsVoYuFK/ggI8oVWih+y3dB8cKfIL9jHiOAch/7d+AAAAAAAAAAJUC+QAAA8+WQABhqAAAAAAAAAwOQ==",
	}, {
		name:          "large price with source",
		op:            &txnbuild.ManageSellOffer{Selling: testBTC, Buying: txnbuild.NativeAsset{}, Amount: "0.0012345", Price: price.MustParse("312456.7890123"), SourceAccount: testSourceAccount},
		wantSubmitted: "AAAAAQAAAADophqGHmCvYPgHc+BjRuXHLL5Z3K3aN2CNWO9CUR2f3AAAAAMAAAABQlRDAAAAAAA7mRE4Dv6Yi6CokA6xz+RPNm99vpRr7QdyQPf2JN8VxQAAAAAAAAAAAAAwOR/djj4AAAavAAAAAAAAAAA=",
		wantUpdate:    "AAAAAQAAAADophqGHmCvYPgHc+BjRuXHLL5Z3K3aN2CNWO9CUR2f3AAAAAMAAAABQlRDAAAAAAA7mRE4Dv6Yi6CokA6xz+RPNm99vpRr7QdyQPf2JN8VxQAAAAAAAAAAAAAwOR/djj4AAAavAAAAAAAAAAA=",
	}, {
		name:          "price exactly 0.001 is not flipped",
		op:            &txnbuild.ManageSellOffer{Selling: txnbuild.NativeAsset{}, Buying: testBTC, Amount: "500.0000000", Price: price.MustParse("0.0010000")},
		wantSubmitted: "AAAAAAAAAAMAAAAAAAAAAUJUQwAAAAAAO5kROA7+mIugqJAOsc/kTzZvfb6Ua+0HckD39iTfFcUAAAABKgXyAAAAAAEAAAPoAAAAAAAAAAA=",
		wantUpdate:    "AAAAAAAAAAMAAAAAAAAAAUJUQwAAAAAAO5kROA7+mIugqJAOsc/kTzZvfb6Ua+0HckD39iTfFcUAAAABKgXyAAAAAAEAAAPoAAAAAAAAAAA=",
	}, {
		name:          "tiny price is flipped to buy offer",
		op:            &txnbuild.ManageSellOffer{Selling: txnbuild.NativeAsset{}, Buying: testBTC, Amount: "500.0000000", Price: price.MustParse("0.0000032")},
		wantSubmitted: "AAAAAAAAAAwAAAAAAAAAAUJUQwAAAAAAO5kROA7+mIugqJAOsc/kTzZvfb6Ua+0HckD39iTfFcUAAAABKgXyAAAExLQAAAABAAAAAAAAAAA=",
		wantUpdate:    "AAAAAAAAAAwAAAAAAAAAAUJUQwAAAAAAO5kROA7+mIugqJAOsc/kTzZvfb6Ua+0HckD39iTfFcUAAAAAAAA+gAAExLQAAAABAAAAAAAAAAA=",
	}, {
		name:          "tiny price flip keeps offer id and source",
		op:            &txnbuild.ManageSellOffer{Selling: txnbuild.NativeAsset{}, Buying: testBTC, Amount: "12.3456789", Price: price.MustParse("0.0009999"), OfferID: 777, SourceAccount: testSourceAccount},
		wantSubmitted: "AAAAAQAAAADophqGHmCvYPgHc+BjRuXHLL5Z3K3aN2CNWO9CUR2f3AAAAAwAAAAAAAAAAUJUQwAAAAAAO5kROA7+mIugqJAOsc/kTzZvfb6Ua+0HckD39iTfFcUAAAAAB1vNFQX2CBEAAYagAAAAAAAAAwk=",
		wantUpdate:    "AAAAAQAAAADophqGHmCvYPgHc+BjRuXHLL5Z3K3aN2CNWO9CUR2f3AAAAAwAAAAAAAAAAUJUQwAAAAAAO5kROA7+mIugqJAOsc/kTzZvfb6Ua+0HckD39iTfFcUAAAAAAAHiNAX2CBEAAYagAAAAAAAAAAA=",
	}, {
		name:          "delete with normal price",
		op:            &txnbuild.ManageSellOffer{Selling: testCoupon, Buying: txnbuild.NativeAsset{}, Amount: "0", Price: price.MustParse("9.9900100"), OfferID: 42},
		wantSubmitted: "AAAAAAAAAAMAAAACQ09VUE9OAAAAAAAAAAAAAFjMsVoYuFK/ggI8oVWih+y3dB8cKfIL9jHiOAch/7d+AAAAAAAAAAAAAAAAAA8+WQABhqAAAAAAAAAAKg==",
		wantUpdate:    "AAAAAAAAAAMAAAACQ09VUE9OAAAAAAAAAAAAAFjMsVoYuFK/ggI8oVWih+y3dB8cKfIL9jHiOAch/7d+AAAAAAAAAAAAAAAAAA8+WQABhqAAAAAAAAAAKg==",
	}, {
		name:          "delete with tiny price",
		op:            &txnbuild.ManageSellOffer{Selling: txnbuild.NativeAsset{}, Buying: testBTC, Amount: "0", Price: price.MustParse("0.0000032"), OfferID: 43},
		wantSubmitted: "AAAAAAAAAAwAAAAAAAAAAUJUQwAAAAAAO5kROA7+mIugqJAOsc/kTzZvfb6Ua+0HckD39iTfFcUAAAAAAAAAAAAExLQAAAABAAAAAAAAACs=",
		wantUpdate:    "AAAAAAAAAAwAAAAAAAAAAUJUQwAAAAAAO5kROA7+mIugqJAOsc/kTzZvfb6Ua+0HckD39iTfFcUAAAAAAAAAAAAExLQAAAABAAAAAAAAAAA=",
	},
}

func normalizeToXDR(t *testing.T, ops []txnbuild.Operation, times int) string {
	var e error
	for i := 0; i < times; i++ {
		ops, e = NormalizeOfferOps(ops)
		require.NoError(t, e)
	}
	require.Len(t, ops, 1)

	op, e := ops[0].BuildXDR()
	require.NoError(t, e)
	s, e := xdr.MarshalBase64(op)
	require.NoError(t, e)
	return s
}

func TestNormalizeOfferOps_MatchesOldSDKConversion(t *testing.T) {
	for _, k := range normalizeTestCases {
		t.Run(k.name, func(t *testing.T) {
			assert.Equal(t, k.wantSubmitted, normalizeToXDR(t, []txnbuild.Operation{k.op}, 1), "single conversion (submit)")
			assert.Equal(t, k.wantUpdate, normalizeToXDR(t, []txnbuild.Operation{k.op}, 2), "double conversion (update)")
		})
	}
}

func TestNormalizeOfferOps_PassesThroughOtherOps(t *testing.T) {
	ops := []txnbuild.Operation{
		&txnbuild.ManageData{Name: "term", Value: []byte("1000")},
		&txnbuild.ManageSellOffer{Selling: testCoupon, Buying: txnbuild.NativeAsset{}, Amount: "1.0000000", Price: price.MustParse("2")},
	}

	got, e := NormalizeOfferOps(ops)
	if assert.NoError(t, e) && assert.Len(t, got, 2) {
		assert.Same(t, ops[0], got[0])
		assert.IsType(t, &txnbuild.ManageSellOffer{}, got[1])
	}
}

func TestNormalizeOfferOps_InvalidAmount(t *testing.T) {
	_, e := NormalizeOfferOps([]txnbuild.Operation{
		&txnbuild.ManageSellOffer{Selling: testCoupon, Buying: txnbuild.NativeAsset{}, Amount: "not a number", Price: price.MustParse("2")},
	})
	assert.Error(t, e)
}
