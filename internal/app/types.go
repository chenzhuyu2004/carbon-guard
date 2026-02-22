package app

import "time"

type ModelContext struct {
	Runner string
	Load   float64
	PUE    float64
}

type RunInput struct {
	Duration    int
	Region      string
	SegmentsRaw string
	LiveZone    string
	Model       ModelContext
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
	WaitCost  float64
	Model     ModelContext
}

type SuggestOutput struct {
	CurrentCI              float64
	BestWindowStartUTC     time.Time
	BestWindowEndUTC       time.Time
	ExpectedEmissionKg     float64
	EmissionReductionVsNow float64
}

type SuggestionAnalysis struct {
	// Current* describes the first valid window at evaluation start.
	// Current* 描述评估起点对应的首个有效窗口。
	CurrentEmission float64
	CurrentStart    time.Time
	CurrentEnd      time.Time
	CurrentScore    float64
	// Best* describes the minimal-score window under current objective.
	// Best* 描述当前目标函数下评分最小的窗口。
	BestStart    time.Time
	BestEnd      time.Time
	BestEmission float64
	BestScore    float64
	Reduction    float64
}

type RunAwareInput struct {
	Zone           string
	Duration       int
	Threshold      float64
	ThresholdEnter float64
	ThresholdExit  float64
	Lookahead      int
	Model          ModelContext
	MaxWait        time.Duration
	// NoRegretMaxDelay disables waiting for marginal gains when the best window is too far away.
	// NoRegretMaxDelay 用于限制“收益很小却等待太久”的场景；<=0 表示关闭。
	NoRegretMaxDelay time.Duration
	// NoRegretMinReductionPct defines the minimum expected reduction (%) required to justify waiting.
	// NoRegretMinReductionPct 定义“值得等待”的最小预期减排百分比；<=0 表示关闭。
	NoRegretMinReductionPct float64
	PollEvery               time.Duration
	StatusFunc              func(string)
}

type RunAwareOutput struct {
	Message string
}

type ZoneResult struct {
	Zone      string
	Emission  float64
	Score     float64
	BestStart time.Time
	BestEnd   time.Time
}

type OptimizeInput struct {
	Zones     []string
	Duration  int
	Lookahead int
	WaitCost  float64
	Model     ModelContext
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
	// ResampleFillMode controls cross-zone alignment: forward|strict.
	// ResampleFillMode 控制跨区域对齐策略：forward|strict。
	Zones            []string
	Duration         int
	Lookahead        int
	WaitCost         float64
	ResampleFillMode string
	// ResampleMaxFillAge is only effective in forward mode.
	// ResampleMaxFillAge 仅在 forward 模式下生效。
	ResampleMaxFillAge time.Duration
	Model              ModelContext
	Timeout            time.Duration
}

type OptimizeGlobalOutput struct {
	BestZone  string
	BestStart time.Time
	BestEnd   time.Time
	Emission  float64
	Reduction float64
	// Resample* echoes effective policy so outputs are audit-friendly.
	// Resample* 回显实际生效策略，便于审计与复现。
	ResampleFillMode          string
	ResampleMaxFillAgeSeconds int64
}
