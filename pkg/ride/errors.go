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
	ComplexityLimitExceed
)

type EvaluationError uint

type evaluationError struct {
	ErrorType       EvaluationError
	OriginalError   error
	CallStack       []string
	SpentComplexity int
}

func (e evaluationError) Error() string {
	return e.OriginalError.Error()
}

func (e EvaluationError) New(msg string) error {
	return evaluationError{ErrorType: e, OriginalError: errors.New(msg)}
}

func (e EvaluationError) Errorf(msg string, args ...interface{}) error {
	return evaluationError{ErrorType: e, OriginalError: errors.Errorf(msg, args...)}
}

func (e EvaluationError) Wrap(err error, msg string) error {
	return e.Wrapf(err, msg)
}

func (e EvaluationError) Wrapf(err error, msg string, args ...interface{}) error {
	return evaluationError{ErrorType: e, OriginalError: errors.Wrapf(err, msg, args...)}
}

func GetEvaluationErrorType(err error) EvaluationError {
	if ee, ok := err.(evaluationError); ok {
		return ee.ErrorType
	}
	return Undefined
}

func EvaluationErrorCallStack(err error) []string {
	if ee, ok := err.(evaluationError); ok {
		return ee.CallStack
	}
	return nil
}

func EvaluationErrorSpentComplexity(err error) int {
	if ee, ok := err.(evaluationError); ok {
		return ee.SpentComplexity
	}
	return 0
}

func EvaluationErrorPush(err error, format string, args ...interface{}) error {
	if ee, ok := err.(evaluationError); ok {
		ee.CallStack = append([]string{fmt.Sprintf(format, args...)}, ee.CallStack...)
		return ee
	}
	return errors.Wrapf(err, format, args...)
}

func EvaluationErrorSetComplexity(err error, complexity int) error {
	if ee, ok := err.(evaluationError); ok {
		ee.SpentComplexity = complexity
		return ee
	}
	return err
}
