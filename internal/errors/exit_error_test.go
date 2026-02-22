package errors

import (
	stderrors "errors"
	"testing"
)

func TestNewNilReturnsNil(t *testing.T) {
	if err := New(nil, ProviderError); err != nil {
		t.Fatalf("New(nil, code) = %v, expected nil", err)
	}
}

func TestNewWrapsAndPreservesCode(t *testing.T) {
	base := stderrors.New("provider failed")
	err := New(base, ProviderError)
	if err == nil {
		t.Fatalf("New() returned nil")
	}

	if code := GetCode(err); code != ProviderError {
		t.Fatalf("GetCode(New(...)) = %d, expected %d", code, ProviderError)
	}

	if !stderrors.Is(err, base) {
		t.Fatalf("wrapped error should match base error")
	}
}

func TestNewfAndDefaultGetCode(t *testing.T) {
	err := Newf(MaxWaitExceeded, "waited %d minutes", 15)
	if err == nil {
		t.Fatalf("Newf() returned nil")
	}
	if got := err.Error(); got != "waited 15 minutes" {
		t.Fatalf("Newf().Error() = %q, expected %q", got, "waited 15 minutes")
	}
	if code := GetCode(err); code != MaxWaitExceeded {
		t.Fatalf("GetCode(Newf(...)) = %d, expected %d", code, MaxWaitExceeded)
	}

	plain := stderrors.New("plain error")
	if code := GetCode(plain); code != InputError {
		t.Fatalf("GetCode(plain error) = %d, expected %d", code, InputError)
	}
}
