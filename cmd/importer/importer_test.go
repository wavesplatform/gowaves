package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvokeExecutionGoroutinesFlag(t *testing.T) {
	newCfg := func(num int) *cfg {
		return &cfg{blockchainPath: "blockchain", dataDirPath: "data", invokeExecutionGoroutines: num}
	}
	for _, num := range []int{0, 1, 8} {
		c := newCfg(num)
		require.NoError(t, c.validateFlags())
		assert.Equal(t, num, c.params(100).InvokeExecutionGoroutinesNum)
	}
	require.ErrorContains(t, newCfg(-1).validateFlags(), "invoke-execution-goroutines-num")
}
