package utilities

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeErrorMessage(errMsg string, args ...any) string {
	if len(args) > 0 {
		for i := range args {
			msg := fmt.Sprintf("%v", args[i])
			errMsg += " " + msg
		}
	}
	return errMsg
}

func StatusCodesCheck(t *testing.T, goCode, scalaCode int, b ConsideredTransaction, args ...any) {
	errMsg := makeErrorMessage("Response code mismatch", args...)
	assert.Equalf(t, goCode, b.Resp.ResponseGo.StatusCode, "Node Go: "+errMsg)
	if b.Resp.ResponseScala != nil {
		assert.Equalf(t, scalaCode, b.Resp.ResponseScala.StatusCode, "Node Scala: "+errMsg)
	}
}

func TxInfoCheck(t *testing.T, errGo, errScala error, args ...any) {
	errMsg := makeErrorMessage("Failed to get TransactionInfo in testcase: ", args...)
	assert.NoErrorf(t, errGo, "Node Go: "+errMsg)
	assert.NoErrorf(t, errScala, "Node Scala: "+errMsg)
}

func WavesDiffBalanceCheck(t *testing.T, expected, actualGo, actualScala int64, args ...any) {
	errMsg := makeErrorMessage("Difference balance in Waves mismatch", args...)
	assert.Equalf(t, int(expected), int(actualGo), "Node Go: "+errMsg)
	assert.Equalf(t, int(expected), int(actualScala), "Node Scala: "+errMsg)
}

func AssetDiffBalanceCheck(t *testing.T, expected, actualGo, actualScala int64, args ...any) {
	errMsg := makeErrorMessage("Asset balance mismatch", args...)
	assert.Equalf(t, int(expected), int(actualGo), "Node Go: "+errMsg)
	assert.Equalf(t, int(expected), int(actualScala), "Node Scala: "+errMsg)
}

func AddressByAliasCheck(t *testing.T, expected, actualGo, actualScala []byte, args ...any) {
	errMsg := makeErrorMessage("Address mismatch alias", args...)
	assert.Equalf(t, expected, actualGo, "Node Go: "+errMsg)
	assert.Equalf(t, expected, actualScala, "Node Scala"+errMsg)
}

func AssetScriptCheck(t *testing.T, expected, actualGo, actualScala []byte, args ...any) {
	errMsg := makeErrorMessage("Script bytes mismatch", args...)
	assert.Equalf(t, expected, actualGo, "Node Go: "+errMsg)
	assert.Equalf(t, expected, actualScala, "Node Scala: "+errMsg)
}

func AssetNameCheck(t *testing.T, expected, actualGo, actualScala string, args ...any) {
	errMsg := makeErrorMessage("Asset Name mismatch", args...)
	assert.Equalf(t, expected, actualGo, "Node Go: "+errMsg)
	assert.Equalf(t, expected, actualScala, "Node Scala: "+errMsg)
}

func AssetDescriptionCheck(t *testing.T, expected, actualGo, actualScala string, args ...any) {
	errMsg := makeErrorMessage("Asset Description mismatch", args...)
	assert.Equalf(t, expected, actualGo, "Node Go: "+errMsg)
	assert.Equalf(t, expected, actualScala, "Node Scala: "+errMsg)
}

func ErrorMessageCheck(t *testing.T, expectedErrGo, expectedErrScala string, actualErrGo,
	actualErrScala error, args ...any) {
	errMsg := makeErrorMessage("Error message mismatch", args...)
	assert.ErrorContainsf(t, actualErrGo, expectedErrGo, "Node Go: "+errMsg)
	assert.ErrorContainsf(t, actualErrScala, expectedErrScala, "Node Scala: "+errMsg)
}

func MinersSumDiffBalanceInWavesCheck(t *testing.T, expected int64, actualGo, actualScala int64,
	args ...any) {
	errMsg := makeErrorMessage("Miners Sum Diff Balance mismatch", args...)
	assert.Equalf(t, int(expected), int(actualGo), "Node Go: "+errMsg)
	assert.Equalf(t, int(expected), int(actualScala), "Node Scala: "+errMsg)
}

func DaoDiffBalanceInWavesCheck(t *testing.T, expected int64, actualGo, actualScala int64,
	args ...any) {
	errMsg := makeErrorMessage("Dao Diff Balance mismatch", args...)
	assert.Equalf(t, int(expected), int(actualGo), "Node Go: "+errMsg)
	assert.Equalf(t, int(expected), int(actualScala), "Node Scala: "+errMsg)
}

func XtnBuyBackDiffBalanceInWavesCheck(t *testing.T, expected int64, actualGo, actualScala int64,
	args ...any) {
	errMsg := makeErrorMessage("Xtn buy back Diff Balance mismatch", args...)
	assert.Equalf(t, int(expected), int(actualGo), "Node Go: "+errMsg)
	assert.Equalf(t, int(expected), int(actualScala), "Node Scala: "+errMsg)
}

func TermCheck(t *testing.T, expected uint64, actualGo, actualScala uint64, args ...any) {
	errMsg := makeErrorMessage("Terms are mismatch", args...)
	assert.Equalf(t, int(expected), int(actualGo), "Node Go: "+errMsg)
	assert.Equalf(t, int(expected), int(actualScala), "Node Scala: "+errMsg)
}

func VotingIntervalStartCheck(t *testing.T, expected uint64, actualGo, actualScala uint64, args ...any) {
	errMsg := makeErrorMessage("VotingIntervalStart parameters are mismatch", args...)
	assert.Equalf(t, int(expected), int(actualGo), "Node Go: "+errMsg)
	assert.Equalf(t, int(expected), int(actualScala), "Node Scala: "+errMsg)
}

func NextCheckParameterCheck(t *testing.T, expected uint64, actualGo, actualScala uint64, args ...any) {
	errMsg := makeErrorMessage("NextChecks are mismatch", args...)
	assert.Equalf(t, int(expected), int(actualGo), "Node Go: "+errMsg)
	assert.Equalf(t, int(expected), int(actualScala), "Node Scala: "+errMsg)
}
