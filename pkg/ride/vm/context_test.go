package vm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewContext(t *testing.T) {
	require.NotNil(t, NewContext(nil, nil, 'W'))
}
