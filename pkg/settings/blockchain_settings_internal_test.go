package settings

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockchainSettings(t *testing.T) {
	doTest := func(fileName string, bt BlockchainType) func(t *testing.T) {
		return func(t *testing.T) {
			expected, err := os.ReadFile(fileName)
			require.NoError(t, err)

			s := mustLoadEmbeddedSettings(bt) // intentionally checking must function
			actual, err := json.Marshal(s)
			require.NoError(t, err)

			assert.JSONEq(t, string(expected), string(actual))
		}
	}
	t.Run("stagenet", doTest(stagenetFile, StageNet))
	t.Run("testnet", doTest(testnetFile, TestNet))
	t.Run("mainnet", doTest(mainnetFile, MainNet))
}
