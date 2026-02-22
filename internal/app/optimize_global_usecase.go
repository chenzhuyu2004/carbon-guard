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
	if err := validateZones(in.Zones); err != nil {
		return OptimizeGlobalOutput{}, err
	}
	if err := validateDurationSeconds(in.Duration); err != nil {
		return OptimizeGlobalOutput{}, err
	}
	if err := validateLookaheadHours(in.Lookahead); err != nil {
		return OptimizeGlobalOutput{}, err
	}
	if err := validateDurationWithinLookahead(in.Duration, in.Lookahead); err != nil {
		return OptimizeGlobalOutput{}, err
	}
	if err := validateWaitCost(in.WaitCost); err != nil {
		return OptimizeGlobalOutput{}, err
	}
	if in.Timeout <= 0 {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: timeout must be > 0", ErrInput)
	}
	model, err := normalizeModel(in.Model)
	if err != nil {
		return OptimizeGlobalOutput{}, err
	}

	requestStart := time.Now().UTC()
	windowEnd := requestStart.Add(time.Duration(in.Lookahead) * time.Hour).UTC()

	ctx, cancel := context.WithTimeout(ctx, in.Timeout)
	defer cancel()

	zoneForecasts := make(map[string][]scheduling.ForecastPoint, len(in.Zones))
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

	step := scheduling.InferResampleStep(zoneForecasts)
	timeAxis, alignedForecasts := scheduling.BuildResampledIntersection(in.Zones, zoneForecasts, step)
	if len(timeAxis) == 0 {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: no common timestamps across zones", ErrNoValidWindow)
	}

	evaluators := make(map[string]scheduling.EmissionEvaluator, len(in.Zones))
	for _, zone := range in.Zones {
		evaluator, ok := scheduling.BuildEmissionEvaluator(alignedForecasts[zone], windowEnd)
		if !ok {
			continue
		}
		evaluators[zone] = evaluator
	}

	bestFound := false
	bestEmission := 0.0
	bestScore := 0.0
	bestZone := ""
	bestStart := time.Time{}
	worstEmission := 0.0
	worstScore := 0.0
	worstFound := false

	for _, start := range timeAxis {
		for _, zone := range in.Zones {
			evaluator, ok := evaluators[zone]
			if !ok {
				continue
			}

			emission, ok := evaluator.EstimateAt(start.UTC(), in.Duration, model.Runner, model.Load, model.PUE)
			if !ok {
				continue
			}
			waitHours := maxFloat(start.UTC().Sub(requestStart).Hours(), 0)
			score := emission + in.WaitCost*waitHours

			if !bestFound || score < bestScore || (score == bestScore && emission < bestEmission) {
				bestFound = true
				bestEmission = emission
				bestScore = score
				bestZone = zone
				bestStart = start
			}

			if !worstFound || score > worstScore || (score == worstScore && emission > worstEmission) {
				worstFound = true
				worstEmission = emission
				worstScore = score
			}
		}
	}

	if !bestFound {
		return OptimizeGlobalOutput{}, fmt.Errorf("%w: no valid full window found across zones and timestamps", ErrNoValidWindow)
	}

	reduction := 0.0
	if worstFound && worstScore > 0 {
		reduction = (worstScore - bestScore) / worstScore * 100
	}

	return OptimizeGlobalOutput{
		BestZone:  bestZone,
		BestStart: bestStart.UTC(),
		BestEnd:   bestStart.Add(time.Duration(in.Duration) * time.Second).UTC(),
		Emission:  bestEmission,
		Reduction: reduction,
	}, nil
}
