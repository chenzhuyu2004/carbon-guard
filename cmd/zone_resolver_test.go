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

	_, err := resolveZone("", zoneModeStrict)
	if err == nil {
		t.Fatalf("expected error in strict mode without explicit zone")
	}
}

func TestResolveZoneFallbackUsesEnv(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv(envZoneDefault, " us-ny ")

	got, err := resolveZone("", zoneModeFallback)
	if err != nil {
		t.Fatalf("resolveZone() unexpected error: %v", err)
	}
	if got.Zone != "US-NY" {
		t.Fatalf("zone = %q, expected %q", got.Zone, "US-NY")
	}
	if got.Source != "env" {
		t.Fatalf("source = %q, expected %q", got.Source, "env")
	}
	if got.Confidence != "medium" {
		t.Fatalf("confidence = %q, expected %q", got.Confidence, "medium")
	}
}

func TestResolveZoneExplicitOverridesEnv(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv(envZoneDefault, "FR")

	got, err := resolveZone("de", zoneModeFallback)
	if err != nil {
		t.Fatalf("resolveZone() unexpected error: %v", err)
	}
	if got.Zone != "DE" {
		t.Fatalf("zone = %q, expected %q", got.Zone, "DE")
	}
	if got.Source != "cli" {
		t.Fatalf("source = %q, expected %q", got.Source, "cli")
	}
	if got.Confidence != "high" {
		t.Fatalf("confidence = %q, expected %q", got.Confidence, "high")
	}
}

func TestResolveZoneInvalidMode(t *testing.T) {
	_, err := resolveZone("DE", "invalid")
	if err == nil {
		t.Fatalf("expected error for invalid zone mode")
	}
}

func TestResolveZoneAutoFromLocale(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv("LANG", "de_DE.UTF-8")

	got, err := resolveZone("", zoneModeAuto)
	if err != nil {
		t.Fatalf("resolveZone() unexpected error: %v", err)
	}
	if got.Zone != "DE" {
		t.Fatalf("zone = %q, expected %q", got.Zone, "DE")
	}
	if got.Source != "auto:locale" {
		t.Fatalf("source = %q, expected %q", got.Source, "auto:locale")
	}
	if got.Confidence != "low" {
		t.Fatalf("confidence = %q, expected %q", got.Confidence, "low")
	}
}

func TestResolveZoneAutoFromTZ(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv("TZ", "Europe/Paris")

	got, err := resolveZone("", zoneModeAuto)
	if err != nil {
		t.Fatalf("resolveZone() unexpected error: %v", err)
	}
	if got.Zone != "FR" {
		t.Fatalf("zone = %q, expected %q", got.Zone, "FR")
	}
	if got.Source != "auto:tz" {
		t.Fatalf("source = %q, expected %q", got.Source, "auto:tz")
	}
}

func TestResolveZonesStrictRequiresExplicit(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv(envZonesDefault, "DE,FR")

	_, err := resolveZones("", zoneModeStrict)
	if err == nil {
		t.Fatalf("expected error in strict mode without explicit zones")
	}
}

func TestResolveZonesFallbackUsesEnv(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv(envZonesDefault, " de, FR , ,us-ny ")

	got, err := resolveZones("", zoneModeFallback)
	if err != nil {
		t.Fatalf("resolveZones() unexpected error: %v", err)
	}
	want := []string{"DE", "FR", "US-NY"}
	if !reflect.DeepEqual(got.Zones, want) {
		t.Fatalf("zones = %#v, expected %#v", got.Zones, want)
	}
	if got.Source != "env" {
		t.Fatalf("source = %q, expected %q", got.Source, "env")
	}
	if got.Confidence != "medium" {
		t.Fatalf("confidence = %q, expected %q", got.Confidence, "medium")
	}
}

func TestResolveZonesExplicitOverridesEnv(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv(envZonesDefault, "DE,FR")

	got, err := resolveZones("pl,uk", zoneModeFallback)
	if err != nil {
		t.Fatalf("resolveZones() unexpected error: %v", err)
	}
	want := []string{"PL", "UK"}
	if !reflect.DeepEqual(got.Zones, want) {
		t.Fatalf("zones = %#v, expected %#v", got.Zones, want)
	}
	if got.Source != "cli" {
		t.Fatalf("source = %q, expected %q", got.Source, "cli")
	}
	if got.Confidence != "high" {
		t.Fatalf("confidence = %q, expected %q", got.Confidence, "high")
	}
}

func TestResolveZonesAutoFromLocale(t *testing.T) {
	clearZoneHintEnv(t)
	t.Setenv("LANG", "fr_FR.UTF-8")

	got, err := resolveZones("", zoneModeAuto)
	if err != nil {
		t.Fatalf("resolveZones() unexpected error: %v", err)
	}
	want := []string{"FR"}
	if !reflect.DeepEqual(got.Zones, want) {
		t.Fatalf("zones = %#v, expected %#v", got.Zones, want)
	}
	if got.Source != "auto:locale" {
		t.Fatalf("source = %q, expected %q", got.Source, "auto:locale")
	}
}
