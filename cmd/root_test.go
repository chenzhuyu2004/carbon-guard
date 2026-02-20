package cmd

import (
	"math"
	"testing"

	"github.com/czy/carbon-guard/internal/calculator"
	"github.com/czy/carbon-guard/internal/ci"
)

type fakeCIProvider struct {
	value         float64
	err           error
	calls         int
	lastZone      string
	forecast      []ci.ForecastPoint
	forecastErr   error
	forecastHours int
}

func (f *fakeCIProvider) GetCurrentCI(zone string) (float64, error) {
	f.calls++
	f.lastZone = zone
	return f.value, f.err
}

func (f *fakeCIProvider) GetForecastCI(zone string, hours int) ([]ci.ForecastPoint, error) {
	f.calls++
	f.lastZone = zone
	f.forecastHours = hours
	return f.forecast, f.forecastErr
}

func TestCalculateEmissionsUsesLiveCIProvider(t *testing.T) {
	provider := &fakeCIProvider{value: 0.8}

	got, err := calculateEmissions(300, "ubuntu", "eu", 0.6, 1.2, "", "DE", provider)
	if err != nil {
		t.Fatalf("calculateEmissions() unexpected error: %v", err)
	}

	expected := calculator.EstimateEmissionsWithSegments(
		[]calculator.Segment{{Duration: 300, CI: 0.8}},
		"ubuntu",
		0.6,
		1.2,
	)
	if math.Abs(got-expected) > 1e-9 {
		t.Fatalf("calculateEmissions() = %v, expected %v", got, expected)
	}

	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, expected 1", provider.calls)
	}
	if provider.lastZone != "DE" {
		t.Fatalf("provider zone = %q, expected %q", provider.lastZone, "DE")
	}
}
