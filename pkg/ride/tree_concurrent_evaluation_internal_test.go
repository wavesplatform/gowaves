package ride

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

// TestConcurrentTreeEvaluation evaluates callables and the verifier of the same script tree from several
// goroutines. The state keeps parsed trees in a cache and hands the same nodes to every evaluation, so
// the evaluation must not modify the tree. The test is meaningful under the race detector.
func TestConcurrentTreeEvaluation(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 3 #-}
	   {-# CONTENT_TYPE DAPP #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}

	   func getPreviousAnswer(address: String) = {
	     address
	   }

	   @Callable(i)
	   func tellme(question: String) = {
	       let answer = getPreviousAnswer(question)

	       WriteSet([
	           DataEntry(answer + "_q", question),
	           DataEntry(answer + "_a", answer)
	           ])
	   }

	   @Callable(invocation)
	   func default() = {
	       let sender0 = invocation.caller.bytes
	       WriteSet([DataEntry("a", "b"), DataEntry("sender", sender0)])
	   }

	   @Verifier(tx)
	   func verify() = {
	       getPreviousAnswer(toString(tx.sender)) == "1"
	   }
	*/
	const (
		src = "AAIDAAAAAAAAAAAAAAABAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEAAAAHYWRkcmVzcwUAAAAHYWRkcmVzcwAAAAIAAAAB" +
			"aQEAAAAGdGVsbG1lAAAAAQAAAAhxdWVzdGlvbgQAAAAGYW5zd2VyCQEAAAARZ2V0UHJldmlvdXNBbnN3ZXIAAAABBQAAAAhxdWVz" +
			"dGlvbgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9x" +
			"BQAAAAhxdWVzdGlvbgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9hBQAAAAZhbnN3" +
			"ZXIFAAAAA25pbAAAAAppbnZvY2F0aW9uAQAAAAdkZWZhdWx0AAAAAAQAAAAHc2VuZGVyMAgIBQAAAAppbnZvY2F0aW9uAAAABmNh" +
			"bGxlcgAAAAVieXRlcwkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAABYQIAAAABYgkABEwA" +
			"AAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAGc2VuZGVyBQAAAAdzZW5kZXIwBQAAAANuaWwAAAABAAAAAnR4AQAAAAZ2ZXJpZnkA" +
			"AAAACQAAAAAAAAIJAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEJAAQlAAAAAQgFAAAAAnR4AAAABnNlbmRlcgIAAAABMcP91gY="
		goroutines = 8
		rounds     = 200
		// An environment is reused by all rounds of a goroutine and accumulates the spent complexity.
		complexityLimit = 100_000_000
	)
	_, tree := parseBase64Script(t, src)
	_, pristine := parseBase64Script(t, src) // A copy that is never evaluated.
	// In the state a tree is estimated before it gets to the cache, do the same with both copies.
	for _, tr := range []*ast.Tree{tree, pristine} {
		_, err := EstimateTree(tr, 3)
		require.NoError(t, err)
	}

	tellme := proto.NewFunctionCall("tellme", proto.Arguments{proto.NewStringArgument("question")})
	evaluate := func(callEnv, verifyEnv environment) ([]proto.ScriptAction, []proto.ScriptAction, bool, error) {
		// Like the scripts storage, give every evaluation its own tree header over the shared nodes.
		local := *tree
		def, err := CallFunction(callEnv, &local, proto.FunctionCall{})
		if err != nil {
			return nil, nil, false, err
		}
		named, err := CallFunction(callEnv, &local, tellme)
		if err != nil {
			return nil, nil, false, err
		}
		ver, err := CallVerifier(verifyEnv, &local)
		if err != nil {
			return nil, nil, false, err
		}
		return def.ScriptActions(), named.ScriptActions(), ver.Result(), nil
	}
	newEnvs := func() (environment, environment) {
		tx := byte_helpers.InvokeScriptWithProofs.Transaction.Clone()
		callEnv := newTestEnv(t).withLibVersion(ast.LibV3).withComplexityLimit(complexityLimit).
			withInvokeTransaction(tx).toEnv()
		verifyEnv := newTestEnv(t).withTransaction(tx).withLibVersion(ast.LibV3).
			withComplexityLimit(complexityLimit).toEnv()
		return callEnv, verifyEnv
	}

	callEnv, verifyEnv := newEnvs()
	expDefault, expNamed, expVerifier, err := evaluate(callEnv, verifyEnv)
	require.NoError(t, err)
	require.Len(t, expDefault, 2)
	require.Len(t, expNamed, 2)

	// Environments are created here, because the test helpers are not meant to be used from other goroutines.
	type envs struct{ call, verify environment }
	prepared := make([]envs, goroutines)
	for i := range prepared {
		prepared[i].call, prepared[i].verify = newEnvs()
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, e := range prepared {
		wg.Go(func() {
			<-start
			for range rounds {
				def, named, ver, evErr := evaluate(e.call, e.verify)
				if !assert.NoError(t, evErr) || !assert.Equal(t, expDefault, def) ||
					!assert.Equal(t, expNamed, named) || !assert.Equal(t, expVerifier, ver) {
					return
				}
			}
		})
	}
	close(start)
	wg.Wait()
	assert.Equal(t, pristine, tree, "evaluation must not modify the tree")
}
