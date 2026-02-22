package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	appsvc "github.com/chenzhuyu2004/carbon-guard/internal/app"
	"github.com/chenzhuyu2004/carbon-guard/internal/ci"
	cgerrors "github.com/chenzhuyu2004/carbon-guard/internal/errors"
)

type OptimizeGlobalResult struct {
	DurationSeconds     int     `json:"duration_seconds"`
	BestZone            string  `json:"best_zone"`
	BestWindowStartUTC  string  `json:"best_window_start_utc"`
	BestWindowEndUTC    string  `json:"best_window_end_utc"`
	EmissionKg          float64 `json:"emission_kg"`
	ReductionVsWorstPct float64 `json:"reduction_vs_worst_pct"`
}

func optimizeGlobal(args []string) error {
	fs := flag.NewFlagSet("optimize-global", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	zones := fs.String("zones", "", "comma-separated Electricity Maps zones")
	duration := fs.Int("duration", 0, "duration in seconds")
	lookahead := fs.Int("lookahead", 6, "forecast lookahead in hours")
	timeoutStr := fs.String("timeout", "30s", "operation timeout")
	outputMode := fs.String("output", "text", "output format: text|json")
	cacheDirRaw := fs.String("cache-dir", "~/.carbon-guard", "forecast cache directory")
	cacheTTLRaw := fs.String("cache-ttl", "10m", "forecast cache TTL")

	if err := fs.Parse(args); err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}
	if *outputMode != "text" && *outputMode != "json" {
		return cgerrors.Newf(cgerrors.InputError, "output must be text or json")
	}

	if *zones == "" {
		return cgerrors.Newf(cgerrors.InputError, "zones is required")
	}
	if *duration <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "duration must be > 0")
	}
	if *lookahead <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "lookahead must be > 0")
	}
	timeout, err := time.ParseDuration(*timeoutStr)
	if err != nil || timeout <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "invalid timeout duration")
	}
	cacheDir, cacheTTL, err := parseCacheConfig(*cacheDirRaw, *cacheTTLRaw)
	if err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}
	if *duration > *lookahead*3600 {
		return cgerrors.Newf(cgerrors.InputError, "duration %ds exceeds forecast coverage %ds", *duration, *lookahead*3600)
	}

	zoneList := splitZones(*zones)
	if len(zoneList) == 0 {
		return cgerrors.Newf(cgerrors.InputError, "zones is required")
	}

	apiKey := os.Getenv("ELECTRICITY_MAPS_API_KEY")
	if apiKey == "" {
		return cgerrors.Newf(cgerrors.InputError, "missing ELECTRICITY_MAPS_API_KEY")
	}

	base := &ci.ElectricityMapsProvider{APIKey: apiKey}
	provider := &ci.CachedProvider{
		Inner:    base,
		CacheDir: cacheDir,
		TTL:      cacheTTL,
	}
	service := appsvc.New(newProviderAdapter(provider))
	out, err := service.OptimizeGlobal(context.Background(), appsvc.OptimizeGlobalInput{
		Zones:     zoneList,
		Duration:  *duration,
		Lookahead: *lookahead,
		Runner:    defaultSchedulingRunner,
		Load:      defaultSchedulingLoad,
		PUE:       defaultSchedulingPUE,
		Timeout:   timeout,
	})
	if err != nil {
		return mapAppError(err)
	}

	if *outputMode == "json" {
		payload := OptimizeGlobalResult{
			DurationSeconds:     *duration,
			BestZone:            out.BestZone,
			BestWindowStartUTC:  out.BestStart.UTC().Format(time.RFC3339),
			BestWindowEndUTC:    out.BestEnd.UTC().Format(time.RFC3339),
			EmissionKg:          out.Emission,
			ReductionVsWorstPct: out.Reduction,
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return cgerrors.Newf(cgerrors.ProviderError, "failed to serialize optimize-global result")
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("Global optimal execution plan")
	fmt.Println()
	fmt.Printf("Start (UTC): %s\n", out.BestStart.UTC().Format("15:04"))
	fmt.Printf("Zone: %s\n", out.BestZone)
	fmt.Printf("Emission: %.3f kg\n", out.Emission)
	fmt.Printf("Improvement vs worst plan: %.2f %%\n", out.Reduction)
	return nil
}
