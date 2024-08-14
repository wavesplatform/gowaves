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
	NegativeBalanceAfterPayment // special error type for unusual scala node behaviour
)

type EvaluationError uint

type evaluationError interface {
	error
	Unwrap() error
	ErrorType() EvaluationError
	CallStack() []string
	SpentComplexity() int

	SetComplexity(complexity int)
	PushCallStackf(format string, args ...interface{})
}

func newEvaluationError(t EvaluationError, err error) evaluationError {
	evErr := implEvaluationError{errorType: t, originalError: err}
	if t == NegativeBalanceAfterPayment { // wrap the error
		return &doubledComplexityImplEvaluationError{evErr}
	}
	return &evErr
}

type implEvaluationError struct {
	errorType        EvaluationError
	originalError    error
	reverseCallStack []string
	spentComplexity  int
}

func (e *implEvaluationError) Error() string { return e.originalError.Error() }

func (e *implEvaluationError) Unwrap() error { return e.originalError }

func (e *implEvaluationError) ErrorType() EvaluationError { return e.errorType }

func (e *implEvaluationError) CallStack() []string {
	callStack := make([]string, 0, len(e.reverseCallStack))
	for i := len(e.reverseCallStack) - 1; i >= 0; i-- {
		callStack = append(callStack, e.reverseCallStack[i])
	}
	return callStack
}

func (e *implEvaluationError) SpentComplexity() int { return e.spentComplexity }

func (e *implEvaluationError) SetComplexity(complexity int) { e.spentComplexity = complexity }

func (e *implEvaluationError) PushCallStackf(format string, args ...interface{}) {
	e.reverseCallStack = append(e.reverseCallStack, fmt.Sprintf(format, args...))
}

type doubledComplexityImplEvaluationError struct{ implEvaluationError }

func (e *doubledComplexityImplEvaluationError) SpentComplexity() int { return e.spentComplexity * 2 }

func (e EvaluationError) New(msg string) error {
	return newEvaluationError(e, errors.New(msg))
}

func (e EvaluationError) Errorf(msg string, args ...interface{}) error {
	return newEvaluationError(e, errors.Errorf(msg, args...))
}

func (e EvaluationError) Wrap(err error, msg string) error {
	return newEvaluationError(e, errors.Wrap(err, msg))
}

func (e EvaluationError) Wrapf(err error, msg string, args ...interface{}) error {
	return newEvaluationError(e, errors.Wrapf(err, msg, args...))
}

func GetEvaluationErrorType(err error) EvaluationError {
	var target evaluationError
	if errors.As(err, &target) {
		return target.ErrorType()
	}
	return Undefined
}

func EvaluationErrorCallStack(err error) []string {
	var target evaluationError
	if errors.As(err, &target) {
		return target.CallStack()
	}
	return nil
}

func EvaluationErrorSpentComplexity(err error) int {
	var target evaluationError
	if errors.As(err, &target) {
		return target.SpentComplexity()
	}
	return 0
}

func EvaluationErrorPushf(err error, format string, args ...interface{}) error {
	var target evaluationError
	if errors.As(err, &target) {
		target.PushCallStackf(format, args...) // change the internal error, wrapped hierarchy is not affected
		return err                             // return the original error with updated call stack
	}
	return errors.Wrapf(err, format, args...)
}

func EvaluationErrorSetComplexity(err error, complexity int) error {
	var target evaluationError
	if errors.As(err, &target) {
		target.SetComplexity(complexity) // change the internal error, wrapped hierarchy is not affected
		return err                       // return the original error with updated complexity
	}
	return err
}
