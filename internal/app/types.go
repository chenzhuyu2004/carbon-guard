package app

import "time"

type RunInput struct {
	Duration    int
	Runner      string
	Region      string
	Load        float64
	PUE         float64
	SegmentsRaw string
	LiveZone    string
}

type RunResult struct {
	DurationSeconds     int
	EmissionsKg         float64
	EnergyITKWh         float64
	EnergyTotalKWh      float64
	EffectiveCIKgPerKWh float64
}

type SuggestInput struct {
	Zone      string
	Duration  int
	Threshold float64
	Lookahead int
	Runner    string
	Load      float64
	PUE       float64
}

type SuggestOutput struct {
	CurrentCI              float64
	BestWindowStartUTC     time.Time
	BestWindowEndUTC       time.Time
	ExpectedEmissionKg     float64
	EmissionReductionVsNow float64
}

type SuggestionAnalysis struct {
	CurrentEmission float64
	CurrentStart    time.Time
	CurrentEnd      time.Time
	BestStart       time.Time
	BestEnd         time.Time
	BestEmission    float64
	Reduction       float64
}

type RunAwareInput struct {
	Zone       string
	Duration   int
	Threshold  float64
	Lookahead  int
	Runner     string
	Load       float64
	PUE        float64
	MaxWait    time.Duration
	PollEvery  time.Duration
	StatusFunc func(string)
}

type RunAwareOutput struct {
	Message string
}

type ZoneResult struct {
	Zone      string
	Emission  float64
	BestStart time.Time
	BestEnd   time.Time
}

type OptimizeInput struct {
	Zones     []string
	Duration  int
	Lookahead int
	Runner    string
	Load      float64
	PUE       float64
	Timeout   time.Duration
}

type OptimizeOutput struct {
	Results   []ZoneResult
	Failures  map[string]string
	Best      ZoneResult
	Worst     ZoneResult
	Reduction float64
}

type OptimizeGlobalInput struct {
	Zones     []string
	Duration  int
	Lookahead int
	Runner    string
	Load      float64
	PUE       float64
	Timeout   time.Duration
}

type OptimizeGlobalOutput struct {
	BestZone  string
	BestStart time.Time
	BestEnd   time.Time
	Emission  float64
	Reduction float64
}
