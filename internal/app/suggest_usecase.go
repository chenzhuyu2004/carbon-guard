package app

import (
	"context"
	"fmt"
	"time"

	"github.com/czy/carbon-guard/internal/domain/scheduling"
)

func (a *App) AnalyzeBestWindow(ctx context.Context, zone string, duration int, lookahead int) (SuggestionAnalysis, error) {
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
	if duration > lookahead*3600 {
		return SuggestionAnalysis{}, fmt.Errorf("%w: duration %ds exceeds lookahead window %ds", ErrInput, duration, lookahead*3600)
	}

	currentCI, err := a.provider.GetCurrentCI(ctx, zone)
	if err != nil {
		return SuggestionAnalysis{}, wrapProviderError(err)
	}

	forecast, err := a.provider.GetForecastCI(ctx, zone, lookahead)
	if err != nil {
		return SuggestionAnalysis{}, wrapProviderError(err)
	}
	if len(forecast) == 0 {
		return SuggestionAnalysis{}, fmt.Errorf("%w: no forecast points found for zone %s", ErrNoValidWindow, zone)
	}

	maxCoverage := len(forecast) * 3600
	if maxCoverage < duration {
		return SuggestionAnalysis{}, fmt.Errorf("%w: forecast does not cover full duration: need %ds but only %ds available", ErrNoValidWindow, duration, maxCoverage)
	}

	currentEmission, ok := scheduling.EstimateWindowEmissions(forecast, duration, "ubuntu", 0.6, 1.2)
	if !ok {
		return SuggestionAnalysis{}, fmt.Errorf("%w: forecast does not cover full duration: need %ds within lookahead %dh", ErrNoValidWindow, duration, lookahead)
	}

	bestEmission := currentEmission
	currentStart := forecast[0].Timestamp.UTC()
	currentEnd := currentStart.Add(time.Duration(duration) * time.Second).UTC()
	bestStart := currentStart

	for i := 1; i < len(forecast); i++ {
		emission, ok := scheduling.EstimateWindowEmissions(forecast[i:], duration, "ubuntu", 0.6, 1.2)
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
		CurrentCI:       currentCI,
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

	analysis, err := a.AnalyzeBestWindow(ctx, in.Zone, in.Duration, in.Lookahead)
	if err != nil {
		return SuggestOutput{}, err
	}

	bestStart := analysis.BestStart
	bestEnd := analysis.BestEnd
	bestEmission := analysis.BestEmission
	reduction := analysis.Reduction

	if analysis.CurrentCI <= in.Threshold && analysis.CurrentEmission <= analysis.BestEmission*1.05 {
		bestStart = analysis.CurrentStart
		bestEnd = analysis.CurrentEnd
		bestEmission = analysis.CurrentEmission
		reduction = 0
	}

	return SuggestOutput{
		CurrentCI:              analysis.CurrentCI,
		BestWindowStartUTC:     bestStart.UTC(),
		BestWindowEndUTC:       bestEnd.UTC(),
		ExpectedEmissionKg:     bestEmission,
		EmissionReductionVsNow: reduction,
	}, nil
}
