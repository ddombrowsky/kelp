package terminator

import (
	"encoding/base64"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/kelp/model"
)

func encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// makeBotData returns the manage-data entries that describe one bot, keyed the same way as the account data
func makeBotData(bk model.BotKey, lastUpdated string) map[string]string {
	return map[string]string{
		bk.FullKey(0): encode(lastUpdated),
		bk.FullKey(1): encode(bk.AssetBaseCode),
		bk.FullKey(2): encode(bk.AssetBaseIssuer),
		bk.FullKey(3): encode(bk.AssetQuoteCode),
		bk.FullKey(4): encode(bk.AssetQuoteIssuer),
	}
}

// fullKey returns the manage-data key for one part of a bot key
func fullKey(bk model.BotKey, part int) string {
	return bk.FullKey(part)
}

func mergeData(maps ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

// sortBotList orders bots by lastUpdated since reconstructBotList builds its result from a map
func sortBotList(l []botKeyPair) {
	sort.Slice(l, func(i, j int) bool { return l[i].lastUpdated < l[j].lastUpdated })
}

// bot keys are built fresh each time because BotKey caches its key and hash internally on first use,
// which would make it unequal to the keys that reconstructBotList decodes
func botXlmCoupon() model.BotKey {
	return model.BotKey{
		AssetBaseCode:    "native",
		AssetBaseIssuer:  "",
		AssetQuoteCode:   "COUPON",
		AssetQuoteIssuer: "GBMMZMK2DC4FFP4CAI6KCVNCQ7WLO5A7DQU7EC7WGHRDQBZB763X4OQI",
	}
}

func botUsdcXlm() model.BotKey {
	return model.BotKey{
		AssetBaseCode:    "USDC",
		AssetBaseIssuer:  "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
		AssetQuoteCode:   "native",
		AssetQuoteIssuer: "",
	}
}

func TestReconstructBotList(t *testing.T) {
	testCases := []struct {
		name string
		data map[string]string
		want []botKeyPair
	}{
		{
			name: "no data",
			data: map[string]string{},
			want: []botKeyPair{},
		}, {
			name: "one bot",
			data: makeBotData(botXlmCoupon(), "1000"),
			want: []botKeyPair{{dataKey: botXlmCoupon(), lastUpdated: 1000}},
		}, {
			name: "two bots",
			data: mergeData(makeBotData(botXlmCoupon(), "1000"), makeBotData(botUsdcXlm(), "2000")),
			want: []botKeyPair{
				{dataKey: botXlmCoupon(), lastUpdated: 1000},
				{dataKey: botUsdcXlm(), lastUpdated: 2000},
			},
		}, {
			name: "ignores data entries that are not bot keys",
			data: mergeData(makeBotData(botXlmCoupon(), "1000"), map[string]string{
				terminatorKey: encode("5000"),
				"other":       encode("value"),
			}),
			want: []botKeyPair{{dataKey: botXlmCoupon(), lastUpdated: 1000}},
		}, {
			name: "ignores unknown key parts",
			data: mergeData(makeBotData(botXlmCoupon(), "1000"), map[string]string{
				fullKey(botXlmCoupon(), 9): encode("ignored"),
			}),
			want: []botKeyPair{{dataKey: botXlmCoupon(), lastUpdated: 1000}},
		},
	}

	for _, k := range testCases {
		t.Run(k.name, func(t *testing.T) {
			got, e := reconstructBotList(k.data)
			if !assert.NoError(t, e) {
				return
			}

			sortBotList(got)
			assert.Equal(t, k.want, got)
		})
	}
}

func TestReconstructBotList_Errors(t *testing.T) {
	testCases := []struct {
		name string
		data map[string]string
	}{
		{
			name: "value is not base64",
			data: map[string]string{fullKey(botXlmCoupon(), 1): "not base64!"},
		}, {
			name: "timestamp is not a number",
			data: makeBotData(botXlmCoupon(), "yesterday"),
		},
	}

	for _, k := range testCases {
		t.Run(k.name, func(t *testing.T) {
			_, e := reconstructBotList(k.data)
			assert.Error(t, e)
		})
	}
}

func TestExcludeActiveBots(t *testing.T) {
	old := botKeyPair{dataKey: botXlmCoupon(), lastUpdated: 1000}
	recent := botKeyPair{dataKey: botUsdcXlm(), lastUpdated: 3000}
	botList := []botKeyPair{old, recent}

	testCases := []struct {
		name         string
		cutoffMillis int64
		want         []botKeyPair
	}{
		{
			name:         "all bots active",
			cutoffMillis: 500,
			want:         []botKeyPair{},
		}, {
			name:         "bot updated exactly at the cutoff is still active",
			cutoffMillis: 1000,
			want:         []botKeyPair{},
		}, {
			name:         "only the bot updated before the cutoff is inactive",
			cutoffMillis: 2000,
			want:         []botKeyPair{old},
		}, {
			name:         "all bots inactive",
			cutoffMillis: 5000,
			want:         []botKeyPair{old, recent},
		},
	}

	for _, k := range testCases {
		t.Run(k.name, func(t *testing.T) {
			assert.Equal(t, k.want, excludeActiveBots(botList, k.cutoffMillis))
		})
	}

	assert.Equal(t, []botKeyPair{}, excludeActiveBots([]botKeyPair{}, 2000))
}
