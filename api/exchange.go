package api

import (
	"fmt"
	"math"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stellar/kelp/model"
	"github.com/stellar/kelp/stellargohorizonclientv300/build"
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
	SubmitOps(ops []build.TransactionMutator, submitMode SubmitMode, asyncCallback func(hash string, e error)) error
	SubmitOpsSynch(ops []build.TransactionMutator, submitMode SubmitMode, asyncCallback func(hash string, e error)) error // forced synchronous version of SubmitOps
	GetBalanceHack(asset hProtocol.Asset) (*Balance, error)
	LoadOffersHack() ([]hProtocol.Offer, error)
	Constrainable
	OrderbookFetcher
	FillTrackable
}

// ConvertOperation2TM is a temporary adapter to support transitioning from the old Go SDK to the new SDK without having to bump the major version
func ConvertOperation2TM(ops []txnbuild.Operation) []build.TransactionMutator {
	muts := []build.TransactionMutator{}
	for _, o := range ops {
		var mob build.ManageOfferBuilder
		if mso, ok := o.(*txnbuild.ManageSellOffer); ok {
                        fmt.Println(mso);
			mob = build.ManageOffer(
				false,
				build.Amount(mso.Amount),
				build.Rate{
					Selling: build.Asset{Code: mso.Selling.GetCode(), Issuer: mso.Selling.GetIssuer(), Native: mso.Selling.IsNative()},
					Buying:  build.Asset{Code: mso.Buying.GetCode(), Issuer: mso.Buying.GetIssuer(), Native: mso.Buying.IsNative()},
					Price:   build.Price(mso.Price),
				},
				build.OfferID(mso.OfferID),
			)
			if mso.SourceAccount != "" {
				mob.Mutate(build.SourceAccount{AddressOrSeed: mso.SourceAccount})
			}
		} else if mso, ok := o.(*txnbuild.ManageBuyOffer); ok {
			fmt.Println(mso);
			mob = build.ManageOffer(
				true,
				build.Amount(mso.Amount),
				build.Rate{
					Selling: build.Asset{Code: mso.Selling.GetCode(), Issuer: mso.Selling.GetIssuer(), Native: mso.Selling.IsNative()},
					Buying:  build.Asset{Code: mso.Buying.GetCode(), Issuer: mso.Buying.GetIssuer(), Native: mso.Buying.IsNative()},
					Price:   build.Price(mso.Price),
				},
				build.OfferID(mso.OfferID),
			)
			if mso.SourceAccount != "" {
				mob.Mutate(build.SourceAccount{AddressOrSeed: mso.SourceAccount})
			}
		} else {
			panic(fmt.Sprintf("could not convert txnbuild.Operation to build.TransactionMutator: %v\n", o))
		}
		muts = append(muts, mob)
	}
	return muts
}

// ConvertTM2Operation is a temporary adapter to support transitioning from the old Go SDK to the new SDK without having to bump the major version
func ConvertTM2Operation(muts []build.TransactionMutator) []txnbuild.Operation {
	msos := ConvertTM2MSO(muts)
	return ConvertMSO2Ops(msos)
}

// ConvertTM2MSO converts mutators from the old SDK to ManageSellOffer ops in the new one.
func ConvertTM2MSO(muts []build.TransactionMutator) []*txnbuild.Operation {
	msos := []*txnbuild.Operation{}
	for _, m := range muts {
		var mso txnbuild.Operation
		if mob, ok := m.(build.ManageOfferBuilder); ok {
			mso = convertMOB2MSO(mob)
		} else if mob, ok := m.(*build.ManageOfferBuilder); ok {
			mso = convertMOB2MSO(*mob)
		} else {
			panic(fmt.Sprintf("could not convert build.TransactionMutator to txnbuild.Operation: %v (type=%T)\n", m, m))
		}
		msos = append(msos, &mso)
	}
	return msos
}

// ConvertMSO2Ops converts manage sell offers into Operations.
func ConvertMSO2Ops(msos []*txnbuild.Operation) []txnbuild.Operation {
	ops := []txnbuild.Operation{}
	for _, mso := range msos {
		ops = append(ops, *mso)
	}
        fmt.Println(ops)
	return ops
}

func convertMOB2MSO(mob build.ManageOfferBuilder) txnbuild.Operation {
    if (mob.PassiveOffer || (float64(mob.MO.Price.N)/float64(mob.MO.Price.D) < 0.001)) {
        fmt.Println("WARNING: flipping buy/sell because of tiny sell price (or passive hack)");
        var pricestr string
        if (mob.PassiveOffer) {
            fmt.Println("WARNING: PassiveOffer hack")
            fmt.Println(mob.PO.Price)
            fmt.Println(mob.PO.Amount)
            // flip the amount into terms of buying, and into 
            // decimal from stroops
            amtstr := fmt.Sprintf("%.7f",
                (float64(mob.PO.Amount) * float64(mob.PO.Price.D)) /
                (float64(mob.PO.Price.N) * math.Pow(10, 7)))
            fmt.Println(amtstr)
            pricestr = fmt.Sprintf("%.7f", float64(mob.PO.Price.N)/float64(mob.PO.Price.D))
            mso := &txnbuild.ManageBuyOffer{
                    Amount:  amtstr,
                    OfferID: int64(mob.MO.OfferId),
                    Price:   pricestr,
            }
            if mob.O.SourceAccount != nil {
                    mso.SourceAccount = mob.O.SourceAccount.Address()
            }

            if mob.PO.Buying.Type == xdr.AssetTypeAssetTypeNative {
                    mso.Buying = txnbuild.NativeAsset{}
            } else {
                    var tipe, code, issuer string
                    mob.PO.Buying.MustExtract(&tipe, &code, &issuer)
                    mso.Buying = txnbuild.CreditAsset{
                            Code:   code,
                            Issuer: issuer,
                    }
            }

            if mob.PO.Selling.Type == xdr.AssetTypeAssetTypeNative {
                    mso.Selling = txnbuild.NativeAsset{}
            } else {
                    var tipe, code, issuer string
                    mob.PO.Selling.MustExtract(&tipe, &code, &issuer)
                    mso.Selling = txnbuild.CreditAsset{
                            Code:   code,
                            Issuer: issuer,
                    }
            }
            return mso
        } else {
            pricestr = fmt.Sprintf("%.7f", float64(mob.MO.Price.D)/float64(mob.MO.Price.N))
            mso := &txnbuild.ManageBuyOffer{
                    Amount:  fmt.Sprintf("%.7f", float64(mob.MO.Amount)/math.Pow(10, 7)),
                    OfferID: int64(mob.MO.OfferId),
                    Price:   pricestr,
            }
            if mob.O.SourceAccount != nil {
                    mso.SourceAccount = mob.O.SourceAccount.Address()
            }

            if mob.MO.Buying.Type == xdr.AssetTypeAssetTypeNative {
                    mso.Buying = txnbuild.NativeAsset{}
            } else {
                    var tipe, code, issuer string
                    mob.MO.Buying.MustExtract(&tipe, &code, &issuer)
                    mso.Buying = txnbuild.CreditAsset{
                            Code:   code,
                            Issuer: issuer,
                    }
            }

            if mob.MO.Selling.Type == xdr.AssetTypeAssetTypeNative {
                    mso.Selling = txnbuild.NativeAsset{}
            } else {
                    var tipe, code, issuer string
                    mob.MO.Selling.MustExtract(&tipe, &code, &issuer)
                    mso.Selling = txnbuild.CreditAsset{
                            Code:   code,
                            Issuer: issuer,
                    }
            }
	    return mso
        }
    } else {
	mso := &txnbuild.ManageSellOffer{
		Amount:  fmt.Sprintf("%.7f", float64(mob.MO.Amount)/math.Pow(10, 7)),
		OfferID: int64(mob.MO.OfferId),
		Price:   fmt.Sprintf("%.7f", float64(mob.MO.Price.N)/float64(mob.MO.Price.D)),
	}
	if mob.O.SourceAccount != nil {
		mso.SourceAccount = mob.O.SourceAccount.Address()
	}

	if mob.MO.Buying.Type == xdr.AssetTypeAssetTypeNative {
		mso.Buying = txnbuild.NativeAsset{}
	} else {
		var tipe, code, issuer string
		mob.MO.Buying.MustExtract(&tipe, &code, &issuer)
		mso.Buying = txnbuild.CreditAsset{
			Code:   code,
			Issuer: issuer,
		}
	}

	if mob.MO.Selling.Type == xdr.AssetTypeAssetTypeNative {
		mso.Selling = txnbuild.NativeAsset{}
	} else {
		var tipe, code, issuer string
		mob.MO.Selling.MustExtract(&tipe, &code, &issuer)
		mso.Selling = txnbuild.CreditAsset{
			Code:   code,
			Issuer: issuer,
		}
	}

	return mso
    }
}
