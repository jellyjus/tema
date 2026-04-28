package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS markets (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS market_prices (
			id BIGSERIAL PRIMARY KEY,
			market_id TEXT NOT NULL REFERENCES markets(id),
			probability DOUBLE PRECISION NOT NULL CHECK (probability >= 0 AND probability <= 1),
			volume DOUBLE PRECISION NOT NULL DEFAULT 0,
			timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,

		`CREATE INDEX IF NOT EXISTS idx_market_prices_market_id ON market_prices(market_id)`,
		`CREATE INDEX IF NOT EXISTS idx_market_prices_timestamp ON market_prices(timestamp)`,

		`CREATE TABLE IF NOT EXISTS relations (
			id BIGSERIAL PRIMARY KEY,
			source_market_id TEXT NOT NULL REFERENCES markets(id),
			target_market_id TEXT NOT NULL REFERENCES markets(id),
			relation_type TEXT NOT NULL CHECK (relation_type IN ('positive', 'negative')),
			weight DOUBLE PRECISION NOT NULL CHECK (weight > 0 AND weight <= 1),
			UNIQUE(source_market_id, target_market_id)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_relations_target ON relations(target_market_id)`,
		`CREATE INDEX IF NOT EXISTS idx_relations_source ON relations(source_market_id)`,

		`CREATE TABLE IF NOT EXISTS signals (
			id BIGSERIAL PRIMARY KEY,
			market_id TEXT NOT NULL REFERENCES markets(id),
			market_probability DOUBLE PRECISION NOT NULL,
			expected_probability DOUBLE PRECISION NOT NULL,
			edge DOUBLE PRECISION NOT NULL,
			adjusted_edge DOUBLE PRECISION NOT NULL DEFAULT 0,
			direction TEXT NOT NULL CHECK (direction IN ('BUY YES', 'BUY NO')),
			behavior TEXT NOT NULL DEFAULT 'neutral' CHECK (behavior IN ('crowd', 'neutral')),
			bet_size DOUBLE PRECISION NOT NULL DEFAULT 0,
			timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS market_behavior (
			id BIGSERIAL PRIMARY KEY,
			market_id TEXT NOT NULL REFERENCES markets(id),
			price_change DOUBLE PRECISION NOT NULL DEFAULT 0,
			volume_change DOUBLE PRECISION NOT NULL DEFAULT 0,
			volatility DOUBLE PRECISION NOT NULL DEFAULT 0,
			sentiment_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS trades (
			id BIGSERIAL PRIMARY KEY,
			market_id TEXT NOT NULL REFERENCES markets(id),
			direction TEXT NOT NULL CHECK (direction IN ('BUY YES', 'BUY NO')),
			entry_price DOUBLE PRECISION NOT NULL,
			exit_price DOUBLE PRECISION,
			bet_size DOUBLE PRECISION NOT NULL,
			pnl DOUBLE PRECISION,
			status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
			timestamp_open TIMESTAMPTZ NOT NULL DEFAULT now(),
			timestamp_close TIMESTAMPTZ
		)`,

		`ALTER TABLE signals ADD COLUMN IF NOT EXISTS bet_size DOUBLE PRECISION NOT NULL DEFAULT 0`,
	}

	for i, q := range queries {
		if _, err := pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
	}
	return nil
}
