package modeler

import (
	"fmt"
	"math"

	"tema/internal/model"
)

type ExpectedResult struct {
	TargetMarketID   string
	ExpectedProb     float64
	MarketProb       float64
	RelationsUsed    int
	SkippedRelations []string
}

func ComputeExpected(
	targetMarketID string,
	marketProb float64,
	relations []model.Relation,
	prices map[string]float64,
) ExpectedResult {
	result := ExpectedResult{
		TargetMarketID: targetMarketID,
		MarketProb:     marketProb,
	}

	sumContribution := 0.0
	sumWeights := 0.0

	for _, rel := range relations {
		sourceProb, ok := prices[rel.SourceMarketID]
		if !ok {
			result.SkippedRelations = append(result.SkippedRelations,
				fmt.Sprintf("source %s: no price", rel.SourceMarketID))
			continue
		}

		var contribution float64
		switch rel.RelationType {
		case model.RelationTypePositive:
			contribution = sourceProb * rel.Weight
		case model.RelationTypeNegative:
			contribution = (1 - sourceProb) * rel.Weight
		default:
			result.SkippedRelations = append(result.SkippedRelations,
				fmt.Sprintf("relation %d: unknown type %s", rel.ID, rel.RelationType))
			continue
		}

		sumContribution += contribution
		sumWeights += rel.Weight
		result.RelationsUsed++
	}

	if result.RelationsUsed == 0 {
		return result
	}

	if sumWeights > 0 {
		result.ExpectedProb = sumContribution / sumWeights
	} else {
		result.ExpectedProb = sumContribution
	}

	result.ExpectedProb = math.Max(0, math.Min(1, result.ExpectedProb))

	return result
}

func ComputeAllExpected(
	relationsByTarget map[string][]model.Relation,
	prices map[string]float64,
) []ExpectedResult {
	var results []ExpectedResult

	for targetID, relations := range relationsByTarget {
		marketProb, ok := prices[targetID]
		if !ok {
			continue
		}

		result := ComputeExpected(targetID, marketProb, relations, prices)
		if result.RelationsUsed == 0 {
			continue
		}
		results = append(results, result)
	}

	return results
}
