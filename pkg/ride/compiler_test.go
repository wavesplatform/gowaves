package ride

import (
	"encoding/base64"
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

var defaultState = &MockSmartState{
	NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
		return byte_helpers.TransferWithProofs.Transaction, nil
	},
	RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
		return nil, errors.New("not found")
	},
	RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
		v, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			return nil, err
		}
		return &proto.IntegerDataEntry{
			Value: v,
		}, nil
	},
}

var defaultEnv = &MockRideEnvironment{
	transactionFunc: testTransferObject,
	stateFunc: func() types.SmartState {
		return defaultState
	},
	schemeFunc: func() byte {
		return 'T'
	},
	thisFunc: func() rideType {
		return rideAddress{}
	},
	invocationFunc: func() rideObject {
		return rideObject{}
	},
	heightFunc: func() rideInt {
		return rideInt(100500)
	},
}

func TestCompiler(t *testing.T) {
	env := defaultEnv
	for _, test := range []struct {
		comment string
		source  string
		env     RideEnvironment
		res     bool
	}{
		{`V1: true`, "AQa3b8tH", env, true},
		{`V1: false`, `AQfeYll6`, nil, false},
		{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn", env, true},
		{`V3: let x = true; x`, "BAQAAAABeAYFAAAAAXhUb/5M", env, true},
		{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=", nil, true},
		{`V1: let i = 1; let s = "string"; toString(i) == s`, "BAQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABc6Y8UOc=", env, false},
		{`V3: let i = 12345; let s = "12345"; toString(i) == s`, "AwQAAAABaQAAAAAAAAAwOQQAAAABcwIAAAAFMTIzNDUJAAAAAAAAAgkAAaQAAAABBQAAAAFpBQAAAAFz1B1iCw==", nil, true},
		{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E", env, true},
		{`V3: if (false) then {let r = true; r} else {let r = false; r}`, "AwMHBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXI+tfo1", env, false},
		{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==", env, true},
		{`V3: func a() = 1; a() == 2`, "BAoBAAAAAWEAAAAAAAAAAAAAAAABCQAAAAAAAAIJAQAAAAFhAAAAAAAAAAAAAAAAAsVdmuc=", env, false},
		{`V3: func abc() = true; abc()`, "BAoBAAAAA2FiYwAAAAAGCQEAAAADYWJjAAAAANHu1ew=", env, true},
		{`V3: func id(v: Boolean) = v; id(true)`, "BAoBAAAAAmlkAAAAAQAAAAF2BQAAAAF2CQEAAAACaWQAAAABBglAaUs=", env, true},
		{`V3: 1 == 1`, "BAkAAAAAAAACAAAAAAAAAAABAAAAAAAAAAABq0EiMw==", env, true},
		{`V3: (1 == 1) == (1 == 1)`, "BAkAAAAAAAACCQAAAAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEJAAAAAAAAAgAAAAAAAAAAAQAAAAAAAAAAAWXKjzM=", env, true},
		{`V3: let x = 1; func add(i: Int) = i + 1; add(x) == 2`, "AwQAAAABeAAAAAAAAAAAAQoBAAAAA2FkZAAAAAEAAAABaQkAAGQAAAACBQAAAAFpAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAF4AAAAAAAAAAACfr6U6w==", env, true},
		{`V3: let x = if (true) then true else false; x`, "BAQAAAABeAMGBgcFAAAAAXgCINPC", env, true},
		{`V3: let b = base16'0000000000000001'; func add(b: ByteVector) = toInt(b) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAWIJAABkAAAAAgkABLEAAAABBQAAAAFiAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAACX00biA==", nil, true},
		{`V3: let b = base16'0000000000000001'; func add(v: ByteVector) = toInt(v) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAXYJAABkAAAAAgkABLEAAAABBQAAAAF2AAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAACI7gYxg==", nil, true},
		{`V3: let b = base16'0000000000000001'; func add(v: ByteVector) = toInt(b) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAXYJAABkAAAAAgkABLEAAAABBQAAAAFiAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAAChRvwnQ==", nil, true},
		{`V3: let data = base64'AAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLw=='; func getStock(data:ByteVector) = toInt(take(drop(data, 8), 8)); getStock(data) == 1`, `AwQAAAAEZGF0YQEAAABwAAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLwoBAAAACGdldFN0b2NrAAAAAQAAAARkYXRhCQAEsQAAAAEJAADJAAAAAgkAAMoAAAACBQAAAARkYXRhAAAAAAAAAAAIAAAAAAAAAAAICQAAAAAAAAIJAQAAAAhnZXRTdG9jawAAAAEFAAAABGRhdGEAAAAAAAAAAAFCtabi`, env, true},
		{`V3: let ref = 999; func g(a: Int) = ref; func f(ref: Int) = g(ref); f(1) == 999`, "AwQAAAADcmVmAAAAAAAAAAPnCgEAAAABZwAAAAEAAAABYQUAAAADcmVmCgEAAAABZgAAAAEAAAADcmVmCQEAAAABZwAAAAEFAAAAA3JlZgkAAAAAAAACCQEAAAABZgAAAAEAAAAAAAAAAAEAAAAAAAAAA+fjknmW", env, true},
		{`let x = 5; 6 > 4`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGAAAAAAAAAAAEYSW6XA==`, nil, true},
		{`let x = 5; 6 > x`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGBQAAAAF4Gh24hw==`, nil, true},
		{`let x = 5; 6 >= x`, `AQQAAAABeAAAAAAAAAAABQkAAGcAAAACAAAAAAAAAAAGBQAAAAF4jlxXHA==`, nil, true},
		{`let x = {let y = true;y}x`, `BAQAAAABeAQAAAABeQYFAAAAAXkFAAAAAXhCPj2C`, nil, true},
		{`let x =  throw(); true`, `AQQAAAABeAkBAAAABXRocm93AAAAAAa7bgf4`, nil, true},
		{`let x =  throw(); true || x`, `AQQAAAABeAkBAAAABXRocm93AAAAAAMGBgUAAAABeKRnLds=`, env, true},
		{`tx == tx`, "BAkAAAAAAAACBQAAAAJ0eAUAAAACdHhnqgP4", env, true},
		{`func getInt(key: String) = {match getInteger(this, key) {case x: Int => x; case _ => 0;}}; let a = getInt("5"); let b = getInt("6"); a == b`, "BAoBAAAABmdldEludAAAAAEAAAADa2V5BAAAAAckbWF0Y2gwCQAEGgAAAAIFAAAABHRoaXMFAAAAA2tleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAF4BQAAAAckbWF0Y2gwBQAAAAF4AAAAAAAAAAAABAAAAAFhCQEAAAAGZ2V0SW50AAAAAQIAAAABNQQAAAABYgkBAAAABmdldEludAAAAAECAAAAATYJAAAAAAAAAgUAAAABYQUAAAABYkOIJQA=", env, false},
		{`func abc() = {func in() = true; in()}; abc()`, "BAoBAAAAA2FiYwAAAAAKAQAAAAJpbgAAAAAGCQEAAAACaW4AAAAACQEAAAADYWJjAAAAADpBKyM=", env, true},
		{`func inc(v: Int) = v + 1; func call(inc: Int) = inc(inc); call(2) == 3`, "AwoBAAAAA2luYwAAAAEAAAABdgkAAGQAAAACBQAAAAF2AAAAAAAAAAABCgEAAAAEY2FsbAAAAAEAAAADaW5jCQEAAAADaW5jAAAAAQUAAAADaW5jCQAAAAAAAAIJAQAAAARjYWxsAAAAAQAAAAAAAAAAAgAAAAAAAAAAAxgTXMY=", env, true},
		{`func abc(addr: Address) = addr == tx.sender;abc(tx.sender)`, "BAoBAAAAA2FiYwAAAAEAAAAEYWRkcgkAAAAAAAACBQAAAARhZGRyCAUAAAACdHgAAAAGc2VuZGVyCQEAAAADYWJjAAAAAQgFAAAAAnR4AAAABnNlbmRlckJrXFI=", env, true},
		{`let y = [{let x = 1;x}];true`, "BAQAAAABeQkABEwAAAACBAAAAAF4AAAAAAAAAAABBQAAAAF4BQAAAANuaWwGua/TXw==", env, true},
		{`tx.id == base58''`, `AQkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAAJBtD70=`, env, false},
		{`tx.id == base58'H5C8bRzbUTMePSDVVxjiNKDUwk6CKzfZGTP2Rs7aCjsV'`, `BAkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAIO7N5luRDUgN1SJ4kFmy/Ni8U2H6k7bpszok5tlLlRVgHwSHyg==`, env, false},
		{`tx.id == tx.id`, `BAkAAAAAAAACCAUAAAACdHgAAAACaWQIBQAAAAJ0eAAAAAJpZHErpOM=`, env, true},
		{`let x = tx.id == base58'a';true`, `AQQAAAABeAkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAASEGjR0kcA==`, env, true},
		{`tx.proofs[0] != base58'' && tx.proofs[1] == base58''`, `BAMJAQAAAAIhPQAAAAIJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAAEAAAAACQAAAAAAAAIJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAQEAAAAAB106gzM=`, env, true},
		{`match tx {case t : TransferTransaction | MassTransferTransaction | ExchangeTransaction => true; case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABNFeGNoYW5nZVRyYW5zYWN0aW9uBgMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24GCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAGB6Ilvok=`, env, true},
		{`V2: match transactionById(tx.id) {case  t: Unit => false case _ => true}`, `AgQAAAAHJG1hdGNoMAkAA+gAAAABCAUAAAACdHgAAAACaWQDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABFVuaXQEAAAAAXQFAAAAByRtYXRjaDAHBp9TFcQ=`, env, true},
		{`Up() == UP`, `AwkAAAAAAAACCQEAAAACVXAAAAAABQAAAAJVUPGUxeg=`, env, true},
		{`HalfUp() == HALFUP`, `AwkAAAAAAAACCQEAAAAGSGFsZlVwAAAAAAUAAAAGSEFMRlVQbUfpTQ==`, nil, true},
		{`let a0 = NoAlg() == NOALG; let a1 = Md5() == MD5; let a2 = Sha1() == SHA1; let a3 = Sha224() == SHA224; let a4 = Sha256() == SHA256; let a5 = Sha384() == SHA384; let a6 = Sha512() == SHA512; let a7 = Sha3224() == SHA3224; let a8 = Sha3256() == SHA3256; let a9 = Sha3384() == SHA3384; let a10 = Sha3512() == SHA3512; a0 && a1 && a2 && a3 && a4 && a5 && a6 && a7 && a8 && a9 && a10`, `AwQAAAACYTAJAAAAAAAAAgkBAAAABU5vQWxnAAAAAAUAAAAFTk9BTEcEAAAAAmExCQAAAAAAAAIJAQAAAANNZDUAAAAABQAAAANNRDUEAAAAAmEyCQAAAAAAAAIJAQAAAARTaGExAAAAAAUAAAAEU0hBMQQAAAACYTMJAAAAAAAAAgkBAAAABlNoYTIyNAAAAAAFAAAABlNIQTIyNAQAAAACYTQJAAAAAAAAAgkBAAAABlNoYTI1NgAAAAAFAAAABlNIQTI1NgQAAAACYTUJAAAAAAAAAgkBAAAABlNoYTM4NAAAAAAFAAAABlNIQTM4NAQAAAACYTYJAAAAAAAAAgkBAAAABlNoYTUxMgAAAAAFAAAABlNIQTUxMgQAAAACYTcJAAAAAAAAAgkBAAAAB1NoYTMyMjQAAAAABQAAAAdTSEEzMjI0BAAAAAJhOAkAAAAAAAACCQEAAAAHU2hhMzI1NgAAAAAFAAAAB1NIQTMyNTYEAAAAAmE5CQAAAAAAAAIJAQAAAAdTaGEzMzg0AAAAAAUAAAAHU0hBMzM4NAQAAAADYTEwCQAAAAAAAAIJAQAAAAdTaGEzNTEyAAAAAAUAAAAHU0hBMzUxMgMDAwMDAwMDAwMFAAAAAmEwBQAAAAJhMQcFAAAAAmEyBwUAAAACYTMHBQAAAAJhNAcFAAAAAmE1BwUAAAACYTYHBQAAAAJhNwcFAAAAAmE4BwUAAAACYTkHBQAAAANhMTAHRc/wAA==`, env, true},
		{`Unit() == unit`, `AwkAAAAAAAACCQEAAAAEVW5pdAAAAAAFAAAABHVuaXTstg1G`, env, true},
		{`V4: let a = 1; let b = 2; let c = 3; let d = 4; let (x, y) = ((a+b), (c+d)); x + y == 10`, `BAQAAAABYQAAAAAAAAAAAQQAAAABYgAAAAAAAAAAAgQAAAABYwAAAAAAAAAAAwQAAAABZAAAAAAAAAAABAQAAAAJJHQwMTI2MTUzCQAFFAAAAAIJAABkAAAAAgUAAAABYQUAAAABYgkAAGQAAAACBQAAAAFjBQAAAAFkBAAAAAF4CAUAAAAJJHQwMTI2MTUzAAAAAl8xBAAAAAF5CAUAAAAJJHQwMTI2MTUzAAAAAl8yCQAAAAAAAAIJAABkAAAAAgUAAAABeAUAAAABeQAAAAAAAAAACrqIL8U=`, env, true},
		{`V4: func A() = (123, "xxx"); A()._1 == 123`, `BAoBAAAAAUEAAAAACQAFFAAAAAIAAAAAAAAAAHsCAAAAA3h4eAkAAAAAAAACCAkBAAAAAUEAAAAAAAAAAl8xAAAAAAAAAAB7Ge9bZg==`, env, true},
		{`V4: func A() = (123, "xxx"); func B() = A()._1; B() == 123`, `BAoBAAAAAUEAAAAACQAFFAAAAAIAAAAAAAAAAHsCAAAAA3h4eAoBAAAAAUIAAAAACAkBAAAAAUEAAAAAAAAAAl8xCQAAAAAAAAIJAQAAAAFCAAAAAAAAAAAAAAAAe7iMtSQ=`, env, true},
	} {
		src, err := base64.StdEncoding.DecodeString(test.source)
		require.NoError(t, err, test.comment)

		tree, err := Parse(src)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, tree, test.comment)

		tree = MustExpand(tree)
		require.True(t, tree.Expanded)

		script, err := CompileTree("", tree)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, script, test.comment)

		res, err := script.Verify(test.env)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, res, test.comment)
		r, ok := res.(ScriptResult)
		assert.True(t, ok, test.comment)
		assert.Equal(t, test.res, r.Result(), test.comment)
	}
}

func TestCallExternal(t *testing.T) {
	// 1 == 1
	n := &FunctionCallNode{
		Name: "0",
		Arguments: []Node{
			&LongNode{
				Value: 1,
			},
			&LongNode{
				Value: 1,
			},
		},
	}

	f, err := compileFunction("", 3, []Node{n}, false, true)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			OpReturn,
			OpReturn,
			OpRef, 0, 1,
			OpRef, 0, 1,
			OpExternalCall, 0, 3, 0, 2,
			OpReturn,
			OpReturn,
		},
		f.ByteCode)
}

func TestIfConditionRightByteCode(t *testing.T) {
	// let x = if (true) then true else false; x
	n := &AssignmentNode{
		Name: "x",
		Expression: &ConditionalNode{
			Condition:       &BooleanNode{Value: true},
			TrueExpression:  &BooleanNode{Value: true},
			FalseExpression: &BooleanNode{Value: false},
		},
		Block: &ReferenceNode{
			Name: "x",
		},
	}

	f, err := compileFunction("", 3, []Node{n}, false, true)
	require.NoError(t, err)

	rs, err := f.Verify(nil)
	require.NoError(t, err)
	require.Equal(t, true, rs.Result())
}

func TestCall(t *testing.T) {
	// let i = 1; let s = "string"; toString(i) == s
	source := `BAQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABc6Y8UOc=`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	rs, err := script.Verify(nil)
	require.NoError(t, err)
	require.Equal(t, 2, len(rs.Calls()))
	require.Equal(t, false, rs.Result())
}

func TestDoubleCall(t *testing.T) {
	//func a() = 1; a() == 1
	n := &FunctionDeclarationNode{
		Name:      "a",
		Arguments: nil,
		Body: &LongNode{
			Value: 1,
		},
		Block: &FunctionCallNode{
			Name: "0",
			Arguments: []Node{
				&FunctionCallNode{
					Name:      "a",
					Arguments: nil,
				},
				&LongNode{
					Value: 1,
				},
			},
		},
	}

	f, err := compileFunction("", 3, []Node{n}, false, true)
	require.NoError(t, err)

	rs, err := f.Verify(nil)
	require.NoError(t, err)
	require.Equal(t, true, rs.Result())
}

func TestCallWithConstArg(t *testing.T) {
	// func id(v: Boolean) = v; id(true)
	n := &FunctionDeclarationNode{
		Name:      "id",
		Arguments: []string{"v"},
		Body:      &ReferenceNode{Name: "v"},
		Block: &FunctionCallNode{
			Name: "id",
			Arguments: []Node{
				&BooleanNode{
					Value: true,
				},
			},
		},
		invocationParameter: "",
	}

	f, err := compileFunction("", 3, []Node{n}, false, true)
	require.NoError(t, err)

	rs, err := f.Verify(nil)
	require.NoError(t, err)
	require.Equal(t, true, rs.Result())
}

func TestMultipleCallConstantFuncArgument(t *testing.T) {
	// func id(v: Boolean) = v && v; id(true)
	source := `BAoBAAAAAmlkAAAAAQAAAAF2AwUAAAABdgUAAAABdgcJAQAAAAJpZAAAAAEG3g2xRQ==`

	state := &MockSmartState{NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
		return byte_helpers.TransferWithProofs.Transaction, nil
	}}
	env := &MockRideEnvironment{
		transactionFunc: testTransferObject,
		stateFunc: func() types.SmartState {
			return state
		},
		schemeFunc: func() byte {
			return 'T'
		},
		checkMessageLengthFunc: func(in1 int) bool {
			return true
		},
	}

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	res, err := script.Verify(env)
	require.NoError(t, err)
	assert.NotNil(t, res)
	r, ok := res.(ScriptResult)
	assert.True(t, ok)
	assert.Equal(t, true, r.Result())
}

func TestIfStmt(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 4 #-}
	   {-# CONTENT_TYPE EXPRESSION #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}
	   func id(v: Boolean) = {
	       if (v) then {
	           let x = throw("a")
	           1
	       } else {
	           let x = throw("b")
	           2
	       }
	   }
	   1 == id(true)
	*/
	source := `BAoBAAAAAmlkAAAAAQAAAAF2AwUAAAABdgQAAAABeAkAAAIAAAABAgAAAAFhAAAAAAAAAAABBAAAAAF4CQAAAgAAAAECAAAAAWIAAAAAAAAAAAIJAAAAAAAAAgAAAAAAAAAAAQkBAAAAAmlkAAAAAQYYAiEb`
	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
	}
	env := &MockRideEnvironment{
		transactionFunc: testTransferObject,
		stateFunc: func() types.SmartState {
			return state
		},
		schemeFunc: func() byte {
			return 'T'
		},
		checkMessageLengthFunc: func(in1 int) bool {
			return true
		},
	}

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	require.NotNil(t, script)

	res, err := script.Verify(env)
	require.NoError(t, err)
	require.NotNil(t, res)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	assert.Equal(t, true, r.Result())
}

func TestDappMultipleFunctions(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 3 #-}
	   {-# CONTENT_TYPE DAPP #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}
	   @Callable(i)
	   func abc(question: String) = {
	       WriteSet([DataEntry("a", 5)])
	   }
	   @Callable(i)
	   func cba(question: String) = {
	       WriteSet([DataEntry("a", 6)])
	   }
	*/
	source := "AAIDAAAAAAAAAAwIARIDCgEIEgMKAQgAAAAAAAAAAgAAAAFpAQAAAANhYmMAAAABAAAACHF1ZXN0aW9uCQEAAAAIV3JpdGVTZXQAAAABCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACAgAAAAFhAAAAAAAAAAAFBQAAAANuaWwAAAABaQEAAAADY2JhAAAAAQAAAAhxdWVzdGlvbgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAABYQAAAAAAAAAABgUAAAADbmlsAAAAAFEpRso="
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileDapp("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	rs, err := script.Invoke(defaultEnv, "abc", []rideType{rideString("")})
	require.NoError(t, err)

	require.Equal(t, true, rs.Result())
	require.Equal(t,
		[]proto.ScriptAction{
			&proto.DataEntryScriptAction{
				Entry: &proto.IntegerDataEntry{Value: 5, Key: "a"},
			},
		}, []proto.ScriptAction(rs.ScriptActions()))

	rs, err = script.Invoke(defaultEnv, "cba", []rideType{rideString("")})
	require.NoError(t, err)

	require.Equal(t, true, rs.Result())
	require.Equal(t,
		[]proto.ScriptAction{
			&proto.DataEntryScriptAction{
				Entry: &proto.IntegerDataEntry{Value: 6, Key: "a"},
			},
		}, []proto.ScriptAction(rs.ScriptActions()))
}

func TestList(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		let oraclesList = ["3MSNMcqyweiM9cWpvf4Fn8GAWeuPstxj2hK"]
		func getNumberByAddressAndKey(address: Address, key: Int) = 0
		func getOracleProvideHeight(owner: String, height: Int) = getNumberByAddressAndKey(addressFromStringValue(owner), height)
		getOracleProvideHeight(oraclesList[0], height) == 1
	*/
	source := `BAQAAAALb3JhY2xlc0xpc3QJAARMAAAAAgIAAAAjM01TTk1jcXl3ZWlNOWNXcHZmNEZuOEdBV2V1UHN0eGoyaEsFAAAAA25pbAoBAAAAGGdldE51bWJlckJ5QWRkcmVzc0FuZEtleQAAAAIAAAAHYWRkcmVzcwAAAANrZXkAAAAAAAAAAAAKAQAAABZnZXRPcmFjbGVQcm92aWRlSGVpZ2h0AAAAAgAAAAVvd25lcgAAAAZoZWlnaHQJAQAAABhnZXROdW1iZXJCeUFkZHJlc3NBbmRLZXkAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA2MikAAAABBQAAAAVvd25lcgUAAAAGaGVpZ2h0CQAAAAAAAAIJAQAAABZnZXRPcmFjbGVQcm92aWRlSGVpZ2h0AAAAAgkAAZEAAAACBQAAAAtvcmFjbGVzTGlzdAAAAAAAAAAAAAUAAAAGaGVpZ2h0AAAAAAAAAAABHUHhjA==`

	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
			t.Log("key: ", key)
			return nil, errors.New("not found")
		},
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			t.Logf("acc: %q, key %s", account, key)
			return &proto.IntegerDataEntry{
				Value: 0,
			}, nil
		},
	}
	env := &MockRideEnvironment{
		transactionFunc: testTransferObject,
		stateFunc: func() types.SmartState {
			return state
		},
		schemeFunc: func() byte {
			return 'T'
		},
		thisFunc: func() rideType {
			return rideAddress{}
		},
		heightFunc: func() rideInt {
			return 100500
		},
	}

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	res, err := script.Verify(env)
	require.NoError(t, err)
	assert.NotNil(t, res)
	r, ok := res.(ScriptResult)
	assert.True(t, ok)
	assert.Equal(t, false, r.Result())
}

func TestFunctionCallInline(t *testing.T) {
	/*
	   func abc() = 5
	   func cba() = 10
	   if abc() == cba() then {
	       true
	   } else {
	       false
	   }
	*/
	source := `BAoBAAAAA2FiYwAAAAAAAAAAAAAAAAUKAQAAAANjYmEAAAAAAAAAAAAAAAAKAwkAAAAAAAACCQEAAAADYWJjAAAAAAkBAAAAA2NiYQAAAAAGB0hjUOM=`

	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
			t.Log("key: ", key)
			return nil, errors.New("not found")
		},
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			t.Logf("acc: %q, key %s", account, key)
			return &proto.IntegerDataEntry{
				Value: 0,
			}, nil
		},
	}
	env := &MockRideEnvironment{
		transactionFunc: testTransferObject,
		stateFunc: func() types.SmartState {
			return state
		},
		schemeFunc: func() byte {
			return 'T'
		},
		thisFunc: func() rideType {
			return rideAddress{}
		},
		heightFunc: func() rideInt {
			return 1
		},
	}

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)
	rs, _ := script.Verify(env)
	require.Equal(t, rs.Result(), false)
}

func TestNoDuplicateCallToState(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 3 #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}
	   {-# CONTENT_TYPE DAPP #-}
	   func getStringByAddressAndKey(address: Address, key: String) = match getString(address, key) {
	       case a: String =>
	           a
	       case _ =>
	           ""
	   }
	   func getStringByKey(key: String) = match getString(this, key) {
	       case a: String =>
	           a
	       case _ =>
	           ""
	   }
	   let LastConfirmTxKey = "last_confirm_tx"
	   let NeutrinoContractKey = "neutrino_contract"
	   let ControlContractKey = "control_contract"
	   let neutrinoContract = addressFromStringValue(getStringByKey(NeutrinoContractKey))
	   let controlContract = addressFromStringValue(getStringByAddressAndKey(neutrinoContract, ControlContractKey))
	   let lastConfirmTx = getStringByAddressAndKey(controlContract, LastConfirmTxKey)
	   @Verifier(tx)
	   func verify () = (lastConfirmTx == toBase58String(tx.id))
	*/
	source := `AAIDAAAAAAAAAAIIAQAAAAgBAAAAGGdldFN0cmluZ0J5QWRkcmVzc0FuZEtleQAAAAIAAAAHYWRkcmVzcwAAAANrZXkEAAAAByRtYXRjaDAJAAQdAAAAAgUAAAAHYWRkcmVzcwUAAAADa2V5AwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAAZTdHJpbmcEAAAAAWEFAAAAByRtYXRjaDAFAAAAAWECAAAAAAEAAAAOZ2V0U3RyaW5nQnlLZXkAAAABAAAAA2tleQQAAAAHJG1hdGNoMAkABB0AAAACBQAAAAR0aGlzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABlN0cmluZwQAAAABYQUAAAAHJG1hdGNoMAUAAAABYQIAAAAAAAAAABBMYXN0Q29uZmlybVR4S2V5AgAAAA9sYXN0X2NvbmZpcm1fdHgAAAAAE05ldXRyaW5vQ29udHJhY3RLZXkCAAAAEW5ldXRyaW5vX2NvbnRyYWN0AAAAABJDb250cm9sQ29udHJhY3RLZXkCAAAAEGNvbnRyb2xfY29udHJhY3QAAAAAEG5ldXRyaW5vQ29udHJhY3QJAQAAABxAZXh0clVzZXIoYWRkcmVzc0Zyb21TdHJpbmcpAAAAAQkBAAAADmdldFN0cmluZ0J5S2V5AAAAAQUAAAATTmV1dHJpbm9Db250cmFjdEtleQAAAAAPY29udHJvbENvbnRyYWN0CQEAAAAcQGV4dHJVc2VyKGFkZHJlc3NGcm9tU3RyaW5nKQAAAAEJAQAAABhnZXRTdHJpbmdCeUFkZHJlc3NBbmRLZXkAAAACBQAAABBuZXV0cmlub0NvbnRyYWN0BQAAABJDb250cm9sQ29udHJhY3RLZXkAAAAADWxhc3RDb25maXJtVHgJAQAAABhnZXRTdHJpbmdCeUFkZHJlc3NBbmRLZXkAAAACBQAAAA9jb250cm9sQ29udHJhY3QFAAAAEExhc3RDb25maXJtVHhLZXkAAAAAAAAAAQAAAAJ0eAEAAAAGdmVyaWZ5AAAAAAkAAAAAAAACBQAAAA1sYXN0Q29uZmlybVR4CQACWAAAAAEIBQAAAAJ0eAAAAAJpZJO+lgc=`

	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
			switch key {
			case "neutrino_contract":
				return &proto.StringDataEntry{Value: "3MVHscMp4C3JjeaEiZB6fxeomPZdYEHyamY"}, nil
			case "control_contract":
				return &proto.StringDataEntry{Value: "3MQdbE6dK59FHxh5rf4biQdyXhdEf3L1R5W"}, nil
			case "last_confirm_tx":
				return &proto.StringDataEntry{Value: "3M9uzVzrAAYEKSHXzKaPhw7iQjwDi9BRJysHZHpbqXJm"}, nil

			}
			panic(key)
		},
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			v, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				return nil, err
			}
			return &proto.IntegerDataEntry{
				Value: v,
			}, nil
		},
	}
	env := &MockRideEnvironment{
		transactionFunc: testTransferObject,
		stateFunc: func() types.SmartState {
			return state
		},
		schemeFunc: func() byte {
			return 'S'
		},
		thisFunc: func() rideType {
			b := [26]byte{1, 83, 122, 149, 83, 66, 227, 147, 59, 198, 33, 214, 105, 255, 17, 4, 168, 100, 213, 112, 143, 31, 192, 98, 166, 126}
			return rideAddress(b)
		},
		heightFunc: func() rideInt {
			return 1
		},
	}

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.NoError(t, err)
	require.False(t, rs.Result())
}

func TestDappVerifyVm(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 3 #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}
	   {-# CONTENT_TYPE DAPP #-}
	   @Verifier(tx)
	   func verify () = sigVerify(tx.bodyBytes, tx.proofs[0], tx.senderPublicKey)
	*/
	source := `AAIDAAAAAAAAAAIIAQAAAAAAAAAAAAAAAQAAAAJ0eAEAAAAGdmVyaWZ5AAAAAAkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAAIBQAAAAJ0eAAAAA9zZW5kZXJQdWJsaWNLZXlQ99ml`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	env := &MockRideEnvironment{
		transactionFunc: testTransferObject,
		schemeFunc: func() byte {
			return 'S'
		},
		checkMessageLengthFunc: func(in1 int) bool {
			return true
		},
	}

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.Equal(t, rs.Result(), true)
}

func TestMultipleProperty(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 3 #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}
	   {-# CONTENT_TYPE EXPRESSION #-}

	   match (tx) {
	       case e:ExchangeTransaction => isDefined(e.sellOrder.assetPair.priceAsset)
	       case _ => throw("err")
	     }
	*/source := `AwQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE0V4Y2hhbmdlVHJhbnNhY3Rpb24EAAAAAWUFAAAAByRtYXRjaDAJAQAAAAlpc0RlZmluZWQAAAABCAgIBQAAAAFlAAAACXNlbGxPcmRlcgAAAAlhc3NldFBhaXIAAAAKcHJpY2VBc3NldAkAAAIAAAABAgAAAANlcnIsqB0K`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
	}

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.Equal(t, rs.Result(), true)
}

func TestProperty(t *testing.T) {
	t.Run("test simple property", func(t *testing.T) {
		n := &PropertyNode{
			Name:   "id",
			Object: &ReferenceNode{Name: "tx"},
		}
		tree := &Tree{
			LibVersion: 3,
			AppVersion: scriptApplicationVersion,
			Verifier:   n,
		}

		script, err := CompileVerifier("", tree)
		require.NoError(t, err)
		assert.NotNil(t, script)

		env := &MockRideEnvironment{
			transactionFunc: testExchangeWithProofsToObject,
		}

		require.Equal(t,
			[]byte{
				OpReturn,
				OpReturn,
				OpRef, 255, 255,
				OpRef, 0, 202,
				OpProperty,
				OpReturn,
				OpReturn,
				OpReturn,
			},
			script.ByteCode)
		_, err = script.run(env, nil)
		require.NoError(t, err)
	})
	t.Run("test multiple property", func(t *testing.T) {
		n := &PropertyNode{
			Name: "assetPair",
			Object: &PropertyNode{
				Name:   "sellOrder",
				Object: &ReferenceNode{Name: "tx"},
			}}
		tree := &Tree{
			LibVersion: 3,
			AppVersion: scriptApplicationVersion,
			Verifier:   n,
		}

		script, err := CompileVerifier("", tree)
		require.NoError(t, err)
		assert.NotNil(t, script)

		env := &MockRideEnvironment{
			transactionFunc: testExchangeWithProofsToObject,
		}

		require.Equal(t,
			[]byte{
				OpReturn,
				OpReturn,
				OpRef, 0, 203,
				OpRef, 0, 204,
				OpProperty,
				OpReturn,
				OpReturn,
				OpReturn,

				OpRef, 255, 255,
				OpRef, 0, 205,
				OpProperty,
				OpReturn,
				OpReturn,
			},
			script.ByteCode)
		_, err = script.run(env, nil)
		require.NoError(t, err)
	})
}

func TestCacheInMain(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 4 #-}
	   {-# CONTENT_TYPE EXPRESSION #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}

	   let x = 1 + 1
	   x == x
	*/
	source := `BAQAAAABeAkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABCQAAAAAAAAIFAAAAAXgFAAAAAXgu3TzS`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
	}

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.Equal(t, 2, len(rs.Calls())) // plus & eq
	require.Equal(t, rs.Result(), true)
}

func TestCacheFunctionArgumentsCalls(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 4 #-}
	   {-# CONTENT_TYPE EXPRESSION #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}
	   func abc() = {
	       1 + 1
	   }
	   let info = abc()
	   info == info
	*/
	source := `BAoBAAAAA2FiYwAAAAAJAABkAAAAAgAAAAAAAAAAAQAAAAAAAAAAAQQAAAAEaW5mbwkBAAAAA2FiYwAAAAAJAAAAAAAAAgUAAAAEaW5mbwUAAAAEaW5mby35E+E=`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
	}

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.Equal(t, 2, len(rs.Calls()))
	require.Equal(t, true, rs.Result())
}

func TestCacheInFunc(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 4 #-}
	   {-# CONTENT_TYPE EXPRESSION #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}

	   func abc() = {
	       let x = 1 + 1
	       x == x
	   }
	   abc()
	*/
	source := `BAoBAAAAA2FiYwAAAAAEAAAAAXgJAABkAAAAAgAAAAAAAAAAAQAAAAAAAAAAAQkAAAAAAAACBQAAAAF4BQAAAAF4CQEAAAADYWJjAAAAAJz8J24=`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
	}

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.Equal(t, 2, len(rs.Calls()))
	require.Equal(t, rs.Result(), true)
}

func TestCacheFuncArgs(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 4 #-}
	   {-# CONTENT_TYPE EXPRESSION #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}
	   func abc(x: Int) = x == x
	   let y = getIntegerValue(this, "a")
	   abc(y)
	*/
	source := `BAoBAAAAA2FiYwAAAAEAAAABeAkAAAAAAAACBQAAAAF4BQAAAAF4BAAAAAF5CQEAAAARQGV4dHJOYXRpdmUoMTA1MCkAAAACBQAAAAR0aGlzAgAAAAFhCQEAAAADYWJjAAAAAQUAAAABeYsrE7g=`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)
	state := &MockSmartState{
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			return &proto.IntegerDataEntry{
				Value: 1,
			}, nil
		},
	}
	env := &MockRideEnvironment{
		stateFunc: func() types.SmartState {
			return state
		},
		thisFunc: func() rideType {
			return rideAddress{}
		},
	}

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.Equal(t, true, rs.Result())
	require.Equal(t, 2, len(rs.Calls()))
	// only 1 native call to state
	require.Equal(t, "@extrNative(1050)", rs.Calls()[0].name)
	// native compare
	require.Equal(t, "0", rs.Calls()[1].name)
}

func TestLetInLet(t *testing.T) {
	/*
	   let x = {
	   	let y = true;
	   	y;
	   }
	   x
	*/
	source := `BAQAAAABeAQAAAABeQYFAAAAAXkFAAAAAXhCPj2C`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
	}

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.Equal(t, true, rs.Result())
}

func TestDApp(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 3 #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}
	   {-# CONTENT_TYPE DAPP #-}
	   @Callable(i)
	   func deposit () = {
	       let pmt = extract(i.payment)
	       if (isDefined(pmt.assetId))
	           then throw("can hold waves only at the moment")
	           else {
	               let currentKey = toBase58String(i.caller.bytes)
	               let currentAmount =             match getInteger(this, currentKey) {
	                   case a: Int =>
	                       a
	                   case _ =>
	                       0
	               }
	               let newAmount = (currentAmount + pmt.amount)
	               WriteSet([DataEntry(currentKey, newAmount)])
	               }
	       }
	   @Callable(i)
	   func withdraw (amount) = {
	       let currentKey = toBase58String(i.caller.bytes)
	       let currentAmount =     match getInteger(this, currentKey) {
	           case a: Int =>
	               a
	           case _ =>
	               0
	       }
	       let newAmount = (currentAmount - amount)
	       if ((0 > amount))
	           then throw("Can't withdraw negative amount")
	           else if ((0 > newAmount))
	               then throw("Not enough balance")
	               else ScriptResult(WriteSet([DataEntry(currentKey, newAmount)]), TransferSet([ScriptTransfer(i.caller, amount, unit)]))
	       }
	   @Verifier(tx)
	   func verify () = sigVerify(tx.bodyBytes, tx.proofs[0], tx.senderPublicKey)
	*/
	source := `AAIDAAAAAAAAAAkIARIAEgMKAQEAAAAAAAAAAgAAAAFpAQAAAAdkZXBvc2l0AAAAAAQAAAADcG10CQEAAAAHZXh0cmFjdAAAAAEIBQAAAAFpAAAAB3BheW1lbnQDCQEAAAAJaXNEZWZpbmVkAAAAAQgFAAAAA3BtdAAAAAdhc3NldElkCQAAAgAAAAECAAAAIWNhbiBob2xkIHdhdmVzIG9ubHkgYXQgdGhlIG1vbWVudAQAAAAKY3VycmVudEtleQkAAlgAAAABCAgFAAAAAWkAAAAGY2FsbGVyAAAABWJ5dGVzBAAAAA1jdXJyZW50QW1vdW50BAAAAAckbWF0Y2gwCQAEGgAAAAIFAAAABHRoaXMFAAAACmN1cnJlbnRLZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAA0ludAQAAAABYQUAAAAHJG1hdGNoMAUAAAABYQAAAAAAAAAAAAQAAAAJbmV3QW1vdW50CQAAZAAAAAIFAAAADWN1cnJlbnRBbW91bnQIBQAAAANwbXQAAAAGYW1vdW50CQEAAAAIV3JpdGVTZXQAAAABCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACBQAAAApjdXJyZW50S2V5BQAAAAluZXdBbW91bnQFAAAAA25pbAAAAAFpAQAAAAh3aXRoZHJhdwAAAAEAAAAGYW1vdW50BAAAAApjdXJyZW50S2V5CQACWAAAAAEICAUAAAABaQAAAAZjYWxsZXIAAAAFYnl0ZXMEAAAADWN1cnJlbnRBbW91bnQEAAAAByRtYXRjaDAJAAQaAAAAAgUAAAAEdGhpcwUAAAAKY3VycmVudEtleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAFhBQAAAAckbWF0Y2gwBQAAAAFhAAAAAAAAAAAABAAAAAluZXdBbW91bnQJAABlAAAAAgUAAAANY3VycmVudEFtb3VudAUAAAAGYW1vdW50AwkAAGYAAAACAAAAAAAAAAAABQAAAAZhbW91bnQJAAACAAAAAQIAAAAeQ2FuJ3Qgd2l0aGRyYXcgbmVnYXRpdmUgYW1vdW50AwkAAGYAAAACAAAAAAAAAAAABQAAAAluZXdBbW91bnQJAAACAAAAAQIAAAASTm90IGVub3VnaCBiYWxhbmNlCQEAAAAMU2NyaXB0UmVzdWx0AAAAAgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgUAAAAKY3VycmVudEtleQUAAAAJbmV3QW1vdW50BQAAAANuaWwJAQAAAAtUcmFuc2ZlclNldAAAAAEJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyBQAAAAZhbW91bnQFAAAABHVuaXQFAAAAA25pbAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAACAUAAAACdHgAAAAPc2VuZGVyUHVibGljS2V54232jg==`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileTree("", tree)
	require.NoError(t, err)
	require.NotNil(t, script)
	require.Equal(t, 4, len(script.EntryPoints))
}

func BenchmarkVm(b *testing.B) {
	source := "BAoBAAAABmdldEludAAAAAEAAAADa2V5BAAAAAckbWF0Y2gwCQAEGgAAAAIFAAAABHRoaXMFAAAAA2tleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAF4BQAAAAckbWF0Y2gwBQAAAAF4AAAAAAAAAAAABAAAAAFhCQEAAAAGZ2V0SW50AAAAAQIAAAABNQQAAAABYgkBAAAABmdldEludAAAAAECAAAAATYJAAAAAAAAAgUAAAABYQUAAAABYkOIJQA="
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(b, err)

	tree, err := Parse(src)
	require.NoError(b, err)

	script, err := CompileTree("", tree)
	require.NoError(b, err)

	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
			return nil, errors.New("not found")
		},
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			v, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				return nil, err
			}
			return &proto.IntegerDataEntry{
				Value: v,
			}, nil
		},
	}

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
		checkMessageLengthFunc: func(in1 int) bool {
			return true
		},
		stateFunc: func() types.SmartState {
			return state
		},
		thisFunc: func() rideType {
			return rideAddress{}
		},
	}

	b.ReportAllocs()
	b.StopTimer()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		rs, err := script.Verify(env)
		require.NoError(b, err)
		require.Equal(b, 5, len(rs.Calls()))
		b.StopTimer()
		require.NoError(b, err)
	}
}

func BenchmarkTree(b *testing.B) {
	source := "BAoBAAAABmdldEludAAAAAEAAAADa2V5BAAAAAckbWF0Y2gwCQAEGgAAAAIFAAAABHRoaXMFAAAAA2tleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAF4BQAAAAckbWF0Y2gwBQAAAAF4AAAAAAAAAAAABAAAAAFhCQEAAAAGZ2V0SW50AAAAAQIAAAABNQQAAAABYgkBAAAABmdldEludAAAAAECAAAAATYJAAAAAAAAAgUAAAABYQUAAAABYkOIJQA="
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(b, err)

	tree, err := Parse(src)
	require.NoError(b, err)

	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
			return nil, errors.New("not found")
		},
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			v, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				return nil, err
			}
			return &proto.IntegerDataEntry{
				Value: v,
			}, nil
		},
	}

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
		checkMessageLengthFunc: func(in1 int) bool {
			return true
		},
		stateFunc: func() types.SmartState {
			return state
		},
		thisFunc: func() rideType {
			return rideAddress{}
		},
	}

	b.ReportAllocs()
	b.StopTimer()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		rs, err := CallTreeVerifier(env, tree)
		b.StopTimer()
		require.NoError(b, err)
		require.Equal(b, 5, len(rs.Calls()))

	}
}

func BenchmarkVmWithDeserialize(b *testing.B) {
	source := "BAoBAAAABmdldEludAAAAAEAAAADa2V5BAAAAAckbWF0Y2gwCQAEGgAAAAIFAAAABHRoaXMFAAAAA2tleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAF4BQAAAAckbWF0Y2gwBQAAAAF4AAAAAAAAAAAABAAAAAFhCQEAAAAGZ2V0SW50AAAAAQIAAAABNQQAAAABYgkBAAAABmdldEludAAAAAECAAAAATYJAAAAAAAAAgUAAAABYQUAAAABYkOIJQA="
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(b, err)

	tree, err := Parse(src)
	require.NoError(b, err)

	script, err := CompileTree("", tree)
	require.NoError(b, err)

	ser := NewSerializer()
	err = script.Serialize(ser)
	require.NoError(b, err)

	bts := ser.Source()

	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
			return nil, errors.New("not found")
		},
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			v, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				return nil, err
			}
			return &proto.IntegerDataEntry{
				Value: v,
			}, nil
		},
	}

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
		checkMessageLengthFunc: func(in1 int) bool {
			return true
		},
		stateFunc: func() types.SmartState {
			return state
		},
		thisFunc: func() rideType {
			return rideAddress{}
		},
	}

	b.ReportAllocs()
	b.StopTimer()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		exe, err := DeserializeExecutable(bts)
		require.NoError(b, err)
		rs, err := exe.Verify(env)
		require.NoError(b, err)
		require.Equal(b, 5, len(rs.Calls()))
		b.StopTimer()
		require.NoError(b, err)
	}
}

func BenchmarkTreeWithDeserialize(b *testing.B) {
	source := "BAoBAAAABmdldEludAAAAAEAAAADa2V5BAAAAAckbWF0Y2gwCQAEGgAAAAIFAAAABHRoaXMFAAAAA2tleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAF4BQAAAAckbWF0Y2gwBQAAAAF4AAAAAAAAAAAABAAAAAFhCQEAAAAGZ2V0SW50AAAAAQIAAAABNQQAAAABYgkBAAAABmdldEludAAAAAECAAAAATYJAAAAAAAAAgUAAAABYQUAAAABYkOIJQA="
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(b, err)

	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
			return nil, errors.New("not found")
		},
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			v, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				return nil, err
			}
			return &proto.IntegerDataEntry{
				Value: v,
			}, nil
		},
	}

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
		checkMessageLengthFunc: func(in1 int) bool {
			return true
		},
		stateFunc: func() types.SmartState {
			return state
		},
		thisFunc: func() rideType {
			return rideAddress{}
		},
	}

	b.ReportAllocs()
	b.StopTimer()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		tree, err := Parse(src)
		require.NoError(b, err)
		rs, err := CallTreeVerifier(env, tree)
		b.StopTimer()
		require.NoError(b, err)
		require.Equal(b, 5, len(rs.Calls()))
	}
}

func TestFuncInCondState(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 4 #-}
	   {-# CONTENT_TYPE EXPRESSION #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}
	   if (true) then {
	       func a() = {
	           true
	       }
	       a()
	   } else {
	       func b() = {
	           false
	       }
	       b()
	   }
	*/
	source := `BAMGCgEAAAABYQAAAAAGCQEAAAABYQAAAAAKAQAAAAFiAAAAAAcJAQAAAAFiAAAAAObLaEQ=`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)
	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
			return nil, errors.New("not found")
		},
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			v, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				return nil, err
			}
			return &proto.IntegerDataEntry{
				Value: v,
			}, nil
		},
		NewestAccountBalanceFunc: func(account proto.Recipient, asset []byte) (uint64, error) {
			return 0, nil
		},
	}

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
		checkMessageLengthFunc: func(in1 int) bool {
			return true
		},
		stateFunc: func() types.SmartState {
			return state
		},
		thisFunc: func() rideType {
			return rideAddress{}
		},
	}

	tree, err := Parse(src)
	require.NoError(t, err)
	rs1, _ := CallTreeVerifier(env, tree)
	for i, c := range rs1.Calls() {
		t.Log(i, " ", c)
	}

	exe, err := CompileTree("", tree)
	require.NoError(t, err)

	rs2, err := exe.Verify(env)
	require.NoError(t, err)
	for i, c := range rs2.Calls() {
		t.Log(i, " ", c)
	}
	require.True(t, rs1.Eq(rs2))
}

func TestFuncNameAndParameterCollision(t *testing.T) {
	/*
		{-# STDLIB_VERSION 3 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		func inc(v: Int) = v + 1
		func call(inc: Int) = inc(inc)
		call(2) == 3
	*/
	source := `AwoBAAAAA2luYwAAAAEAAAABdgkAAGQAAAACBQAAAAF2AAAAAAAAAAABCgEAAAAEY2FsbAAAAAEAAAADaW5jCQEAAAADaW5jAAAAAQUAAAADaW5jCQAAAAAAAAIJAQAAAARjYWxsAAAAAQAAAAAAAAAAAgAAAAAAAAAAAxgTXMY=`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)
	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
			return nil, errors.New("not found")
		},
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			v, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				return nil, err
			}
			return &proto.IntegerDataEntry{
				Value: v,
			}, nil
		},
		NewestAccountBalanceFunc: func(account proto.Recipient, asset []byte) (uint64, error) {
			return 0, nil
		},
	}

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
		checkMessageLengthFunc: func(in1 int) bool {
			return true
		},
		stateFunc: func() types.SmartState {
			return state
		},
		thisFunc: func() rideType {
			return rideAddress{}
		},
		invocationFunc: func() rideObject {
			return rideObject{}
		},
	}

	tree, err := Parse(src)
	require.NoError(t, err)

	exe, err := CompileTree("", tree)
	require.NoError(t, err)
	rs, err := exe.Verify(env)
	require.NoError(t, err)
	assert.True(t, rs.Result())
}

func TestShadowedVariable(t *testing.T) {
	/*
	   let height = height
	   height != 0
	*/
	source := `AwoBAAAAD2dldFByaWNlSGlzdG9yeQAAAAEAAAAGaGVpZ2h0BQAAAAZoZWlnaHQJAQAAAAIhPQAAAAIJAQAAAA9nZXRQcmljZUhpc3RvcnkAAAABBQAAAAZoZWlnaHQAAAAAAAAAAADe0Skk`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	tree = MustExpand(tree)
	require.Equal(t, "(let height@getPriceHistory = { height }; height@getPriceHistory != 0)", DecompileTree(tree))

	script, err := CompileTree("", tree)
	require.NoError(t, err)

	rs, err := script.Verify(defaultEnv)
	require.NoError(t, err)
	require.Equal(t, true, rs.Result())
}

func TestShadowedVariableInConditionStmt(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 3 #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}
	   {-# CONTENT_TYPE EXPRESSION #-}

	   let prevOrder = false
	   func internal(prevOrder: Boolean) = {
	       if (prevOrder)
	           then false
	           else true
	   }
	   if (false)
	     then false
	     else internal(prevOrder)
	*/
	source := `AwQAAAAJcHJldk9yZGVyBwoBAAAACGludGVybmFsAAAAAQAAAAlwcmV2T3JkZXIDBQAAAAlwcmV2T3JkZXIHBgMHBwkBAAAACGludGVybmFsAAAAAQUAAAAJcHJldk9yZGVyxqI+QQ==`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	tree = MustExpand(tree)
	require.True(t, tree.Expanded)
	script, err := CompileTree("", tree)
	require.NoError(t, err)

	rs, err := script.Verify(defaultEnv)
	require.NoError(t, err)
	require.Equal(t, true, rs.Result())
}

func TestFailedCompilationOnPropertyAssignment(t *testing.T) {
	source := "AAIEAAAAAAAAABYIAhIAEgASBQoDCAgIEgASBQoDCAEBAAAARwEAAAARa2V5QWNjdW11bGF0ZWRGZWUAAAAAAgAAABIlc19fYWNjdW11bGF0ZWRGZWUBAAAADmtleVVjb2xsYXRlcmFsAAAAAAIAAAAPJXNfX3Vjb2xsYXRlcmFsAQAAABlrZXlUb3RhbExlbmRlZEF0T3RoZXJBY2NzAAAAAAIAAAAaJXNfX3RvdGFsTGVuZGVkQXRPdGhlckFjY3MBAAAAE2tleUFzc2V0TG9ja2VkVG90YWwAAAABAAAAB2Fzc2V0SWQJAAEsAAAAAgIAAAAYJXMlc19fYXNzZXRMb2NrZWRUb3RhbF9fBQAAAAdhc3NldElkAQAAABNrZXlBY2NvdW50T3BlcmF0aW9uAAAAAwAAAAx1bmxvY2tIZWlnaHQAAAAHYWRkcmVzcwAAAAZzdGF0dXMJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACAgAAAB4lcyVzJWQlc19fZGVmb0Fzc2V0T3BlcmF0aW9uX18FAAAAB2FkZHJlc3MCAAAAAl9fCQABpAAAAAEFAAAADHVubG9ja0hlaWdodAIAAAACX18FAAAABnN0YXR1cwEAAAAKa2V5RmFjdG9yeQAAAAACAAAACyVzX19mYWN0b3J5AQAAABprZXlMZW5kZWRBbW91bnRCeUFzc2V0Q29kZQAAAAEAAAAJYXNzZXRDb2RlCQABLAAAAAICAAAAHSVzJXNfX2xlbmRlZEJhc2VBc3NldEFtb3VudF9fBQAAAAlhc3NldENvZGUBAAAACGtleVByaWNlAAAAAQAAAAlhc3NldENvZGUJAAEsAAAAAgIAAAANJXMlc19fcHJpY2VfXwUAAAAJYXNzZXRDb2RlAAAAABRJZHhPcGVyYXRpb25BbW91bnRJbgAAAAAAAAAAAQAAAAATSWR4T3BlcmF0aW9uQXNzZXRJbgAAAAAAAAAAAgAAAAARSWR4T3BlcmF0aW9uUHJpY2UAAAAAAAAAAAMAAAAAFUlkeE9wZXJhdGlvbkFtb3VudE91dAAAAAAAAAAABAAAAAAUSWR4T3BlcmF0aW9uQXNzZXRPdXQAAAAAAAAAAAUBAAAAFmFzc2V0RGF0YVN3YXBPcGVyYXRpb24AAAAHAAAACGFtb3VudEluAAAAB2Fzc2V0SW4AAAAFcHJpY2UAAAAJYW1vdW50T3V0AAAACGFzc2V0T3V0AAAADGJydXR0b0Ftb3VudAAAAAlmZWVBbW91bnQJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgIAAAAQJWQlcyVkJXMlZCVkJWRfXwkAAaQAAAABBQAAAAhhbW91bnRJbgIAAAACX18FAAAAB2Fzc2V0SW4CAAAAAl9fCQABpAAAAAEFAAAACWFtb3VudE91dAIAAAACX18FAAAACGFzc2V0T3V0AgAAAAJfXwkAAaQAAAABBQAAAAVwcmljZQIAAAACX18JAAGkAAAAAQUAAAAMYnJ1dHRvQW1vdW50AgAAAAJfXwkAAaQAAAABBQAAAAlmZWVBbW91bnQBAAAAF2Fzc2V0RGF0YVJlYmFsYW5jZVRyYWNlAAAABQAAAA9kZWJ0b3JBc3NldENvZGUAAAAHZGVidFBtdAAAAAdiYXNlUG10AAAAD2xlbmRlZEFtdEJlZm9yZQAAAA5sZW5kZWRBbXRBZnRlcgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACAgAAABAlcyVzJWQlcyVkJWQlZF9fBQAAAA9kZWJ0b3JBc3NldENvZGUCAAAAAl9fCQACWAAAAAEJAQAAAAV2YWx1ZQAAAAEIBQAAAAdkZWJ0UG10AAAAB2Fzc2V0SWQCAAAAAl9fCQABpAAAAAEIBQAAAAdkZWJ0UG10AAAABmFtb3VudAIAAAACX18JAAJYAAAAAQkBAAAABXZhbHVlAAAAAQgFAAAAB2Jhc2VQbXQAAAAHYXNzZXRJZAIAAAACX18JAAGkAAAAAQgFAAAAB2Jhc2VQbXQAAAAGYW1vdW50AgAAAAJfXwkAAaQAAAABBQAAAA9sZW5kZWRBbXRCZWZvcmUCAAAAAl9fCQABpAAAAAEFAAAADmxlbmRlZEFtdEFmdGVyAQAAABxhc3NldFJlYWRTd2FwRGF0YUFycmF5T3JGYWlsAAAAAQAAAA9hY2NPcGVyYXRpb25LZXkEAAAAE2FjY09wZXJhdGlvbkRhdGFTdHIJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABB0AAAACBQAAAAR0aGlzBQAAAA9hY2NPcGVyYXRpb25LZXkJAAEsAAAAAgIAAAAqVGhlcmUgaXMgbm8gcmVxdWVzdCBmb3IgcGFzc2VkIGFyZ3VtZW50czogBQAAAA9hY2NPcGVyYXRpb25LZXkJAAS1AAAAAgUAAAATYWNjT3BlcmF0aW9uRGF0YVN0cgIAAAACX18AAAAAB251bGxJbnQA//////////8AAAAAB251bGxTdHICAAAABE5VTEwAAAAACmZhY3RvcnlBY2MJAQAAABFAZXh0ck5hdGl2ZSgxMDYyKQAAAAEJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABB0AAAACBQAAAAR0aGlzCQEAAAAKa2V5RmFjdG9yeQAAAAAJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAAEk5vIGNvbmZpZyBhdCB0aGlzPQkABCUAAAABBQAAAAR0aGlzAgAAAAkgZm9yIGtleT0JAQAAAAprZXlGYWN0b3J5AAAAAAEAAAAVa2V5RmFjdG9yeURlYnRBc3NldElkAAAAAAIAAAAfJXMlc19fY29tbW9uQ29uZmlnX19kZWJ0QXNzZXRJZAEAAAASa2V5RmFjdG9yeUFzc2V0Q2ZnAAAAAQAAAA9hc3NldEFkZHJlc3NTdHIJAAEsAAAAAgkAASwAAAACAgAAABMlcyVzJXNfX2RlZm9Bc3NldF9fBQAAAA9hc3NldEFkZHJlc3NTdHICAAAACF9fY29uZmlnAQAAABprZXlGYWN0b3J5QXNzZXRDdXJyZW50UG9vbAAAAAEAAAAPYXNzZXRBY2NBZGRyZXNzCQABLAAAAAIJAAEsAAAAAgIAAAATJXMlcyVzX19kZWZvQXNzZXRfXwkABCUAAAABBQAAAA9hc3NldEFjY0FkZHJlc3MCAAAADV9fY3VycmVudFBvb2wBAAAAIGtleUZhY3RvcnlEZWZvQWRkcmVzc0J5QXNzZXRDb2RlAAAAAQAAAAlhc3NldENvZGUJAAEsAAAAAgkAASwAAAACAgAAABMlcyVzJXNfX2RlZm9Bc3NldF9fBQAAAAlhc3NldENvZGUCAAAAFF9fYWRkcmVzc0J5QXNzZXRDb2RlAQAAABZmYWN0b3J5UmVhZERlYnRBc3NldElkAAAAAAkBAAAAE3ZhbHVlT3JFcnJvck1lc3NhZ2UAAAACCQAEHQAAAAIFAAAACmZhY3RvcnlBY2MJAQAAABVrZXlGYWN0b3J5RGVidEFzc2V0SWQAAAAACQABLAAAAAIJAAEsAAAAAgkAASwAAAACAgAAABVObyBjb25maWcgYXQgZmFjdG9yeT0JAAQlAAAAAQUAAAAKZmFjdG9yeUFjYwIAAAAJIGZvciBrZXk9CQEAAAAVa2V5RmFjdG9yeURlYnRBc3NldElkAAAAAAEAAAAcZmFjdG9yeVJlYWRBc3NldENmZ0J5QWRkcmVzcwAAAAEAAAAPYXNzZXRBZGRyZXNzU3RyCQAEtQAAAAIJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABB0AAAACBQAAAApmYWN0b3J5QWNjCQEAAAASa2V5RmFjdG9yeUFzc2V0Q2ZnAAAAAQUAAAAPYXNzZXRBZGRyZXNzU3RyCQABLAAAAAIJAAEsAAAAAgkAASwAAAACAgAAABVObyBjb25maWcgYXQgZmFjdG9yeT0JAAQlAAAAAQUAAAAKZmFjdG9yeUFjYwIAAAAJIGZvciBrZXk9CQEAAAASa2V5RmFjdG9yeUFzc2V0Q2ZnAAAAAQUAAAAPYXNzZXRBZGRyZXNzU3RyAgAAAAJfXwEAAAAZZmFjdG9yeVJlYWRBc3NldENmZ0J5Q29kZQAAAAEAAAAJYXNzZXRDb2RlBAAAAA9hc3NldEFkZHJlc3NTdHIJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABB0AAAACBQAAAApmYWN0b3J5QWNjCQEAAAAga2V5RmFjdG9yeURlZm9BZGRyZXNzQnlBc3NldENvZGUAAAABBQAAAAlhc3NldENvZGUJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAAFU5vIGNvbmZpZyBhdCBmYWN0b3J5PQkABCUAAAABBQAAAApmYWN0b3J5QWNjAgAAAAkgZm9yIGtleT0JAQAAACBrZXlGYWN0b3J5RGVmb0FkZHJlc3NCeUFzc2V0Q29kZQAAAAEFAAAACWFzc2V0Q29kZQkABRQAAAACBQAAAA9hc3NldEFkZHJlc3NTdHIJAQAAABxmYWN0b3J5UmVhZEFzc2V0Q2ZnQnlBZGRyZXNzAAAAAQUAAAAPYXNzZXRBZGRyZXNzU3RyAAAAABBJZHhEZWZvQXNzZXRDb2RlAAAAAAAAAAABAAAAAA5JZHhEZWZvQXNzZXRJZAAAAAAAAAAAAgAAAAASSWR4RGVmb0Fzc2V0U3RhdHVzAAAAAAAAAAADAAAAABBJZHhQcmljZURlY2ltYWxzAAAAAAAAAAAEAAAAAA5JZHhCYXNlQXNzZXRJZAAAAAAAAAAABQAAAAAYSWR4T3ZlckNvbGxhdGVyYWxQZXJjZW50AAAAAAAAAAAGAAAAAA5JZHhNaW5Jbml0UG9vbAAAAAAAAAAABwAAAAAVSWR4UHJpY2VPcmFjbGVBZGRyZXNzAAAAAAAAAAAIAAAAABBJZHhNaW5CdXlQYXltZW50AAAAAAAAAAAJAAAAABFJZHhNaW5TZWxsUGF5bWVudAAAAAAAAAAACgAAAAASSWR4QnV5TG9ja0ludGVydmFsAAAAAAAAAAALAAAAABNJZHhTZWxsTG9ja0ludGVydmFsAAAAAAAAAAAMAAAAABBJZHhCdXlGZWVQZXJjZW50AAAAAAAAAAANAAAAABFJZHhTZWxsRmVlUGVyY2VudAAAAAAAAAAADgAAAAAMdGhpc0NmZ0FycmF5CQEAAAAcZmFjdG9yeVJlYWRBc3NldENmZ0J5QWRkcmVzcwAAAAEJAAQlAAAAAQUAAAAEdGhpcwAAAAANZGVmb0Fzc2V0Q29kZQkAAZEAAAACBQAAAAx0aGlzQ2ZnQXJyYXkFAAAAEElkeERlZm9Bc3NldENvZGUAAAAAC2RlZm9Bc3NldElkCQACWQAAAAEJAAGRAAAAAgUAAAAMdGhpc0NmZ0FycmF5BQAAAA5JZHhEZWZvQXNzZXRJZAAAAAAOcHJpY2VPcmFjbGVBY2MJAQAAABFAZXh0ck5hdGl2ZSgxMDYyKQAAAAEJAAGRAAAAAgUAAAAMdGhpc0NmZ0FycmF5BQAAABVJZHhQcmljZU9yYWNsZUFkZHJlc3MAAAAAFW92ZXJDb2xsYXRlcmFsUGVyY2VudAkBAAAADXBhcnNlSW50VmFsdWUAAAABCQABkQAAAAIFAAAADHRoaXNDZmdBcnJheQUAAAAYSWR4T3ZlckNvbGxhdGVyYWxQZXJjZW50AAAAAA5iYXNlQXNzZXRJZFN0cgkAAZEAAAACBQAAAAx0aGlzQ2ZnQXJyYXkFAAAADklkeEJhc2VBc3NldElkAAAAAAtiYXNlQXNzZXRJZAkAAlkAAAABBQAAAA5iYXNlQXNzZXRJZFN0cgAAAAANcHJpY2VEZWNpbWFscwkBAAAADXBhcnNlSW50VmFsdWUAAAABCQABkQAAAAIFAAAADHRoaXNDZmdBcnJheQUAAAAQSWR4UHJpY2VEZWNpbWFscwAAAAARbWluQmFzaWNCdXlBbW91bnQJAQAAAA1wYXJzZUludFZhbHVlAAAAAQkAAZEAAAACBQAAAAx0aGlzQ2ZnQXJyYXkFAAAAEElkeE1pbkJ1eVBheW1lbnQAAAAAEm1pblN5bnRoU2VsbEFtb3VudAkBAAAADXBhcnNlSW50VmFsdWUAAAABCQABkQAAAAIFAAAADHRoaXNDZmdBcnJheQUAAAARSWR4TWluU2VsbFBheW1lbnQAAAAAD2J1eUxvY2tJbnRlcnZhbAkBAAAADXBhcnNlSW50VmFsdWUAAAABCQABkQAAAAIFAAAADHRoaXNDZmdBcnJheQUAAAASSWR4QnV5TG9ja0ludGVydmFsAAAAABBzZWxsTG9ja0ludGVydmFsCQEAAAANcGFyc2VJbnRWYWx1ZQAAAAEJAAGRAAAAAgUAAAAMdGhpc0NmZ0FycmF5BQAAABNJZHhTZWxsTG9ja0ludGVydmFsAAAAAA1idXlGZWVQZXJjZW50CQEAAAANcGFyc2VJbnRWYWx1ZQAAAAEJAAGRAAAAAgUAAAAMdGhpc0NmZ0FycmF5BQAAABBJZHhCdXlGZWVQZXJjZW50AAAAAA5zZWxsRmVlUGVyY2VudAkBAAAADXBhcnNlSW50VmFsdWUAAAABCQABkQAAAAIFAAAADHRoaXNDZmdBcnJheQUAAAARSWR4U2VsbEZlZVBlcmNlbnQBAAAAE2NvbnRyb2xBY2NSZWFkUHJpY2UAAAABAAAACWFzc2V0Q29kZQkBAAAAE3ZhbHVlT3JFcnJvck1lc3NhZ2UAAAACCQAEGgAAAAIFAAAADnByaWNlT3JhY2xlQWNjCQEAAAAIa2V5UHJpY2UAAAABBQAAAAlhc3NldENvZGUJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAAGE5vIHByaWNlIGF0IHByaWNlT3JhY2xlPQkABCUAAAABBQAAAA5wcmljZU9yYWNsZUFjYwIAAAAJIGZvciBrZXk9CQEAAAAIa2V5UHJpY2UAAAABBQAAAAlhc3NldENvZGUBAAAAG2NvbnRyb2xBY2NSZWFkQ3VycklkeE9yRmFpbAAAAAAJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABBoAAAACBQAAAA5wcmljZU9yYWNsZUFjYwIAAAAHY3VycklkeAkAASwAAAACAgAAABlObyBjdXJySWR4IGF0IGNvbnRyb2xBY2M9CQAEJQAAAAEFAAAADnByaWNlT3JhY2xlQWNjAQAAABdjb250cm9sQWNjUmVhZElkeEhlaWdodAAAAAEAAAADaWR4BAAAAAxpZHhIZWlnaHRLZXkJAAEsAAAAAgIAAAAKaWR4SGVpZ2h0XwkAAaQAAAABBQAAAANpZHgJAQAAAAt2YWx1ZU9yRWxzZQAAAAIJAAQaAAAAAgUAAAAOcHJpY2VPcmFjbGVBY2MFAAAADGlkeEhlaWdodEtleQAAAAAAAAAAAAEAAAAbY29udHJvbEFjY1JlYWRQcmljZUJ5SGVpZ2h0AAAAAQAAAAtwcmljZUhlaWdodAQAAAAQcHJpY2VCeUhlaWdodEtleQkAASwAAAACAgAAAAZwcmljZV8JAAGkAAAAAQUAAAALcHJpY2VIZWlnaHQJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABBoAAAACBQAAAA5wcmljZU9yYWNsZUFjYwUAAAAQcHJpY2VCeUhlaWdodEtleQkAASwAAAACCQABLAAAAAIJAAEsAAAAAgIAAAADTm8gBQAAABBwcmljZUJ5SGVpZ2h0S2V5AgAAAA8gYXQgY29udHJvbEFjYz0JAAQlAAAAAQUAAAAOcHJpY2VPcmFjbGVBY2MBAAAAEWdldFN0YWtpbmdCYWxhbmNlAAAAAAkBAAAAC3ZhbHVlT3JFbHNlAAAAAgkABBoAAAACBQAAAAR0aGlzCQABLAAAAAIJAAEsAAAAAgkAASwAAAACAgAAAAxycGRfYmFsYW5jZV8FAAAADmJhc2VBc3NldElkU3RyAgAAAAFfCQAEJQAAAAEFAAAABHRoaXMAAAAAAAAAAAAAAAAAC3Vjb2xsYXRlcmFsCQEAAAALdmFsdWVPckVsc2UAAAACCQAEGgAAAAIFAAAABHRoaXMJAQAAAA5rZXlVY29sbGF0ZXJhbAAAAAAAAAAAAAAAAAAAAAAADmFjY3VtdWxhdGVkRmVlCQEAAAALdmFsdWVPckVsc2UAAAACCQAEGgAAAAIFAAAABHRoaXMJAQAAABFrZXlBY2N1bXVsYXRlZEZlZQAAAAAAAAAAAAAAAAAAAAAADmN1cnJQb29sQW1vdW50CQEAAAARQGV4dHJOYXRpdmUoMTA1MCkAAAACBQAAAApmYWN0b3J5QWNjCQEAAAAaa2V5RmFjdG9yeUFzc2V0Q3VycmVudFBvb2wAAAABBQAAAAR0aGlzAAAAABdkb3VibGVDaGVja0Jhc2ljQmFsYW5jZQkAAGQAAAACCQAD8AAAAAIFAAAABHRoaXMFAAAAC2Jhc2VBc3NldElkCQEAAAARZ2V0U3Rha2luZ0JhbGFuY2UAAAAAAAAAABZkb3VibGVDaGVja1Vjb2xsYXRlcmFsAwkAAGYAAAACAAAAAAAAAAAABQAAAAt1Y29sbGF0ZXJhbAAAAAAAAAAAAAUAAAALdWNvbGxhdGVyYWwAAAAAGWRvdWJsZUNoZWNrQ3VyclBvb2xBbW91bnQJAABlAAAAAgUAAAAXZG91YmxlQ2hlY2tCYXNpY0JhbGFuY2UFAAAAC3Vjb2xsYXRlcmFsAAAAAAVwcmljZQkBAAAAE2NvbnRyb2xBY2NSZWFkUHJpY2UAAAABCQABkQAAAAIFAAAADHRoaXNDZmdBcnJheQUAAAAQSWR4RGVmb0Fzc2V0Q29kZQAAAAAJb3ZlclByaWNlCQAAaQAAAAIJAABoAAAAAgkAAGQAAAACBQAAAA1wcmljZURlY2ltYWxzBQAAABVvdmVyQ29sbGF0ZXJhbFBlcmNlbnQFAAAABXByaWNlBQAAAA1wcmljZURlY2ltYWxzAAAAAAhlbWlzc2lvbggJAQAAAAV2YWx1ZQAAAAEJAAPsAAAAAQUAAAALZGVmb0Fzc2V0SWQAAAAIcXVhbnRpdHkAAAAAE2Jhc2ljQXNzZXRMb2NrZWRBbXQJAQAAAAt2YWx1ZU9yRWxzZQAAAAIJAAQaAAAAAgUAAAAEdGhpcwkBAAAAE2tleUFzc2V0TG9ja2VkVG90YWwAAAABBQAAAA5iYXNlQXNzZXRJZFN0cgAAAAAAAAAAAAAAAAAUYXZhaWxhYmxlUG9vbEJhbGFuY2UJAABlAAAAAgUAAAAOY3VyclBvb2xBbW91bnQFAAAAE2Jhc2ljQXNzZXRMb2NrZWRBbXQBAAAAEGludGVybmFsQnV5QXNzZXQAAAAGAAAACnNlbGxlckFkZHIAAAAHc2VsbEFtdAAAAAtzZWxsQXNzZXRJZAAAAAptaW5TZWxsQW10AAAADWJ1eTJzZWxsUHJpY2UAAAAKZmVlUGVyY2VudAQAAAAYYXZhaWxhYmxlRGVmb0Fzc2V0SW5Qb29sCQAAZQAAAAIJAABrAAAAAwUAAAAUYXZhaWxhYmxlUG9vbEJhbGFuY2UFAAAADXByaWNlRGVjaW1hbHMFAAAACW92ZXJQcmljZQUAAAAIZW1pc3Npb24EAAAAHmZ1bGxEZWZvQXNzZXRBbW91bnROb1Bvb2xMaW1pdAkAAGsAAAADBQAAAAdzZWxsQW10BQAAAA1wcmljZURlY2ltYWxzBQAAAA1idXkyc2VsbFByaWNlBAAAABlmdWxsRGVmb0Fzc2V0QW1vdW50QnJ1dHRvAwkAAGYAAAACBQAAAB5mdWxsRGVmb0Fzc2V0QW1vdW50Tm9Qb29sTGltaXQFAAAAGGF2YWlsYWJsZURlZm9Bc3NldEluUG9vbAUAAAAYYXZhaWxhYmxlRGVmb0Fzc2V0SW5Qb29sBQAAAB5mdWxsRGVmb0Fzc2V0QW1vdW50Tm9Qb29sTGltaXQEAAAAD2RlZm9Bc3NldEFtb3VudAkAAGsAAAADCQAAZQAAAAIFAAAADXByaWNlRGVjaW1hbHMFAAAACmZlZVBlcmNlbnQFAAAAGWZ1bGxEZWZvQXNzZXRBbW91bnRCcnV0dG8FAAAADXByaWNlRGVjaW1hbHMEAAAACWZlZUFtb3VudAkAAGUAAAACBQAAABlmdWxsRGVmb0Fzc2V0QW1vdW50QnJ1dHRvBQAAAA9kZWZvQXNzZXRBbW91bnQEAAAAGHJlcXVpcmVkQmFzaWNBc3NldEFtb3VudAkAAGsAAAADBQAAABlmdWxsRGVmb0Fzc2V0QW1vdW50QnJ1dHRvBQAAAA1idXkyc2VsbFByaWNlBQAAAA1wcmljZURlY2ltYWxzBAAAAAZjaGFuZ2UJAABlAAAAAgUAAAAHc2VsbEFtdAUAAAAYcmVxdWlyZWRCYXNpY0Fzc2V0QW1vdW50AwkAAGcAAAACAAAAAAAAAAAABQAAABhhdmFpbGFibGVEZWZvQXNzZXRJblBvb2wJAAACAAAAAQkAASwAAAACCQABLAAAAAICAAAAGGltcG9zc2libGUgdG8gaXNzdWUgbmV3IAUAAAANZGVmb0Fzc2V0Q29kZQIAAAAXOiBub3QgZW5vdWdoIGNvbGxhdGVyYWwDCQAAZgAAAAIFAAAACm1pblNlbGxBbXQFAAAAB3NlbGxBbXQJAAACAAAAAQkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAAGGltcG9zc2libGUgdG8gaXNzdWUgbmV3IAUAAAANZGVmb0Fzc2V0Q29kZQIAAAAKOiBwYXltZW50PQkAAaQAAAABBQAAAAdzZWxsQW10AgAAABhpcyBsZXNzIHRoZW4gbWluIGFtb3VudD0JAAGkAAAAAQUAAAAKbWluU2VsbEFtdAkABRQAAAACCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACCQEAAAAOa2V5VWNvbGxhdGVyYWwAAAAACQAAZAAAAAIFAAAAC3Vjb2xsYXRlcmFsBQAAABhyZXF1aXJlZEJhc2ljQXNzZXRBbW91bnQJAARMAAAAAgkBAAAAC1N0cmluZ0VudHJ5AAAAAgkBAAAAE2tleUFjY291bnRPcGVyYXRpb24AAAADBQAAAAZoZWlnaHQJAAQlAAAAAQUAAAAKc2VsbGVyQWRkcgIAAAAIRklOSVNIRUQJAQAAABZhc3NldERhdGFTd2FwT3BlcmF0aW9uAAAABwUAAAAHc2VsbEFtdAkAAlgAAAABBQAAAAtzZWxsQXNzZXRJZAUAAAAFcHJpY2UFAAAAD2RlZm9Bc3NldEFtb3VudAkAAlgAAAABBQAAAAtkZWZvQXNzZXRJZAUAAAAZZnVsbERlZm9Bc3NldEFtb3VudEJydXR0bwUAAAAJZmVlQW1vdW50CQAETAAAAAIJAQAAAAdSZWlzc3VlAAAAAwUAAAALZGVmb0Fzc2V0SWQJAABkAAAAAgUAAAAPZGVmb0Fzc2V0QW1vdW50BQAAAAlmZWVBbW91bnQGCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMFAAAACnNlbGxlckFkZHIFAAAAD2RlZm9Bc3NldEFtb3VudAUAAAALZGVmb0Fzc2V0SWQJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwUAAAAKc2VsbGVyQWRkcgUAAAAGY2hhbmdlBQAAAAtzZWxsQXNzZXRJZAkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgkBAAAAEWtleUFjY3VtdWxhdGVkRmVlAAAAAAkAAGQAAAACBQAAAA5hY2N1bXVsYXRlZEZlZQUAAAAJZmVlQW1vdW50BQAAAANuaWwFAAAABmNoYW5nZQAAAAUAAAABaQEAAAAIYnV5QXNzZXQAAAAABAAAAANwbXQJAQAAAAV2YWx1ZQAAAAEJAAGRAAAAAggFAAAAAWkAAAAIcGF5bWVudHMAAAAAAAAAAAAEAAAACnBtdEFzc2V0SWQJAQAAAAV2YWx1ZQAAAAEIBQAAAANwbXQAAAAHYXNzZXRJZAMJAQAAAAIhPQAAAAIFAAAACnBtdEFzc2V0SWQFAAAAC2Jhc2VBc3NldElkCQAAAgAAAAEJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAANVBheW1lbnQgYXNzZXQgaWQgZG9lc24ndCBtYXRjaCBiYXNpYyBhc3NldDogZXhwZWN0ZWQ9CQACWAAAAAEFAAAAC2Jhc2VBc3NldElkAgAAAAggYWN0dWFsPQkAAlgAAAABBQAAAApwbXRBc3NldElkCAkBAAAAEGludGVybmFsQnV5QXNzZXQAAAAGCAUAAAABaQAAAAZjYWxsZXIIBQAAAANwbXQAAAAGYW1vdW50BQAAAApwbXRBc3NldElkBQAAABFtaW5CYXNpY0J1eUFtb3VudAUAAAAFcHJpY2UFAAAADWJ1eUZlZVBlcmNlbnQAAAACXzEAAAABaQEAAAAJc2VsbEFzc2V0AAAAAAQAAAADcG10CQEAAAAFdmFsdWUAAAABCQABkQAAAAIIBQAAAAFpAAAACHBheW1lbnRzAAAAAAAAAAAABAAAAAhwbXRBc3NldAkBAAAABXZhbHVlAAAAAQgFAAAAA3BtdAAAAAdhc3NldElkBAAAAA1jYWxsZXJBZGRyZXNzCQAEJQAAAAEIBQAAAAFpAAAABmNhbGxlcgMJAQAAAAIhPQAAAAIFAAAACHBtdEFzc2V0BQAAAAtkZWZvQXNzZXRJZAkAAAIAAAABCQABLAAAAAIJAAEsAAAAAgkAASwAAAACAgAAACNJbnZhbGlkIHBheW1lbnQgYXNzZXQgaWQ6IGV4cGVjdGVkPQkAAlgAAAABBQAAAAtkZWZvQXNzZXRJZAIAAAAIIGFjdHVhbD0JAAJYAAAAAQUAAAAIcG10QXNzZXQDCQAAZgAAAAIFAAAAEm1pblN5bnRoU2VsbEFtb3VudAgFAAAAA3BtdAAAAAZhbW91bnQJAAACAAAAAQkAASwAAAACCQABLAAAAAIJAAEsAAAAAgIAAAA6UGF5bWVudCBhbW91bnQgbGVzcyB0aGVuIG1pbmluaW1hbCBhbGxvd2VkOiBwYXltZW50QW1vdW50PQkAAaQAAAABCAUAAAADcG10AAAABmFtb3VudAIAAAALIG1pbkFtb3VudD0JAAGkAAAAAQUAAAASbWluU3ludGhTZWxsQW1vdW50BAAAABJwbXREZWZvQW1vdW50R3Jvc3MIBQAAAANwbXQAAAAGYW1vdW50BAAAAA1kZWZvQW1vdW50RmVlCQAAawAAAAMJAABlAAAAAgUAAAANcHJpY2VEZWNpbWFscwUAAAAOc2VsbEZlZVBlcmNlbnQFAAAAEnBtdERlZm9BbW91bnRHcm9zcwUAAAANcHJpY2VEZWNpbWFscwQAAAAScG10RGVmb0Ftb3VudE5vRmVlCQAAZQAAAAIFAAAAEnBtdERlZm9BbW91bnRHcm9zcwUAAAANZGVmb0Ftb3VudEZlZQQAAAAdYmFzZUFzc2V0QW1vdW50Tm9CYWxhbmNlTGltaXQJAABrAAAAAwUAAAAScG10RGVmb0Ftb3VudE5vRmVlBQAAAAVwcmljZQUAAAANcHJpY2VEZWNpbWFscwQAAAAUYmFzZUFzc2V0QW1vdW50Tm9GZWUDCQAAZgAAAAIFAAAAHWJhc2VBc3NldEFtb3VudE5vQmFsYW5jZUxpbWl0BQAAABlkb3VibGVDaGVja0N1cnJQb29sQW1vdW50BQAAABlkb3VibGVDaGVja0N1cnJQb29sQW1vdW50BQAAAB1iYXNlQXNzZXRBbW91bnROb0JhbGFuY2VMaW1pdAQAAAAXcmVxdWlyZWREZWZvQXNzZXRBbW91bnQJAABrAAAAAwUAAAAUYmFzZUFzc2V0QW1vdW50Tm9GZWUFAAAADXByaWNlRGVjaW1hbHMFAAAABXByaWNlBAAAAAZjaGFuZ2UJAABlAAAAAgUAAAAScG10RGVmb0Ftb3VudE5vRmVlBQAAABdyZXF1aXJlZERlZm9Bc3NldEFtb3VudAkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgkBAAAADmtleVVjb2xsYXRlcmFsAAAAAAkAAGUAAAACBQAAAAt1Y29sbGF0ZXJhbAUAAAAUYmFzZUFzc2V0QW1vdW50Tm9GZWUJAARMAAAAAgkBAAAAC1N0cmluZ0VudHJ5AAAAAgkBAAAAE2tleUFjY291bnRPcGVyYXRpb24AAAADBQAAAAZoZWlnaHQFAAAADWNhbGxlckFkZHJlc3MCAAAACEZJTklTSEVECQEAAAAWYXNzZXREYXRhU3dhcE9wZXJhdGlvbgAAAAcIBQAAAANwbXQAAAAGYW1vdW50CQACWAAAAAEFAAAACHBtdEFzc2V0BQAAAAVwcmljZQUAAAAUYmFzZUFzc2V0QW1vdW50Tm9GZWUFAAAADmJhc2VBc3NldElkU3RyBQAAABJwbXREZWZvQW1vdW50R3Jvc3MFAAAADWRlZm9BbW91bnRGZWUJAARMAAAAAgkBAAAABEJ1cm4AAAACBQAAAAtkZWZvQXNzZXRJZAUAAAAXcmVxdWlyZWREZWZvQXNzZXRBbW91bnQJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyBQAAABRiYXNlQXNzZXRBbW91bnROb0ZlZQUAAAALYmFzZUFzc2V0SWQJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyBQAAAAZjaGFuZ2UFAAAAC2RlZm9Bc3NldElkCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACCQEAAAARa2V5QWNjdW11bGF0ZWRGZWUAAAAACQAAZAAAAAIFAAAADmFjY3VtdWxhdGVkRmVlBQAAAA1kZWZvQW1vdW50RmVlBQAAAANuaWwAAAABaQEAAAANY3Jvc3NFeGNoYW5nZQAAAAMAAAAKYnV5QXNzZXRJZAAAABNidXlBc3NldENvZGVDb25maXJtAAAAFHNlbGxBc3NldENvZGVDb25maXJtBAAAAANwbXQJAQAAAAV2YWx1ZQAAAAEJAAGRAAAAAggFAAAAAWkAAAAIcGF5bWVudHMAAAAAAAAAAAAEAAAACHBtdEFzc2V0CQEAAAAFdmFsdWUAAAABCAUAAAADcG10AAAAB2Fzc2V0SWQEAAAAC3BtdEFzc2V0U3RyCQACWAAAAAEFAAAACHBtdEFzc2V0BAAAAAlwbXRBbW91bnQIBQAAAANwbXQAAAAGYW1vdW50BAAAAA1jYWxsZXJBZGRyZXNzCQAEJQAAAAEIBQAAAAFpAAAABmNhbGxlcgQAAAALYnV5QXNzZXRDZmcFAAAADHRoaXNDZmdBcnJheQQAAAAOc2VsbEFzc2V0VHVwbGUJAQAAABlmYWN0b3J5UmVhZEFzc2V0Q2ZnQnlDb2RlAAAAAQUAAAAUc2VsbEFzc2V0Q29kZUNvbmZpcm0EAAAADHNlbGxBc3NldENmZwgFAAAADnNlbGxBc3NldFR1cGxlAAAAAl8yBAAAABNzZWxsQXNzZXRBY2NBZGRyZXNzCQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAAQmAAAAAQgFAAAADnNlbGxBc3NldFR1cGxlAAAAAl8xCQABLAAAAAICAAAAMWNvdWxkbid0IHBhcnNlIGFkZHJlc3MgZnJvbSBzdHJpbmcgZm9yIGFzc2V0Q29kZT0FAAAAFHNlbGxBc3NldENvZGVDb25maXJtBAAAAAptaW5TZWxsUG10CQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAAS2AAAAAQkAAZEAAAACBQAAAAxzZWxsQXNzZXRDZmcFAAAAEUlkeE1pblNlbGxQYXltZW50CQABLAAAAAICAAAAIW1pblNlbGxQbXQgcGFyc2luZyBlcnJvcjogcmF3VmFsPQkAAZEAAAACBQAAAAxzZWxsQXNzZXRDZmcFAAAAEUlkeE1pblNlbGxQYXltZW50AwkBAAAAAiE9AAAAAgkAAZEAAAACBQAAAAx0aGlzQ2ZnQXJyYXkFAAAADklkeERlZm9Bc3NldElkBQAAAApidXlBc3NldElkCQAAAgAAAAEJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAAMGJ1eUFzc2V0IGNvbmZpcm1hdGlvbiBmYWlsZWQ6IGJ1eUFzc2V0SWRDb25maXJtPQkAAZEAAAACBQAAAAx0aGlzQ2ZnQXJyYXkFAAAADklkeERlZm9Bc3NldElkAgAAABAgQlVUIGJ1eUFzc2V0SWQ9BQAAAApidXlBc3NldElkAwkBAAAAAiE9AAAAAgkAAZEAAAACBQAAAAxzZWxsQXNzZXRDZmcFAAAADklkeERlZm9Bc3NldElkBQAAAAtwbXRBc3NldFN0cgkAAAIAAAABCQABLAAAAAIJAAEsAAAAAgkAASwAAAACAgAAADJzZWxsQXNzZXQgY29uZmlybWF0aW9uIGZhaWxlZDogc2VsbEFzc2V0SWRDb25maXJtPQkAAZEAAAACBQAAAAxzZWxsQXNzZXRDZmcFAAAADklkeERlZm9Bc3NldElkAgAAAA1CVVQgcG10QXNzZXQ9BQAAAAtwbXRBc3NldFN0cgMJAQAAAAIhPQAAAAIJAAGRAAAAAgUAAAAMdGhpc0NmZ0FycmF5BQAAABJJZHhEZWZvQXNzZXRTdGF0dXMCAAAABklTU1VFRAkAAAIAAAABCQABLAAAAAIJAAEsAAAAAgkAASwAAAACAgAAACx0b0Fzc2V0IGhhcyBub3QgYmVlbiBpc3N1ZWQgeWV0OiBidXlBc3NldElkPQUAAAAKYnV5QXNzZXRJZAIAAAAMIEJVVCBzdGF0dXM9CQABkQAAAAIFAAAADHRoaXNDZmdBcnJheQUAAAASSWR4RGVmb0Fzc2V0U3RhdHVzAwkBAAAAAiE9AAAAAgkAAZEAAAACBQAAAAxzZWxsQXNzZXRDZmcFAAAAEklkeERlZm9Bc3NldFN0YXR1cwIAAAAGSVNTVUVECQAAAgAAAAEJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAAMmZyb21Bc3NldENmZyBoYXMgbm90IGJlZW4gaXNzdWVkIHlldDogc2VsbEFzc2V0SWQ9BQAAAAtwbXRBc3NldFN0cgIAAAAMIEJVVCBzdGF0dXM9CQABkQAAAAIFAAAADHNlbGxBc3NldENmZwUAAAASSWR4RGVmb0Fzc2V0U3RhdHVzBAAAABBidXlBc3NldFVzZFByaWNlBQAAAAVwcmljZQQAAAARc2VsbEFzc2V0VXNkUHJpY2UJAQAAABNjb250cm9sQWNjUmVhZFByaWNlAAAAAQUAAAAUc2VsbEFzc2V0Q29kZUNvbmZpcm0EAAAADWJ1eTJzZWxsUHJpY2UJAABrAAAAAwUAAAAQYnV5QXNzZXRVc2RQcmljZQUAAAANcHJpY2VEZWNpbWFscwUAAAARc2VsbEFzc2V0VXNkUHJpY2UEAAAAC2RlYnRBc3NldElkCQACWQAAAAEJAQAAABZmYWN0b3J5UmVhZERlYnRBc3NldElkAAAAAAQAAAAIdXNkbkRlYnQJAABrAAAAAwUAAAARc2VsbEFzc2V0VXNkUHJpY2UFAAAACXBtdEFtb3VudAUAAAANcHJpY2VEZWNpbWFscwQAAAAWdG90YWxMZW5kZWRBdE90aGVyQWNjcwkBAAAAC3ZhbHVlT3JFbHNlAAAAAgkABBoAAAACBQAAAAR0aGlzCQEAAAAZa2V5VG90YWxMZW5kZWRBdE90aGVyQWNjcwAAAAAAAAAAAAAAAAAEAAAAGmxlbmRlZEFtb3VudEJ5QXNzZXRDb2RlS2V5CQEAAAAaa2V5TGVuZGVkQW1vdW50QnlBc3NldENvZGUAAAABBQAAABRzZWxsQXNzZXRDb2RlQ29uZmlybQQAAAAUbGVuZGVkQW10QnlBc3NldENvZGUJAQAAAAt2YWx1ZU9yRWxzZQAAAAIJAAQaAAAAAgUAAAAEdGhpcwUAAAAabGVuZGVkQW1vdW50QnlBc3NldENvZGVLZXkAAAAAAAAAAAAEAAAADmJ1eUFzc2V0UmVzdWx0CQEAAAAQaW50ZXJuYWxCdXlBc3NldAAAAAYIBQAAAAFpAAAABmNhbGxlcgUAAAAJcG10QW1vdW50BQAAAAhwbXRBc3NldAUAAAAKbWluU2VsbFBtdAUAAAANYnV5MnNlbGxQcmljZQkAAGsAAAADBQAAAA1idXlGZWVQZXJjZW50AAAAAAAAAACWAAAAAAAAAABkCQAETQAAAAIJAARNAAAAAggFAAAADmJ1eUFzc2V0UmVzdWx0AAAAAl8xCQEAAAAMSW50ZWdlckVudHJ5AAAAAgUAAAAabGVuZGVkQW1vdW50QnlBc3NldENvZGVLZXkJAABkAAAAAgUAAAAUbGVuZGVkQW10QnlBc3NldENvZGUFAAAACHVzZG5EZWJ0CQEAAAAMSW50ZWdlckVudHJ5AAAAAgkBAAAAGWtleVRvdGFsTGVuZGVkQXRPdGhlckFjY3MAAAAACQAAZAAAAAIFAAAAFnRvdGFsTGVuZGVkQXRPdGhlckFjY3MFAAAACHVzZG5EZWJ0AAAAAWkBAAAADnJlYmFsYW5jZURlYnRzAAAAAAQAAAALZGVidEFzc2V0SWQJAAJZAAAAAQkBAAAAFmZhY3RvcnlSZWFkRGVidEFzc2V0SWQAAAAABAAAAAhkZWJ0UG10MAkBAAAABXZhbHVlAAAAAQkAAZEAAAACCAUAAAABaQAAAAhwYXltZW50cwAAAAAAAAAAAAQAAAANZGVidFBtdEFzc2V0MAkBAAAABXZhbHVlAAAAAQgFAAAACGRlYnRQbXQwAAAAB2Fzc2V0SWQEAAAACGJhc2VQbXQxCQEAAAAFdmFsdWUAAAABCQABkQAAAAIIBQAAAAFpAAAACHBheW1lbnRzAAAAAAAAAAABBAAAAA1iYXNlUG10QXNzZXQxCQEAAAAFdmFsdWUAAAABCAUAAAAIYmFzZVBtdDEAAAAHYXNzZXRJZAQAAAANZGVidG9yQWRkcmVzcwkABCUAAAABCAUAAAABaQAAAAZjYWxsZXIEAAAADmRlYnRvckFzc2V0Q2ZnCQEAAAAcZmFjdG9yeVJlYWRBc3NldENmZ0J5QWRkcmVzcwAAAAEFAAAADWRlYnRvckFkZHJlc3MEAAAAD2RlYnRvckFzc2V0Q29kZQkAAZEAAAACBQAAAA5kZWJ0b3JBc3NldENmZwUAAAAQSWR4RGVmb0Fzc2V0Q29kZQQAAAAabGVuZGVkQW1vdW50QnlBc3NldENvZGVLZXkJAQAAABprZXlMZW5kZWRBbW91bnRCeUFzc2V0Q29kZQAAAAEFAAAAD2RlYnRvckFzc2V0Q29kZQQAAAAJbGVuZGVkQW10CQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAAQaAAAAAgUAAAAEdGhpcwUAAAAabGVuZGVkQW1vdW50QnlBc3NldENvZGVLZXkJAAEsAAAAAgIAAAANTm8gZGVidHMgZm9yIAUAAAAPZGVidG9yQXNzZXRDb2RlAwkBAAAAAiE9AAAAAgUAAAALZGVidEFzc2V0SWQFAAAADWRlYnRQbXRBc3NldDAJAAACAAAAAQkAASwAAAACCQABLAAAAAIJAAEsAAAAAgIAAAA0aW52YWxpZCBkZWJ0IGFzc2V0IGlkIGluIHRoZSBmaXJzdCBwYXltZXQ6IGV4cGVjdGVkPQkAAlgAAAABBQAAAAtkZWJ0QXNzZXRJZAIAAAAIIGFjdHVhbD0JAAJYAAAAAQUAAAANZGVidFBtdEFzc2V0MAMJAQAAAAIhPQAAAAIFAAAAC2Jhc2VBc3NldElkBQAAAA1iYXNlUG10QXNzZXQxCQAAAgAAAAEJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAANmludmFsaWQgYmFzZSBhc3NldCBpZCBpbiB0aGUgc2Vjb25kIHBheW1lbnQ6IGV4cGVjdGVkPQkAAlgAAAABBQAAAAtiYXNlQXNzZXRJZAIAAAAIIGFjdHVhbD0JAAJYAAAAAQUAAAANYmFzZVBtdEFzc2V0MQMJAQAAAAIhPQAAAAIIBQAAAAhkZWJ0UG10MAAAAAZhbW91bnQIBQAAAAhiYXNlUG10MQAAAAZhbW91bnQJAAACAAAAAQIAAAA/Zmlyc3QgcGF5bWVudCBhbW91bnQgZG9lc24ndCBtYXRjaCB0byB0aGUgc2Vjb25kIHBheW1lbnQgYW1vdW50AwkAAGcAAAACAAAAAAAAAAAABQAAAAlsZW5kZWRBbXQJAAACAAAAAQkAASwAAAACAgAAACdsZW5kZWRBbXQgaXMgbGVzcyB0aGVuIHplcm86IGxlbmRlZEFtdD0JAAGkAAAAAQUAAAAJbGVuZGVkQW10AwkAAGcAAAACAAAAAAAAAAAACAUAAAAIZGVidFBtdDAAAAAGYW1vdW50CQAAAgAAAAEJAAEsAAAAAgIAAAA1YXR0YWNoZWQgcGF5bWVudCBtdXN0IGJlIGdyZWF0ZXIgdGhlbiAwOiBwbXQwLmFtb3VudD0JAAGkAAAAAQgFAAAACGRlYnRQbXQwAAAABmFtb3VudAMJAABmAAAAAggFAAAACGRlYnRQbXQwAAAABmFtb3VudAUAAAAJbGVuZGVkQW10CQAAAgAAAAEJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAANGF0dGFjaGVkIHBheW1lbnQgaXMgZ3JhdGVyIHRoYW4gcmVxdWlyZWQ6IHBtdEFtb3VudD0JAAGkAAAAAQgFAAAACGRlYnRQbXQwAAAABmFtb3VudAIAAAALIGxlbmRlZEFtdD0JAAGkAAAAAQUAAAAJbGVuZGVkQW10BAAAABZ0b3RhbExlbmRlZEF0T3RoZXJBY2NzCQEAAAALdmFsdWVPckVsc2UAAAACCQAEGgAAAAIFAAAABHRoaXMJAQAAABlrZXlUb3RhbExlbmRlZEF0T3RoZXJBY2NzAAAAAAAAAAAAAAAAAAQAAAAObGVuZGVkQW10QWZ0ZXIJAABlAAAAAgUAAAAJbGVuZGVkQW10CAUAAAAIZGVidFBtdDAAAAAGYW1vdW50CQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACBQAAABpsZW5kZWRBbW91bnRCeUFzc2V0Q29kZUtleQUAAAAObGVuZGVkQW10QWZ0ZXIJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAIJAQAAABlrZXlUb3RhbExlbmRlZEF0T3RoZXJBY2NzAAAAAAkAAGUAAAACBQAAABZ0b3RhbExlbmRlZEF0T3RoZXJBY2NzCAUAAAAIZGVidFBtdDAAAAAGYW1vdW50CQAETAAAAAIJAQAAAAtTdHJpbmdFbnRyeQAAAAIJAAEsAAAAAgIAAAAWJXMlc19fcmViYWxhbmNlVHJhY2VfXwkAAlgAAAABCAUAAAABaQAAAA10cmFuc2FjdGlvbklkCQEAAAAXYXNzZXREYXRhUmViYWxhbmNlVHJhY2UAAAAFBQAAAA9kZWJ0b3JBc3NldENvZGUFAAAACGRlYnRQbXQwBQAAAAhiYXNlUG10MQUAAAAJbGVuZGVkQW10BQAAAA5sZW5kZWRBbXRBZnRlcgUAAAADbmlsAAAAAWkBAAAACHdpdGhkcmF3AAAAAwAAAA5hY2NvdW50QWRkcmVzcwAAAAx1bmxvY2tIZWlnaHQAAAADaWR4BAAAAA9hY2NPcGVyYXRpb25LZXkJAQAAABNrZXlBY2NvdW50T3BlcmF0aW9uAAAAAwUAAAAMdW5sb2NrSGVpZ2h0BQAAAA5hY2NvdW50QWRkcmVzcwIAAAAHUEVORElORwQAAAAVYWNjT3BlcmF0aW9uRGF0YUFycmF5CQEAAAAcYXNzZXRSZWFkU3dhcERhdGFBcnJheU9yRmFpbAAAAAEFAAAAD2FjY09wZXJhdGlvbktleQQAAAAIYW1vdW50SW4JAQAAAA1wYXJzZUludFZhbHVlAAAAAQkAAZEAAAACBQAAABVhY2NPcGVyYXRpb25EYXRhQXJyYXkFAAAAFElkeE9wZXJhdGlvbkFtb3VudEluBAAAAAdhc3NldEluCQABkQAAAAIFAAAAFWFjY09wZXJhdGlvbkRhdGFBcnJheQUAAAATSWR4T3BlcmF0aW9uQXNzZXRJbgQAAAAIYXNzZXRPdXQJAAGRAAAAAgUAAAAVYWNjT3BlcmF0aW9uRGF0YUFycmF5BQAAABRJZHhPcGVyYXRpb25Bc3NldE91dAMJAABmAAAAAgUAAAAMdW5sb2NrSGVpZ2h0BQAAAAZoZWlnaHQJAAACAAAAAQkAASwAAAACCQABLAAAAAICAAAADFBsZWFzZSB3YWl0IAkAAaQAAAABBQAAAAx1bmxvY2tIZWlnaHQCAAAAFyB0byB3aXRoZHJhdyB5b3VyIGZ1bmRzBAAAABBhY2NvdW50TG9ja2VkQW10BQAAAAhhbW91bnRJbgMJAABnAAAAAgAAAAAAAAAAAAUAAAAQYWNjb3VudExvY2tlZEFtdAkAAAIAAAABAgAAABFMb2NrZWRBbW91bnQgPD0gMAQAAAATYXNzZXRMb2NrZWRUb3RhbEtleQkBAAAAE2tleUFzc2V0TG9ja2VkVG90YWwAAAABCQABkQAAAAIFAAAAFWFjY09wZXJhdGlvbkRhdGFBcnJheQUAAAATSWR4T3BlcmF0aW9uQXNzZXRJbgQAAAAUY3VyckFzc2V0TG9ja2VkVG90YWwJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABBoAAAACBQAAAAR0aGlzBQAAABNhc3NldExvY2tlZFRvdGFsS2V5CQABLAAAAAIJAAEsAAAAAgIAAAAgU3RhdGUgY29udGFpbnMgc2VsbEFzc2V0UmVxdWVzdD0FAAAAD2FjY09wZXJhdGlvbktleQIAAAATIEJVVCBubyB0b3RhbExvY2tlZAQAAAAJaWR4SGVpZ2h0CQEAAAAXY29udHJvbEFjY1JlYWRJZHhIZWlnaHQAAAABBQAAAANpZHgEAAAADXByZXZJZHhIZWlnaHQJAQAAABdjb250cm9sQWNjUmVhZElkeEhlaWdodAAAAAEJAABlAAAAAgUAAAADaWR4AAAAAAAAAAABBAAAAAdjdXJySWR4CQEAAAAbY29udHJvbEFjY1JlYWRDdXJySWR4T3JGYWlsAAAAAAMDAwkAAGYAAAACBQAAAANpZHgFAAAAB2N1cnJJZHgGCQAAZgAAAAIFAAAADHVubG9ja0hlaWdodAUAAAAJaWR4SGVpZ2h0BgMJAQAAAAIhPQAAAAIFAAAADXByZXZJZHhIZWlnaHQAAAAAAAAAAAAJAABnAAAAAgUAAAANcHJldklkeEhlaWdodAUAAAAMdW5sb2NrSGVpZ2h0BwkAAAIAAAABCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACAgAAABdpbnZhbGlkIHByaWNlIGlkeDogaWR4PQkAAaQAAAABBQAAAANpZHgCAAAACSBjdXJySWR4PQkAAaQAAAABBQAAAAdjdXJySWR4AgAAAAsgaWR4SGVpZ2h0PQkAAaQAAAABBQAAAAlpZHhIZWlnaHQCAAAADiB1bmxvY2tIZWlnaHQ9CQABpAAAAAEFAAAADHVubG9ja0hlaWdodAIAAAAPIHByZXZJZHhIZWlnaHQ9CQABpAAAAAEFAAAADXByZXZJZHhIZWlnaHQEAAAAEHN5bnRoMmJhc2ljUHJpY2UJAQAAABtjb250cm9sQWNjUmVhZFByaWNlQnlIZWlnaHQAAAABBQAAAAlpZHhIZWlnaHQEAAAADGFzc2V0SW5CeXRlcwkAAlkAAAABBQAAAAdhc3NldEluBAAAAA0kdDAxODYyMzE5NDkzAwkAAAAAAAACBQAAAAxhc3NldEluQnl0ZXMFAAAAC2Jhc2VBc3NldElkBAAAAAhzeW50aEFtdAkAAGsAAAADBQAAAAhhbW91bnRJbgUAAAANcHJpY2VEZWNpbWFscwUAAAAQc3ludGgyYmFzaWNQcmljZQkABRYAAAAEBQAAAAhzeW50aEFtdAkBAAAAB1JlaXNzdWUAAAADBQAAAAtkZWZvQXNzZXRJZAUAAAAIc3ludGhBbXQGCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCQEAAAARQGV4dHJOYXRpdmUoMTA2MikAAAABBQAAAA5hY2NvdW50QWRkcmVzcwUAAAAIc3ludGhBbXQFAAAAC2RlZm9Bc3NldElkCQEAAAAMSW50ZWdlckVudHJ5AAAAAgkBAAAADmtleVVjb2xsYXRlcmFsAAAAAAkAAGQAAAACBQAAAAt1Y29sbGF0ZXJhbAUAAAAIYW1vdW50SW4DCQAAAAAAAAIFAAAADGFzc2V0SW5CeXRlcwUAAAALZGVmb0Fzc2V0SWQEAAAACGJhc2ljQW10CQAAawAAAAMFAAAACGFtb3VudEluBQAAABBzeW50aDJiYXNpY1ByaWNlBQAAAA1wcmljZURlY2ltYWxzBAAAAA5uZXdVY29sbGF0ZXJhbAkAAGUAAAACBQAAAAt1Y29sbGF0ZXJhbAUAAAAIYmFzaWNBbXQJAAUWAAAABAUAAAAIYmFzaWNBbXQJAQAAAARCdXJuAAAAAgUAAAALYmFzZUFzc2V0SWQFAAAACGJhc2ljQW10CQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCQEAAAARQGV4dHJOYXRpdmUoMTA2MikAAAABBQAAAA5hY2NvdW50QWRkcmVzcwUAAAAIYmFzaWNBbXQFAAAAC2Jhc2VBc3NldElkCQEAAAAMSW50ZWdlckVudHJ5AAAAAgkBAAAADmtleVVjb2xsYXRlcmFsAAAAAAkAAGUAAAACBQAAAAt1Y29sbGF0ZXJhbAUAAAAIYmFzaWNBbXQJAAACAAAAAQkAASwAAAACAgAAABRVbnN1cHBvcnRlZCBhc3NldEluPQUAAAAHYXNzZXRJbgQAAAAJYW1vdW50T3V0CAUAAAANJHQwMTg2MjMxOTQ5MwAAAAJfMQQAAAALYnVybk9ySXNzdWUIBQAAAA0kdDAxODYyMzE5NDkzAAAAAl8yBAAAABR0cmFuc2ZlclN5bnRoT3JCYXNpYwgFAAAADSR0MDE4NjIzMTk0OTMAAAACXzMEAAAAEHVjb2xsYXRlcmFsRW50cnkIBQAAAA0kdDAxODYyMzE5NDkzAAAAAl80AwkAAGYAAAACAAAAAAAAAAAACQAAZQAAAAIFAAAAFGN1cnJBc3NldExvY2tlZFRvdGFsBQAAABBhY2NvdW50TG9ja2VkQW10CQAAAgAAAAEJAAEsAAAAAgkAASwAAAACAgAAABRJbnZhbGlkIGRhdGEgc3RhdGU6IAUAAAATYXNzZXRMb2NrZWRUb3RhbEtleQIAAAAMIGxlc3MgdGhlbiAwCQAETQAAAAIJAARNAAAAAgkABE0AAAACCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACBQAAABNhc3NldExvY2tlZFRvdGFsS2V5CQAAZQAAAAIFAAAAFGN1cnJBc3NldExvY2tlZFRvdGFsBQAAABBhY2NvdW50TG9ja2VkQW10CQAETAAAAAIJAQAAAAtEZWxldGVFbnRyeQAAAAEFAAAAD2FjY09wZXJhdGlvbktleQkABEwAAAACCQEAAAALU3RyaW5nRW50cnkAAAACCQEAAAATa2V5QWNjb3VudE9wZXJhdGlvbgAAAAMFAAAADHVubG9ja0hlaWdodAUAAAAOYWNjb3VudEFkZHJlc3MCAAAACEZJTklTSEVECQEAAAAWYXNzZXREYXRhU3dhcE9wZXJhdGlvbgAAAAcFAAAACGFtb3VudEluBQAAAAdhc3NldEluBQAAABBzeW50aDJiYXNpY1ByaWNlBQAAAAlhbW91bnRPdXQFAAAACGFzc2V0T3V0AAAAAAAAAAAAAAAAAAAAAAAABQAAAANuaWwFAAAAC2J1cm5Pcklzc3VlBQAAABR0cmFuc2ZlclN5bnRoT3JCYXNpYwUAAAAQdWNvbGxhdGVyYWxFbnRyeQAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAEAAAAByRtYXRjaDAFAAAAAnR4AwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABdJbnZva2VTY3JpcHRUcmFuc2FjdGlvbgQAAAADaW52BQAAAAckbWF0Y2gwAwkBAAAAAiE9AAAAAggFAAAAA2ludgAAAAhmdW5jdGlvbgIAAAAOcmViYWxhbmNlRGVidHMJAAACAAAAAQIAAAAjb25seSByZWJhbGFuY2VEZWJ0cyBjYWxsIGlzIGFsbG93ZWQDCQEAAAACIT0AAAACCQABkQAAAAIJAQAAABxmYWN0b3J5UmVhZEFzc2V0Q2ZnQnlBZGRyZXNzAAAAAQkABCUAAAABCQAEJAAAAAEIBQAAAANpbnYAAAAEZEFwcAUAAAASSWR4RGVmb0Fzc2V0U3RhdHVzAgAAAAZJU1NVRUQJAAACAAAAAQIAAAAZb25seSBkZWZvIGRhcHAgaXMgYWxsb3dlZAMJAABmAAAAAggFAAAAA2ludgAAAANmZWUJAABoAAAAAgAAAAAAAAADhAAAAAAAAAAD6AkAAAIAAAABCQABLAAAAAICAAAAKGZlZSBhbW91bnQgaXMgZ3JlYXRlciB0aGFuIG1heCBhbGxvd2VkOiAJAAGkAAAAAQgFAAAAA2ludgAAAANmZWUDCQEAAAAJaXNEZWZpbmVkAAAAAQgFAAAAA2ludgAAAApmZWVBc3NldElkCQAAAgAAAAECAAAAI29ubHkgV2F2ZXMgaXMgYWxsb3dlZCBhcyBmZWVBc3NldElkBgkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAAIBQAAAAJ0eAAAAA9zZW5kZXJQdWJsaWNLZXnZqYEL"
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	tree = MustExpand(tree)
	require.True(t, tree.Expanded)
	_, err = CompileTree("", tree)
	require.NoError(t, err)
}

func TestPropertyAssignment(t *testing.T) {
	/*
		let owner = Address(base58'3MpENPBGgEAefMN27XvegNEbjAyohkGueii')
		func a(addr: Address) = if addr == owner then (nil, "OWNER") else ([StringEntry("CLIENT", addr.toString())], "CLIENT")

		@Callable(i)
		func call() = {
		  a(i.caller)._1
		}
	*/
	source := "AAIEAAAAAAAAAAQIAhIAAAAAAgAAAAAFb3duZXIJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVQDdVnTDlsUNa8Yx+PtFt6k/zbLY1cIbJcBAAAAAWEAAAABAAAABGFkZHIDCQAAAAAAAAIFAAAABGFkZHIFAAAABW93bmVyCQAFFAAAAAIFAAAAA25pbAIAAAAFT1dORVIJAAUUAAAAAgkABEwAAAACCQEAAAALU3RyaW5nRW50cnkAAAACAgAAAAZDTElFTlQJAAQlAAAAAQUAAAAEYWRkcgUAAAADbmlsAgAAAAZDTElFTlQAAAABAAAAAWkBAAAABGNhbGwAAAAACAkBAAAAAWEAAAABCAUAAAABaQAAAAZjYWxsZXIAAAACXzEAAAAATkiWqg=="
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	tree = MustExpand(tree)
	require.True(t, tree.Expanded)
	_, err = CompileTree("", tree)
	require.NoError(t, err)
}
