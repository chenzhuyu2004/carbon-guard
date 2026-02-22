package app

import (
	"context"
	"fmt"
	"time"

	"github.com/chenzhuyu2004/carbon-guard/internal/domain/scheduling"
)

func (a *App) AnalyzeBestWindow(
	ctx context.Context,
	zone string,
	duration int,
	lookahead int,
	runner string,
	load float64,
	pue float64,
) (SuggestionAnalysis, error) {
	if a == nil || a.provider == nil {
		return SuggestionAnalysis{}, fmt.Errorf("%w: provider is not configured", ErrProvider)
	}
	if zone == "" {
		return SuggestionAnalysis{}, fmt.Errorf("%w: zone is required", ErrInput)
	}
	if duration <= 0 {
		return SuggestionAnalysis{}, fmt.Errorf("%w: duration must be > 0", ErrInput)
	}
	if lookahead <= 0 {
		return SuggestionAnalysis{}, fmt.Errorf("%w: lookahead must be > 0", ErrInput)
	}
	if load < 0 || load > 1 {
		return SuggestionAnalysis{}, fmt.Errorf("%w: load must be between 0 and 1", ErrInput)
	}
	if pue < 1.0 {
		return SuggestionAnalysis{}, fmt.Errorf("%w: pue must be >= 1.0", ErrInput)
	}
	if duration > lookahead*3600 {
		return SuggestionAnalysis{}, fmt.Errorf("%w: duration %ds exceeds lookahead window %ds", ErrInput, duration, lookahead*3600)
	}

	forecast, err := a.provider.GetForecastCI(ctx, zone, lookahead)
	if err != nil {
		return SuggestionAnalysis{}, wrapProviderError(err)
	}
	forecast = scheduling.NormalizeForecastUTC(forecast)
	if len(forecast) == 0 {
		return SuggestionAnalysis{}, fmt.Errorf("%w: no forecast points found for zone %s", ErrNoValidWindow, zone)
	}

	maxCoverage := scheduling.ForecastCoverageSeconds(forecast)
	if maxCoverage < duration {
		return SuggestionAnalysis{}, fmt.Errorf("%w: forecast does not cover full duration: need %ds but only %ds available", ErrNoValidWindow, duration, maxCoverage)
	}

	currentEmission, ok := scheduling.EstimateWindowEmissions(forecast, duration, runner, load, pue)
	if !ok {
		return SuggestionAnalysis{}, fmt.Errorf("%w: forecast does not cover full duration: need %ds within lookahead %dh", ErrNoValidWindow, duration, lookahead)
	}

	bestEmission := currentEmission
	currentStart := forecast[0].Timestamp.UTC()
	currentEnd := currentStart.Add(time.Duration(duration) * time.Second).UTC()
	bestStart := currentStart

	for i := 1; i < len(forecast); i++ {
		emission, ok := scheduling.EstimateWindowEmissions(forecast[i:], duration, runner, load, pue)
		if !ok {
			break
		}
		if emission < bestEmission {
			bestEmission = emission
			bestStart = forecast[i].Timestamp.UTC()
		}
	}

	bestStart = bestStart.UTC()
	bestEnd := bestStart.Add(time.Duration(duration) * time.Second).UTC()
	reduction := 0.0
	if currentEmission > 0 {
		reduction = (currentEmission - bestEmission) / currentEmission * 100
	}

	return SuggestionAnalysis{
		CurrentEmission: currentEmission,
		CurrentStart:    currentStart,
		CurrentEnd:      currentEnd,
		BestStart:       bestStart,
		BestEnd:         bestEnd,
		BestEmission:    bestEmission,
		Reduction:       reduction,
	}, nil
}

func (a *App) Suggest(ctx context.Context, in SuggestInput) (SuggestOutput, error) {
	if in.Zone == "" {
		return SuggestOutput{}, fmt.Errorf("%w: zone is required", ErrInput)
	}
	if in.Duration <= 0 {
		return SuggestOutput{}, fmt.Errorf("%w: duration must be > 0", ErrInput)
	}
	if in.Threshold <= 0 {
		return SuggestOutput{}, fmt.Errorf("%w: threshold must be > 0", ErrInput)
	}
	if in.Lookahead <= 0 {
		return SuggestOutput{}, fmt.Errorf("%w: lookahead must be > 0", ErrInput)
	}
	if in.Load < 0 || in.Load > 1 {
		return SuggestOutput{}, fmt.Errorf("%w: load must be between 0 and 1", ErrInput)
	}
	if in.PUE < 1.0 {
		return SuggestOutput{}, fmt.Errorf("%w: pue must be >= 1.0", ErrInput)
	}

	analysis, err := a.AnalyzeBestWindow(ctx, in.Zone, in.Duration, in.Lookahead, in.Runner, in.Load, in.PUE)
	if err != nil {
		return SuggestOutput{}, err
	}
	currentCI, err := a.provider.GetCurrentCI(ctx, in.Zone)
	if err != nil {
		return SuggestOutput{}, wrapProviderError(err)
	}

	bestStart := analysis.BestStart
	bestEnd := analysis.BestEnd
	bestEmission := analysis.BestEmission
	reduction := analysis.Reduction

	if currentCI <= in.Threshold && analysis.CurrentEmission <= analysis.BestEmission*1.05 {
		bestStart = analysis.CurrentStart
		bestEnd = analysis.CurrentEnd
		bestEmission = analysis.CurrentEmission
		reduction = 0
	}

	return SuggestOutput{
		CurrentCI:              currentCI,
		BestWindowStartUTC:     bestStart.UTC(),
		BestWindowEndUTC:       bestEnd.UTC(),
		ExpectedEmissionKg:     bestEmission,
		EmissionReductionVsNow: reduction,
	}, nil
}
