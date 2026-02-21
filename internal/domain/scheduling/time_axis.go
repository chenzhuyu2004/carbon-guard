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
