package plugins

import (
	"fmt"
	"log"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/config"
	"github.com/stellar/kelp/api"
	"github.com/stellar/kelp/model"
	"github.com/stellar/kelp/support/utils"
)

// strategyFactoryData is a data container that has all the information needed to make a strategy
type strategyFactoryData struct {
	sdex            *SDEX
	ieif            *IEIF
	tradingPair     *model.TradingPair
	assetBase       *hProtocol.Asset
	assetQuote      *hProtocol.Asset
	stratConfigPath string
	filterFactory   *FilterFactory
}

// StrategyContainer contains the strategy factory method along with some metadata
type StrategyContainer struct {
	SortOrder   uint8
	Description string
	NeedsConfig bool
	Complexity  string
	makeFn      func(strategyFactoryData strategyFactoryData) (api.Strategy, error)
}

// strategies is a map of all the strategies available
var strategies = map[string]StrategyContainer{
	"buysell": {
		SortOrder:   1,
		Description: "Creates buy and sell offers based on a reference price with a pre-specified liquidity depth",
		NeedsConfig: true,
		Complexity:  "Beginner",
		makeFn: func(strategyFactoryData strategyFactoryData) (api.Strategy, error) {
			var cfg BuySellConfig
			err := config.Read(strategyFactoryData.stratConfigPath, &cfg)
			utils.CheckConfigError(cfg, err, strategyFactoryData.stratConfigPath)
			utils.LogConfig(cfg)
			s, e := makeBuySellStrategy(strategyFactoryData.sdex, strategyFactoryData.tradingPair, strategyFactoryData.ieif, strategyFactoryData.assetBase, strategyFactoryData.assetQuote, &cfg)
			if e != nil {
				return nil, fmt.Errorf("makeFn failed: %s", e)
			}
			return s, nil
		},
	},
	"sell": {
		SortOrder:   0,
		Description: "Creates sell offers based on a reference price with a pre-specified liquidity depth",
		NeedsConfig: true,
		Complexity:  "Beginner",
		makeFn: func(strategyFactoryData strategyFactoryData) (api.Strategy, error) {
			var cfg sellConfig
			err := config.Read(strategyFactoryData.stratConfigPath, &cfg)
			utils.CheckConfigError(cfg, err, strategyFactoryData.stratConfigPath)
			utils.LogConfig(cfg)
			s, e := makeSellStrategy(strategyFactoryData.sdex, strategyFactoryData.tradingPair, strategyFactoryData.ieif, strategyFactoryData.assetBase, strategyFactoryData.assetQuote, &cfg)
			if e != nil {
				return nil, fmt.Errorf("makeFn failed: %s", e)
			}
			return s, nil
		},
	},
	"balanced": {
		SortOrder:   4,
		Description: "Dynamically prices two tokens based on their relative demand",
		NeedsConfig: true,
		Complexity:  "Intermediate",
		makeFn: func(strategyFactoryData strategyFactoryData) (api.Strategy, error) {
			var cfg balancedConfig
			err := config.Read(strategyFactoryData.stratConfigPath, &cfg)
			utils.CheckConfigError(cfg, err, strategyFactoryData.stratConfigPath)
			utils.LogConfig(cfg)
			return makeBalancedStrategy(strategyFactoryData.sdex, strategyFactoryData.tradingPair, strategyFactoryData.ieif, strategyFactoryData.assetBase, strategyFactoryData.assetQuote, &cfg), nil
		},
	},
	"delete": {
		SortOrder:   3,
		Description: "Deletes all orders for the configured orderbook",
		NeedsConfig: false,
		Complexity:  "Beginner",
		makeFn: func(strategyFactoryData strategyFactoryData) (api.Strategy, error) {
			return makeDeleteStrategy(strategyFactoryData.sdex, strategyFactoryData.assetBase, strategyFactoryData.assetQuote), nil
		},
	},
	"pendulum": {
		SortOrder:   2,
		Description: "Oscillating bids and asks like a pendulum based on last trade price as the equilibrium poistion",
		NeedsConfig: true,
		Complexity:  "Beginner",
		makeFn: func(strategyFactoryData strategyFactoryData) (api.Strategy, error) {
			var cfg pendulumConfig
			err := config.Read(strategyFactoryData.stratConfigPath, &cfg)
			utils.CheckConfigError(cfg, err, strategyFactoryData.stratConfigPath)
			utils.LogConfig(cfg)
			return makePendulumStrategy(
				strategyFactoryData.sdex,
				strategyFactoryData.ieif,
				strategyFactoryData.assetBase,
				strategyFactoryData.assetQuote,
				&cfg,
				strategyFactoryData.tradingPair,
			), nil
		},
	},
	"sell_twap": {
		SortOrder:   5,
		Description: "Creates sell offers by distributing orders over time for a given day using a twap metric",
		NeedsConfig: true,
		Complexity:  "Intermediate",
		makeFn: func(strategyFactoryData strategyFactoryData) (api.Strategy, error) {
			var cfg sellTwapConfig
			err := config.Read(strategyFactoryData.stratConfigPath, &cfg)
			utils.CheckConfigError(cfg, err, strategyFactoryData.stratConfigPath)
			utils.LogConfig(cfg)
			s, e := makeSellTwapStrategy(
				strategyFactoryData.sdex,
				strategyFactoryData.tradingPair,
				strategyFactoryData.ieif,
				strategyFactoryData.assetBase,
				strategyFactoryData.assetQuote,
				strategyFactoryData.filterFactory,
				&cfg,
			)
			if e != nil {
				return nil, fmt.Errorf("makeFn failed: %s", e)
			}
			return s, nil
		},
	},
	"buy_twap": {
		SortOrder:   6,
		Description: "Creates buy offers by distributing orders over time for a given day using a twap metric",
		NeedsConfig: true,
		Complexity:  "Intermediate",
		makeFn: func(strategyFactoryData strategyFactoryData) (api.Strategy, error) {
			// reuse the sellTwapConfig struct since we need the same info for buyTwap
			var cfg sellTwapConfig
			err := config.Read(strategyFactoryData.stratConfigPath, &cfg)
			utils.CheckConfigError(cfg, err, strategyFactoryData.stratConfigPath)
			utils.LogConfig(cfg)
			s, e := makeBuyTwapStrategy(
				strategyFactoryData.sdex,
				strategyFactoryData.tradingPair,
				strategyFactoryData.ieif,
				strategyFactoryData.assetBase,
				strategyFactoryData.assetQuote,
				strategyFactoryData.filterFactory,
				&cfg,
			)
			if e != nil {
				return nil, fmt.Errorf("make Fn failed: %s", e)
			}
			return s, nil
		},
	},
}

// MakeStrategy makes a strategy
func MakeStrategy(
	sdex *SDEX,
	ieif *IEIF,
	tradingPair *model.TradingPair,
	assetBase *hProtocol.Asset,
	assetQuote *hProtocol.Asset,
	strategy string,
	stratConfigPath string,
	filterFactory *FilterFactory,
) (api.Strategy, error) {
	log.Printf("Making strategy: %s\n", strategy)
	if s, ok := strategies[strategy]; ok {
		if s.NeedsConfig && stratConfigPath == "" {
			return nil, fmt.Errorf("the '%s' strategy needs a config file", strategy)
		}

		s, e := s.makeFn(strategyFactoryData{
			sdex:            sdex,
			ieif:            ieif,
			tradingPair:     tradingPair,
			assetBase:       assetBase,
			assetQuote:      assetQuote,
			stratConfigPath: stratConfigPath,
			filterFactory:   filterFactory,
		})
		if e != nil {
			return nil, fmt.Errorf("cannot make '%s' strategy: %s", strategy, e)
		}
		return s, nil
	}

	return nil, fmt.Errorf("invalid strategy type: %s", strategy)
}

// Strategies returns the list of strategies along with metadata
func Strategies() map[string]StrategyContainer {
	return strategies
}
