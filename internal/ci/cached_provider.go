package ci

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CachedProvider struct {
	Inner    Provider
	CacheDir string
	TTL      time.Duration
}

type ForecastCacheFile struct {
	FetchedAt string          `json:"fetched_at"`
	Forecast  []ForecastPoint `json:"forecast"`
}

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

	points, err := c.Inner.GetForecastCI(ctx, zone, hours)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	c.writeForecastCache(ctx, cachePath, points)
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
