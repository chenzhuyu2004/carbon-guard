package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/chenzhuyu2004/carbon-guard/internal/domain/scheduling"
)

func (a *App) OptimizeGlobal(ctx context.Context, in OptimizeGlobalInput) (OptimizeGlobalOutput, error) {
	if a == nil || a.provider == nil {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: provider is not configured", ErrProvider)
	}
	if len(in.Zones) == 0 {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: zones is required", ErrInput)
	}
	if in.Duration <= 0 {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: duration must be > 0", ErrInput)
	}
	if in.Lookahead <= 0 {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: lookahead must be > 0", ErrInput)
	}
	if in.Duration > in.Lookahead*3600 {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: duration %ds exceeds forecast coverage %ds", ErrInput, in.Duration, in.Lookahead*3600)
	}
	if in.Timeout <= 0 {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: timeout must be > 0", ErrInput)
	}
	model, err := normalizeModel(in.Model)
	if err != nil {
		return OptimizeGlobalOutput{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, in.Timeout)
	defer cancel()

	zoneForecasts := make(map[string][]scheduling.ForecastPoint, len(in.Zones))
	zoneIndexes := make(map[string]map[int64]int, len(in.Zones))
	errCh := make(chan error, len(in.Zones))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, zone := range in.Zones {
		zone := zone
		wg.Add(1)
		go func() {
			defer wg.Done()

			forecast, err := a.provider.GetForecastCI(ctx, zone, in.Lookahead)
			if err != nil {
				errCh <- fmt.Errorf("zone %s failed: %w", zone, wrapProviderError(err))
				cancel()
				return
			}
			if len(forecast) == 0 {
				errCh <- fmt.Errorf("%w: zone %s failed: no forecast points", ErrNoValidWindow, zone)
				cancel()
				return
			}

			normalized := scheduling.NormalizeForecastUTC(forecast)
			mu.Lock()
			zoneForecasts[zone] = normalized
			zoneIndexes[zone] = scheduling.BuildForecastIndex(normalized)
			mu.Unlock()
		}()
	}

	wg.Wait()
	close(errCh)

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: operation timed out", ErrTimeout)
	}

	for err := range errCh {
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || (errors.Is(err, context.Canceled) && errors.Is(ctx.Err(), context.DeadlineExceeded)) {
				return OptimizeGlobalOutput{}, fmt.Errorf("%w: operation timed out", ErrTimeout)
			}
			if errors.Is(err, ErrNoValidWindow) {
				return OptimizeGlobalOutput{}, err
			}
			return OptimizeGlobalOutput{}, fmt.Errorf("%w: %v", ErrProvider, err)
		}
	}

	timeAxis := scheduling.IntersectTimestamps(in.Zones, zoneForecasts)
	if len(timeAxis) == 0 {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: no common timestamps across zones", ErrNoValidWindow)
	}

	bestFound := false
	bestEmission := 0.0
	bestZone := ""
	bestStart := time.Time{}
	worstEmission := 0.0
	worstFound := false

	for _, start := range timeAxis {
		for _, zone := range in.Zones {
			index, ok := zoneIndexes[zone][start.Unix()]
			if !ok {
				continue
			}

			emission, ok := scheduling.EstimateWindowEmissions(zoneForecasts[zone][index:], in.Duration, model.Runner, model.Load, model.PUE)
			if !ok {
				continue
			}

			if !bestFound || emission < bestEmission {
				bestFound = true
				bestEmission = emission
				bestZone = zone
				bestStart = start
			}

			if !worstFound || emission > worstEmission {
				worstFound = true
				worstEmission = emission
			}
		}
	}

	if !bestFound {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: no valid full window found across zones and timestamps", ErrNoValidWindow)
	}

	reduction := 0.0
	if worstFound && worstEmission > 0 {
		reduction = (worstEmission - bestEmission) / worstEmission * 100
	}

	return OptimizeGlobalOutput{
		BestZone:  bestZone,
		BestStart: bestStart.UTC(),
		BestEnd:   bestStart.Add(time.Duration(in.Duration) * time.Second).UTC(),
		Emission:  bestEmission,
		Reduction: reduction,
	}, nil
}
