package ci

import (
	"errors"
	"fmt"
)

type ErrorKind string

const (
	ErrorKindAuth        ErrorKind = "auth"
	ErrorKindRateLimit   ErrorKind = "rate_limit"
	ErrorKindNetwork     ErrorKind = "network"
	ErrorKindUpstream    ErrorKind = "upstream"
	ErrorKindInvalidData ErrorKind = "invalid_data"
)

type ProviderError struct {
	Kind       ErrorKind
	Operation  string
	Zone       string
	StatusCode int
	Err        error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "provider error"
	}

	base := fmt.Sprintf("provider %s error", e.Kind)
	if e.Operation != "" {
		base = fmt.Sprintf("%s during %s", base, e.Operation)
	}
	if e.Zone != "" {
		base = fmt.Sprintf("%s for zone %s", base, e.Zone)
	}
	if e.StatusCode > 0 {
		base = fmt.Sprintf("%s (status %d)", base, e.StatusCode)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", base, e.Err)
	}
	return base
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewProviderError(kind ErrorKind, operation string, zone string, err error) error {
	if err == nil {
		return nil
	}
	return &ProviderError{
		Kind:      kind,
		Operation: operation,
		Zone:      zone,
		Err:       err,
	}
}

func NewProviderStatusError(kind ErrorKind, operation string, zone string, statusCode int, err error) error {
	if err == nil {
		return nil
	}
	return &ProviderError{
		Kind:       kind,
		Operation:  operation,
		Zone:       zone,
		StatusCode: statusCode,
		Err:        err,
	}
}

func IsKind(err error, kind ErrorKind) bool {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Kind == kind
	}
	return false
}
