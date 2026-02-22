package scheduling

import (
	"time"

	"github.com/chenzhuyu2004/carbon-guard/internal/calculator"
)

const defaultForecastSliceSeconds = 3600

func IsWithinWindow(now time.Time, start time.Time, end time.Time) bool {
	now = now.UTC()
	start = start.UTC()
	end = end.UTC()
	return (now.Equal(start) || now.After(start)) && now.Before(end)
}

func ForecastCoverageSeconds(points []ForecastPoint, windowEnd time.Time) int {
	total := 0
	for i := range points {
		total += inferredSliceDurationSeconds(points, i, windowEnd)
	}
	return total
}

func EstimateWindowEmissions(
	points []ForecastPoint,
	duration int,
	runner string,
	load float64,
	pue float64,
	windowEnd time.Time,
) (float64, bool) {
	if duration <= 0 {
		return 0, false
	}

	remaining := duration
	segments := make([]calculator.Segment, 0)

	for i, point := range points {
		if remaining <= 0 {
			break
		}

		segmentDuration := inferredSliceDurationSeconds(points, i, windowEnd)
		if segmentDuration <= 0 {
			continue
		}
		if remaining < segmentDuration {
			segmentDuration = remaining
		}

		segments = append(segments, calculator.Segment{
			Duration: segmentDuration,
			CI:       point.CI,
		})
		remaining -= segmentDuration
	}

	if remaining > 0 {
		return 0, false
	}

	emission := calculator.EstimateEmissionsWithSegments(segments, runner, load, pue)
	return emission, true
}

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
