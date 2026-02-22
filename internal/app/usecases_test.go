package app

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/chenzhuyu2004/carbon-guard/internal/domain/scheduling"
)

type fakeProvider struct {
	currentByZone  map[string]float64
	currentErr     error
	forecastByZone map[string][]scheduling.ForecastPoint
	forecastErr    error
}

func (f *fakeProvider) GetCurrentCI(_ context.Context, zone string) (float64, error) {
	if f.currentErr != nil {
		return 0, f.currentErr
	}
	if f.currentByZone == nil {
		return 0.4, nil
	}
	if v, ok := f.currentByZone[zone]; ok {
		return v, nil
	}
	return 0.4, nil
}

func (f *fakeProvider) GetForecastCI(_ context.Context, zone string, _ int) ([]scheduling.ForecastPoint, error) {
	if f.forecastErr != nil {
		return nil, f.forecastErr
	}
	if f.forecastByZone == nil {
		return nil, nil
	}
	if v, ok := f.forecastByZone[zone]; ok {
		return v, nil
	}
	return nil, nil
}

func TestSuggestValidationReturnsErrInput(t *testing.T) {
	a := New(&fakeProvider{})
	_, err := a.Suggest(context.Background(), SuggestInput{})
	if !errors.Is(err, ErrInput) {
		t.Fatalf("expected ErrInput, got %v", err)
	}
}

func TestAnalyzeBestWindowNoForecastReturnsErrNoValidWindow(t *testing.T) {
	a := New(&fakeProvider{
		currentByZone:  map[string]float64{"DE": 0.4},
		forecastByZone: map[string][]scheduling.ForecastPoint{"DE": {}},
	})

	_, err := a.AnalyzeBestWindow(context.Background(), "DE", 300, 1, time.Now().UTC(), ModelContext{
		Runner: "ubuntu",
		Load:   0.6,
		PUE:    1.2,
	}, 0)
	if !errors.Is(err, ErrNoValidWindow) {
		t.Fatalf("expected ErrNoValidWindow, got %v", err)
	}
}

func TestAnalyzeBestWindowWaitCostChangesBestWindow(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second).Add(10 * time.Minute)
	a := New(&fakeProvider{
		forecastByZone: map[string][]scheduling.ForecastPoint{
			"DE": {
				{Timestamp: now, CI: 0.8},
				{Timestamp: now.Add(time.Hour), CI: 0.1},
				{Timestamp: now.Add(2 * time.Hour), CI: 0.1},
			},
		},
	})

	model := ModelContext{
		Runner: "ubuntu",
		Load:   0.6,
		PUE:    1.2,
	}

	withoutWaitCost, err := a.AnalyzeBestWindow(context.Background(), "DE", 3600, 4, now, model, 0)
	if err != nil {
		t.Fatalf("AnalyzeBestWindow() unexpected error without wait-cost: %v", err)
	}
	if !withoutWaitCost.BestStart.Equal(now.Add(time.Hour)) {
		t.Fatalf("best start without wait-cost = %s, expected %s", withoutWaitCost.BestStart, now.Add(time.Hour))
	}

	withWaitCost, err := a.AnalyzeBestWindow(context.Background(), "DE", 3600, 4, now, model, 0.2)
	if err != nil {
		t.Fatalf("AnalyzeBestWindow() unexpected error with wait-cost: %v", err)
	}
	if !withWaitCost.BestStart.Equal(now) {
		t.Fatalf("best start with wait-cost = %s, expected %s", withWaitCost.BestStart, now)
	}
}

func TestAnalyzeBestWindowClipsForecastByEvalStart(t *testing.T) {
	evalStart := time.Now().UTC().Truncate(time.Second).Add(2 * time.Minute)
	a := New(&fakeProvider{
		forecastByZone: map[string][]scheduling.ForecastPoint{
			"DE": {
				{Timestamp: evalStart.Add(-30 * time.Minute), CI: 0.1},
				{Timestamp: evalStart.Add(30 * time.Minute), CI: 0.8},
				{Timestamp: evalStart.Add(90 * time.Minute), CI: 0.8},
			},
		},
	})

	analysis, err := a.AnalyzeBestWindow(context.Background(), "DE", 3600, 2, evalStart, ModelContext{
		Runner: "ubuntu",
		Load:   0.6,
		PUE:    1.2,
	}, 0)
	if err != nil {
		t.Fatalf("AnalyzeBestWindow() unexpected error: %v", err)
	}
	if analysis.CurrentStart.Before(evalStart) {
		t.Fatalf("current window start should be clipped to eval start, got %s, evalStart %s", analysis.CurrentStart, evalStart)
	}
}

func TestOptimizeUsesScoreWhenWaitCostProvided(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second).Add(10 * time.Minute)
	a := New(&fakeProvider{
		forecastByZone: map[string][]scheduling.ForecastPoint{
			"DE": {
				{Timestamp: now, CI: 0.8},
				{Timestamp: now.Add(time.Hour), CI: 0.1},
				{Timestamp: now.Add(2 * time.Hour), CI: 0.1},
			},
			"FR": {
				{Timestamp: now, CI: 0.4},
				{Timestamp: now.Add(time.Hour), CI: 0.4},
				{Timestamp: now.Add(2 * time.Hour), CI: 0.4},
			},
		},
	})

	out, err := a.Optimize(context.Background(), OptimizeInput{
		Zones:     []string{"DE", "FR"},
		Duration:  3600,
		Lookahead: 4,
		WaitCost:  0.2,
		Model: ModelContext{
			Runner: "ubuntu",
			Load:   0.6,
			PUE:    1.2,
		},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("Optimize() unexpected error: %v", err)
	}
	if len(out.Results) != 2 {
		t.Fatalf("results length = %d, expected 2", len(out.Results))
	}
	if out.Best.Zone != "FR" {
		t.Fatalf("best zone = %s, expected FR", out.Best.Zone)
	}
	if out.Results[0].Score > out.Results[1].Score {
		t.Fatalf("results are not sorted by score ascending")
	}
}

func TestSuggestImmediateRunUsesScoreObjective(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second).Add(5 * time.Minute)
	a := New(&fakeProvider{
		currentByZone: map[string]float64{"DE": 0.6},
		forecastByZone: map[string][]scheduling.ForecastPoint{
			"DE": {
				{Timestamp: now, CI: 0.6},
				{Timestamp: now.Add(time.Hour), CI: 0.1},
				{Timestamp: now.Add(2 * time.Hour), CI: 0.1},
			},
		},
	})

	out, err := a.Suggest(context.Background(), SuggestInput{
		Zone:      "DE",
		Duration:  3600,
		Threshold: 1.0,
		Lookahead: 3,
		WaitCost:  0.2,
		Model: ModelContext{
			Runner: "ubuntu",
			Load:   0.6,
			PUE:    1.2,
		},
	})
	if err != nil {
		t.Fatalf("Suggest() unexpected error: %v", err)
	}

	energyTotal := (3600.0 * (110.0 + (220.0-110.0)*0.6) / 1000.0 / 3600.0) * 1.2
	wantNow := energyTotal * 0.6

	if math.Abs(out.ExpectedEmissionKg-wantNow) > 1e-9 {
		t.Fatalf("ExpectedEmissionKg = %.12f, expected %.12f", out.ExpectedEmissionKg, wantNow)
	}
	if out.EmissionReductionVsNow != 0 {
		t.Fatalf("EmissionReductionVsNow = %.6f, expected 0", out.EmissionReductionVsNow)
	}
}

func TestRunLiveProviderMissingReturnsErrProvider(t *testing.T) {
	a := New(nil)
	_, err := a.Run(context.Background(), RunInput{
		Duration: 300,
		Region:   "global",
		LiveZone: "DE",
		Model: ModelContext{
			Runner: "ubuntu",
			Load:   0.6,
			PUE:    1.2,
		},
	})
	if !errors.Is(err, ErrProvider) {
		t.Fatalf("expected ErrProvider, got %v", err)
	}
}

func TestOptimizeAllFailuresReturnsErrProvider(t *testing.T) {
	a := New(&fakeProvider{forecastErr: errors.New("provider unavailable")})

	_, err := a.Optimize(context.Background(), OptimizeInput{
		Zones:     []string{"DE", "FR"},
		Duration:  300,
		Lookahead: 1,
		Model: ModelContext{
			Runner: "ubuntu",
			Load:   0.6,
			PUE:    1.2,
		},
		Timeout: time.Second,
	})
	if !errors.Is(err, ErrProvider) {
		t.Fatalf("expected ErrProvider, got %v", err)
	}
}

func TestOptimizeGlobalDurationValidationReturnsErrInput(t *testing.T) {
	a := New(&fakeProvider{})
	_, err := a.OptimizeGlobal(context.Background(), OptimizeGlobalInput{
		Zones:     []string{"DE"},
		Duration:  7201,
		Lookahead: 2,
		Model: ModelContext{
			Runner: "ubuntu",
			Load:   0.6,
			PUE:    1.2,
		},
		Timeout: time.Second,
	})
	if !errors.Is(err, ErrInput) {
		t.Fatalf("expected ErrInput, got %v", err)
	}
}

func TestOptimizeGlobalStrictResampleNoIntersectionReturnsErrNoValidWindow(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second).Add(10 * time.Minute)
	a := New(&fakeProvider{
		forecastByZone: map[string][]scheduling.ForecastPoint{
			"DE": {
				{Timestamp: now, CI: 0.4},
				{Timestamp: now.Add(time.Hour), CI: 0.5},
				{Timestamp: now.Add(2 * time.Hour), CI: 0.6},
			},
			"FR": {
				{Timestamp: now.Add(5 * time.Minute), CI: 0.3},
				{Timestamp: now.Add(65 * time.Minute), CI: 0.4},
				{Timestamp: now.Add(125 * time.Minute), CI: 0.5},
			},
		},
	})

	_, err := a.OptimizeGlobal(context.Background(), OptimizeGlobalInput{
		Zones:            []string{"DE", "FR"},
		Duration:         1800,
		Lookahead:        3,
		ResampleFillMode: "strict",
		Model: ModelContext{
			Runner: "ubuntu",
			Load:   0.6,
			PUE:    1.2,
		},
		Timeout: time.Second,
	})
	if !errors.Is(err, ErrNoValidWindow) {
		t.Fatalf("expected ErrNoValidWindow, got %v", err)
	}
}

func TestOptimizeGlobalInvalidResampleModeReturnsErrInput(t *testing.T) {
	a := New(&fakeProvider{})
	_, err := a.OptimizeGlobal(context.Background(), OptimizeGlobalInput{
		Zones:            []string{"DE", "FR"},
		Duration:         1800,
		Lookahead:        3,
		ResampleFillMode: "invalid",
		Model: ModelContext{
			Runner: "ubuntu",
			Load:   0.6,
			PUE:    1.2,
		},
		Timeout: time.Second,
	})
	if !errors.Is(err, ErrInput) {
		t.Fatalf("expected ErrInput, got %v", err)
	}
}

func TestRunAwareMaxWaitExceededReturnsErrMaxWaitExceeded(t *testing.T) {
	now := time.Now().UTC().Add(2 * time.Hour)
	a := New(&fakeProvider{
		currentByZone: map[string]float64{"DE": 0.9},
		forecastByZone: map[string][]scheduling.ForecastPoint{
			"DE": {
				{Timestamp: now, CI: 0.5},
				{Timestamp: now.Add(time.Hour), CI: 0.5},
			},
		},
	})

	_, err := a.RunAware(context.Background(), RunAwareInput{
		Zone:           "DE",
		Duration:       300,
		ThresholdEnter: 0.1,
		ThresholdExit:  0.2,
		Lookahead:      3,
		Model: ModelContext{
			Runner: "ubuntu",
			Load:   0.6,
			PUE:    1.2,
		},
		MaxWait:   20 * time.Millisecond,
		PollEvery: 5 * time.Millisecond,
	})
	if !errors.Is(err, ErrMaxWaitExceeded) {
		t.Fatalf("expected ErrMaxWaitExceeded, got %v", err)
	}
}

func TestRunAwareThresholdOrderValidationReturnsErrInput(t *testing.T) {
	now := time.Now().UTC().Add(2 * time.Hour)
	a := New(&fakeProvider{
		currentByZone: map[string]float64{"DE": 0.9},
		forecastByZone: map[string][]scheduling.ForecastPoint{
			"DE": {
				{Timestamp: now, CI: 0.5},
				{Timestamp: now.Add(time.Hour), CI: 0.5},
			},
		},
	})

	_, err := a.RunAware(context.Background(), RunAwareInput{
		Zone:           "DE",
		Duration:       300,
		ThresholdEnter: 0.5,
		ThresholdExit:  0.4,
		Lookahead:      3,
		Model: ModelContext{
			Runner: "ubuntu",
			Load:   0.6,
			PUE:    1.2,
		},
		MaxWait:   20 * time.Millisecond,
		PollEvery: 5 * time.Millisecond,
	})
	if !errors.Is(err, ErrInput) {
		t.Fatalf("expected ErrInput, got %v", err)
	}
}

func TestRunAwareRunsWhenBelowThresholdEnter(t *testing.T) {
	now := time.Now().UTC().Add(2 * time.Hour)
	a := New(&fakeProvider{
		currentByZone: map[string]float64{"DE": 0.39},
		forecastByZone: map[string][]scheduling.ForecastPoint{
			"DE": {
				{Timestamp: now, CI: 0.5},
				{Timestamp: now.Add(time.Hour), CI: 0.5},
			},
		},
	})

	out, err := a.RunAware(context.Background(), RunAwareInput{
		Zone:           "DE",
		Duration:       300,
		ThresholdEnter: 0.40,
		ThresholdExit:  0.45,
		Lookahead:      3,
		Model: ModelContext{
			Runner: "ubuntu",
			Load:   0.6,
			PUE:    1.2,
		},
		MaxWait:   200 * time.Millisecond,
		PollEvery: 5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("RunAware() unexpected error: %v", err)
	}
	if out.Message == "" {
		t.Fatalf("expected non-empty run-aware message")
	}
}
