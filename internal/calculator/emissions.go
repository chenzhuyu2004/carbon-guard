package calculator

import "github.com/czy/carbon-guard/pkg"

type PowerProfile struct {
	Idle float64
	Peak float64
}

type Segment struct {
	Duration int
	CI       float64
}

var runnerProfiles = map[string]PowerProfile{
	"ubuntu":  {Idle: 110, Peak: 220},
	"windows": {Idle: 150, Peak: 300},
	"macos":   {Idle: 100, Peak: 200},
}

var regionCarbonIntensity = map[string]float64{
	"global": 0.4,
	"china":  0.58,
	"us":     0.38,
	"eu":     0.28,
}

func EstimateEmissionsKg(durationSeconds int) float64 {
	return float64(durationSeconds) * pkg.PowerWatts * pkg.EmissionsFactorKgPerKWh / pkg.WattsPerKilowatt / pkg.SecondsPerHour
}

func EstimateEmissionsAdvanced(duration int, runner string, region string, load float64, pue float64) float64 {
	profile, ok := runnerProfiles[runner]
	if !ok {
		profile = runnerProfiles["ubuntu"]
	}

	ci, ok := regionCarbonIntensity[region]
	if !ok {
		ci = regionCarbonIntensity["global"]
	}

	power := profile.Idle + (profile.Peak-profile.Idle)*load
	energyKWh := float64(duration) * power / 1000.0 / 3600.0
	energyTotal := energyKWh * pue
	return energyTotal * ci
}

func EstimateEmissionsWithSegments(
	segments []Segment,
	runner string,
	load float64,
	pue float64,
) float64 {
	profile, ok := runnerProfiles[runner]
	if !ok {
		profile = runnerProfiles["ubuntu"]
	}

	power := profile.Idle + (profile.Peak-profile.Idle)*load

	total := 0.0
	for _, segment := range segments {
		energyKWh := float64(segment.Duration) * power / 1000.0 / 3600.0
		total += energyKWh * pue * segment.CI
	}

	return total
}
