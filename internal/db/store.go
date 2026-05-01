package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tema/internal/fetcher"
	"tema/internal/model"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}
	config.MinConns = 2
	config.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *Store) UpsertMarket(ctx context.Context, m model.Market) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO markets (id, title, created_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title`,
		m.ID, m.Title, m.CreatedAt,
	)
	return err
}

func (s *Store) InsertPrice(ctx context.Context, p model.MarketPrice) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO market_prices (market_id, probability, volume, timestamp)
		 VALUES ($1, $2, $3, $4)`,
		p.MarketID, p.Probability, p.Volume, p.Timestamp,
	)
	return err
}

func (s *Store) GetLatestPrices(ctx context.Context) (map[string]model.MarketPrice, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT ON (market_id)
		        id, market_id, probability, volume, timestamp
		 FROM market_prices
		 ORDER BY market_id, timestamp DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]model.MarketPrice)
	for rows.Next() {
		var p model.MarketPrice
		if err := rows.Scan(&p.ID, &p.MarketID, &p.Probability, &p.Volume, &p.Timestamp); err != nil {
			return nil, err
		}
		result[p.MarketID] = p
	}
	return result, rows.Err()
}

func (s *Store) GetPreviousPrices(ctx context.Context, before time.Time) (map[string]model.MarketPrice, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT ON (market_id)
		        id, market_id, probability, volume, timestamp
		 FROM market_prices
		 WHERE timestamp < $1
		 ORDER BY market_id, timestamp DESC`,
		before,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]model.MarketPrice)
	for rows.Next() {
		var p model.MarketPrice
		if err := rows.Scan(&p.ID, &p.MarketID, &p.Probability, &p.Volume, &p.Timestamp); err != nil {
			return nil, err
		}
		result[p.MarketID] = p
	}
	return result, rows.Err()
}

func (s *Store) CreateRelation(ctx context.Context, input model.RelationInput) (model.Relation, error) {
	var r model.Relation
	err := s.pool.QueryRow(ctx,
		`INSERT INTO relations (source_market_id, target_market_id, relation_type, weight)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (source_market_id, target_market_id)
		 DO UPDATE SET relation_type = EXCLUDED.relation_type, weight = EXCLUDED.weight
		 RETURNING id, source_market_id, target_market_id, relation_type, weight`,
		input.SourceMarketID, input.TargetMarketID, input.RelationType, input.Weight,
	).Scan(&r.ID, &r.SourceMarketID, &r.TargetMarketID, &r.RelationType, &r.Weight)
	return r, err
}

func (s *Store) DeleteRelation(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM relations WHERE id = $1`, id)
	return err
}

func (s *Store) ListRelations(ctx context.Context) ([]model.Relation, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, source_market_id, target_market_id, relation_type, weight FROM relations ORDER BY target_market_id, source_market_id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []model.Relation
	for rows.Next() {
		var r model.Relation
		if err := rows.Scan(&r.ID, &r.SourceMarketID, &r.TargetMarketID, &r.RelationType, &r.Weight); err != nil {
			return nil, err
		}
		relations = append(relations, r)
	}
	return relations, rows.Err()
}

func (s *Store) GetRelationsForTarget(ctx context.Context, targetMarketID string) ([]model.Relation, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, source_market_id, target_market_id, relation_type, weight
		 FROM relations WHERE target_market_id = $1`,
		targetMarketID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []model.Relation
	for rows.Next() {
		var r model.Relation
		if err := rows.Scan(&r.ID, &r.SourceMarketID, &r.TargetMarketID, &r.RelationType, &r.Weight); err != nil {
			return nil, err
		}
		relations = append(relations, r)
	}
	return relations, rows.Err()
}

func (s *Store) GetAllRelations(ctx context.Context) ([]model.Relation, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, source_market_id, target_market_id, relation_type, weight FROM relations`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []model.Relation
	for rows.Next() {
		var r model.Relation
		if err := rows.Scan(&r.ID, &r.SourceMarketID, &r.TargetMarketID, &r.RelationType, &r.Weight); err != nil {
			return nil, err
		}
		relations = append(relations, r)
	}
	return relations, rows.Err()
}

func (s *Store) ListMarkets(ctx context.Context) ([]model.Market, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, title, created_at FROM markets ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var markets []model.Market
	for rows.Next() {
		var m model.Market
		if err := rows.Scan(&m.ID, &m.Title, &m.CreatedAt); err != nil {
			return nil, err
		}
		markets = append(markets, m)
	}
	return markets, rows.Err()
}

func (s *Store) SaveFetchedMarkets(ctx context.Context, fetched []fetcher.FetchedMarket) (int, error) {
	now := time.Now()
	saved := 0
	for _, fm := range fetched {
		if fm.Probability < 0 || fm.Probability > 1 {
			continue
		}

		err := s.UpsertMarket(ctx, model.Market{
			ID:        fm.ID,
			Title:     fm.Title,
			CreatedAt: now,
		})
		if err != nil {
			return saved, fmt.Errorf("upsert market %s: %w", fm.ID, err)
		}

		err = s.InsertPrice(ctx, model.MarketPrice{
			MarketID:    fm.ID,
			Probability: fm.Probability,
			Volume:      fm.Volume,
			Timestamp:   now,
		})
		if err != nil {
			return saved, fmt.Errorf("insert price %s: %w", fm.ID, err)
		}
		saved++
	}
	return saved, nil
}

func (s *Store) InsertSignal(ctx context.Context, sig model.Signal) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO signals (market_id, market_probability, expected_probability, edge, adjusted_edge, direction, behavior, bet_size, timestamp)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		 ON CONFLICT DO NOTHING`,
		sig.MarketID, sig.MarketProbability, sig.ExpectedProb, sig.Edge, sig.AdjustedEdge, sig.Direction, sig.Behavior, sig.BetSize,
	)
	return err
}

func (s *Store) ClearSignals(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM signals`)
	return err
}

func (s *Store) ArchiveSignals(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO signals_history (market_id, market_probability, expected_probability, edge, adjusted_edge, direction, behavior, bet_size, timestamp)
		 SELECT market_id, market_probability, expected_probability, edge, adjusted_edge, direction, behavior, bet_size, timestamp FROM signals`,
	)
	return err
}

func (s *Store) GetMarketVolumes(ctx context.Context) (map[string]float64, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT ON (market_id)
		        market_id, volume
		 FROM market_prices
		 ORDER BY market_id, timestamp DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	volumes := make(map[string]float64)
	for rows.Next() {
		var marketID string
		var volume float64
		if err := rows.Scan(&marketID, &volume); err != nil {
			return nil, err
		}
		volumes[marketID] = volume
	}
	return volumes, rows.Err()
}

func (s *Store) ListSignals(ctx context.Context, limit int) ([]model.Signal, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, market_id, market_probability, expected_probability, edge, adjusted_edge, direction, behavior, bet_size, timestamp
		 FROM signals
		 ORDER BY timestamp DESC
		 LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signals []model.Signal
	for rows.Next() {
		var sig model.Signal
		if err := rows.Scan(&sig.ID, &sig.MarketID, &sig.MarketProbability, &sig.ExpectedProb, &sig.Edge, &sig.AdjustedEdge, &sig.Direction, &sig.Behavior, &sig.BetSize, &sig.Timestamp); err != nil {
			return nil, err
		}
		signals = append(signals, sig)
	}
	return signals, rows.Err()
}

func (s *Store) GetMarketTitle(ctx context.Context, marketID string) string {
	var title string
	_ = s.pool.QueryRow(ctx, `SELECT title FROM markets WHERE id = $1`, marketID).Scan(&title)
	return title
}

func (s *Store) HasOpenTrade(ctx context.Context, marketID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM trades WHERE market_id = $1 AND status = 'open')`,
		marketID,
	).Scan(&exists)
	return exists, err
}

func (s *Store) InsertTrade(ctx context.Context, t model.Trade) (model.Trade, error) {
	var trade model.Trade
	err := s.pool.QueryRow(ctx,
		`INSERT INTO trades (market_id, direction, entry_price, bet_size, status)
		 VALUES ($1, $2, $3, $4, 'open')
		 RETURNING id, market_id, direction, entry_price, exit_price, bet_size, pnl, status, timestamp_open, timestamp_close`,
		t.MarketID, t.Direction, t.EntryPrice, t.BetSize,
	).Scan(&trade.ID, &trade.MarketID, &trade.Direction, &trade.EntryPrice, &trade.ExitPrice, &trade.BetSize, &trade.PnL, &trade.Status, &trade.TimestampOpen, &trade.TimestampClose)
	return trade, err
}

func (s *Store) ListTrades(ctx context.Context, limit int) ([]model.Trade, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, market_id, direction, entry_price, exit_price, bet_size, pnl, status, timestamp_open, timestamp_close
		 FROM trades ORDER BY timestamp_open DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []model.Trade
	for rows.Next() {
		var t model.Trade
		if err := rows.Scan(&t.ID, &t.MarketID, &t.Direction, &t.EntryPrice, &t.ExitPrice, &t.BetSize, &t.PnL, &t.Status, &t.TimestampOpen, &t.TimestampClose); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}
	return trades, rows.Err()
}

func (s *Store) CloseTrade(ctx context.Context, id int64, exitPrice float64) (model.Trade, error) {
	var t model.Trade
	err := s.pool.QueryRow(ctx,
		`UPDATE trades SET exit_price = $2,
		 pnl = CASE
			WHEN direction = 'BUY YES' THEN
				CASE WHEN $2 >= 0.5 THEN bet_size * (1.0 - entry_price)
					 ELSE -bet_size * entry_price END
			ELSE
				CASE WHEN $2 < 0.5 THEN bet_size * (1.0 - entry_price)
					 ELSE -bet_size * entry_price END
			END,
		 status = 'closed',
		 timestamp_close = now()
		 WHERE id = $1
		 RETURNING id, market_id, direction, entry_price, exit_price, bet_size, pnl, status, timestamp_open, timestamp_close`,
		id, exitPrice,
	).Scan(&t.ID, &t.MarketID, &t.Direction, &t.EntryPrice, &t.ExitPrice, &t.BetSize, &t.PnL, &t.Status, &t.TimestampOpen, &t.TimestampClose)
	return t, err
}

type TradeMetrics struct {
	TotalPnL    float64 `json:"total_pnl"`
	TotalVolume float64 `json:"total_volume"`
	ROI         float64 `json:"roi"`
	WinRate     float64 `json:"win_rate"`
	TotalTrades int     `json:"total_trades"`
	Wins        int     `json:"wins"`
	Losses      int     `json:"losses"`
}

func (s *Store) GetTradeMetrics(ctx context.Context) (*TradeMetrics, error) {
	var m TradeMetrics
	err := s.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(pnl) FILTER (WHERE status = 'closed'), 0),
			COALESCE(SUM(bet_size) FILTER (WHERE status = 'closed'), 0),
			COUNT(*) FILTER (WHERE status = 'closed'),
			COALESCE(SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) FILTER (WHERE status = 'closed'), 0),
			COALESCE(SUM(CASE WHEN pnl <= 0 THEN 1 ELSE 0 END) FILTER (WHERE status = 'closed'), 0)
		 FROM trades`,
	).Scan(&m.TotalPnL, &m.TotalVolume, &m.TotalTrades, &m.Wins, &m.Losses)
	if err != nil {
		return nil, err
	}
	if m.TotalVolume > 0 {
		m.ROI = m.TotalPnL / m.TotalVolume
	}
	if m.TotalTrades > 0 {
		m.WinRate = float64(m.Wins) / float64(m.TotalTrades)
	}
	return &m, nil
}

func (s *Store) CountOpenTrades(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM trades WHERE status = 'open'`,
	).Scan(&count)
	return count, err
}

func (s *Store) GetTotalExposure(ctx context.Context) (float64, error) {
	var exposure float64
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(bet_size), 0) FROM trades WHERE status = 'open'`,
	).Scan(&exposure)
	return exposure, err
}

func (s *Store) GetRelationCountForTarget(ctx context.Context, targetMarketID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM relations WHERE target_market_id = $1`,
		targetMarketID,
	).Scan(&count)
	return count, err
}

type EdgeBucket struct {
	Range    string  `json:"range"`
	Count    int     `json:"count"`
	AvgPnL   float64 `json:"avg_pnl"`
	TotalPnL float64 `json:"total_pnl"`
	WinRate  float64 `json:"winrate"`
}

func (s *Store) GetEdgePerformance(ctx context.Context) ([]EdgeBucket, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT
			CASE
				WHEN abs(s.adjusted_edge) >= 0.10 AND abs(s.adjusted_edge) < 0.15 THEN '0.10-0.15'
				WHEN abs(s.adjusted_edge) >= 0.15 AND abs(s.adjusted_edge) < 0.25 THEN '0.15-0.25'
				WHEN abs(s.adjusted_edge) >= 0.25 THEN '>0.25'
			END AS range_label,
			COUNT(*),
			COALESCE(AVG(t.pnl), 0),
			COALESCE(SUM(t.pnl), 0),
			COALESCE(SUM(CASE WHEN t.pnl > 0 THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0)
		 FROM trades t
		 JOIN signals s ON s.market_id = t.market_id
		 WHERE t.status = 'closed'
		   AND abs(s.adjusted_edge) >= 0.10
		 GROUP BY range_label
		 ORDER BY range_label`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buckets := make([]EdgeBucket, 0)
	for rows.Next() {
		var b EdgeBucket
		if err := rows.Scan(&b.Range, &b.Count, &b.AvgPnL, &b.TotalPnL, &b.WinRate); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}
