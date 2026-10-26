package api

import (
	hProtocol "github.com/stellar/go-stellar-sdk/protocols/horizon"
	"github.com/stellar/go-stellar-sdk/txnbuild"
	"github.com/stellar/kelp/model"
)

// TradeHistoryResult is the result of a GetTradeHistory call
// this should be the same object as TradesResult but it's a separate object for backwards compatibility
type TradeHistoryResult struct {
	Cursor interface{}
	Trades []model.Trade
}

// FillTracker knows how to track fills against open orders
type FillTracker interface {
	GetPair() (pair *model.TradingPair)
	// TrackFills should be executed in a new thread
	TrackFills() error
	IsRunningInBackground() bool
	FillTrackSingleIteration() ([]model.Trade, error)
	RegisterHandler(handler FillHandler)
	NumHandlers() uint8
}

// FillHandler is invoked by the FillTracker (once registered) anytime an order is filled
type FillHandler interface {
	HandleFill(trade model.Trade) error
}

// TradeFetcher is the common method between FillTrackable and exchange
// temporarily extracted out from TradeAPI so SDEX has the flexibility to only implement this rather than exchange and FillTrackable
type TradeFetcher interface {
	GetTradeHistory(pair model.TradingPair, maybeCursorStart interface{}, maybeCursorEnd interface{}) (*TradeHistoryResult, error)
}

// FillTrackable enables any implementing exchange to support fill tracking
type FillTrackable interface {
	TradeFetcher
	GetLatestTradeCursor() (interface{}, error)
}

// Constrainable extracts out the method that SDEX can implement for now
type Constrainable interface {
	// return nil if the constraint does not exist for the exchange
	GetOrderConstraints(pair *model.TradingPair) *model.OrderConstraints

	OverrideOrderConstraints(pair *model.TradingPair, override *model.OrderConstraintsOverride)
}

// OrderbookFetcher extracts out the method that should go into ExchangeShim for now
type OrderbookFetcher interface {
	GetOrderBook(pair *model.TradingPair, maxCount int32) (*model.OrderBook, error)
}

// Balance repesents various aspects of an asset's balance
type Balance struct {
	Balance float64
	Trust   float64
	Reserve float64
}

// ExchangeShim is the interface the trader uses to place orders on SDEX
type ExchangeShim interface {
	SubmitOps(ops []txnbuild.Operation, submitMode SubmitMode, asyncCallback func(hash string, e error)) error
	SubmitOpsSynch(ops []txnbuild.Operation, submitMode SubmitMode, asyncCallback func(hash string, e error)) error // forced synchronous version of SubmitOps
	GetBalanceHack(asset hProtocol.Asset) (*Balance, error)
	LoadOffersHack() ([]hProtocol.Offer, error)
	Constrainable
	OrderbookFetcher
	FillTrackable
}
