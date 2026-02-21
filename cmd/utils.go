package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	appsvc "github.com/chenzhuyu2004/carbon-guard/internal/app"
	cgerrors "github.com/chenzhuyu2004/carbon-guard/internal/errors"
)

func mapAppError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, appsvc.ErrInput):
		return cgerrors.New(err, cgerrors.InputError)
	case errors.Is(err, appsvc.ErrProvider):
		return cgerrors.New(err, cgerrors.ProviderError)
	case errors.Is(err, appsvc.ErrNoValidWindow):
		return cgerrors.New(err, cgerrors.NoValidWindow)
	case errors.Is(err, appsvc.ErrMaxWaitExceeded):
		return cgerrors.New(err, cgerrors.MaxWaitExceeded)
	case errors.Is(err, appsvc.ErrMissedOptimalWindow):
		return cgerrors.New(err, cgerrors.MissedWindow)
	case errors.Is(err, appsvc.ErrTimeout), errors.Is(err, context.DeadlineExceeded):
		return cgerrors.New(err, cgerrors.Timeout)
	default:
		return cgerrors.New(err, cgerrors.ProviderError)
	}
}

func splitZones(raw string) []string {
	items := strings.Split(raw, ",")
	zones := make([]string, 0, len(items))
	for _, item := range items {
		zone := strings.ToUpper(strings.TrimSpace(item))
		if zone == "" {
			continue
		}
		zones = append(zones, zone)
	}
	return zones
}

func parseCacheConfig(cacheDirRaw string, cacheTTLRaw string) (string, time.Duration, error) {
	cacheTTL, err := time.ParseDuration(cacheTTLRaw)
	if err != nil || cacheTTL < 0 {
		return "", 0, fmt.Errorf("invalid cache-ttl duration")
	}

	cacheDir, err := expandHomeDir(cacheDirRaw)
	if err != nil {
		return "", 0, err
	}
	if strings.TrimSpace(cacheDir) == "" {
		return "", 0, fmt.Errorf("cache-dir must not be empty")
	}

	return cacheDir, cacheTTL, nil
}

func expandHomeDir(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
