package app

import (
	"context"
	"fmt"
	"time"

	"github.com/chenzhuyu2004/carbon-guard/internal/calculator"
	"github.com/chenzhuyu2004/carbon-guard/internal/domain/scheduling"
)

// AnalyzeBestWindow evaluates candidate execution windows for a single zone.
// AnalyzeBestWindow 对单个区域的候选执行窗口进行评估。
//
// Objective:
//
//	score = emission_kg + wait_cost * wait_hours
//
// 目标函数：
//
//	score = emission_kg + wait_cost * wait_hours
//
// The function keeps backward compatibility when waitCost == 0
// (pure emission minimization).
// 当 waitCost == 0 时保持向后兼容（退化为纯排放最小化）。
func (a *App) AnalyzeBestWindow(
	ctx context.Context,
	zone string,
	duration int,
	lookahead int,
	evalStart time.Time,
	model ModelContext,
	waitCost float64,
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
	if err := validateWaitCost(waitCost); err != nil {
		return SuggestionAnalysis{}, err
	}

	// Use one explicit UTC anchor to keep multi-zone/multi-call comparisons stable.
	// 使用统一 UTC 锚点，保证多区域/多次调用的可比性与稳定性。
	evalStart = resolveEvalStart(evalStart)
	forecast, err := a.provider.GetForecastCI(ctx, zone, lookahead)
	if err != nil {
		return SuggestionAnalysis{}, wrapProviderError(err)
	}
	windowEnd := evalStart.Add(time.Duration(lookahead) * time.Hour).UTC()
	forecast = scheduling.NormalizeForecastUTC(forecast)
	// Clip in app layer so provider remains a pure transport adapter.
	// 在 app 层裁剪时间窗口，保持 provider 仅负责传输与解析。
	forecast = clipForecastToWindow(forecast, evalStart, lookahead)
	if len(forecast) == 0 {
		return SuggestionAnalysis{}, fmt.Errorf("%w: no forecast points found for zone %s", ErrNoValidWindow, zone)
	}

	evaluator, ok := scheduling.BuildEmissionEvaluator(forecast, windowEnd)
	if !ok {
		return SuggestionAnalysis{}, fmt.Errorf("%w: no forecast points found for zone %s", ErrNoValidWindow, zone)
	}

	maxCoverage := evaluator.CoverageSeconds()
	if maxCoverage < duration {
		return SuggestionAnalysis{}, fmt.Errorf("%w: forecast does not cover full duration: need %ds but only %ds available", ErrNoValidWindow, duration, maxCoverage)
	}

	currentWindow, bestWindow, ok := scheduling.FindBestWindowAtForecastStarts(
		forecast,
		evaluator,
		duration,
		model.Runner,
		model.Load,
		model.PUE,
	)
	if !ok {
		return SuggestionAnalysis{}, fmt.Errorf("%w: forecast does not cover full duration: need %ds within lookahead %dh", ErrNoValidWindow, duration, lookahead)
	}

	currentEmission := currentWindow.Emission
	currentStart := currentWindow.Start.UTC()
	currentEnd := currentWindow.End.UTC()
	// Wait penalty only applies to future windows; negative wait is clamped to zero.
	// 等待惩罚仅对未来窗口生效；负等待时间会被钳制为 0。
	currentScore := currentEmission + waitCost*maxFloat(currentStart.Sub(evalStart).Hours(), 0)
	bestStart := bestWindow.Start.UTC()
	bestEnd := bestWindow.End.UTC()
	bestEmission := bestWindow.Emission
	bestScore := bestEmission + waitCost*maxFloat(bestStart.Sub(evalStart).Hours(), 0)

	for _, point := range forecast {
		start := point.Timestamp.UTC()
		emission, ok := evaluator.EstimateAt(start, duration, model.Runner, model.Load, model.PUE)
		if !ok {
			break
		}

		score := emission + waitCost*maxFloat(start.Sub(evalStart).Hours(), 0)
		if score < bestScore || (score == bestScore && emission < bestEmission) {
			bestScore = score
			bestEmission = emission
			bestStart = start.UTC()
			bestEnd = start.Add(time.Duration(duration) * time.Second).UTC()
		}
	}

	reduction := 0.0
	if currentEmission > 0 {
		reduction = (currentEmission - bestEmission) / currentEmission * 100
	}

	return SuggestionAnalysis{
		CurrentEmission: currentEmission,
		CurrentStart:    currentStart,
		CurrentEnd:      currentEnd,
		CurrentScore:    currentScore,
		BestStart:       bestStart,
		BestEnd:         bestEnd,
		BestEmission:    bestEmission,
		BestScore:       bestScore,
		Reduction:       reduction,
	}, nil
}

// Suggest returns user-facing recommendation for one zone.
// Suggest 返回单区域的用户侧执行建议。
//
// It combines forecast-based optimization with current CI check:
// if "run now" score is close enough to best score and under threshold,
// it recommends immediate execution.
// 该函数将 forecast 优化与当前 CI 判断结合：
// 若“立即执行”的评分足够接近最优评分且低于阈值，则建议立即执行。
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
	if err := validateWaitCost(in.WaitCost); err != nil {
		return SuggestOutput{}, err
	}
	model, err := normalizeModel(in.Model)
	if err != nil {
		return SuggestOutput{}, err
	}
	evalStart := time.Now().UTC()

	analysis, err := a.AnalyzeBestWindow(ctx, in.Zone, in.Duration, in.Lookahead, evalStart, model, in.WaitCost)
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

	// "Run now" has zero waiting cost, so score == current emission.
	// “立即执行”不产生等待成本，因此 score 等于当前排放。
	nowScore := currentEmissionNow
	if currentCI <= in.Threshold && nowScore <= analysis.BestScore*1.05 {
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

// maxFloat returns the larger of a and b.
// maxFloat 返回 a 与 b 中的较大值。
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
