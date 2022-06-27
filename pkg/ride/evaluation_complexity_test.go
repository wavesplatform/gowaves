package ride

import (
	"encoding/base64"
	"strconv"
	"strings"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type testStage struct {
	env             *mockRideEnvironment
	state           *WrappedState
	inv             rideObject
	this            proto.WavesAddress
	libVersion      ast.LibraryVersion
	rideV5Activated bool
	rideV6Activated bool
	trees           map[proto.WavesAddress]*ast.Tree
	publicKeys      map[proto.WavesAddress]crypto.PublicKey
}

func makeTestStage(inv, tx rideObject, this proto.WavesAddress, rideV5, rideV6 bool, libVersion ast.LibraryVersion,
	trees map[proto.WavesAddress]*ast.Tree, publicKeys map[proto.WavesAddress]crypto.PublicKey) *testStage {
	r := &testStage{
		inv:             inv,
		this:            this,
		libVersion:      libVersion,
		rideV5Activated: rideV5,
		rideV6Activated: rideV6,
		trees:           trees,
		publicKeys:      publicKeys,
	}
	r.env = &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		thisFunc: func() rideType {
			return rideAddress(r.this)
		},
		transactionFunc: func() rideObject {
			return tx
		},
		invocationFunc: func() rideObject {
			return r.inv
		},
		checkMessageLengthFunc: v3check,
		setInvocationFunc: func(inv rideObject) {
			r.inv = inv
		},
		validateInternalPaymentsFunc: trueFn,
		maxDataEntriesSizeFunc: func() int {
			return proto.MaxDataEntriesScriptActionsSizeInBytesV2
		},
		blockV5ActivatedFunc: func() bool {
			return true
		},
		rideV5ActivatedFunc: func() bool {
			return r.rideV5Activated
		},
		rideV6ActivatedFunc: func() bool {
			return r.rideV6Activated
		},
		isProtobufTxFunc: isProtobufTx,
		libVersionFunc: func() ast.LibraryVersion {
			return r.libVersion
		},
	}
	state := &MockSmartState{
		NewestScriptByAccountFunc: func(recipient proto.Recipient) (*ast.Tree, error) {
			if tree, ok := r.trees[*recipient.Address]; ok {
				return tree, nil
			}
			return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
		},
		NewestScriptPKByAddrFunc: func(addr proto.WavesAddress) (crypto.PublicKey, error) {
			if pk, ok := r.publicKeys[addr]; ok {
				return pk, nil
			}
			return crypto.PublicKey{}, errors.Errorf("unexpected address %s", addr.String())
		},
		NewestRecipientToAddressFunc: func(recipient proto.Recipient) (*proto.WavesAddress, error) {
			if _, ok := r.trees[*recipient.Address]; ok {
				return recipient.Address, nil
			}
			return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
		},
		NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
			if _, ok := r.trees[*account.Address]; ok {
				return 10_00000000, nil
			}
			return 0, errors.Errorf("unxepected account '%s'", account.String())
		},
		NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
			if _, ok := r.trees[*account.Address]; ok {
				return &proto.FullWavesBalance{
					Regular:    10_00000000,
					Generating: 10_00000000,
					Available:  10_00000000,
					Effective:  10_00000000,
					LeaseIn:    0,
					LeaseOut:   0,
				}, nil
			}
			return &proto.FullWavesBalance{}, nil
		},
	}
	r.state = &WrappedState{
		diff:                      newDiffState(state),
		cle:                       r.env.this().(rideAddress),
		scheme:                    r.env.scheme(),
		rootScriptLibVersion:      r.libVersion,
		rootActionsCountValidator: proto.NewScriptActionsCountValidator(),
	}
	r.env.stateFunc = func() types.SmartState {
		return r.state
	}
	r.env.setNewDAppAddressFunc = func(address proto.WavesAddress) {
		r.this = address
		r.state.cle = rideAddress(address) // We have to update wrapped state's `cle`
	}
	return r
}

// parseArguments converts string of comma separated string with prefix encoded values to the slice of proto.Argument.
// To encode proto.StringArgument use string "s'xxx'", "xxx" is the argument value.
// To encode proto.IntegerArgument use string "i'12345'", 12345 is the value of the argument.
// To encode proto.BooleanArgument use string "b'true'" or "b'false'".
// To encode proto.BinaryArgument use string "b58'1111'" or "b64'dGVzdA=='.
// Eg, "s'TEST', i'123456', b'true', "b64'dGVzdA=='"
func parseArguments(t *testing.T, arguments string) proto.Arguments {
	if arguments == "" {
		return proto.Arguments{}
	}
	args := strings.Split(arguments, ",")
	r := make(proto.Arguments, len(args))
	for i, a := range args {
		sp := strings.Split(strings.TrimSpace(a), "'")
		switch sp[0] {
		case "s":
			r[i] = &proto.StringArgument{Value: sp[1]}
		case "i":
			v, err := strconv.ParseInt(sp[1], 10, 64)
			require.NoError(t, err)
			r[i] = &proto.IntegerArgument{Value: v}
		case "b":
			v, err := strconv.ParseBool(sp[1])
			require.NoError(t, err)
			r[i] = &proto.BooleanArgument{Value: v}
		case "b64":
			v, err := base64.StdEncoding.DecodeString(sp[1])
			require.NoError(t, err)
			r[i] = &proto.BinaryArgument{Value: v}
		case "b58":
			v, err := base58.Decode(sp[1])
			require.NoError(t, err)
			r[i] = &proto.BinaryArgument{Value: v}
		default:
			t.Fatalf("unsupported argument prefix '%s'", sp[0])
		}
	}
	return r
}

func makeInvokeTransactionTestObjects(t *testing.T, senderPK crypto.PublicKey, dAppAddress proto.WavesAddress, functionName, arguments string) (rideObject, rideObject) {
	call := proto.FunctionCall{
		Default:   false,
		Name:      functionName,
		Arguments: parseArguments(t, arguments),
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              makeRandomTxID(t),
		Proofs:          proto.NewProofs(),
		ChainID:         proto.TestNetScheme,
		SenderPK:        senderPK,
		ScriptRecipient: proto.NewRecipientFromAddress(dAppAddress),
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1624967106278,
	}
	invObj, err := invocationToObject(5, proto.TestNetScheme, tx)
	require.NoError(t, err)
	txObj, err := transactionToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	return invObj, txObj
}

func TestComplexitiesV5V6(t *testing.T) {
	_, dApp1PK, dApp1 := makeAddressAndPK(t, "DAPP1")    // 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz
	_, dApp2PK, dApp2 := makeAddressAndPK(t, "DAPP2")    // 3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1
	_, dApp3PK, dApp3 := makeAddressAndPK(t, "DAPP3")    // 3N186hYM5PFwGdkVUsLJaBvpPEECrSj5CJh
	_, dApp4PK, dApp4 := makeAddressAndPK(t, "DAPP4")    // 3Mtfbvy5nEGNR2ZNAWJUHauEfFsBysAr1S6
	_, senderPK, sender := makeAddressAndPK(t, "SENDER") // 3N8CkZAyS4XcDoJTJoKNuNk2xmNKmQj7myW

	/* On dApp1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let dapp2 = Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1')

	let message = base58'emsY'
	let pub = base58'HnU9jfhpMcQNaG5yQ46eR43RnkWKGxerw2zVrbpnbGof'
	let sig = base58'4uXfw7162zaopAkTNa7eo6YK2mJsTiHGJL3dCtRRH63z1nrdoHBHyhbvrfZovkxf2jKsi2vPsaP2XykfZmUiwPeg'

	let complex = sigVerify(message, sig, pub) && sigVerify(message, sig, pub) && sigVerify(message, sig, pub) && sigVerify_64Kb(message, sig, pub)
	let complexCase4 = sigVerify(message, sig, pub) && sigVerify(message, sig, pub) && sigVerify(message, sig, pub) && sigVerify_64Kb(message, sig, pub)
	let complexCase56 = sigVerify(message, sig, pub) && sigVerify(message, sig, pub) && sigVerify(message, sig, pub) && sigVerify_32Kb(message, sig, pub)

	@Callable(i)
	func case1() = {
	  strict inv = invoke(dapp2, "case1", [complex], [])
	  []
	}

	@Callable(i)
	func case2() = {
	  strict inv = invoke(dapp2, "case2", [complex], [])
	  []
	}

	@Callable(i)
	func case3() = {
	  strict inv = invoke(dapp2, "case3", [complex], [])
	  []
	}

	@Callable(i)
	func case4() = {
	  strict inv = invoke(dapp2, "case4", [complex && complexCase4], [])
	  []
	}

	@Callable(i)
	func case5() = {
	  strict inv = invoke(dapp2, "case5", [complexCase56], [])
	  []
	}

	@Callable(i)
	func case6() = {
	  strict inv = invoke(dapp2, "case6", [complexCase56], [])
	  []
	}
	*/
	code1 := "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"
	_, tree1 := parseBase64Script(t, code1)

	assert.NotNil(t, tree1)
	/* On dApp2
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let dapp3 = Address(base58'3N186hYM5PFwGdkVUsLJaBvpPEECrSj5CJh')

	let message = base58'emsY'
	let pub = base58'HnU9jfhpMcQNaG5yQ46eR43RnkWKGxerw2zVrbpnbGof'
	let sig = base58'4uXfw7162zaopAkTNa7eo6YK2mJsTiHGJL3dCtRRH63z1nrdoHBHyhbvrfZovkxf2jKsi2vPsaP2XykfZmUiwPeg'

	let complexCase1 = sigVerify_8Kb(message, sig, pub)
	let complexCase2 = sigVerify_32Kb(message, sig, pub) && sigVerify_16Kb(message, sig, pub)
	let complexCase34 = sigVerify(message, sig, pub) && sigVerify(message, sig, pub)
	let complexCase5 = sigVerify_8Kb(message, sig, pub) && sigVerify_8Kb(message, sig, pub)
	let complexCase6 = sigVerify_64Kb(message, sig, pub) && sigVerify_32Kb(message, sig, pub)

	@Callable(i)
	func case1(bool:Boolean) = {
	  strict inv = invoke(dapp3, "case1", [complexCase1], [])
	  []
	}

	@Callable(i)
	func case2(bool:Boolean) = {
	  strict inv = invoke(dapp3, "case2", [complexCase2], [])
	  []
	}

	@Callable(i)
	func case3(bool:Boolean) = {
	  strict inv = invoke(dapp3, "case3", [complexCase34], [])
	  []
	}

	@Callable(i)
	func case4(bool:Boolean) = {
	  if (complexCase34) then {
	    [ScriptTransfer(dapp3, 99900000000, unit)]
	   } else []
	}

	@Callable(i)
	func case5(bool:Boolean) = {
	  strict inv = invoke(dapp3, "case5", [complexCase5], [])
	  []
	}

	@Callable(i)
	func case6(bool:Boolean) = {
	  strict inv = invoke(dapp3, "case6", [complexCase6], [])
	  []
	}
	*/
	code2 := "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"
	_, tree2 := parseBase64Script(t, code2)

	/* On dApp3
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let message = base58'emsY'
	let pub = base58'HnU9jfhpMcQNaG5yQ46eR43RnkWKGxerw2zVrbpnbGof'
	let sig = base58'4uXfw7162zaopAkTNa7eo6YK2mJsTiHGJL3dCtRRH63z1nrdoHBHyhbvrfZovkxf2jKsi2vPsaP2XykfZmUiwPeg'

	let dapp4 = Address(base58'3Mtfbvy5nEGNR2ZNAWJUHauEfFsBysAr1S6')

	let complexCase56 = sigVerify_16Kb(message, sig, pub) && sigVerify_8Kb(message, sig, pub)

	@Callable(i)
	func case1(bool:Boolean) = {
	  if (sigVerify(message, sig, pub)) then
	    [ScriptTransfer(i.caller, 99900000000, unit)]
	  else []
	}

	@Callable(i)
	func case2(bool:Boolean) = {
	  [ScriptTransfer(i.caller, 99900000000, unit)]
	}

	@Callable(i)
	func case3(bool:Boolean) = {
	  [ScriptTransfer(i.caller, 99900000000, unit)]
	}

	@Callable(i)
	func case5(bool:Boolean) = {
	  strict inv = invoke(dapp4, "case5", [complexCase56], [])
	  []
	}

	@Callable(i)
	func case6(bool:Boolean) = {
	  strict inv = invoke(dapp4, "case6", [complexCase56], [])
	  []
	}
	*/
	code3 := "AAIFAAAAAAAAABsIAhIDCgEEEgMKAQQSAwoBBBIDCgEEEgMKAQQAAAAFAAAAAAdtZXNzYWdlAQAAAANwdWsAAAAAA3B1YgEAAAAg+WDTl1J9TPwoZK08IrX5nepU6IE6+Sop9sM4jqz/Ti4AAAAAA3NpZwEAAABAw1mP1rioR7NALQUl9VfqzYKgyTigqanddUm2A1+z5zJdAr+g4iKaVKvD4agOP4Nsv84I8Ht1zgDkT9N9jRU0gQAAAAAFZGFwcDQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVQ0G4/8L+t0U77mKK9peiJfObsVZZe0WycAAAAADWNvbXBsZXhDYXNlNTYDCQAJxQAAAAMFAAAAB21lc3NhZ2UFAAAAA3NpZwUAAAADcHViCQAJxAAAAAMFAAAAB21lc3NhZ2UFAAAAA3NpZwUAAAADcHViBwAAAAUAAAABaQEAAAAFY2FzZTEAAAABAAAABGJvb2wDCQAB9AAAAAMFAAAAB21lc3NhZ2UFAAAAA3NpZwUAAAADcHViCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAXQoEHAAUAAAAEdW5pdAUAAAADbmlsBQAAAANuaWwAAAABaQEAAAAFY2FzZTIAAAABAAAABGJvb2wJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyAAAAABdCgQcABQAAAAR1bml0BQAAAANuaWwAAAABaQEAAAAFY2FzZTMAAAABAAAABGJvb2wJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyAAAAABdCgQcABQAAAAR1bml0BQAAAANuaWwAAAABaQEAAAAFY2FzZTUAAAABAAAABGJvb2wEAAAAA2ludgkAA/wAAAAEBQAAAAVkYXBwNAIAAAAFY2FzZTUJAARMAAAAAgUAAAANY29tcGxleENhc2U1NgUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAA2ludgUAAAADaW52BQAAAANuaWwJAAACAAAAAQIAAAAkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuAAAAAWkBAAAABWNhc2U2AAAAAQAAAARib29sBAAAAANpbnYJAAP8AAAABAUAAAAFZGFwcDQCAAAABWNhc2U2CQAETAAAAAIFAAAADWNvbXBsZXhDYXNlNTYFAAAAA25pbAUAAAADbmlsAwkAAAAAAAACBQAAAANpbnYFAAAAA2ludgUAAAADbmlsCQAAAgAAAAECAAAAJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgAAAAAmhwaJ"
	_, tree3 := parseBase64Script(t, code3)

	/* On dApp4
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func case5(bool:Boolean) = {
	  [ScriptTransfer(i.caller, 99900000000, unit)]
	}

	@Callable(i)
	func case6(bool:Boolean) = {
	  [ScriptTransfer(i.caller, 99900000000, unit)]
	}
	*/
	code4 := "AAIFAAAAAAAAAAwIAhIDCgEEEgMKAQQAAAAAAAAAAgAAAAFpAQAAAAVjYXNlNQAAAAEAAAAEYm9vbAkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAF0KBBwAFAAAABHVuaXQFAAAAA25pbAAAAAFpAQAAAAVjYXNlNgAAAAEAAAAEYm9vbAkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAF0KBBwAFAAAABHVuaXQFAAAAA25pbAAAAABv44L5"
	_, tree4 := parseBase64Script(t, code4)

	invObj, txObj := makeInvokeTransactionTestObjects(t, senderPK, dApp1, "case1", "")
	tst := makeTestStage(invObj, txObj, dApp1, true, false, ast.LibV5,
		map[proto.WavesAddress]*ast.Tree{dApp1: tree1, dApp2: tree2, dApp3: tree3, dApp4: tree4},
		map[proto.WavesAddress]crypto.PublicKey{dApp1: dApp1PK, dApp2: dApp2PK, dApp3: dApp3PK, dApp4: dApp4PK, sender: senderPK})
	_, err := CallFunction(tst.env, tree1, "case1", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 797+130, EvaluationErrorSpentComplexity(err))

	tst.rideV6Activated = true
	_, err = CallFunction(tst.env, tree1, "case1", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 1105, EvaluationErrorSpentComplexity(err))

	invObj, txObj = makeInvokeTransactionTestObjects(t, senderPK, dApp1, "case2", "")
	tst = makeTestStage(invObj, txObj, dApp1, true, false, ast.LibV5,
		map[proto.WavesAddress]*ast.Tree{dApp1: tree1, dApp2: tree2, dApp3: tree3, dApp4: tree4},
		map[proto.WavesAddress]crypto.PublicKey{dApp1: dApp1PK, dApp2: dApp2PK, dApp3: dApp3PK, dApp4: dApp4PK, sender: senderPK})
	_, err = CallFunction(tst.env, tree1, "case2", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 797+214, EvaluationErrorSpentComplexity(err))

	tst.rideV6Activated = true
	_, err = CallFunction(tst.env, tree1, "case2", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 985, EvaluationErrorSpentComplexity(err))

	invObj, txObj = makeInvokeTransactionTestObjects(t, senderPK, dApp1, "case3", "")
	tst = makeTestStage(invObj, txObj, dApp1, true, false, ast.LibV5,
		map[proto.WavesAddress]*ast.Tree{dApp1: tree1, dApp2: tree2, dApp3: tree3, dApp4: tree4},
		map[proto.WavesAddress]crypto.PublicKey{dApp1: dApp1PK, dApp2: dApp2PK, dApp3: dApp3PK, dApp4: dApp4PK, sender: senderPK})
	_, err = CallFunction(tst.env, tree1, "case3", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 797+487, EvaluationErrorSpentComplexity(err))

	tst.rideV6Activated = true
	_, err = CallFunction(tst.env, tree1, "case3", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 1258, EvaluationErrorSpentComplexity(err))

	invObj, txObj = makeInvokeTransactionTestObjects(t, senderPK, dApp1, "case4", "")
	tst = makeTestStage(invObj, txObj, dApp1, true, false, ast.LibV5,
		map[proto.WavesAddress]*ast.Tree{dApp1: tree1, dApp2: tree2, dApp3: tree3, dApp4: tree4},
		map[proto.WavesAddress]crypto.PublicKey{dApp1: dApp1PK, dApp2: dApp2PK, dApp3: dApp3PK, dApp4: dApp4PK, sender: senderPK})
	_, err = CallFunction(tst.env, tree1, "case4", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 1516, EvaluationErrorSpentComplexity(err))

	tst.rideV6Activated = true
	_, err = CallFunction(tst.env, tree1, "case4", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 1884, EvaluationErrorSpentComplexity(err))

	invObj, txObj = makeInvokeTransactionTestObjects(t, senderPK, dApp1, "case5", "")
	tst = makeTestStage(invObj, txObj, dApp1, true, false, ast.LibV5,
		map[proto.WavesAddress]*ast.Tree{dApp1: tree1, dApp2: tree2, dApp3: tree3, dApp4: tree4},
		map[proto.WavesAddress]crypto.PublicKey{dApp1: dApp1PK, dApp2: dApp2PK, dApp3: dApp3PK, dApp4: dApp4PK, sender: senderPK})
	_, err = CallFunction(tst.env, tree1, "case5", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 765+181+191, EvaluationErrorSpentComplexity(err))

	tst.rideV6Activated = true
	_, err = CallFunction(tst.env, tree1, "case5", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 1101, EvaluationErrorSpentComplexity(err))

	invObj, txObj = makeInvokeTransactionTestObjects(t, senderPK, dApp1, "case6", "")
	tst = makeTestStage(invObj, txObj, dApp1, true, false, ast.LibV5,
		map[proto.WavesAddress]*ast.Tree{dApp1: tree1, dApp2: tree2, dApp3: tree3, dApp4: tree4},
		map[proto.WavesAddress]crypto.PublicKey{dApp1: dApp1PK, dApp2: dApp2PK, dApp3: dApp3PK, dApp4: dApp4PK, sender: senderPK})
	_, err = CallFunction(tst.env, tree1, "case6", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 765+259+191, EvaluationErrorSpentComplexity(err))

	tst.rideV6Activated = true
	_, err = CallFunction(tst.env, tree1, "case6", parseArguments(t, ""))
	require.Error(t, err)
	assert.Equal(t, 1179, EvaluationErrorSpentComplexity(err))
}

func TestSelfInvokeComplexities(t *testing.T) {
	_, dApp1PK, dApp1 := makeAddressAndPK(t, "DAPP1")    // 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz
	_, senderPK, sender := makeAddressAndPK(t, "SENDER") // 3N8CkZAyS4XcDoJTJoKNuNk2xmNKmQj7myW

	/* On dApp1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func call( r: Int ) = {
	  if( r == 0 ) then [ ScriptTransfer(Address(base58'3N8CkZAyS4XcDoJTJoKNuNk2xmNKmQj7myW'), 1000000000, unit ) ] else
	  let f = fraction( fraction( r, 1, 1 ), 1, 1 )
	  strict g = invoke( this, "call", [ f - 1 ], [] )
	  []
	}
	*/
	code := "AAIFAAAAAAAAAAcIAhIDCgEBAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAEAAAABcgMJAAAAAAAAAgUAAAABcgAAAAAAAAAAAAkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCQEAAAAHQWRkcmVzcwAAAAEBAAAAGgFUyJlKpWZVh64i0Ak7F/0vFuHDEV0ZUdLtAAAAAAA7msoABQAAAAR1bml0BQAAAANuaWwEAAAAAWYJAABrAAAAAwkAAGsAAAADBQAAAAFyAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABBAAAAAFnCQAD/AAAAAQFAAAABHRoaXMCAAAABGNhbGwJAARMAAAAAgkAAGUAAAACBQAAAAFmAAAAAAAAAAABBQAAAANuaWwFAAAAA25pbAMJAAAAAAAAAgUAAAABZwUAAAABZwUAAAADbmlsCQAAAgAAAAECAAAAJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgAAAAAyP+y9"
	_, tree := parseBase64Script(t, code)

	invObj, txObj := makeInvokeTransactionTestObjects(t, senderPK, dApp1, "call", "i'9'")
	tst := makeTestStage(invObj, txObj, dApp1, true, false, ast.LibV5,
		map[proto.WavesAddress]*ast.Tree{dApp1: tree},
		map[proto.WavesAddress]crypto.PublicKey{dApp1: dApp1PK, sender: senderPK})
	_, err := CallFunction(tst.env, tree, "call", parseArguments(t, "i'9'"))
	require.Error(t, err)
	assert.Equal(t, 904, EvaluationErrorSpentComplexity(err))

	tst.rideV6Activated = true
	_, err = CallFunction(tst.env, tree, "call", parseArguments(t, "i'9'"))
	require.Error(t, err)
	assert.Equal(t, 958, EvaluationErrorSpentComplexity(err))

	tst.rideV6Activated = false
	_, err = CallFunction(tst.env, tree, "call", parseArguments(t, "i'10'"))
	require.Error(t, err)
	assert.Equal(t, 1017, EvaluationErrorSpentComplexity(err))

	tst.rideV6Activated = true
	_, err = CallFunction(tst.env, tree, "call", parseArguments(t, "i'10'"))
	require.Error(t, err)
	assert.Equal(t, 1064, EvaluationErrorSpentComplexity(err))

}
