package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	appsvc "github.com/chenzhuyu2004/carbon-guard/internal/app"
	"github.com/chenzhuyu2004/carbon-guard/internal/ci"
	cgerrors "github.com/chenzhuyu2004/carbon-guard/internal/errors"
	"github.com/chenzhuyu2004/carbon-guard/internal/report"
)

func run(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	duration := fs.Int("duration", 0, "duration in seconds")
	runner := fs.String("runner", "ubuntu", "runner type (ubuntu/windows/macos)")
	region := fs.String("region", "global", "region carbon intensity")
	load := fs.Float64("load", 0.6, "CPU load factor (0-1)")
	pue := fs.Float64("pue", 1.2, "data center PUE (>=1.0)")
	segmentsStr := fs.String("segments", "", "dynamic CI segments (duration:ci,...)")
	liveZone := fs.String("live-ci", "", "fetch live carbon intensity for zone")
	budgetKg := fs.Float64("budget-kg", 0, "carbon budget in kgCO2 (optional)")
	baselineKg := fs.Float64("baseline-kg", 0, "baseline emissions in kgCO2 for comparison (optional)")
	failOnBudget := fs.Bool("fail-on-budget", false, "exit non-zero when emissions exceed budget")
	asJSON := fs.Bool("json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		return cgerrors.New(err, cgerrors.InputError)
	}
	if *budgetKg < 0 {
		return cgerrors.Newf(cgerrors.InputError, "budget-kg must be >= 0")
	}
	if *baselineKg < 0 {
		return cgerrors.Newf(cgerrors.InputError, "baseline-kg must be >= 0")
	}
	if *failOnBudget && *budgetKg <= 0 {
		return cgerrors.Newf(cgerrors.InputError, "fail-on-budget requires budget-kg > 0")
	}

	var provider ci.Provider
	if *liveZone != "" {
		apiKey := os.Getenv("ELECTRICITY_MAPS_API_KEY")
		if apiKey == "" {
			return cgerrors.Newf(cgerrors.InputError, "missing ELECTRICITY_MAPS_API_KEY")
		}
		provider = &ci.ElectricityMapsProvider{APIKey: apiKey}
	}
	service := appsvc.New(newProviderAdapter(provider))
	result, err := service.Run(context.Background(), appsvc.RunInput{
		Duration:    *duration,
		Runner:      *runner,
		Region:      *region,
		Load:        *load,
		PUE:         *pue,
		SegmentsRaw: *segmentsStr,
		LiveZone:    *liveZone,
	})
	if err != nil {
		return mapAppError(err)
	}

	output := report.BuildFromEmissions(result.DurationSeconds, *asJSON, result.EmissionsKg, report.BuildOptions{
		BudgetKg:   *budgetKg,
		BaselineKg: *baselineKg,
	})
	fmt.Print(output)

	if *failOnBudget && *budgetKg > 0 && result.EmissionsKg > *budgetKg {
		return cgerrors.Newf(cgerrors.BudgetExceeded, "carbon budget exceeded: emissions %.4f kgCO2 > budget %.4f kgCO2", result.EmissionsKg, *budgetKg)
	}
	return nil
}
