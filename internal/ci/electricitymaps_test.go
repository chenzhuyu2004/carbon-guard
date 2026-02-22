package ci

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupElectricityMapsTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(handler)
	oldLatestURL := electricityMapsLatestURL
	oldForecastURL := electricityMapsForecastURL
	oldClient := electricityMapsHTTPClient

	electricityMapsLatestURL = srv.URL + "/latest"
	electricityMapsForecastURL = srv.URL + "/forecast"
	electricityMapsHTTPClient = srv.Client()

	t.Cleanup(func() {
		electricityMapsLatestURL = oldLatestURL
		electricityMapsForecastURL = oldForecastURL
		electricityMapsHTTPClient = oldClient
		srv.Close()
	})

	return srv
}

func TestGetCurrentCIConvertsToKg(t *testing.T) {
	setupElectricityMapsTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("zone"); got != "DE" {
			t.Fatalf("zone query = %q, expected %q", got, "DE")
		}
		if got := r.Header.Get("auth-token"); got != "test-key" {
			t.Fatalf("auth-token = %q, expected %q", got, "test-key")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"carbonIntensity": 400}`))
	})

	provider := &ElectricityMapsProvider{APIKey: "test-key"}
	got, err := provider.GetCurrentCI(context.Background(), "DE")
	if err != nil {
		t.Fatalf("GetCurrentCI() unexpected error: %v", err)
	}
	if math.Abs(got-0.4) > 1e-9 {
		t.Fatalf("GetCurrentCI() = %v, expected %v", got, 0.4)
	}
}

func TestGetCurrentCINon200IncludesBody(t *testing.T) {
	setupElectricityMapsTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
	})

	provider := &ElectricityMapsProvider{APIKey: "test-key"}
	_, err := provider.GetCurrentCI(context.Background(), "DE")
	if err == nil {
		t.Fatalf("expected error for non-200 status")
	}
	if !strings.Contains(err.Error(), "429") || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("error does not include status/body: %v", err)
	}
}

func TestGetForecastCIConvertsAndSorts(t *testing.T) {
	now := time.Now().UTC()
	insideWindow := now.Add(30 * time.Minute).Format(time.RFC3339)
	outsideWindow := now.Add(2 * time.Hour).Format(time.RFC3339)
	past := now.Add(-15 * time.Minute).Format(time.RFC3339)

	setupElectricityMapsTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/forecast" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("zone"); got != "DE" {
			t.Fatalf("zone query = %q, expected %q", got, "DE")
		}
		if got := r.Header.Get("auth-token"); got != "test-key" {
			t.Fatalf("auth-token = %q, expected %q", got, "test-key")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "forecast": [
		    {"datetime": "` + insideWindow + `", "carbonIntensity": 500},
		    {"datetime": "` + outsideWindow + `", "carbonIntensity": 300},
		    {"datetime": "` + past + `", "carbonIntensity": 600}
		  ]
		}`))
	})

	provider := &ElectricityMapsProvider{APIKey: "test-key"}
	points, err := provider.GetForecastCI(context.Background(), "DE", 1)
	if err != nil {
		t.Fatalf("GetForecastCI() unexpected error: %v", err)
	}
	if len(points) != 3 {
		t.Fatalf("GetForecastCI() returned %d points, expected 3", len(points))
	}
	if !points[0].Timestamp.Before(points[1].Timestamp) || !points[1].Timestamp.Before(points[2].Timestamp) {
		t.Fatalf("GetForecastCI() points are not sorted by timestamp")
	}
	if math.Abs(points[0].CI-0.6) > 1e-9 {
		t.Fatalf("GetForecastCI().CI[0] = %v, expected %v", points[0].CI, 0.6)
	}
	if math.Abs(points[1].CI-0.5) > 1e-9 {
		t.Fatalf("GetForecastCI().CI[1] = %v, expected %v", points[1].CI, 0.5)
	}
	if math.Abs(points[2].CI-0.3) > 1e-9 {
		t.Fatalf("GetForecastCI().CI[2] = %v, expected %v", points[2].CI, 0.3)
	}
}

func TestGetCurrentCIMissingAPIKey(t *testing.T) {
	provider := &ElectricityMapsProvider{}
	_, err := provider.GetCurrentCI(context.Background(), "DE")
	if err == nil {
		t.Fatalf("expected missing api key error")
	}
	if !strings.Contains(err.Error(), "ELECTRICITY_MAPS_API_KEY") {
		t.Fatalf("error does not mention ELECTRICITY_MAPS_API_KEY: %v", err)
	}
}
