package fride

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNames(t *testing.T) {
	assert.Equal(t, "!", functionNameV2(0))
	assert.Equal(t, "!=", functionNameV3(1))
	assert.Equal(t, "wavesBalance", functionNameV2(67))
	assert.Equal(t, "wavesBalance", functionNameV4(180))
}
