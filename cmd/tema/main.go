package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tema/internal/behavior"
	"tema/internal/config"
	"tema/internal/db"
	"tema/internal/fetcher"
	"tema/internal/model"
	"tema/internal/modeler"
	"tema/internal/server"
	"tema/internal/signaler"
	"tema/internal/sizer"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := db.NewStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer store.Close()

	if err := db.Migrate(ctx, store.Pool()); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("db migrated")

	f := fetcher.New()
	srv := server.New(store, cfg.BasePath)

	behavCfg := behavior.Config{PriceChangeThreshold: cfg.PriceChangeThreshold}
	sizerCfg := sizer.Config{
		Bankroll:         cfg.Bankroll,
		K:                cfg.RiskK,
		MaxBetPct:        0.05,
		MinBetPct:        0.01,
		MaxTotalPct:      0.25,
		ExtremeBoundPct:  0.5,
		ExtremeThreshold: 0.9,
	}
	go runFetcher(ctx, f, store, cfg.FetchInterval, cfg.SignalThreshold, cfg.MinVolume, behavCfg, sizerCfg)

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: srv.Handler(),
	}

	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	httpServer.Shutdown(shutdownCtx)
	cancel()
}

func runFetcher(ctx context.Context, f *fetcher.Fetcher, store *db.Store, interval time.Duration, threshold, minVolume float64, behavCfg behavior.Config, sizerCfg sizer.Config) {
	fetchAndSave := func() {
		log.Println("fetching markets...")
		markets, err := f.FetchActiveMarkets(ctx)
		if err != nil {
			log.Printf("fetch error: %v", err)
			return
		}
		saved, err := store.SaveFetchedMarkets(ctx, markets)
		if err != nil {
			log.Printf("save error: %v", err)
			return
		}
		log.Printf("saved %d/%d markets", saved, len(markets))

		generateSignals(ctx, store, threshold, minVolume, behavCfg, sizerCfg)
	}

	fetchAndSave()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fetchAndSave()
		}
	}
}

func generateSignals(ctx context.Context, store *db.Store, threshold, minVolume float64, behavCfg behavior.Config, sizerCfg sizer.Config) {
	relations, err := store.GetAllRelations(ctx)
	if err != nil {
		log.Printf("relations error: %v", err)
		return
	}
	if len(relations) == 0 {
		return
	}

	currentPrices, err := store.GetLatestPrices(ctx)
	if err != nil {
		log.Printf("prices error: %v", err)
		return
	}

	volumes, err := store.GetMarketVolumes(ctx)
	if err != nil {
		log.Printf("volumes error: %v", err)
		return
	}

	var now time.Time
	for _, p := range currentPrices {
		if p.Timestamp.After(now) {
			now = p.Timestamp
		}
	}
	previousPrices, _ := store.GetPreviousPrices(ctx, now)

	priceMap := make(map[string]float64, len(currentPrices))
	for id, p := range currentPrices {
		priceMap[id] = p.Probability
	}

	byTarget := make(map[string][]model.Relation)
	for _, r := range relations {
		byTarget[r.TargetMarketID] = append(byTarget[r.TargetMarketID], r)
	}

	expectedResults := modeler.ComputeAllExpected(byTarget, priceMap)
	if len(expectedResults) == 0 {
		return
	}

	behaviors := behavior.Analyze(currentPrices, previousPrices, behavCfg)

	signals := signaler.GenerateSignals(expectedResults, volumes, behaviors, signaler.Config{
		Threshold: threshold,
		MinVolume: minVolume,
	})

	if len(signals) == 0 {
		log.Println("no signals above threshold")
		return
	}

	sized := sizer.Size(signals, sizerCfg)

	if err := store.ClearSignals(ctx); err != nil {
		log.Printf("clear signals: %v", err)
	}

	log.Printf("=== %d signal(s) ===", len(sized))
	for _, s := range sized {
		fmt.Println(signaler.FormatSignal(s.Signal))

		err := store.InsertSignal(ctx, model.Signal{
			MarketID:          s.MarketID,
			MarketProbability: s.MarketProb,
			ExpectedProb:      s.ExpectedProb,
			Edge:              s.Edge,
			AdjustedEdge:      s.AdjustedEdge,
			Direction:         s.Direction,
			Behavior:          string(s.Behavior),
			BetSize:           s.BetSize,
		})
		if err != nil {
			log.Printf("insert signal %s: %v", s.MarketID, err)
		}

		hasOpen, err := store.HasOpenTrade(ctx, s.MarketID)
		if err != nil {
			log.Printf("check open trade %s: %v", s.MarketID, err)
		}
		if !hasOpen {
			_, err = store.InsertTrade(ctx, model.Trade{
				MarketID:   s.MarketID,
				Direction:  string(s.Direction),
				EntryPrice: s.MarketProb,
				BetSize:    s.BetSize,
			})
			if err != nil {
				log.Printf("insert trade %s: %v", s.MarketID, err)
			}
		}
	}
}
