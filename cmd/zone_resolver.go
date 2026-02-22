package cmd

import (
	"fmt"
	"os"
	"strings"
)

const (
	zoneModeStrict   = "strict"
	zoneModeFallback = "fallback"

	envZoneDefault  = "CARBON_GUARD_ZONE"
	envZonesDefault = "CARBON_GUARD_ZONES"
)

type resolvedZone struct {
	Zone       string
	Source     string
	Confidence string
}

type resolvedZones struct {
	Zones      []string
	Source     string
	Confidence string
}

func resolveZone(explicit string, mode string) (resolvedZone, error) {
	mode, err := normalizeZoneMode(mode)
	if err != nil {
		return resolvedZone{}, err
	}

	zone := normalizeZone(explicit)
	if zone != "" {
		return resolvedZone{
			Zone:       zone,
			Source:     "cli",
			Confidence: "high",
		}, nil
	}

	if mode == zoneModeFallback {
		envZone := normalizeZone(os.Getenv(envZoneDefault))
		if envZone != "" {
			return resolvedZone{
				Zone:       envZone,
				Source:     "env",
				Confidence: "medium",
			}, nil
		}
	}

	return resolvedZone{}, fmt.Errorf("zone is required")
}

func resolveZones(explicit string, mode string) (resolvedZones, error) {
	mode, err := normalizeZoneMode(mode)
	if err != nil {
		return resolvedZones{}, err
	}

	zones := splitZones(explicit)
	if len(zones) > 0 {
		return resolvedZones{
			Zones:      zones,
			Source:     "cli",
			Confidence: "high",
		}, nil
	}

	if mode == zoneModeFallback {
		envZones := splitZones(os.Getenv(envZonesDefault))
		if len(envZones) > 0 {
			return resolvedZones{
				Zones:      envZones,
				Source:     "env",
				Confidence: "medium",
			}, nil
		}
	}

	return resolvedZones{}, fmt.Errorf("zones is required")
}

func normalizeZoneMode(mode string) (string, error) {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = zoneModeFallback
	}
	switch mode {
	case zoneModeStrict, zoneModeFallback:
		return mode, nil
	default:
		return "", fmt.Errorf("zone-mode must be strict or fallback")
	}
}

func normalizeZone(zone string) string {
	return strings.ToUpper(strings.TrimSpace(zone))
}
