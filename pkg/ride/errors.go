package ride

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	Undefined = EvaluationError(iota)
	UserError
	RuntimeError
	InternalInvocationError
	EvaluationFailure
)

type EvaluationError uint

type evaluationError struct {
	errorType     EvaluationError
	originalError error
	callStack     []string
}

func (e evaluationError) Error() string {
	return e.originalError.Error()
}

func (e EvaluationError) New(msg string) error {
	return evaluationError{errorType: e, originalError: errors.New(msg)}
}

func (e EvaluationError) Errorf(msg string, args ...interface{}) error {
	return evaluationError{errorType: e, originalError: errors.Errorf(msg, args...)}
}

func (e EvaluationError) Wrap(err error, msg string) error {
	return e.Wrapf(err, msg)
}

func (e EvaluationError) Wrapf(err error, msg string, args ...interface{}) error {
	return evaluationError{errorType: e, originalError: errors.Wrapf(err, msg, args...)}
}

func GetEvaluationErrorType(err error) EvaluationError {
	if ee, ok := err.(evaluationError); ok {
		return ee.errorType
	}
	return Undefined
}

func EvaluationErrorCallStack(err error) []string {
	if ee, ok := err.(evaluationError); ok {
		return ee.callStack
	}
	return nil
}

func EvaluationErrorPush(err error, format string, args ...interface{}) error {
	if ee, ok := err.(evaluationError); ok {
		ee.callStack = append([]string{fmt.Sprintf(format, args...)}, ee.callStack...)
		return ee
	}
	return errors.Wrapf(err, format, args...)
}
