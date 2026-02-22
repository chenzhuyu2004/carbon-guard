package app

import (
	"context"
	"errors"
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

	_, err := a.AnalyzeBestWindow(context.Background(), "DE", 300, 1, ModelContext{
		Runner: "ubuntu",
		Load:   0.6,
		PUE:    1.2,
	})
	if !errors.Is(err, ErrNoValidWindow) {
		t.Fatalf("expected ErrNoValidWindow, got %v", err)
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
		Zone:      "DE",
		Duration:  300,
		Threshold: 0.1,
		Lookahead: 3,
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
