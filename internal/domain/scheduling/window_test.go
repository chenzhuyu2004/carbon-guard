package scheduling

import (
	"math"
	"testing"
	"time"
)

func TestIsWithinWindow(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	if !IsWithinWindow(start, start, end) {
		t.Fatalf("expected start to be within window")
	}
	if !IsWithinWindow(start.Add(30*time.Minute), start, end) {
		t.Fatalf("expected middle point to be within window")
	}
	if IsWithinWindow(end, start, end) {
		t.Fatalf("expected end to be outside window (exclusive)")
	}
}

func TestEstimateWindowEmissionsCompleteWindow(t *testing.T) {
	points := []ForecastPoint{
		{Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), CI: 0.4},
		{Timestamp: time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC), CI: 0.8},
	}
	windowEnd := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	got, ok := EstimateWindowEmissions(points, 5400, "ubuntu", 0.5, 1.2, windowEnd)
	if !ok {
		t.Fatalf("EstimateWindowEmissions() reported incomplete window")
	}

	// runner=ubuntu -> idle=110W, peak=220W, load=0.5 => 165W
	power := 165.0
	energyFirst := 3600.0 * power / 1000.0 / 3600.0
	energySecond := 1800.0 * power / 1000.0 / 3600.0
	want := energyFirst*1.2*0.4 + energySecond*1.2*0.8

	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("EstimateWindowEmissions() = %.12f, expected %.12f", got, want)
	}
}

func TestEstimateWindowEmissionsIncompleteWindow(t *testing.T) {
	points := []ForecastPoint{
		{Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), CI: 0.4},
	}
	windowEnd := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	_, ok := EstimateWindowEmissions(points, 7200, "ubuntu", 0.6, 1.2, windowEnd)
	if ok {
		t.Fatalf("expected incomplete window when duration exceeds forecast coverage")
	}
}

func TestEstimateWindowEmissionsUsesActualTimestampIntervals(t *testing.T) {
	points := []ForecastPoint{
		{Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), CI: 0.2},
		{Timestamp: time.Date(2026, 1, 1, 10, 30, 0, 0, time.UTC), CI: 0.8},
	}
	windowEnd := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	got, ok := EstimateWindowEmissions(points, 3600, "ubuntu", 0.6, 1.2, windowEnd)
	if !ok {
		t.Fatalf("EstimateWindowEmissions() reported incomplete window")
	}

	// runner=ubuntu -> idle=110W, peak=220W, load=0.6 => 176W
	power := 176.0
	energyFirst := 1800.0 * power / 1000.0 / 3600.0
	energySecond := 1800.0 * power / 1000.0 / 3600.0
	want := energyFirst*1.2*0.2 + energySecond*1.2*0.8

	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("EstimateWindowEmissions() = %.12f, expected %.12f", got, want)
	}
}

func TestForecastCoverageSecondsUsesTimestampSpacing(t *testing.T) {
	points := []ForecastPoint{
		{Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), CI: 0.3},
		{Timestamp: time.Date(2026, 1, 1, 10, 30, 0, 0, time.UTC), CI: 0.4},
		{Timestamp: time.Date(2026, 1, 1, 11, 30, 0, 0, time.UTC), CI: 0.5},
	}
	windowEnd := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	got := ForecastCoverageSeconds(points, windowEnd)
	want := 1800 + 3600 + 1800
	if got != want {
		t.Fatalf("ForecastCoverageSeconds() = %d, expected %d", got, want)
	}
}
