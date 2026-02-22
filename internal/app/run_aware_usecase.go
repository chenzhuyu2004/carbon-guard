package app

import (
	"context"
	"fmt"
	"time"

	"github.com/chenzhuyu2004/carbon-guard/internal/domain/scheduling"
)

// RunAware waits for greener conditions and decides when to run.
// RunAware 根据碳强度等待更绿色时机，并决定何时执行。
//
// Deterministic rules:
// 1) Best forecast window is computed once at start.
// 2) Forecast is not refreshed in loop; only current CI is refreshed.
// 3) Exit when entering best window OR CI <= threshold-enter OR deadline exceeded.
// 确定性规则：
// 1) 最优 forecast 窗口仅在开始时计算一次。
// 2) 循环内不刷新 forecast，只刷新当前 CI。
// 3) 进入最优窗口 / CI<=threshold-enter / 超过截止时间时退出。
func (a *App) RunAware(ctx context.Context, in RunAwareInput) (RunAwareOutput, error) {
	if a == nil || a.provider == nil {
		return RunAwareOutput{}, fmt.Errorf("%w: provider is not configured", ErrProvider)
	}
	if in.Zone == "" {
		return RunAwareOutput{}, fmt.Errorf("%w: zone is required", ErrInput)
	}
	if err := validateDurationSeconds(in.Duration); err != nil {
		return RunAwareOutput{}, err
	}
	thresholdEnter, thresholdExit, err := resolveRunAwareThresholds(in)
	if err != nil {
		return RunAwareOutput{}, err
	}
	if err := validateLookaheadHours(in.Lookahead); err != nil {
		return RunAwareOutput{}, err
	}
	if err := validateDurationWithinLookahead(in.Duration, in.Lookahead); err != nil {
		return RunAwareOutput{}, err
	}
	if err := validateMaxWait(in.MaxWait); err != nil {
		return RunAwareOutput{}, err
	}
	if err := validateNoRegretConfig(in.NoRegretMaxDelay, in.NoRegretMinReductionPct); err != nil {
		return RunAwareOutput{}, err
	}
	model, err := normalizeModel(in.Model)
	if err != nil {
		return RunAwareOutput{}, err
	}

	startTime := time.Now().UTC()
	deadline := startTime.Add(in.MaxWait)

	analysis, err := a.AnalyzeBestWindow(ctx, in.Zone, in.Duration, in.Lookahead, startTime, model, 0)
	if err != nil {
		return RunAwareOutput{}, err
	}
	if !time.Now().UTC().Before(deadline) {
		return RunAwareOutput{}, fmt.Errorf("%w: Max wait exceeded", ErrMaxWaitExceeded)
	}

	bestStart := analysis.BestStart.UTC()
	bestEnd := analysis.BestEnd.UTC()
	if shouldRunNowByNoRegretGuard(startTime, bestStart, analysis.Reduction, in.NoRegretMaxDelay, in.NoRegretMinReductionPct) {
		message := fmt.Sprintf(
			"No-regret guard triggered: waiting %s for only %.2f%% expected reduction",
			bestStart.Sub(startTime).Round(time.Second),
			analysis.Reduction,
		)
		return RunAwareOutput{Message: message}, nil
	}

	pollEvery := in.PollEvery
	if pollEvery <= 0 {
		pollEvery = 15 * time.Minute
	}

	for {
		now := time.Now().UTC()
		if !now.Before(deadline) {
			return RunAwareOutput{}, fmt.Errorf("%w: Max wait exceeded", ErrMaxWaitExceeded)
		}

		// Window end is exclusive; reaching end means optimal window is missed.
		// 窗口右边界是开区间；到达 end 即视为错过最优窗口。
		if !now.Before(bestEnd) {
			return RunAwareOutput{}, fmt.Errorf("%w: Missed optimal window", ErrMissedOptimalWindow)
		}

		if scheduling.IsWithinWindow(now, bestStart, bestEnd) {
			return RunAwareOutput{Message: "Entering optimal carbon window"}, nil
		}

		currentCI, err := a.provider.GetCurrentCI(ctx, in.Zone)
		if err != nil {
			return RunAwareOutput{}, wrapProviderError(err)
		}

		if currentCI <= thresholdEnter {
			return RunAwareOutput{Message: "CI dropped below threshold-enter, running now"}, nil
		}

		// Wait duration is dynamically bounded by remaining max-wait budget.
		// 等待时长会被剩余 max-wait 预算动态限制。
		wait := pollEvery
		remaining := deadline.Sub(now)
		if remaining < wait {
			wait = remaining
		}
		if wait <= 0 {
			return RunAwareOutput{}, fmt.Errorf("%w: Max wait exceeded", ErrMaxWaitExceeded)
		}
		if in.StatusFunc != nil {
			if currentCI >= thresholdExit {
				in.StatusFunc(fmt.Sprintf("CI too high (%.2f >= %.2f)", currentCI, thresholdExit))
			} else {
				in.StatusFunc(fmt.Sprintf("CI in hysteresis band (%.2f < %.2f < %.2f)", thresholdEnter, currentCI, thresholdExit))
			}
			waitSeconds := int(wait.Round(time.Second).Seconds())
			if waitSeconds < 1 {
				waitSeconds = 1
			}
			in.StatusFunc(fmt.Sprintf("Waiting %ds...", waitSeconds))
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			if ctx.Err() == context.DeadlineExceeded {
				return RunAwareOutput{}, fmt.Errorf("%w: operation timed out", ErrTimeout)
			}
			return RunAwareOutput{}, fmt.Errorf("%w: %v", ErrProvider, ctx.Err())
		case <-timer.C:
		}
	}
}

func resolveRunAwareThresholds(in RunAwareInput) (float64, float64, error) {
	enter := in.ThresholdEnter
	exit := in.ThresholdExit
	legacy := in.Threshold

	if enter <= 0 {
		enter = legacy
	}
	if exit <= 0 {
		exit = legacy
	}
	if enter <= 0 {
		return 0, 0, fmt.Errorf("%w: threshold-enter must be > 0", ErrInput)
	}
	if exit <= 0 {
		return 0, 0, fmt.Errorf("%w: threshold-exit must be > 0", ErrInput)
	}
	if enter > exit {
		return 0, 0, fmt.Errorf("%w: threshold-enter must be <= threshold-exit", ErrInput)
	}

	return enter, exit, nil
}

func shouldRunNowByNoRegretGuard(
	startTime time.Time,
	bestStart time.Time,
	bestReductionPct float64,
	maxDelay time.Duration,
	minReductionPct float64,
) bool {
	if maxDelay <= 0 || minReductionPct <= 0 {
		return false
	}

	waitDelay := bestStart.UTC().Sub(startTime.UTC())
	if waitDelay <= 0 {
		return false
	}
	return waitDelay > maxDelay && bestReductionPct < minReductionPct
}
