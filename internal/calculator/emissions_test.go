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
