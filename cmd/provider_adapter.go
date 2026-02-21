package cmd

import (
	"context"

	appsvc "github.com/chenzhuyu2004/carbon-guard/internal/app"
	"github.com/chenzhuyu2004/carbon-guard/internal/ci"
	"github.com/chenzhuyu2004/carbon-guard/internal/domain/scheduling"
)

type providerAdapter struct {
	inner ci.Provider
}

func newProviderAdapter(inner ci.Provider) appsvc.Provider {
	if inner == nil {
		return nil
	}
	return &providerAdapter{inner: inner}
}

func (p *providerAdapter) GetCurrentCI(ctx context.Context, zone string) (float64, error) {
	return p.inner.GetCurrentCI(ctx, zone)
}

func (p *providerAdapter) GetForecastCI(ctx context.Context, zone string, hours int) ([]scheduling.ForecastPoint, error) {
	points, err := p.inner.GetForecastCI(ctx, zone, hours)
	if err != nil {
		return nil, err
	}

	out := make([]scheduling.ForecastPoint, len(points))
	for i, point := range points {
		out[i] = scheduling.ForecastPoint{
			Timestamp: point.Timestamp,
			CI:        point.CI,
		}
	}
	return out, nil
}
