package models

type PowerProfile struct {
	Idle float64
	Peak float64
}

var RunnerProfiles = map[string]PowerProfile{
	"ubuntu":  {Idle: 110, Peak: 220},
	"windows": {Idle: 150, Peak: 300},
	"macos":   {Idle: 100, Peak: 200},
}

var RegionCarbonIntensity = map[string]float64{
	"global": 0.4,
	"china":  0.58,
	"us":     0.38,
	"eu":     0.28,
}
