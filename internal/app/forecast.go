package app

import (
	"time"

	"github.com/chenzhuyu2004/carbon-guard/internal/domain/scheduling"
)

func resolveEvalStart(evalStart time.Time) time.Time {
	if evalStart.IsZero() {
		return time.Now().UTC()
	}
	return evalStart.UTC()
}

func clipForecastToWindow(
	points []scheduling.ForecastPoint,
	evalStart time.Time,
	lookahead int,
) []scheduling.ForecastPoint {
	if len(points) == 0 || lookahead <= 0 {
		return nil
	}

	start := resolveEvalStart(evalStart)
	limit := start.Add(time.Duration(lookahead) * time.Hour).UTC()
	clipped := make([]scheduling.ForecastPoint, 0, len(points))

	for _, point := range points {
		ts := point.Timestamp.UTC()
		if ts.Before(start) || ts.After(limit) {
			continue
		}
		clipped = append(clipped, scheduling.ForecastPoint{
			Timestamp: ts,
			CI:        point.CI,
		})
	}

	return clipped
}
