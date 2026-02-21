package ci

import (
	"context"
	"time"
)

type ForecastPoint struct {
	Timestamp time.Time
	CI        float64 // kgCO2/kWh
}

type Provider interface {
	GetCurrentCI(ctx context.Context, zone string) (float64, error)
	GetForecastCI(ctx context.Context, zone string, hours int) ([]ForecastPoint, error)
}
