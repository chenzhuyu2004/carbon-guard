package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvConfigPath = "CARBON_GUARD_CONFIG"
	EnvCacheDir   = "CARBON_GUARD_CACHE_DIR"
	EnvCacheTTL   = "CARBON_GUARD_CACHE_TTL"
	EnvTimeout    = "CARBON_GUARD_TIMEOUT"
	EnvOutput     = "CARBON_GUARD_OUTPUT"
	EnvZone       = "CARBON_GUARD_ZONE"
	EnvZones      = "CARBON_GUARD_ZONES"
	EnvZoneMode   = "CARBON_GUARD_ZONE_MODE"
)

const (
	DefaultCacheDir = "~/.carbon-guard"
	DefaultCacheTTL = "10m"
	DefaultTimeout  = "30s"
	DefaultOutput   = "text"
	DefaultZone     = ""
	DefaultZones    = ""
	DefaultZoneMode = "fallback"
)

type Shared struct {
	ConfigPath string
	CacheDir   string
	CacheTTL   string
	Timeout    string
	Output     string
	Zone       string
	Zones      string
	ZoneMode   string
}

type fileConfig struct {
	CacheDir string `json:"cache_dir"`
	CacheTTL string `json:"cache_ttl"`
	Timeout  string `json:"timeout"`
	Output   string `json:"output"`
	Zone     string `json:"zone"`
	Zones    string `json:"zones"`
	ZoneMode string `json:"zone_mode"`
}

func Resolve(rawConfigPath string) (Shared, error) {
	cfg := Shared{
		ConfigPath: "",
		CacheDir:   DefaultCacheDir,
		CacheTTL:   DefaultCacheTTL,
		Timeout:    DefaultTimeout,
		Output:     DefaultOutput,
		Zone:       DefaultZone,
		Zones:      DefaultZones,
		ZoneMode:   DefaultZoneMode,
	}

	configPath := strings.TrimSpace(rawConfigPath)
	if configPath == "" {
		configPath = strings.TrimSpace(os.Getenv(EnvConfigPath))
	}

	if configPath != "" {
		expanded, err := expandHomeDir(configPath)
		if err != nil {
			return Shared{}, err
		}
		fileCfg, err := loadFileConfig(expanded)
		if err != nil {
			return Shared{}, err
		}
		cfg.ConfigPath = configPath
		if fileCfg.CacheDir != "" {
			cfg.CacheDir = fileCfg.CacheDir
		}
		if fileCfg.CacheTTL != "" {
			cfg.CacheTTL = fileCfg.CacheTTL
		}
		if fileCfg.Timeout != "" {
			cfg.Timeout = fileCfg.Timeout
		}
		if fileCfg.Output != "" {
			cfg.Output = fileCfg.Output
		}
		if fileCfg.Zone != "" {
			cfg.Zone = fileCfg.Zone
		}
		if fileCfg.Zones != "" {
			cfg.Zones = fileCfg.Zones
		}
		if fileCfg.ZoneMode != "" {
			cfg.ZoneMode = fileCfg.ZoneMode
		}
	}

	if v := strings.TrimSpace(os.Getenv(EnvCacheDir)); v != "" {
		cfg.CacheDir = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvCacheTTL)); v != "" {
		cfg.CacheTTL = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvTimeout)); v != "" {
		cfg.Timeout = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvOutput)); v != "" {
		cfg.Output = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvZone)); v != "" {
		cfg.Zone = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvZones)); v != "" {
		cfg.Zones = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvZoneMode)); v != "" {
		cfg.ZoneMode = v
	}

	return cfg, nil
}

func loadFileConfig(path string) (fileConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return fileConfig{}, fmt.Errorf("read config file %q: %w", path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	var cfg fileConfig
	if err := decoder.Decode(&cfg); err != nil {
		return fileConfig{}, fmt.Errorf("parse config file %q: %w", path, err)
	}
	return cfg, nil
}

func expandHomeDir(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
