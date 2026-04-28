package sizer

import (
	"math"

	"tema/internal/signaler"
)

type Config struct {
	Bankroll         float64
	K                float64
	MaxBetPct        float64
	MinBetPct        float64
	MaxTotalPct      float64
	ExtremeBoundPct  float64
	ExtremeThreshold float64
}

func DefaultConfig() Config {
	return Config{
		Bankroll:         1000,
		K:                0.5,
		MaxBetPct:        0.05,
		MinBetPct:        0.01,
		MaxTotalPct:      0.25,
		ExtremeBoundPct:  0.5,
		ExtremeThreshold: 0.9,
	}
}

type SizedSignal struct {
	signaler.Signal
	BetSize float64
}

func Size(signals []signaler.Signal, cfg Config) []SizedSignal {
	if len(signals) == 0 {
		return nil
	}

	maxBet := cfg.Bankroll * cfg.MaxBetPct
	minBet := cfg.Bankroll * cfg.MinBetPct

	result := make([]SizedSignal, 0, len(signals))
	for _, s := range signals {
		bet := cfg.Bankroll * cfg.K * s.AbsEdge

		if s.MarketProb < (1-cfg.ExtremeThreshold) || s.MarketProb > cfg.ExtremeThreshold {
			bet *= cfg.ExtremeBoundPct
		}

		bet = math.Max(bet, minBet)
		bet = math.Min(bet, maxBet)

		result = append(result, SizedSignal{
			Signal:  s,
			BetSize: bet,
		})
	}

	totalBet := 0.0
	for i := range result {
		totalBet += result[i].BetSize
	}

	maxTotal := cfg.Bankroll * cfg.MaxTotalPct
	if totalBet > maxTotal && totalBet > 0 {
		scale := maxTotal / totalBet
		for i := range result {
			result[i].BetSize *= scale
		}
	}

	return result
}
