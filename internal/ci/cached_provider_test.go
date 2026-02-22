package ci

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type fakeInnerProvider struct {
	current       float64
	currentErr    error
	forecast      []ForecastPoint
	forecastErr   error
	currentCalls  int
	forecastCalls int
}

func (f *fakeInnerProvider) GetCurrentCI(_ context.Context, _ string) (float64, error) {
	f.currentCalls++
	if f.currentErr != nil {
		return 0, f.currentErr
	}
	return f.current, nil
}

func (f *fakeInnerProvider) GetForecastCI(_ context.Context, _ string, _ int) ([]ForecastPoint, error) {
	f.forecastCalls++
	if f.forecastErr != nil {
		return nil, f.forecastErr
	}
	return f.forecast, nil
}

func TestCachedProviderGetForecastCIUsesCacheWithinTTL(t *testing.T) {
	dir := t.TempDir()
	inner := &fakeInnerProvider{
		forecast: []ForecastPoint{{Timestamp: time.Now().UTC().Add(time.Minute), CI: 0.4}},
	}
	provider := &CachedProvider{Inner: inner, CacheDir: dir, TTL: 10 * time.Minute}

	ctx := context.Background()
	first, err := provider.GetForecastCI(ctx, "DE", 2)
	if err != nil {
		t.Fatalf("first GetForecastCI() error: %v", err)
	}
	second, err := provider.GetForecastCI(ctx, "DE", 2)
	if err != nil {
		t.Fatalf("second GetForecastCI() error: %v", err)
	}

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected forecast lengths: %d and %d", len(first), len(second))
	}
	if inner.forecastCalls != 1 {
		t.Fatalf("inner forecast calls = %d, expected 1", inner.forecastCalls)
	}

	cacheFile := filepath.Join(dir, "forecast_DE_2.json")
	if _, err := os.Stat(cacheFile); err != nil {
		t.Fatalf("expected cache file at %s: %v", cacheFile, err)
	}
}

func TestCachedProviderGetForecastCIExpiresCache(t *testing.T) {
	inner := &fakeInnerProvider{
		forecast: []ForecastPoint{{Timestamp: time.Now().UTC().Add(time.Minute), CI: 0.4}},
	}
	provider := &CachedProvider{Inner: inner, CacheDir: t.TempDir(), TTL: time.Millisecond}

	ctx := context.Background()
	if _, err := provider.GetForecastCI(ctx, "DE", 2); err != nil {
		t.Fatalf("first GetForecastCI() error: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	if _, err := provider.GetForecastCI(ctx, "DE", 2); err != nil {
		t.Fatalf("second GetForecastCI() error: %v", err)
	}

	if inner.forecastCalls != 2 {
		t.Fatalf("inner forecast calls = %d, expected 2 after ttl expiry", inner.forecastCalls)
	}
}

func TestCachedProviderRespectsContextCancellation(t *testing.T) {
	inner := &fakeInnerProvider{
		forecast: []ForecastPoint{{Timestamp: time.Now().UTC().Add(time.Minute), CI: 0.4}},
	}
	provider := &CachedProvider{Inner: inner, CacheDir: t.TempDir(), TTL: 10 * time.Minute}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.GetForecastCI(ctx, "DE", 2)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if inner.forecastCalls != 0 {
		t.Fatalf("inner provider should not be called when context canceled")
	}
}

type blockingInnerProvider struct {
	mu            sync.Mutex
	forecastCalls int
	release       chan struct{}
}

func (b *blockingInnerProvider) GetCurrentCI(_ context.Context, _ string) (float64, error) {
	return 0.4, nil
}

func (b *blockingInnerProvider) GetForecastCI(_ context.Context, _ string, _ int) ([]ForecastPoint, error) {
	b.mu.Lock()
	b.forecastCalls++
	b.mu.Unlock()
	<-b.release
	return []ForecastPoint{{Timestamp: time.Now().UTC().Add(time.Minute), CI: 0.4}}, nil
}

func TestCachedProviderDeduplicatesConcurrentMisses(t *testing.T) {
	inner := &blockingInnerProvider{release: make(chan struct{})}
	provider := &CachedProvider{
		Inner:    inner,
		CacheDir: t.TempDir(),
		TTL:      10 * time.Minute,
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := provider.GetForecastCI(context.Background(), "DE", 2)
			errCh <- err
		}()
	}

	time.Sleep(10 * time.Millisecond)
	close(inner.release)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("GetForecastCI() returned error: %v", err)
		}
	}

	inner.mu.Lock()
	calls := inner.forecastCalls
	inner.mu.Unlock()
	if calls != 1 {
		t.Fatalf("inner forecast calls = %d, expected 1", calls)
	}
}
