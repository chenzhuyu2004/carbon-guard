package cmd

import (
	"reflect"
	"testing"
)

func clearZoneHintEnv(t *testing.T) {
	t.Helper()
	t.Setenv(envZoneDefault, "")
	t.Setenv(envZonesDefault, "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "")
	t.Setenv("TZ", "")
}

func TestResolveZoneStrictRequiresExplicit(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv(envZoneDefault, "DE")

	_, err := resolveZone("", zoneModeStrict, "FR")
	if err == nil {
		t.Fatalf("expected error in strict mode without explicit zone")
	}
}

func TestResolveZonePriorityCLIThenEnvThenConfig(t *testing.T) {
	t.Run("cli overrides env and config", func(t *testing.T) {
		clearZoneHintEnv(t)
		t.Setenv(envZoneDefault, "FR")
		got, err := resolveZone("de", zoneModeFallback, "PL")
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
		got, err := resolveZone("", zoneModeFallback, "FR")
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "US-NY" || got.Source != "env" || !got.FallbackUsed {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})

	t.Run("config used when env is empty", func(t *testing.T) {
		clearZoneHintEnv(t)
		got, err := resolveZone("", zoneModeFallback, "uk")
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "GB" || got.Source != "config" || !got.FallbackUsed {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})
}

func TestResolveZoneAutoFallback(t *testing.T) {
	t.Run("locale", func(t *testing.T) {
		clearZoneHintEnv(t)
		t.Setenv("LANG", "de_DE.UTF-8")

		got, err := resolveZone("", zoneModeAuto, "")
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "DE" || got.Source != "auto:locale" || got.Reason == "" || !got.FallbackUsed {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})

	t.Run("timezone", func(t *testing.T) {
		clearZoneHintEnv(t)
		t.Setenv("TZ", "Europe/Paris")

		got, err := resolveZone("", zoneModeAuto, "")
		if err != nil {
			t.Fatalf("resolveZone() unexpected error: %v", err)
		}
		if got.Zone != "FR" || got.Source != "auto:tz" {
			t.Fatalf("unexpected resolution: %#v", got)
		}
	})
}

func TestResolveZoneValidation(t *testing.T) {
	clearZoneHintEnv(t)

	_, err := resolveZone("bad zone", zoneModeFallback, "")
	if err == nil {
		t.Fatalf("expected invalid zone format error")
	}

	t.Setenv(envZoneDefault, "bad zone")
	_, err = resolveZone("", zoneModeFallback, "")
	if err == nil {
		t.Fatalf("expected invalid env zone error")
	}
}

func TestResolveZoneInvalidMode(t *testing.T) {
	_, err := resolveZone("DE", "invalid", "")
	if err == nil {
		t.Fatalf("expected error for invalid zone mode")
	}
}

func TestResolveZonesStrictRequiresExplicit(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv(envZonesDefault, "DE,FR")

	_, err := resolveZones("", zoneModeStrict, "PL")
	if err == nil {
		t.Fatalf("expected error in strict mode without explicit zones")
	}
}

func TestResolveZonesPriorityAndNormalization(t *testing.T) {
	t.Run("cli with dedupe and alias normalization", func(t *testing.T) {
		clearZoneHintEnv(t)
		got, err := resolveZones(" de, FR,uk,DE ", zoneModeFallback, "")
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
		got, err := resolveZones("", zoneModeFallback, "PL,IT")
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
		got, err := resolveZones("", zoneModeFallback, "pl,it")
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

	got, err := resolveZones("", zoneModeAuto, "")
	if err != nil {
		t.Fatalf("resolveZones() unexpected error: %v", err)
	}
	want := []string{"FR"}
	if !reflect.DeepEqual(got.Zones, want) || got.Source != "auto:locale" || !got.FallbackUsed {
		t.Fatalf("unexpected resolution: %#v", got)
	}
}

func TestResolveZonesValidation(t *testing.T) {
	clearZoneHintEnv(t)
	_, err := resolveZones("DE,bad zone", zoneModeFallback, "")
	if err == nil {
		t.Fatalf("expected invalid zone list error")
	}

	t.Setenv(envZonesDefault, "DE,bad zone")
	_, err = resolveZones("", zoneModeFallback, "")
	if err == nil {
		t.Fatalf("expected invalid env zone list error")
	}
}
