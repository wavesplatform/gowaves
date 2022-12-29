package ride

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

func TestConstructorsDifferentVersions(t *testing.T) {
	sender := newTestAccount(t, "SENDER") // 3N8CkZAyS4XcDoJTJoKNuNk2xmNKmQj7myW
	dApp1 := newTestAccount(t, "DAPP1")   // 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz

	// {-# STDLIB_VERSION 3 #-}
	// {-# CONTENT_TYPE DAPP #-}
	// {-# SCRIPT_TYPE ACCOUNT #-}

	// @Callable(i)
	// func f() = {
	// 	let invV3 = Invocation(unit, i.caller, i.callerPublicKey, i.transactionId, 500, i.feeAssetId)
	// 	WriteSet([DataEntry("fee", invV3.fee)])
	// }
	srcV3 := "AAIDAAAAAAAAAAQIARIAAAAAAAAAAAEAAAABaQEAAAABZgAAAAAEAAAABWludlYzCQEAAAAKSW52b2NhdGlvbgAAAAYFAAAABHVuaXQIBQAAAAFpAAAABmNhbGxlcggFAAAAAWkAAAAPY2FsbGVyUHVibGljS2V5CAUAAAABaQAAAA10cmFuc2FjdGlvbklkAAAAAAAAAAH0CAUAAAABaQAAAApmZWVBc3NldElkCQEAAAAIV3JpdGVTZXQAAAABCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACAgAAAANmZWUIBQAAAAVpbnZWMwAAAANmZWUFAAAAA25pbAAAAACtXWn7"
	entryV3 := &proto.IntegerDataEntry{Key: "fee", Value: 500}

	// {-# STDLIB_VERSION 4 #-}
	// {-# CONTENT_TYPE DAPP #-}
	// {-# SCRIPT_TYPE ACCOUNT #-}

	// @Callable(i)
	// func f() = {
	// 	let invV4 = Invocation([AttachedPayment(unit, 100)], i.caller, i.callerPublicKey, i.transactionId, 500, i.feeAssetId)
	// 	[IntegerEntry("fee", invV4.fee)]
	// }
	srcV4 := "AAIEAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAABZgAAAAAEAAAABWludlY0CQEAAAAKSW52b2NhdGlvbgAAAAYJAARMAAAAAgkBAAAAD0F0dGFjaGVkUGF5bWVudAAAAAIFAAAABHVuaXQAAAAAAAAAAGQFAAAAA25pbAgFAAAAAWkAAAAGY2FsbGVyCAUAAAABaQAAAA9jYWxsZXJQdWJsaWNLZXkIBQAAAAFpAAAADXRyYW5zYWN0aW9uSWQAAAAAAAAAAfQIBQAAAAFpAAAACmZlZUFzc2V0SWQJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAICAAAAA2ZlZQgFAAAABWludlY0AAAAA2ZlZQUAAAADbmlsAAAAAPhB07Y="
	entryV4 := &proto.IntegerDataEntry{Key: "fee", Value: 500}

	// {-# STDLIB_VERSION 5 #-}
	// {-# CONTENT_TYPE DAPP #-}
	// {-# SCRIPT_TYPE ACCOUNT #-}

	// @Callable(i)
	// func f() = {
	// 	let invV5 = Invocation([AttachedPayment(unit, 100)], i.caller, i.callerPublicKey, i.transactionId, i.fee, i.feeAssetId, i.originCaller, i.originCallerPublicKey)
	// 	[BinaryEntry("originCaller", invV5.originCaller.bytes)]
	// }
	srcV5 := "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAABZgAAAAAEAAAABWludlY1CQEAAAAKSW52b2NhdGlvbgAAAAgJAARMAAAAAgkBAAAAD0F0dGFjaGVkUGF5bWVudAAAAAIFAAAABHVuaXQAAAAAAAAAAGQFAAAAA25pbAgFAAAAAWkAAAAGY2FsbGVyCAUAAAABaQAAAA9jYWxsZXJQdWJsaWNLZXkIBQAAAAFpAAAADXRyYW5zYWN0aW9uSWQIBQAAAAFpAAAAA2ZlZQgFAAAAAWkAAAAKZmVlQXNzZXRJZAgFAAAAAWkAAAAMb3JpZ2luQ2FsbGVyCAUAAAABaQAAABVvcmlnaW5DYWxsZXJQdWJsaWNLZXkJAARMAAAAAgkBAAAAC0JpbmFyeUVudHJ5AAAAAgIAAAAMb3JpZ2luQ2FsbGVyCAgFAAAABWludlY1AAAADG9yaWdpbkNhbGxlcgAAAAVieXRlcwUAAAADbmlsAAAAAOPaTJU="
	entryV5 := &proto.BinaryDataEntry{Key: "originCaller", Value: sender.address().Bytes()}

	check := func(src string, dataEntry proto.DataEntry) {
		_, tree := parseBase64Script(t, src)

		env := newTestEnv(t).withLibVersion(ast.LibV6).withComplexityLimit(ast.LibV6, 52000).withRideV6Activated().
			withSender(sender).withThis(dApp1).withDApp(dApp1).withTree(dApp1, tree).
			withInvocation("test")

		res, err := CallFunction(env.toEnv(), tree, "f", proto.Arguments{})
		require.NoError(t, err)
		require.Equal(t, 1, len(res.ScriptActions()))

		dataAction, ok := res.ScriptActions()[0].(*proto.DataEntryScriptAction)
		require.Truef(t, ok, "expected DataEntryScriptAction, got: %T", res.ScriptActions()[0])

		require.Equal(t, dataAction.Entry, dataEntry)
	}

	check(srcV3, entryV3)
	check(srcV4, entryV4)
	check(srcV5, entryV5)
}
