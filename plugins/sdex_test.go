package plugins

import (
	"testing"

	hProtocol "github.com/stellar/go-stellar-sdk/protocols/horizon"
	"github.com/stellar/go-stellar-sdk/txnbuild"
	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/kelp/api"
	"github.com/stellar/kelp/support/utils"
)

var testUSD = hProtocol.Asset{Type: "credit_alphanum4", Code: "USD", Issuer: "GDUKMGUGDZQK6YHYA5Z6AY2G4XDSZPSZ3SW5UN3ARVMO6QSRDWP5YLEX"}
var testBTC = hProtocol.Asset{Type: "credit_alphanum4", Code: "BTC", Issuer: "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"}

// makeOfflineSDEX returns an SDEX whose balance and liability lookups are served from the IEIF caches, so offers can
// be built without a horizon instance
func makeOfflineSDEX() *SDEX {
	ieif := MakeIEIF()
	plenty := api.Balance{Balance: 1e9, Trust: 1e12, Reserve: 0}
	for _, asset := range []hProtocol.Asset{utils.NativeAsset, testUSD, testBTC} {
		ieif.cachedBalances[asset] = plenty
		ieif.cachedLiabilities[asset] = Liabilities{}
	}

	const account = "GBMMZMK2DC4FFP4CAI6KCVNCQ7WLO5A7DQU7EC7WGHRDQBZB763X4OQI"
	return &SDEX{
		ieif:           ieif,
		TradingAccount: account,
		SourceAccount:  account,
	}
}

// requireOfferPrice checks the offer is a sell offer carrying exactly the wanted price, including once encoded
func requireOfferPrice(t *testing.T, offer *txnbuild.ManageSellOffer, want xdr.Price) {
	require.NotNil(t, offer)
	assert.Equal(t, want, offer.Price)

	op, e := offer.BuildXDR()
	require.NoError(t, e)
	assert.Equal(t, xdr.OperationTypeManageSellOffer, op.Body.Type)
	assert.Equal(t, want, op.Body.ManageSellOfferOp.Price)
}

func TestCreateSellOffer_TinyPriceKeepsFullPrecision(t *testing.T) {
	sdex := makeOfflineSDEX()

	// selling USD for BTC at 1/83000 BTC per USD, and at a level 0.1% higher
	offer, e := sdex.CreateSellOffer(testUSD, testBTC, 1.0/83000, 100, 0)
	require.NoError(t, e)
	requireOfferPrice(t, offer, xdr.Price{N: 1, D: 83000})
	assert.Equal(t, "100.0000000", offer.Amount)

	offer, e = sdex.CreateSellOffer(testUSD, testBTC, 1.001/83000, 100, 0)
	require.NoError(t, e)
	requireOfferPrice(t, offer, xdr.Price{N: 1001, D: 83000000})
}

func TestCreateBuyOffer_IsAnInvertedSellOffer(t *testing.T) {
	sdex := makeOfflineSDEX()

	// buying 0.5 BTC at 83000 USD each is placed as a sell offer of 41500 USD at 1/83000 BTC per USD
	offer, e := sdex.CreateBuyOffer(testBTC, testUSD, 83000, 0.5, 0)
	require.NoError(t, e)
	requireOfferPrice(t, offer, xdr.Price{N: 1, D: 83000})
	assert.Equal(t, "41500.0000000", offer.Amount)
	assert.Equal(t, "USD", offer.Selling.GetCode())
	assert.Equal(t, "BTC", offer.Buying.GetCode())
}

func TestModifySellOffer_KeepsOfferID(t *testing.T) {
	sdex := makeOfflineSDEX()
	existing := hProtocol.Offer{ID: 777, Selling: testUSD, Buying: testBTC}

	offer, e := sdex.ModifySellOffer(existing, 1.002/83000, 50, 0)
	require.NoError(t, e)
	requireOfferPrice(t, offer, xdr.Price{N: 501, D: 41500000})
	assert.Equal(t, int64(777), offer.OfferID)
}

func TestCreateSellOffer_UnrepresentablePrice(t *testing.T) {
	sdex := makeOfflineSDEX()

	// below 1/math.MaxInt32 the price cannot be stored on the ledger in any form
	_, e := sdex.CreateSellOffer(testUSD, testBTC, 1e-10, 100, 0)
	assert.Error(t, e)
}
