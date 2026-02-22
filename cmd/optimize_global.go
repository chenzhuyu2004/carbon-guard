package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	appsvc "github.com/chenzhuyu2004/carbon-guard/internal/app"
	cgerrors "github.com/chenzhuyu2004/carbon-guard/internal/errors"
	"github.com/chenzhuyu2004/carbon-guard/pkg"
)

type OptimizeGlobalResult struct {
	SchemaVersion             string  `json:"schema_version"`
	DurationSeconds           int     `json:"duration_seconds"`
	ZonesSource               string  `json:"zones_source"`
	ZonesConfidence           string  `json:"zones_confidence"`
	ZonesReason               string  `json:"zones_reason"`
	ZonesFallbackUsed         bool    `json:"zones_fallback_used"`
	BestZone                  string  `json:"best_zone"`
	BestWindowStartUTC        string  `json:"best_window_start_utc"`
	BestWindowEndUTC          string  `json:"best_window_end_utc"`
	EmissionKg                float64 `json:"emission_kg"`
	ReductionVsWorstPct       float64 `json:"reduction_vs_worst_pct"`
	ResampleFillMode          string  `json:"resample_fill_mode"`
	ResampleMaxFillAgeSeconds int64   `json:"resample_max_fill_age_seconds"`
}

func optimizeGlobal(args []string) error {
	defaults, err := resolveSharedDefaults(args)
	if err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}

	fs := flag.NewFlagSet("optimize-global", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	addConfigFlag(fs, defaults.ConfigPath)
	zones := fs.String("zones", "", "comma-separated Electricity Maps zones")
	zoneMode := fs.String("zone-mode", defaults.ZoneMode, "zone resolution mode: strict|fallback|auto")
	duration := fs.Int("duration", 0, "duration in seconds")
	lookahead := fs.Int("lookahead", 6, "forecast lookahead in hours")
	waitCost := fs.Float64("wait-cost", 0, "waiting penalty in kgCO2 per hour")
	resampleFill := fs.String("resample-fill", "forward", "resample fill mode: forward|strict")
	resampleMaxFillAgeRaw := fs.String("resample-max-fill-age", "", "max forward-fill age (e.g. 30m). empty uses default")
	timeoutStr := addTimeoutFlag(fs, defaults.Timeout)
	outputMode := addOutputFlag(fs, defaults.Output)
	cacheDirRaw, cacheTTLRaw := addCacheFlags(fs, defaults.CacheDir, defaults.CacheTTL)

	if err := fs.Parse(args); err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}
	if err := validateOutputMode(*outputMode); err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}

	if *duration <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "duration must be > 0")
	}
	if *lookahead <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "lookahead must be > 0")
	}
	timeout, err := parseTimeout(*timeoutStr)
	if err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}
	cacheDir, cacheTTL, err := parseCacheConfig(*cacheDirRaw, *cacheTTLRaw)
	if err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}
	if *duration > *lookahead*3600 {
		return cgerrors.Newf(cgerrors.InputError, "duration %ds exceeds forecast coverage %ds", *duration, *lookahead*3600)
	}
	if *waitCost < 0 {
		return cgerrors.Newf(cgerrors.InputError, "wait-cost must be >= 0")
	}
	resampleMaxFillAge := time.Duration(0)
	if *resampleMaxFillAgeRaw != "" {
		parsed, err := time.ParseDuration(*resampleMaxFillAgeRaw)
		if err != nil || parsed < 0 {
			return cgerrors.Newf(cgerrors.InputError, "resample-max-fill-age must be a non-negative duration")
		}
		resampleMaxFillAge = parsed
	}

	resolvedZones, err := resolveZones(*zones, *zoneMode, defaults.Zones, autoHints{
		ZoneHint:     defaults.ZoneHint,
		CountryHint:  defaults.CountryHint,
		TimezoneHint: defaults.TimezoneHint,
	})
	if err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}

	apiKey := os.Getenv("ELECTRICITY_MAPS_API_KEY")
	if apiKey == "" {
		return cgerrors.Newf(cgerrors.InputError, "missing ELECTRICITY_MAPS_API_KEY")
	}

	service := appsvc.New(newProviderAdapter(buildLiveProvider(apiKey, cacheDir, cacheTTL)))
	out, err := service.OptimizeGlobal(context.Background(), appsvc.OptimizeGlobalInput{
		Zones:              resolvedZones.Zones,
		Duration:           *duration,
		Lookahead:          *lookahead,
		WaitCost:           *waitCost,
		ResampleFillMode:   *resampleFill,
		ResampleMaxFillAge: resampleMaxFillAge,
		Model:              defaultModelContext(),
		Timeout:            timeout,
	})
	if err != nil {
		return mapAppError(err)
	}

	if *outputMode == "json" {
		payload := OptimizeGlobalResult{
			SchemaVersion:             pkg.JSONSchemaVersion,
			DurationSeconds:           *duration,
			ZonesSource:               resolvedZones.Source,
			ZonesConfidence:           resolvedZones.Confidence,
			ZonesReason:               resolvedZones.Reason,
			ZonesFallbackUsed:         resolvedZones.FallbackUsed,
			BestZone:                  out.BestZone,
			BestWindowStartUTC:        out.BestStart.UTC().Format(time.RFC3339),
			BestWindowEndUTC:          out.BestEnd.UTC().Format(time.RFC3339),
			EmissionKg:                out.Emission,
			ReductionVsWorstPct:       out.Reduction,
			ResampleFillMode:          out.ResampleFillMode,
			ResampleMaxFillAgeSeconds: out.ResampleMaxFillAgeSeconds,
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
	fmt.Printf(
		"Resolved Zones Source: %s (confidence: %s, reason: %s, fallback_used: %t)\n",
		resolvedZones.Source,
		resolvedZones.Confidence,
		resolvedZones.Reason,
		resolvedZones.FallbackUsed,
	)
	fmt.Printf("Start (UTC): %s\n", out.BestStart.UTC().Format("15:04"))
	fmt.Printf("Zone: %s\n", out.BestZone)
	fmt.Printf("Emission: %.3f kg\n", out.Emission)
	fmt.Printf("Improvement vs worst plan: %.2f %%\n", out.Reduction)
	fmt.Printf("Resample mode: %s (max fill age: %ds)\n", out.ResampleFillMode, out.ResampleMaxFillAgeSeconds)
	return nil
}
