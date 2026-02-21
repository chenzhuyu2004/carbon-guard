package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	appsvc "github.com/chenzhuyu2004/carbon-guard/internal/app"
	"github.com/chenzhuyu2004/carbon-guard/internal/ci"
	cgerrors "github.com/chenzhuyu2004/carbon-guard/internal/errors"
)

func suggest(args []string) error {
	fs := flag.NewFlagSet("suggest", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	zone := fs.String("zone", "", "electricity maps zone")
	duration := fs.Int("duration", 0, "duration in seconds")
	threshold := fs.Float64("threshold", 0.35, "current CI threshold in kgCO2/kWh")
	lookahead := fs.Int("lookahead", 6, "forecast lookahead in hours")

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

	apiKey := os.Getenv("ELECTRICITY_MAPS_API_KEY")
	if apiKey == "" {
		return cgerrors.Newf(cgerrors.InputError, "missing ELECTRICITY_MAPS_API_KEY")
	}

	provider := &ci.ElectricityMapsProvider{APIKey: apiKey}
	service := appsvc.New(newProviderAdapter(provider))
	out, err := service.Suggest(context.Background(), appsvc.SuggestInput{
		Zone:      *zone,
		Duration:  *duration,
		Threshold: *threshold,
		Lookahead: *lookahead,
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
