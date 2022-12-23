package utilities

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeErrorMessage(errMsg string, args ...interface{}) string {
	if len(args) > 0 {
		for i := 0; i < len(args); i++ {
			msg := fmt.Sprintf("%v", args[i])
			errMsg += " " + msg
		}
	}
	return errMsg
}

func StatusCodesCheck(t *testing.T, goCode, scalaCode int, b ConsideredTransaction, args ...interface{}) {
	errMsg := makeErrorMessage("Response code mismatch", args...)
	assert.Equalf(t, goCode, b.Resp.ResponseGo.Response.StatusCode, "Node Go: "+errMsg)
	if b.Resp.ResponseScala != nil {
		assert.Equalf(t, scalaCode, b.Resp.ResponseScala.Response.StatusCode, "Node Scala: "+errMsg)
	}
}

func TxInfoCheck(t *testing.T, errGo, errScala error, args ...interface{}) {
	errMsg := makeErrorMessage("Failed to get TransactionInfo in testcase: ", args...)
	assert.NoErrorf(t, errGo, "Node Go: "+errMsg)
	assert.NoErrorf(t, errScala, "Node Scala: "+errMsg)
}

func WavesDiffBalanceCheck(t *testing.T, expected, actualGo, actualScala int64, args ...interface{}) {
	errMsg := makeErrorMessage("Difference balance in Waves mismatch", args...)
	assert.Equalf(t, expected, actualGo, "Node Go: "+errMsg)
	assert.Equalf(t, expected, actualScala, "Node Scala: "+errMsg)
}

func AssetBalanceCheck(t *testing.T, expected, actualGo, actualScala int64, args ...interface{}) {
	errMsg := makeErrorMessage("Asset balance mismatch", args...)
	assert.Equalf(t, expected, actualGo, "Node Go: "+errMsg)
	assert.Equalf(t, expected, actualScala, "Node Scala: "+errMsg)
}

func AddressByAliasCheck(t *testing.T, expected, actualGo, actualScala []byte, args ...interface{}) {
	errMsg := makeErrorMessage("Address mismatch alias", args...)
	assert.Equalf(t, expected, actualGo, "Node Go: "+errMsg)
	assert.Equalf(t, expected, actualScala, "Node Scala"+errMsg)
}

func ErrorMessageCheck(t *testing.T, expectedErrGo, expectedErrScala string, actualErrGo,
	actualErrScala error, args ...interface{}) {
	errMsg := makeErrorMessage("Error message mismatch", args...)
	assert.ErrorContainsf(t, actualErrGo, expectedErrGo, "Node Go: "+errMsg)
	assert.ErrorContainsf(t, actualErrScala, expectedErrScala, "Node Scala: "+errMsg)
}
