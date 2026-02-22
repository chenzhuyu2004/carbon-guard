package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/chenzhuyu2004/carbon-guard/internal/calculator"
	"github.com/chenzhuyu2004/carbon-guard/pkg/models"
)

type runComputation struct {
	DurationSeconds int
	EmissionsKg     float64
	EnergyITKWh     float64
	EnergyTotalKWh  float64
}

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

	computation, err := a.calculateEmissions(ctx, in)
	if err != nil {
		return RunResult{}, err
	}
	effectiveCI := 0.0
	if computation.EnergyTotalKWh > 0 {
		effectiveCI = computation.EmissionsKg / computation.EnergyTotalKWh
	}

	return RunResult{
		DurationSeconds:     computation.DurationSeconds,
		EmissionsKg:         computation.EmissionsKg,
		EnergyITKWh:         computation.EnergyITKWh,
		EnergyTotalKWh:      computation.EnergyTotalKWh,
		EffectiveCIKgPerKWh: effectiveCI,
	}, nil
}

func (a *App) calculateEmissions(ctx context.Context, in RunInput) (runComputation, error) {
	if in.SegmentsRaw != "" {
		segments, err := parseSegments(in.SegmentsRaw)
		if err != nil {
			return runComputation{}, err
		}
		duration := sumSegmentDurations(segments)
		energyIT, energyTotal := estimateEnergyKWh(duration, in.Runner, in.Load, in.PUE)
		return runComputation{
			DurationSeconds: duration,
			EmissionsKg:     calculator.EstimateEmissionsWithSegments(segments, in.Runner, in.Load, in.PUE),
			EnergyITKWh:     energyIT,
			EnergyTotalKWh:  energyTotal,
		}, nil
	}

	if in.LiveZone != "" {
		if a == nil || a.provider == nil {
			return runComputation{}, fmt.Errorf("%w: live ci provider is not configured", ErrProvider)
		}
		ciValue, err := a.provider.GetCurrentCI(ctx, in.LiveZone)
		if err != nil {
			return runComputation{}, wrapProviderError(err)
		}
		segments := []calculator.Segment{{Duration: in.Duration, CI: ciValue}}
		energyIT, energyTotal := estimateEnergyKWh(in.Duration, in.Runner, in.Load, in.PUE)
		return runComputation{
			DurationSeconds: in.Duration,
			EmissionsKg:     calculator.EstimateEmissionsWithSegments(segments, in.Runner, in.Load, in.PUE),
			EnergyITKWh:     energyIT,
			EnergyTotalKWh:  energyTotal,
		}, nil
	}

	energyIT, energyTotal := estimateEnergyKWh(in.Duration, in.Runner, in.Load, in.PUE)
	return runComputation{
		DurationSeconds: in.Duration,
		EmissionsKg:     calculator.EstimateEmissionsAdvanced(in.Duration, in.Runner, in.Region, in.Load, in.PUE),
		EnergyITKWh:     energyIT,
		EnergyTotalKWh:  energyTotal,
	}, nil
}

func estimateEnergyKWh(duration int, runner string, load float64, pue float64) (float64, float64) {
	profile, ok := models.RunnerProfiles[runner]
	if !ok {
		profile = models.RunnerProfiles["ubuntu"]
	}

	power := profile.Idle + (profile.Peak-profile.Idle)*load
	energyIT := float64(duration) * power / 1000.0 / 3600.0
	return energyIT, energyIT * pue
}

func sumSegmentDurations(segments []calculator.Segment) int {
	total := 0
	for _, segment := range segments {
		total += segment.Duration
	}
	return total
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
