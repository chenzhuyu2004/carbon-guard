package ci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CachedProvider struct {
	Inner    Provider
	CacheDir string
	TTL      time.Duration

	mu       sync.Mutex
	inflight map[string]*forecastCall
}

type ForecastCacheFile struct {
	FetchedAt string          `json:"fetched_at"`
	Forecast  []ForecastPoint `json:"forecast"`
}

type forecastCall struct {
	done   chan struct{}
	points []ForecastPoint
	err    error
}

const (
	cacheLockPollInterval = 100 * time.Millisecond
	cacheLockStaleAfter   = 2 * time.Minute
)

func (c *CachedProvider) GetCurrentCI(ctx context.Context, zone string) (float64, error) {
	if c.Inner == nil {
		return 0, fmt.Errorf("cached provider inner provider is nil")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return c.Inner.GetCurrentCI(ctx, zone)
}

func (c *CachedProvider) GetForecastCI(ctx context.Context, zone string, hours int) ([]ForecastPoint, error) {
	if c.Inner == nil {
		return nil, fmt.Errorf("cached provider inner provider is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cachePath := c.forecastCachePath(zone, hours)
	if c.TTL > 0 {
		if points, ok := c.readForecastCache(ctx, cachePath); ok {
			return points, nil
		}
	}

	callKey := fmt.Sprintf("%s:%d", sanitizeCacheToken(zone), hours)
	call, leader := c.acquireInflight(callKey)
	if !leader {
		return c.awaitInflight(ctx, call)
	}

	var points []ForecastPoint
	var err error
	defer c.finishInflight(callKey, call, points, err)

	var unlock func()
	if c.TTL > 0 {
		unlock, err = c.acquireFileLock(ctx, cachePath+".lock")
		if err != nil {
			return nil, err
		}
		if unlock != nil {
			defer unlock()
			if cached, ok := c.readForecastCache(ctx, cachePath); ok {
				points = cached
				return points, nil
			}
		}
	}

	points, err = c.Inner.GetForecastCI(ctx, zone, hours)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if c.TTL > 0 {
		c.writeForecastCache(ctx, cachePath, points)
	}
	return points, nil
}

func (c *CachedProvider) forecastCachePath(zone string, hours int) string {
	file := fmt.Sprintf("forecast_%s_%d.json", sanitizeCacheToken(zone), hours)
	return filepath.Join(c.CacheDir, file)
}

func (c *CachedProvider) readForecastCache(ctx context.Context, path string) ([]ForecastPoint, bool) {
	if err := ctx.Err(); err != nil {
		return nil, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	if err := ctx.Err(); err != nil {
		return nil, false
	}

	var cached ForecastCacheFile
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}
	fetchedAt, err := time.Parse(time.RFC3339, cached.FetchedAt)
	if err != nil {
		return nil, false
	}

	if time.Since(fetchedAt.UTC()) >= c.TTL {
		return nil, false
	}
	return cached.Forecast, true
}

func (c *CachedProvider) writeForecastCache(ctx context.Context, path string, points []ForecastPoint) {
	if err := ctx.Err(); err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}

	payload := ForecastCacheFile{
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
		Forecast:  points,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return
	}
	if err := ctx.Err(); err != nil {
		return
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		return
	}
	if err := tmp.Sync(); err != nil {
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}
	if err := ctx.Err(); err != nil {
		return
	}

	_ = os.Rename(tmpPath, path)
}

func (c *CachedProvider) acquireInflight(key string) (*forecastCall, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.inflight == nil {
		c.inflight = make(map[string]*forecastCall)
	}
	if existing, ok := c.inflight[key]; ok {
		return existing, false
	}

	call := &forecastCall{done: make(chan struct{})}
	c.inflight[key] = call
	return call, true
}

func (c *CachedProvider) awaitInflight(ctx context.Context, call *forecastCall) ([]ForecastPoint, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.done:
		if call.err != nil {
			return nil, call.err
		}
		return call.points, nil
	}
}

func (c *CachedProvider) finishInflight(key string, call *forecastCall, points []ForecastPoint, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	call.points = points
	call.err = err
	close(call.done)
	delete(c.inflight, key)
}

func (c *CachedProvider) acquireFileLock(ctx context.Context, lockPath string) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, nil
	}

	for {
		lockFile, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_, _ = lockFile.WriteString(time.Now().UTC().Format(time.RFC3339Nano))
			_ = lockFile.Close()
			return func() {
				_ = os.Remove(lockPath)
			}, nil
		}

		if !errors.Is(err, os.ErrExist) {
			return nil, nil
		}

		info, statErr := os.Stat(lockPath)
		if statErr == nil && time.Since(info.ModTime()) > cacheLockStaleAfter {
			_ = os.Remove(lockPath)
			continue
		}

		timer := time.NewTimer(cacheLockPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func sanitizeCacheToken(value string) string {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return "UNKNOWN"
	}

	var b strings.Builder
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}
