package scheduling

import (
	"reflect"
	"testing"
	"time"
)

func TestNormalizeForecastUTCSortsAndConverts(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	points := []ForecastPoint{
		{
			Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, loc),
			CI:        0.3,
		},
		{
			Timestamp: time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC),
			CI:        0.5,
		},
	}

	got := NormalizeForecastUTC(points)
	if len(got) != 2 {
		t.Fatalf("NormalizeForecastUTC() len = %d, expected 2", len(got))
	}
	if !got[0].Timestamp.Equal(time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC)) {
		t.Fatalf("first timestamp = %v, expected %v", got[0].Timestamp, time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC))
	}
	if got[0].Timestamp.Location() != time.UTC || got[1].Timestamp.Location() != time.UTC {
		t.Fatalf("timestamps should be normalized to UTC")
	}
	if !got[0].Timestamp.Before(got[1].Timestamp) {
		t.Fatalf("expected sorted ascending timestamps")
	}
}

func TestBuildForecastIndexKeepsFirstOccurrence(t *testing.T) {
	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	points := []ForecastPoint{
		{Timestamp: ts, CI: 0.3},
		{Timestamp: ts, CI: 0.5},
		{Timestamp: ts.Add(1 * time.Hour), CI: 0.7},
	}

	index := BuildForecastIndex(points)
	if len(index) != 2 {
		t.Fatalf("BuildForecastIndex() len = %d, expected 2", len(index))
	}
	if got := index[ts.Unix()]; got != 0 {
		t.Fatalf("index for duplicated timestamp = %d, expected 0", got)
	}
}

func TestIntersectTimestampsReturnsSortedCommonUTC(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	zones := []string{"DE", "FR", "PL"}
	zoneForecasts := map[string][]ForecastPoint{
		"DE": {
			{Timestamp: t1, CI: 0.4},
			{Timestamp: t2, CI: 0.5},
		},
		"FR": {
			{Timestamp: t0, CI: 0.2},
			{Timestamp: t1, CI: 0.3},
			{Timestamp: t1, CI: 0.35},
		},
		"PL": {
			{Timestamp: t1, CI: 0.6},
			{Timestamp: t2, CI: 0.7},
		},
	}

	got := IntersectTimestamps(zones, zoneForecasts)
	want := []time.Time{t1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("IntersectTimestamps() = %#v, expected %#v", got, want)
	}
}

func TestBuildResampledIntersectionAlignsOffsetSeries(t *testing.T) {
	zones := []string{"DE", "FR"}
	de := []ForecastPoint{
		{Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), CI: 0.4},
		{Timestamp: time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC), CI: 0.5},
		{Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), CI: 0.6},
	}
	fr := []ForecastPoint{
		{Timestamp: time.Date(2026, 1, 1, 10, 5, 0, 0, time.UTC), CI: 0.3},
		{Timestamp: time.Date(2026, 1, 1, 11, 5, 0, 0, time.UTC), CI: 0.4},
		{Timestamp: time.Date(2026, 1, 1, 12, 5, 0, 0, time.UTC), CI: 0.5},
	}

	timeAxis, aligned := BuildResampledIntersection(
		zones,
		map[string][]ForecastPoint{
			"DE": de,
			"FR": fr,
		},
		time.Hour,
	)

	wantAxis := []time.Time{
		time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	if !reflect.DeepEqual(timeAxis, wantAxis) {
		t.Fatalf("BuildResampledIntersection() axis = %#v, expected %#v", timeAxis, wantAxis)
	}

	if len(aligned["DE"]) != 2 || len(aligned["FR"]) != 2 {
		t.Fatalf("unexpected aligned series lengths: DE=%d FR=%d", len(aligned["DE"]), len(aligned["FR"]))
	}

	if aligned["DE"][0].CI != 0.5 || aligned["DE"][1].CI != 0.6 {
		t.Fatalf("unexpected DE resampled CI values: %#v", aligned["DE"])
	}
	if aligned["FR"][0].CI != 0.3 || aligned["FR"][1].CI != 0.4 {
		t.Fatalf("unexpected FR resampled CI values: %#v", aligned["FR"])
	}
}

func TestInferResampleStepFromForecastData(t *testing.T) {
	zoneForecasts := map[string][]ForecastPoint{
		"DE": {
			{Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), CI: 0.4},
			{Timestamp: time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC), CI: 0.5},
		},
		"FR": {
			{Timestamp: time.Date(2026, 1, 1, 10, 30, 0, 0, time.UTC), CI: 0.3},
			{Timestamp: time.Date(2026, 1, 1, 11, 30, 0, 0, time.UTC), CI: 0.4},
		},
	}

	got := InferResampleStep(zoneForecasts)
	if got != time.Hour {
		t.Fatalf("InferResampleStep() = %s, expected %s", got, time.Hour)
	}
}
