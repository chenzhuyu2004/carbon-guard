package calculator

import "github.com/czy/carbon-guard/pkg"

var runnerPower = map[string]float64{
	"ubuntu":  220,
	"windows": 300,
	"macos":   200,
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

func EstimateEmissionsAdvanced(duration int, runner string, region string, load float64) float64 {
	power, ok := runnerPower[runner]
	if !ok {
		power = runnerPower["ubuntu"]
	}

	ci, ok := regionCarbonIntensity[region]
	if !ok {
		ci = regionCarbonIntensity["global"]
	}

	energyKWh := float64(duration) * power * load / 1000.0 / 3600.0
	return energyKWh * ci
}
