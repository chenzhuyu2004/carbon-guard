package cmd

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/czy/carbon-guard/internal/calculator"
	"github.com/czy/carbon-guard/internal/ci"
	"github.com/czy/carbon-guard/internal/report"
)

func Execute() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		run(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func run(args []string) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	duration := fs.Int("duration", 0, "duration in seconds")
	runner := fs.String("runner", "ubuntu", "runner type (ubuntu/windows/macos)")
	region := fs.String("region", "global", "region carbon intensity")
	load := fs.Float64("load", 0.6, "CPU load factor (0-1)")
	pue := fs.Float64("pue", 1.2, "data center PUE (>=1.0)")
	segmentsStr := fs.String("segments", "", "dynamic CI segments (duration:ci,...)")
	liveZone := fs.String("live-ci", "", "fetch live carbon intensity for zone")
	asJSON := fs.Bool("json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *duration <= 0 {
		fmt.Fprintln(os.Stderr, "duration must be > 0")
		os.Exit(1)
	}

	if *load < 0 || *load > 1 {
		fmt.Fprintln(os.Stderr, "load must be between 0 and 1")
		os.Exit(1)
	}

	if *pue < 1.0 {
		fmt.Fprintln(os.Stderr, "pue must be >= 1.0")
		os.Exit(1)
	}

	var provider ci.Provider
	if *liveZone != "" {
		apiKey := os.Getenv("ELECTRICITY_MAPS_API_KEY")
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "missing ELECTRICITY_MAPS_API_KEY")
			os.Exit(1)
		}
		provider = &ci.ElectricityMapsProvider{APIKey: apiKey}
	}

	emissions, err := calculateEmissions(*duration, *runner, *region, *load, *pue, *segmentsStr, *liveZone, provider)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Print(report.BuildFromEmissions(*duration, *asJSON, emissions))
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: carbon-guard run --duration <seconds> [--json]")
}

func calculateEmissions(
	duration int,
	runner string,
	region string,
	load float64,
	pue float64,
	segmentsStr string,
	liveZone string,
	provider ci.Provider,
) (float64, error) {
	if segmentsStr != "" {
		segments, err := parseSegments(segmentsStr)
		if err != nil {
			return 0, err
		}
		return calculator.EstimateEmissionsWithSegments(segments, runner, load, pue), nil
	}

	if liveZone != "" {
		if provider == nil {
			return 0, fmt.Errorf("live ci provider is not configured")
		}
		ciValue, err := provider.GetCurrentCI(liveZone)
		if err != nil {
			return 0, err
		}
		segments := []calculator.Segment{{Duration: duration, CI: ciValue}}
		return calculator.EstimateEmissionsWithSegments(segments, runner, load, pue), nil
	}

	return calculator.EstimateEmissionsAdvanced(duration, runner, region, load, pue), nil
}

func parseSegments(raw string) ([]calculator.Segment, error) {
	items := strings.Split(raw, ",")
	segments := make([]calculator.Segment, 0, len(items))

	for _, item := range items {
		item = strings.TrimSpace(item)
		parts := strings.Split(item, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid segment format: %s", item)
		}

		duration, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || duration <= 0 {
			return nil, fmt.Errorf("invalid segment duration: %s", parts[0])
		}

		ci, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil || ci <= 0 {
			return nil, fmt.Errorf("invalid segment ci: %s", parts[1])
		}

		segments = append(segments, calculator.Segment{
			Duration: duration,
			CI:       ci,
		})
	}

	return segments, nil
}
