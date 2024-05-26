package ride

import (
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
