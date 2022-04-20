package ride

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

var (
	nativeFoldTestState = &MockSmartState{}
	nativeFoldTestEnv   = &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		validateInternalPaymentsFunc: func() bool {
			return false
		},
		stateFunc: func() types.SmartState {
			return nativeFoldTestState
		},
		libVersionFunc: func() int {
			return 5
		},
		rideV6ActivatedFunc: func() bool {
			return true
		},
	}
)

func evaluateFold(t *testing.T, code string) {
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	_, err = CallVerifier(nativeFoldTestEnv, tree)
	require.Error(t, err)
	foldId := "450"
	expectedError := EvaluationFailure.Errorf("failed to find system function '%s'", foldId).Error()
	require.EqualError(t, err, expectedError)
}

func TestNotExistNativeFoldString(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		func sum(a: String, b: Int) = "(" + a + "+" + toString(b) + ")"

		fold_20([1,2,3,4,5,6,7,8,9,10,11,12,13], "0", sum) == "(((((((((((((0+1)+2)+3)+4)+5)+6)+7)+8)+9)+10)+11)+12)+13)"
	*/
	code := "BgEKAQNzdW0CAWEBYgkArAICCQCsAgIJAKwCAgkArAICAgEoBQFhAgErCQCkAwEFAWICASkJAAACCQDCAwMJAMwIAgABCQDMCAIAAgkAzAgCAAMJAMwIAgAECQDMCAIABQkAzAgCAAYJAMwIAgAHCQDMCAIACAkAzAgCAAkJAMwIAgAKCQDMCAIACwkAzAgCAAwJAMwIAgANBQNuaWwCATACA3N1bQI5KCgoKCgoKCgoKCgoKDArMSkrMikrMykrNCkrNSkrNikrNykrOCkrOSkrMTApKzExKSsxMikrMTMpW4xQtQ=="
	evaluateFold(t, code)
}

func TestNotExistNativeFoldSum(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		func sum(a: Int, b: Int) = a + b

		fold_20([1,2,3,4,5,6,7,8,9,10,11,12,13], 0, sum) == 91
	*/
	code := "BgEKAQNzdW0CAWEBYgkAZAIFAWEFAWIJAAACCQDCAwMJAMwIAgABCQDMCAIAAgkAzAgCAAMJAMwIAgAECQDMCAIABQkAzAgCAAYJAMwIAgAHCQDMCAIACAkAzAgCAAkJAMwIAgAKCQDMCAIACwkAzAgCAAwJAMwIAgANBQNuaWwAAAIDc3VtAFtN86UP"
	evaluateFold(t, code)
}
