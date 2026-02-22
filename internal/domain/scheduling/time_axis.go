package scheduling

import (
	"sort"
	"time"
)

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

func BuildResampledIntersection(
	zones []string,
	zoneForecasts map[string][]ForecastPoint,
	step time.Duration,
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

	maxFillAge := 2 * step
	zoneSamples := make(map[string]map[int64]float64, len(zones))
	for _, zone := range zones {
		zoneSamples[zone] = resampleZoneOnAxis(zoneForecasts[zone], axis, maxFillAge)
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

func resampleZoneOnAxis(points []ForecastPoint, axis []time.Time, maxFillAge time.Duration) map[int64]float64 {
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
		if maxFillAge > 0 && t.UTC().Sub(source) > maxFillAge {
			continue
		}

		out[t.UTC().Unix()] = points[idx].CI
	}

	return out
}

func floorToStep(t time.Time, step time.Duration) time.Time {
	stepSec := int64(step.Seconds())
	if stepSec <= 0 {
		return t.UTC()
	}

	sec := t.UTC().Unix()
	rem := sec % stepSec
	return time.Unix(sec-rem, 0).UTC()
}

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
