package ci

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type retryStubProvider struct {
	mu            sync.Mutex
	currentCalls  int
	forecastCalls int
	currentValue  float64
	currentErrs   []error
}

func (p *retryStubProvider) GetCurrentCI(_ context.Context, _ string) (float64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentCalls++
	if len(p.currentErrs) > 0 {
		err := p.currentErrs[0]
		p.currentErrs = p.currentErrs[1:]
		return 0, err
	}
	return p.currentValue, nil
}

func (p *retryStubProvider) GetForecastCI(_ context.Context, _ string, _ int) ([]ForecastPoint, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.forecastCalls++
	return []ForecastPoint{{Timestamp: time.Now().UTC(), CI: 0.4}}, nil
}

func TestRetryMiddlewareRetriesHTTP429(t *testing.T) {
	stub := &retryStubProvider{
		currentValue: 0.42,
		currentErrs: []error{
			&HTTPStatusError{StatusCode: 429, Status: "429 Too Many Requests", Body: "rate limited"},
			&HTTPStatusError{StatusCode: 503, Status: "503 Service Unavailable", Body: "overloaded"},
		},
	}

	p := WithRetry(RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    2 * time.Millisecond,
		Jitter:      0,
	})(stub)

	got, err := p.GetCurrentCI(context.Background(), "DE")
	if err != nil {
		t.Fatalf("GetCurrentCI() unexpected error: %v", err)
	}
	if got != 0.42 {
		t.Fatalf("GetCurrentCI() = %v, expected 0.42", got)
	}

	stub.mu.Lock()
	calls := stub.currentCalls
	stub.mu.Unlock()
	if calls != 3 {
		t.Fatalf("current calls = %d, expected 3", calls)
	}
}

func TestRetryMiddlewareDoesNotRetryInputErrors(t *testing.T) {
	baseErr := errors.New("invalid zone")
	stub := &retryStubProvider{
		currentErrs: []error{baseErr},
	}

	p := WithRetry(RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    1 * time.Millisecond,
		Jitter:      0,
	})(stub)

	_, err := p.GetCurrentCI(context.Background(), "DE")
	if !errors.Is(err, baseErr) {
		t.Fatalf("expected original error, got %v", err)
	}

	stub.mu.Lock()
	calls := stub.currentCalls
	stub.mu.Unlock()
	if calls != 1 {
		t.Fatalf("current calls = %d, expected 1", calls)
	}
}
