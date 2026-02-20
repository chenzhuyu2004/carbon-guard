package report

import (
	"fmt"
	"math"

	"github.com/czy/carbon-guard/internal/calculator"
)

const divider = "-----------------------------------"

func Build(durationSeconds int, asJSON bool) string {
	emissions := round4(calculator.EstimateEmissionsKg(durationSeconds))

	if asJSON {
		return fmt.Sprintf("{\n  \"duration_seconds\": %d,\n  \"emissions_kg\": %.4f\n}\n", durationSeconds, emissions)
	}

	return fmt.Sprintf("%s\nCarbon Report\n%s\nDuration: %ds\nEstimated Emissions: %.4f kgCO2\n%s\n", divider, divider, durationSeconds, emissions, divider)
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
