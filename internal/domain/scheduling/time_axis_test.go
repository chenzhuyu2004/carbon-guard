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
