package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL          string
	FetchInterval        time.Duration
	Port                 string
	SignalThreshold      float64
	MinVolume            float64
	PriceChangeThreshold float64
	Bankroll             float64
	RiskK                float64
	BasePath             string
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:pass@localhost:5432/tema?sslmode=disable"
	}

	intervalStr := os.Getenv("FETCH_INTERVAL")
	if intervalStr == "" {
		intervalStr = "60"
	}
	intervalSec, err := strconv.Atoi(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid FETCH_INTERVAL: %w", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	threshold := 0.10
	if t := os.Getenv("SIGNAL_THRESHOLD"); t != "" {
		if v, err := strconv.ParseFloat(t, 64); err == nil {
			threshold = v
		}
	}

	minVolume := 0.0
	if v := os.Getenv("MIN_VOLUME"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			minVolume = f
		}
	}

	priceChangeThreshold := 0.05
	if v := os.Getenv("PRICE_CHANGE_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			priceChangeThreshold = f
		}
	}

	bankroll := 1000.0
	if v := os.Getenv("BANKROLL"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			bankroll = f
		}
	}

	riskK := 0.5
	if v := os.Getenv("RISK_K"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			riskK = f
		}
	}

	basePath := os.Getenv("BASE_PATH")

	return &Config{
		DatabaseURL:          dbURL,
		FetchInterval:        time.Duration(intervalSec) * time.Second,
		Port:                 port,
		SignalThreshold:      threshold,
		MinVolume:            minVolume,
		PriceChangeThreshold: priceChangeThreshold,
		Bankroll:             bankroll,
		RiskK:                riskK,
		BasePath:             basePath,
	}, nil
}
