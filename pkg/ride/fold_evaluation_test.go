package ride

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

func evaluateFold(t *testing.T, code string, expectedError string) {
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := serialization.Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	env := newTestEnv(t).withComplexityLimit(ast.LibV5, 26000).toEnv()
	_, err = CallVerifier(env, tree)
	require.Error(t, err)
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
	expectedError := "failed to evaluate block after declaration of function 'sum': failed to call system function '0': failed to materialize argument 1: failed to find system function '450'" //nolint:lll
	evaluateFold(t, code, expectedError)
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
	expectedError := "failed to evaluate block after declaration of function 'sum': failed to call system function '0': failed to materialize argument 1: failed to find system function '450'" //nolint:lll
	evaluateFold(t, code, expectedError)
}
