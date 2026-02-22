package cmd

import (
	"flag"
	"fmt"
	"time"
)

func addOutputFlag(fs *flag.FlagSet) *string {
	return fs.String("output", "text", "output format: text|json")
}

func addTimeoutFlag(fs *flag.FlagSet) *string {
	return fs.String("timeout", "30s", "operation timeout")
}

func addCacheFlags(fs *flag.FlagSet) (*string, *string) {
	cacheDirRaw := fs.String("cache-dir", "~/.carbon-guard", "forecast cache directory")
	cacheTTLRaw := fs.String("cache-ttl", "10m", "forecast cache TTL")
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
