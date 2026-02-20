package cmd

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

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
	case "suggest":
		suggest(os.Args[2:])
	case "run-aware":
		runAware(os.Args[2:])
	case "optimize":
		optimize(os.Args[2:])
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
	fmt.Fprintln(os.Stderr, "usage: carbon-guard <run|suggest|run-aware|optimize> [flags]")
}

func suggest(args []string) {
	fs := flag.NewFlagSet("suggest", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	zone := fs.String("zone", "", "electricity maps zone")
	duration := fs.Int("duration", 0, "duration in seconds")
	threshold := fs.Float64("threshold", 0.35, "current CI threshold in kgCO2/kWh")
	lookahead := fs.Int("lookahead", 6, "forecast lookahead in hours")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *zone == "" {
		fmt.Fprintln(os.Stderr, "zone is required")
		os.Exit(1)
	}
	if *duration <= 0 {
		fmt.Fprintln(os.Stderr, "duration must be > 0")
		os.Exit(1)
	}
	if *threshold <= 0 {
		fmt.Fprintln(os.Stderr, "threshold must be > 0")
		os.Exit(1)
	}
	if *lookahead <= 0 {
		fmt.Fprintln(os.Stderr, "lookahead must be > 0")
		os.Exit(1)
	}

	apiKey := os.Getenv("ELECTRICITY_MAPS_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "missing ELECTRICITY_MAPS_API_KEY")
		os.Exit(1)
	}

	provider := &ci.ElectricityMapsProvider{APIKey: apiKey}
	out, err := buildSuggestion(*zone, *duration, *threshold, *lookahead, provider)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Print(out)
}

func runAware(args []string) {
	fs := flag.NewFlagSet("run-aware", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	zone := fs.String("zone", "", "electricity maps zone")
	duration := fs.Int("duration", 0, "duration in seconds")
	threshold := fs.Float64("threshold", 0.35, "current CI threshold in kgCO2/kWh")
	lookahead := fs.Int("lookahead", 6, "forecast lookahead in hours")
	maxWait := fs.Float64("max-wait", 6, "maximum wait time in hours")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *zone == "" {
		fmt.Fprintln(os.Stderr, "zone is required")
		os.Exit(1)
	}
	if *duration <= 0 {
		fmt.Fprintln(os.Stderr, "duration must be > 0")
		os.Exit(1)
	}
	if *threshold <= 0 {
		fmt.Fprintln(os.Stderr, "threshold must be > 0")
		os.Exit(1)
	}
	if *lookahead <= 0 {
		fmt.Fprintln(os.Stderr, "lookahead must be > 0")
		os.Exit(1)
	}
	if *maxWait <= 0 {
		fmt.Fprintln(os.Stderr, "max-wait must be > 0")
		os.Exit(1)
	}

	apiKey := os.Getenv("ELECTRICITY_MAPS_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "missing ELECTRICITY_MAPS_API_KEY")
		os.Exit(1)
	}

	provider := &ci.ElectricityMapsProvider{APIKey: apiKey}
	analysis, err := analyzeSuggestion(*zone, *duration, *threshold, *lookahead, provider)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	bestStart := analysis.BestStart
	bestEnd := analysis.BestEnd
	bestEmission := analysis.BestEmission
	currentCI := analysis.CurrentCI
	_ = bestEmission

	startTime := time.Now()
	maxWaitDuration := time.Duration(*maxWait * float64(time.Hour))
	deadline := startTime.Add(maxWaitDuration)

	for {
		now := time.Now()
		if now.After(deadline) {
			fmt.Println("Max wait exceeded")
			os.Exit(10)
		}

		if isWithinWindow(now.UTC(), bestStart, bestEnd) {
			fmt.Println("Entering optimal carbon window")
			return
		}

		if currentCI <= *threshold {
			fmt.Println("CI dropped below threshold, running now")
			return
		}

		fmt.Printf("CI too high (%.2f > %.2f)\n", currentCI, *threshold)
		fmt.Println("Waiting 15m...")

		wait := 15 * time.Minute
		remaining := time.Until(deadline)
		if remaining < wait {
			wait = remaining
		}
		if wait <= 0 {
			fmt.Println("Max wait exceeded")
			os.Exit(10)
		}

		time.Sleep(wait)

		currentCI, err = provider.GetCurrentCI(*zone)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

type ZoneResult struct {
	Zone      string
	Emission  float64
	BestStart time.Time
	BestEnd   time.Time
}

func optimize(args []string) {
	fs := flag.NewFlagSet("optimize", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	zones := fs.String("zones", "", "comma-separated Electricity Maps zones")
	duration := fs.Int("duration", 0, "duration in seconds")
	lookahead := fs.Int("lookahead", 6, "forecast lookahead in hours")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *zones == "" {
		fmt.Fprintln(os.Stderr, "zones is required")
		os.Exit(1)
	}
	if *duration <= 0 {
		fmt.Fprintln(os.Stderr, "duration must be > 0")
		os.Exit(1)
	}
	if *lookahead <= 0 {
		fmt.Fprintln(os.Stderr, "lookahead must be > 0")
		os.Exit(1)
	}

	zoneList := splitZones(*zones)
	if len(zoneList) == 0 {
		fmt.Fprintln(os.Stderr, "zones is required")
		os.Exit(1)
	}

	apiKey := os.Getenv("ELECTRICITY_MAPS_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "missing ELECTRICITY_MAPS_API_KEY")
		os.Exit(1)
	}

	provider := &ci.ElectricityMapsProvider{APIKey: apiKey}
	results := make([]ZoneResult, 0, len(zoneList))

	for _, zone := range zoneList {
		analysis, err := analyzeSuggestion(zone, *duration, -1, *lookahead, provider)
		if err != nil {
			fmt.Fprintf(os.Stderr, "zone %s failed: %v\n", zone, err)
			continue
		}

		results = append(results, ZoneResult{
			Zone:      zone,
			Emission:  analysis.BestEmission,
			BestStart: analysis.BestStart,
			BestEnd:   analysis.BestEnd,
		})
	}

	if len(results) == 0 {
		os.Exit(1)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Emission < results[j].Emission
	})

	best := results[0]
	worst := results[len(results)-1]
	reduction := 0.0
	if worst.Emission > 0 {
		reduction = (worst.Emission - best.Emission) / worst.Emission * 100
	}

	fmt.Printf("Zone comparison (duration=%ds):\n\n", *duration)
	for _, result := range results {
		fmt.Printf("%s -> %.3f kg\n", result.Zone, result.Emission)
	}
	fmt.Printf("\nBest zone: %s\n", best.Zone)
	fmt.Printf("Best window: %s - %s\n", best.BestStart.Local().Format("15:04"), best.BestEnd.Local().Format("15:04"))
	fmt.Printf("Reduction vs worst: %.2f %%\n", reduction)
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

func splitZones(raw string) []string {
	items := strings.Split(raw, ",")
	zones := make([]string, 0, len(items))
	for _, item := range items {
		zone := strings.TrimSpace(item)
		if zone == "" {
			continue
		}
		zones = append(zones, zone)
	}
	return zones
}

type suggestionAnalysis struct {
	CurrentCI       float64
	CurrentEmission float64
	BestStart       time.Time
	BestEnd         time.Time
	BestEmission    float64
	Reduction       float64
}

func analyzeSuggestion(zone string, duration int, threshold float64, lookahead int, provider ci.Provider) (suggestionAnalysis, error) {
	currentCI, err := provider.GetCurrentCI(zone)
	if err != nil {
		return suggestionAnalysis{}, err
	}

	forecast, err := provider.GetForecastCI(zone, lookahead)
	if err != nil {
		return suggestionAnalysis{}, err
	}
	if len(forecast) == 0 {
		return suggestionAnalysis{}, fmt.Errorf("no forecast points found for zone %s", zone)
	}

	currentEmission, ok := estimateWindowEmissions(forecast, duration)
	if !ok {
		return suggestionAnalysis{}, fmt.Errorf("insufficient forecast coverage for duration %d", duration)
	}

	bestEmission := currentEmission
	bestStart := forecast[0].Timestamp

	for i := 1; i < len(forecast); i++ {
		emission, ok := estimateWindowEmissions(forecast[i:], duration)
		if !ok {
			break
		}

		if emission < bestEmission {
			bestEmission = emission
			bestStart = forecast[i].Timestamp
		}
	}

	// If current CI is already below threshold and close to best, prefer immediate execution.
	if currentCI <= threshold && currentEmission <= bestEmission*1.05 {
		bestEmission = currentEmission
		bestStart = forecast[0].Timestamp
	}

	bestEnd := bestStart.Add(time.Duration(duration) * time.Second)
	reduction := 0.0
	if currentEmission > 0 {
		reduction = (currentEmission - bestEmission) / currentEmission * 100
	}

	return suggestionAnalysis{
		CurrentCI:       currentCI,
		CurrentEmission: currentEmission,
		BestStart:       bestStart,
		BestEnd:         bestEnd,
		BestEmission:    bestEmission,
		Reduction:       reduction,
	}, nil
}

func buildSuggestion(zone string, duration int, threshold float64, lookahead int, provider ci.Provider) (string, error) {
	analysis, err := analyzeSuggestion(zone, duration, threshold, lookahead, provider)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"Current CI: %.4f kg/kWh\nBest execution window: %s - %s\nExpected emission: %.4f kg\nEmission reduction vs now: %.2f %%\n",
		analysis.CurrentCI,
		analysis.BestStart.Local().Format("15:04"),
		analysis.BestEnd.Local().Format("15:04"),
		analysis.BestEmission,
		analysis.Reduction,
	), nil
}

func isWithinWindow(now time.Time, start time.Time, end time.Time) bool {
	return (now.Equal(start) || now.After(start)) && now.Before(end)
}

func estimateWindowEmissions(points []ci.ForecastPoint, duration int) (float64, bool) {
	remaining := duration
	segments := make([]calculator.Segment, 0)

	for _, point := range points {
		if remaining <= 0 {
			break
		}

		segmentDuration := 3600
		if remaining < segmentDuration {
			segmentDuration = remaining
		}

		segments = append(segments, calculator.Segment{
			Duration: segmentDuration,
			CI:       point.CI,
		})
		remaining -= segmentDuration
	}

	if remaining > 0 {
		return 0, false
	}

	emission := calculator.EstimateEmissionsWithSegments(segments, "ubuntu", 0.6, 1.2)
	return emission, true
}
