package calculator

import (
	"math"
	"testing"
)

func TestEstimateEmissionsKg(t *testing.T) {
	durationSeconds := 300
	expected := float64(durationSeconds) * 200 * 0.4 / 1000 / 3600

	got := EstimateEmissionsKg(durationSeconds)
	if math.Abs(got-expected) > 1e-9 {
		t.Fatalf("EstimateEmissionsKg(%d) = %v, expected %v", durationSeconds, got, expected)
	}
}

func TestEstimateEmissionsAdvanced(t *testing.T) {
	tolerance := 1e-9

	duration := 300
	load := 0.6
	expected := float64(duration) * 220 * load / 1000.0 / 3600.0 * 0.4
	got := EstimateEmissionsAdvanced(duration, "ubuntu", "global", load)
	if math.Abs(got-expected) > tolerance {
		t.Fatalf("EstimateEmissionsAdvanced(%d, ubuntu, global, %v) = %v, expected %v", duration, load, got, expected)
	}

	fallbackRunnerExpected := float64(duration) * 220 * 0.5 / 1000.0 / 3600.0 * 0.38
	fallbackRunnerGot := EstimateEmissionsAdvanced(duration, "unknown-runner", "us", 0.5)
	if math.Abs(fallbackRunnerGot-fallbackRunnerExpected) > tolerance {
		t.Fatalf("unknown runner fallback = %v, expected %v", fallbackRunnerGot, fallbackRunnerExpected)
	}

	fallbackRegionExpected := float64(duration) * 300 * 0.5 / 1000.0 / 3600.0 * 0.4
	fallbackRegionGot := EstimateEmissionsAdvanced(duration, "windows", "unknown-region", 0.5)
	if math.Abs(fallbackRegionGot-fallbackRegionExpected) > tolerance {
		t.Fatalf("unknown region fallback = %v, expected %v", fallbackRegionGot, fallbackRegionExpected)
	}
}
