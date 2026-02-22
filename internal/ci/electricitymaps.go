package ci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const defaultElectricityMapsLatestURL = "https://api.electricitymaps.com/v3/carbon-intensity/latest"
const defaultElectricityMapsForecastURL = "https://api.electricitymaps.com/v3/carbon-intensity/forecast"

var electricityMapsHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

var electricityMapsLatestURL = defaultElectricityMapsLatestURL
var electricityMapsForecastURL = defaultElectricityMapsForecastURL

type ElectricityMapsProvider struct {
	APIKey string
}

type HTTPStatusError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPStatusError) Error() string {
	if e == nil {
		return "http status error"
	}
	return fmt.Sprintf("electricity maps api status: %s (%s)", e.Status, e.Body)
}

func (p *ElectricityMapsProvider) GetCurrentCI(ctx context.Context, zone string) (float64, error) {
	if p.APIKey == "" {
		return 0, fmt.Errorf("missing ELECTRICITY_MAPS_API_KEY: set an Electricity Maps API key to use live carbon data")
	}
	if zone == "" {
		return 0, fmt.Errorf("missing electricity maps zone")
	}

	endpoint, err := url.Parse(electricityMapsLatestURL)
	if err != nil {
		return 0, fmt.Errorf("build electricity maps url: %w", err)
	}

	query := endpoint.Query()
	query.Set("zone", zone)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("create electricity maps request: %w", err)
	}
	req.Header.Set("auth-token", p.APIKey)

	resp, err := electricityMapsHTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call electricity maps api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       readErrorBody(resp.Body),
		}
	}

	var body struct {
		CarbonIntensity float64 `json:"carbonIntensity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("decode electricity maps response: %w", err)
	}
	if body.CarbonIntensity <= 0 {
		return 0, fmt.Errorf("invalid carbonIntensity value: %v", body.CarbonIntensity)
	}

	return body.CarbonIntensity / 1000.0, nil
}

func (p *ElectricityMapsProvider) GetForecastCI(ctx context.Context, zone string, hours int) ([]ForecastPoint, error) {
	if p.APIKey == "" {
		return nil, fmt.Errorf("missing ELECTRICITY_MAPS_API_KEY: set an Electricity Maps API key to use forecast carbon data")
	}
	if zone == "" {
		return nil, fmt.Errorf("missing electricity maps zone")
	}
	if hours <= 0 {
		return nil, fmt.Errorf("hours must be > 0")
	}

	endpoint, err := url.Parse(electricityMapsForecastURL)
	if err != nil {
		return nil, fmt.Errorf("build electricity maps forecast url: %w", err)
	}

	query := endpoint.Query()
	query.Set("zone", zone)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create electricity maps forecast request: %w", err)
	}
	req.Header.Set("auth-token", p.APIKey)

	resp, err := electricityMapsHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call electricity maps forecast api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       readErrorBody(resp.Body),
		}
	}

	var body struct {
		Forecast []struct {
			Datetime        string  `json:"datetime"`
			CarbonIntensity float64 `json:"carbonIntensity"`
		} `json:"forecast"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode electricity maps forecast response: %w", err)
	}

	now := time.Now().UTC()
	limit := now.Add(time.Duration(hours) * time.Hour)
	points := make([]ForecastPoint, 0, len(body.Forecast))

	for _, item := range body.Forecast {
		if item.CarbonIntensity <= 0 {
			return nil, fmt.Errorf("invalid forecast carbonIntensity value: %v", item.CarbonIntensity)
		}

		timestamp, err := parseForecastTime(item.Datetime)
		if err != nil {
			return nil, err
		}
		timestamp = timestamp.UTC()
		if timestamp.Before(now) || timestamp.After(limit) {
			continue
		}

		points = append(points, ForecastPoint{
			Timestamp: timestamp,
			CI:        item.CarbonIntensity / 1000.0,
		})
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	return points, nil
}

func readErrorBody(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil {
		return "unable to read response body"
	}

	text := strings.TrimSpace(string(data))
	if text == "" {
		return "empty response body"
	}
	return text
}

func parseForecastTime(value string) (time.Time, error) {
	timestamp, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return timestamp, nil
	}

	timestamp, err = time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return timestamp, nil
	}

	return time.Time{}, fmt.Errorf("invalid forecast datetime: %s", value)
}
