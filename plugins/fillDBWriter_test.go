package plugins

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarketID(t *testing.T) {
	testCases := []struct {
		exchangeName string
		baseAsset    string
		quoteAsset   string
		wantMarketID string
	}{
		{
			exchangeName: "sdex",
			baseAsset:    "XLM",
			quoteAsset:   "USD",
			wantMarketID: "20ec53d283",
		}, {
			exchangeName: "sdex",
			baseAsset:    "XLM",
			quoteAsset:   "BTC",
			wantMarketID: "52c60e7e19",
		}, {
			exchangeName: "sdex",
			baseAsset:    "XLM",
			quoteAsset:   "USDC",
			wantMarketID: "0591dd7844",
		},
	}

	for _, k := range testCases {
		t.Run(fmt.Sprintf("%s_%s_%s", k.exchangeName, k.baseAsset, k.quoteAsset), func(t *testing.T) {
			marketID := MakeMarketID(k.exchangeName, k.baseAsset, k.quoteAsset)

			assert.Equal(t, k.wantMarketID, marketID)
		})
	}
}
