package utils

import (
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCounter_IncEachTransaction(t *testing.T) {
	c := NewCounter()
	require.NotNil(t, c)

	assert.Equal(t, 0, len(c.Get()))
	c.IncEachTransaction()
	assert.Equal(t, 1, len(c.Get()))
}

func TestCounter_IncUniqueTransaction(t *testing.T) {
	c := NewCounter()
	require.NotNil(t, c)

	assert.Equal(t, 0, len(c.Get()))
	c.IncUniqueTransaction()
	assert.Equal(t, 1, len(c.Get()))
}
