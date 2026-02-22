package app

import (
	"context"
	"fmt"
	"time"

	"github.com/chenzhuyu2004/carbon-guard/internal/calculator"
	"github.com/chenzhuyu2004/carbon-guard/internal/domain/scheduling"
)

func (a *App) AnalyzeBestWindow(
	ctx context.Context,
	zone string,
	duration int,
	lookahead int,
	model ModelContext,
) (SuggestionAnalysis, error) {
	if a == nil || a.provider == nil {
		return SuggestionAnalysis{}, fmt.Errorf("%w: provider is not configured", ErrProvider)
	}
	if zone == "" {
		return SuggestionAnalysis{}, fmt.Errorf("%w: zone is required", ErrInput)
	}
	if err := validateDurationSeconds(duration); err != nil {
		return SuggestionAnalysis{}, err
	}
	if err := validateLookaheadHours(lookahead); err != nil {
		return SuggestionAnalysis{}, err
	}
	var err error
	model, err = normalizeModel(model)
	if err != nil {
		return SuggestionAnalysis{}, err
	}
	if err := validateDurationWithinLookahead(duration, lookahead); err != nil {
		return SuggestionAnalysis{}, err
	}

	requestStart := time.Now().UTC()
	forecast, err := a.provider.GetForecastCI(ctx, zone, lookahead)
	if err != nil {
		return SuggestionAnalysis{}, wrapProviderError(err)
	}
	windowEnd := requestStart.Add(time.Duration(lookahead) * time.Hour).UTC()
	forecast = scheduling.NormalizeForecastUTC(forecast)
	if len(forecast) == 0 {
		return SuggestionAnalysis{}, fmt.Errorf("%w: no forecast points found for zone %s", ErrNoValidWindow, zone)
	}

	maxCoverage := scheduling.ForecastCoverageSeconds(forecast, windowEnd)
	if maxCoverage < duration {
		return SuggestionAnalysis{}, fmt.Errorf("%w: forecast does not cover full duration: need %ds but only %ds available", ErrNoValidWindow, duration, maxCoverage)
	}

	currentEmission, ok := scheduling.EstimateWindowEmissions(forecast, duration, model.Runner, model.Load, model.PUE, windowEnd)
	if !ok {
		return SuggestionAnalysis{}, fmt.Errorf("%w: forecast does not cover full duration: need %ds within lookahead %dh", ErrNoValidWindow, duration, lookahead)
	}

	bestEmission := currentEmission
	currentStart := forecast[0].Timestamp.UTC()
	currentEnd := currentStart.Add(time.Duration(duration) * time.Second).UTC()
	bestStart := currentStart

	for i := 1; i < len(forecast); i++ {
		emission, ok := scheduling.EstimateWindowEmissions(forecast[i:], duration, model.Runner, model.Load, model.PUE, windowEnd)
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
	if err := validateDurationSeconds(in.Duration); err != nil {
		return SuggestOutput{}, err
	}
	if in.Threshold <= 0 {
		return SuggestOutput{}, fmt.Errorf("%w: threshold must be > 0", ErrInput)
	}
	if err := validateLookaheadHours(in.Lookahead); err != nil {
		return SuggestOutput{}, err
	}
	if err := validateDurationWithinLookahead(in.Duration, in.Lookahead); err != nil {
		return SuggestOutput{}, err
	}
	model, err := normalizeModel(in.Model)
	if err != nil {
		return SuggestOutput{}, err
	}

	analysis, err := a.AnalyzeBestWindow(ctx, in.Zone, in.Duration, in.Lookahead, model)
	if err != nil {
		return SuggestOutput{}, err
	}
	currentCI, err := a.provider.GetCurrentCI(ctx, in.Zone)
	if err != nil {
		return SuggestOutput{}, wrapProviderError(err)
	}
	currentEmissionNow := calculator.EstimateEmissionsWithSegments(
		[]calculator.Segment{{Duration: in.Duration, CI: currentCI}},
		model.Runner,
		model.Load,
		model.PUE,
	)

	bestStart := analysis.BestStart
	bestEnd := analysis.BestEnd
	bestEmission := analysis.BestEmission
	reduction := 0.0
	if currentEmissionNow > 0 {
		reduction = (currentEmissionNow - bestEmission) / currentEmissionNow * 100
	}

	if currentCI <= in.Threshold && currentEmissionNow <= analysis.BestEmission*1.05 {
		bestStart = analysis.CurrentStart
		bestEnd = analysis.CurrentEnd
		bestEmission = currentEmissionNow
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
