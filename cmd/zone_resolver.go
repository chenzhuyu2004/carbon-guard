package cmd

import (
	"fmt"
	"os"
	"strings"
)

const (
	zoneModeStrict   = "strict"
	zoneModeFallback = "fallback"
	zoneModeAuto     = "auto"

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

	if mode == zoneModeFallback || mode == zoneModeAuto {
		envZone := normalizeZone(os.Getenv(envZoneDefault))
		if envZone != "" {
			return resolvedZone{
				Zone:       envZone,
				Source:     "env",
				Confidence: "medium",
			}, nil
		}
	}

	if mode == zoneModeAuto {
		if auto, ok := resolveAutoZone(); ok {
			return auto, nil
		}
	}

	return resolvedZone{}, fmt.Errorf("zone is required (set --zone or %s)", envZoneDefault)
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

	if mode == zoneModeFallback || mode == zoneModeAuto {
		envZones := splitZones(os.Getenv(envZonesDefault))
		if len(envZones) > 0 {
			return resolvedZones{
				Zones:      envZones,
				Source:     "env",
				Confidence: "medium",
			}, nil
		}
	}

	if mode == zoneModeAuto {
		if auto, ok := resolveAutoZone(); ok {
			return resolvedZones{
				Zones:      []string{auto.Zone},
				Source:     auto.Source,
				Confidence: auto.Confidence,
			}, nil
		}
	}

	return resolvedZones{}, fmt.Errorf("zones is required (set --zones or %s)", envZonesDefault)
}

func normalizeZoneMode(mode string) (string, error) {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = zoneModeFallback
	}
	switch mode {
	case zoneModeStrict, zoneModeFallback, zoneModeAuto:
		return mode, nil
	default:
		return "", fmt.Errorf("zone-mode must be strict, fallback, or auto")
	}
}

func normalizeZone(zone string) string {
	return strings.ToUpper(strings.TrimSpace(zone))
}

func resolveAutoZone() (resolvedZone, bool) {
	if country, source, ok := detectCountryHint(); ok {
		zone := countryToZone(country)
		if zone != "" {
			return resolvedZone{
				Zone:       zone,
				Source:     source,
				Confidence: "low",
			}, true
		}
	}
	return resolvedZone{}, false
}

func detectCountryHint() (string, string, bool) {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if country, ok := countryFromLocale(os.Getenv(key)); ok {
			return country, "auto:locale", true
		}
	}
	if country, ok := countryFromTZ(os.Getenv("TZ")); ok {
		return country, "auto:tz", true
	}
	return "", "", false
}

func countryFromLocale(locale string) (string, bool) {
	value := strings.TrimSpace(locale)
	if value == "" {
		return "", false
	}
	if idx := strings.Index(value, "."); idx >= 0 {
		value = value[:idx]
	}
	if idx := strings.Index(value, "@"); idx >= 0 {
		value = value[:idx]
	}
	separators := []string{"_", "-"}
	for _, sep := range separators {
		parts := strings.Split(value, sep)
		if len(parts) < 2 {
			continue
		}
		country := strings.ToUpper(strings.TrimSpace(parts[len(parts)-1]))
		if isAlpha2(country) {
			return country, true
		}
	}
	if isAlpha2(strings.ToUpper(value)) {
		return strings.ToUpper(value), true
	}
	return "", false
}

func countryFromTZ(tz string) (string, bool) {
	normalized := strings.TrimSpace(tz)
	if normalized == "" {
		return "", false
	}
	country, ok := tzCountryHints[normalized]
	return country, ok
}

func isAlpha2(value string) bool {
	if len(value) != 2 {
		return false
	}
	for _, ch := range value {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}
	return true
}

func countryToZone(country string) string {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "UK":
		return "GB"
	case "US":
		return "US-NY"
	case "CA":
		return "CA-ON"
	case "AU":
		return "AU-NSW"
	default:
		if isAlpha2(strings.ToUpper(strings.TrimSpace(country))) {
			return strings.ToUpper(strings.TrimSpace(country))
		}
		return ""
	}
}

var tzCountryHints = map[string]string{
	"Europe/Berlin":       "DE",
	"Europe/Paris":        "FR",
	"Europe/Warsaw":       "PL",
	"Europe/London":       "GB",
	"Europe/Madrid":       "ES",
	"Europe/Rome":         "IT",
	"Europe/Amsterdam":    "NL",
	"Europe/Brussels":     "BE",
	"Europe/Stockholm":    "SE",
	"Europe/Oslo":         "NO",
	"Europe/Zurich":       "CH",
	"America/New_York":    "US",
	"America/Chicago":     "US",
	"America/Denver":      "US",
	"America/Los_Angeles": "US",
	"Asia/Shanghai":       "CN",
	"Asia/Chongqing":      "CN",
	"Asia/Beijing":        "CN",
	"Asia/Singapore":      "SG",
	"Asia/Tokyo":          "JP",
	"Asia/Seoul":          "KR",
	"Asia/Kolkata":        "IN",
	"Australia/Sydney":    "AU",
	"Australia/Melbourne": "AU",
}
