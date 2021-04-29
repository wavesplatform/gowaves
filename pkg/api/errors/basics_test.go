package errors

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestApiErrorWithSameIntID(t *testing.T) {
	type testSample struct {
		id           Identifier
		expectedName string
	}

	testData := []struct {
		first  testSample
		second testSample
	}{
		{
			testSample{InvalidNameErrorID, "InvalidNameError"},
			testSample{NegativeAmountErrorID, "NegativeAmountError"},
		},
		{
			testSample{StateCheckFailedErrorID, "StateCheckFailedError"},
			testSample{InsufficientFeeErrorID, "InsufficientFeeError"},
		},
		{
			testSample{ToSelfErrorID, "ToSelfError"},
			testSample{NegativeMinFeeErrorID, "NegativeMinFeeError"},
		},
		{
			testSample{MissingSenderPrivateKeyErrorID, "MissingSenderPrivateKeyError"},
			testSample{NonPositiveAmountErrorID, "NonPositiveAmountError"},
		},
	}

	for _, sample := range testData {
		assert.Equal(t, sample.first.id.IntCode(), sample.second.id.IntCode())
		assert.Equal(t, sample.first.expectedName, errorNames[sample.first.id])
		assert.Equal(t, sample.second.expectedName, errorNames[sample.second.id])
	}
}
