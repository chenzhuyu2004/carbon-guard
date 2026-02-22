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
	fs := flag.NewFlagSet("run-aware", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	zone := fs.String("zone", "", "electricity maps zone")
	duration := fs.Int("duration", 0, "duration in seconds")
	threshold := fs.Float64("threshold", 0.35, "current CI threshold in kgCO2/kWh")
	lookahead := fs.Int("lookahead", 6, "forecast lookahead in hours")
	maxWait := fs.Float64("max-wait", 6, "maximum wait time in hours")
	cacheDirRaw, cacheTTLRaw := addCacheFlags(fs)

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
	if *maxWait <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "max-wait must be > 0")
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
		Zone:      *zone,
		Duration:  *duration,
		Threshold: *threshold,
		Lookahead: *lookahead,
		Model:     defaultModelContext(),
		MaxWait:   time.Duration(*maxWait * float64(time.Hour)),
		PollEvery: 15 * time.Minute,
		StatusFunc: func(msg string) {
			fmt.Println(msg)
		},
	})
	if err != nil {
		return mapAppError(err)
	}
	fmt.Println(out.Message)
	return nil
}
