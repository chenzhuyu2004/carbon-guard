package calculator

import "github.com/czy/carbon-guard/pkg"

func EstimateEmissionsKg(durationSeconds int) float64 {
	return float64(durationSeconds) * pkg.PowerWatts * pkg.EmissionsFactorKgPerKWh / pkg.WattsPerKilowatt / pkg.SecondsPerHour
}
