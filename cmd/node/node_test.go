package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateParamsInvokeExecutionGoroutines(t *testing.T) {
	for _, num := range []int{0, 1, 8} {
		params, err := stateParams(&config{invokeExecutionGoroutines: num}, nil)
		require.NoError(t, err)
		assert.Equal(t, num, params.InvokeExecutionGoroutinesNum)
	}
	_, err := stateParams(&config{invokeExecutionGoroutines: -1}, nil)
	require.ErrorContains(t, err, "invoke-execution-goroutines-num")
}
