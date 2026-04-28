package server

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"tema/internal/db"
	"tema/internal/model"
)

//go:embed index.html
var indexHTML []byte

type Server struct {
	store    *db.Store
	mux      *http.ServeMux
	basePath string
}

func New(store *db.Store, basePath string) *Server {
	s := &Server{store: store, mux: http.NewServeMux(), basePath: basePath}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /api/markets", s.handleListMarkets)
	s.mux.HandleFunc("GET /api/relations", s.handleListRelations)
	s.mux.HandleFunc("POST /api/relations", s.handleCreateRelation)
	s.mux.HandleFunc("DELETE /api/relations/{id}", s.handleDeleteRelation)
	s.mux.HandleFunc("GET /api/prices/latest", s.handleLatestPrices)
	s.mux.HandleFunc("GET /api/signals", s.handleListSignals)
	s.mux.HandleFunc("GET /api/trades", s.handleListTrades)
	s.mux.HandleFunc("POST /api/trades", s.handleOpenTrade)
	s.mux.HandleFunc("POST /api/trades/{id}/close", s.handleCloseTrade)
	s.mux.HandleFunc("GET /api/trades/metrics", s.handleTradeMetrics)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := strings.ReplaceAll(string(indexHTML), "__BASE_PATH__", s.basePath)
	w.Write([]byte(html))
}

func (s *Server) handleListMarkets(w http.ResponseWriter, r *http.Request) {
	markets, err := s.store.ListMarkets(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, markets)
}

func (s *Server) handleListRelations(w http.ResponseWriter, r *http.Request) {
	relations, err := s.store.ListRelations(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type relationRow struct {
		ID                int64   `json:"id"`
		SourceMarketID    string  `json:"source_market_id"`
		SourceMarketTitle string  `json:"source_market_title"`
		TargetMarketID    string  `json:"target_market_id"`
		TargetMarketTitle string  `json:"target_market_title"`
		RelationType      string  `json:"relation_type"`
		Weight            float64 `json:"weight"`
	}

	rows := make([]relationRow, 0, len(relations))
	for _, rel := range relations {
		srcTitle := s.store.GetMarketTitle(r.Context(), rel.SourceMarketID)
		tgtTitle := s.store.GetMarketTitle(r.Context(), rel.TargetMarketID)
		rows = append(rows, relationRow{
			ID:                rel.ID,
			SourceMarketID:    rel.SourceMarketID,
			SourceMarketTitle: srcTitle,
			TargetMarketID:    rel.TargetMarketID,
			TargetMarketTitle: tgtTitle,
			RelationType:      string(rel.RelationType),
			Weight:            rel.Weight,
		})
	}
	writeJSON(w, rows)
}

func (s *Server) handleCreateRelation(w http.ResponseWriter, r *http.Request) {
	var input model.RelationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if input.SourceMarketID == "" || input.TargetMarketID == "" {
		http.Error(w, "source_market_id and target_market_id required", http.StatusBadRequest)
		return
	}
	if input.RelationType != model.RelationTypePositive && input.RelationType != model.RelationTypeNegative {
		http.Error(w, "relation_type must be 'positive' or 'negative'", http.StatusBadRequest)
		return
	}
	if input.Weight <= 0 || input.Weight > 1 {
		http.Error(w, "weight must be in (0, 1]", http.StatusBadRequest)
		return
	}

	rel, err := s.store.CreateRelation(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, rel)
}

func (s *Server) handleDeleteRelation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteRelation(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLatestPrices(w http.ResponseWriter, r *http.Request) {
	prices, err := s.store.GetLatestPrices(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, prices)
}

func (s *Server) handleListSignals(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	signals, err := s.store.ListSignals(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type signalRow struct {
		ID                int64   `json:"id"`
		MarketID          string  `json:"market_id"`
		Title             string  `json:"title"`
		MarketProbability float64 `json:"market_probability"`
		ExpectedProb      float64 `json:"expected_probability"`
		Edge              float64 `json:"edge"`
		AdjustedEdge      float64 `json:"adjusted_edge"`
		Direction         string  `json:"direction"`
		Behavior          string  `json:"behavior"`
		BetSize           float64 `json:"bet_size"`
		Timestamp         string  `json:"timestamp"`
	}

	rows := make([]signalRow, 0, len(signals))
	for _, sig := range signals {
		title := s.store.GetMarketTitle(r.Context(), sig.MarketID)
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		rows = append(rows, signalRow{
			ID:                sig.ID,
			MarketID:          sig.MarketID,
			Title:             title,
			MarketProbability: sig.MarketProbability,
			ExpectedProb:      sig.ExpectedProb,
			Edge:              sig.Edge,
			AdjustedEdge:      sig.AdjustedEdge,
			Direction:         string(sig.Direction),
			Behavior:          sig.Behavior,
			BetSize:           sig.BetSize,
			Timestamp:         sig.Timestamp.Format("2006-01-02 15:04:05"),
		})
	}
	writeJSON(w, rows)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) handleListTrades(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	trades, err := s.store.ListTrades(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type tradeRow struct {
		ID             int64    `json:"id"`
		MarketID       string   `json:"market_id"`
		Title          string   `json:"title"`
		Direction      string   `json:"direction"`
		EntryPrice     float64  `json:"entry_price"`
		ExitPrice      *float64 `json:"exit_price"`
		BetSize        float64  `json:"bet_size"`
		PnL            *float64 `json:"pnl"`
		Status         string   `json:"status"`
		TimestampOpen  string   `json:"timestamp_open"`
		TimestampClose *string  `json:"timestamp_close"`
	}

	rows := make([]tradeRow, 0, len(trades))
	for _, t := range trades {
		title := s.store.GetMarketTitle(r.Context(), t.MarketID)
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		var closeStr *string
		if t.TimestampClose != nil {
			cs := t.TimestampClose.Format("2006-01-02 15:04:05")
			closeStr = &cs
		}
		rows = append(rows, tradeRow{
			ID:             t.ID,
			MarketID:       t.MarketID,
			Title:          title,
			Direction:      t.Direction,
			EntryPrice:     t.EntryPrice,
			ExitPrice:      t.ExitPrice,
			BetSize:        t.BetSize,
			PnL:            t.PnL,
			Status:         string(t.Status),
			TimestampOpen:  t.TimestampOpen.Format("2006-01-02 15:04:05"),
			TimestampClose: closeStr,
		})
	}
	writeJSON(w, rows)
}

func (s *Server) handleOpenTrade(w http.ResponseWriter, r *http.Request) {
	var input struct {
		MarketID   string  `json:"market_id"`
		Direction  string  `json:"direction"`
		EntryPrice float64 `json:"entry_price"`
		BetSize    float64 `json:"bet_size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if input.MarketID == "" || input.Direction == "" || input.EntryPrice <= 0 || input.BetSize <= 0 {
		http.Error(w, "market_id, direction, entry_price (>0), bet_size (>0) required", http.StatusBadRequest)
		return
	}
	if input.Direction != "BUY YES" && input.Direction != "BUY NO" {
		http.Error(w, "direction must be 'BUY YES' or 'BUY NO'", http.StatusBadRequest)
		return
	}

	t, err := s.store.InsertTrade(r.Context(), model.Trade{
		MarketID:   input.MarketID,
		Direction:  input.Direction,
		EntryPrice: input.EntryPrice,
		BetSize:    input.BetSize,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, t)
}

func (s *Server) handleCloseTrade(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var input struct {
		ExitPrice float64 `json:"exit_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if input.ExitPrice <= 0 || input.ExitPrice > 1 {
		http.Error(w, "exit_price must be in (0, 1]", http.StatusBadRequest)
		return
	}

	t, err := s.store.CloseTrade(r.Context(), id, input.ExitPrice)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, t)
}

func (s *Server) handleTradeMetrics(w http.ResponseWriter, r *http.Request) {
	m, err := s.store.GetTradeMetrics(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, m)
}
