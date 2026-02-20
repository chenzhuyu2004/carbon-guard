package ci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"
)

const electricityMapsLatestURL = "https://api.electricitymaps.com/v3/carbon-intensity/latest"
const electricityMapsForecastURL = "https://api.electricitymaps.com/v3/carbon-intensity/forecast"

type ElectricityMapsProvider struct {
	APIKey string
}

func (p *ElectricityMapsProvider) GetCurrentCI(zone string) (float64, error) {
	if p.APIKey == "" {
		return 0, fmt.Errorf("missing electricity maps api key")
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

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("create electricity maps request: %w", err)
	}
	req.Header.Set("auth-token", p.APIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call electricity maps api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("electricity maps api status: %s", resp.Status)
	}

	var body struct {
		CarbonIntensity float64 `json:"carbonIntensity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("decode electricity maps response: %w", err)
	}

	return body.CarbonIntensity / 1000.0, nil
}

func (p *ElectricityMapsProvider) GetForecastCI(zone string, hours int) ([]ForecastPoint, error) {
	if p.APIKey == "" {
		return nil, fmt.Errorf("missing electricity maps api key")
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

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create electricity maps forecast request: %w", err)
	}
	req.Header.Set("auth-token", p.APIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call electricity maps forecast api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("electricity maps forecast api status: %s", resp.Status)
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
