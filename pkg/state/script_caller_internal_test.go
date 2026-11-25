package state

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

// assertEqualEvaluationErrors compares errors ignoring the stack traces captured at their creation.
func assertEqualEvaluationErrors(t *testing.T, expected, actual error) {
	t.Helper()
	if expected == nil {
		assert.NoError(t, actual)
		return
	}
	require.Error(t, actual)
	assert.Equal(t, expected.Error(), actual.Error())
	assert.Equal(t, ride.GetEvaluationErrorType(expected), ride.GetEvaluationErrorType(actual))
	assert.Equal(t, ride.EvaluationErrorSpentComplexity(expected), ride.EvaluationErrorSpentComplexity(actual))
}

func assertEqualInvokeEvaluations(t *testing.T, expected, actual invokeEvaluation) {
	t.Helper()
	assert.Equal(t, expected.result, actual.result)
	assert.Equal(t, expected.functionCall, actual.functionCall)
	assert.Equal(t, expected.spentComplexity, actual.spentComplexity)
	assert.Equal(t, expected.workaround, actual.workaround)
	assertEqualEvaluationErrors(t, expected.err, actual.err)
}

func TestEvaluateInvokeScriptWithProofsIsPure(t *testing.T) {
	const (
		depositAmount       = 34
		complexityBefore    = 12345 // Arbitrary non-zero value to detect any change of the counter.
		rideV5KeyValueCount = 99
	)
	for _, test := range []struct {
		name            string
		dApp            string
		rideV5Activated bool
		payments        proto.ScriptPayments
		fc              proto.FunctionCall
		fails           bool
	}{
		{
			name:     "successful call before RideV5",
			dApp:     "dapp.base64",
			payments: proto.ScriptPayments{{Amount: depositAmount}},
			fc:       proto.NewFunctionCall("deposit", proto.Arguments{}),
		},
		{
			name: "script error before RideV5",
			dApp: "dapp.base64",
			fc: proto.NewFunctionCall("withdraw",
				proto.Arguments{&proto.IntegerArgument{Value: depositAmount}}),
			fails: true,
		},
		{
			name:            "successful call over wrapped state after RideV5",
			dApp:            "ride5_recursive_invoke.base64",
			rideV5Activated: true,
			fc: proto.NewFunctionCall("keyvalue", proto.Arguments{
				&proto.IntegerArgument{Value: rideV5KeyValueCount},
				&proto.StringArgument{Value: strings.Repeat("0", 100)},
			}),
		},
		{
			name:            "script error over wrapped state after RideV5",
			dApp:            "ride5_recursive_invoke.base64",
			rideV5Activated: true,
			fc: proto.NewFunctionCall("keyvalue", proto.Arguments{
				&proto.IntegerArgument{Value: rideV5KeyValueCount},
				&proto.StringArgument{Value: strings.Repeat("0", 150)},
			}),
			fails: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			to := createInvokeApplierTestObjects(t)
			info := to.fallibleValidationParams(t)
			info.acceptFailed = test.rideV5Activated
			info.blockV5Activated = test.rideV5Activated
			info.rideV5Activated = test.rideV5Activated
			to.setDApp(t, test.dApp, testGlobal.recipientInfo)
			to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, depositAmount+invokeFee)

			sc := to.state.appender.ia.sc
			dApp := testGlobal.recipientInfo.addr
			tree, err := to.state.stor.scriptsStorage.newestScriptByAddr(dApp)
			require.NoError(t, err)
			tx := createInvokeScriptWithProofs(t, test.payments, test.fc, feeAsset, invokeFee)

			sc.recentTxComplexity = complexityBefore
			first, err := evaluateInvokeScriptWithProofs(to.state, sc.settings, tree, tx, info, dApp)
			require.NoError(t, err)
			second, err := evaluateInvokeScriptWithProofs(to.state, sc.settings, tree, tx, info, dApp)
			require.NoError(t, err)

			// Both evaluations give the same outcome and none of them accounts complexity.
			assertEqualInvokeEvaluations(t, first, second)
			assert.Equal(t, uint64(complexityBefore), sc.recentTxComplexity)
			assert.Zero(t, sc.totalComplexity)
			assert.Equal(t, test.fc, first.functionCall)
			assert.False(t, first.workaround)
			if test.fails {
				require.Error(t, first.err)
				assert.Nil(t, first.result)
				assert.Equal(t, ride.EvaluationErrorSpentComplexity(first.err), first.spentComplexity)
			} else {
				require.NoError(t, first.err)
				require.NotNil(t, first.result)
				assert.Equal(t, first.result.Complexity(), first.spentComplexity)
			}
			if test.rideV5Activated {
				assert.Positive(t, first.spentComplexity)
			}

			// Accounting of the evaluation is equivalent to the whole invokeFunction.
			sc.resetComplexity()
			accountedRes, accountedErr := sc.accountInvokeEvaluation(first, dApp, nil, info)
			accountedComplexity := sc.recentTxComplexity
			assert.Positive(t, accountedComplexity)
			if test.rideV5Activated {
				assert.Equal(t, uint64(first.spentComplexity), accountedComplexity)
			}

			sc.resetComplexity()
			res, invokeErr := sc.invokeFunction(tree, nil, tx, info, dApp)
			assert.Equal(t, res, accountedRes)
			assertEqualEvaluationErrors(t, invokeErr, accountedErr)
			assert.Equal(t, accountedComplexity, sc.recentTxComplexity)
			if test.fails {
				assertEqualEvaluationErrors(t, first.err, invokeErr)
			} else {
				require.NoError(t, invokeErr)
				assert.Equal(t, first.result, res)
			}
		})
	}
}

// TestInvokeFunctionWorkaround checks the invokes of MainNet that are handled without the evaluation of
// the script: the outcome and the accounted complexity are fixed.
func TestInvokeFunctionWorkaround(t *testing.T) {
	const workaroundComplexity = 16154
	for _, id := range []string{
		"DGXQ69rv3PbwVa6TeT7AQy2pyhpeCawNwGdWcC3wdfVh",
		"GqKtPzT4judzqSPxtLpzeoZpZcdULeW2rGtGLYADoqmj",
		"DAcwgX2UkJ1zWYPE3ABrQQtRGamWDdJwCherhoVzkvJP",
	} {
		t.Run(id, func(t *testing.T) {
			to := createInvokeApplierTestObjects(t)
			info := to.fallibleValidationParams(t)
			info.acceptFailed, info.blockV5Activated, info.rideV5Activated = true, true, true
			to.setDApp(t, "ride5_recursive_invoke.base64", testGlobal.recipientInfo)
			sc := to.state.appender.ia.sc
			require.Equal(t, proto.MainNetScheme, sc.settings.AddressSchemeCharacter)
			dApp := testGlobal.recipientInfo.addr
			tree, err := to.state.stor.scriptsStorage.newestScriptByAddr(dApp)
			require.NoError(t, err)
			// The function exists and succeeds, but the transaction with this ID is not evaluated.
			fc := proto.NewFunctionCall("keyvalue", proto.Arguments{
				&proto.IntegerArgument{Value: 1}, &proto.StringArgument{Value: "0"},
			})
			regular := createInvokeScriptWithProofs(t, nil, fc, feeAsset, invokeFee)
			ev, err := evaluateInvokeScriptWithProofs(to.state, sc.settings, tree, regular, info, dApp)
			require.NoError(t, err)
			require.NoError(t, ev.err)
			require.False(t, ev.workaround)

			tx := createInvokeScriptWithProofs(t, nil, fc, feeAsset, invokeFee)
			digest := crypto.MustDigestFromBase58(id)
			tx.ID = &digest
			ev, err = evaluateInvokeScriptWithProofs(to.state, sc.settings, tree, tx, info, dApp)
			require.NoError(t, err)
			assert.True(t, ev.workaround)
			assert.Nil(t, ev.result)
			assert.Equal(t, proto.FunctionCall{}, ev.functionCall)
			assert.Equal(t, workaroundComplexity, ev.spentComplexity)
			require.Error(t, ev.err)
			assert.Equal(t, ride.RuntimeError, ride.GetEvaluationErrorType(ev.err))
			assert.Equal(t, workaroundComplexity, ride.EvaluationErrorSpentComplexity(ev.err))

			sc.resetComplexity()
			res, invokeErr := sc.invokeFunction(tree, nil, tx, info, dApp)
			assert.Nil(t, res)
			assertEqualEvaluationErrors(t, ev.err, invokeErr)
			assert.Equal(t, uint64(workaroundComplexity), sc.recentTxComplexity)
		})
	}
}

func TestAccountInvokeEvaluationErrors(t *testing.T) {
	const spent = 100
	to := createInvokeApplierTestObjects(t)
	sc := to.state.appender.ia.sc
	// The dApp has no script, so before RideV5 the estimation based accounting fails.
	dApp := testGlobal.recipientInfo.addr
	fc := proto.NewFunctionCall("deposit", proto.Arguments{})
	evalErr := ride.EvaluationErrorSetComplexity(ride.RuntimeError.Errorf("evaluation failed"), spent)

	t.Run("after RideV5", func(t *testing.T) {
		info := to.fallibleValidationParams(t)
		info.rideV5Activated = true
		for _, workaround := range []bool{false, true} {
			sc.resetComplexity()
			ev := invokeEvaluation{functionCall: fc, err: evalErr, spentComplexity: spent, workaround: workaround}
			res, err := sc.accountInvokeEvaluation(ev, dApp, nil, info)
			assert.Nil(t, res)
			assert.Equal(t, evalErr, err) // The evaluation error is returned as is.
			assert.Equal(t, uint64(spent), sc.recentTxComplexity)
		}
	})
	t.Run("accounting failure before RideV5", func(t *testing.T) {
		info := to.fallibleValidationParams(t)
		const workaroundMsg = "failed to append function complexity for workaround invoke tx"
		for _, workaround := range []bool{false, true} {
			sc.resetComplexity()
			ev := invokeEvaluation{functionCall: fc, err: evalErr, spentComplexity: spent, workaround: workaround}
			res, err := sc.accountInvokeEvaluation(ev, dApp, nil, info)
			assert.Nil(t, res)
			// The accounting error replaces the evaluation error.
			require.Error(t, err)
			assert.NotEqual(t, evalErr, err)
			assert.Equal(t, workaround, strings.Contains(err.Error(), workaroundMsg))
			assert.Zero(t, sc.recentTxComplexity)
		}
	})
}
