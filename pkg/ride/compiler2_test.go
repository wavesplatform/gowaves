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

func Test22(t *testing.T) {
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
		{fcall1, "BAoBAAAABmdldEludAAAAAEAAAADa2V5BAAAAAckbWF0Y2gwCQAEGgAAAAIFAAAABHRoaXMFAAAAA2tleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAF4BQAAAAckbWF0Y2gwBQAAAAF4AAAAAAAAAAAABAAAAAFhCQEAAAAGZ2V0SW50AAAAAQIAAAABNQQAAAABYgkBAAAABmdldEludAAAAAECAAAAATYJAAAAAAAAAgUAAAABYQUAAAABYkOIJQA=", env, false},
		{finf, "BAoBAAAAA2FiYwAAAAAKAQAAAAJpbgAAAAAGCQEAAAACaW4AAAAACQEAAAADYWJjAAAAADpBKyM=", env, true},
		{intersectNames, "AwoBAAAAA2luYwAAAAEAAAABdgkAAGQAAAACBQAAAAF2AAAAAAAAAAABCgEAAAAEY2FsbAAAAAEAAAADaW5jCQEAAAADaW5jAAAAAQUAAAADaW5jCQAAAAAAAAIJAQAAAARjYWxsAAAAAQAAAAAAAAAAAgAAAAAAAAAAAxgTXMY=", env, true},
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
	} {
		src, err := base64.StdEncoding.DecodeString(test.source)
		require.NoError(t, err, test.comment)

		tree, err := Parse(src)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, tree, test.comment)

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

// 1 == 1
func TestCallExternal(t *testing.T) {
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

// let x = if (true) then true else false; x
func TestIfConditionRightByteCode(t *testing.T) {
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

	/**
	require.Equal(t,
		[]byte{
			OpReturn,
			OpRef, 0, 1,
			OpClearCache, 0, 1,
			OpReturn,
			OpRef, 0, 2,
			OpJumpIfFalse, 0, 0x12, 0, 0x16, 0, 0x1a,
			OpRef, 0, 3,
			OpReturn,
			OpRef, 0, 4,
			OpReturn,
			OpReturn,
		},
		f.ByteCode)

	/**/

	rs, err := f.Verify(nil)
	require.NoError(t, err)
	require.Equal(t, true, rs.Result())
}

// let i = 1; let s = "string"; toString(i) == s
func TestCall(t *testing.T) {
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

//func a() = 1; a() == 1
func TestDoubleCall(t *testing.T) {
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

// func id(v: Boolean) = v; id(true)
func TestCallWithConstArg(t *testing.T) {
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

// func id(v: Boolean) = v && v; id(true)
func TestMultipleCallConstantFuncArgument(t *testing.T) {
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
func TestIfStmt(t *testing.T) {
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

	for _, l := range r.calls {
		t.Log(l)
	}

	assert.Equal(t, true, r.Result())
}

/*
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func abc(question: String) = {
    WriteSet([
        DataEntry("a", 5)
        ])
}

@Callable(i)
func cba(question: String) = {
    WriteSet([
        DataEntry("a", 6)
        ])
}
*/
func TestDappMultipleFunctions(t *testing.T) {
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

/*

func finalizeCurrentPrice() {
	let prices = 1100(
					getOracleProvideHeight(
						401(oraclesList,0),
						height
					),
					1100(getOracleProvideHeight(401(oraclesList,1),height),
					1100(getOracleProvideHeight(401(oraclesList,2,),height,),
					1100(getOracleProvideHeight(401(oraclesList,3,),height,),
					1100(getOracleProvideHeight(401(oraclesList,4,),height,),nil,),),),),);
	let priceProvidingCount = ((((if (!=(401(prices,0,),0,)) { 1 } else { 0 } + if (!=(401(prices,1,),0,)) { 1 } else { 0 }) + if (!=(401(prices,2,),0,)) { 1 } else { 0 }) + if (!=(401(prices,3,),0,)) { 1 } else { 0 }) + if (!=(401(prices,4,),0,)) { 1 } else { 0 });
	let priceSum = ((((401(prices,0,) + 401(prices,1,)) + 401(prices,2,)) + 401(prices,3,)) + 401(prices,4,));
	let newPrice = 105(priceSum,priceProvidingCount,);
	if (isBlocked) {
		2("contract is blocked")
	} else {
		if (102(bftCoefficientOracle,priceProvidingCount)) {
			2(
				300(
					300(
						420(bftCoefficientOracle),
						"/5 oracles need to set a price "
					),
					420(priceProvidingCount)
				)
			)
		} else {
			if (if (103(newPrice,(price + 105(104(price,percentPriceOffset,),100,)),)) { true } else { 103((price - 105(104(price,percentPriceOffset,),100,)),newPrice,) }) { WriteSet(1100(DataEntry(IsBlockedKey,true,),1100(DataEntry(getBlackSwarmPriceKey(height,),newPrice,),nil,),),) } else { let newPriceIndex = (priceIndex + 1); WriteSet(1100(DataEntry(PriceKey,newPrice,),1100(DataEntry(getPriceHistoryKey(height,),newPrice,),1100(DataEntry(PriceIndexKey,newPriceIndex,),1100(DataEntry(getHeightPriceByIndexKey(newPriceIndex,),height,),nil,),),),),) } } } }


*/

func Test777(t *testing.T) {
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

/*
func abc() = 5
func cba() = 10
if abc() == cba() then {
    true
} else {
    false
}
*/
func Test888(t *testing.T) {
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

	/**
	require.Equal(t,
		[]byte{
			OpReturn,
			OpRef, 0, 0,
			OpRef, 0, 0,
			OpCall, 0, 0, 0, 2,
			OpJumpIfFalse, 0, 0, 0, 0, 0, 0,
			OpRef, 0, 0, OpReturn, //true branch
			OpRef, 0, 0, OpReturn, //false branch
			OpReturn,
			OpRef, 0, 0, OpReturn, // function cba
			OpRef, 0, 0, OpReturn, // function abc
		},
		script.ByteCode)
	/**/

	rs, _ := script.Verify(env)
	require.Equal(t, rs.Result(), false)
	//require.Equal(t, err.Error(), "terminated execution by throw with message \"1\"")
}

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
func TestNoDuplicateCallToState(t *testing.T) {
	source := `AAIDAAAAAAAAAAIIAQAAAAgBAAAAGGdldFN0cmluZ0J5QWRkcmVzc0FuZEtleQAAAAIAAAAHYWRkcmVzcwAAAANrZXkEAAAAByRtYXRjaDAJAAQdAAAAAgUAAAAHYWRkcmVzcwUAAAADa2V5AwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAAZTdHJpbmcEAAAAAWEFAAAAByRtYXRjaDAFAAAAAWECAAAAAAEAAAAOZ2V0U3RyaW5nQnlLZXkAAAABAAAAA2tleQQAAAAHJG1hdGNoMAkABB0AAAACBQAAAAR0aGlzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABlN0cmluZwQAAAABYQUAAAAHJG1hdGNoMAUAAAABYQIAAAAAAAAAABBMYXN0Q29uZmlybVR4S2V5AgAAAA9sYXN0X2NvbmZpcm1fdHgAAAAAE05ldXRyaW5vQ29udHJhY3RLZXkCAAAAEW5ldXRyaW5vX2NvbnRyYWN0AAAAABJDb250cm9sQ29udHJhY3RLZXkCAAAAEGNvbnRyb2xfY29udHJhY3QAAAAAEG5ldXRyaW5vQ29udHJhY3QJAQAAABxAZXh0clVzZXIoYWRkcmVzc0Zyb21TdHJpbmcpAAAAAQkBAAAADmdldFN0cmluZ0J5S2V5AAAAAQUAAAATTmV1dHJpbm9Db250cmFjdEtleQAAAAAPY29udHJvbENvbnRyYWN0CQEAAAAcQGV4dHJVc2VyKGFkZHJlc3NGcm9tU3RyaW5nKQAAAAEJAQAAABhnZXRTdHJpbmdCeUFkZHJlc3NBbmRLZXkAAAACBQAAABBuZXV0cmlub0NvbnRyYWN0BQAAABJDb250cm9sQ29udHJhY3RLZXkAAAAADWxhc3RDb25maXJtVHgJAQAAABhnZXRTdHJpbmdCeUFkZHJlc3NBbmRLZXkAAAACBQAAAA9jb250cm9sQ29udHJhY3QFAAAAEExhc3RDb25maXJtVHhLZXkAAAAAAAAAAQAAAAJ0eAEAAAAGdmVyaWZ5AAAAAAkAAAAAAAACBQAAAA1sYXN0Q29uZmlybVR4CQACWAAAAAEIBQAAAAJ0eAAAAAJpZJO+lgc=`

	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		//RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
		//	t.Log("key: ", key)
		//	return nil, errors.New("not found")
		//},
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
	for _, c := range rs.Calls() {
		t.Log(c)
	}
	//t.Log(rs.Calls())
	require.NoError(t, err)

	//t.Log(rs.Calls())
	require.False(t, rs.Result())
}

//type points struct {
//	value []point `cbor:"0,keyasint"`
//}

//func TestSerialize(t *testing.T) {
//
//	//m := points{
//	//	value: []point{
//	//		{value: rideBoolean(true)},
//	//		{value: rideInt(5), constant: true},
//	//	},
//	//}
//	m := point{
//		position:  43,
//		value:     rideUnit{},
//		fn:        nil,
//		constant:  true,
//		debugInfo: "bla",
//	}
//
//	rs, err := cbor.Marshal(m)
//	require.NoError(t, err)
//
//	t.Log(rs)
//
//	var m2 point
//
//	err = cbor.Unmarshal(rs, &m2)
//	require.NoError(t, err)
//	t.Log(m2)
//
//}

/*
{-# STDLIB_VERSION 3 #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# CONTENT_TYPE DAPP #-}

@Verifier(tx)
func verify () = sigVerify(tx.bodyBytes, tx.proofs[0], tx.senderPublicKey)
*/
func TestDappVerifyVm(t *testing.T) {
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

/*
{-# STDLIB_VERSION 3 #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# CONTENT_TYPE EXPRESSION #-}

match (tx) {
    case e:ExchangeTransaction => isDefined(e.sellOrder.assetPair.priceAsset)
    case _ => throw("err")
  }
*/
func TestMultipleProperty(t *testing.T) {
	source := `AwQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE0V4Y2hhbmdlVHJhbnNhY3Rpb24EAAAAAWUFAAAAByRtYXRjaDAJAQAAAAlpc0RlZmluZWQAAAABCAgIBQAAAAFlAAAACXNlbGxPcmRlcgAAAAlhc3NldFBhaXIAAAAKcHJpY2VBc3NldAkAAAIAAAABAgAAAANlcnIsqB0K`
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

/*
{-# STDLIB_VERSION 4 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}

let x = 1 + 1
x == x
*/
func TestCacheInMain(t *testing.T) {
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

	/**
	require.Equal(t, []byte{
		OpReturn,
		0xe, 0x0, 0xc9,
		OpReturn,
		OpRef, 0x0, 0xc9,
		OpCache, 0x0, 0xc9,
		OpRef, 0x0, 0xc9,
		OpCache, 0x0, 0xc9,
		OpExternalCall, 0x0, 0x3, 0x0, 0x2, OpReturn,
	}, script.ByteCode)
	/**/

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.Equal(t, 2, len(rs.Calls())) // plus & eq
	require.Equal(t, rs.Result(), true)
}

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
func TestCacheFunctionArgumentsCalls(t *testing.T) {
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
func TestCacheInFunc(t *testing.T) {
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

/*
{-# STDLIB_VERSION 4 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}
func abc(x: Int) = x == x
let y = getIntegerValue(this, "a")
abc(y)
*/
func TestCacheFuncArgs(t *testing.T) {
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

/*
let x = {
	let y = true;
	y;
}
x
*/
func TestLetInLet(t *testing.T) {
	source := `BAQAAAABeAQAAAABeQYFAAAAAXkFAAAAAXhCPj2C`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	script, err := CompileVerifier("", tree)
	t.Log(Decompiler(tree.Verifier))
	require.NoError(t, err)
	assert.NotNil(t, script)

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
	}

	/* *
	require.Equal(t,
		[]byte{
			OpReturn,
			OpRef, 0, 1,
			OpClearCache, 0, 2,
			OpClearCache, 0, 1,
			OpReturn,
			OpRef, 0, 2,
			//OpReturn,
		},
		script.ByteCode[:11])
	/**/

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.Equal(t, true, rs.Result())
}

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
func TestDDaa(t *testing.T) {
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

func TestFuncInCondState(t *testing.T) {
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

/*

 */

func Test111111(t *testing.T) {
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
	//rs, err := CallTreeVerifier(env, tree)
	//for i, c := range rs.Calls() {
	//	t.Log(i, " ", c)
	//}

	exe, err := CompileTree("", tree)
	require.NoError(t, err)
	rs, err := exe.Verify(env)
	require.NoError(t, err)
	for i, c := range rs.Calls() {
		t.Log(i, " ", c)
	}
}

/**
let height = height
height != 0
*/
func TestShadowedVariable(t *testing.T) {
	source := `AwoBAAAAD2dldFByaWNlSGlzdG9yeQAAAAEAAAAGaGVpZ2h0BQAAAAZoZWlnaHQJAQAAAAIhPQAAAAIJAQAAAA9nZXRQcmljZUhpc3RvcnkAAAABBQAAAAZoZWlnaHQAAAAAAAAAAADe0Skk`

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	tree = MustExpand(tree)
	require.Equal(t, "(let height = { height }; height != 0)", DecompileTree(tree))

	script, err := CompileTree("", tree)
	require.NoError(t, err)

	rs, err := script.Verify(defaultEnv)
	require.NoError(t, err)
	require.Equal(t, true, rs.Result())
}

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
func TestNoDuplicateCallToState2(t *testing.T) {
	source := `AAIDAAAAAAAAABYIARIAEgQKAgEIEgQKAggBEgQKAggBAAAAGgEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABAAAAA2tleQQAAAAHJG1hdGNoMAkABBoAAAACBQAAAAR0aGlzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAA0ludAQAAAABYQUAAAAHJG1hdGNoMAUAAAABYQAAAAAAAAAAAAEAAAAOZ2V0U3RyaW5nQnlLZXkAAAABAAAAA2tleQQAAAAHJG1hdGNoMAkABB0AAAACBQAAAAR0aGlzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABlN0cmluZwQAAAABYQUAAAAHJG1hdGNoMAUAAAABYQIAAAAAAQAAAAxnZXRCb29sQnlLZXkAAAABAAAAA2tleQQAAAAHJG1hdGNoMAkABBsAAAACBQAAAAR0aGlzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAB0Jvb2xlYW4EAAAAAWEFAAAAByRtYXRjaDAFAAAAAWEHAQAAABhnZXROdW1iZXJCeUFkZHJlc3NBbmRLZXkAAAACAAAAB2FkZHJlc3MAAAADa2V5BAAAAAckbWF0Y2gwCQAEGgAAAAIJAQAAABxAZXh0clVzZXIoYWRkcmVzc0Zyb21TdHJpbmcpAAAAAQUAAAAHYWRkcmVzcwUAAAADa2V5AwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAANJbnQEAAAAAWEFAAAAByRtYXRjaDAFAAAAAWEAAAAAAAAAAAABAAAAGGdldFN0cmluZ0J5QWRkcmVzc0FuZEtleQAAAAIAAAAHYWRkcmVzcwAAAANrZXkEAAAAByRtYXRjaDAJAAQdAAAAAgUAAAAHYWRkcmVzcwUAAAADa2V5AwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAAZTdHJpbmcEAAAAAWEFAAAAByRtYXRjaDAFAAAAAWECAAAAAAAAAAASTmV1dHJpbm9Bc3NldElkS2V5AgAAABFuZXV0cmlub19hc3NldF9pZAAAAAATTmV1dHJpbm9Db250cmFjdEtleQIAAAARbmV1dHJpbm9fY29udHJhY3QAAAAACkJhbGFuY2VLZXkCAAAAC3JwZF9iYWxhbmNlAAAAABBMYXN0Q29uZmlybVR4S2V5AgAAAA9sYXN0X2NvbmZpcm1fdHgAAAAAEkNvbnRyb2xDb250cmFjdEtleQIAAAAQY29udHJvbF9jb250cmFjdAEAAAARZ2V0VXNlckJhbGFuY2VLZXkAAAACAAAABW93bmVyAAAAB2Fzc2V0SWQJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgUAAAAKQmFsYW5jZUtleQIAAAABXwUAAAAHYXNzZXRJZAIAAAABXwUAAAAFb3duZXIBAAAAFWdldENvbnRyYWN0QmFsYW5jZUtleQAAAAEAAAAHYXNzZXRJZAkAASwAAAACCQABLAAAAAIFAAAACkJhbGFuY2VLZXkCAAAAAV8FAAAAB2Fzc2V0SWQBAAAAFGdldEV4cGlyZVByb3Bvc2FsS2V5AAAAAQAAAARoYXNoCQABLAAAAAIJAAEsAAAAAgIAAAAPcHJvcG9zYWxfZXhwaXJlAgAAAAFfBQAAAARoYXNoAQAAABNnZXRPd25lclByb3Bvc2FsS2V5AAAAAQAAAARoYXNoCQABLAAAAAIJAAEsAAAAAgIAAAAOcHJvcG9zYWxfb3duZXICAAAAAV8FAAAABGhhc2gBAAAAF2dldEFyZ3VtZW50c1Byb3Bvc2FsS2V5AAAAAQAAAARoYXNoCQABLAAAAAIJAAEsAAAAAgIAAAAScHJvcG9zYWxfYXJndW1lbnRzAgAAAAFfBQAAAARoYXNoAQAAAApnZXRWb3RlS2V5AAAAAgAAAAVvd25lcgAAAARoYXNoCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAADXByb3Bvc2FsX3ZvdGUCAAAAAV8FAAAABW93bmVyAgAAAAFfBQAAAARoYXNoAAAAABBuZXV0cmlub0NvbnRyYWN0CQEAAAAcQGV4dHJVc2VyKGFkZHJlc3NGcm9tU3RyaW5nKQAAAAEJAQAAAA5nZXRTdHJpbmdCeUtleQAAAAEFAAAAE05ldXRyaW5vQ29udHJhY3RLZXkAAAAAD2NvbnRyb2xDb250cmFjdAkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABCQEAAAAYZ2V0U3RyaW5nQnlBZGRyZXNzQW5kS2V5AAAAAgUAAAAQbmV1dHJpbm9Db250cmFjdAUAAAASQ29udHJvbENvbnRyYWN0S2V5AAAAAA1sYXN0Q29uZmlybVR4CQEAAAAYZ2V0U3RyaW5nQnlBZGRyZXNzQW5kS2V5AAAAAgUAAAAPY29udHJvbENvbnRyYWN0BQAAABBMYXN0Q29uZmlybVR4S2V5AAAAAA9uZXV0cmlub0Fzc2V0SWQJAAJZAAAAAQkBAAAAGGdldFN0cmluZ0J5QWRkcmVzc0FuZEtleQAAAAIFAAAAEG5ldXRyaW5vQ29udHJhY3QFAAAAEk5ldXRyaW5vQXNzZXRJZEtleQEAAAASZ2V0Q29udHJhY3RCYWxhbmNlAAAAAQAAAAdhc3NldElkCQEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABCQEAAAAVZ2V0Q29udHJhY3RCYWxhbmNlS2V5AAAAAQUAAAAHYXNzZXRJZAEAAAAOZ2V0VXNlckJhbGFuY2UAAAACAAAABW93bmVyAAAAB2Fzc2V0SWQJAQAAAA5nZXROdW1iZXJCeUtleQAAAAEJAQAAABFnZXRVc2VyQmFsYW5jZUtleQAAAAIFAAAABW93bmVyBQAAAAdhc3NldElkAQAAABFnZXRFeHBpcmVQcm9wb3NhbAAAAAEAAAAEaGFzaAkBAAAADmdldE51bWJlckJ5S2V5AAAAAQkBAAAAFGdldEV4cGlyZVByb3Bvc2FsS2V5AAAAAQUAAAAEaGFzaAEAAAAQZ2V0T3duZXJQcm9wb3NhbAAAAAEAAAAEaGFzaAkBAAAADmdldFN0cmluZ0J5S2V5AAAAAQkBAAAAE2dldE93bmVyUHJvcG9zYWxLZXkAAAABBQAAAARoYXNoAQAAABRnZXRBcmd1bWVudHNQcm9wb3NhbAAAAAEAAAAEaGFzaAkBAAAADmdldFN0cmluZ0J5S2V5AAAAAQkBAAAAF2dldEFyZ3VtZW50c1Byb3Bvc2FsS2V5AAAAAQUAAAAEaGFzaAEAAAAHZ2V0Vm90ZQAAAAIAAAAFb3duZXIAAAAEaGFzaAkBAAAADmdldFN0cmluZ0J5S2V5AAAAAQkBAAAACmdldFZvdGVLZXkAAAACBQAAAAVvd25lcgUAAAAEaGFzaAAAAAQAAAABaQEAAAAMbG9ja05ldXRyaW5vAAAAAAQAAAADcG10CQEAAAAHZXh0cmFjdAAAAAEIBQAAAAFpAAAAB3BheW1lbnQDCQEA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`

	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		//RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
		//	t.Log("key: ", key)
		//	return nil, errors.New("not found")
		//},
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

	rs1, err := CallTreeVerifier(env, tree)
	//rs1, err := CallTreeVerifier(env, MustExpand(tree))
	require.NoError(t, err)
	for _, c := range rs1.Calls() {
		t.Log(c)
	}
	t.Log("")
	t.Log("")

	script, err := CompileVerifier("", MustExpand(tree))
	require.NoError(t, err)
	assert.NotNil(t, script)

	rs2, err := script.Verify(env)
	require.NoError(t, err)
	for _, c := range rs2.Calls() {
		t.Log(c)
	}
	//t.Log(rs.Calls())
	require.NoError(t, err)

	//t.Log(rs.Calls())
	require.False(t, rs2.Result())
}
