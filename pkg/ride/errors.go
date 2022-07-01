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
	errorType       EvaluationError
	originalError   error
	callStack       []string
	spentComplexity int
	complexities    []int
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

func EvaluationErrorReverseComplexitiesList(err error) []int {
	if ee, ok := err.(evaluationError); ok {
		return ee.complexities
	}
	return nil
}

func EvaluationErrorSpentComplexity(err error) int {
	if ee, ok := err.(evaluationError); ok {
		return ee.spentComplexity
	}
	return 0
}

func EvaluationErrorPush(err error, format string, args ...interface{}) error {
	if ee, ok := err.(evaluationError); ok {
		elem := fmt.Sprintf(format, args...)
		if cap(ee.callStack) > len(ee.callStack) { // reusing the same memory area
			ee.callStack = append(ee.callStack[:1], ee.callStack...)
			ee.callStack[0] = elem
		} else { // allocating memory
			ee.callStack = append([]string{elem}, ee.callStack...)
		}
		return ee
	}
	return errors.Wrapf(err, format, args...)
}

func EvaluationErrorPushComplexity(err error, complexity int) error {
	if ee, ok := err.(evaluationError); ok {
		ee.complexities = append(ee.complexities, complexity)
		ee.spentComplexity += complexity
		return ee
	}
	return err
}

func EvaluationErrorAddComplexity(err error, complexity int) error {
	if ee, ok := err.(evaluationError); ok {
		ee.spentComplexity += complexity
		return ee
	}
	return err
}
