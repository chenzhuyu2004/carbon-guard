package scheduling

import "time"

type ForecastPoint struct {
	Timestamp time.Time
	CI        float64 // kgCO2/kWh
}
