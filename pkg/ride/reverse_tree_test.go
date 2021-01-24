package ride

import (
	"encoding/base64"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

func Test33(t *testing.T) {
	state := &MockSmartState{
		NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
			return byte_helpers.TransferWithProofs.Transaction, nil
		},
		RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
			t.Log("key: ", key)
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
	}
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
		{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=", env, true},
		{`V1: let i = 1; let s = "string"; toString(i) == s`, "BAQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABc6Y8UOc=", env, false},
		{`V3: let i = 12345; let s = "12345"; toString(i) == s`, "AwQAAAABaQAAAAAAAAAwOQQAAAABcwIAAAAFMTIzNDUJAAAAAAAAAgkAAaQAAAABBQAAAAFpBQAAAAFz1B1iCw==", env, true},
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
		{`func abc(addr: Address) = addr == tx.sender;abc(tx.sender)`, "BAoBAAAAA2FiYwAAAAEAAAAEYWRkcgkAAAAAAAACBQAAAARhZGRyCAUAAAACdHgAAAAGc2VuZGVyCQEAAAADYWJjAAAAAQgFAAAAAnR4AAAABnNlbmRlckJrXFI=", env, true},
		//{`let y = [{let x = 1;x}];true`, "BAQAAAABeQkABEwAAAACBAAAAAF4AAAAAAAAAAABBQAAAAF4BQAAAANuaWwGua/TXw==", env, true},
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
		require.NotNil(t, tree, test.comment)

		exe, err := CompileFlatTree(tree)
		require.NoError(t, err)

		res, err := exe.Verify(test.env)
		require.NoError(t, err, test.comment)
		require.NotNil(t, res, test.comment)

		t.Log(res.Calls())

		r, ok := res.(ScriptResult)
		require.True(t, ok, test.comment)
		require.Equal(t, test.res, r.Result(), test.comment)
	}
}

/*
func abc(key: String) = {
	let x = 1
	let y = 2
	x + y
}
*/
/*
func TestReverseFunc(t *testing.T) {
	n := &FunctionDeclarationNode{
		Name:      "abc",
		Arguments: []string{"key"},
		Body: &AssignmentNode{
			Name:       "x",
			Expression: &LongNode{Value: 1},
			Block: &AssignmentNode{
				Name:       "y",
				Expression: &LongNode{Value: 2},
				Block: &FunctionCallNode{
					Name: "+",
					Arguments: []Node{
						&ReferenceNode{Name: "x"},
						&ReferenceNode{Name: "y"},
					},
				},
			},
		},
	}

	rs := reverseTree(n, nil)

	require.Equal(t, &RFunc{
		Invocation: "",
		Name:       "abc",
		Arguments:  []string{"key"},
		Body: &RCall{
			Name: "+",
			Arguments: []RNode{
				&RRef{Name: "x"},
				&RRef{Name: "y"},
			},
			Assigments: []*RLet{
				{Name: "x", Body: &RLong{Value: 1}},
				{Name: "y", Body: &RLong{Value: 2}},
			},
		},
	}, rs)

}

*/

/*
func abc(key: String) = {
	match getInteger(this, key) {
		case a: Int =>
			a
		case _ =>
			0
}
*/
/*
func TestReverseFunc2(t *testing.T) {
	n := &FunctionDeclarationNode{
		Name:      "abc",
		Arguments: []string{"key"},
		Body: &AssignmentNode{
			Name: "$match0",
			Expression: &FunctionCallNode{
				Name: "1050",
				Arguments: []Node{
					&ReferenceNode{Name: "this"},
					&ReferenceNode{Name: "key"},
				},
			},
			Block: &ConditionalNode{
				Condition: &FunctionCallNode{
					Name: "1",
					Arguments: []Node{
						&ReferenceNode{Name: "$match0"},
						&StringNode{Value: "Int"},
					},
				},
				TrueExpression: &AssignmentNode{
					Name:       "a",
					Expression: &ReferenceNode{Name: "$match0"},
					Block:      &ReferenceNode{Name: "a"},
				},
				FalseExpression: &LongNode{Value: 0},
			},
		},
	}

	rs := reverseTree(n, nil)

	require.Equal(t, &RFunc{
		Invocation: "",
		Name:       "abc",
		Arguments:  []string{"key"},
		Body: &RCond{
			Cond: &RCall{
				Name: "1",
				Arguments: []RNode{
					&RRef{Name: "$match0"},
					&RString{Value: "Int"},
				},
			},
			True: &RRef{
				Name: "a",
				Assigments: []*RLet{
					{
						Name: "a",
						Body: &RRef{Name: "$match0"},
					},
				},
			},
			False: &RLong{Value: 0},
			Assigments: []*RLet{
				{Name: "$match0", Body: &RCall{
					Name: "1050",
					Arguments: []RNode{
						&RRef{Name: "this"},
						&RRef{Name: "key"},
					},
				}},
			},
		},
		//Assigments: []*RLet{
		//	{
		//		Name: "$match0",
		//		N:    0,
		//		Body: nil,
		//	},
		//},
	}, rs)
}
*/

/*
func TestReverse2(t *testing.T) {
	//{`V3: let x = true; x`, "BAQAAAABeAYFAAAAAXhUb/5M", env, true},
	source := `BAQAAAABeAYFAAAAAXhUb/5M`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	require.NotNil(t, tree)

	reversed := reverseTree3(tree.Verifier, nil, nil)
	require.Equal(t, []RNode{
		&RDef{Name: "x"},
		&RReferenceNode{Name: "x"},
		&RLet{Name: "x"},
		&RConst{Value: rideBoolean(true)},
	}, reversed)

	//env := &MockRideEnvironment{
	//	transactionFunc: testExchangeWithProofsToObject,
	//}
	//
	//rs, err := script.Run(env, nil)
	//require.NoError(t, err)
	//require.Equal(t, 2, len(rs.Calls()))
	//require.Equal(t, rs.Result(), true)
}
*/

/*
{-# STDLIB_VERSION 3 #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# CONTENT_TYPE EXPRESSION #-}

match (tx) {
    case e:ExchangeTransaction => isDefined(e.sellOrder.assetPair.priceAsset)
    case _ => throw("err")
  }
*/
func TestMultipleProperty2(t *testing.T) {
	source := `AwQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE0V4Y2hhbmdlVHJhbnNhY3Rpb24EAAAAAWUFAAAAByRtYXRjaDAJAQAAAAlpc0RlZmluZWQAAAABCAgIBQAAAAFlAAAACXNlbGxPcmRlcgAAAAlhc3NldFBhaXIAAAAKcHJpY2VBc3NldAkAAAIAAAABAgAAAANlcnIsqB0K`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	require.NotNil(t, tree)

	script, err := CompileFlatTree(tree)
	require.NoError(t, err)
	require.NotNil(t, script)

	env := &MockRideEnvironment{
		transactionFunc: testExchangeWithProofsToObject,
	}

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.Equal(t, rs.Result(), true)
}

func TestProperty2(t *testing.T) {
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

		script, err := CompileFlatTree(tree)
		require.NoError(t, err)
		require.NotNil(t, script)

		env := &MockRideEnvironment{
			transactionFunc: testExchangeWithProofsToObject,
		}

		require.Equal(t,
			[]byte{
				OpReturn,
				OpReturn,
				OpRef, 255, 255,
				OpRef, 0, 2,
				OpProperty,
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

		script, err := CompileFlatTree(tree)
		require.NoError(t, err)
		require.NotNil(t, script)

		env := &MockRideEnvironment{
			transactionFunc: testExchangeWithProofsToObject,
		}

		require.Equal(t,
			[]byte{
				OpReturn,
				OpReturn,
				OpRef, 255, 255,
				OpRef, 0, 2,
				OpProperty,
				OpRef, 0, 3,
				OpProperty,
				OpReturn,
			},
			script.ByteCode)
		_, err = script.run(env, nil)
		require.NoError(t, err)
	})
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
func TestDappMultipleFunctions2(t *testing.T) {
	source := "AAIDAAAAAAAAAAwIARIDCgEIEgMKAQgAAAAAAAAAAgAAAAFpAQAAAANhYmMAAAABAAAACHF1ZXN0aW9uCQEAAAAIV3JpdGVTZXQAAAABCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACAgAAAAFhAAAAAAAAAAAFBQAAAANuaWwAAAABaQEAAAADY2JhAAAAAQAAAAhxdWVzdGlvbgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAABYQAAAAAAAAAABgUAAAADbmlsAAAAAFEpRso="

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	require.NotNil(t, tree)

	script, err := CompileFlatTree(tree)
	require.NoError(t, err)
	require.NotNil(t, script)

	rs, err := script.Invoke(nil, "abc", []rideType{rideString(""), rideString("")})
	require.NoError(t, err)

	require.Equal(t, true, rs.Result())
	require.Equal(t,
		[]proto.ScriptAction{
			&proto.DataEntryScriptAction{
				Entry: &proto.IntegerDataEntry{Value: 5, Key: "a"},
			},
		}, []proto.ScriptAction(rs.ScriptActions()))

	rs, err = script.Invoke(nil, "cba", []rideType{rideString(""), rideString("")})
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
func TestIfStmt2(t *testing.T) {
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

	script, err := CompileFlatTree(tree)
	require.NoError(t, err)

	res, err := script.Verify(env)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)

	for _, l := range r.calls {
		t.Log(l)
	}

	require.Equal(t, true, r.Result())
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
func TestDapp3(t *testing.T) {
	source := `AAIDAAAAAAAAAAkIARIAEgMKAQEAAAAAAAAAAgAAAAFpAQAAAAdkZXBvc2l0AAAAAAQAAAADcG10CQEAAAAHZXh0cmFjdAAAAAEIBQAAAAFpAAAAB3BheW1lbnQDCQEAAAAJaXNEZWZpbmVkAAAAAQgFAAAAA3BtdAAAAAdhc3NldElkCQAAAgAAAAECAAAAIWNhbiBob2xkIHdhdmVzIG9ubHkgYXQgdGhlIG1vbWVudAQAAAAKY3VycmVudEtleQkAAlgAAAABCAgFAAAAAWkAAAAGY2FsbGVyAAAABWJ5dGVzBAAAAA1jdXJyZW50QW1vdW50BAAAAAckbWF0Y2gwCQAEGgAAAAIFAAAABHRoaXMFAAAACmN1cnJlbnRLZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAA0ludAQAAAABYQUAAAAHJG1hdGNoMAUAAAABYQAAAAAAAAAAAAQAAAAJbmV3QW1vdW50CQAAZAAAAAIFAAAADWN1cnJlbnRBbW91bnQIBQAAAANwbXQAAAAGYW1vdW50CQEAAAAIV3JpdGVTZXQAAAABCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACBQAAAApjdXJyZW50S2V5BQAAAAluZXdBbW91bnQFAAAAA25pbAAAAAFpAQAAAAh3aXRoZHJhdwAAAAEAAAAGYW1vdW50BAAAAApjdXJyZW50S2V5CQACWAAAAAEICAUAAAABaQAAAAZjYWxsZXIAAAAFYnl0ZXMEAAAADWN1cnJlbnRBbW91bnQEAAAAByRtYXRjaDAJAAQaAAAAAgUAAAAEdGhpcwUAAAAKY3VycmVudEtleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAFhBQAAAAckbWF0Y2gwBQAAAAFhAAAAAAAAAAAABAAAAAluZXdBbW91bnQJAABlAAAAAgUAAAANY3VycmVudEFtb3VudAUAAAAGYW1vdW50AwkAAGYAAAACAAAAAAAAAAAABQAAAAZhbW91bnQJAAACAAAAAQIAAAAeQ2FuJ3Qgd2l0aGRyYXcgbmVnYXRpdmUgYW1vdW50AwkAAGYAAAACAAAAAAAAAAAABQAAAAluZXdBbW91bnQJAAACAAAAAQIAAAASTm90IGVub3VnaCBiYWxhbmNlCQEAAAAMU2NyaXB0UmVzdWx0AAAAAgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgUAAAAKY3VycmVudEtleQUAAAAJbmV3QW1vdW50BQAAAANuaWwJAQAAAAtUcmFuc2ZlclNldAAAAAEJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyBQAAAAZhbW91bnQFAAAABHVuaXQFAAAAA25pbAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAACAUAAAACdHgAAAAPc2VuZGVyUHVibGljS2V54232jg==`
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
	require.NotNil(t, tree)

	script, err := CompileFlatTree(tree)
	require.NoError(t, err)
	require.NotNil(t, script)
	require.Equal(t, 4, len(script.EntryPoints))

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.NotNil(t, rs)
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

@Verifier(tx)
func verify () = sigVerify(tx.bodyBytes, tx.proofs[0], tx.senderPublicKey)
*/
func TestDappVerifier(t *testing.T) {
	source := `AAIDAAAAAAAAAAQIARIAAAAAAAAAAAEAAAABaQEAAAAHZGVwb3NpdAAAAAAEAAAAA3BtdAkBAAAAB2V4dHJhY3QAAAABCAUAAAABaQAAAAdwYXltZW50AwkBAAAACWlzRGVmaW5lZAAAAAEIBQAAAANwbXQAAAAHYXNzZXRJZAkAAAIAAAABAgAAACFjYW4gaG9sZCB3YXZlcyBvbmx5IGF0IHRoZSBtb21lbnQEAAAACmN1cnJlbnRLZXkJAAJYAAAAAQgIBQAAAAFpAAAABmNhbGxlcgAAAAVieXRlcwQAAAANY3VycmVudEFtb3VudAQAAAAHJG1hdGNoMAkABBoAAAACBQAAAAR0aGlzBQAAAApjdXJyZW50S2V5AwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAANJbnQEAAAAAWEFAAAAByRtYXRjaDAFAAAAAWEAAAAAAAAAAAAEAAAACW5ld0Ftb3VudAkAAGQAAAACBQAAAA1jdXJyZW50QW1vdW50CAUAAAADcG10AAAABmFtb3VudAkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgUAAAAKY3VycmVudEtleQUAAAAJbmV3QW1vdW50BQAAAANuaWwAAAABAAAAAnR4AQAAAAZ2ZXJpZnkAAAAACQAB9AAAAAMIBQAAAAJ0eAAAAAlib2R5Qnl0ZXMJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAAgFAAAAAnR4AAAAD3NlbmRlclB1YmxpY0tleVRzVhY=`
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
	require.NotNil(t, tree)

	script, err := CompileFlatTree(tree)
	require.NoError(t, err)
	require.NotNil(t, script)

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.NotNil(t, rs)
}

func TestLetInLet2(t *testing.T) {
	//{`let x = {let y = true;y}x`, `BAQAAAABeAQAAAABeQYFAAAAAXkFAAAAAXhCPj2C`, nil, true},
	source := `AwQAAAABeAQAAAABeQAAAAAAAAAABQYFAAAAAXhy1aZr`
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
	require.NotNil(t, tree)

	script, err := CompileFlatTree(tree)
	require.NoError(t, err)
	require.NotNil(t, script)

	rs, err := script.Verify(env)
	require.NoError(t, err)
	require.NotNil(t, rs)
}

func TestFlatProperty(t *testing.T) {
	n := &PropertyNode{
		Name: "assetPair",
		Object: &PropertyNode{
			Name:   "sellOrder",
			Object: &ReferenceNode{Name: "tx"},
		}}
	rs := flatProperty(n)
	require.Equal(t, []RNode{
		&RReferenceNode{Name: "tx"},
		&RConst{Value: rideString("sellOrder")},
		&RProperty{},
		&RConst{Value: rideString("assetPair")},
		&RProperty{},
	}, rs)
}
