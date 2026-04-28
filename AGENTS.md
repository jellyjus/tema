# AGENTS.md

## Project

Go project (`tema`, Go 1.26.2) — prediction market signal detector. Ingests Polymarket data, builds multi-factor probability models from manually-defined market relations, calculates edge, and generates buy signals with position sizing and P&L tracking.

## Commands

```bash
go build ./...          # build all packages
go vet ./...            # lint
go run cmd/tema/main.go # run service (requires running PostgreSQL)
```

## Environment

| Variable                 | Default                                                        | Description                              |
|--------------------------|----------------------------------------------------------------|------------------------------------------|
| `DATABASE_URL`           | `postgres://postgres:pass@localhost:5432/tema?sslmode=disable` | PostgreSQL connection string             |
| `FETCH_INTERVAL`         | `60`                                                           | Seconds between market data fetches      |
| `PORT`                   | `8080`                                                         | HTTP server port                         |
| `SIGNAL_THRESHOLD`       | `0.01`                                                         | Minimum `adjusted_edge` for a signal     |
| `MIN_VOLUME`             | `0`                                                            | Minimum market volume to include         |
| `PRICE_CHANGE_THRESHOLD` | `0.05`                                                         | Price change % to flag as crowd behavior |
| `BANKROLL`               | `1000`                                                         | Total capital for position sizing ($)    |
| `RISK_K`                 | `0.5`                                                          | Risk coefficient for bet sizing          |

DB schema is auto-migrated on startup via `db.Migrate()`.

## Architecture

Data flow: `fetch markets → store prices → define relations → compute expected probability → calculate edge → filter signals → position sizing → open trades → output`

### Implemented (steps 1–9)

- **Step 1 — Data ingestion**: `internal/fetcher` wraps `polymarket-kit/go-client/gamma` SDK. Fetches active markets via Gamma API (no auth required for reads). Uses `OutcomePrices[0]` as `probability` (YES price).
- **Step 2 — Storage**: `internal/db` with PostgreSQL via `pgxpool`. Tables: `markets`, `market_prices` (append-only time-series), `relations`, `signals`, `market_behavior`, `trades`.
- **Step 3 — Relations CRUD**: HTTP API for creating/querying/deleting `source → target` relations with type (positive/negative) and weight.
- **Step 4 — Expected probability**: `internal/modeler` — weighted sum: `expected = Σ(probability × weight)` for positive, `(1 - probability) × weight` for negative. Normalized by sum of weights. Clamped to [0, 1].
- **Step 5 — Edge & signals**: `internal/signaler` — `edge = expected - market`, direction (BUY YES/NO), threshold filter, strength tiers (weak/medium/strong). Sorted by `|adjustedEdge|` desc. Saved to `signals` table.
- **Step 6 — Behavioral analysis**: `internal/behavior` — compares current vs previous prices. If `|price_change| > threshold` AND `volume_change > 0` → `crowd` (confidence=1.1), else `neutral` (1.0). `adjusted_edge = edge × confidence`.
- **Step 7 — Signal output**: HTML dashboard at `/` (Vue 3 via CDN, no build step). Four tabs: signals, relations, trades (with metrics), markets. API endpoints for all resources.
- **Step 8 — Position sizing**: `internal/sizer` — `bet = bankroll × k × |adjusted_edge|`. Min 1%, max 5% of bankroll. Extreme probability markets (<0.1 or >0.9) get 50% reduction. Total exposure capped at 25% of bankroll with proportional scaling. Configurable via `BANKROLL` and `RISK_K` env vars. `bet_size` stored in `signals` table.
- **Step 9 — P&L tracking**: `internal/db` trades table with open/close lifecycle. Trades auto-created from signals (one open trade per market, no duplicates). PnL calculated on close: BUY YES wins if event resolves true, BUY NO wins if resolves false. Dashboard shows total PnL, ROI, win rate, win/loss counts. Close trade via UI with exit_price input.

### Signal deduplication

Each fetch cycle clears `signals` table before re-inserting fresh signals. This avoids duplicates — signals represent the current state, not a historical log.

### Trade lifecycle

1. Signal generated → auto-creates `open` trade with `entry_price = market_probability`, `bet_size` from sizer
2. No duplicate open trades per market (`HasOpenTrade` check)
3. User closes trade via UI, providing `exit_price` (resolution outcome: 0–1)
4. PnL auto-calculated: win = `bet × (1 - entry)`, loss = `bet × entry`

## Key domain concepts

- **Market**: Polymarket event; `probability = yes_price` (0–1)
- **Relation**: directional causal link `source → target` with type (positive/negative) and weight (0–1); must be interpretable
- **Edge**: `expected_probability - market_price`; actionable threshold 10–15%
- **Adjusted edge**: `edge × confidence`; confidence = 1.0 (neutral) or 1.1 (crowd)
- **Bet size**: `bankroll × k × |adjusted_edge|`, clamped to [1%, 5%], halved for extreme probabilities, scaled down if total exposure > 25%
- **Trade**: paper trade tracking entry/exit prices and PnL

## HTTP API

| Method | Path | Description |
|---|---|---|
| GET | `/api/markets` | List all stored markets |
| GET | `/api/relations` | List all relations (with market titles) |
| POST | `/api/relations` | Create relation (JSON: `source_market_id`, `target_market_id`, `relation_type`, `weight`) |
| DELETE | `/api/relations/{id}` | Delete relation |
| GET | `/api/prices/latest` | Get latest price for each market |
| GET | `/api/signals?limit=50` | List recent signals (includes `bet_size`) |
| GET | `/api/trades?limit=100` | List trades (with market titles) |
| POST | `/api/trades` | Open trade (JSON: `market_id`, `direction`, `entry_price`, `bet_size`) |
| POST | `/api/trades/{id}/close` | Close trade (JSON: `exit_price` 0–1, auto-calculates PnL) |
| GET | `/api/trades/metrics` | Aggregate metrics: `total_pnl`, `roi`, `win_rate`, `total_trades`, `wins`, `losses` |
| GET | `/` | HTML dashboard (Vue 3, no build step) |

## Project structure

```
cmd/tema/main.go        — entrypoint, daemon + HTTP server + signal pipeline
internal/
  behavior/behavior.go  — crowd detection, confidence multiplier
  config/config.go       — env config (DATABASE_URL, FETCH_INTERVAL, PORT, SIGNAL_THRESHOLD, MIN_VOLUME, PRICE_CHANGE_THRESHOLD, BANKROLL, RISK_K)
  db/migrate.go          — schema migrations (7 tables + ALTER for bet_size)
  db/store.go            — all DB operations (UpsertMarket, InsertPrice, GetLatestPrices, GetPreviousPrices, CreateRelation, DeleteRelation, ListRelations, GetAllRelations, InsertSignal, ClearSignals, ListSignals, GetMarketTitle, GetMarketVolumes, SaveFetchedMarkets, InsertTrade, ListTrades, CloseTrade, HasOpenTrade, GetTradeMetrics)
  fetcher/fetcher.go     — Polymarket Gamma API client
  model/model.go          — domain types (Market, MarketPrice, Relation, RelationInput, Signal, Trade, TradeStatus, SignalDirection)
  modeler/modeler.go      — expected probability calculation
  server/server.go        — HTTP API handlers + //go:embed index.html
  server/index.html       — Vue 3 dashboard (4 tabs: signals, relations, trades, markets)
  signaler/signaler.go    — edge, direction, threshold filtering
  sizer/sizer.go          — position sizing (bankroll, k-factor, min/max bets, exposure cap)
```

## Language

Docs and domain terms are in Russian. Code identifiers in English; Russian only in user-facing output.

## Dependencies

- `github.com/HuakunShen/polymarket-kit/go-client/gamma` — Polymarket data fetching (read-only, no auth)
- `github.com/jackc/pgx/v5/pgxpool` — PostgreSQL driver