package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	appsvc "github.com/chenzhuyu2004/carbon-guard/internal/app"
	cgerrors "github.com/chenzhuyu2004/carbon-guard/internal/errors"
)

func suggest(args []string) error {
	defaults, err := resolveSharedDefaults(args)
	if err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}

	fs := flag.NewFlagSet("suggest", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	addConfigFlag(fs, defaults.ConfigPath)
	zone := fs.String("zone", "", "electricity maps zone")
	duration := fs.Int("duration", 0, "duration in seconds")
	threshold := fs.Float64("threshold", 0.35, "current CI threshold in kgCO2/kWh")
	lookahead := fs.Int("lookahead", 6, "forecast lookahead in hours")
	waitCost := fs.Float64("wait-cost", 0, "waiting penalty in kgCO2 per hour")
	cacheDirRaw, cacheTTLRaw := addCacheFlags(fs, defaults.CacheDir, defaults.CacheTTL)

	if err := fs.Parse(args); err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}

	if *zone == "" {
		return cgerrors.Newf(cgerrors.InputError, "zone is required")
	}
	if *duration <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "duration must be > 0")
	}
	if *threshold <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "threshold must be > 0")
	}
	if *lookahead <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "lookahead must be > 0")
	}
	if *waitCost < 0 {
		return cgerrors.Newf(cgerrors.InputError, "wait-cost must be >= 0")
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
	out, err := service.Suggest(context.Background(), appsvc.SuggestInput{
		Zone:      *zone,
		Duration:  *duration,
		Threshold: *threshold,
		Lookahead: *lookahead,
		WaitCost:  *waitCost,
		Model:     defaultModelContext(),
	})
	if err != nil {
		return mapAppError(err)
	}

	fmt.Printf(
		"Current CI: %.4f kg/kWh\nBest execution window (UTC): %s - %s\nExpected emission: %.4f kg\nEmission reduction vs now: %.2f %%\n",
		out.CurrentCI,
		out.BestWindowStartUTC.UTC().Format("15:04"),
		out.BestWindowEndUTC.UTC().Format("15:04"),
		out.ExpectedEmissionKg,
		out.EmissionReductionVsNow,
	)
	return nil
}
