package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/chenzhuyu2004/carbon-guard/internal/calculator"
)

func (a *App) Run(ctx context.Context, in RunInput) (RunResult, error) {
	if in.Duration <= 0 {
		return RunResult{}, fmt.Errorf("%w: duration must be > 0", ErrInput)
	}
	if in.Load < 0 || in.Load > 1 {
		return RunResult{}, fmt.Errorf("%w: load must be between 0 and 1", ErrInput)
	}
	if in.PUE < 1.0 {
		return RunResult{}, fmt.Errorf("%w: pue must be >= 1.0", ErrInput)
	}

	emissions, err := a.calculateEmissions(ctx, in)
	if err != nil {
		return RunResult{}, err
	}

	return RunResult{
		DurationSeconds: in.Duration,
		EmissionsKg:     emissions,
	}, nil
}

func (a *App) calculateEmissions(ctx context.Context, in RunInput) (float64, error) {
	if in.SegmentsRaw != "" {
		segments, err := parseSegments(in.SegmentsRaw)
		if err != nil {
			return 0, err
		}
		return calculator.EstimateEmissionsWithSegments(segments, in.Runner, in.Load, in.PUE), nil
	}

	if in.LiveZone != "" {
		if a == nil || a.provider == nil {
			return 0, fmt.Errorf("%w: live ci provider is not configured", ErrProvider)
		}
		ciValue, err := a.provider.GetCurrentCI(ctx, in.LiveZone)
		if err != nil {
			return 0, wrapProviderError(err)
		}
		segments := []calculator.Segment{{Duration: in.Duration, CI: ciValue}}
		return calculator.EstimateEmissionsWithSegments(segments, in.Runner, in.Load, in.PUE), nil
	}

	return calculator.EstimateEmissionsAdvanced(in.Duration, in.Runner, in.Region, in.Load, in.PUE), nil
}

func parseSegments(raw string) ([]calculator.Segment, error) {
	items := strings.Split(raw, ",")
	segments := make([]calculator.Segment, 0, len(items))

	for _, item := range items {
		item = strings.TrimSpace(item)
		parts := strings.Split(item, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("%w: invalid segment format: %s", ErrInput, item)
		}

		duration, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || duration <= 0 {
			return nil, fmt.Errorf("%w: invalid segment duration: %s", ErrInput, parts[0])
		}

		ci, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil || ci <= 0 {
			return nil, fmt.Errorf("%w: invalid segment ci: %s", ErrInput, parts[1])
		}

		segments = append(segments, calculator.Segment{
			Duration: duration,
			CI:       ci,
		})
	}

	return segments, nil
}
