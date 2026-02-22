package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	zoneModeStrict   = "strict"
	zoneModeFallback = "fallback"
	zoneModeAuto     = "auto"

	envZoneDefault    = "CARBON_GUARD_ZONE"
	envZonesDefault   = "CARBON_GUARD_ZONES"
	envZoneHint       = "CARBON_GUARD_ZONE_HINT"
	envCountryHint    = "CARBON_GUARD_COUNTRY_HINT"
	envTimezoneHint   = "CARBON_GUARD_TIMEZONE_HINT"
	envTimezoneSystem = "TZ"
)

var zonePattern = regexp.MustCompile(`^[A-Z]{2}(?:-[A-Z0-9]+)*$`)

type resolvedZone struct {
	Zone         string
	Source       string
	Confidence   string
	Reason       string
	FallbackUsed bool
}

type resolvedZones struct {
	Zones        []string
	Source       string
	Confidence   string
	Reason       string
	FallbackUsed bool
}

type autoHints struct {
	ZoneHint     string
	CountryHint  string
	TimezoneHint string
}

func resolveZone(explicit string, mode string, configZone string, hints autoHints) (resolvedZone, error) {
	mode, err := normalizeZoneMode(mode)
	if err != nil {
		return resolvedZone{}, err
	}

	if zone, ok, err := parseSingleZone(explicit); err != nil {
		return resolvedZone{}, err
	} else if ok {
		return resolvedZone{
			Zone:         zone,
			Source:       "cli",
			Confidence:   "high",
			Reason:       "provided by --zone",
			FallbackUsed: false,
		}, nil
	}

	if mode == zoneModeStrict {
		return resolvedZone{}, fmt.Errorf("zone is required in strict mode (set --zone)")
	}

	if raw := strings.TrimSpace(os.Getenv(envZoneDefault)); raw != "" {
		zone, ok, err := parseSingleZone(raw)
		if err != nil {
			return resolvedZone{}, fmt.Errorf("invalid %s: %w", envZoneDefault, err)
		}
		if ok {
			return resolvedZone{
				Zone:         zone,
				Source:       "env",
				Confidence:   "medium",
				Reason:       "from " + envZoneDefault,
				FallbackUsed: true,
			}, nil
		}
	}

	if raw := strings.TrimSpace(configZone); raw != "" {
		zone, ok, err := parseSingleZone(raw)
		if err != nil {
			return resolvedZone{}, fmt.Errorf("invalid config zone: %w", err)
		}
		if ok {
			return resolvedZone{
				Zone:         zone,
				Source:       "config",
				Confidence:   "medium",
				Reason:       "from config zone",
				FallbackUsed: true,
			}, nil
		}
	}

	if mode == zoneModeAuto {
		if auto, ok, err := resolveAutoZone(hints); err != nil {
			return resolvedZone{}, err
		} else if ok {
			return auto, nil
		}
	}

	if mode == zoneModeAuto {
		return resolvedZone{}, fmt.Errorf("zone is required (set --zone or %s or config zone; auto inference unavailable)", envZoneDefault)
	}
	return resolvedZone{}, fmt.Errorf("zone is required (set --zone or %s or config zone)", envZoneDefault)
}

func resolveZones(explicit string, mode string, configZones string, hints autoHints) (resolvedZones, error) {
	mode, err := normalizeZoneMode(mode)
	if err != nil {
		return resolvedZones{}, err
	}

	if zones, ok, err := parseZoneList(explicit); err != nil {
		return resolvedZones{}, err
	} else if ok {
		return resolvedZones{
			Zones:        zones,
			Source:       "cli",
			Confidence:   "high",
			Reason:       "provided by --zones",
			FallbackUsed: false,
		}, nil
	}

	if mode == zoneModeStrict {
		return resolvedZones{}, fmt.Errorf("zones are required in strict mode (set --zones)")
	}

	if raw := strings.TrimSpace(os.Getenv(envZonesDefault)); raw != "" {
		zones, ok, err := parseZoneList(raw)
		if err != nil {
			return resolvedZones{}, fmt.Errorf("invalid %s: %w", envZonesDefault, err)
		}
		if ok {
			return resolvedZones{
				Zones:        zones,
				Source:       "env",
				Confidence:   "medium",
				Reason:       "from " + envZonesDefault,
				FallbackUsed: true,
			}, nil
		}
	}

	if raw := strings.TrimSpace(configZones); raw != "" {
		zones, ok, err := parseZoneList(raw)
		if err != nil {
			return resolvedZones{}, fmt.Errorf("invalid config zones: %w", err)
		}
		if ok {
			return resolvedZones{
				Zones:        zones,
				Source:       "config",
				Confidence:   "medium",
				Reason:       "from config zones",
				FallbackUsed: true,
			}, nil
		}
	}

	if mode == zoneModeAuto {
		if auto, ok, err := resolveAutoZone(hints); err != nil {
			return resolvedZones{}, err
		} else if ok {
			return resolvedZones{
				Zones:        []string{auto.Zone},
				Source:       auto.Source,
				Confidence:   auto.Confidence,
				Reason:       auto.Reason,
				FallbackUsed: true,
			}, nil
		}
	}

	if mode == zoneModeAuto {
		return resolvedZones{}, fmt.Errorf("zones are required (set --zones or %s or config zones; auto inference unavailable)", envZonesDefault)
	}
	return resolvedZones{}, fmt.Errorf("zones are required (set --zones or %s or config zones)", envZonesDefault)
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

func parseSingleZone(raw string) (string, bool, error) {
	zone := normalizeZoneAlias(raw)
	if zone == "" {
		return "", false, nil
	}
	if !zonePattern.MatchString(zone) {
		return "", false, fmt.Errorf("invalid zone format %q", zone)
	}
	return zone, true, nil
}

func parseZoneList(raw string) ([]string, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false, nil
	}

	items := strings.Split(trimmed, ",")
	zones := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		zone := normalizeZoneAlias(item)
		if zone == "" {
			continue
		}
		if !zonePattern.MatchString(zone) {
			return nil, false, fmt.Errorf("invalid zone format %q", zone)
		}
		if _, ok := seen[zone]; ok {
			continue
		}
		seen[zone] = struct{}{}
		zones = append(zones, zone)
	}

	if len(zones) == 0 {
		return nil, false, nil
	}
	return zones, true, nil
}

func normalizeZoneAlias(value string) string {
	zone := strings.ToUpper(strings.TrimSpace(value))
	switch zone {
	case "UK":
		return "GB"
	default:
		return zone
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func resolveAutoZone(hints autoHints) (resolvedZone, bool, error) {
	if zoneRaw := firstNonEmpty(strings.TrimSpace(hints.ZoneHint), strings.TrimSpace(os.Getenv(envZoneHint))); zoneRaw != "" {
		zone, ok, err := parseSingleZone(zoneRaw)
		if err != nil {
			return resolvedZone{}, false, fmt.Errorf("invalid zone hint: %w", err)
		}
		if ok {
			return resolvedZone{
				Zone:         zone,
				Source:       "auto:zone-hint",
				Confidence:   "high",
				Reason:       "from zone hint",
				FallbackUsed: true,
			}, true, nil
		}
	}

	if countryRaw := firstNonEmpty(strings.TrimSpace(hints.CountryHint), strings.TrimSpace(os.Getenv(envCountryHint))); countryRaw != "" {
		country := strings.ToUpper(strings.TrimSpace(countryRaw))
		zone := countryToAutoZone(country)
		if zone == "" {
			return resolvedZone{}, false, fmt.Errorf("invalid country hint %q", countryRaw)
		}
		return resolvedZone{
			Zone:         zone,
			Source:       "auto:country-hint",
			Confidence:   "medium",
			Reason:       "from country hint",
			FallbackUsed: true,
		}, true, nil
	}

	if tzRaw := firstNonEmpty(strings.TrimSpace(hints.TimezoneHint), strings.TrimSpace(os.Getenv(envTimezoneHint))); tzRaw != "" {
		if country, ok := countryFromTZ(tzRaw); ok {
			zone := countryToAutoZone(country)
			if zone != "" {
				return resolvedZone{
					Zone:         zone,
					Source:       "auto:timezone-hint",
					Confidence:   "medium",
					Reason:       "from timezone hint",
					FallbackUsed: true,
				}, true, nil
			}
		}
		return resolvedZone{}, false, fmt.Errorf("invalid timezone hint %q", tzRaw)
	}

	if country, source, reason, ok := detectCountryHint(); ok {
		zone := countryToAutoZone(country)
		if zone != "" {
			return resolvedZone{
				Zone:         zone,
				Source:       source,
				Confidence:   "low",
				Reason:       reason,
				FallbackUsed: true,
			}, true, nil
		}
	}
	return resolvedZone{}, false, nil
}

func detectCountryHint() (string, string, string, bool) {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if country, ok := countryFromLocale(os.Getenv(key)); ok {
			return country, "auto:locale", "inferred from " + key, true
		}
	}
	if country, ok := countryFromTZ(os.Getenv(envTimezoneSystem)); ok {
		return country, "auto:tz", "inferred from " + envTimezoneSystem, true
	}
	return "", "", "", false
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
	for _, sep := range []string{"_", "-"} {
		parts := strings.Split(value, sep)
		if len(parts) < 2 {
			continue
		}
		country := strings.ToUpper(strings.TrimSpace(parts[len(parts)-1]))
		if isAlpha2(country) {
			return country, true
		}
	}
	value = strings.ToUpper(strings.TrimSpace(value))
	if isAlpha2(value) {
		return value, true
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

func countryToAutoZone(country string) string {
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
