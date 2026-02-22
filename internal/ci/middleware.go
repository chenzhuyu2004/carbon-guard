package ci

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

type Middleware func(Provider) Provider

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Jitter      float64
}

type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
}

type PipelineConfig struct {
	Timeout   time.Duration
	Retry     RetryConfig
	RateLimit RateLimitConfig
	CacheDir  string
	CacheTTL  time.Duration
	Metrics   MetricsRecorder
}

type MetricsRecorder interface {
	ObserveCall(operation string, zone string, duration time.Duration, err error)
}

type NopMetricsRecorder struct{}

func (NopMetricsRecorder) ObserveCall(string, string, time.Duration, error) {}

func Chain(base Provider, middlewares ...Middleware) Provider {
	p := base
	for i := len(middlewares) - 1; i >= 0; i-- {
		if middlewares[i] == nil {
			continue
		}
		p = middlewares[i](p)
	}
	return p
}

func NewPipeline(base Provider, cfg PipelineConfig) Provider {
	if base == nil {
		return nil
	}

	p := base
	if cfg.Timeout > 0 {
		p = WithTimeout(cfg.Timeout)(p)
	}
	if cfg.Retry.MaxAttempts > 1 {
		p = WithRetry(cfg.Retry)(p)
	}
	if cfg.RateLimit.RequestsPerSecond > 0 {
		p = WithRateLimit(cfg.RateLimit)(p)
	}
	if cfg.CacheDir != "" && cfg.CacheTTL >= 0 {
		p = &CachedProvider{
			Inner:    p,
			CacheDir: cfg.CacheDir,
			TTL:      cfg.CacheTTL,
		}
	}
	if cfg.Metrics != nil {
		p = WithMetrics(cfg.Metrics)(p)
	}

	return p
}

func WithTimeout(timeout time.Duration) Middleware {
	return func(next Provider) Provider {
		return &timeoutProvider{
			next:    next,
			timeout: timeout,
		}
	}
}

type timeoutProvider struct {
	next    Provider
	timeout time.Duration
}

func (p *timeoutProvider) GetCurrentCI(ctx context.Context, zone string) (float64, error) {
	callCtx, cancel := withCallTimeout(ctx, p.timeout)
	defer cancel()
	return p.next.GetCurrentCI(callCtx, zone)
}

func (p *timeoutProvider) GetForecastCI(ctx context.Context, zone string, hours int) ([]ForecastPoint, error) {
	callCtx, cancel := withCallTimeout(ctx, p.timeout)
	defer cancel()
	return p.next.GetForecastCI(callCtx, zone, hours)
}

func withCallTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining <= timeout {
			return ctx, func() {}
		}
	}
	return context.WithTimeout(ctx, timeout)
}

func WithRetry(cfg RetryConfig) Middleware {
	return func(next Provider) Provider {
		return &retryProvider{
			next: next,
			cfg:  normalizeRetryConfig(cfg),
			rng:  newLockedRand(),
		}
	}
}

type retryProvider struct {
	next Provider
	cfg  RetryConfig
	rng  *lockedRand
}

func (p *retryProvider) GetCurrentCI(ctx context.Context, zone string) (float64, error) {
	var value float64
	err := p.retry(ctx, func(callCtx context.Context) error {
		v, err := p.next.GetCurrentCI(callCtx, zone)
		if err != nil {
			return err
		}
		value = v
		return nil
	})
	return value, err
}

func (p *retryProvider) GetForecastCI(ctx context.Context, zone string, hours int) ([]ForecastPoint, error) {
	var points []ForecastPoint
	err := p.retry(ctx, func(callCtx context.Context) error {
		v, err := p.next.GetForecastCI(callCtx, zone, hours)
		if err != nil {
			return err
		}
		points = v
		return nil
	})
	return points, err
}

func (p *retryProvider) retry(ctx context.Context, call func(context.Context) error) error {
	var lastErr error
	for attempt := 1; attempt <= p.cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := call(ctx)
		if err == nil {
			return nil
		}
		lastErr = err

		if attempt == p.cfg.MaxAttempts || !isRetryableError(err) {
			break
		}

		delay := p.backoffDelay(attempt)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return lastErr
}

func (p *retryProvider) backoffDelay(attempt int) time.Duration {
	delay := p.cfg.BaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= p.cfg.MaxDelay {
			delay = p.cfg.MaxDelay
			break
		}
	}

	if p.cfg.Jitter <= 0 {
		return delay
	}

	spread := float64(delay) * p.cfg.Jitter
	random := (p.rng.Float64()*2 - 1) * spread
	jittered := float64(delay) + random
	if jittered < float64(time.Millisecond) {
		return time.Millisecond
	}
	return time.Duration(jittered)
}

func normalizeRetryConfig(cfg RetryConfig) RetryConfig {
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 200 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 2 * time.Second
	}
	if cfg.MaxDelay < cfg.BaseDelay {
		cfg.MaxDelay = cfg.BaseDelay
	}
	if cfg.Jitter < 0 {
		cfg.Jitter = 0
	}
	if cfg.Jitter > 1 {
		cfg.Jitter = 1
	}
	return cfg
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var statusErr *HTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode == 429 || statusErr.StatusCode >= 500
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "status: 429") || strings.Contains(text, "status: 500") || strings.Contains(text, "status: 502") || strings.Contains(text, "status: 503") || strings.Contains(text, "status: 504")
}

func WithRateLimit(cfg RateLimitConfig) Middleware {
	return func(next Provider) Provider {
		return &rateLimitProvider{
			next:    next,
			limiter: newTokenBucket(cfg),
		}
	}
}

type rateLimitProvider struct {
	next    Provider
	limiter *tokenBucket
}

func (p *rateLimitProvider) GetCurrentCI(ctx context.Context, zone string) (float64, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return 0, err
	}
	return p.next.GetCurrentCI(ctx, zone)
}

func (p *rateLimitProvider) GetForecastCI(ctx context.Context, zone string, hours int) ([]ForecastPoint, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return p.next.GetForecastCI(ctx, zone, hours)
}

type tokenBucket struct {
	mu    sync.Mutex
	rate  float64
	burst float64
	last  time.Time
	token float64
}

func newTokenBucket(cfg RateLimitConfig) *tokenBucket {
	rate := cfg.RequestsPerSecond
	if rate <= 0 {
		rate = 1
	}
	burst := cfg.Burst
	if burst < 1 {
		burst = 1
	}

	return &tokenBucket{
		rate:  rate,
		burst: float64(burst),
		last:  time.Now(),
		token: float64(burst),
	}
}

func (b *tokenBucket) Wait(ctx context.Context) error {
	for {
		wait := b.consume()
		if wait <= 0 {
			return nil
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (b *tokenBucket) consume() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now

	b.token = math.Min(b.burst, b.token+elapsed*b.rate)
	if b.token >= 1 {
		b.token -= 1
		return 0
	}

	deficit := 1 - b.token
	waitSeconds := deficit / b.rate
	wait := time.Duration(waitSeconds * float64(time.Second))
	if wait < time.Millisecond {
		wait = time.Millisecond
	}
	return wait
}

func WithMetrics(recorder MetricsRecorder) Middleware {
	return func(next Provider) Provider {
		return &metricsProvider{
			next:     next,
			recorder: recorder,
		}
	}
}

type metricsProvider struct {
	next     Provider
	recorder MetricsRecorder
}

func (p *metricsProvider) GetCurrentCI(ctx context.Context, zone string) (value float64, err error) {
	start := time.Now()
	defer func() {
		p.recorder.ObserveCall("GetCurrentCI", zone, time.Since(start), err)
	}()
	value, err = p.next.GetCurrentCI(ctx, zone)
	return value, err
}

func (p *metricsProvider) GetForecastCI(ctx context.Context, zone string, hours int) (points []ForecastPoint, err error) {
	start := time.Now()
	defer func() {
		p.recorder.ObserveCall("GetForecastCI", zone, time.Since(start), err)
	}()
	points, err = p.next.GetForecastCI(ctx, zone, hours)
	return points, err
}

type lockedRand struct {
	mu sync.Mutex
	r  *rand.Rand
}

func newLockedRand() *lockedRand {
	return &lockedRand{
		r: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *lockedRand) Float64() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.r.Float64()
}
