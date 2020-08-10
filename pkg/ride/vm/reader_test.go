package vm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReader_String(t *testing.T) {
	code := []byte{0, 1, 120}
	r := NewReader(code)
	require.Equal(t, "x", r.String())
}

func TestReader_Pos(t *testing.T) {
	code := []byte{0, 1, 120}
	r := NewReader(code)
	_ = r.String()
	require.Equal(t, 3, r.Pos())
}
