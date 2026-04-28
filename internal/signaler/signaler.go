package signaler

import (
	"fmt"
	"math"
	"sort"

	"tema/internal/behavior"
	"tema/internal/model"
	"tema/internal/modeler"
)

type SignalStrength string

const (
	StrengthWeak   SignalStrength = "weak"
	StrengthMedium SignalStrength = "medium"
	StrengthStrong SignalStrength = "strong"
)

type Signal struct {
	MarketID     string
	MarketProb   float64
	ExpectedProb float64
	Edge         float64
	AdjustedEdge float64
	AbsEdge      float64
	Direction    model.SignalDirection
	Strength     SignalStrength
	Behavior     behavior.Behavior
	Confidence   float64
	Volume       float64
}

type Config struct {
	Threshold float64
	MinVolume float64
}

func DefaultConfig() Config {
	return Config{
		Threshold: 0.10,
		MinVolume: 0,
	}
}

func GenerateSignals(
	expectedResults []modeler.ExpectedResult,
	volumes map[string]float64,
	behaviors map[string]behavior.Result,
	cfg Config,
) []Signal {
	var signals []Signal

	for _, r := range expectedResults {
		if r.RelationsUsed == 0 {
			continue
		}

		edge := r.ExpectedProb - r.MarketProb

		vol := volumes[r.TargetMarketID]
		if vol < cfg.MinVolume {
			continue
		}

		b := behaviors[r.TargetMarketID]

		adjustedEdge := edge * b.Confidence
		absAdjustedEdge := math.Abs(adjustedEdge)

		if absAdjustedEdge < cfg.Threshold {
			continue
		}

		var direction model.SignalDirection
		if adjustedEdge > 0 {
			direction = model.DirectionBuyYES
		} else {
			direction = model.DirectionBuyNO
		}

		var strength SignalStrength
		switch {
		case absAdjustedEdge >= 0.25:
			strength = StrengthStrong
		case absAdjustedEdge >= 0.15:
			strength = StrengthMedium
		default:
			strength = StrengthWeak
		}

		signals = append(signals, Signal{
			MarketID:     r.TargetMarketID,
			MarketProb:   r.MarketProb,
			ExpectedProb: r.ExpectedProb,
			Edge:         edge,
			AdjustedEdge: adjustedEdge,
			AbsEdge:      absAdjustedEdge,
			Direction:    direction,
			Strength:     strength,
			Behavior:     b.Behavior,
			Confidence:   b.Confidence,
			Volume:       vol,
		})
	}

	sort.Slice(signals, func(i, j int) bool {
		return signals[i].AbsEdge > signals[j].AbsEdge
	})

	return signals
}

func FormatSignal(s Signal) string {
	return fmt.Sprintf("%-40s  market=%.3f  expected=%.3f  edge=%+.3f  adj=%+.3f  dir=%-7s  str=%-6s  beh=%-7s  conf=%.1f  vol=%.0f",
		s.MarketID, s.MarketProb, s.ExpectedProb, s.Edge, s.AdjustedEdge,
		s.Direction, s.Strength, s.Behavior, s.Confidence, s.Volume)
}
