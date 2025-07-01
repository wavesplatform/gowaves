package utxpool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCleaner(t *testing.T) {
	require.NotNil(t, NewCleaner(nil, nil, nil))
}
