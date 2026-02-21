package scheduling

import (
	"time"

	"github.com/czy/carbon-guard/internal/calculator"
)

func IsWithinWindow(now time.Time, start time.Time, end time.Time) bool {
	now = now.UTC()
	start = start.UTC()
	end = end.UTC()
	return (now.Equal(start) || now.After(start)) && now.Before(end)
}

func EstimateWindowEmissions(points []ForecastPoint, duration int, runner string, load float64, pue float64) (float64, bool) {
	remaining := duration
	segments := make([]calculator.Segment, 0)

	for _, point := range points {
		if remaining <= 0 {
			break
		}

		segmentDuration := 3600
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
