package plugins

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikhilsaraf/go-tools/multithreading"
	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
	hProtocol "github.com/stellar/go-stellar-sdk/protocols/horizon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/kelp/model"
)

const historyTestAccount = "GBWX2DWBW7MFJR2ZBHJHL4MQ4ULTYIP5WBWQAHHZ3GGR5Q64OCWYSBOV"

// another asset this account trades in, through a different bot
var testBLND = hProtocol.Asset{Type: "credit_alphanum4", Code: "BLND", Issuer: testBTC.Issuer}

// fakeHorizon serves the trades of one account, in every pair (like horizon's /accounts/{id}/trades), plus the effects
// used to tell whether each of our trades was a buy or a sell. It fails the test if any page of trades is read twice
// (polling the empty page past the newest trade is expected).
type fakeHorizon struct {
	t      *testing.T
	trades []map[string]interface{}

	lock             sync.Mutex
	cursorsRequested map[string]int
}

func newFakeHorizon(t *testing.T) (*fakeHorizon, *httptest.Server) {
	f := &fakeHorizon{t: t, cursorsRequested: map[string]int{}}
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	return f, server
}

// addTrade appends a trade by our account selling baseAsset for quoteAsset, with paging tokens increasing in order
func (f *fakeHorizon) addTrade(serverURL string, baseAsset hProtocol.Asset, quoteAsset hProtocol.Asset) string {
	opID := 1000 + len(f.trades)
	id := fmt.Sprintf("%d-0", opID)
	f.trades = append(f.trades, map[string]interface{}{
		"_links": map[string]interface{}{
			"operation": map[string]string{"href": fmt.Sprintf("%s/operations/%d", serverURL, opID)},
		},
		"id":                   id,
		"paging_token":         id,
		"ledger_close_time":    time.Unix(1700000000+int64(opID), 0).UTC().Format(time.RFC3339),
		"trade_type":           "orderbook",
		"base_account":         historyTestAccount,
		"base_amount":          "1.0000000",
		"base_asset_type":      baseAsset.Type,
		"base_asset_code":      baseAsset.Code,
		"base_asset_issuer":    baseAsset.Issuer,
		"counter_account":      "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
		"counter_amount":       "2.0000000",
		"counter_asset_type":   quoteAsset.Type,
		"counter_asset_code":   quoteAsset.Code,
		"counter_asset_issuer": quoteAsset.Issuer,
		"base_is_seller":       true,
		"price":                map[string]string{"n": "2", "d": "1"},
	})
	return id
}

func (f *fakeHorizon) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/hal+json")

	switch {
	case strings.HasPrefix(r.URL.Path, "/accounts/") && strings.HasSuffix(r.URL.Path, "/trades"):
		f.serveTrades(w, r)
	case strings.HasPrefix(r.URL.Path, "/operations/") && strings.HasSuffix(r.URL.Path, "/effects"):
		f.serveEffects(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"title": "Resource Missing", "status": 404}`))
	}
}

func (f *fakeHorizon) serveTrades(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	records := []map[string]interface{}{}
	for _, trade := range f.trades {
		if cursor == "" || trade["paging_token"].(string) > cursor {
			records = append(records, trade)
		}
		if len(records) == limit {
			break
		}
	}

	f.lock.Lock()
	f.cursorsRequested[cursor]++
	repeated := f.cursorsRequested[cursor] > 1 && len(records) > 0
	f.lock.Unlock()
	if repeated {
		// answer like horizon's rate limiter so a looping caller stops instead of hanging the test
		f.t.Errorf("trades page requested again for cursor %q", cursor)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"title": "Rate Limit Exceeded", "status": 429, "detail": "rate limit exceeded"}`))
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"_links":    map[string]interface{}{},
		"_embedded": map[string]interface{}{"records": records},
	})
}

// serveEffects reports our account selling the base asset of the trade for its counter asset
func (f *fakeHorizon) serveEffects(w http.ResponseWriter, r *http.Request) {
	opPath := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/operations/"), "/effects")
	records := []map[string]interface{}{}
	for _, trade := range f.trades {
		if strings.TrimSuffix(trade["id"].(string), "-0") == opPath {
			records = append(records, map[string]interface{}{
				"type":                "trade",
				"account":             historyTestAccount,
				"sold_asset_type":     trade["base_asset_type"],
				"sold_asset_code":     trade["base_asset_code"],
				"sold_asset_issuer":   trade["base_asset_issuer"],
				"bought_asset_type":   trade["counter_asset_type"],
				"bought_asset_code":   trade["counter_asset_code"],
				"bought_asset_issuer": trade["counter_asset_issuer"],
			})
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"_links":    map[string]interface{}{"next": map[string]string{"href": r.URL.String()}},
		"_embedded": map[string]interface{}{"records": records},
	})
}

// makeHistorySDEX returns an SDEX for the BTC/USD pair that reads trades from the fake horizon server
func makeHistorySDEX(serverURL string) *SDEX {
	pair := &model.TradingPair{Base: model.Asset("BTC"), Quote: model.Asset("USD")}
	return &SDEX{
		API:            &horizonclient.Client{HorizonURL: serverURL + "/", HTTP: http.DefaultClient},
		TradingAccount: historyTestAccount,
		pair:           pair,
		assetMap:       map[model.Asset]hProtocol.Asset{pair.Base: testBTC, pair.Quote: testUSD},
	}
}

// More than a full page of trades in another pair comes before our own trades. This used to request the first page
// again forever, since the cursor only moved past trades in the bot's own pair.
func TestGetTradeHistory_SkipsPagesOfOtherPairsTrades(t *testing.T) {
	horizon, server := newFakeHorizon(t)
	for i := 0; i < 2*maxPageLimit+50; i++ {
		horizon.addTrade(server.URL, testBLND, testUSD)
	}
	first := horizon.addTrade(server.URL, testBTC, testUSD)
	second := horizon.addTrade(server.URL, testBTC, testUSD)

	sdex := makeHistorySDEX(server.URL)
	result, e := sdex.GetTradeHistory(*sdex.pair, "0", nil)
	require.NoError(t, e)

	if assert.Len(t, result.Trades, 2) {
		assert.Equal(t, first, result.Trades[0].TransactionID.String())
		assert.Equal(t, second, result.Trades[1].TransactionID.String())
		assert.Equal(t, model.OrderActionSell, result.Trades[0].OrderAction)
	}
	assert.Equal(t, second, result.Cursor)
}

// with no trades at all in the bot's pair, the cursor still moves past everything that was read
func TestGetTradeHistory_NoTradesInPairStillAdvancesCursor(t *testing.T) {
	horizon, server := newFakeHorizon(t)
	var last string
	for i := 0; i < 2*maxPageLimit; i++ {
		last = horizon.addTrade(server.URL, testBLND, testUSD)
	}

	sdex := makeHistorySDEX(server.URL)
	result, e := sdex.GetTradeHistory(*sdex.pair, "0", nil)
	require.NoError(t, e)
	assert.Len(t, result.Trades, 0)
	assert.Equal(t, last, result.Cursor)
}

// the fill tracker keeps the advanced cursor even when none of the trades were in its pair, so they are not read again
func TestFillTracker_AdvancesCursorPastOtherPairsTrades(t *testing.T) {
	horizon, server := newFakeHorizon(t)
	var last string
	for i := 0; i < maxPageLimit+10; i++ {
		last = horizon.addTrade(server.URL, testBLND, testUSD)
	}

	sdex := makeHistorySDEX(server.URL)
	tracker := MakeFillTracker(sdex.pair, multithreading.MakeThreadTracker(), sdex, 0, 0, "0")
	trades, e := tracker.FillTrackSingleIteration()
	require.NoError(t, e)
	assert.Len(t, trades, 0)

	// a second iteration starts from the advanced cursor, so no page is read twice (the fake server checks this)
	trades, e = tracker.FillTrackSingleIteration()
	require.NoError(t, e)
	assert.Len(t, trades, 0)
	assert.Equal(t, last, tracker.(*FillTracker).lastCursor)
}

// more trades in our pair than one call returns: the cursor stops at the last trade returned, so none are skipped
func TestGetTradeHistory_FetchLimitDoesNotSkipTrades(t *testing.T) {
	horizon, server := newFakeHorizon(t)
	ids := []string{}
	for i := 0; i < sdexTradesFetchLimit+50; i++ {
		ids = append(ids, horizon.addTrade(server.URL, testBTC, testUSD))
		// interleave trades in another pair so the pages and the fetch limit do not line up
		if i%3 == 0 {
			horizon.addTrade(server.URL, testBLND, testUSD)
		}
	}

	sdex := makeHistorySDEX(server.URL)
	got := []string{}
	cursor := interface{}("0")
	for call := 0; call < 5 && len(got) < len(ids); call++ {
		result, e := sdex.GetTradeHistory(*sdex.pair, cursor, nil)
		require.NoError(t, e)
		require.LessOrEqual(t, len(result.Trades), sdexTradesFetchLimit)
		for _, trade := range result.Trades {
			got = append(got, trade.TransactionID.String())
		}
		cursor = result.Cursor
	}
	assert.Equal(t, ids, got)
}
