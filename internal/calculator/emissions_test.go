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
	idleExpected := float64(duration) * 110 / 1000.0 / 3600.0 * 1.0 * 0.4
	idleGot := EstimateEmissionsAdvanced(duration, "ubuntu", "global", 0, 1.0)
	if math.Abs(idleGot-idleExpected) > tolerance {
		t.Fatalf("load=0 (idle) = %v, expected %v", idleGot, idleExpected)
	}

	peakExpected := float64(duration) * 220 / 1000.0 / 3600.0 * 1.0 * 0.4
	peakGot := EstimateEmissionsAdvanced(duration, "ubuntu", "global", 1, 1.0)
	if math.Abs(peakGot-peakExpected) > tolerance {
		t.Fatalf("load=1 (peak) = %v, expected %v", peakGot, peakExpected)
	}

	fallbackRunnerPower := 110 + (220-110)*0.5
	fallbackRunnerExpected := float64(duration) * fallbackRunnerPower / 1000.0 / 3600.0 * 1.0 * 0.38
	fallbackRunnerGot := EstimateEmissionsAdvanced(duration, "unknown-runner", "us", 0.5, 1.0)
	if math.Abs(fallbackRunnerGot-fallbackRunnerExpected) > tolerance {
		t.Fatalf("unknown runner fallback = %v, expected %v", fallbackRunnerGot, fallbackRunnerExpected)
	}

	fallbackRegionPower := 150 + (300-150)*0.5
	fallbackRegionExpected := float64(duration) * fallbackRegionPower / 1000.0 / 3600.0 * 1.0 * 0.4
	fallbackRegionGot := EstimateEmissionsAdvanced(duration, "windows", "unknown-region", 0.5, 1.0)
	if math.Abs(fallbackRegionGot-fallbackRegionExpected) > tolerance {
		t.Fatalf("unknown region fallback = %v, expected %v", fallbackRegionGot, fallbackRegionExpected)
	}

	itEnergy := float64(duration) * 220 / 1000.0 / 3600.0
	pueOneExpected := itEnergy * 1.0 * 0.4
	pueOneGot := EstimateEmissionsAdvanced(duration, "ubuntu", "global", 1.0, 1.0)
	if math.Abs(pueOneGot-pueOneExpected) > tolerance {
		t.Fatalf("pue=1.0 = %v, expected %v", pueOneGot, pueOneExpected)
	}

	pueOneTwoExpected := itEnergy * 1.2 * 0.4
	pueOneTwoGot := EstimateEmissionsAdvanced(duration, "ubuntu", "global", 1.0, 1.2)
	if math.Abs(pueOneTwoGot-pueOneTwoExpected) > tolerance {
		t.Fatalf("pue=1.2 = %v, expected %v", pueOneTwoGot, pueOneTwoExpected)
	}
	if math.Abs(pueOneTwoGot-pueOneGot*1.2) > tolerance {
		t.Fatalf("pue scaling mismatch: got %v, expected %v", pueOneTwoGot, pueOneGot*1.2)
	}
}
