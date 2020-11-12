package ride

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

func Test_ccc(t *testing.T) {

	version := 3

	// let x = 5; 6 > x
	ast := &AssignmentNode{
		Name: "x",
		Expression: &LongNode{
			Value: 5,
		},
		Block: &FunctionCallNode{
			Name: "102", // gt
			Arguments: []Node{
				&LongNode{
					Value: 6,
				},
				&ReferenceNode{
					Name: "x",
				},
			},
		},
	}
	rs, err := compileSimpleScript(version, ast)

	fSelect, err := selectFunctions(version)
	require.NoError(t, err)

	v := vm{
		code:      rs.ByteCode,
		ip:        int(rs.EntryPoints[""]),
		constants: rs.Constants,
		functions: fSelect,
	}

	sr, err := v.run()
	require.NoError(t, err)
	require.True(t, sr.Result())
}

func Test22(t *testing.T) {
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
	}
	for _, test := range []struct {
		comment string
		source  string
		env     RideEnvironment
		res     bool
	}{
		{`V1: true`, "AQa3b8tH", nil, true},
		{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn", nil, true},
		{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=", nil, true},
		{`V1: let i = 1; let s = "string"; toString(i) == s`, "AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=", env, false},
		{`V3: let i = 12345; let s = "12345"; toString(i) == s`, "AwQAAAABaQAAAAAAAAAwOQQAAAABcwIAAAAFMTIzNDUJAAAAAAAAAgkAAaQAAAABBQAAAAFpBQAAAAFz1B1iCw==", nil, true},
		{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E", nil, true},
		{`V3: if (false) then {let r = true; r} else {let r = false; r}`, "AwMHBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXI+tfo1", nil, false},
		{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==", env, true},
		{`V3: func a() = 1; a() == 2`, "BAoBAAAAAWEAAAAAAAAAAAAAAAABCQAAAAAAAAIJAQAAAAFhAAAAAAAAAAAAAAAAAsVdmuc=", env, false},
		{`V3: func id(v: Boolean) = v; id(true)`, "BAoBAAAAAmlkAAAAAQAAAAF2BQAAAAF2CQEAAAACaWQAAAABBglAaUs=", env, true},
		{`V3: 1 == 1`, "BAkAAAAAAAACAAAAAAAAAAABAAAAAAAAAAABq0EiMw==", env, true},
		{`V3: let x = 1; func add(i: Int) = i + 1; add(x) == 2`, "AwQAAAABeAAAAAAAAAAAAQoBAAAAA2FkZAAAAAEAAAABaQkAAGQAAAACBQAAAAFpAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAF4AAAAAAAAAAACfr6U6w==", env, true},
		{`V3: let b = base16'0000000000000001'; func add(b: ByteVector) = toInt(b) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAWIJAABkAAAAAgkABLEAAAABBQAAAAFiAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAACX00biA==", nil, true},
		{`V3: let b = base16'0000000000000001'; func add(v: ByteVector) = toInt(v) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAXYJAABkAAAAAgkABLEAAAABBQAAAAF2AAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAACI7gYxg==", nil, true},
		{`V3: let b = base16'0000000000000001'; func add(v: ByteVector) = toInt(b) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAXYJAABkAAAAAgkABLEAAAABBQAAAAFiAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAAChRvwnQ==", nil, true},
		{`V3: let data = base64'AAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLw=='; func getStock(data:ByteVector) = toInt(take(drop(data, 8), 8)); getStock(data) == 1`, `AwQAAAAEZGF0YQEAAABwAAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLwoBAAAACGdldFN0b2NrAAAAAQAAAARkYXRhCQAEsQAAAAEJAADJAAAAAgkAAMoAAAACBQAAAARkYXRhAAAAAAAAAAAIAAAAAAAAAAAICQAAAAAAAAIJAQAAAAhnZXRTdG9jawAAAAEFAAAABGRhdGEAAAAAAAAAAAFCtabi`, nil, true},
		{`V3: let ref = 999; func g(a: Int) = ref; func f(ref: Int) = g(ref); f(1) == 999`, "AwQAAAADcmVmAAAAAAAAAAPnCgEAAAABZwAAAAEAAAABYQUAAAADcmVmCgEAAAABZgAAAAEAAAADcmVmCQEAAAABZwAAAAEFAAAAA3JlZgkAAAAAAAACCQEAAAABZgAAAAEAAAAAAAAAAAEAAAAAAAAAA+fjknmW", nil, true},
		{`let x = 5; 6 > 4`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGAAAAAAAAAAAEYSW6XA==`, nil, true},
		{`let x = 5; 6 > x`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGBQAAAAF4Gh24hw==`, nil, true},
		{`let x = 5; 6 >= x`, `AQQAAAABeAAAAAAAAAAABQkAAGcAAAACAAAAAAAAAAAGBQAAAAF4jlxXHA==`, nil, true},
		{`false`, `AQfeYll6`, nil, false},
		{`let x =  throw(); true`, `AQQAAAABeAkBAAAABXRocm93AAAAAAa7bgf4`, nil, true},
		{`let x =  throw(); true || x`, `AQQAAAABeAkBAAAABXRocm93AAAAAAMGBgUAAAABeKRnLds=`, env, true},
		// Global variables
		{`tx == tx`, "BAkAAAAAAAACBQAAAAJ0eAUAAAACdHhnqgP4", env, true},
		{`tx.id == base58''`, `AQkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAAJBtD70=`, env, false},
		//{`tx.id == base58'H5C8bRzbUTMePSDVVxjiNKDUwk6CKzfZGTP2Rs7aCjsV'`, `BAkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAIO7N5luRDUgN1SJ4kFmy/Ni8U2H6k7bpszok5tlLlRVgHwSHyg==`, env, true},
		{`let x = tx.id == base58'a';true`, `AQQAAAABeAkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAASEGjR0kcA==`, env, true},
		{`tx.proofs[0] != base58'' && tx.proofs[1] == base58''`, `BAMJAQAAAAIhPQAAAAIJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAAEAAAAACQAAAAAAAAIJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAQEAAAAAB106gzM=`, env, true},
		{`match tx {case t : TransferTransaction | MassTransferTransaction | ExchangeTransaction => true; case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABNFeGNoYW5nZVRyYW5zYWN0aW9uBgMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24GCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAGB6Ilvok=`, env, true},
		{`V2: match transactionById(tx.id) {case  t: Unit => false case _ => true}`, `AgQAAAAHJG1hdGNoMAkAA+gAAAABCAUAAAACdHgAAAACaWQDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABFVuaXQEAAAAAXQFAAAAByRtYXRjaDAHBp9TFcQ=`, env, true},
		//{`Up() == UP`, `AwkAAAAAAAACCQEAAAACVXAAAAAABQAAAAJVUPGUxeg=`, env, true},
		//{`HalfUp() == HALFUP`, `AwkAAAAAAAACCQEAAAAGSGFsZlVwAAAAAAUAAAAGSEFMRlVQbUfpTQ==`, nil, true},
		//{`let a0 = NoAlg() == NOALG; let a1 = Md5() == MD5; let a2 = Sha1() == SHA1; let a3 = Sha224() == SHA224; let a4 = Sha256() == SHA256; let a5 = Sha384() == SHA384; let a6 = Sha512() == SHA512; let a7 = Sha3224() == SHA3224; let a8 = Sha3256() == SHA3256; let a9 = Sha3384() == SHA3384; let a10 = Sha3512() == SHA3512; a0 && a1 && a2 && a3 && a4 && a5 && a6 && a7 && a8 && a9 && a10`, `AwQAAAACYTAJAAAAAAAAAgkBAAAABU5vQWxnAAAAAAUAAAAFTk9BTEcEAAAAAmExCQAAAAAAAAIJAQAAAANNZDUAAAAABQAAAANNRDUEAAAAAmEyCQAAAAAAAAIJAQAAAARTaGExAAAAAAUAAAAEU0hBMQQAAAACYTMJAAAAAAAAAgkBAAAABlNoYTIyNAAAAAAFAAAABlNIQTIyNAQAAAACYTQJAAAAAAAAAgkBAAAABlNoYTI1NgAAAAAFAAAABlNIQTI1NgQAAAACYTUJAAAAAAAAAgkBAAAABlNoYTM4NAAAAAAFAAAABlNIQTM4NAQAAAACYTYJAAAAAAAAAgkBAAAABlNoYTUxMgAAAAAFAAAABlNIQTUxMgQAAAACYTcJAAAAAAAAAgkBAAAAB1NoYTMyMjQAAAAABQAAAAdTSEEzMjI0BAAAAAJhOAkAAAAAAAACCQEAAAAHU2hhMzI1NgAAAAAFAAAAB1NIQTMyNTYEAAAAAmE5CQAAAAAAAAIJAQAAAAdTaGEzMzg0AAAAAAUAAAAHU0hBMzM4NAQAAAADYTEwCQAAAAAAAAIJAQAAAAdTaGEzNTEyAAAAAAUAAAAHU0hBMzUxMgMDAwMDAwMDAwMFAAAAAmEwBQAAAAJhMQcFAAAAAmEyBwUAAAACYTMHBQAAAAJhNAcFAAAAAmE1BwUAAAACYTYHBQAAAAJhNwcFAAAAAmE4BwUAAAACYTkHBQAAAANhMTAHRc/wAA==`, env, true},
		//{`Unit() == unit`, `AwkAAAAAAAACCQEAAAAEVW5pdAAAAAAFAAAABHVuaXTstg1G`, env, true},
	} {
		src, err := base64.StdEncoding.DecodeString(test.source)
		require.NoError(t, err, test.comment)

		tree, err := Parse(src)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, tree, test.comment)

		script, err := CompileSimpleScript(tree)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, script, test.comment)

		res, err := script.Run(test.env)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, res, test.comment)
		r, ok := res.(ScriptResult)
		assert.True(t, ok, test.comment)
		assert.Equal(t, test.res, r.Result(), test.comment)
	}
}

func buildCode(i ...interface{}) ([]byte, uint16) {
	marks := make(map[int]uint16)
	b := new(bytes.Buffer)
	for _, inf := range i {
		switch n := inf.(type) {
		case byte:
			b.WriteByte(n)
		case int:
			b.WriteByte(byte(n))
		case mark:
			marks[n.id] = uint16(b.Len())
		case toMark:
			b.Write(encode(marks[n.id]))
		}
	}
	return b.Bytes(), marks[0]
}

type mark struct {
	id int
}

func at(id int) mark {
	return mark{
		id: id,
	}
}

type toMark struct {
	id int
}

func to(id int) toMark {
	return toMark{id}
}

func TestBuildCode(t *testing.T) {
	rs, entryPoint := buildCode(1, at(2), 2, at(0), to(2))
	require.Equal(t, []byte{1, 2, 0, 1}, rs)
	require.Equal(t, uint16(2), entryPoint)
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

	f, err := compileSimpleScript(3, n)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			OpPush, 0, 0,
			OpPush, 0, 1,
			OpExternalCall, 0, 3, 0, 2,
			OpReturn,
		},
		f.ByteCode)
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

	f, err := compileSimpleScript(3, n)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			OpRef, 0, 1,
			OpReturn,

			OpCall, 0, 0,
			OpRef, 0, 3,
			OpExternalCall, 0, 3, 0, 2,
			OpReturn,
		},
		f.ByteCode)

	require.EqualValues(t, 4, f.EntryPoints[""])

	rs, err := f.Run(nil)
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

	f, err := compileSimpleScript(3, n)
	require.NoError(t, err)

	bt := []byte{
		//OpUseArg, 0, 1, OpReturn, // arguments section
		OpRef, 0, 1, // Function execution code. One line: reference to `v` argument.
		OpReturn,

		// call function
		OpSetArg, 0, 2, 0, 1,
		OpCall, 0, 0,

		OpReturn,
	}

	require.Equal(t, bt, f.ByteCode)

	f.ByteCode = bt
	f.EntryPoints[""] = 4

	rs, err := f.Run(nil)
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

	script, err := CompileSimpleScript(tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	res, err := script.Run(env)
	require.NoError(t, err)
	assert.NotNil(t, res)
	r, ok := res.(ScriptResult)
	assert.True(t, ok)
	assert.Equal(t, true, r.Result())

	//f, err := compileSimpleScript(3, n)
	//require.NoError(t, err)

	//require.Equal(t,
	//	[]byte{
	//		OpUseArg, 0, 1, OpReturn, // arguments section
	//		OpJump, 0, 0, // Function execution code. One line: reference to `v` argument.
	//		OpReturn,
	//
	//		OpTrue, OpReturn, // define constant
	//
	//		// call function
	//		OpSetArg, 0, 1, 0, 9,
	//		OpCall, 0, 4,
	//
	//		OpReturn,
	//	},
	//	f.ByteCode)
	//
	//rs, err := f.Run(nil)
	//require.NoError(t, err)
	//require.Equal(t, true, rs.Result())
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

func TestCompileDapp(t *testing.T) {
	source := "AAIDAAAAAAAAAAkIARIAEgMKAQEAAAAAAAAAAgAAAAFpAQAAAAdkZXBvc2l0AAAAAAQAAAADcG10CQEAAAAHZXh0cmFjdAAAAAEIBQAAAAFpAAAAB3BheW1lbnQDCQEAAAAJaXNEZWZpbmVkAAAAAQgFAAAAA3BtdAAAAAdhc3NldElkCQAAAgAAAAECAAAAIWNhbiBob2xkIHdhdmVzIG9ubHkgYXQgdGhlIG1vbWVudAQAAAAKY3VycmVudEtleQkAAlgAAAABCAgFAAAAAWkAAAAGY2FsbGVyAAAABWJ5dGVzBAAAAA1jdXJyZW50QW1vdW50BAAAAAckbWF0Y2gwCQAEGgAAAAIFAAAABHRoaXMFAAAACmN1cnJlbnRLZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAA0ludAQAAAABYQUAAAAHJG1hdGNoMAUAAAABYQAAAAAAAAAAAAQAAAAJbmV3QW1vdW50CQAAZAAAAAIFAAAADWN1cnJlbnRBbW91bnQIBQAAAANwbXQAAAAGYW1vdW50CQEAAAAIV3JpdGVTZXQAAAABCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACBQAAAApjdXJyZW50S2V5BQAAAAluZXdBbW91bnQFAAAAA25pbAAAAAFpAQAAAAh3aXRoZHJhdwAAAAEAAAAGYW1vdW50BAAAAApjdXJyZW50S2V5CQACWAAAAAEICAUAAAABaQAAAAZjYWxsZXIAAAAFYnl0ZXMEAAAADWN1cnJlbnRBbW91bnQEAAAAByRtYXRjaDAJAAQaAAAAAgUAAAAEdGhpcwUAAAAKY3VycmVudEtleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAFhBQAAAAckbWF0Y2gwBQAAAAFhAAAAAAAAAAAABAAAAAluZXdBbW91bnQJAABlAAAAAgUAAAANY3VycmVudEFtb3VudAUAAAAGYW1vdW50AwkAAGYAAAACAAAAAAAAAAAABQAAAAZhbW91bnQJAAACAAAAAQIAAAAeQ2FuJ3Qgd2l0aGRyYXcgbmVnYXRpdmUgYW1vdW50AwkAAGYAAAACAAAAAAAAAAAABQAAAAluZXdBbW91bnQJAAACAAAAAQIAAAASTm90IGVub3VnaCBiYWxhbmNlCQEAAAAMU2NyaXB0UmVzdWx0AAAAAgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgUAAAAKY3VycmVudEtleQUAAAAJbmV3QW1vdW50BQAAAANuaWwJAQAAAAtUcmFuc2ZlclNldAAAAAEJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyBQAAAAZhbW91bnQFAAAABHVuaXQFAAAAA25pbAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAACAUAAAACdHgAAAAPc2VuZGVyUHVibGljS2V54232jg=="
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

	script, err := CompileSimpleScript(tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	res, err := script.Run(env)
	require.NoError(t, err)
	assert.NotNil(t, res)
	r, ok := res.(ScriptResult)
	assert.True(t, ok)
	assert.Equal(t, true, r.Result())
}

/*


base64:AwoBAAAAAWYAAAABAAAAA2tleQQAAAAHJG1hdGNoMAkABBwAAAACBQAAAAR0aGlzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAACkJ5dGVWZWN0b3IEAAAAAWEFAAAAByRtYXRjaDAAAAAAAAAAAAEAAAAAAAAAAAAEAAAAAWEJAQAAAAFmAAAAAQIAAAABYQQAAAABYgkBAAAAAWYAAAABAgAAAAFiBAAAAAFjCQEAAAABZgAAAAECAAAAAWMEAAAAAWQJAQAAAAFmAAAAAQIAAAABZAQAAAABZQkBAAAAAWYAAAABAgAAAAFlAwkAAAAAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIFAAAAAWEFAAAAAWIFAAAAAWMFAAAAAWQFAAAAAWUAAAAAAAAAAAUJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAACAUAAAACdHgAAAAPc2VuZGVyUHVibGljS2V5B4xspLY=


*/

func Test2121(t *testing.T) {
	source := `AwoBAAAAAWYAAAABAAAAA2tleQQAAAAHJG1hdGNoMAkABBwAAAACBQAAAAR0aGlzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAACkJ5dGVWZWN0b3IEAAAAAWEFAAAAByRtYXRjaDAAAAAAAAAAAAEAAAAAAAAAAAAEAAAAAWEJAQAAAAFmAAAAAQIAAAABYQQAAAABYgkBAAAAAWYAAAABAgAAAAFiBAAAAAFjCQEAAAABZgAAAAECAAAAAWMEAAAAAWQJAQAAAAFmAAAAAQIAAAABZAQAAAABZQkBAAAAAWYAAAABAgAAAAFlAwkAAAAAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIFAAAAAWEFAAAAAWIFAAAAAWMFAAAAAWQFAAAAAWUAAAAAAAAAAAUJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAACAUAAAACdHgAAAAPc2VuZGVyUHVibGljS2V5B4xspLY=`
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
		thisFunc: func() rideType {
			return testTransferObject()
		},
	}

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	t.Log(detree(tree.Verifier))

	script, err := CompileSimpleScript(tree)
	require.NoError(t, err)
	assert.NotNil(t, script)

	res, err := script.Run(env)
	require.NoError(t, err)
	assert.NotNil(t, res)
	r, ok := res.(ScriptResult)
	assert.True(t, ok)
	assert.Equal(t, true, r.Result())
}

func detree(tree Node) string {
	s := &strings.Builder{}
	detree_(s, tree, 0)
	return s.String()
}

func detree_(s *strings.Builder, tree Node, shift int) {
	switch n := tree.(type) {
	case *FunctionDeclarationNode:
		s.WriteString(fmt.Sprintf("func %s(", n.Name))
		for _, a := range n.Arguments {
			s.WriteString(a)
			s.WriteString(",")
		}
		s.WriteString(") {\n")
		s.WriteString(strings.Repeat(" ", shift+1))
		detree_(s, n.Body, shift+1)
		s.WriteString("}\n")
		detree_(s, n.Block, shift)

	case *AssignmentNode:
		s.WriteString(fmt.Sprintf("let %s = ", n.Name))
		detree_(s, n.Expression, shift)
		s.WriteString("\n")
		detree_(s, n.Block, shift)

	case *ConditionalNode:
		s.WriteString(strings.Repeat(" ", shift*4))
		s.WriteString(fmt.Sprintf("if ("))
		detree_(s, n.Condition, shift)
		s.WriteString(") {\n")
		s.WriteString(strings.Repeat(" ", (shift+1)*4))
		detree_(s, n.TrueExpression, shift+1)
		s.WriteString("} else {\n")
		s.WriteString(strings.Repeat(" ", (shift+1)*4))
		detree_(s, n.FalseExpression, shift+1)
		s.WriteString("}\n")
	case *FunctionCallNode:
		s.WriteString(n.Name)
		s.WriteString("(")
		for _, a := range n.Arguments {
			detree_(s, a, shift)
			s.WriteString(",")
		}
		s.WriteString(")")
	case *ReferenceNode:
		s.WriteString(n.Name)
	case *StringNode:
		s.WriteString(`"`)
		s.WriteString(n.Value)
		s.WriteString(`"`)
	case *PropertyNode:
		detree_(s, n.Object, shift)
		s.WriteString(".")
		s.WriteString(n.Name)
	case *BooleanNode:
		s.WriteString(fmt.Sprintf("%t", n.Value))
	case *LongNode:
		s.WriteString(fmt.Sprintf("%d", n.Value))
	default:
		panic(fmt.Sprintf("unknown type %T", n))
	}
	//return s.String()
}
