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

type OptimizeZoneOutput struct {
	Zone         string  `json:"zone"`
	EmissionKg   float64 `json:"emission_kg"`
	BestStartUTC string  `json:"best_start_utc"`
	BestEndUTC   string  `json:"best_end_utc"`
}

type OptimizeResult struct {
	DurationSeconds     int                  `json:"duration_seconds"`
	Zones               []OptimizeZoneOutput `json:"zones"`
	BestZone            string               `json:"best_zone"`
	BestWindowStartUTC  string               `json:"best_window_start_utc"`
	BestWindowEndUTC    string               `json:"best_window_end_utc"`
	EmissionKg          float64              `json:"emission_kg"`
	ReductionVsWorstPct float64              `json:"reduction_vs_worst_pct"`
}

func optimize(args []string) error {
	fs := flag.NewFlagSet("optimize", flag.ContinueOnError)
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
	if *duration > *lookahead*3600 {
		return cgerrors.Newf(cgerrors.InputError, "duration %ds exceeds forecast coverage %ds", *duration, *lookahead*3600)
	}
	timeout, err := time.ParseDuration(*timeoutStr)
	if err != nil || timeout <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "invalid timeout duration")
	}
	cacheDir, cacheTTL, err := parseCacheConfig(*cacheDirRaw, *cacheTTLRaw)
	if err != nil {
		return cgerrors.New(err, cgerrors.InputError)
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
	out, err := service.Optimize(context.Background(), appsvc.OptimizeInput{
		Zones:     zoneList,
		Duration:  *duration,
		Lookahead: *lookahead,
		Timeout:   timeout,
	})
	if err != nil {
		return mapAppError(err)
	}

	if *outputMode != "json" {
		for _, line := range appsvc.FormatZoneFailures(out.Failures) {
			fmt.Fprintln(os.Stderr, line)
		}
	}

	if *outputMode == "json" {
		zoneOutputs := make([]OptimizeZoneOutput, 0, len(out.Results))
		for _, result := range out.Results {
			zoneOutputs = append(zoneOutputs, OptimizeZoneOutput{
				Zone:         result.Zone,
				EmissionKg:   result.Emission,
				BestStartUTC: result.BestStart.UTC().Format(time.RFC3339),
				BestEndUTC:   result.BestEnd.UTC().Format(time.RFC3339),
			})
		}

		payload := OptimizeResult{
			DurationSeconds:     *duration,
			Zones:               zoneOutputs,
			BestZone:            out.Best.Zone,
			BestWindowStartUTC:  out.Best.BestStart.UTC().Format(time.RFC3339),
			BestWindowEndUTC:    out.Best.BestEnd.UTC().Format(time.RFC3339),
			EmissionKg:          out.Best.Emission,
			ReductionVsWorstPct: out.Reduction,
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return cgerrors.Newf(cgerrors.ProviderError, "failed to serialize optimize result")
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Zone comparison (duration=%ds):\n\n", *duration)
	for _, result := range out.Results {
		fmt.Printf("%s -> %.3f kg\n", result.Zone, result.Emission)
	}
	fmt.Printf("\nBest zone: %s\n", out.Best.Zone)
	fmt.Printf("Best window (UTC): %s - %s\n", out.Best.BestStart.UTC().Format("15:04"), out.Best.BestEnd.UTC().Format("15:04"))
	fmt.Printf("Reduction vs worst: %.2f %%\n", out.Reduction)
	return nil
}
