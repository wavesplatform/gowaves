package ride

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestEvaluationErrorAs(t *testing.T) {
	var (
		target evaluationError
		res    bool
	)
	err := errors.Wrap(error(evaluationError(&implEvaluationError{})), "test")
	assert.NotPanics(t, func() { res = errors.As(err, &target) })
	assert.True(t, res)
	// nil case
	nilErr := evaluationError(nil)
	assert.NotPanics(t, func() { res = errors.As(nilErr, &target) })
	assert.False(t, res)
}

func TestDoubledComplexityEvaluationError_SpentComplexity(t *testing.T) {
	tests := []struct {
		complexity int
		expected   int
	}{
		{complexity: 0, expected: 0},
		{complexity: 1, expected: 2},
		{complexity: 2, expected: 4},
	}
	for i, test := range tests {
		num := i + 1
		t.Run(fmt.Sprintf("raw_%d", num), func(t *testing.T) {
			var rawErr evaluationError = &doubledComplexityImplEvaluationError{implEvaluationError{
				spentComplexity: test.complexity,
			}}
			assert.Equal(t, test.expected, rawErr.SpentComplexity())
			assert.Equal(t, test.expected, EvaluationErrorSpentComplexity(rawErr))
		})
		t.Run(fmt.Sprintf("wrapped_%d", num), func(t *testing.T) {
			var err error = new(doubledComplexityImplEvaluationError)
			err = errors.Wrap(err, "test")
			err = EvaluationErrorSetComplexity(err, test.complexity)
			assert.Equal(t, test.expected, EvaluationErrorSpentComplexity(err))
		})
	}
}

func TestNewEvaluationError(t *testing.T) {
	tests := []struct {
		errT       EvaluationError
		complexity int
		expected   int
	}{
		{errT: Undefined, complexity: 21, expected: 21},
		{errT: UserError, complexity: 42, expected: 42},
		{errT: RuntimeError, complexity: 84, expected: 84},
		{errT: InternalInvocationError, complexity: 168, expected: 168},
		{errT: EvaluationFailure, complexity: 336, expected: 336},
		{errT: ComplexityLimitExceed, complexity: 672, expected: 672},
		{errT: NegativeBalanceAfterPayment, complexity: 1344, expected: 2688},
		{errT: NegativeBalanceAfterPayment, complexity: 1, expected: 2},
		{errT: NegativeBalanceAfterPayment, complexity: 0, expected: 0},
	}
	for i, test := range tests {
		num := i + 1
		origErr := errors.New("test")
		t.Run(fmt.Sprintf("raw_%d", num), func(t *testing.T) {
			rawErr := newEvaluationError(test.errT, origErr)
			rawErr.SetComplexity(test.complexity)
			assert.Equal(t, test.expected, rawErr.SpentComplexity())
			assert.Equal(t, test.errT, rawErr.ErrorType())
		})
		t.Run(fmt.Sprintf("wrapped_%d", num), func(t *testing.T) {
			err := errors.Wrap(newEvaluationError(test.errT, origErr), "test-test")
			err = EvaluationErrorSetComplexity(err, test.complexity)
			assert.Equal(t, test.expected, EvaluationErrorSpentComplexity(err))
			assert.Equal(t, test.errT, GetEvaluationErrorType(err))
		})
	}
}
