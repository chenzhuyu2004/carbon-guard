package scheduling

import (
	"sort"
	"time"

	"github.com/chenzhuyu2004/carbon-guard/internal/calculator"
)

const defaultForecastSliceSeconds = 3600

// WindowEstimate represents one executable window and its estimated emission.
// WindowEstimate 表示一个可执行时间窗口及其估算排放量。
type WindowEstimate struct {
	Start    time.Time
	End      time.Time
	Emission float64
}

// EmissionEvaluator precomputes piecewise CI integrals for O(1) range-mean queries.
// EmissionEvaluator 预计算分段 CI 的前缀积分，用于 O(1) 区间均值计算。
//
// All offsets are in seconds relative to base.
// 所有偏移量均以 base 为参考、单位为秒。
type EmissionEvaluator struct {
	base        time.Time
	starts      []int64
	ends        []int64
	ci          []float64
	prefix      []float64
	coverageSec int
	coverageEnd int64
}

type integralCursor struct {
	e   EmissionEvaluator
	idx int
}

// IsWithinWindow checks whether now is in [start, end) under UTC.
// IsWithinWindow 判断当前时刻是否位于 UTC 窗口 [start, end)。
func IsWithinWindow(now time.Time, start time.Time, end time.Time) bool {
	now = now.UTC()
	start = start.UTC()
	end = end.UTC()
	return (now.Equal(start) || now.After(start)) && now.Before(end)
}

// BuildEmissionEvaluator builds an evaluator from timestamped CI points and a window end bound.
// BuildEmissionEvaluator 根据 CI 时间序列与窗口终点构建排放评估器。
//
// Each point contributes a segment duration inferred from neighboring points.
// 每个点的有效时长由相邻点间隔推断得到。
func BuildEmissionEvaluator(points []ForecastPoint, windowEnd time.Time) (EmissionEvaluator, bool) {
	if len(points) == 0 {
		return EmissionEvaluator{}, false
	}

	base := points[0].Timestamp.UTC()
	starts := make([]int64, 0, len(points))
	ends := make([]int64, 0, len(points))
	ci := make([]float64, 0, len(points))

	for i, point := range points {
		segmentDuration := inferredSliceDurationSeconds(points, i, windowEnd)
		if segmentDuration <= 0 {
			continue
		}

		start := int64(point.Timestamp.UTC().Sub(base).Seconds())
		if start < 0 {
			continue
		}
		end := start + int64(segmentDuration)
		if end <= start {
			continue
		}

		starts = append(starts, start)
		ends = append(ends, end)
		ci = append(ci, point.CI)
	}

	if len(starts) == 0 {
		return EmissionEvaluator{}, false
	}

	prefix := make([]float64, len(starts)+1)
	coverageSec := 0
	for i := range starts {
		prefix[i+1] = prefix[i] + float64(ends[i]-starts[i])*ci[i]
		coverageSec += int(ends[i] - starts[i])
	}

	return EmissionEvaluator{
		base:        base,
		starts:      starts,
		ends:        ends,
		ci:          ci,
		prefix:      prefix,
		coverageSec: coverageSec,
		coverageEnd: ends[len(ends)-1],
	}, true
}

// ForecastCoverageSeconds returns total evaluable seconds from input points.
// ForecastCoverageSeconds 返回输入序列可评估的总秒数。
func ForecastCoverageSeconds(points []ForecastPoint, windowEnd time.Time) int {
	evaluator, ok := BuildEmissionEvaluator(points, windowEnd)
	if !ok {
		return 0
	}
	return evaluator.CoverageSeconds()
}

// EstimateWindowEmissions estimates emissions starting at the first point for a fixed duration.
// EstimateWindowEmissions 以首个点为起点，估算固定时长的排放量。
func EstimateWindowEmissions(
	points []ForecastPoint,
	duration int,
	runner string,
	load float64,
	pue float64,
	windowEnd time.Time,
) (float64, bool) {
	if len(points) == 0 {
		return 0, false
	}
	evaluator, ok := BuildEmissionEvaluator(points, windowEnd)
	if !ok {
		return 0, false
	}
	return evaluator.EstimateAt(points[0].Timestamp.UTC(), duration, runner, load, pue)
}

// CoverageSeconds returns the evaluator's total covered duration in seconds.
// CoverageSeconds 返回评估器可覆盖的总秒数。
func (e EmissionEvaluator) CoverageSeconds() int {
	return e.coverageSec
}

// EstimateAt estimates emission for [start, start+duration) with model parameters.
// EstimateAt 在给定模型参数下估算区间 [start, start+duration) 的排放量。
func (e EmissionEvaluator) EstimateAt(
	start time.Time,
	duration int,
	runner string,
	load float64,
	pue float64,
) (float64, bool) {
	startOffset := int64(start.UTC().Sub(e.base).Seconds())
	return e.EstimateAtOffset(startOffset, duration, runner, load, pue)
}

// EstimateAtOffset estimates emission by offset seconds from base.
// EstimateAtOffset 通过相对 base 的秒偏移估算排放量。
//
// It computes mean CI by integral difference and applies energy model in calculator.
// 其核心是通过积分差计算平均 CI，再套用 calculator 中的能耗模型。
func (e EmissionEvaluator) EstimateAtOffset(
	startOffset int64,
	duration int,
	runner string,
	load float64,
	pue float64,
) (float64, bool) {
	if duration <= 0 || len(e.starts) == 0 {
		return 0, false
	}
	if startOffset < e.starts[0] {
		return 0, false
	}

	endOffset := startOffset + int64(duration)
	if endOffset > e.coverageEnd {
		return 0, false
	}

	ciSeconds := e.integralAt(endOffset) - e.integralAt(startOffset)
	avgCI := ciSeconds / float64(duration)
	segments := []calculator.Segment{
		{
			Duration: duration,
			CI:       avgCI,
		},
	}
	emission := calculator.EstimateEmissionsWithSegments(segments, runner, load, pue)
	return emission, true
}

// FindBestWindowAtForecastStarts scans candidate starts at forecast timestamps.
// FindBestWindowAtForecastStarts 在 forecast 时间戳上扫描候选起点。
//
// Returns:
// - current: first valid window (acts as "run now" baseline in this axis)
// - best: minimal-emission window among all valid starts
// 返回值：
// - current：第一个有效窗口（作为该时间轴上的“立即执行”基准）
// - best：所有有效起点中排放最小的窗口
func FindBestWindowAtForecastStarts(
	points []ForecastPoint,
	evaluator EmissionEvaluator,
	duration int,
	runner string,
	load float64,
	pue float64,
) (WindowEstimate, WindowEstimate, bool) {
	if duration <= 0 || len(points) == 0 || len(evaluator.starts) == 0 {
		return WindowEstimate{}, WindowEstimate{}, false
	}

	startCursor := evaluator.newCursor()
	endCursor := evaluator.newCursor()
	segment := []calculator.Segment{{Duration: duration}}

	found := false
	var current WindowEstimate
	var best WindowEstimate

	for _, point := range points {
		start := point.Timestamp.UTC()
		startOffset := int64(start.Sub(evaluator.base).Seconds())
		if startOffset < evaluator.starts[0] {
			continue
		}

		endOffset := startOffset + int64(duration)
		if endOffset > evaluator.coverageEnd {
			break
		}

		ciSeconds := endCursor.integralAt(endOffset) - startCursor.integralAt(startOffset)
		avgCI := ciSeconds / float64(duration)
		segment[0].CI = avgCI
		emission := calculator.EstimateEmissionsWithSegments(segment, runner, load, pue)

		candidate := WindowEstimate{
			Start:    start,
			End:      start.Add(time.Duration(duration) * time.Second).UTC(),
			Emission: emission,
		}

		if !found {
			current = candidate
			best = candidate
			found = true
			continue
		}

		if candidate.Emission < best.Emission {
			best = candidate
		}
	}

	if !found {
		return WindowEstimate{}, WindowEstimate{}, false
	}
	return current, best, true
}

// integralAt returns cumulative (CI * seconds) from base to offset.
// integralAt 返回从 base 到 offset 的累计积分（CI * 秒）。
func (e EmissionEvaluator) integralAt(offset int64) float64 {
	if len(e.starts) == 0 {
		return 0
	}
	if offset <= e.starts[0] {
		return 0
	}
	if offset >= e.coverageEnd {
		return e.prefix[len(e.prefix)-1]
	}

	idx := sort.Search(len(e.ends), func(i int) bool {
		return e.ends[i] > offset
	})
	if idx >= len(e.ends) {
		return e.prefix[len(e.prefix)-1]
	}
	if offset <= e.starts[idx] {
		return e.prefix[idx]
	}
	return e.prefix[idx] + float64(offset-e.starts[idx])*e.ci[idx]
}

// newCursor creates a monotonic integral cursor for linear-time scans.
// newCursor 创建单调游标，用于线性扫描时减少重复查找。
func (e EmissionEvaluator) newCursor() *integralCursor {
	return &integralCursor{e: e}
}

// integralAt evaluates cumulative integral using cursor state.
// integralAt 使用游标状态计算累计积分。
func (c *integralCursor) integralAt(offset int64) float64 {
	e := c.e
	if len(e.starts) == 0 {
		return 0
	}
	if offset <= e.starts[0] {
		return 0
	}
	if offset >= e.coverageEnd {
		return e.prefix[len(e.prefix)-1]
	}

	for c.idx < len(e.ends) && offset >= e.ends[c.idx] {
		c.idx++
	}
	if c.idx >= len(e.ends) {
		return e.prefix[len(e.prefix)-1]
	}
	if offset <= e.starts[c.idx] {
		return e.prefix[c.idx]
	}
	return e.prefix[c.idx] + float64(offset-e.starts[c.idx])*e.ci[c.idx]
}

// inferredSliceDurationSeconds infers one point's effective duration in seconds.
// inferredSliceDurationSeconds 推断单个点的有效持续时间（秒）。
//
// Priority:
// 1) next - current
// 2) current - previous
// 3) defaultForecastSliceSeconds
// 优先级：
// 1) 下一个点与当前点间隔
// 2) 当前点与上一个点间隔
// 3) 默认值 defaultForecastSliceSeconds
func inferredSliceDurationSeconds(points []ForecastPoint, idx int, windowEnd time.Time) int {
	if idx < 0 || idx >= len(points) {
		return 0
	}

	base := points[idx].Timestamp.UTC()
	maxAvailable := int(windowEnd.UTC().Sub(base).Seconds())
	if maxAvailable <= 0 {
		return 0
	}

	candidate := 0
	if idx+1 < len(points) {
		candidate = int(points[idx+1].Timestamp.UTC().Sub(base).Seconds())
	} else if idx-1 >= 0 {
		candidate = int(base.Sub(points[idx-1].Timestamp.UTC()).Seconds())
	} else {
		candidate = defaultForecastSliceSeconds
	}

	if candidate <= 0 {
		return 0
	}
	if candidate > maxAvailable {
		candidate = maxAvailable
	}
	return candidate
}
