package cmd

import (
	"reflect"
	"testing"
)

func clearZoneHintEnv(t *testing.T) {
	t.Helper()
	t.Setenv(envZoneDefault, "")
	t.Setenv(envZonesDefault, "")
	t.Setenv(envZoneHint, "")
	t.Setenv(envCountryHint, "")
	t.Setenv(envTimezoneHint, "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "")
	t.Setenv(envTimezoneSystem, "")
}

func TestResolveZoneStrictRequiresExplicit(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv(envZoneDefault, "DE")

	_, err := resolveZone("", zoneModeStrict, "FR", autoHints{})
	if err == nil {
		t.Fatalf("expected error in strict mode without explicit zone")
	}
}

func TestResolveZonePriorityCLIThenEnvThenConfig(t *testing.T) {
	t.Run("cli overrides env and config", func(t *testing.T) {
		clearZoneHintEnv(t)
		t.Setenv(envZoneDefault, "FR")
		got, err := resolveZone("de", zoneModeFallback, "PL", autoHints{})
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "DE" || got.Source != "cli" || got.FallbackUsed {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})

	t.Run("env overrides config", func(t *testing.T) {
		clearZoneHintEnv(t)
		t.Setenv(envZoneDefault, " us-ny ")
		got, err := resolveZone("", zoneModeFallback, "FR", autoHints{})
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "US-NY" || got.Source != "env" || !got.FallbackUsed {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})

	t.Run("config used when env is empty", func(t *testing.T) {
		clearZoneHintEnv(t)
		got, err := resolveZone("", zoneModeFallback, "uk", autoHints{})
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "GB" || got.Source != "config" || !got.FallbackUsed {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})
}

func TestResolveZoneAutoHintPriority(t *testing.T) {
	t.Run("zone hint from defaults", func(t *testing.T) {
		clearZoneHintEnv(t)
		t.Setenv("LANG", "de_DE.UTF-8")
		got, err := resolveZone("", zoneModeAuto, "", autoHints{ZoneHint: "FR"})
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "FR" || got.Source != "auto:zone-hint" {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})

	t.Run("country hint when no zone hint", func(t *testing.T) {
		clearZoneHintEnv(t)
		got, err := resolveZone("", zoneModeAuto, "", autoHints{CountryHint: "de"})
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "DE" || got.Source != "auto:country-hint" {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})

	t.Run("timezone hint when no zone/country hint", func(t *testing.T) {
		clearZoneHintEnv(t)
		got, err := resolveZone("", zoneModeAuto, "", autoHints{TimezoneHint: "Europe/Paris"})
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "FR" || got.Source != "auto:timezone-hint" {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})

	t.Run("env hints used when defaults empty", func(t *testing.T) {
		clearZoneHintEnv(t)
		t.Setenv(envZoneHint, "CA-ON")
		got, err := resolveZone("", zoneModeAuto, "", autoHints{})
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "CA-ON" || got.Source != "auto:zone-hint" {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})
}

func TestResolveZoneAutoFallback(t *testing.T) {
	t.Run("locale", func(t *testing.T) {
		clearZoneHintEnv(t)
		t.Setenv("LANG", "de_DE.UTF-8")

		got, err := resolveZone("", zoneModeAuto, "", autoHints{})
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "DE" || got.Source != "auto:locale" || got.Reason == "" || !got.FallbackUsed {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})

	t.Run("timezone", func(t *testing.T) {
		clearZoneHintEnv(t)
		t.Setenv(envTimezoneSystem, "Europe/Paris")

		got, err := resolveZone("", zoneModeAuto, "", autoHints{})
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "FR" || got.Source != "auto:tz" {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})

	t.Run("unsupported locale country requires explicit hint", func(t *testing.T) {
		clearZoneHintEnv(t)
		t.Setenv("LANG", "en_US.UTF-8")

		_, err := resolveZone("", zoneModeAuto, "", autoHints{})
		if err == nil {
			t.Fatalf("expected error for unsupported locale-only inference")
		}
	})
}

func TestResolveZoneValidation(t *testing.T) {
	clearZoneHintEnv(t)

	_, err := resolveZone("bad zone", zoneModeFallback, "", autoHints{})
	if err == nil {
		t.Fatalf("expected invalid zone format error")
	}

	t.Setenv(envZoneDefault, "bad zone")
	_, err = resolveZone("", zoneModeFallback, "", autoHints{})
	if err == nil {
		t.Fatalf("expected invalid env zone error")
	}

	_, err = resolveZone("", zoneModeAuto, "", autoHints{ZoneHint: "bad zone"})
	if err == nil {
		t.Fatalf("expected invalid zone hint error")
	}

	_, err = resolveZone("", zoneModeAuto, "", autoHints{CountryHint: "bad"})
	if err == nil {
		t.Fatalf("expected invalid country hint error")
	}

	_, err = resolveZone("", zoneModeAuto, "", autoHints{CountryHint: "US"})
	if err == nil {
		t.Fatalf("expected unsupported country hint error")
	}

	_, err = resolveZone("", zoneModeAuto, "", autoHints{TimezoneHint: "Mars/Base"})
	if err == nil {
		t.Fatalf("expected invalid timezone hint error")
	}
}

func TestResolveZoneInvalidMode(t *testing.T) {
	_, err := resolveZone("DE", "invalid", "", autoHints{})
	if err == nil {
		t.Fatalf("expected error for invalid zone mode")
	}
}

func TestResolveZonesStrictRequiresExplicit(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv(envZonesDefault, "DE,FR")

	_, err := resolveZones("", zoneModeStrict, "PL", autoHints{})
	if err == nil {
		t.Fatalf("expected error in strict mode without explicit zones")
	}
}

func TestResolveZonesPriorityAndNormalization(t *testing.T) {
	t.Run("cli with dedupe and alias normalization", func(t *testing.T) {
		clearZoneHintEnv(t)
		got, err := resolveZones(" de, FR,uk,DE ", zoneModeFallback, "", autoHints{})
		if err != nil {
			t.Fatalf("resolveZones() unexpected error: %v", err)
		}
		want := []string{"DE", "FR", "GB"}
		if !reflect.DeepEqual(got.Zones, want) || got.Source != "cli" || got.FallbackUsed {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})

	t.Run("env overrides config", func(t *testing.T) {
		clearZoneHintEnv(t)
		t.Setenv(envZonesDefault, "de,fr")
		got, err := resolveZones("", zoneModeFallback, "PL,IT", autoHints{})
		if err != nil {
			t.Fatalf("resolveZones() unexpected error: %v", err)
		}
		want := []string{"DE", "FR"}
		if !reflect.DeepEqual(got.Zones, want) || got.Source != "env" || !got.FallbackUsed {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})

	t.Run("config used when env is empty", func(t *testing.T) {
		clearZoneHintEnv(t)
		got, err := resolveZones("", zoneModeFallback, "pl,it", autoHints{})
		if err != nil {
			t.Fatalf("resolveZones() unexpected error: %v", err)
		}
		want := []string{"PL", "IT"}
		if !reflect.DeepEqual(got.Zones, want) || got.Source != "config" || !got.FallbackUsed {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})
}

func TestResolveZonesAutoFallback(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv("LANG", "fr_FR.UTF-8")

	got, err := resolveZones("", zoneModeAuto, "", autoHints{})
	if err != nil {
		t.Fatalf("resolveZones() unexpected error: %v", err)
	}
	want := []string{"FR"}
	if !reflect.DeepEqual(got.Zones, want) || got.Source != "auto:locale" || !got.FallbackUsed {
		t.Fatalf("unexpected resolution: %#v", got)
	}
}

func TestResolveZonesAutoHint(t *testing.T) {
	clearZoneHintEnv(t)
	got, err := resolveZones("", zoneModeAuto, "", autoHints{CountryHint: "DE"})
	if err != nil {
		t.Fatalf("resolveZones() unexpected error: %v", err)
	}
	want := []string{"DE"}
	if !reflect.DeepEqual(got.Zones, want) || got.Source != "auto:country-hint" {
		t.Fatalf("unexpected resolution: %#v", got)
	}
}

func TestResolveZonesValidation(t *testing.T) {
	clearZoneHintEnv(t)
	_, err := resolveZones("DE,bad zone", zoneModeFallback, "", autoHints{})
	if err == nil {
		t.Fatalf("expected invalid zone list error")
	}

	t.Setenv(envZonesDefault, "DE,bad zone")
	_, err = resolveZones("", zoneModeFallback, "", autoHints{})
	if err == nil {
		t.Fatalf("expected invalid env zone list error")
	}
}
