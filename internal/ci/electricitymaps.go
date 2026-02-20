package ci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const electricityMapsLatestURL = "https://api.electricitymap.org/v3/carbon-intensity/latest"

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

	resp, err := http.DefaultClient.Do(req)
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
