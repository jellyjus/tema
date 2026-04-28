package fetcher

import (
	"context"
	"fmt"
	"strconv"

	gamma "github.com/HuakunShen/polymarket-kit/go-client/gamma"
)

type FetchedMarket struct {
	ID          string
	Title       string
	Probability float64
	Volume      float64
}

type Fetcher struct {
	sdk *gamma.GammaSDK
}

func New() *Fetcher {
	return &Fetcher{
		sdk: gamma.NewGammaSDK(nil),
	}
}

func (f *Fetcher) FetchActiveMarkets(ctx context.Context) ([]FetchedMarket, error) {
	markets, err := f.sdk.GetActiveMarkets(&gamma.UpdatedMarketQuery{
		Limit:     gamma.IntPtr(500),
		Order:     gamma.StringPtr("volume24hr"),
		Ascending: gamma.BoolPtr(false),
	})
	if err != nil {
		return nil, fmt.Errorf("fetch active markets: %w", err)
	}

	result := make([]FetchedMarket, 0, len(markets))
	for _, m := range markets {
		fm, err := mapMarket(m)
		if err != nil {
			continue
		}
		result = append(result, fm)
	}

	return result, nil
}

func mapMarket(m gamma.Market) (FetchedMarket, error) {
	if m.ID == "" {
		return FetchedMarket{}, fmt.Errorf("empty id")
	}
	if len(m.OutcomePrices) == 0 {
		return FetchedMarket{}, fmt.Errorf("no outcome prices for %s", m.ID)
	}

	yesPrice, err := strconv.ParseFloat(m.OutcomePrices[0], 64)
	if err != nil {
		return FetchedMarket{}, fmt.Errorf("parse yes price %s: %w", m.ID, err)
	}

	return FetchedMarket{
		ID:          m.ID,
		Title:       m.Question,
		Probability: yesPrice,
		Volume:      m.VolumeNum,
	}, nil
}
