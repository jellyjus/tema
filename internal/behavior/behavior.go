package behavior

import (
	"tema/internal/model"
)

type Behavior string

const (
	Crowd   Behavior = "crowd"
	Neutral Behavior = "neutral"
)

type Result struct {
	MarketID     string
	Behavior     Behavior
	Confidence   float64
	PriceChange  float64
	VolumeChange float64
}

type Config struct {
	PriceChangeThreshold float64
}

func DefaultConfig() Config {
	return Config{
		PriceChangeThreshold: 0.05,
	}
}

func Analyze(
	current map[string]model.MarketPrice,
	previous map[string]model.MarketPrice,
	cfg Config,
) map[string]Result {
	results := make(map[string]Result)

	for id, cur := range current {
		prev, ok := previous[id]
		if !ok {
			results[id] = Result{
				MarketID:   id,
				Behavior:   Neutral,
				Confidence: 1.0,
			}
			continue
		}

		priceChange := cur.Probability - prev.Probability
		volumeChange := cur.Volume - prev.Volume

		var behavior Behavior
		var confidence float64

		if abs(priceChange) > cfg.PriceChangeThreshold && volumeChange > 0 {
			behavior = Crowd
			confidence = 1.1
		} else {
			behavior = Neutral
			confidence = 1.0
		}

		results[id] = Result{
			MarketID:     id,
			Behavior:     behavior,
			Confidence:   confidence,
			PriceChange:  priceChange,
			VolumeChange: volumeChange,
		}
	}

	return results
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
