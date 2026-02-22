package cmd

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"

	appsvc "github.com/chenzhuyu2004/carbon-guard/internal/app"
	"github.com/chenzhuyu2004/carbon-guard/internal/calculator"
	"github.com/chenzhuyu2004/carbon-guard/internal/ci"
	cgerrors "github.com/chenzhuyu2004/carbon-guard/internal/errors"
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

func (f *fakeCIProvider) GetCurrentCI(_ context.Context, zone string) (float64, error) {
	f.calls++
	f.lastZone = zone
	return f.value, f.err
}

func (f *fakeCIProvider) GetForecastCI(_ context.Context, zone string, hours int) ([]ci.ForecastPoint, error) {
	f.calls++
	f.lastZone = zone
	f.forecastHours = hours
	return f.forecast, f.forecastErr
}

func TestCalculateEmissionsUsesLiveCIProvider(t *testing.T) {
	provider := &fakeCIProvider{value: 0.8}
	service := appsvc.New(newProviderAdapter(provider))

	got, err := service.Run(context.Background(), appsvc.RunInput{
		Duration: 300,
		Runner:   "ubuntu",
		Region:   "eu",
		Load:     0.6,
		PUE:      1.2,
		LiveZone: "DE",
	})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	expected := calculator.EstimateEmissionsWithSegments(
		[]calculator.Segment{{Duration: 300, CI: 0.8}},
		"ubuntu",
		0.6,
		1.2,
	)
	if math.Abs(got.EmissionsKg-expected) > 1e-9 {
		t.Fatalf("Run().EmissionsKg = %v, expected %v", got.EmissionsKg, expected)
	}

	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, expected 1", provider.calls)
	}
	if provider.lastZone != "DE" {
		t.Fatalf("provider zone = %q, expected %q", provider.lastZone, "DE")
	}
}

func TestSplitZonesNormalizesAndDropsEmpty(t *testing.T) {
	got := splitZones(" de, FR , ,pl ")
	want := []string{"DE", "FR", "PL"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitZones() = %#v, expected %#v", got, want)
	}
}

func TestAnalyzeBestWindowDurationExceedsLookahead(t *testing.T) {
	provider := &fakeCIProvider{}
	service := appsvc.New(newProviderAdapter(provider))
	_, err := service.AnalyzeBestWindow(context.Background(), "DE", 7201, 2, "ubuntu", 0.6, 1.2)
	if err == nil {
		t.Fatalf("expected error when duration exceeds lookahead")
	}
}

func TestAnalyzeBestWindowCoverageError(t *testing.T) {
	now := time.Now().UTC()
	provider := &fakeCIProvider{
		value: 0.4,
		forecast: []ci.ForecastPoint{
			{Timestamp: now, CI: 0.4},
		},
	}

	service := appsvc.New(newProviderAdapter(provider))
	_, err := service.AnalyzeBestWindow(context.Background(), "DE", 7200, 3, "ubuntu", 0.6, 1.2)
	if err == nil {
		t.Fatalf("expected coverage error")
	}
}

func TestRunBudgetExceededReturnsDedicatedCode(t *testing.T) {
	err := run([]string{
		"--duration", "300",
		"--budget-kg", "0.001",
		"--fail-on-budget",
	})
	if err == nil {
		t.Fatalf("expected budget exceeded error")
	}

	if code := cgerrors.GetCode(err); code != cgerrors.BudgetExceeded {
		t.Fatalf("error code = %d, expected %d", code, cgerrors.BudgetExceeded)
	}
}
