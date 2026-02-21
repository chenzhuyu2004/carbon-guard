package calculator

import (
	"github.com/chenzhuyu2004/carbon-guard/pkg"
	"github.com/chenzhuyu2004/carbon-guard/pkg/models"
)

type Segment struct {
	Duration int
	CI       float64
}

func EstimateEmissionsKg(durationSeconds int) float64 {
	return float64(durationSeconds) * pkg.PowerWatts * pkg.EmissionsFactorKgPerKWh / pkg.WattsPerKilowatt / pkg.SecondsPerHour
}

func EstimateEmissionsAdvanced(duration int, runner string, region string, load float64, pue float64) float64 {
	profile, ok := models.RunnerProfiles[runner]
	if !ok {
		profile = models.RunnerProfiles["ubuntu"]
	}

	ci, ok := models.RegionCarbonIntensity[region]
	if !ok {
		ci = models.RegionCarbonIntensity["global"]
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
	profile, ok := models.RunnerProfiles[runner]
	if !ok {
		profile = models.RunnerProfiles["ubuntu"]
	}

	power := profile.Idle + (profile.Peak-profile.Idle)*load

	total := 0.0
	for _, segment := range segments {
		energyKWh := float64(segment.Duration) * power / 1000.0 / 3600.0
		total += energyKWh * pue * segment.CI
	}

	return total
}
