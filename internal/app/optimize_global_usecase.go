package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chenzhuyu2004/carbon-guard/internal/domain/scheduling"
)

// OptimizeGlobal finds the globally best execution plan across time and zones.
// OptimizeGlobal 在“时间 + 区域”联合空间内寻找全局最优执行方案。
//
// Objective:
//
//	score = emission_kg + wait_cost * wait_hours
//
// 目标函数：
//
//	score = emission_kg + wait_cost * wait_hours
//
// Forecast fetching is concurrent, but all scoring uses one UTC anchor (requestStart)
// to avoid cross-zone drift.
// forecast 拉取是并发的，但评分统一使用 requestStart（UTC）作为锚点，
// 以避免跨区域比较偏移。
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
	if err := validateResampleConfig(in.ResampleFillMode, in.ResampleMaxFillAge); err != nil {
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
			normalized := scheduling.NormalizeForecastUTC(forecast)
			// Apply lookahead clipping in app layer for deterministic orchestration.
			// 在 app 层执行 lookahead 裁剪，保证编排层语义可控且一致。
			normalized = clipForecastToWindow(normalized, requestStart, in.Lookahead)
			if len(normalized) == 0 {
				errCh <- fmt.Errorf("%w: zone %s failed: no forecast points in lookahead window", ErrNoValidWindow, zone)
				cancel()
				return
			}
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

	// Infer axis cadence from data, then apply explicit resample policy.
	// 先根据数据推断时间轴步长，再应用显式重采样策略。
	step := scheduling.InferResampleStep(zoneForecasts)
	resampleOptions := resolveResampleOptions(in, step)
	timeAxis, alignedForecasts := scheduling.BuildResampledIntersectionWithOptions(
		in.Zones,
		zoneForecasts,
		step,
		resampleOptions,
	)
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
			// Negative wait is clamped for safety; only future delay is penalized.
			// 对负等待时间进行钳制；仅惩罚未来等待。
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
		BestZone:                  bestZone,
		BestStart:                 bestStart.UTC(),
		BestEnd:                   bestStart.Add(time.Duration(in.Duration) * time.Second).UTC(),
		Emission:                  bestEmission,
		Reduction:                 reduction,
		ResampleFillMode:          string(resampleOptions.FillMode),
		ResampleMaxFillAgeSeconds: int64(resampleOptions.MaxFillAge.Round(time.Second).Seconds()),
	}, nil
}

// resolveResampleOptions normalizes CLI/app input into domain resample policy.
// resolveResampleOptions 将 CLI/app 输入归一化为 domain 重采样策略。
func resolveResampleOptions(in OptimizeGlobalInput, step time.Duration) scheduling.ResampleOptions {
	mode := strings.TrimSpace(strings.ToLower(in.ResampleFillMode))
	if mode == "" {
		mode = string(scheduling.FillModeForward)
	}

	options := scheduling.ResampleOptions{}
	switch mode {
	case string(scheduling.FillModeStrict):
		options.FillMode = scheduling.FillModeStrict
		options.MaxFillAge = 0
	default:
		options.FillMode = scheduling.FillModeForward
		if in.ResampleMaxFillAge > 0 {
			options.MaxFillAge = in.ResampleMaxFillAge
		} else {
			options.MaxFillAge = 2 * step
		}
	}

	return options
}
