package ci

import "time"

type ForecastPoint struct {
	Timestamp time.Time
	CI        float64 // kgCO2/kWh
}

type Provider interface {
	GetCurrentCI(zone string) (float64, error)
	GetForecastCI(zone string, hours int) ([]ForecastPoint, error)
}
