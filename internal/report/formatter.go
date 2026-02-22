package report

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/chenzhuyu2004/carbon-guard/pkg"
)

const divider = "-----------------------------------"

type BuildOptions struct {
	BudgetKg            float64
	BaselineKg          float64
	EnergyTotalKWh      float64
	EffectiveCIKgPerKWh float64
}

type emissionUnit struct {
	Symbol   string
	Multiple float64
}

var commonEmissionUnits = []emissionUnit{
	{Symbol: "GtCO2", Multiple: 1e-12},
	{Symbol: "MtCO2", Multiple: 1e-9},
	{Symbol: "ktCO2", Multiple: 1e-6},
	{Symbol: "tCO2", Multiple: 1e-3},
	{Symbol: "kgCO2", Multiple: 1},
	{Symbol: "gCO2", Multiple: 1e3},
	{Symbol: "mgCO2", Multiple: 1e6},
}

const (
	autoUnitMinValue = 1.0
	autoUnitMaxValue = 1000.0
)

func BuildFromEmissions(durationSeconds int, asJSON bool, emissions float64, opts BuildOptions) string {
	emissions = round4(emissions)
	budgetKg := round4(opts.BudgetKg)
	baselineKg := round4(opts.BaselineKg)
	budgetExceeded := budgetKg > 0 && emissions > budgetKg

	if asJSON {
		payload := map[string]any{
			"duration_seconds": durationSeconds,
			"emissions_kg":     emissions,
		}
		if budgetKg > 0 {
			payload["budget_kg"] = budgetKg
			payload["budget_exceeded"] = budgetExceeded
		}
		if baselineKg > 0 {
			payload["baseline_kg"] = baselineKg
			payload["delta_vs_baseline_pct"] = round2(deltaVsBaselinePct(emissions, baselineKg))
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Sprintf("{\n  \"duration_seconds\": %d,\n  \"emissions_kg\": %.4f\n}\n", durationSeconds, emissions)
		}
		return string(data) + "\n"
	}

	smartphoneCharges, evKilometers := buildComparisons(emissions, opts)
	score, emoji := carbonScore(emissions, durationSeconds, opts)
	emissionsLine := formatEmissionDisplay(emissions)
	report := fmt.Sprintf(
		"%s\nCarbon Report\n%s\nDuration: %ds\nEstimated Emissions: %s\nCarbon Score: %s %s\nFun Facts:\n- Equivalent to charging %.2f smartphones\n- Equivalent to driving %.2f km in an EV\n",
		divider,
		divider,
		durationSeconds,
		emissionsLine,
		score,
		emoji,
		smartphoneCharges,
		evKilometers,
	)

	if budgetKg > 0 {
		status := "within budget"
		if budgetExceeded {
			status = "budget exceeded"
		}
		report += fmt.Sprintf("Budget: %s (%s)\n", formatEmissionDisplay(budgetKg), status)
	}
	if baselineKg > 0 {
		report += fmt.Sprintf("Baseline: %s (delta: %.2f%%)\n", formatEmissionDisplay(baselineKg), deltaVsBaselinePct(emissions, baselineKg))
	}

	return report + divider + "\n"
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func buildComparisons(emissionsKg float64, opts BuildOptions) (float64, float64) {
	energyKWh := opts.EnergyTotalKWh
	if energyKWh <= 0 {
		if emissionsKg <= 0 {
			return 0, 0
		}
		energyKWh = emissionsKg / pkg.EmissionsFactorKgPerKWh
	}
	if energyKWh <= 0 {
		return 0, 0
	}

	smartphoneCharges := energyKWh / pkg.SmartphoneChargeKWh
	evKilometers := energyKWh / pkg.EVKilometerKWh

	return smartphoneCharges, evKilometers
}

func carbonScore(emissionsKg float64, durationSeconds int, opts BuildOptions) (string, string) {
	ciGPerKWh := 0.0
	if opts.EffectiveCIKgPerKWh > 0 {
		ciGPerKWh = opts.EffectiveCIKgPerKWh * 1000
	} else {
		if durationSeconds <= 0 {
			return "C", "🏭"
		}
		energyKWh := float64(durationSeconds) * pkg.PowerWatts / pkg.WattsPerKilowatt / pkg.SecondsPerHour
		if energyKWh <= 0 {
			return "C", "🏭"
		}
		ciGPerKWh = (emissionsKg / energyKWh) * 1000
	}
	if ciGPerKWh <= 0 {
		return "C", "🏭"
	}

	switch {
	case ciGPerKWh < pkg.CIScoreGreenMaxGPerKWh:
		return "A", "🌿"
	case ciGPerKWh < pkg.CIScoreYellowMaxGPerKWh:
		return "B", "⚠️"
	default:
		return "C", "🏭"
	}
}

func deltaVsBaselinePct(emissionsKg float64, baselineKg float64) float64 {
	if baselineKg <= 0 {
		return 0
	}
	return (emissionsKg - baselineKg) / baselineKg * 100
}

func formatEmissionDisplay(emissionsKg float64) string {
	primary := autoScaledEmission(emissionsKg)
	kgRef := fmt.Sprintf("%.4f kgCO2", round4(emissionsKg))
	if primary.Symbol == "kgCO2" {
		return kgRef
	}
	return fmt.Sprintf("%s (%s)", formatScaledEmission(primary.Value, primary.Symbol), kgRef)
}

func formatScaledEmission(value float64, symbol string) string {
	abs := math.Abs(value)
	switch {
	case abs >= 100:
		return fmt.Sprintf("%.1f %s", value, symbol)
	case abs >= 10:
		return fmt.Sprintf("%.2f %s", value, symbol)
	case abs >= 1:
		return fmt.Sprintf("%.2f %s", value, symbol)
	case abs >= 0.1:
		return fmt.Sprintf("%.3f %s", value, symbol)
	default:
		return fmt.Sprintf("%.4f %s", value, symbol)
	}
}

type scaledEmission struct {
	Value  float64
	Symbol string
}

func autoScaledEmission(emissionsKg float64) scaledEmission {
	if emissionsKg == 0 {
		return scaledEmission{Value: 0, Symbol: "kgCO2"}
	}

	absKg := math.Abs(emissionsKg)
	for _, u := range commonEmissionUnits {
		v := absKg * u.Multiple
		if v >= autoUnitMinValue && v < autoUnitMaxValue {
			return scaledEmission{
				Value:  signedScaled(emissionsKg, u.Multiple),
				Symbol: u.Symbol,
			}
		}
	}

	smallest := commonEmissionUnits[len(commonEmissionUnits)-1]
	largest := commonEmissionUnits[0]
	if absKg*smallest.Multiple < autoUnitMinValue {
		return scaledEmission{
			Value:  signedScaled(emissionsKg, smallest.Multiple),
			Symbol: smallest.Symbol,
		}
	}
	return scaledEmission{
		Value:  signedScaled(emissionsKg, largest.Multiple),
		Symbol: largest.Symbol,
	}
}

func signedScaled(kg float64, multiple float64) float64 {
	return kg * multiple
}
