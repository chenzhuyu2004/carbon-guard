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

func ForecastCoverageSeconds(points []ForecastPoint) int {
	total := 0
	for i := range points {
		total += inferredSliceDurationSeconds(points, i)
	}
	return total
}

func EstimateWindowEmissions(points []ForecastPoint, duration int, runner string, load float64, pue float64) (float64, bool) {
	if duration <= 0 {
		return 0, false
	}

	remaining := duration
	segments := make([]calculator.Segment, 0)

	for i, point := range points {
		if remaining <= 0 {
			break
		}

		segmentDuration := inferredSliceDurationSeconds(points, i)
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

func inferredSliceDurationSeconds(points []ForecastPoint, idx int) int {
	if idx < 0 || idx >= len(points) {
		return 0
	}

	base := points[idx].Timestamp.UTC()
	for i := idx + 1; i < len(points); i++ {
		delta := int(points[i].Timestamp.UTC().Sub(base).Seconds())
		if delta > 0 {
			return delta
		}
	}

	for i := idx - 1; i >= 0; i-- {
		delta := int(base.Sub(points[i].Timestamp.UTC()).Seconds())
		if delta > 0 {
			return delta
		}
	}

	return defaultForecastSliceSeconds
}
