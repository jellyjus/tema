package model

import "time"

type Market struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type MarketPrice struct {
	ID          int64     `json:"id"`
	MarketID    string    `json:"market_id"`
	Probability float64   `json:"probability"`
	Volume      float64   `json:"volume"`
	Timestamp   time.Time `json:"timestamp"`
}

type RelationType string

const (
	RelationTypePositive RelationType = "positive"
	RelationTypeNegative RelationType = "negative"
)

type Relation struct {
	ID             int64        `json:"id"`
	SourceMarketID string       `json:"source_market_id"`
	TargetMarketID string       `json:"target_market_id"`
	RelationType   RelationType `json:"relation_type"`
	Weight         float64      `json:"weight"`
}

type RelationInput struct {
	SourceMarketID string       `json:"source_market_id"`
	TargetMarketID string       `json:"target_market_id"`
	RelationType   RelationType `json:"relation_type"`
	Weight         float64      `json:"weight"`
}

type SignalDirection string

const (
	DirectionBuyYES SignalDirection = "BUY YES"
	DirectionBuyNO  SignalDirection = "BUY NO"
)

type Signal struct {
	ID                int64           `json:"id"`
	MarketID          string          `json:"market_id"`
	MarketProbability float64         `json:"market_probability"`
	ExpectedProb      float64         `json:"expected_probability"`
	Edge              float64         `json:"edge"`
	AdjustedEdge      float64         `json:"adjusted_edge"`
	Direction         SignalDirection `json:"direction"`
	Behavior          string          `json:"behavior"`
	BetSize           float64         `json:"bet_size"`
	Timestamp         time.Time       `json:"timestamp"`
}

type TradeStatus string

const (
	TradeStatusOpen   TradeStatus = "open"
	TradeStatusClosed TradeStatus = "closed"
)

type Trade struct {
	ID             int64       `json:"id"`
	MarketID       string      `json:"market_id"`
	Direction      string      `json:"direction"`
	EntryPrice     float64     `json:"entry_price"`
	ExitPrice      *float64    `json:"exit_price"`
	BetSize        float64     `json:"bet_size"`
	PnL            *float64    `json:"pnl"`
	Status         TradeStatus `json:"status"`
	TimestampOpen  time.Time   `json:"timestamp_open"`
	TimestampClose *time.Time  `json:"timestamp_close"`
}
