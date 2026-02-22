package cmd

import (
	"flag"
	"fmt"
	"time"

	cgconfig "github.com/chenzhuyu2004/carbon-guard/internal/config"
)

func resolveSharedDefaults(args []string) (cgconfig.Shared, error) {
	configPath, _ := parseStringFlag(args, "config")
	return cgconfig.Resolve(configPath)
}

func addConfigFlag(fs *flag.FlagSet, defaultPath string) {
	fs.String("config", defaultPath, "path to JSON config file")
}

func addOutputFlag(fs *flag.FlagSet, defaultValue string) *string {
	return fs.String("output", defaultValue, "output format: text|json")
}

func addTimeoutFlag(fs *flag.FlagSet, defaultValue string) *string {
	return fs.String("timeout", defaultValue, "operation timeout")
}

func addCacheFlags(fs *flag.FlagSet, defaultDir string, defaultTTL string) (*string, *string) {
	cacheDirRaw := fs.String("cache-dir", defaultDir, "forecast cache directory")
	cacheTTLRaw := fs.String("cache-ttl", defaultTTL, "forecast cache TTL")
	return cacheDirRaw, cacheTTLRaw
}

func validateOutputMode(mode string) error {
	if mode != "text" && mode != "json" {
		return fmt.Errorf("output must be text or json")
	}
	return nil
}

func parseTimeout(timeoutRaw string) (time.Duration, error) {
	timeout, err := time.ParseDuration(timeoutRaw)
	if err != nil || timeout <= 0 {
		return 0, fmt.Errorf("invalid timeout duration")
	}
	return timeout, nil
}
