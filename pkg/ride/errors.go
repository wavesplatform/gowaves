package ride

import "github.com/pkg/errors"

type ThrowError struct {
	msg string
}

func NewThrowError(msg string) *ThrowError {
	return &ThrowError{msg: msg}
}

func (a *ThrowError) Error() string {
	return a.msg
}

func IsThrowErr(err error) bool {
	_, ok := errors.Cause(err).(*ThrowError)
	return ok
}
