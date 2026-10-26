package api

import (
	"fmt"
	"math"

	"github.com/stellar/go-stellar-sdk/amount"
	"github.com/stellar/go-stellar-sdk/price"
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

// tinyPriceThreshold is the sell price below which a sell offer is flipped into the equivalent buy offer
const tinyPriceThreshold = 0.001

// NormalizeOfferOps rewrites offer operations before they are submitted:
//   - a sell offer priced below tinyPriceThreshold is flipped into a buy offer with the inverted price, keeping the
//     amount, offer ID and source account. A tiny sell price can only be expressed coarsely as a ratio of 32-bit
//     integers, whereas its inverse is a large number that can be expressed precisely.
//   - any other sell offer keeps its type, with its price rounded to 7 decimal places.
//   - a buy offer is rewritten with its amount divided by its price, its price rounded to 7 decimal places, and its
//     offer ID reset to 0 (so it is always placed as a new offer).
//   - any other operation is returned unchanged.
//
// SDEX applies this to every submission, and the trader also applies it to the strategy's update operations before
// submitting them, so a tiny-priced sell offer from an update goes through both of the first and third rules.
// This reproduces, operation for operation, the conversion that previously went through the deprecated "build"
// package of the old stellar/go SDK.
func NormalizeOfferOps(ops []txnbuild.Operation) ([]txnbuild.Operation, error) {
	normalized := []txnbuild.Operation{}
	for _, o := range ops {
		switch op := o.(type) {
		case *txnbuild.ManageSellOffer:
			n, e := normalizeSellOffer(op)
			if e != nil {
				return nil, e
			}
			normalized = append(normalized, n)
		case *txnbuild.ManageBuyOffer:
			n, e := normalizeBuyOffer(op)
			if e != nil {
				return nil, e
			}
			normalized = append(normalized, n)
		default:
			normalized = append(normalized, o)
		}
	}
	return normalized, nil
}

func normalizeSellOffer(op *txnbuild.ManageSellOffer) (txnbuild.Operation, error) {
	stroops, e := amount.Parse(op.Amount)
	if e != nil {
		return nil, fmt.Errorf("could not parse amount of sell offer (%+v): %s", *op, e)
	}
	amountString := fmt.Sprintf("%.7f", float64(stroops)/math.Pow(10, 7))
	p := op.Price

	if float64(p.N)/float64(p.D) < tinyPriceThreshold {
		invertedPrice, e := price.Parse(fmt.Sprintf("%.7f", float64(p.D)/float64(p.N)))
		if e != nil {
			return nil, fmt.Errorf("could not invert price of sell offer (%+v): %s", *op, e)
		}
		return &txnbuild.ManageBuyOffer{
			Selling:       op.Selling,
			Buying:        op.Buying,
			Amount:        amountString,
			Price:         invertedPrice,
			OfferID:       op.OfferID,
			SourceAccount: op.SourceAccount,
		}, nil
	}

	roundedPrice, e := price.Parse(fmt.Sprintf("%.7f", float64(p.N)/float64(p.D)))
	if e != nil {
		return nil, fmt.Errorf("could not round price of sell offer (%+v): %s", *op, e)
	}
	return &txnbuild.ManageSellOffer{
		Selling:       op.Selling,
		Buying:        op.Buying,
		Amount:        amountString,
		Price:         roundedPrice,
		OfferID:       op.OfferID,
		SourceAccount: op.SourceAccount,
	}, nil
}

func normalizeBuyOffer(op *txnbuild.ManageBuyOffer) (txnbuild.Operation, error) {
	stroops, e := amount.Parse(op.Amount)
	if e != nil {
		return nil, fmt.Errorf("could not parse amount of buy offer (%+v): %s", *op, e)
	}
	p := op.Price

	roundedPrice, e := price.Parse(fmt.Sprintf("%.7f", float64(p.N)/float64(p.D)))
	if e != nil {
		return nil, fmt.Errorf("could not round price of buy offer (%+v): %s", *op, e)
	}
	return &txnbuild.ManageBuyOffer{
		Selling:       op.Selling,
		Buying:        op.Buying,
		Amount:        fmt.Sprintf("%.7f", (float64(stroops)*float64(p.D))/(float64(p.N)*math.Pow(10, 7))),
		Price:         roundedPrice,
		SourceAccount: op.SourceAccount,
	}, nil
}
