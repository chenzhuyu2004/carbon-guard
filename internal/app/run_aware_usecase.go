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
	if in.Duration <= 0 {
		return RunAwareOutput{}, fmt.Errorf("%w: duration must be > 0", ErrInput)
	}
	if in.Threshold <= 0 {
		return RunAwareOutput{}, fmt.Errorf("%w: threshold must be > 0", ErrInput)
	}
	if in.Lookahead <= 0 {
		return RunAwareOutput{}, fmt.Errorf("%w: lookahead must be > 0", ErrInput)
	}
	if in.MaxWait <= 0 {
		return RunAwareOutput{}, fmt.Errorf("%w: max-wait must be > 0", ErrInput)
	}
	model, err := normalizeModel(in.Model)
	if err != nil {
		return RunAwareOutput{}, err
	}

	startTime := time.Now().UTC()
	deadline := startTime.Add(in.MaxWait)

	analysis, err := a.AnalyzeBestWindow(ctx, in.Zone, in.Duration, in.Lookahead, model)
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

		if currentCI <= in.Threshold {
			return RunAwareOutput{Message: "CI dropped below threshold, running now"}, nil
		}

		if in.StatusFunc != nil {
			in.StatusFunc(fmt.Sprintf("CI too high (%.2f > %.2f)", currentCI, in.Threshold))
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
