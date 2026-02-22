package scheduling

import (
	"sort"
	"time"
)

// FillMode controls how missing points are handled when aligning zones on a shared time axis.
// FillMode 用于控制多区域对齐到同一时间轴时，缺失点如何处理。
type FillMode string

const (
	// FillModeForward keeps the last known value for a bounded age window.
	// FillModeForward 表示在允许的时间窗口内使用“前值填充”。
	FillModeForward FillMode = "forward"
	// FillModeStrict requires exact timestamp matches (no forward fill).
	// FillModeStrict 表示必须精确时间戳匹配（不进行前值填充）。
	FillModeStrict FillMode = "strict"
)

// ResampleOptions defines alignment behavior for cross-zone resampling.
// ResampleOptions 定义跨区域重采样时的对齐策略。
type ResampleOptions struct {
	// FillMode chooses strict matching or bounded forward-fill behavior.
	// FillMode 选择严格匹配或受限前值填充。
	FillMode FillMode
	// MaxFillAge is only used by forward mode; <=0 falls back to default (2*step).
	// MaxFillAge 仅在 forward 模式生效；<=0 时回退到默认值（2*step）。
	MaxFillAge time.Duration
}

// NormalizeForecastUTC converts timestamps to UTC and sorts points by timestamp ascending.
// NormalizeForecastUTC 将时间戳统一到 UTC，并按升序排序。
func NormalizeForecastUTC(points []ForecastPoint) []ForecastPoint {
	out := make([]ForecastPoint, len(points))
	for i, point := range points {
		out[i] = ForecastPoint{
			Timestamp: point.Timestamp.UTC(),
			CI:        point.CI,
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.Before(out[j].Timestamp)
	})

	return out
}

// BuildForecastIndex builds a first-occurrence index keyed by Unix timestamp (seconds).
// BuildForecastIndex 按 Unix 秒时间戳构建“首次出现”索引。
func BuildForecastIndex(points []ForecastPoint) map[int64]int {
	index := make(map[int64]int, len(points))
	for i, point := range points {
		ts := point.Timestamp.UTC().Unix()
		if _, exists := index[ts]; !exists {
			index[ts] = i
		}
	}
	return index
}

// IntersectTimestamps returns UTC timestamps that appear in every zone series.
// IntersectTimestamps 返回所有区域都存在的 UTC 公共时间戳集合。
func IntersectTimestamps(zones []string, zoneForecasts map[string][]ForecastPoint) []time.Time {
	if len(zones) == 0 {
		return nil
	}

	counts := make(map[int64]int)
	for _, zone := range zones {
		seen := make(map[int64]struct{})
		for _, point := range zoneForecasts[zone] {
			ts := point.Timestamp.UTC().Unix()
			seen[ts] = struct{}{}
		}

		for ts := range seen {
			counts[ts]++
		}
	}

	common := make([]time.Time, 0)
	for ts, count := range counts {
		if count == len(zones) {
			common = append(common, time.Unix(ts, 0).UTC())
		}
	}

	sort.Slice(common, func(i, j int) bool {
		return common[i].Before(common[j])
	})

	return common
}

// InferResampleStep infers the smallest positive cadence in source data, clamped to >=5m.
// InferResampleStep 推断源数据中最小正采样间隔，并限制最小为 5 分钟。
func InferResampleStep(zoneForecasts map[string][]ForecastPoint) time.Duration {
	minDeltaSec := int64(0)

	for _, points := range zoneForecasts {
		for i := 1; i < len(points); i++ {
			delta := points[i].Timestamp.UTC().Unix() - points[i-1].Timestamp.UTC().Unix()
			if delta <= 0 {
				continue
			}
			if minDeltaSec == 0 || delta < minDeltaSec {
				minDeltaSec = delta
			}
		}
	}

	if minDeltaSec == 0 {
		return time.Hour
	}
	if minDeltaSec < int64((5 * time.Minute).Seconds()) {
		minDeltaSec = int64((5 * time.Minute).Seconds())
	}
	return time.Duration(minDeltaSec) * time.Second
}

// BuildResampledIntersection is a backward-compatible wrapper that uses default resample options.
// BuildResampledIntersection 是向后兼容入口，使用默认重采样策略。
func BuildResampledIntersection(
	zones []string,
	zoneForecasts map[string][]ForecastPoint,
	step time.Duration,
) ([]time.Time, map[string][]ForecastPoint) {
	return BuildResampledIntersectionWithOptions(zones, zoneForecasts, step, ResampleOptions{})
}

// BuildResampledIntersectionWithOptions aligns all zones on a shared UTC grid.
// BuildResampledIntersectionWithOptions 将所有区域对齐到同一 UTC 网格。
//
// The function first computes the overlap range, builds a regular axis by step,
// then samples each zone under the chosen fill policy, and finally keeps only
// timestamps present in every zone after sampling.
// 该函数先计算重叠时间范围，再按 step 构建规则时间轴，
// 然后按填充策略对每个区域采样，最后保留采样后所有区域都存在的时间戳。
func BuildResampledIntersectionWithOptions(
	zones []string,
	zoneForecasts map[string][]ForecastPoint,
	step time.Duration,
	options ResampleOptions,
) ([]time.Time, map[string][]ForecastPoint) {
	if len(zones) == 0 || step <= 0 {
		return nil, nil
	}

	var maxFirst time.Time
	var minLast time.Time
	for i, zone := range zones {
		points := zoneForecasts[zone]
		if len(points) == 0 {
			return nil, nil
		}

		first := points[0].Timestamp.UTC()
		last := points[len(points)-1].Timestamp.UTC()
		if i == 0 || first.After(maxFirst) {
			maxFirst = first
		}
		if i == 0 || last.Before(minLast) {
			minLast = last
		}
	}

	start := ceilToStep(maxFirst, step)
	end := floorToStep(minLast, step)
	if start.After(end) {
		return nil, nil
	}

	axis := buildUTCGrid(start, end, step)
	if len(axis) == 0 {
		return nil, nil
	}

	strict, maxFillAge := normalizeResampleOptions(step, options)
	zoneSamples := make(map[string]map[int64]float64, len(zones))
	for _, zone := range zones {
		zoneSamples[zone] = resampleZoneOnAxis(zoneForecasts[zone], axis, maxFillAge, strict)
	}

	counts := make(map[int64]int)
	for _, zone := range zones {
		for ts := range zoneSamples[zone] {
			counts[ts]++
		}
	}

	common := make([]time.Time, 0, len(axis))
	for _, t := range axis {
		if counts[t.Unix()] == len(zones) {
			common = append(common, t.UTC())
		}
	}
	if len(common) == 0 {
		return nil, nil
	}

	aligned := make(map[string][]ForecastPoint, len(zones))
	for _, zone := range zones {
		points := make([]ForecastPoint, 0, len(common))
		samples := zoneSamples[zone]
		for _, t := range common {
			ci, ok := samples[t.Unix()]
			if !ok {
				continue
			}
			points = append(points, ForecastPoint{
				Timestamp: t.UTC(),
				CI:        ci,
			})
		}
		aligned[zone] = points
	}

	return common, aligned
}

// normalizeResampleOptions resolves user options into an executable policy.
// normalizeResampleOptions 将输入选项归一化为可执行策略。
func normalizeResampleOptions(step time.Duration, options ResampleOptions) (strict bool, maxFillAge time.Duration) {
	switch options.FillMode {
	case "", FillModeForward:
		maxFillAge = options.MaxFillAge
		if maxFillAge <= 0 {
			maxFillAge = 2 * step
		}
		return false, maxFillAge
	case FillModeStrict:
		return true, 0
	default:
		maxFillAge = options.MaxFillAge
		if maxFillAge <= 0 {
			maxFillAge = 2 * step
		}
		return false, maxFillAge
	}
}

// buildUTCGrid builds an inclusive UTC grid [start, end] with fixed step.
// buildUTCGrid 构建包含端点的 UTC 规则网格 [start, end]。
func buildUTCGrid(start time.Time, end time.Time, step time.Duration) []time.Time {
	if step <= 0 || start.After(end) {
		return nil
	}

	total := int(end.Sub(start)/step) + 1
	grid := make([]time.Time, 0, total)
	for t := start.UTC(); !t.After(end.UTC()); t = t.Add(step) {
		grid = append(grid, t.UTC())
	}
	return grid
}

// resampleZoneOnAxis samples one zone on the target axis.
// resampleZoneOnAxis 将单个区域序列采样到目标时间轴。
//
// strict=true requires exact timestamp equality.
// strict=false allows bounded forward-fill controlled by maxFillAge.
// strict=true 表示必须精确匹配时间戳。
// strict=false 表示允许在 maxFillAge 内进行前值填充。
func resampleZoneOnAxis(points []ForecastPoint, axis []time.Time, maxFillAge time.Duration, strict bool) map[int64]float64 {
	out := make(map[int64]float64, len(axis))
	if len(points) == 0 {
		return out
	}

	idx := 0
	for _, t := range axis {
		for idx+1 < len(points) && !points[idx+1].Timestamp.UTC().After(t.UTC()) {
			idx++
		}

		source := points[idx].Timestamp.UTC()
		if source.After(t.UTC()) {
			continue
		}
		if strict && !source.Equal(t.UTC()) {
			continue
		}
		if maxFillAge > 0 && t.UTC().Sub(source) > maxFillAge {
			continue
		}

		out[t.UTC().Unix()] = points[idx].CI
	}

	return out
}

// floorToStep rounds time down to the nearest step boundary in UTC.
// floorToStep 将 UTC 时间向下对齐到最近 step 边界。
func floorToStep(t time.Time, step time.Duration) time.Time {
	stepSec := int64(step.Seconds())
	if stepSec <= 0 {
		return t.UTC()
	}

	sec := t.UTC().Unix()
	rem := sec % stepSec
	return time.Unix(sec-rem, 0).UTC()
}

// ceilToStep rounds time up to the nearest step boundary in UTC.
// ceilToStep 将 UTC 时间向上对齐到最近 step 边界。
func ceilToStep(t time.Time, step time.Duration) time.Time {
	stepSec := int64(step.Seconds())
	if stepSec <= 0 {
		return t.UTC()
	}

	sec := t.UTC().Unix()
	rem := sec % stepSec
	if rem == 0 {
		return time.Unix(sec, 0).UTC()
	}
	return time.Unix(sec+(stepSec-rem), 0).UTC()
}
