package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDefaults(t *testing.T) {
	t.Setenv(EnvConfigPath, "")
	t.Setenv(EnvCacheDir, "")
	t.Setenv(EnvCacheTTL, "")
	t.Setenv(EnvTimeout, "")
	t.Setenv(EnvOutput, "")

	got, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}

	if got.CacheDir != DefaultCacheDir {
		t.Fatalf("CacheDir = %q, expected %q", got.CacheDir, DefaultCacheDir)
	}
	if got.CacheTTL != DefaultCacheTTL {
		t.Fatalf("CacheTTL = %q, expected %q", got.CacheTTL, DefaultCacheTTL)
	}
	if got.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %q, expected %q", got.Timeout, DefaultTimeout)
	}
	if got.Output != DefaultOutput {
		t.Fatalf("Output = %q, expected %q", got.Output, DefaultOutput)
	}
}

func TestResolveConfigAndEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "carbon-guard.json")
	content := `{
  "cache_dir": "/tmp/from-file",
  "cache_ttl": "15m",
  "timeout": "45s",
  "output": "json"
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	t.Setenv(EnvConfigPath, path)
	t.Setenv(EnvCacheTTL, "20m")
	t.Setenv(EnvOutput, "text")

	got, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}

	if got.ConfigPath != path {
		t.Fatalf("ConfigPath = %q, expected %q", got.ConfigPath, path)
	}
	if got.CacheDir != "/tmp/from-file" {
		t.Fatalf("CacheDir = %q, expected %q", got.CacheDir, "/tmp/from-file")
	}
	if got.CacheTTL != "20m" {
		t.Fatalf("CacheTTL = %q, expected %q", got.CacheTTL, "20m")
	}
	if got.Timeout != "45s" {
		t.Fatalf("Timeout = %q, expected %q", got.Timeout, "45s")
	}
	if got.Output != "text" {
		t.Fatalf("Output = %q, expected %q", got.Output, "text")
	}
}

func TestResolveExplicitConfigPathBeatsEnvPath(t *testing.T) {
	dir := t.TempDir()

	fileA := filepath.Join(dir, "a.json")
	if err := os.WriteFile(fileA, []byte(`{"cache_ttl":"11m"}`), 0o644); err != nil {
		t.Fatalf("WriteFile(a) error: %v", err)
	}

	fileB := filepath.Join(dir, "b.json")
	if err := os.WriteFile(fileB, []byte(`{"cache_ttl":"33m"}`), 0o644); err != nil {
		t.Fatalf("WriteFile(b) error: %v", err)
	}

	t.Setenv(EnvConfigPath, fileA)
	got, err := Resolve(fileB)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}
	if got.ConfigPath != fileB {
		t.Fatalf("ConfigPath = %q, expected %q", got.ConfigPath, fileB)
	}
	if got.CacheTTL != "33m" {
		t.Fatalf("CacheTTL = %q, expected %q", got.CacheTTL, "33m")
	}
}
