package utils

import (
	"context"
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCounter_IncEachTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewCounter(ctx)
	require.NotNil(t, c)

	assert.Equal(t, 0, len(c.Get()))
	c.IncEachTransaction()
	assert.Equal(t, 1, len(c.Get()))
}

func TestCounter_IncUniqueTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewCounter(ctx)
	require.NotNil(t, c)

	assert.Equal(t, 0, len(c.Get()))
	c.IncUniqueTransaction()
	assert.Equal(t, 1, len(c.Get()))
}

func TestCounter_Clear(t *testing.T) {
	c := Counter{
		resendTransactionCount: map[string]Count{"a": {}, "b": {}, "c": {}},
	}

	c.clear(2)
	assert.Equal(t, 2, len(c.resendTransactionCount))
	assert.Equal(t, "a", c.Get()[0].Time)
}
