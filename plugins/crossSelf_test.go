package plugins

import (
	"testing"

	"github.com/stellar/kelp/api"
	"github.com/stellar/kelp/model"
	"github.com/stretchr/testify/assert"
)

func floatPtr(f float64) *float64 {
	return &f
}

func levelsAt(prices ...float64) []api.Level {
	levels := []api.Level{}
	for _, p := range prices {
		levels = append(levels, api.Level{
			Price:  *model.NumberFromFloat(p, 13),
			Amount: *model.NumberFromFloat(1.0, 7),
		})
	}
	return levels
}

func TestWouldCrossOpposite(t *testing.T) {
	testCases := []struct {
		name          string
		price         float64
		oppositePrice *float64
		want          bool
	}{
		{"no opposite price", 100.0, nil, false},
		{"zero opposite price", 100.0, floatPtr(0), false},
		{"zero price", 0, floatPtr(0.01), false},
		// buy 1 base for 0.0099 quote (selling quote at 1/0.0099) vs sell 1 base for 0.01 quote: spread, no cross
		{"spread", 1 / 0.0099, floatPtr(0.01), false},
		// prices exactly meet: the offers match, so this crosses
		{"equal", 100.0, floatPtr(0.01), true},
		// buy price above the sell price
		{"crossed", 1 / 0.0101, floatPtr(0.01), true},
	}

	for _, kase := range testCases {
		t.Run(kase.name, func(t *testing.T) {
			assert.Equal(t, kase.want, wouldCrossOpposite(kase.price, kase.oppositePrice))
		})
	}
}

// TestComputePriceTriggers_CrossSelfFromLog reproduces the op_cross_self failure seen on BLNT/USDC: the existing buy
// offer was within PRICE_TOLERANCE of its target so it was kept, while the sell side's new top offer was priced below it.
func TestComputePriceTriggers_CrossSelfFromLog(t *testing.T) {
	priceTolerance := 0.03
	sellTopPrice := 0.0063938443973         // new sell level 1 (quote per base)
	buyTargetPrice := 156.4473516007377     // buy level 1 in the buy side's frame (base per quote)
	existingBuyOfferPrice := 1 / 0.00641610 // existing buy level 1, priced 0.00641610 quote per base
	existingSellOfferPrice := 0.00651371    // an existing sell offer that sits above the new buy target
	buyTopPrice := buyTargetPrice

	// buy side: within tolerance, but crosses the new sell top so it must be repriced
	priceTrigger, crossTrigger := computePriceTriggers(existingBuyOfferPrice, buyTargetPrice, priceTolerance, &sellTopPrice)
	assert.False(t, priceTrigger, "existing buy offer is within price tolerance")
	assert.True(t, crossTrigger, "existing buy offer crosses the new sell top")

	// without knowing the opposite side we would (wrongly) keep it, which is the old behavior
	priceTrigger, crossTrigger = computePriceTriggers(existingBuyOfferPrice, buyTargetPrice, priceTolerance, nil)
	assert.False(t, priceTrigger)
	assert.False(t, crossTrigger)

	// once repriced to its target, the buy offer no longer crosses the sell top
	assert.False(t, wouldCrossOpposite(buyTargetPrice, &sellTopPrice))

	// sell side: an existing sell offer above the new buy top is left alone
	_, crossTrigger = computePriceTriggers(existingSellOfferPrice, 0.0064130342065, priceTolerance, &buyTopPrice)
	assert.False(t, crossTrigger)
}

func TestComputePriceTriggers_OutsideTolerance(t *testing.T) {
	priceTrigger, crossTrigger := computePriceTriggers(1.04, 1.0, 0.03, nil)
	assert.True(t, priceTrigger)
	assert.False(t, crossTrigger)

	priceTrigger, _ = computePriceTriggers(0.96, 1.0, 0.03, nil)
	assert.True(t, priceTrigger)

	priceTrigger, _ = computePriceTriggers(1.02, 1.0, 0.03, nil)
	assert.False(t, priceTrigger)
}

func TestDesiredTopPrice(t *testing.T) {
	s := &sellSideStrategy{}
	assert.Nil(t, s.desiredTopPrice())

	s.desiredLevels = levelsAt(0.0064130342065, 0.0063938443973, 0.0064322701165)
	top := s.desiredTopPrice()
	if assert.NotNil(t, top) {
		assert.InDelta(t, 0.0063938443973, *top, 1e-13)
	}
}

func TestComposeStrategyExchangeTopPrices(t *testing.T) {
	buy := &sellSideStrategy{desiredLevels: levelsAt(156.4473516007377, 156.9166710303976)}
	sell := &sellSideStrategy{desiredLevels: levelsAt(0.0063938443973, 0.0064130342065)}
	s := &composeStrategy{buyStrat: buy, sellStrat: sell}

	s.exchangeTopPrices()
	if assert.NotNil(t, buy.oppositeTopPrice) && assert.NotNil(t, sell.oppositeTopPrice) {
		assert.InDelta(t, 0.0063938443973, *buy.oppositeTopPrice, 1e-13)
		assert.InDelta(t, 156.4473516007377, *sell.oppositeTopPrice, 1e-10)
	}

	// a side with nothing to place clears the opposite bound on the next iteration
	sell.desiredLevels = []api.Level{}
	s.exchangeTopPrices()
	assert.Nil(t, buy.oppositeTopPrice)
	assert.NotNil(t, sell.oppositeTopPrice)
}

func TestComposeStrategyExchangeTopPrices_NonCrossAwareSide(t *testing.T) {
	buy := &sellSideStrategy{desiredLevels: levelsAt(156.4473516007377)}
	s := &composeStrategy{buyStrat: buy, sellStrat: &deleteSideStrategy{}}

	s.exchangeTopPrices()
	assert.Nil(t, buy.oppositeTopPrice)
}
