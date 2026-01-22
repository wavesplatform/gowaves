package ride

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	ridec "github.com/wavesplatform/gowaves/pkg/ride/compiler"
)

func TestRideVersionsCoverage(t *testing.T) {
	// This test fails when a new Ride version is added
	// but this file has not been updated accordingly.
	//
	// Its purpose is to draw attention to other tests in this file
	// that must be reviewed and updated after introducing a new Ride version.
	covered := map[ast.LibraryVersion]struct{}{
		ast.LibV1: {},
		ast.LibV2: {},
		ast.LibV3: {},
		ast.LibV4: {},
		ast.LibV5: {},
		ast.LibV6: {},
		ast.LibV7: {},
		ast.LibV8: {},
		ast.LibV9: {},
	}

	for v := ast.LibV1; v <= ast.CurrentMaxLibraryVersion(); v++ {
		_, ok := covered[v]
		assert.True(t, ok,
			"New Ride version was added, but tests were not updated. "+
				"Please add the new version to this test and review other tests in the same file.",
		)
	}
}

func TestCheckMaxChainInvokeComplexityByVersion(t *testing.T) {
	_, err := MaxChainInvokeComplexityByVersion(ast.CurrentMaxLibraryVersion())
	assert.NoError(t, err, "Please add new Ride version to pkg/ride/constraints.go:21")
}

func TestCompilation(t *testing.T) {
	const script = `
		{-# STDLIB_VERSION %d #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		true`
	const dapp = `
		{-# STDLIB_VERSION %d #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		@Verifier(tx)
		func verify() = true`

	for name, test := range map[string]struct {
		minVersion ast.LibraryVersion
		template   string
	}{
		"script compilation": {ast.LibV1, script},
		"dApp compilation":   {ast.LibV4, dapp},
	} {
		t.Run(name, func(t *testing.T) {
			for v := test.minVersion; v <= ast.CurrentMaxLibraryVersion(); v++ {
				src := fmt.Sprintf(test.template, v)
				_, errs := ridec.CompileToTree(src)
				assert.Empty(t, errs,
					"Please update pkg/ride/compiler package to support new library version.")
			}
		})
	}
}

func TestEstimateTree(t *testing.T) {
	const maxEstimatorVersion = 4
	const script = `
		{-# STDLIB_VERSION %d #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		true`
	for ev := 1; ev <= maxEstimatorVersion; ev++ {
		for v := ast.LibV1; v <= ast.CurrentMaxLibraryVersion(); v++ {
			tree, errs := ridec.CompileToTree(fmt.Sprintf(script, v))
			require.Empty(t, errs)
			_, err := EstimateTree(tree, ev)
			assert.NoError(t, err, "Please, update estimation to support new library version")
		}
	}
}

func TestEvaluation(t *testing.T) {
	const script = `
		{-# STDLIB_VERSION %d #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		true`
	const dapp = `
		{-# STDLIB_VERSION %d #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		@Verifier(tx)
		func verify() = true`

	for name, test := range map[string]struct {
		minVersion ast.LibraryVersion
		template   string
	}{
		"script evaluation": {ast.LibV1, script},
		"dApp evaluation":   {ast.LibV4, dapp},
	} {
		t.Run(name, func(t *testing.T) {
			for v := test.minVersion; v <= ast.CurrentMaxLibraryVersion(); v++ {
				src := fmt.Sprintf(test.template, v)
				tree, errs := ridec.CompileToTree(src)
				require.Empty(t, errs)
				te := newTestEnv(t).withProtobufTx().withLibVersion(v).withComplexityLimit(2000).toEnv()
				res, err := CallVerifier(te, tree)
				assert.NoError(t, err, "Please, update evaluation to support new library version")
				require.NoError(t, err)
				assert.True(t, res.Result())
			}
		})
	}
}
