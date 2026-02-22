package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	appsvc "github.com/chenzhuyu2004/carbon-guard/internal/app"
	cgerrors "github.com/chenzhuyu2004/carbon-guard/internal/errors"
)

func runAware(args []string) error {
	defaults, err := resolveSharedDefaults(args)
	if err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}

	fs := flag.NewFlagSet("run-aware", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	addConfigFlag(fs, defaults.ConfigPath)
	zone := fs.String("zone", "", "electricity maps zone")
	zoneMode := fs.String("zone-mode", defaults.ZoneMode, "zone resolution mode: strict|fallback|auto")
	duration := fs.Int("duration", 0, "duration in seconds")
	threshold := fs.Float64("threshold", 0.35, "legacy CI threshold in kgCO2/kWh (used when threshold-enter/exit are unset)")
	thresholdEnter := fs.Float64("threshold-enter", -1, "run when CI is <= this threshold in kgCO2/kWh")
	thresholdExit := fs.Float64("threshold-exit", -1, "continue waiting when CI is >= this threshold in kgCO2/kWh")
	lookahead := fs.Int("lookahead", 6, "forecast lookahead in hours")
	maxWait := fs.Float64("max-wait", 6, "maximum wait time in hours")
	maxDelayForGainRaw := fs.String("max-delay-for-gain", "0s", "no-regret guard: maximum acceptable delay before waiting is skipped")
	minReductionForWait := fs.Float64("min-reduction-for-wait", 0, "no-regret guard: minimum expected reduction percentage required to justify waiting")
	cacheDirRaw, cacheTTLRaw := addCacheFlags(fs, defaults.CacheDir, defaults.CacheTTL)

	if err := fs.Parse(args); err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}

	if *duration <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "duration must be > 0")
	}
	if *lookahead <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "lookahead must be > 0")
	}
	if *maxWait <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "max-wait must be > 0")
	}
	if *minReductionForWait < 0 {
		return cgerrors.Newf(cgerrors.InputError, "min-reduction-for-wait must be >= 0")
	}
	maxDelayForGain, err := time.ParseDuration(*maxDelayForGainRaw)
	if err != nil || maxDelayForGain < 0 {
		return cgerrors.Newf(cgerrors.InputError, "max-delay-for-gain must be a non-negative duration")
	}

	effectiveEnter := *thresholdEnter
	effectiveExit := *thresholdExit
	if effectiveEnter <= 0 {
		effectiveEnter = *threshold
	}
	if effectiveExit <= 0 {
		effectiveExit = *threshold
	}
	if effectiveEnter <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "threshold-enter must be > 0")
	}
	if effectiveExit <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "threshold-exit must be > 0")
	}
	if effectiveEnter > effectiveExit {
		return cgerrors.Newf(cgerrors.InputError, "threshold-enter must be <= threshold-exit")
	}
	resolvedZone, err := resolveZone(*zone, *zoneMode, defaults.Zone, autoHints{
		ZoneHint:     defaults.ZoneHint,
		CountryHint:  defaults.CountryHint,
		TimezoneHint: defaults.TimezoneHint,
	})
	if err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}

	cacheDir, cacheTTL, err := parseCacheConfig(*cacheDirRaw, *cacheTTLRaw)
	if err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}

	apiKey := os.Getenv("ELECTRICITY_MAPS_API_KEY")
	if apiKey == "" {
		return cgerrors.Newf(cgerrors.InputError, "missing ELECTRICITY_MAPS_API_KEY")
	}

	service := appsvc.New(newProviderAdapter(buildLiveProvider(apiKey, cacheDir, cacheTTL)))
	out, err := service.RunAware(context.Background(), appsvc.RunAwareInput{
		Zone:                    resolvedZone.Zone,
		Duration:                *duration,
		Threshold:               *threshold,
		ThresholdEnter:          effectiveEnter,
		ThresholdExit:           effectiveExit,
		Lookahead:               *lookahead,
		Model:                   defaultModelContext(),
		MaxWait:                 time.Duration(*maxWait * float64(time.Hour)),
		NoRegretMaxDelay:        maxDelayForGain,
		NoRegretMinReductionPct: *minReductionForWait,
		PollEvery:               15 * time.Minute,
		StatusFunc: func(msg string) {
			fmt.Println(msg)
		},
	})
	if err != nil {
		return mapAppError(err)
	}
	fmt.Printf(
		"Resolved Zone: %s (source: %s, confidence: %s, reason: %s, fallback_used: %t)\n",
		resolvedZone.Zone,
		resolvedZone.Source,
		resolvedZone.Confidence,
		resolvedZone.Reason,
		resolvedZone.FallbackUsed,
	)
	fmt.Println(out.Message)
	return nil
}
