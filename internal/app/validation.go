package app

import (
	"fmt"
	"strings"
	"time"
)

const (
	maxDurationSeconds = 24 * 60 * 60
	maxLookaheadHours  = 7 * 24
	maxZonesCount      = 64
	maxWaitDuration    = 7 * 24 * time.Hour
)

func validateDurationSeconds(duration int) error {
	if duration <= 0 {
		return fmt.Errorf("%w: duration must be > 0", ErrInput)
	}
	if duration > maxDurationSeconds {
		return fmt.Errorf("%w: duration must be <= %d seconds", ErrInput, maxDurationSeconds)
	}
	return nil
}

func validateLookaheadHours(lookahead int) error {
	if lookahead <= 0 {
		return fmt.Errorf("%w: lookahead must be > 0", ErrInput)
	}
	if lookahead > maxLookaheadHours {
		return fmt.Errorf("%w: lookahead must be <= %d hours", ErrInput, maxLookaheadHours)
	}
	return nil
}

func validateDurationWithinLookahead(duration int, lookahead int) error {
	if duration > lookahead*3600 {
		return fmt.Errorf("%w: duration %ds exceeds lookahead window %ds", ErrInput, duration, lookahead*3600)
	}
	return nil
}

func validateZones(zones []string) error {
	if len(zones) == 0 {
		return fmt.Errorf("%w: zones is required", ErrInput)
	}
	if len(zones) > maxZonesCount {
		return fmt.Errorf("%w: zones count must be <= %d", ErrInput, maxZonesCount)
	}
	for _, zone := range zones {
		if zone == "" {
			return fmt.Errorf("%w: zone must not be empty", ErrInput)
		}
	}
	return nil
}

func validateMaxWait(maxWait time.Duration) error {
	if maxWait <= 0 {
		return fmt.Errorf("%w: max-wait must be > 0", ErrInput)
	}
	if maxWait > maxWaitDuration {
		return fmt.Errorf("%w: max-wait must be <= %s", ErrInput, maxWaitDuration)
	}
	return nil
}

func validateWaitCost(waitCost float64) error {
	if waitCost < 0 {
		return fmt.Errorf("%w: wait-cost must be >= 0", ErrInput)
	}
	return nil
}

func validateNoRegretConfig(maxDelay time.Duration, minReductionPct float64) error {
	if maxDelay < 0 {
		return fmt.Errorf("%w: max-delay-for-gain must be >= 0", ErrInput)
	}
	if minReductionPct < 0 {
		return fmt.Errorf("%w: min-reduction-for-wait must be >= 0", ErrInput)
	}
	return nil
}

func validateResampleConfig(fillMode string, maxFillAge time.Duration) error {
	mode := strings.TrimSpace(strings.ToLower(fillMode))
	if mode != "" && mode != "forward" && mode != "strict" {
		return fmt.Errorf("%w: resample-fill must be one of forward|strict", ErrInput)
	}
	if maxFillAge < 0 {
		return fmt.Errorf("%w: resample-max-fill-age must be >= 0", ErrInput)
	}
	return nil
}
