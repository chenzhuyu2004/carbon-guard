package app

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInput               = errors.New("input error")
	ErrProvider            = errors.New("provider error")
	ErrNoValidWindow       = errors.New("no valid window")
	ErrMaxWaitExceeded     = errors.New("max wait exceeded")
	ErrMissedOptimalWindow = errors.New("missed optimal window")
	ErrTimeout             = errors.New("timeout")
)

func wrapProviderError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInput) ||
		errors.Is(err, ErrProvider) ||
		errors.Is(err, ErrNoValidWindow) ||
		errors.Is(err, ErrMaxWaitExceeded) ||
		errors.Is(err, ErrMissedOptimalWindow) ||
		errors.Is(err, ErrTimeout) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	}
	return fmt.Errorf("%w: %v", ErrProvider, err)
}
