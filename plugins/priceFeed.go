package plugins

import (
	"fmt"

	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
	"github.com/stellar/kelp/api"
)

// privateSdexHack is a temporary hack struct for SDEX price feeds pending refactor
type privateSdexHack struct {
	API     *horizonclient.Client
	Ieif    *IEIF
	Network string
}

// privateSdexHackVar is a temporary hack variable for SDEX price feeds pending refactor
var privateSdexHackVar *privateSdexHack

// SetPrivateSdexHack sets the privateSdexHack variable which is temporary until the pending SDEX price feed refactor
func SetPrivateSdexHack(api *horizonclient.Client, ieif *IEIF, network string) error {
	if privateSdexHackVar != nil {
		return fmt.Errorf("privateSdexHack is already set: %+v", privateSdexHackVar)
	}

	privateSdexHackVar = &privateSdexHack{
		API:     api,
		Ieif:    ieif,
		Network: network,
	}
	return nil
}

// MakePriceFeed makes a PriceFeed
func MakePriceFeed(feedType string, url string) (api.PriceFeed, error) {
	switch feedType {
	case "crypto":
		return newCMCFeed(url), nil
	case "fiat":
		return newFiatFeed(url), nil
	case "fiat-oxr":
		return newFiatFeedOxr(url), nil
	case "fixed":
		return newFixedFeed(url)
	case "sdex":
		sdex, e := makeSDEXFeed(url)
		if e != nil {
			return nil, fmt.Errorf("error occurred while making the SDEX price feed: %s", e)
		}
		return sdex, nil
	case "function":
		fnFeed, e := makeFunctionPriceFeed(url)
		if e != nil {
			return nil, fmt.Errorf("error while making function feed for URL '%s': %s", url, e)
		}
		return fnFeed, nil
	}
	return nil, fmt.Errorf("unable to make price feed for feedType=%s and url=%s", feedType, url)
}

// MakeFeedPair is the factory method that we expose
func MakeFeedPair(dataTypeA, dataFeedAUrl, dataTypeB, dataFeedBUrl string) (*api.FeedPair, error) {
	feedA, e := MakePriceFeed(dataTypeA, dataFeedAUrl)
	if e != nil {
		return nil, fmt.Errorf("cannot make a feed pair because of an error when making priceFeed A: %s", e)
	}

	feedB, e := MakePriceFeed(dataTypeB, dataFeedBUrl)
	if e != nil {
		return nil, fmt.Errorf("cannot make a feed pair because of an error when making priceFeed B: %s", e)
	}

	return &api.FeedPair{
		FeedA: feedA,
		FeedB: feedB,
	}, nil
}
