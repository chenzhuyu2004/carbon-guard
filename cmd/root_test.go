package cmd

import (
	"context"
	"math"
	"os"
	"path/filepath"
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
		Region:   "eu",
		LiveZone: "DE",
		Model: appsvc.ModelContext{
			Runner: "ubuntu",
			Load:   0.6,
			PUE:    1.2,
		},
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
	_, err := service.AnalyzeBestWindow(context.Background(), "DE", 7201, 2, appsvc.ModelContext{
		Runner: "ubuntu",
		Load:   0.6,
		PUE:    1.2,
	}, 0)
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
	_, err := service.AnalyzeBestWindow(context.Background(), "DE", 7200, 3, appsvc.ModelContext{
		Runner: "ubuntu",
		Load:   0.6,
		PUE:    1.2,
	}, 0)
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

func TestDetectJSONOutputRun(t *testing.T) {
	if !detectJSONOutput("run", []string{"--duration", "300", "--json"}) {
		t.Fatalf("expected run output mode to detect json")
	}
	if detectJSONOutput("run", []string{"--duration", "300", "--json=false"}) {
		t.Fatalf("expected run output mode to detect non-json when json=false")
	}
}

func TestDetectJSONOutputOptimize(t *testing.T) {
	if !detectJSONOutput("optimize", []string{"--zones", "DE,FR", "--duration", "300", "--output=json"}) {
		t.Fatalf("expected optimize output mode to detect json with equals syntax")
	}
	if !detectJSONOutput("optimize-global", []string{"--zones", "DE,FR", "--duration", "300", "--output", "json"}) {
		t.Fatalf("expected optimize-global output mode to detect json with split syntax")
	}
	if detectJSONOutput("optimize", []string{"--zones", "DE,FR", "--duration", "300", "--output", "text"}) {
		t.Fatalf("expected optimize output mode to detect text")
	}
}

func TestDetectJSONOutputOptimizeFromEnvDefault(t *testing.T) {
	t.Setenv("CARBON_GUARD_OUTPUT", "json")
	t.Setenv("CARBON_GUARD_CONFIG", "")
	if !detectJSONOutput("optimize", []string{"--zones", "DE", "--duration", "300"}) {
		t.Fatalf("expected optimize output mode to detect json from env default")
	}
}

func TestDetectJSONOutputOptimizeFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "carbon-guard.json")
	content := []byte(`{"output":"json"}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() unexpected error: %v", err)
	}

	if !detectJSONOutput("optimize-global", []string{"--config", path, "--zones", "DE", "--duration", "300"}) {
		t.Fatalf("expected optimize-global output mode to detect json from config file")
	}
}
