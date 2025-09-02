package utilities

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
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

func DataEntryAndKeyCheck(
	t *testing.T,
	expected *waves.DataEntry,
	actualGo, actualScala *waves.DataEntry,
	args ...any,
) {
	errMsg := makeErrorMessage("DataEntry parameters are mismatch", args...)
	assert.Equalf(t, expected.Key, actualGo.Key, "Node Go: "+errMsg)
	assert.Equalf(t, expected.Key, actualScala.Key, "Node Scala: "+errMsg)
	assert.Equalf(t, expected.Value, actualGo.Value, "Node Go: "+errMsg)
	assert.Equalf(t, expected.Value, actualScala.Value, "Node Scala: "+errMsg)
}

func dataEntrySliceSort(data []*waves.DataEntry) {
	slices.SortFunc(data, func(a, b *waves.DataEntry) int {
		return strings.Compare(a.Key, b.Key)
	})
}

func DataEntriesAndKeysCheck(t *testing.T, expected []*waves.DataEntry, actualGo, actualScala []*waves.DataEntry) {
	assert.Equalf(t, len(expected), len(actualScala), "Node Scala: Count of data entries is mismatch")
	assert.Equalf(t, len(expected), len(actualGo), "Node Go: Count of data entries is mismatch")
	dataEntrySliceSort(expected)
	dataEntrySliceSort(actualGo)
	dataEntrySliceSort(actualScala)
	for i, entry := range expected {
		assert.Equalf(t, entry.Key, actualGo[i].Key,
			"Node Go: Data entries keys %s %s are mismatch", entry.Key, actualGo[i].Key)
		assert.Equalf(t, entry.Key, actualScala[i].Key,
			"Node Scala: Data entries keys %s %s are mismatch", entry.Key, actualScala[i].Key)
		assert.Equalf(t, entry.Value, actualGo[i].Value,
			"Node Go: Data entries values %v %v are mismatch", entry.Value, actualGo[i].Value)
		assert.Equalf(t, entry.Value, actualScala[i].Value,
			"Node Scala: Data entries values %v %v are mismatch", entry.Value, actualScala[i].Value)
	}
}
func ApplicationStatusCheck(t *testing.T, expected, actualGo, actualScala string, args ...any) {
	errMsg := makeErrorMessage("Application status mismatch", args...)
	assert.Equalf(t, expected, actualGo, "Node Go: "+errMsg)
	assert.Equalf(t, expected, actualScala, "Node Scala: "+errMsg)
}
