package app

import (
	"context"
	"fmt"
	"time"

	"github.com/chenzhuyu2004/carbon-guard/internal/domain/scheduling"
)

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
	model, err := normalizeModel(in.Model)
	if err != nil {
		return RunAwareOutput{}, err
	}

	startTime := time.Now().UTC()
	deadline := startTime.Add(in.MaxWait)

	analysis, err := a.AnalyzeBestWindow(ctx, in.Zone, in.Duration, in.Lookahead, model, 0)
	if err != nil {
		return RunAwareOutput{}, err
	}
	if !time.Now().UTC().Before(deadline) {
		return RunAwareOutput{}, fmt.Errorf("%w: Max wait exceeded", ErrMaxWaitExceeded)
	}

	bestStart := analysis.BestStart.UTC()
	bestEnd := analysis.BestEnd.UTC()
	pollEvery := in.PollEvery
	if pollEvery <= 0 {
		pollEvery = 15 * time.Minute
	}

	for {
		now := time.Now().UTC()
		if !now.Before(deadline) {
			return RunAwareOutput{}, fmt.Errorf("%w: Max wait exceeded", ErrMaxWaitExceeded)
		}

		if now.After(bestEnd) {
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

		if in.StatusFunc != nil {
			if currentCI >= thresholdExit {
				in.StatusFunc(fmt.Sprintf("CI too high (%.2f >= %.2f)", currentCI, thresholdExit))
			} else {
				in.StatusFunc(fmt.Sprintf("CI in hysteresis band (%.2f < %.2f < %.2f)", thresholdEnter, currentCI, thresholdExit))
			}
			in.StatusFunc("Waiting 15m...")
		}

		wait := pollEvery
		remaining := deadline.Sub(time.Now().UTC())
		if remaining < wait {
			wait = remaining
		}
		if wait <= 0 {
			return RunAwareOutput{}, fmt.Errorf("%w: Max wait exceeded", ErrMaxWaitExceeded)
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
