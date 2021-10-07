package ride

import (
	"github.com/pkg/errors"
)

const (
	UserError = EvaluationError(iota)
	RuntimeError
	EvaluationFailure
)

type EvaluationError uint

type evaluationError struct {
	errorType     EvaluationError
	originalError error
	//TODO: Implement call stack like in Scala
}

func (e evaluationError) Error() string {
	return e.originalError.Error()
}

func (e EvaluationError) New(msg string) error {
	return evaluationError{errorType: e, originalError: errors.New(msg)}
}

func (e EvaluationError) Errorf(msg string, args ...interface{}) error {
	return evaluationError{errorType: e, originalError: errors.Errorf(msg, args)}
}

func (e EvaluationError) Wrap(err error, msg string) error {
	return e.Wrapf(err, msg)
}

func (e EvaluationError) Wrapf(err error, msg string, args ...interface{}) error {
	return evaluationError{errorType: e, originalError: errors.Wrapf(err, msg, args)}
}
