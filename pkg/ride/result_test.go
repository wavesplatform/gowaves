package ride

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResult_Eq(t *testing.T) {
	require.True(t, ScriptResult{}.Eq(ScriptResult{}))
}
