package app

import (
	"context"

	"github.com/czy/carbon-guard/internal/domain/scheduling"
)

type Provider interface {
	GetCurrentCI(ctx context.Context, zone string) (float64, error)
	GetForecastCI(ctx context.Context, zone string, hours int) ([]scheduling.ForecastPoint, error)
}
