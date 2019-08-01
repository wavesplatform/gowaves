package state_changed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFuncHandler_Handle(t *testing.T) {
	b := false
	h := NewFuncHandler(func() {
		b = true
	})
	h.Handle()
	require.True(t, b)
}
