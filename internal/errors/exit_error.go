package errors

import (
	stderrors "errors"
	"fmt"
)

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(err error, code int) error {
	if err == nil {
		return nil
	}
	return &ExitError{Code: code, Err: err}
}

func Newf(code int, format string, args ...any) error {
	return &ExitError{
		Code: code,
		Err:  fmt.Errorf(format, args...),
	}
}

func GetCode(err error) int {
	var exitErr *ExitError
	if stderrors.As(err, &exitErr) {
		return exitErr.Code
	}
	return InputError
}
