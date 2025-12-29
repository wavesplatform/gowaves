package ride

import (
	"encoding/base64"
	stderrs "errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	ridec "github.com/wavesplatform/gowaves/pkg/ride/compiler"
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

		env := newTestEnv(t).withLibVersion(ast.LibV6).withComplexityLimit(52000).withRideV6Activated().
			withSender(sender).withThis(dApp1).withDApp(dApp1).withTree(dApp1, tree).
			withInvocation("test")

		res, err := CallFunction(env.toEnv(), tree, proto.NewFunctionCall("f", proto.Arguments{}))
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

func mustField[T rideType](t *testing.T, obj rideType, field string) T {
	val, err := obj.get(field)
	require.NoError(t, err)
	var zeroT T
	casted, ok := val.(T)
	require.Truef(t, ok, "field '%s' is not of expected type %T, but %T", field, zeroT, val)
	return casted
}

type optsProvenPart struct {
	txVarName      string
	emptyBodyBytes bool
	checkProofs    bool
}

func provenPart(
	t *testing.T,
	env reducedReadOnlyEnv,
	tx proto.Transaction,
	opts ...func(part *optsProvenPart),
) string {
	ops := &optsProvenPart{"t", false, true}
	for _, o := range opts {
		o(ops)
	}
	obj, err := transactionToObject(env, tx)
	require.NoError(t, err)

	rideProofs, err := obj.get(proofsField)
	if err != nil {
		msg := fmt.Sprintf("type '%s' has no property '%s'", obj.instanceOf(), proofsField)
		require.EqualError(t, err, msg)
		rideProofs = slices.Repeat(rideList{rideByteVector(nil)}, rideProofsCount)
	}
	proofsArr, ok := rideProofs.(rideList)
	require.True(t, ok)
	require.Equal(t, rideProofsCount, len(proofsArr))

	bodyBytesCheck := fmt.Sprintf("%s.bodyBytes.size() == 0", ops.txVarName)
	if !ops.emptyBodyBytes {
		body, bErr := proto.MarshalTxBody(env.scheme(), tx)
		require.NoError(t, bErr)
		bodyBytesCheck = fmt.Sprintf("blake2b256(%s.bodyBytes) == base64'%s'",
			ops.txVarName, base64.StdEncoding.EncodeToString(crypto.MustFastHash(body).Bytes()),
		)
	}
	rnd := rand.Uint32()
	var proofsChecks string
	if ops.checkProofs {
		for i := range proofsArr {
			bv, okP := proofsArr[i].(rideByteVector)
			require.Truef(t, okP, "proofs[%d] is not rideByteVector, but %T", i, proofsArr[i])
			num := int(rnd) + i
			proofsChecks += fmt.Sprintf("strict proof%d = %s.proofs[%d] == %s || throw(\"proof[%d] mismatch\")\n",
				num, ops.txVarName, i, bv.String(), i,
			)
		}
	}
	return fmt.Sprintf(`
		strict id%d = %s.id == %s || throw("id mismatch")
		strict fee%d = %s.fee == %d || throw("fee mismatch")
		strict timestamp%d = %s.timestamp == %d || throw("timestamp mismatch")
		strict bodyBytes%d = %s || throw("bodyBytes mismatch")
		strict sender%d = %s.sender == Address(base58'%s') || throw("sender mismatch")
		strict senderPublicKey%d = %s.senderPublicKey == %s || throw("senderPublicKey mismatch")
		strict version%d = %s.version == %d || throw("version mismatch")
		%s
	`,
		rnd, ops.txVarName, mustField[rideByteVector](t, obj, idField).String(),
		rnd, ops.txVarName, mustField[rideInt](t, obj, feeField),
		rnd, ops.txVarName, mustField[rideInt](t, obj, timestampField),
		rnd, bodyBytesCheck,
		rnd, ops.txVarName, proto.WavesAddress(mustField[rideAddress](t, obj, senderField)).String(),
		rnd, ops.txVarName, mustField[rideByteVector](t, obj, senderPublicKeyField).String(),
		rnd, ops.txVarName, mustField[rideInt](t, obj, versionField),
		proofsChecks,
	)
}

func TestRideCommitToGenerationTransactionConstruction(t *testing.T) {
	const (
		seed           = "SENDER"
		txVersion      = 1
		txFee          = 100000
		genPeriodStart = 11
		timestamp      = 12345
		scheme         = proto.TestNetScheme
	)
	acc := newTestAccount(t, seed)
	blsSK, err := bls.GenerateSecretKey([]byte(seed))
	require.NoError(t, err)
	blsPK, err := blsSK.PublicKey()
	require.NoError(t, err)
	_, sig, err := bls.ProvePoP(blsSK, blsPK, genPeriodStart)
	require.NoError(t, err)
	tx := proto.NewUnsignedCommitToGenerationWithProofs(txVersion, acc.pk, genPeriodStart, blsPK, sig, txFee, timestamp)
	err = tx.Sign(scheme, acc.sk)
	require.NoError(t, err)

	env := newTestEnv(t).withScheme(scheme).withLibVersion(ast.LibV9).
		withComplexityLimit(5000).withRideV6Activated().
		withSender(acc).withTransaction(tx)

	script := fmt.Sprintf(`
	{-# STDLIB_VERSION 9 #-}
	{-# CONTENT_TYPE EXPRESSION #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	match tx {
	 case t: CommitToGenerationTransaction =>
		%s
		strict endorserPublicKey = t.endorserPublicKey == base58'%s' || throw("endorserPublicKey mismatch")
		strict generationPeriodStart = t.generationPeriodStart == %d || throw("generationPeriodStart mismatch")
		strict commitmentSignature = t.commitmentSignature == base58'%s' || throw("commitmentSignature mismatch")
		true
	 case _ => throw("tx has bad type")
	}`,
		provenPart(t, env.toEnv(), tx),
		blsPK.String(),
		genPeriodStart,
		sig.String(),
	)
	tree, errs := ridec.CompileToTree(script)
	require.NoError(t, stderrs.Join(errs...))

	r, err := CallVerifier(env.toEnv(), tree)
	require.NoError(t, err)
	require.True(t, r.Result())
}
