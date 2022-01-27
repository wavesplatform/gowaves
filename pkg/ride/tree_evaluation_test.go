package ride

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

var (
	te = &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{}
		},
		rideV6ActivatedFunc: noRideV6,
	}
)

func TestSimpleScriptEvaluation(t *testing.T) {
	state := &MockSmartState{NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
		return testTransferWithProofs(), nil
	}}
	env := &mockRideEnvironment{
		transactionFunc: testTransferObject,
		stateFunc: func() types.SmartState {
			return state
		},
		schemeFunc: func() byte {
			return 'T'
		},
		rideV6ActivatedFunc: noRideV6,
	}
	for _, test := range []struct {
		comment string
		source  string
		env     environment
		res     bool
	}{
		{`V1: true`, "AQa3b8tH", te, true},
		{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn", te, true},
		{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=", te, true},
		{`V1: let i = 1; let s = "string"; toString(i) == s`, "AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=", te, false},
		{`V3: let i = 12345; let s = "12345"; toString(i) == s`, "AwQAAAABaQAAAAAAAAAwOQQAAAABcwIAAAAFMTIzNDUJAAAAAAAAAgkAAaQAAAABBQAAAAFpBQAAAAFz1B1iCw==", te, true},
		{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E", te, true},
		{`V3: if (false) then {let r = true; r} else {let r = false; r}`, "AwMHBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXI+tfo1", te, false},
		{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==", te, true},
		{`V3: let x = 1; func add(i: Int) = i + 1; add(x) == 2`, "AwQAAAABeAAAAAAAAAAAAQoBAAAAA2FkZAAAAAEAAAABaQkAAGQAAAACBQAAAAFpAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAF4AAAAAAAAAAACfr6U6w==", te, true},
		{`V3: let b = base16'0000000000000001'; func add(b: ByteVector) = toInt(b) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAWIJAABkAAAAAgkABLEAAAABBQAAAAFiAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAACX00biA==", te, true},
		{`V3: let b = base16'0000000000000001'; func add(v: ByteVector) = toInt(v) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAXYJAABkAAAAAgkABLEAAAABBQAAAAF2AAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAACI7gYxg==", te, true},
		{`V3: let b = base16'0000000000000001'; func add(v: ByteVector) = toInt(b) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAXYJAABkAAAAAgkABLEAAAABBQAAAAFiAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAAChRvwnQ==", te, true},
		{`V3: let data = base64'AAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLw=='; func getStock(data:ByteVector) = toInt(take(drop(data, 8), 8)); getStock(data) == 1`, `AwQAAAAEZGF0YQEAAABwAAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLwoBAAAACGdldFN0b2NrAAAAAQAAAARkYXRhCQAEsQAAAAEJAADJAAAAAgkAAMoAAAACBQAAAARkYXRhAAAAAAAAAAAIAAAAAAAAAAAICQAAAAAAAAIJAQAAAAhnZXRTdG9jawAAAAEFAAAABGRhdGEAAAAAAAAAAAFCtabi`, te, true},
		{`V3: let ref = 999; func g(a: Int) = ref; func f(ref: Int) = g(ref); f(1) == 999`, "AwQAAAADcmVmAAAAAAAAAAPnCgEAAAABZwAAAAEAAAABYQUAAAADcmVmCgEAAAABZgAAAAEAAAADcmVmCQEAAAABZwAAAAEFAAAAA3JlZgkAAAAAAAACCQEAAAABZgAAAAEAAAAAAAAAAAEAAAAAAAAAA+fjknmW", te, true},
		{`V4: let ref = 999; func f(ref: Int) = {let a = ref; ref }; f(1) == 999`, "BAQAAAADcmVmAAAAAAAAAAPnCgEAAAABZgAAAAEAAAADcmVmBAAAAAFhBQAAAANyZWYFAAAAA3JlZgkAAAAAAAACCQEAAAABZgAAAAEAAAAAAAAAAAEAAAAAAAAAA+eeAF6w", te, false},
		{`let x = 5; 6 > 4`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGAAAAAAAAAAAEYSW6XA==`, te, true},
		{`let x = 5; 6 > x`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGBQAAAAF4Gh24hw==`, te, true},
		{`let x = 5; 6 >= x`, `AQQAAAABeAAAAAAAAAAABQkAAGcAAAACAAAAAAAAAAAGBQAAAAF4jlxXHA==`, te, true},
		{`false`, `AQfeYll6`, te, false},
		{`let x =  throw(); true`, `AQQAAAABeAkBAAAABXRocm93AAAAAAa7bgf4`, te, true},
		{`let x =  throw(); true || x`, `AQQAAAABeAkBAAAABXRocm93AAAAAAMGBgUAAAABeKRnLds=`, te, true},
		{`tx.id == base58''`, `AQkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAAJBtD70=`, env, false},
		{`tx.id == base58'H5C8bRzbUTMePSDVVxjiNKDUwk6CKzfZGTP2Rs7aCjsV'`, `BAkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAIO7N5luRDUgN1SJ4kFmy/Ni8U2H6k7bpszok5tlLlRVgHwSHyg==`, env, true},
		{`let x = tx.id == base58'a';true`, `AQQAAAABeAkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAASEGjR0kcA==`, env, true},
		{`V4: if tx.proofs[0] != base58'' then tx.proofs[1] == base58'' else false`, `BAMJAQAAAAIhPQAAAAIJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAAEAAAAACQAAAAAAAAIJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAQEAAAAAB106gzM=`, env, true},
		{`match tx {case t : TransferTransaction | MassTransferTransaction | ExchangeTransaction => true; case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABNFeGNoYW5nZVRyYW5zYWN0aW9uBgMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24GCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAGB6Ilvok=`, env, true},
		{`V2: match transactionById(tx.id) {case  t: Unit => false case _ => true}`, `AgQAAAAHJG1hdGNoMAkAA+gAAAABCAUAAAACdHgAAAACaWQDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABFVuaXQEAAAAAXQFAAAAByRtYXRjaDAHBp9TFcQ=`, env, true},
		{`Up() == UP`, `AwkAAAAAAAACCQEAAAACVXAAAAAABQAAAAJVUPGUxeg=`, te, true},
		{`HalfUp() == HALFUP`, `AwkAAAAAAAACCQEAAAAGSGFsZlVwAAAAAAUAAAAGSEFMRlVQbUfpTQ==`, te, true},
		{`let a0 = NoAlg() == NOALG; let a1 = Md5() == MD5; let a2 = Sha1() == SHA1; let a3 = Sha224() == SHA224; let a4 = Sha256() == SHA256; let a5 = Sha384() == SHA384; let a6 = Sha512() == SHA512; let a7 = Sha3224() == SHA3224; let a8 = Sha3256() == SHA3256; let a9 = Sha3384() == SHA3384; let a10 = Sha3512() == SHA3512; a0 && a1 && a2 && a3 && a4 && a5 && a6 && a7 && a8 && a9 && a10`, `AwQAAAACYTAJAAAAAAAAAgkBAAAABU5vQWxnAAAAAAUAAAAFTk9BTEcEAAAAAmExCQAAAAAAAAIJAQAAAANNZDUAAAAABQAAAANNRDUEAAAAAmEyCQAAAAAAAAIJAQAAAARTaGExAAAAAAUAAAAEU0hBMQQAAAACYTMJAAAAAAAAAgkBAAAABlNoYTIyNAAAAAAFAAAABlNIQTIyNAQAAAACYTQJAAAAAAAAAgkBAAAABlNoYTI1NgAAAAAFAAAABlNIQTI1NgQAAAACYTUJAAAAAAAAAgkBAAAABlNoYTM4NAAAAAAFAAAABlNIQTM4NAQAAAACYTYJAAAAAAAAAgkBAAAABlNoYTUxMgAAAAAFAAAABlNIQTUxMgQAAAACYTcJAAAAAAAAAgkBAAAAB1NoYTMyMjQAAAAABQAAAAdTSEEzMjI0BAAAAAJhOAkAAAAAAAACCQEAAAAHU2hhMzI1NgAAAAAFAAAAB1NIQTMyNTYEAAAAAmE5CQAAAAAAAAIJAQAAAAdTaGEzMzg0AAAAAAUAAAAHU0hBMzM4NAQAAAADYTEwCQAAAAAAAAIJAQAAAAdTaGEzNTEyAAAAAAUAAAAHU0hBMzUxMgMDAwMDAwMDAwMFAAAAAmEwBQAAAAJhMQcFAAAAAmEyBwUAAAACYTMHBQAAAAJhNAcFAAAAAmE1BwUAAAACYTYHBQAAAAJhNwcFAAAAAmE4BwUAAAACYTkHBQAAAANhMTAHRc/wAA==`, te, true},
		{`Unit() == unit`, `AwkAAAAAAAACCQEAAAAEVW5pdAAAAAAFAAAABHVuaXTstg1G`, te, true},
		{`V4: let x = 5; let y = x; let z = y; x == z`, `BAQAAAABeAAAAAAAAAAABQQAAAABeQUAAAABeAQAAAABegUAAAABeQkAAAAAAAACBQAAAAF4BQAAAAF6rBJg5A==`, te, true},
		{`V3: func fn(name: String) = name; fn("bbb") == "aaa"`, `AwoBAAAAAmZuAAAAAQAAAARuYW1lBQAAAARuYW1lCQAAAAAAAAIJAQAAAAJmbgAAAAECAAAAA2JiYgIAAAADYWFhbCbxUQ==`, te, false},
		{`V3: let zz = "ccc"; func fn(name: String) = zz; fn("abc") == "ccc"`, `AwQAAAACenoCAAAAA2NjYwoBAAAAAmZuAAAAAQAAAARuYW1lBQAAAAJ6egkAAAAAAAACCQEAAAACZm4AAAABAgAAAANhYmMCAAAAA2NjYyBIzew=`, te, true},
		{`V4: stack1`, `BAoBAAAAAmYxAAAAAgAAAAF4AAAAAXkJAAEsAAAAAgUAAAABeAkAAaQAAAABBQAAAAF5CgEAAAACZjIAAAABAAAAAXgJAABkAAAAAgUAAAABeAAAAAAAAAAAAQoBAAAABG1haW4AAAABAAAAAXgEAAAAAnIyCQEAAAACZjIAAAABBQAAAAF4BAAAAAFzCQABpAAAAAEFAAAAAXgEAAAAAnIxCQEAAAACZjEAAAACBQAAAAFzBQAAAAJyMgkABEwAAAACBQAAAAJyMQkABEwAAAACBQAAAAJyMgUAAAADbmlsBAAAAAFyCQEAAAAEbWFpbgAAAAEAAAAAAAAAAAEDCQAAAAAAAAIJAAGRAAAAAgUAAAABcgAAAAAAAAAAAAIAAAACMTIJAAAAAAAAAgkAAZEAAAACBQAAAAFyAAAAAAAAAAABAAAAAAAAAAACB/VXFJo=`, te, true},
		{`V4: stack2`, `BAQAAAABYQEAAAAFAQIDBAUKAQAAAAJmMQAAAAEAAAABYQkAASwAAAACBQAAAAFhAgAAAAFfCgEAAAACZjIAAAABAAAAAWIAAAAAAAAAAAAKAQAAAAJmMwAAAAEAAAABYQkBAAAAAmYxAAAAAQUAAAABYQoBAAAABG1haW4AAAAABAAAAAFzAgAAAAN4eHgEAAAAAXgJAQAAAAJmMwAAAAEFAAAAAXMJAARMAAAAAgkBAAAAAmYxAAAAAQUAAAABcwkABEwAAAACBQAAAAF4BQAAAANuaWwJAAAAAAAAAgkAAZEAAAACCQEAAAAEbWFpbgAAAAAAAAAAAAAAAAACAAAABHh4eF9D/pzY`, te, true},
		{`V4: stack3`, `BAQAAAABYQIAAAABMAoBAAAAAmYxAAAAAQAAAAFhCgEAAAACZjIAAAABAAAAAWEKAQAAAAJmMwAAAAEAAAABYQkAASwAAAACBQAAAAFhAgAAAAExBAAAAAFiCQEAAAACZjMAAAABBQAAAAFhCQABLAAAAAIFAAAAAWICAAAAATIEAAAAAWMJAQAAAAJmMgAAAAEFAAAAAWEJAAEsAAAAAgUAAAABYwIAAAABMwkAAAAAAAACCQEAAAACZjEAAAABBQAAAAFhAgAAAAQwMTIz8IamIA==`, te, true},
		{`V4: func x(a: Int) = a + 1; func y(b: Int) = {let c = 1; b + x(c)}; y(1) == 3`, `BAoBAAAAAXgAAAABAAAAAWEJAABkAAAAAgUAAAABYQAAAAAAAAAAAQoBAAAAAXkAAAABAAAAAWIEAAAAAXIDCQAAZgAAAAIFAAAAAWIAAAAAAAAAAAAKAQAAAAF6AAAAAQAAAAFkCQAAZAAAAAIJAQAAAAF4AAAAAQUAAAABZAAAAAAAAAAAAQQAAAABYwAAAAAAAAAAAQkAAGQAAAACBQAAAAFiCQEAAAABegAAAAEFAAAAAWMAAAAAAAAAAAAFAAAAAXIJAAAAAAAAAgkBAAAAAXkAAAABAAAAAAAAAAABAAAAAAAAAAAEno4I3w==`, te, true},
		{`V4: let a = 1; let b = 2; let c = 3; let d = 4; let (x, y) = ((a+b), (c+d)); x + y == 10`, `BAQAAAABYQAAAAAAAAAAAQQAAAABYgAAAAAAAAAAAgQAAAABYwAAAAAAAAAAAwQAAAABZAAAAAAAAAAABAQAAAAJJHQwMTI2MTUzCQAFFAAAAAIJAABkAAAAAgUAAAABYQUAAAABYgkAAGQAAAACBQAAAAFjBQAAAAFkBAAAAAF4CAUAAAAJJHQwMTI2MTUzAAAAAl8xBAAAAAF5CAUAAAAJJHQwMTI2MTUzAAAAAl8yCQAAAAAAAAIJAABkAAAAAgUAAAABeAUAAAABeQAAAAAAAAAACrqIL8U=`, te, true},
	} {
		src, err := base64.StdEncoding.DecodeString(test.source)
		require.NoError(t, err, test.comment)

		tree, err := Parse(src)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, tree, test.comment)

		res, err := CallVerifier(test.env, tree)
		require.NoError(t, err, test.comment)
		require.NotNil(t, res, test.comment)

		r, ok := res.(ScriptResult)
		assert.True(t, ok, test.comment)
		assert.Equal(t, test.res, r.Result(), test.comment)
	}
}

func TestFunctionsEvaluation(t *testing.T) {
	d, err := crypto.NewDigestFromBase58("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	transfer := newTransferTransaction()
	exchange := newExchangeTransaction()
	data := newDataTransaction()
	require.NoError(t, err)
	env := &mockRideEnvironment{
		checkMessageLengthFunc: v3check,
		heightFunc: func() rideInt {
			return 5
		},
		libVersionFunc: func() int {
			return 3
		},
		schemeFunc: func() byte {
			return 'W'
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					if key == "integer" {
						return &proto.IntegerDataEntry{Key: "integer", Value: 100500}, nil
					}
					return nil, errors.New("not found")
				},
				RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
					if key == "boolean" {
						return &proto.BooleanDataEntry{Key: "boolean", Value: true}, nil
					}
					return nil, errors.New("not found")
				},
				RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
					if key == "binary" {
						return &proto.BinaryDataEntry{Key: "binary", Value: []byte("hello")}, nil
					}
					return nil, errors.New("not found")
				},
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					if key == "string" {
						return &proto.StringDataEntry{Key: "string", Value: "world"}, nil
					}
					return nil, errors.New("not found")
				},
				NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
					return 5, nil
				},
				NewestAssetBalanceFunc: func(account proto.Recipient, asset crypto.Digest) (uint64, error) {
					if asset == d {
						return 5, nil
					}
					return 0, nil
				},
				NewestTransactionByIDFunc: func(id []byte) (proto.Transaction, error) {
					return transfer, nil
				},
				NewestTransactionHeightByIDFunc: func(_ []byte) (uint64, error) {
					return 0, proto.ErrNotFound
				},
				IsNotFoundFunc: func(err error) bool {
					return true
				},
			}
		},
		takeStringFunc: v5takeString,
		transactionFunc: func() rideObject {
			obj, err := transferWithProofsToObject('W', transfer)
			if err != nil {
				panic(err)
			}
			return obj
		},
		validateInternalPaymentsFunc: func() bool {
			return false
		},
		rideV6ActivatedFunc: noRideV6,
	}
	envWithDataTX := &mockRideEnvironment{
		transactionFunc: func() rideObject {
			obj, err := dataWithProofsToObject('W', data)
			if err != nil {
				panic(err)
			}
			return obj
		},
		takeStringFunc: v5takeString,
		libVersionFunc: func() int {
			return 3
		},
		rideV6ActivatedFunc: noRideV6,
	}
	envWithExchangeTX := &mockRideEnvironment{
		transactionFunc: func() rideObject {
			obj, err := exchangeWithProofsToObject('W', exchange)
			if err != nil {
				panic(err)
			}
			return obj
		},
		takeStringFunc: v5takeString,
		libVersionFunc: func() int {
			return 3
		},
		rideV6ActivatedFunc: noRideV6,
	}
	for _, test := range []struct {
		name   string
		text   string
		script string
		env    environment
		result bool
		error  bool
	}{
		{`parseIntValue`, `parseInt("12345") == 12345`, `AwkAAAAAAAACCQAEtgAAAAECAAAABTEyMzQ1AAAAAAAAADA57cmovA==`, env, true, false},
		{`value`, `let c = if true then 1 else Unit(); value(c) == 1`, `AwQAAAABYwMGAAAAAAAAAAABCQEAAAAEVW5pdAAAAAAJAAAAAAAAAgkBAAAABXZhbHVlAAAAAQUAAAABYwAAAAAAAAAAARfpQ5M=`, env, true, false},
		{`valueOrErrorMessage`, `let c = if true then 1 else Unit(); valueOrErrorMessage(c, "ALARM!!!") == 1`, `AwQAAAABYwMGAAAAAAAAAAABCQEAAAAEVW5pdAAAAAAJAAAAAAAAAgkBAAAAE3ZhbHVlT3JFcnJvck1lc3NhZ2UAAAACBQAAAAFjAgAAAAhBTEFSTSEhIQAAAAAAAAAAAa5tVyw=`, te, true, false},
		{`addressFromString`, `addressFromString("12345") == unit`, `AwkAAAAAAAACCQEAAAARYWRkcmVzc0Zyb21TdHJpbmcAAAABAgAAAAUxMjM0NQUAAAAEdW5pdJEPLPE=`, env, true, false},
		{`addressFromString`, `addressFromString("3P9DEDP5VbyXQyKtXDUt2crRPn5B7gs6ujc") == Address(base58'3P9DEDP5VbyXQyKtXDUt2crRPn5B7gs6ujc')`, `AwkAAAAAAAACCQEAAAARYWRkcmVzc0Zyb21TdHJpbmcAAAABAgAAACMzUDlERURQNVZieVhReUt0WERVdDJjclJQbjVCN2dzNnVqYwkBAAAAB0FkZHJlc3MAAAABAQAAABoBV0/fzRv7GRFL0qw2njIBPHDG0DpGJ4ecv6EI6ng=`, env, true, false},
		{`addressFromStringValue`, `addressFromStringValue("3P9DEDP5VbyXQyKtXDUt2crRPn5B7gs6ujc") == Address(base58'3P9DEDP5VbyXQyKtXDUt2crRPn5B7gs6ujc')`, `AwkAAAAAAAACCQEAAAAcQGV4dHJVc2VyKGFkZHJlc3NGcm9tU3RyaW5nKQAAAAECAAAAIzNQOURFRFA1VmJ5WFF5S3RYRFV0MmNyUlBuNUI3Z3M2dWpjCQEAAAAHQWRkcmVzcwAAAAEBAAAAGgFXT9/NG/sZEUvSrDaeMgE8cMbQOkYnh5y/56rYHQ==`, env, true, false},
		{`getIntegerFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getInteger(a, "integer") == 100500`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQAEGgAAAAIFAAAAAWECAAAAB2ludGVnZXIAAAAAAAABiJTtgrwb`, env, true, false},
		{`getIntegerValueFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getIntegerValue(a, "integer") == 100500`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA1MCkAAAACBQAAAAFhAgAAAAdpbnRlZ2VyAAAAAAAAAYiUEnGoww==`, env, true, false},
		{`getBooleanFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getBoolean(a, "boolean") == true`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQAEGwAAAAIFAAAAAWECAAAAB2Jvb2xlYW4GQ1SwZw==`, env, true, false},
		{`getBooleanValueFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getBooleanValue(a, "boolean") == true`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA1MSkAAAACBQAAAAFhAgAAAAdib29sZWFuBiG4UlQ=`, env, true, false},
		{`getBinaryFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getBinary(a, "binary") == base16'68656c6c6f'`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQAEHAAAAAIFAAAAAWECAAAABmJpbmFyeQEAAAAFaGVsbG8AbKgo`, env, true, false},
		{`getBinaryValueFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getBinaryValue(a, "binary") == base16'68656c6c6f'`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA1MikAAAACBQAAAAFhAgAAAAZiaW5hcnkBAAAABWhlbGxvJ1b3yw==`, env, true, false},
		{`getStringFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getString(a, "string") == "world"`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA1MikAAAACBQAAAAFhAgAAAAZiaW5hcnkBAAAABWhlbGxvJ1b3yw==`, env, true, false},
		{`getStringValueFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getStringValue(a, "string") == "world"`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQAEHQAAAAIFAAAAAWECAAAABnN0cmluZwIAAAAFd29ybGSFdQnb`, env, true, false},
		{`getIntegerFromArrayByKey`, `let d = [DataEntry("integer", 100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getInteger(d, "integer") == 100500`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEAAAAAIFAAAAAWQCAAAAB2ludGVnZXIAAAAAAAABiJSeStXa`, env, true, false},
		{`getIntegerValueFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getIntegerValue(d, "integer") == 100500`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA0MCkAAAACBQAAAAFkAgAAAAdpbnRlZ2VyAAAAAAAAAYiUmn7ujg==`, env, true, false},
		{`getBooleanFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBoolean(d, "boolean") == true`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEQAAAAIFAAAAAWQCAAAAB2Jvb2xlYW4GaWuehg==`, env, true, false},
		{`getBooleanValueFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBooleanValue(d, "boolean") == true`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA0MSkAAAACBQAAAAFkAgAAAAdib29sZWFuBt3vwJY=`, env, true, false},
		{`getBinaryFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBinary(d, "binary") == base16'68656c6c6f'`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEgAAAAIFAAAAAWQCAAAABmJpbmFyeQEAAAAFaGVsbG+so7oZ`, env, true, false},
		{`getBinaryValueFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBinaryValue(d, "binary") == base16'68656c6c6f'`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA0MikAAAACBQAAAAFkAgAAAAZiaW5hcnkBAAAABWhlbGxvpcldYg==`, env, true, false},
		{`getStringFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getString(d, "string") == "world"`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEwAAAAIFAAAAAWQCAAAABnN0cmluZwIAAAAFd29ybGRFTMLs`, env, true, false},
		{`getStringValueFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getStringValue(d, "string") == "world"`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA0MykAAAACBQAAAAFkAgAAAAZzdHJpbmcCAAAABXdvcmxkCbSDLQ==`, env, true, false},
		{`getIntegerFromArrayByIndex`, `let d = [DataEntry("integer", 100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getInteger(d, 0) == 100500`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAKZ2V0SW50ZWdlcgAAAAIFAAAAAWQAAAAAAAAAAAAAAAAAAAABiJTdCjRc`, env, true, false},
		{`getIntegerValueFromArrayByIndex`, `let d = [DataEntry("integer", 100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getIntegerValue(d, 0) == 100500`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAVQGV4dHJVc2VyKGdldEludGVnZXIpAAAAAgUAAAABZAAAAAAAAAAAAAAAAAAAAAGIlOyDHCY=`, env, true, false},
		{`getBooleanFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBoolean(d, 1) == true`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAKZ2V0Qm9vbGVhbgAAAAIFAAAAAWQAAAAAAAAAAAEGlS0yug==`, env, true, false},
		{`getBooleanValueFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBooleanValue(d, 1) == true`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAVQGV4dHJVc2VyKGdldEJvb2xlYW4pAAAAAgUAAAABZAAAAAAAAAAAAQY8zZ6Y`, env, true, false},
		{`getBinaryFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBinary(d, 2) == base16'68656c6c6f'`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAJZ2V0QmluYXJ5AAAAAgUAAAABZAAAAAAAAAAAAgEAAAAFaGVsbG/jc7GJ`, env, true, false},
		{`getBinaryValueFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBinaryValue(d, 2) == base16'68656c6c6f'`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAUQGV4dHJVc2VyKGdldEJpbmFyeSkAAAACBQAAAAFkAAAAAAAAAAACAQAAAAVoZWxsbwxEPw4=`, env, true, false},
		{`getStringFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getString(d, 3) == "world"`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAJZ2V0U3RyaW5nAAAAAgUAAAABZAAAAAAAAAAAAwIAAAAFd29ybGTcG8rI`, env, true, false},
		{`getStringValueFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getStringValue(d, 3) == "world"`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAUQGV4dHJVc2VyKGdldFN0cmluZykAAAACBQAAAAFkAAAAAAAAAAADAgAAAAV3b3JsZOGBO8c=`, env, true, false},
		{`compare Recipient with Address`, `let a = Address(base58'3PKpKgcwArHQVmYWUg6Ljxx31VueBStUKBR'); match tx {case tt: TransferTransaction => tt.recipient == a case _ => false}`, `AwQAAAABYQkBAAAAB0FkZHJlc3MAAAABAQAAABoBV8Q0LvvkEO83LtpdRUhgK760itMpcq1W7AQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR0BQAAAAckbWF0Y2gwCQAAAAAAAAIIBQAAAAJ0dAAAAAlyZWNpcGllbnQFAAAAAWEHQOLkRA==`, env, false, false},
		{`compare Recipient with Address`, `let a = Address(base58'3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3'); match tx {case tt: TransferTransaction => tt.recipient == a case _ => false}`, `AwQAAAABYQkBAAAAB0FkZHJlc3MAAAABAQAAABoBVwX3L9Q7Ao0/8ZNhoE70/41bHPBwqbd27gQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR0BQAAAAckbWF0Y2gwCQAAAAAAAAIIBQAAAAJ0dAAAAAlyZWNpcGllbnQFAAAAAWEHd9vYmA==`, env, true, false},
		{`compare Address with Recipient`, `let a = Address(base58'3PKpKgcwArHQVmYWUg6Ljxx31VueBStUKBR'); match tx {case tt: TransferTransaction => a == tt.recipient case _ => false}`, `AwQAAAABYQkBAAAAB0FkZHJlc3MAAAABAQAAABoBV8Q0LvvkEO83LtpdRUhgK760itMpcq1W7AQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR0BQAAAAckbWF0Y2gwCQAAAAAAAAIFAAAAAWEIBQAAAAJ0dAAAAAlyZWNpcGllbnQHG1tX4w==`, env, false, false},
		{`compare Address with Recipient`, `let a = Address(base58'3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3'); match tx {case tt: TransferTransaction => a == tt.recipient case _ => false}`, `AwQAAAABYQkBAAAAB0FkZHJlc3MAAAABAQAAABoBVwX3L9Q7Ao0/8ZNhoE70/41bHPBwqbd27gQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR0BQAAAAckbWF0Y2gwCQAAAAAAAAIFAAAAAWEIBQAAAAJ0dAAAAAlyZWNpcGllbnQHw8RWfw==`, env, true, false},

		{`getIntegerFromDataTransactionByKey`, `match tx {case d: DataTransaction => extract(getInteger(d.data, "integer")) == 100500 case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABZAUAAAAHJG1hdGNoMAkAAAAAAAACCQEAAAAHZXh0cmFjdAAAAAEJAAQQAAAAAggFAAAAAWQAAAAEZGF0YQIAAAAHaW50ZWdlcgAAAAAAAAGIlAfN4Sfl`, envWithDataTX, true, false},
		{`getIntegerFromDataTransactionByKey`, `match tx {case dt: DataTransaction => let a = match getInteger(dt.data, "someKey") {case v: Int => v case _ => -1}; a >= 0 case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAACZHQFAAAAByRtYXRjaDAEAAAAAWEEAAAAByRtYXRjaDEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAAB3NvbWVLZXkDCQAAAQAAAAIFAAAAByRtYXRjaDECAAAAA0ludAQAAAABdgUAAAAHJG1hdGNoMQUAAAABdgD//////////wkAAGcAAAACBQAAAAFhAAAAAAAAAAAAB1mStww=`, envWithDataTX, true, false},
		{`getIntegerFromDataTransactionByKey`, `match tx {case dt: DataTransaction => let x = match getInteger(dt.data, "someKey") {case i: Int => true case _ => false};let y = match getInteger(dt.data, "someKey") {case v: Int => v case _ => -1}; x && y >= 0 case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAACZHQFAAAAByRtYXRjaDAEAAAAAXgEAAAAByRtYXRjaDEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAAB3NvbWVLZXkDCQAAAQAAAAIFAAAAByRtYXRjaDECAAAAA0ludAQAAAABaQUAAAAHJG1hdGNoMQYHBAAAAAF5BAAAAAckbWF0Y2gxCQAEEAAAAAIIBQAAAAJkdAAAAARkYXRhAgAAAAdzb21lS2V5AwkAAAEAAAACBQAAAAckbWF0Y2gxAgAAAANJbnQEAAAAAXYFAAAAByRtYXRjaDEFAAAAAXYA//////////8DBQAAAAF4CQAAZwAAAAIFAAAAAXkAAAAAAAAAAAAHB5sznFY=`, envWithDataTX, true, false},
		{`matchIntegerFromDataTransactionByKey`, `let x = match tx {case d: DataTransaction => match getInteger(d.data, "integer") {case i: Int => i case _ => 0}case _ => 0}; x == 100500`, `AQQAAAABeAQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABZAUAAAAHJG1hdGNoMAQAAAAHJG1hdGNoMQkABBAAAAACCAUAAAABZAAAAARkYXRhAgAAAAdpbnRlZ2VyAwkAAAEAAAACBQAAAAckbWF0Y2gxAgAAAANJbnQEAAAAAWkFAAAAByRtYXRjaDEFAAAAAWkAAAAAAAAAAAAAAAAAAAAAAAAJAAAAAAAAAgUAAAABeAAAAAAAAAGIlApOoB4=`, envWithDataTX, true, false},
		{`matchIntegerFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); let i = getInteger(a, "integer"); let x = match i {case i: Int => i case _ => 0}; x == 100500`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwQAAAABaQkABBoAAAACBQAAAAFhAgAAAAdpbnRlZ2VyBAAAAAF4BAAAAAckbWF0Y2gwBQAAAAFpAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAANJbnQEAAAAAWkFAAAAByRtYXRjaDAFAAAAAWkAAAAAAAAAAAAJAAAAAAAAAgUAAAABeAAAAAAAAAGIlKWtlDk=`, env, true, false},
		{`ifIntegerFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); let i = getInteger(a, "integer"); let x = if i != 0 then i else 0; x == 100500`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwQAAAABaQkABBoAAAACBQAAAAFhAgAAAAdpbnRlZ2VyBAAAAAF4AwkBAAAAAiE9AAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQAAAAAAAAAAAAkAAAAAAAACBQAAAAF4AAAAAAAAAYiU1cZgMA==`, env, true, false},

		{`string concatenation`, `let a = base16'cafe'; let b = base16'bebe'; toBase58String(a) + "/" + toBase58String(b) == "GSy/FWu"`, `AwQAAAABYQEAAAACyv4EAAAAAWIBAAAAAr6+CQAAAAAAAAIJAAEsAAAAAgkAASwAAAACCQACWAAAAAEFAAAAAWECAAAAAS8JAAJYAAAAAQUAAAABYgIAAAAHR1N5L0ZXdc2NqKQ=`, env, true, false},
		{`match on ByteVector`, `match tx {case etx: ExchangeTransaction => match etx.sellOrder.assetPair.amountAsset {case ByteVector => true case _ => false} case _ => false}`, `AwQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE0V4Y2hhbmdlVHJhbnNhY3Rpb24EAAAAA2V0eAUAAAAHJG1hdGNoMAQAAAAHJG1hdGNoMQgICAUAAAADZXR4AAAACXNlbGxPcmRlcgAAAAlhc3NldFBhaXIAAAALYW1vdW50QXNzZXQEAAAACkJ5dGVWZWN0b3IFAAAAByRtYXRjaDEGB76y+jI=`, envWithExchangeTX, true, false},

		{`3P8M8XGF2uzDazV5fzdKNxrbC3YqCWScKxw`, ``, `AwoBAAAAGVJlbW92ZVVuZGVyc2NvcmVJZlByZXNlbnQAAAABAAAACXJlbWFpbmluZwMJAABmAAAAAgkAATEAAAABBQAAAAlyZW1haW5pbmcAAAAAAAAAAAAJAAEwAAAAAgUAAAAJcmVtYWluaW5nAAAAAAAAAAABBQAAAAlyZW1haW5pbmcKAQAAABJQYXJzZU5leHRBdHRyaWJ1dGUAAAABAAAACXJlbWFpbmluZwQAAAABcwkAATEAAAABBQAAAAlyZW1haW5pbmcDCQAAZgAAAAIFAAAAAXMAAAAAAAAAAAAEAAAAAm5uCQEAAAANcGFyc2VJbnRWYWx1ZQAAAAEJAAEvAAAAAgUAAAAJcmVtYWluaW5nAAAAAAAAAAACBAAAAAF2CQABLwAAAAIJAAEwAAAAAgUAAAAJcmVtYWluaW5nAAAAAAAAAAACBQAAAAJubgQAAAAMdG1wUmVtYWluaW5nCQABMAAAAAIFAAAACXJlbWFpbmluZwkAAGQAAAACBQAAAAJubgAAAAAAAAAAAgQAAAAOcmVtYWluaW5nU3RhdGUJAQAAABlSZW1vdmVVbmRlcnNjb3JlSWZQcmVzZW50AAAAAQUAAAAMdG1wUmVtYWluaW5nCQAETAAAAAIFAAAAAXYJAARMAAAAAgUAAAAOcmVtYWluaW5nU3RhdGUFAAAAA25pbAkAAAIAAAABAgAAADRFbXB0eSBzdHJpbmcgd2FzIHBhc3NlZCBpbnRvIHBhcnNlTmV4dEF0dHJpYnV0ZSBmdW5jCgEAAAATUGFyc2VHYW1lUmF3RGF0YVN0cgAAAAEAAAALcmF3U3RhdGVTdHIEAAAACWdhbWVTdGF0ZQkBAAAAElBhcnNlTmV4dEF0dHJpYnV0ZQAAAAEFAAAAC3Jhd1N0YXRlU3RyBAAAAAxwbGF5ZXJDaG9pY2UJAQAAABJQYXJzZU5leHRBdHRyaWJ1dGUAAAABCQABkQAAAAIFAAAACWdhbWVTdGF0ZQAAAAAAAAAAAQQAAAAOcGxheWVyUHViS2V5NTgJAQAAABJQYXJzZU5leHRBdHRyaWJ1dGUAAAABCQABkQAAAAIFAAAADHBsYXllckNob2ljZQAAAAAAAAAAAQQAAAANc3RhcnRlZEhlaWdodAkBAAAAElBhcnNlTmV4dEF0dHJpYnV0ZQAAAAEJAAGRAAAAAgUAAAAOcGxheWVyUHViS2V5NTgAAAAAAAAAAAEEAAAABndpbkFtdAkBAAAAElBhcnNlTmV4dEF0dHJpYnV0ZQAAAAEJAAGRAAAAAgUAAAANc3RhcnRlZEhlaWdodAAAAAAAAAAAAQkABEwAAAACCQABkQAAAAIFAAAACWdhbWVTdGF0ZQAAAAAAAAAAAAkABEwAAAACCQABkQAAAAIFAAAADHBsYXllckNob2ljZQAAAAAAAAAAAAkABEwAAAACCQABkQAAAAIFAAAADnBsYXllclB1YktleTU4AAAAAAAAAAAACQAETAAAAAIJAAGRAAAAAgUAAAANc3RhcnRlZEhlaWdodAAAAAAAAAAAAAkABEwAAAACCQABkQAAAAIFAAAABndpbkFtdAAAAAAAAAAAAAUAAAADbmlsCQAAAAAAAAIJAQAAABNQYXJzZUdhbWVSYXdEYXRhU3RyAAAAAQIAAABWMDNXT05fMDUzNTY0Ml80NDM4OXBhNmlOaHgxaEZEcU5abVNBVEp1ZldaMUVMbUtkOUh4eXpQUUtIdWMzXzA3MTYxMDU1N18wOTExNDAwMDAwMF8wMTYJAARMAAAAAgIAAAADV09OCQAETAAAAAICAAAABTM1NjQyCQAETAAAAAICAAAALDM4OXBhNmlOaHgxaEZEcU5abVNBVEp1ZldaMUVMbUtkOUh4eXpQUUtIdWMzCQAETAAAAAICAAAABzE2MTA1NTcJAARMAAAAAgIAAAAJMTE0MDAwMDAwBQAAAANuaWyuDQ4Y`, envWithExchangeTX, true, false},
		//
		{"EQ", `5 == 5`, `AQkAAAAAAAACAAAAAAAAAAAFAAAAAAAAAAAFqWG0Fw==`, env, true, false},
		{"ISINSTANCEOF", `match tx {case t : TransferTransaction => true case _  => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAGB5yQ/+k=`, env, true, false},
		{`THROW`, `true && throw("mess")`, `AQMGCQAAAgAAAAECAAAABG1lc3MH7PDwAQ==`, env, false, true},
		{`SUM_LONG`, `1 + 1 > 0`, `AQkAAGYAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAABiJjSk`, env, true, false},
		{`SUB_LONG`, `2 - 1 > 0`, `AQkAAGYAAAACCQAAZQAAAAIAAAAAAAAAAAIAAAAAAAAAAAEAAAAAAAAAAABqsps1`, env, true, false},
		{`GT_LONG`, `1 > 0`, `AQkAAGYAAAACAAAAAAAAAAABAAAAAAAAAAAAyAIM4w==`, env, true, false},
		{`GE_LONG`, `1 >= 0`, `AQkAAGcAAAACAAAAAAAAAAABAAAAAAAAAAAAm30DnQ==`, env, true, false},
		{`MUL_LONG`, `2 * 2>0`, `AQkAAGYAAAACCQAAaAAAAAIAAAAAAAAAAAIAAAAAAAAAAAIAAAAAAAAAAABCMM5o`, env, true, false},
		{`DIV_LONG`, `4 / 2>0`, `AQkAAGYAAAACCQAAaQAAAAIAAAAAAAAAAAQAAAAAAAAAAAIAAAAAAAAAAAAadVma`, env, true, false},
		{`DIV_LONG`, `10000 / (27+121) == 67`, `AwkAAAAAAAACCQAAaQAAAAIAAAAAAAAAJxAJAABkAAAAAgAAAAAAAAAAGwAAAAAAAAAAeQAAAAAAAAAAQ1vSVaQ=`, env, true, false},
		{`DIV_LONG`, `((98750005 * (100 - 4)) / 100) == 94800004`, `AwkAAAAAAAACCQAAaQAAAAIJAABoAAAAAgAAAAAABeLONQkAAGUAAAACAAAAAAAAAABkAAAAAAAAAAAEAAAAAAAAAABkAAAAAAAFpoiEGMUZcA==`, env, true, false},
		{`MOD_LONG`, `-10 % 6>0`, `AQkAAGYAAAACCQAAagAAAAIA//////////YAAAAAAAAAAAYAAAAAAAAAAAB5rBSH`, env, true, false},
		{`MOD_LONG`, `10000 % 100 == 0`, `AwkAAAAAAAACCQAAagAAAAIAAAAAAAAAJxAAAAAAAAAAAGQAAAAAAAAAAAAmFt9K`, env, true, false},
		{`FRACTION`, `fraction(10, 5, 2)>0`, `AQkAAGYAAAACCQAAawAAAAMAAAAAAAAAAAoAAAAAAAAAAAUAAAAAAAAAAAIAAAAAAAAAAACRyFu2`, env, true, false},
		{`POW`, `pow(12, 1, 3456, 3, 2, Down()) == 187`, `AwkAAAAAAAACCQAAbAAAAAYAAAAAAAAAAAwAAAAAAAAAAAEAAAAAAAAADYAAAAAAAAAAAAMAAAAAAAAAAAIJAQAAAAREb3duAAAAAAAAAAAAAAAAu9llw2M=`, env, true, false},
		{`POW`, `pow(12, 1, 3456, 3, 2, UP) == 188`, `AwkAAAAAAAACCQAAbAAAAAYAAAAAAAAAAAwAAAAAAAAAAAEAAAAAAAAADYAAAAAAAAAAAAMAAAAAAAAAAAIFAAAAAlVQAAAAAAAAAAC8evjDQQ==`, env, true, false},
		{`POW`, `pow(12, 1, 3456, 3, 2, UP) == 187`, `AwkAAAAAAAACCQAAbAAAAAYAAAAAAAAAAAwAAAAAAAAAAAEAAAAAAAAADYAAAAAAAAAAAAMAAAAAAAAAAAIFAAAAAlVQAAAAAAAAAAC7FUMwCQ==`, env, false, false},
		{`LOG`, `log(16, 0, 2, 0, 0, CEILING) == 4`, `AwkAAAAAAAACCQAAbQAAAAYAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAAAFAAAAB0NFSUxJTkcAAAAAAAAAAARh6Dy6`, env, true, false},
		{`LOG`, `log(100, 0, 10, 0, 0, CEILING) == 2`, `AwkAAAAAAAACCQAAbQAAAAYAAAAAAAAAAGQAAAAAAAAAAAAAAAAAAAAAAAoAAAAAAAAAAAAAAAAAAAAAAAAFAAAAB0NFSUxJTkcAAAAAAAAAAAJ7Op42`, env, true, false},

		{`SIZE_BYTES`, `size(base58'abcd') > 0`, `AQkAAGYAAAACCQAAyAAAAAEBAAAAA2QGAgAAAAAAAAAAACMcdM4=`, env, true, false},
		{`TAKE_BYTES`, `size(take(base58'abcd', 2)) == 2`, `AQkAAAAAAAACCQAAyAAAAAEJAADJAAAAAgEAAAADZAYCAAAAAAAAAAACAAAAAAAAAAACccrCZg==`, env, true, false},
		{`DROP_BYTES`, `size(drop(base58'abcd', 2)) > 0`, `AQkAAGYAAAACCQAAyAAAAAEJAADKAAAAAgEAAAADZAYCAAAAAAAAAAACAAAAAAAAAAAA+srbUQ==`, env, true, false},
		{`DROP_BYTES`, `let data = base64'AAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLw=='; func getStock(data:ByteVector) = toInt(take(drop(data, 8), 8)); getStock(data) == 1`, `AwQAAAAEZGF0YQEAAABwAAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLwoBAAAACGdldFN0b2NrAAAAAQAAAARkYXRhCQAEsQAAAAEJAADJAAAAAgkAAMoAAAACBQAAAARkYXRhAAAAAAAAAAAIAAAAAAAAAAAICQAAAAAAAAIJAQAAAAhnZXRTdG9jawAAAAEFAAAABGRhdGEAAAAAAAAAAAFCtabi`, env, true, false},
		{`SUM_BYTES`, `size(base58'ab' + base58'cd') > 0`, `AQkAAGYAAAACCQAAyAAAAAEJAADLAAAAAgEAAAACB5wBAAAAAggSAAAAAAAAAAAAo+LRIA==`, env, true, false},

		{`SUM_STRING`, `"ab"+"cd" == "abcd"`, `AQkAAAAAAAACCQABLAAAAAICAAAAAmFiAgAAAAJjZAIAAAAEYWJjZMBJvls=`, env, true, false},
		{`TAKE_STRING`, `take("abcd", 2) == "ab"`, `AQkAAAAAAAACCQABLwAAAAICAAAABGFiY2QAAAAAAAAAAAICAAAAAmFiiXc+oQ==`, env, true, false},
		{`TAKE_STRING`, `take("", 1) == ""`, `AwkAAAAAAAACCQABLwAAAAICAAAAAAAAAAAAAAAAAQIAAAAAmz5yjQ==`, env, true, false},
		{`DROP_STRING`, `drop("abcd", 2) == "cd"`, `AQkAAAAAAAACCQABMAAAAAICAAAABGFiY2QAAAAAAAAAAAICAAAAAmNkZQdjWQ==`, env, true, false},
		{`SIZE_STRING`, `size("abcd") == 4`, `AQkAAAAAAAACCQABMQAAAAECAAAABGFiY2QAAAAAAAAAAAScZzsq`, env, true, false},

		{`SIZE_LIST`, `size(tx.proofs) == 8`, `AwkAAAAAAAACCQABkAAAAAEIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAgEd23x`, env, true, false},
		{`GET_LIST`, `size(tx.proofs[0]) > 0`, `AQkAAGYAAAACCQAAyAAAAAEJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAAAAAAAAAAAAAFF6iVo=`, env, true, false},
		{`LONG_TO_BYTES`, `toBytes(1) == base58'11111112'`, `AQkAAAAAAAACCQABmgAAAAEAAAAAAAAAAAEBAAAACAAAAAAAAAABm8cc1g==`, env, true, false},
		{`STRING_TO_BYTES`, `toBytes("привет") == base58'4wUjatAwfVDjaHQVX'`, `AQkAAAAAAAACCQABmwAAAAECAAAADNC/0YDQuNCy0LXRggEAAAAM0L/RgNC40LLQtdGCuUGFxw==`, env, true, false},
		{`BOOLEAN_TO_BYTES`, `toBytes(true) == base58'2'`, `AQkAAAAAAAACCQABnAAAAAEGAQAAAAEBJRrQbw==`, env, true, false},
		{`LONG_TO_STRING`, `toString(5) == "5"`, `AQkAAAAAAAACCQABpAAAAAEAAAAAAAAAAAUCAAAAATXPb5tR`, env, true, false},
		{`BOOLEAN_TO_STRING`, `toString(true) == "true"`, `AQkAAAAAAAACCQABpQAAAAEGAgAAAAR0cnVlL6ZrWg==`, env, true, false},

		{`SIGVERIFY`, `sigVerify(tx.bodyBytes, tx.proofs[0], base58'14ovLL9a6xbBfftyxGNLKMdbnzGgnaFQjmgUJGdho6nY')`, `AQkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAABAAAAIAD5y2Wf7zxfv7l+9tcWxyLAbktd9nCbdvFMnxmREqV1igWi3A==`, env, true, false},
		{`KECCAK256`, `keccak256(base58'a') != base58'a'`, `AQkBAAAAAiE9AAAAAgkAAfUAAAABAQAAAAEhAQAAAAEhKeR77g==`, env, true, false},
		{`BLAKE256`, `blake2b256(base58'a') != base58'a'`, `AQkBAAAAAiE9AAAAAgkAAfYAAAABAQAAAAEhAQAAAAEh50D2WA==`, env, true, false},
		{`SHA256`, `sha256(base58'a') != base58'a'`, `AQkBAAAAAiE9AAAAAgkAAfcAAAABAQAAAAEhAQAAAAEhVojmeg==`, env, true, false},
		{`RSAVERIFY`, `let pk = fromBase64String("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAkDg8m0bCDX7fTbBlHZm+BZIHVOfC2I4klRbjSqwFi/eCdfhGjYRYvu/frpSO0LIm0beKOUvwat6DY4dEhNt2PW3UeQvT2udRQ9VBcpwaJlLreCr837sn4fa9UG9FQFaGofSww1O9eBBjwMXeZr1jOzR9RBIwoL1TQkIkZGaDXRltEaMxtNnzotPfF3vGIZZuZX4CjiitHaSC0zlmQrEL3BDqqoLwo3jq8U3Zz8XUMyQElwufGRbZqdFCeiIs/EoHiJm8q8CVExRoxB0H/vE2uDFK/OXLGTgfwnDlrCa/qGt9Zsb8raUSz9IIHx72XB+kOXTt/GOuW7x2dJvTJIqKTwIDAQAB"); let msg = fromBase64String("REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU="); let sig = fromBase64String("OXVKJwtSoenRmwizPtpjh3sCNmOpU1tnXUnyzl+PEI1P9Rx20GkxkIXlysFT2WdbPn/HsfGMwGJW7YhrVkDXy4uAQxUxSgQouvfZoqGSPp1NtM8iVJOGyKiepgB3GxRzQsev2G8Ik47eNkEDVQa47ct9j198Wvnkf88yjSkK0KxR057MWAi20ipNLirW4ZHDAf1giv68mniKfKxsPWahOA/7JYkv18sxcsISQqRXM8nGI1UuSLt9ER7kIzyAk2mgPCiVlj0hoPGUytmbiUqvEM4QaJfCpR0wVO4f/fob6jwKkGT6wbtia+5xCD7bESIHH8ISDrdexZ01QyNP2r4enw=="); rsaVerify(SHA3256, msg, sig, pk)`, `AwQAAAACcGsJAAJbAAAAAQIAAAGITUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUFrRGc4bTBiQ0RYN2ZUYkJsSFptK0JaSUhWT2ZDMkk0a2xSYmpTcXdGaS9lQ2RmaEdqWVJZdnUvZnJwU08wTEltMGJlS09VdndhdDZEWTRkRWhOdDJQVzNVZVF2VDJ1ZFJROVZCY3B3YUpsTHJlQ3I4MzdzbjRmYTlVRzlGUUZhR29mU3d3MU85ZUJCandNWGVacjFqT3pSOVJCSXdvTDFUUWtJa1pHYURYUmx0RWFNeHRObnpvdFBmRjN2R0laWnVaWDRDamlpdEhhU0MwemxtUXJFTDNCRHFxb0x3bzNqcThVM1p6OFhVTXlRRWx3dWZHUmJacWRGQ2VpSXMvRW9IaUptOHE4Q1ZFeFJveEIwSC92RTJ1REZLL09YTEdUZ2Z3bkRsckNhL3FHdDlac2I4cmFVU3o5SUlIeDcyWEIra09YVHQvR091Vzd4MmRKdlRKSXFLVHdJREFRQUIEAAAAA21zZwkAAlsAAAABAgAAAFBSRUlpTjJoRFFVeElKVlF6ZGsxelFTcFhjbFJSZWxFeFZXZCtZR1FvT3l4MEtIZHVQekZtY1U4elVXb3NXaUE3YUZsb09XcGxjbEF4UENVPQQAAAADc2lnCQACWwAAAAECAAABWE9YVktKd3RTb2VuUm13aXpQdHBqaDNzQ05tT3BVMXRuWFVueXpsK1BFSTFQOVJ4MjBHa3hrSVhseXNGVDJXZGJQbi9Ic2ZHTXdHSlc3WWhyVmtEWHk0dUFReFV4U2dRb3V2ZlpvcUdTUHAxTnRNOGlWSk9HeUtpZXBnQjNHeFJ6UXNldjJHOElrNDdlTmtFRFZRYTQ3Y3Q5ajE5OFd2bmtmODh5alNrSzBLeFIwNTdNV0FpMjBpcE5MaXJXNFpIREFmMWdpdjY4bW5pS2ZLeHNQV2FoT0EvN0pZa3YxOHN4Y3NJU1FxUlhNOG5HSTFVdVNMdDlFUjdrSXp5QWsybWdQQ2lWbGowaG9QR1V5dG1iaVVxdkVNNFFhSmZDcFIwd1ZPNGYvZm9iNmp3S2tHVDZ3YnRpYSs1eENEN2JFU0lISDhJU0RyZGV4WjAxUXlOUDJyNGVudz09CQAB+AAAAAQFAAAAB1NIQTMyNTYFAAAAA21zZwUAAAADc2lnBQAAAAJwa8wcz28=`, env, true, false},

		{`TOBASE58`, `toBase58String(base58'a') == "a"`, `AQkAAAAAAAACCQACWAAAAAEBAAAAASECAAAAAWFcT4nY`, env, true, false},
		{`FROMBASE58`, `fromBase58String("a") == base58'a'`, `AQkAAAAAAAACCQACWQAAAAECAAAAAWEBAAAAASEB1Qmd`, env, true, false},
		{`FROMBASE58`, `fromBase58String(extract("")) == base58''`, `AwkAAAAAAAACCQACWQAAAAEJAQAAAAdleHRyYWN0AAAAAQIAAAAAAQAAAAAt2xTN`, env, true, false},
		{`TOBASE64`, `toBase64String(base16'544553547465737454455354') == "VEVTVHRlc3RURVNU"`, `AwkAAAAAAAACCQACWgAAAAEBAAAADFRFU1R0ZXN0VEVTVAIAAAAQVkVWVFZIUmxjM1JVUlZOVd6DVfc=`, env, true, false},
		{`FROMBASE64`, `base16'544553547465737454455354' == fromBase64String("VEVTVHRlc3RURVNU")`, `AwkAAAAAAAACAQAAAAxURVNUdGVzdFRFU1QJAAJbAAAAAQIAAAAQVkVWVFZIUmxjM1JVUlZOVV+c29Q=`, env, true, false},
		{`TOBASE16`, `toBase16String(base64'VEVTVHRlc3RURVNU') == "544553547465737454455354"`, `AwkAAAAAAAACCQACXAAAAAEBAAAADFRFU1R0ZXN0VEVTVAIAAAAYNTQ0NTUzNTQ3NDY1NzM3NDU0NDU1MzU07NMrMQ==`, env, true, false},
		{`FROMBASE16`, `fromBase16String("544553547465737454455354") == base64'VEVTVHRlc3RURVNU'`, `AwkAAAAAAAACCQACXQAAAAECAAAAGDU0NDU1MzU0NzQ2NTczNzQ1NDQ1NTM1NAEAAAAMVEVTVHRlc3RURVNUFBEa5A==`, env, true, false},

		{`CHECKMERKLEPROOF`, `let rootHash = base64'eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk='; let leafData = base64'AAAm+w=='; let merkleProof = base64'ACBSs2di6rY+9N3mrpQVRNZLGAdRX2WBD6XkrOXuhh42XwEgKhB3Aiij6jqLRuQhrwqv6e05kr89tyxkuFYwUuMCQB8AIKLhp/AFQkokTe/NMQnKFL5eTMvDlFejApmJxPY6Rp8XACAWrdgB8DwvPA8D04E9HgUjhKghAn5aqtZnuKcmpLHztQAgd2OG15WYz90r1WipgXwjdq9WhvMIAtvGlm6E3WYY12oAIJXPPVIdbwOTdUJvCgMI4iape2gvR55vsrO2OmJJtZUNASAya23YyBl+EpKytL9+7cPdkeMMWSjk0Bc0GNnqIisofQ=='; checkMerkleProof(rootHash, merkleProof, leafData)`, `AwQAAAAIcm9vdEhhc2gBAAAAIHofX5tx3h2d1wP1HzKQvR0sC1TMgS4JACS9DCFQSKGJBAAAAAhsZWFmRGF0YQEAAAAEAAAm+wQAAAALbWVya2xlUHJvb2YBAAAA7gAgUrNnYuq2PvTd5q6UFUTWSxgHUV9lgQ+l5Kzl7oYeNl8BICoQdwIoo+o6i0bkIa8Kr+ntOZK/PbcsZLhWMFLjAkAfACCi4afwBUJKJE3vzTEJyhS+XkzLw5RXowKZicT2OkafFwAgFq3YAfA8LzwPA9OBPR4FI4SoIQJ+WqrWZ7inJqSx87UAIHdjhteVmM/dK9VoqYF8I3avVobzCALbxpZuhN1mGNdqACCVzz1SHW8Dk3VCbwoDCOImqXtoL0eeb7KztjpiSbWVDQEgMmtt2MgZfhKSsrS/fu3D3ZHjDFko5NAXNBjZ6iIrKH0JAAK8AAAAAwUAAAAIcm9vdEhhc2gFAAAAC21lcmtsZVByb29mBQAAAAhsZWFmRGF0YXe8Icg=`, env, true, false},

		{`GETTRANSACTIONBYID`, `V2: match transactionById(tx.id) {case _: TransferTransaction => true; case _ => false}`, `AgQAAAAHJG1hdGNoMAkAA+gAAAABCAUAAAACdHgAAAACaWQDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24GB9Sc8FA=`, env, true, false},
		{`TRANSACTIONHEIGHTBYID`, `transactionHeightById(base58'aaaa') == 5`, `AQkAAAAAAAACCQAD6QAAAAEBAAAAA2P4ZwAAAAAAAAAABSLhRM4=`, env, false, false},
		{`ACCOUNTASSETBALANCE`, `assetBalance(tx.sender, base58'BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD') == 5`, `AQkAAAAAAAACCQAD6wAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIBAAAAIJxQIls8iGUc1935JolBz6bYc37eoPDtScOAM0lTNhY0AAAAAAAAAAAFjp6PBg==`, env, true, false},
		{`ADDRESSTOSTRING`, `toString(Address(base58'3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpg')) == "3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpg"`, `AwkAAAAAAAACCQAEJQAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVcMIZxOsk2Gw5Avd0ztqi+phtb1Bb83MiUCAAAAIzNQMzMzNnJOU1NVOGJEQXFEYjZTNWpOczhESmIyYmZObXBnkXj7Cg==`, env, true, false},
		{`ADDRESSTOSTRING`, `toString(Address(base58'3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpg')) == "3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpf"`, `AwkAAAAAAAACCQAEJQAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVcMIZxOsk2Gw5Avd0ztqi+phtb1Bb83MiUCAAAAIzNQMzMzNnJOU1NVOGJEQXFEYjZTNWpOczhESmIyYmZObXBmb/6mcg==`, env, false, false},
		{`CONS`, `size([1, "2"]) == 2`, `AwkAAAAAAAACCQABkAAAAAEJAARMAAAAAgAAAAAAAAAAAQkABEwAAAACAgAAAAEyBQAAAANuaWwAAAAAAAAAAAKuUcc0`, env, true, false},
		{`CONS`, `size(cons(1, nil)) == 1`, `AwkAAAAAAAACCQABkAAAAAEJAARMAAAAAgAAAAAAAAAAAQUAAAADbmlsAAAAAAAAAAABX96esw==`, env, true, false},
		{`CONS`, `[1, 2, 3, 4, 5][4] == 5`, `AwkAAAAAAAACCQABkQAAAAIJAARMAAAAAgAAAAAAAAAAAQkABEwAAAACAAAAAAAAAAACCQAETAAAAAIAAAAAAAAAAAMJAARMAAAAAgAAAAAAAAAABAkABEwAAAACAAAAAAAAAAAFBQAAAANuaWwAAAAAAAAAAAQAAAAAAAAAAAVrPjYC`, env, true, false},
		{`CONS`, `[1, 2, 3, 4, 5][4] == 4`, `AwkAAAAAAAACCQABkQAAAAIJAARMAAAAAgAAAAAAAAAAAQkABEwAAAACAAAAAAAAAAACCQAETAAAAAIAAAAAAAAAAAMJAARMAAAAAgAAAAAAAAAABAkABEwAAAACAAAAAAAAAAAFBQAAAANuaWwAAAAAAAAAAAQAAAAAAAAAAASbi8eN`, env, false, false},
		{`UTF8STR`, `toUtf8String(base16'536f6d65207465737420737472696e67') == "Some test string"`, `AwkAAAAAAAACCQAEsAAAAAEBAAAAEFNvbWUgdGVzdCBzdHJpbmcCAAAAEFNvbWUgdGVzdCBzdHJpbme0Wj5y`, env, true, false},
		{`UTF8STR`, `toUtf8String(base16'536f6d65207465737420737472696e67') == "blah-blah-blah"`, `AwkAAAAAAAACCQAEsAAAAAEBAAAAEFNvbWUgdGVzdCBzdHJpbmcCAAAADmJsYWgtYmxhaC1ibGFojpjG3g==`, env, false, false},
		{`TOINT`, `toInt(base16'0000000000003039') == 12345`, `AwkAAAAAAAACCQAEsQAAAAEBAAAACAAAAAAAADA5AAAAAAAAADA5WVzTeQ==`, env, true, false},
		{`TOINT`, `toInt(base16'3930000000000000') == 12345`, `AwkAAAAAAAACCQAEsQAAAAEBAAAACDkwAAAAAAAAAAAAAAAAADA5Vq02Hg==`, env, false, false},
		{`TOINT_OFF`, `toInt(base16'ffffff0000000000003039', 3) == 12345`, `AwkAAAAAAAACCQAEsgAAAAIBAAAAC////wAAAAAAADA5AAAAAAAAAAADAAAAAAAAADA5pGJt2g==`, env, true, false},
		{`TOINT_OFF`, `toInt(base16'ffffff0000000000003039', 2) == 12345`, `AwkAAAAAAAACCQAEsgAAAAIBAAAAC////wAAAAAAADA5AAAAAAAAAAACAAAAAAAAADA57UQA4Q==`, env, false, false},
		{`INDEXOF`, `indexOf("cafe bebe dead beef cafe bebe", "bebe") == 5`, `AwkAAAAAAAACCQAEswAAAAICAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAFyqpjwQ==`, env, true, false},
		{`INDEXOF`, `indexOf("cafe bebe dead beef cafe bebe", "fox") == unit`, `AwkAAAAAAAACCQAEswAAAAICAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAANmb3gFAAAABHVuaXS7twzl`, env, true, false},
		{`INDEXOF`, `indexOf("世界}}世界", "}}") == 2`, `AwkAAAAAAAACCQAEswAAAAICAAAADuS4lueVjH195LiW55WMAgAAAAJ9fQAAAAAAAAAAAjCgf3g=`, env, true, false},
		{`INDEXOFN`, `indexOf("cafe bebe dead beef cafe bebe", "bebe", 0) == 5`, `AwkAAAAAAAACCQAEtAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAAAAAAAAAAAAAFFBPTAA==`, env, true, false},
		{`INDEXOFN`, `indexOf("cafe bebe dead beef cafe bebe", "bebe", 10) == 25`, `AwkAAAAAAAACCQAEtAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAKAAAAAAAAAAAZVBpWMw==`, env, true, false},
		{`INDEXOFN`, `indexOf("cafe bebe dead beef cafe bebe", "dead", 10) == 10`, `AwkAAAAAAAACCQAEtAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARkZWFkAAAAAAAAAAAKAAAAAAAAAAAKstuWEQ==`, env, true, false},
		{`INDEXOFN`, `indexOf("cafe bebe dead beef cafe bebe", "dead", 11) == unit`, `AwkAAAAAAAACCQAEtAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARkZWFkAAAAAAAAAAALBQAAAAR1bml0f2q2UQ==`, env, true, false},
		{`SPLIT`, `split("abcd", "") == ["a", "b", "c", "d"]`, `AwkAAAAAAAACCQAEtQAAAAICAAAABGFiY2QCAAAAAAkABEwAAAACAgAAAAFhCQAETAAAAAICAAAAAWIJAARMAAAAAgIAAAABYwkABEwAAAACAgAAAAFkBQAAAANuaWwrnSMu`, env, true, false},
		{`SPLIT`, `split("one two three", " ") == ["one", "two", "three"]`, `AwkAAAAAAAACCQAEtQAAAAICAAAADW9uZSB0d28gdGhyZWUCAAAAASAJAARMAAAAAgIAAAADb25lCQAETAAAAAICAAAAA3R3bwkABEwAAAACAgAAAAV0aHJlZQUAAAADbmlsdBcUog==`, env, true, false},
		{`PARSEINT`, `parseInt("12345") == 12345`, `AwkAAAAAAAACCQAEtgAAAAECAAAABTEyMzQ1AAAAAAAAADA57cmovA==`, env, true, false},
		{`PARSEINT`, `parseInt("0x12345") == unit`, `AwkAAAAAAAACCQAEtgAAAAECAAAABzB4MTIzNDUFAAAABHVuaXQvncQM`, env, true, false},
		{`LASTINDEXOF`, `lastIndexOf("cafe bebe dead beef cafe bebe", "bebe") == 25`, `AwkAAAAAAAACCQAEtwAAAAICAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAZDUvNng==`, env, true, false},
		{`LASTINDEXOF`, `lastIndexOf("cafe bebe dead beef cafe bebe", "fox") == unit`, `AwkAAAAAAAACCQAEtwAAAAICAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAANmb3gFAAAABHVuaXSK8YYp`, env, true, false},
		{`LASTINDEXOFN`, `lastIndexOf("cafe bebe dead beef cafe bebe", "bebe", 30) == 25`, `AwkAAAAAAAACCQAEuAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAeAAAAAAAAAAAZus4/9A==`, env, true, false},
		{`LASTINDEXOFN`, `lastIndexOf("cafe bebe dead beef cafe bebe", "bebe", 10) == 5`, `AwkAAAAAAAACCQAEuAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAKAAAAAAAAAAAFrGUCxA==`, env, true, false},
		{`LASTINDEXOFN`, `lastIndexOf("cafe bebe dead beef cafe bebe", "dead", 13) == 10`, `AwkAAAAAAAACCQAEuAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARkZWFkAAAAAAAAAAANAAAAAAAAAAAKepNV2A==`, env, true, false},
		{`LASTINDEXOFN`, `lastIndexOf("cafe bebe dead beef cafe bebe", "dead", 11) == 10`, `AwkAAAAAAAACCQAEuAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARkZWFkAAAAAAAAAAALAAAAAAAAAAAKcxKwfA==`, env, true, false},
		{`CONTAINS on incorrect UTF-8 character`, `"x冬x".contains("\ud87e")`, `BAkBAAAACGNvbnRhaW5zAAAAAgIAAAAGePCvoJp4AgAAAAE/5/PEZA==`, env, false, false},
		{`SPLIT on incorrect UTF-8 character`, `"冬x🤦冬".split("\ud87e") == ["冬x🤦冬"]`, `BAkAAAAAAAACCQAEtQAAAAICAAAADfCvoJp48J+kpvCvoJoCAAAAAT8JAARMAAAAAgIAAAAN8K+gmnjwn6Sm8K+gmgUAAAADbmlsLyxljg==`, env, true, false},
	} {
		src, err := base64.StdEncoding.DecodeString(test.script)
		require.NoError(t, err, test.name)

		tree, err := Parse(src)
		require.NoError(t, err, test.name)
		assert.NotNil(t, tree, test.name)

		res, err := CallVerifier(test.env, tree)
		if test.error {
			assert.Error(t, err, "No error in "+test.name)
		} else {
			require.NoError(t, err, "Unexpected error in: "+test.name)
			assert.NotNil(t, res, test.name)
			r, ok := res.(ScriptResult)
			assert.True(t, ok, test.name)
			assert.Equal(t, test.result, r.Result(), test.name)
		}
	}
}

func TestComplexity(t *testing.T) {
	env := &mockRideEnvironment{
		transactionFunc: testTransferObject,
		stateFunc: func() types.SmartState {
			return &MockSmartState{NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
				return testTransferWithProofs(), nil
			}}
		},
		schemeFunc: func() byte {
			return 'T'
		},
		rideV6ActivatedFunc: noRideV6,
	}
	for _, test := range []struct {
		comment    string
		source     string
		complexity int
	}{
		{`V4: let a = 1 + 10 + 100; let b = 1000 + a + 10000; let c = a + b + 100000; c + a == 111333`, "BAQAAAABYQkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAoAAAAAAAAAAGQEAAAAAWIJAABkAAAAAgkAAGQAAAACAAAAAAAAAAPoBQAAAAFhAAAAAAAAACcQBAAAAAFjCQAAZAAAAAIJAABkAAAAAgUAAAABYQUAAAABYgAAAAAAAAGGoAkAAAAAAAACCQAAZAAAAAIFAAAAAWMFAAAAAWEAAAAAAAABsuVAqr8m", 13},
		{`V4: func f(a: Int, b: Int) = {let c = a + b; let d = a - b; c * d - 1}; f(1, 2) == -4`, "BAoBAAAAAWYAAAACAAAAAWEAAAABYgQAAAABYwkAAGQAAAACBQAAAAFhBQAAAAFiBAAAAAFkCQAAZQAAAAIFAAAAAWEFAAAAAWIJAABlAAAAAgkAAGgAAAACBQAAAAFjBQAAAAFkAAAAAAAAAAABCQAAAAAAAAIJAQAAAAFmAAAAAgAAAAAAAAAAAQAAAAAAAAAAAgD//////////Pwcs2o=", 11},
		{`V4:  let x = 1 + 1 + 1; let a = 1 + 1; func f(a: Int, b: Int) = a - b + x; let b = 4; func g(a: Int, b: Int) = a * b; let expected = (a - b + x) * (b - a + x); let actual = g(f(a, b), f(b, a)); actual == expected && actual == expected && x == 3 && a == 2 && b == 4`, "BAQAAAABeAkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEEAAAAAWEJAABkAAAAAgAAAAAAAAAAAQAAAAAAAAAAAQoBAAAAAWYAAAACAAAAAWEAAAABYgkAAGQAAAACCQAAZQAAAAIFAAAAAWEFAAAAAWIFAAAAAXgEAAAAAWIAAAAAAAAAAAQKAQAAAAFnAAAAAgAAAAFhAAAAAWIJAABoAAAAAgUAAAABYQUAAAABYgQAAAAIZXhwZWN0ZWQJAABoAAAAAgkAAGQAAAACCQAAZQAAAAIFAAAAAWEFAAAAAWIFAAAAAXgJAABkAAAAAgkAAGUAAAACBQAAAAFiBQAAAAFhBQAAAAF4BAAAAAZhY3R1YWwJAQAAAAFnAAAAAgkBAAAAAWYAAAACBQAAAAFhBQAAAAFiCQEAAAABZgAAAAIFAAAAAWIFAAAAAWEDAwMDCQAAAAAAAAIFAAAABmFjdHVhbAUAAAAIZXhwZWN0ZWQJAAAAAAAAAgUAAAAGYWN0dWFsBQAAAAhleHBlY3RlZAcJAAAAAAAAAgUAAAABeAAAAAAAAAAAAwcJAAAAAAAAAgUAAAABYQAAAAAAAAAAAgcJAAAAAAAAAgUAAAABYgAAAAAAAAAABAd/cU2j", 47},
		{`V4:  let x = 1 + 1 + 1 + 1 + 1; let y = x + 1; func f(x: Int) = x + 1; func g(x: Int) = x + 1 + 1; func h(x: Int) = x + 1 + 1 + 1; f(g(h(y))) == x + x + 2`, "BAQAAAABeAkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABBAAAAAF5CQAAZAAAAAIFAAAAAXgAAAAAAAAAAAEKAQAAAAFmAAAAAQAAAAF4CQAAZAAAAAIFAAAAAXgAAAAAAAAAAAEKAQAAAAFnAAAAAQAAAAF4CQAAZAAAAAIJAABkAAAAAgUAAAABeAAAAAAAAAAAAQAAAAAAAAAAAQoBAAAAAWgAAAABAAAAAXgJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIFAAAAAXgAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEJAAAAAAAAAgkBAAAAAWYAAAABCQEAAAABZwAAAAEJAQAAAAFoAAAAAQUAAAABeQkAAGQAAAACCQAAZAAAAAIFAAAAAXgFAAAAAXgAAAAAAAAAAAJBsCoy", 21},
		{`V4:  let x = 1 + 1 + 1; let y = {let z = 1; z}; y + x == 4`, "BAQAAAABeAkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEEAAAAAXkEAAAAAXoAAAAAAAAAAAEFAAAAAXoJAAAAAAAAAgkAAGQAAAACBQAAAAF5BQAAAAF4AAAAAAAAAAAECwZAYA==", 7},
		{`V4:  let address = Address(base58'aaaa'); address.bytes == base58'aaaa'`, "BAQAAAAHYWRkcmVzcwkBAAAAB0FkZHJlc3MAAAABAQAAAANj+GcJAAAAAAAAAggFAAAAB2FkZHJlc3MAAAAFYnl0ZXMBAAAAA2P4Z/7QEyM=", 3},
		{`V4:  let x = (1, 2, 3); x._2 == 2`, "BAQAAAABeAkABRUAAAADAAAAAAAAAAABAAAAAAAAAAACAAAAAAAAAAADCQAAAAAAAAIIBQAAAAF4AAAAAl8yAAAAAAAAAAACXAdyJg==", 4},
		{`V4:  let a = 1 + 1; let b = a; func g() = {let a1 = 2 + 2 + 2; let c = a1; c + b + a1}; g() + a == 16`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZwAAAAAEAAAAAmExCQAAZAAAAAIJAABkAAAAAgAAAAAAAAAAAgAAAAAAAAAAAgAAAAAAAAAAAgQAAAABYwUAAAACYTEJAABkAAAAAgkAAGQAAAACBQAAAAFjBQAAAAFiBQAAAAJhMQkAAAAAAAACCQAAZAAAAAIJAQAAAAFnAAAAAAUAAAABYQAAAAAAAAAAEGbAh7s=", 13},
		{`V4:  let a = 1 + 1; let b = a; func g() = {let c = 2 + 2; let d = c; d + b + c}; g() + a == 12`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZwAAAAAEAAAAAWMJAABkAAAAAgAAAAAAAAAAAgAAAAAAAAAAAgQAAAABZAUAAAABYwkAAGQAAAACCQAAZAAAAAIFAAAAAWQFAAAAAWIFAAAAAWMJAAAAAAAAAgkAAGQAAAACCQEAAAABZwAAAAAFAAAAAWEAAAAAAAAAAAxjZ1Td", 12},
		{`V4:  let a = 1 + 1; let b = a; func f() = {let c = 1 + 1; c + b}; a + f() == 6`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZgAAAAAEAAAAAWMJAABkAAAAAgAAAAAAAAAAAQAAAAAAAAAAAQkAAGQAAAACBQAAAAFjBQAAAAFiCQAAAAAAAAIJAABkAAAAAgUAAAABYQkBAAAAAWYAAAAAAAAAAAAAAAAGZR1Q1A==", 9},
		{`V4:  let a = 1 + 1; let b = a; func f() = {let c = 1 + 1; c + b}; f() + a == 6`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZgAAAAAEAAAAAWMJAABkAAAAAgAAAAAAAAAAAQAAAAAAAAAAAQkAAGQAAAACBQAAAAFjBQAAAAFiCQAAAAAAAAIJAABkAAAAAgkBAAAAAWYAAAAABQAAAAFhAAAAAAAAAAAGeznbzA==", 9},
		{`V4: let a = 1 + 1; let b = a; a + b == 4`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCQAAAAAAAAIJAABkAAAAAgUAAAABYQUAAAABYgAAAAAAAAAABClbyII=", 6},
		{`V4: let a = 1 + 1; let b = a; b + a == 4`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCQAAAAAAAAIJAABkAAAAAgUAAAABYgUAAAABYQAAAAAAAAAABApVv5E=", 6},
		{`V4: let a = 1 + 1; let b = a; func f() = b; a + f() == 4`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZgAAAAAFAAAAAWIJAAAAAAAAAgkAAGQAAAACBQAAAAFhCQEAAAABZgAAAAAAAAAAAAAAAASZ9mVe", 6},
		{`V4: let a = 1 + 1; let b = a; func f() = b; f() + a == 4`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZgAAAAAFAAAAAWIJAAAAAAAAAgkAAGQAAAACCQEAAAABZgAAAAAFAAAAAWEAAAAAAAAAAASvoK6u", 6},
	} {
		src, err := base64.StdEncoding.DecodeString(test.source)
		require.NoError(t, err, test.comment)

		tree, err := Parse(src)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, tree, test.comment)

		res, err := CallVerifier(env, tree)
		require.NoError(t, err, test.comment)
		require.NotNil(t, res, test.comment)

		r, ok := res.(ScriptResult)
		assert.True(t, ok, test.comment)
		assert.Equal(t, test.complexity, r.Complexity(), test.comment)
	}
}

func TestOverlapping(t *testing.T) {
	/*{-# STDLIB_VERSION 3 #-}
	  {-# CONTENT_TYPE EXPRESSION #-}
	  {-# SCRIPT_TYPE ACCOUNT #-}
	  let ref = 999
	  func g(a: Int) = ref
	  func f(ref: Int) = g(ref)
	  f(1) == 999
	*/
	s := "AwQAAAADcmVmAAAAAAAAAAPnCgEAAAABZwAAAAEAAAABYQUAAAADcmVmCgEAAAABZgAAAAEAAAADcmVmCQEAAAABZwAAAAEFAAAAA3JlZgkAAAAAAAACCQEAAAABZgAAAAEAAAAAAAAAAAEAAAAAAAAAA+fjknmW"
	src, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallVerifier(te, tree)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	assert.True(t, r.Result())
}

func TestInvokeExpression(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE CALL #-}

		let lease = Lease(Address(base58'3FMdfKQ3yrkrGawp4QYkf8phE6ZMup7hfR2'), 10)
		[lease, BooleanEntry("key", true]
	*/
	s := "BgEEBWxlYXNlCQDECAIJAQdBZGRyZXNzAQEaAUP2ZeK0oJWLGYVbOVovHApDYXsAHYcycskACgkAzAgCBQVsZWFzZQkAzAgCCQEMQm9vbGVhbkVudHJ5AgIDa2V5BgUDbmlss7c8Wg=="
	src, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	env, _ := testInvokeEnv(true)
	res, err := CallVerifier(env, tree)
	require.NoError(t, err)
	require.NotNil(t, res.ScriptActions())
	r, ok := res.(DAppResult)
	require.True(t, ok)
	assert.True(t, r.Result())
}

func TestUserFunctionsInExpression(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 3 #-}
	   {-# CONTENT_TYPE EXPRESSION #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}
	   func g() = 5
	   g() == 5
	*/
	s := `AwoBAAAAAWcAAAAAAAAAAAAAAAAFCQAAAAAAAAIJAQAAAAFnAAAAAAAAAAAAAAAABWtYRqw=`
	src, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallVerifier(te, tree)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	assert.True(t, r.Result())
}

func TestDataFunctions(t *testing.T) {
	secret, public, err := crypto.GenerateKeyPair([]byte("test data transaction"))
	require.NoError(t, err)
	data := proto.NewUnsignedData(1, public, 10000, 1544715621)
	require.NoError(t, data.AppendEntry(&proto.IntegerDataEntry{
		Key:   "integer",
		Value: 100500,
	}))
	require.NoError(t, data.AppendEntry(&proto.BooleanDataEntry{
		Key:   "boolean",
		Value: true,
	}))
	require.NoError(t, data.AppendEntry(&proto.BinaryDataEntry{
		Key:   "binary",
		Value: []byte("hello"),
	}))
	require.NoError(t, data.AppendEntry(&proto.StringDataEntry{
		Key:   "string",
		Value: "world",
	}))
	require.NoError(t, data.Sign(proto.MainNetScheme, secret))
	txObj, err := transactionToObject('W', data)
	require.NoError(t, err)
	env := &mockRideEnvironment{
		transactionFunc: func() rideObject {
			return txObj
		},
		heightFunc: func() rideInt {
			return rideInt(100500)
		},
		rideV6ActivatedFunc: noRideV6,
	}
	for _, test := range []struct {
		name   string
		code   string
		base64 string
		result bool
	}{
		{"DATA_LONG_FROM_ARRAY", `match tx {case t: DataTransaction => getInteger(t.data, "integer") == 100500 case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQAEEAAAAAIIBQAAAAF0AAAABGRhdGECAAAAB2ludGVnZXIAAAAAAAABiJQHp2oJqg==`, true},
		{"DATA_BOOLEAN_FROM_ARRAY", `match tx {case t: DataTransaction => getBoolean(t.data, "boolean") == true case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQAEEQAAAAIIBQAAAAF0AAAABGRhdGECAAAAB2Jvb2xlYW4GBw5ToUs=`, true},
		{"DATA_BYTES_FROM_ARRAY", `match tx {case t: DataTransaction => getBinary(t.data, "binary") == base58'Cn8eVZg' case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQAEEgAAAAIIBQAAAAF0AAAABGRhdGECAAAABmJpbmFyeQEAAAAFaGVsbG8HDogmeQ==`, true},
		{"DATA_STRING_FROM_ARRAY", `match tx {case t: DataTransaction => getString(t.data, "string") == "world" case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQAEEwAAAAIIBQAAAAF0AAAABGRhdGECAAAABnN0cmluZwIAAAAFd29ybGQH7+G/UA==`, true},

		{"UserDataIntegerFromArrayByIndex", `match tx {case t : DataTransaction => getInteger(t.data, 0) == 100500 case _ => true}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQEAAAAKZ2V0SW50ZWdlcgAAAAIIBQAAAAF0AAAABGRhdGEAAAAAAAAAAAAAAAAAAAABiJQGwLSDPw==`, true},
		{"UserDataBooleanFromArrayByIndex", `match tx {case t : DataTransaction => getBoolean(t.data, 1) == true case _ => true}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQEAAAAKZ2V0Qm9vbGVhbgAAAAIIBQAAAAF0AAAABGRhdGEAAAAAAAAAAAEGBk7sdw4=`, true},
		{"UserDataBinaryFromArrayByIndex", `match tx {case t : DataTransaction => getBinary(t.data, 2) == base58'Cn8eVZg' case _ => true}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQEAAAAJZ2V0QmluYXJ5AAAAAggFAAAAAXQAAAAEZGF0YQAAAAAAAAAAAgEAAAAFaGVsbG8GRLZgkQ==`, true},
		{"UserDataStringFromArrayByIndex", `match tx {case t : DataTransaction => getString(t.data, 3) == "world" case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQEAAAAJZ2V0U3RyaW5nAAAAAggFAAAAAXQAAAAEZGF0YQAAAAAAAAAAAwIAAAAFd29ybGQHKKHsFw==`, true},
	} {
		src, err := base64.StdEncoding.DecodeString(test.base64)
		require.NoError(t, err, test.name)

		tree, err := Parse(src)
		require.NoError(t, err, test.name)
		assert.NotNil(t, tree, test.name)

		res, err := CallVerifier(env, tree)
		require.NoError(t, err, test.name)
		r, ok := res.(ScriptResult)
		require.True(t, ok, test.name)
		assert.Equal(t, test.result, r.Result(), test.name)
	}
}

func testInvokeEnv(verifier bool) (environment, *proto.InvokeScriptWithProofs) {
	tx := byte_helpers.InvokeScriptWithProofs.Transaction.Clone()
	txo, err := transactionToObject(proto.MainNetScheme, tx)
	if err != nil {
		panic(err)
	}

	env := &mockRideEnvironment{
		invocationFunc: func() rideObject {
			if !verifier {
				obj, err := invocationToObject(3, proto.MainNetScheme, tx)
				if err != nil {
					panic(err)
				}
				return obj
			}
			return txo
		},
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		txIDFunc: func() rideType {
			return rideBytes(tx.ID.Bytes())
		},
		transactionFunc: func() rideObject {
			return txo
		},
		rideV6ActivatedFunc: noRideV6,
	}
	return env, tx
}

func TestDappCallable(t *testing.T) {
	/*{-# STDLIB_VERSION 3 #-}
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

	  @Verifier(tx)
	  func verify() = {
	      getPreviousAnswer(toString(tx.sender)) == "1"
	  }
	*/
	env, _ := testInvokeEnv(false)
	code := "AAIDAAAAAAAAAAAAAAABAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEAAAAHYWRkcmVzcwUAAAAHYWRkcmVzcwAAAAEAAAABaQEAAAAGdGVsbG1lAAAAAQAAAAhxdWVzdGlvbgQAAAAGYW5zd2VyCQEAAAARZ2V0UHJldmlvdXNBbnN3ZXIAAAABBQAAAAhxdWVzdGlvbgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9xBQAAAAhxdWVzdGlvbgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9hBQAAAAZhbnN3ZXIFAAAAA25pbAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAJAAAAAAAAAgkBAAAAEWdldFByZXZpb3VzQW5zd2VyAAAAAQkABCUAAAABCAUAAAACdHgAAAAGc2VuZGVyAgAAAAEx7gicPQ=="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "tellme", proto.Arguments{proto.NewStringArgument("abc")})
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	require.EqualValues(t,
		&proto.ScriptResult{
			DataEntries: []*proto.DataEntryScriptAction{
				{Entry: &proto.StringDataEntry{Key: "abc_q", Value: "abc"}},
				{Entry: &proto.StringDataEntry{Key: "abc_a", Value: "abc"}},
			},
			Transfers:    make([]*proto.TransferScriptAction, 0),
			Issues:       make([]*proto.IssueScriptAction, 0),
			Reissues:     make([]*proto.ReissueScriptAction, 0),
			Burns:        make([]*proto.BurnScriptAction, 0),
			Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
			Leases:       make([]*proto.LeaseScriptAction, 0),
			LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
		},
		sr,
	)
}

func TestDappDefaultFunc(t *testing.T) {
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
	env, tx := testInvokeEnv(false)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, tx.SenderPK)
	require.NoError(t, err)

	code := "AAIDAAAAAAAAAAAAAAABAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEAAAAHYWRkcmVzcwUAAAAHYWRkcmVzcwAAAAIAAAABaQEAAAAGdGVsbG1lAAAAAQAAAAhxdWVzdGlvbgQAAAAGYW5zd2VyCQEAAAARZ2V0UHJldmlvdXNBbnN3ZXIAAAABBQAAAAhxdWVzdGlvbgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9xBQAAAAhxdWVzdGlvbgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9hBQAAAAZhbnN3ZXIFAAAAA25pbAAAAAppbnZvY2F0aW9uAQAAAAdkZWZhdWx0AAAAAAQAAAAHc2VuZGVyMAgIBQAAAAppbnZvY2F0aW9uAAAABmNhbGxlcgAAAAVieXRlcwkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAABYQIAAAABYgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAGc2VuZGVyBQAAAAdzZW5kZXIwBQAAAANuaWwAAAABAAAAAnR4AQAAAAZ2ZXJpZnkAAAAACQAAAAAAAAIJAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEJAAQlAAAAAQgFAAAAAnR4AAAABnNlbmRlcgIAAAABMcP91gY="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "", proto.Arguments{})
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	require.EqualValues(t,
		&proto.ScriptResult{
			DataEntries: []*proto.DataEntryScriptAction{
				{Entry: &proto.StringDataEntry{Key: "a", Value: "b"}},
				{Entry: &proto.BinaryDataEntry{Key: "sender", Value: addr.Bytes()}},
			},
			Transfers:    make([]*proto.TransferScriptAction, 0),
			Issues:       make([]*proto.IssueScriptAction, 0),
			Reissues:     make([]*proto.ReissueScriptAction, 0),
			Burns:        make([]*proto.BurnScriptAction, 0),
			Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
			Leases:       make([]*proto.LeaseScriptAction, 0),
			LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
		},
		sr,
	)
}

func TestDappVerify(t *testing.T) {
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
	env, _ := testInvokeEnv(true)
	code := "AAIDAAAAAAAAAAAAAAABAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEAAAAHYWRkcmVzcwUAAAAHYWRkcmVzcwAAAAIAAAABaQEAAAAGdGVsbG1lAAAAAQAAAAhxdWVzdGlvbgQAAAAGYW5zd2VyCQEAAAARZ2V0UHJldmlvdXNBbnN3ZXIAAAABBQAAAAhxdWVzdGlvbgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9xBQAAAAhxdWVzdGlvbgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9hBQAAAAZhbnN3ZXIFAAAAA25pbAAAAAppbnZvY2F0aW9uAQAAAAdkZWZhdWx0AAAAAAQAAAAHc2VuZGVyMAgIBQAAAAppbnZvY2F0aW9uAAAABmNhbGxlcgAAAAVieXRlcwkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAABYQIAAAABYgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAGc2VuZGVyBQAAAAdzZW5kZXIwBQAAAANuaWwAAAABAAAAAnR4AQAAAAZ2ZXJpZnkAAAAACQAAAAAAAAIJAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEJAAQlAAAAAQgFAAAAAnR4AAAABnNlbmRlcgIAAAABMcP91gY="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallVerifier(env, tree)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	assert.False(t, r.Result())
}

func TestDappVerifySuccessful(t *testing.T) {
	/*{-# STDLIB_VERSION 3 #-}
	  {-# CONTENT_TYPE DAPP #-}
	  {-# SCRIPT_TYPE ACCOUNT #-}

	  let x = 100500

	  func getPreviousAnswer() = {
	   x
	  }

	  @Verifier(tx)
	  func verify() = {
	     getPreviousAnswer() == 100500
	  }
	*/
	env, _ := testInvokeEnv(true)
	code := "AAIDAAAAAAAAAAAAAAACAAAAAAF4AAAAAAAAAYiUAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAAFAAAAAXgAAAAAAAAAAQAAAAJ0eAEAAAAGdmVyaWZ5AAAAAAkAAAAAAAACCQEAAAARZ2V0UHJldmlvdXNBbnN3ZXIAAAAAAAAAAAAAAYiUa4pU5Q=="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallVerifier(env, tree)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	assert.True(t, r.Result())
}

func TestTransferSet(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 3 #-}
	   {-# CONTENT_TYPE DAPP #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}

	   @Callable(i)
	   func tellme(question: String) = {
	       TransferSet([ScriptTransfer(i.caller, 100, unit)])
	   }
	*/
	env, tx := testInvokeEnv(false)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, tx.SenderPK)
	require.NoError(t, err)
	code := "AAIDAAAAAAAAAAAAAAAAAAAAAQAAAAFpAQAAAAZ0ZWxsbWUAAAABAAAACHF1ZXN0aW9uCQEAAAALVHJhbnNmZXJTZXQAAAABCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAAAAAAZAUAAAAEdW5pdAUAAAADbmlsAAAAAH5a2L0="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "tellme", proto.Arguments{proto.NewIntegerArgument(100500)})
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	scriptTransfer := proto.TransferScriptAction{
		Recipient: proto.NewRecipientFromAddress(addr),
		Amount:    100,
		Asset:     proto.NewOptionalAssetWaves(),
	}
	require.NoError(t, err)
	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	require.EqualValues(t,
		&proto.ScriptResult{
			DataEntries:  make([]*proto.DataEntryScriptAction, 0),
			Transfers:    []*proto.TransferScriptAction{&scriptTransfer},
			Issues:       make([]*proto.IssueScriptAction, 0),
			Reissues:     make([]*proto.ReissueScriptAction, 0),
			Burns:        make([]*proto.BurnScriptAction, 0),
			Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
			Leases:       make([]*proto.LeaseScriptAction, 0),
			LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
		},
		sr,
	)
}

func TestScriptResult(t *testing.T) {
	/*
	   {-# STDLIB_VERSION 3 #-}
	   {-# CONTENT_TYPE DAPP #-}
	   {-# SCRIPT_TYPE ACCOUNT #-}

	   @Callable(i)
	   func tellme(question: String) = {
	       ScriptResult(
	           WriteSet([DataEntry("key", 100)]),
	           TransferSet([ScriptTransfer(i.caller, 100500, unit)])
	       )
	   }
	*/
	env, tx := testInvokeEnv(false)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, tx.SenderPK)
	require.NoError(t, err)
	code := "AAIDAAAAAAAAAAAAAAAAAAAAAQAAAAFpAQAAAAZ0ZWxsbWUAAAABAAAACHF1ZXN0aW9uCQEAAAAMU2NyaXB0UmVzdWx0AAAAAgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAADa2V5AAAAAAAAAABkBQAAAANuaWwJAQAAAAtUcmFuc2ZlclNldAAAAAEJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyAAAAAAAAAYiUBQAAAAR1bml0BQAAAANuaWwAAAAARKRntw=="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "tellme", proto.Arguments{proto.NewIntegerArgument(100)})
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	scriptTransfer := proto.TransferScriptAction{
		Recipient: proto.NewRecipientFromAddress(addr),
		Amount:    100500,
		Asset:     proto.NewOptionalAssetWaves(),
	}
	require.Equal(t,
		&proto.ScriptResult{
			DataEntries:  []*proto.DataEntryScriptAction{{Entry: &proto.IntegerDataEntry{Key: "key", Value: 100}}},
			Transfers:    []*proto.TransferScriptAction{&scriptTransfer},
			Issues:       make([]*proto.IssueScriptAction, 0),
			Reissues:     make([]*proto.ReissueScriptAction, 0),
			Burns:        make([]*proto.BurnScriptAction, 0),
			Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
			Leases:       make([]*proto.LeaseScriptAction, 0),
			LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
		},
		sr,
	)
}

func initWrappedState(state types.SmartState, env *mockRideEnvironment) *WrappedState {
	return &WrappedState{diff: newDiffState(state), cle: env.this().(rideAddress), scheme: env.scheme()}
}

var wrappedSt WrappedState
var firstScript string
var secondScript string
var assetIDIssue crypto.Digest
var addr proto.WavesAddress
var addrPK crypto.PublicKey
var addressCallable proto.WavesAddress
var addressCallablePK crypto.PublicKey

func smartStateDappFromDapp() types.SmartState {
	expectedAsset := crypto.MustDigestFromBase58("13YvHUb3bg7sXgExc6kFcCUKm6WYpJX9rLpHVhiyJNGJ")

	return &MockSmartState{
		NewestLeasingInfoFunc: func(id crypto.Digest) (*proto.LeaseInfo, error) {
			return nil, nil
		},
		GetByteTreeFunc: func(recipient proto.Recipient) (proto.Script, error) {
			var script proto.Script
			var err error
			if recipient.Address.String() == addr.String() {
				script, err = base64.StdEncoding.DecodeString(firstScript)
			}
			if recipient.Address.String() == addressCallable.String() {
				script, err = base64.StdEncoding.DecodeString(secondScript)
			}
			if err != nil {
				return proto.Script{}, err
			}
			return script, nil
		},
		NewestScriptByAssetFunc: func(assetID crypto.Digest) (proto.Script, error) {
			if assetID == expectedAsset {
				script := "BQQAAAALZEFwcEFkZHJlc3MJAAQmAAAAAQIAAAAjM1A4ZVpWS1M3YTR0cm9HY2t5dHhhZWZMQWk5dzdQNWFNbmEEAAAAByRtYXRjaDAFAAAAAnR4AwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAA9CdXJuVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwCQAAAAAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIFAAAAC2RBcHBBZGRyZXNzAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABJSZWlzc3VlVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwCQAAAAAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIFAAAAC2RBcHBBZGRyZXNzAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABlTZXRBc3NldFNjcmlwdFRyYW5zYWN0aW9uBAAAAAJ0eAUAAAAHJG1hdGNoMAkAAAAAAAACCAUAAAACdHgAAAAGc2VuZGVyBQAAAAtkQXBwQWRkcmVzcwMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwCQAAAAAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIFAAAAC2RBcHBBZGRyZXNzAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABNUcmFuc2ZlclRyYW5zYWN0aW9uBAAAAAJ0eAUAAAAHJG1hdGNoMAkAAAAAAAACCAUAAAACdHgAAAAGc2VuZGVyBQAAAAtkQXBwQWRkcmVzcwf56Ssf"

				src, err := base64.StdEncoding.DecodeString(script)
				if err != nil {
					return nil, err
				}
				return src, nil
			}
			return nil, nil
		},
		NewestRecipientToAddressFunc: func(recipient proto.Recipient) (*proto.WavesAddress, error) {
			if recipient.Alias != nil {
				if recipient.Alias.Alias == "alias" {
					addr, err := proto.NewAddressFromString("3MsCoDnBbgzjQ7BgGk9xcruM6JVZ5jF8YCV")
					return &addr, err
				}
			}
			return recipient.Address, nil
		},
		AddingBlockHeightFunc: func() (uint64, error) {
			return 10, nil
		},
		NewestScriptPKByAddrFunc: func(address proto.WavesAddress) (crypto.PublicKey, error) {
			// payments test
			if address.String() == "3P8eZVKS7a4troGckytxaefLAi9w7P5aMna" {
				return crypto.NewPublicKeyFromBase58("FztxsodUc9V7iVzodkGumnZFtHnNTxYSETZfxBFAw9R3")
			}
			if address.String() == "3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv" {
				return crypto.NewPublicKeyFromBase58("pmDSxpnULiroUAerTDFBajffTpqgwVJjtMipQq6DQM5")
			}
			if address.String() == "3MsCoDnBbgzjQ7BgGk9xcruM6JVZ5jF8YCV" {
				return crypto.NewPublicKeyFromBase58("AQj4MhySztn4FB3PxXc1ZcHPknLmGFYEKuSBz2vXeJPY")
			}
			if address == addr {
				return crypto.NewPublicKeyFromBase58("JBjPD1xkTTcYUVhbLp1bLJLcjDKLT3c32RVk9Rue87ZD")
			}
			if address == addressCallable {
				return crypto.NewPublicKeyFromBase58("8TLsCqkkroVot9dVR1WcWUN9Qx96HDfzG3hnx7NpSJA9")
			}

			return crypto.PublicKey{}, errors.Errorf("No pk from address")
		},
		NewestTransactionByIDFunc: func(id []byte) (proto.Transaction, error) {
			return nil, nil
		},
		NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
			return 0, nil
		},
		NewestAssetBalanceFunc: func(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
			return 0, nil
		},
		NewestAddrByAliasFunc: func(alias proto.Alias) (proto.WavesAddress, error) {
			if alias.Alias == "alias" {
				return proto.NewAddressFromString("3MsCoDnBbgzjQ7BgGk9xcruM6JVZ5jF8YCV")
			}
			return proto.WavesAddress{}, errors.New("unexpected alias")
		},
		NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
			return &proto.FullWavesBalance{
				Regular:    0,
				Generating: 0,
				Available:  0,
				Effective:  0,
				LeaseIn:    0,
				LeaseOut:   0,
			}, nil
		},
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			return nil, nil
		},
		RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
			return nil, nil
		},
		RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {

			return nil, nil
		},
		RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {

			return nil, nil
		},
		NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
			return false, nil
		},
		NewestAssetInfoFunc: func(assetID crypto.Digest) (*proto.AssetInfo, error) {
			if assetID == expectedAsset {
				return &proto.AssetInfo{
					ID:              expectedAsset,
					Quantity:        1000,
					Decimals:        '8',
					Issuer:          addressCallable,
					IssuerPublicKey: addressCallablePK,
					Reissuable:      true,
					Scripted:        true,
					Sponsored:       false,
				}, nil
			}
			return nil, nil
		},
		NewestFullAssetInfoFunc: func(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
			if assetID == expectedAsset {
				assetInfo := proto.AssetInfo{
					ID:              expectedAsset,
					Quantity:        1000,
					Decimals:        '8',
					Issuer:          addressCallable,
					IssuerPublicKey: addressCallablePK,
					Reissuable:      true,
					Scripted:        true,
					Sponsored:       false,
				}

				scriptB := "BQQAAAALZEFwcEFkZHJlc3MJAAQmAAAAAQIAAAAjM1A4ZVpWS1M3YTR0cm9HY2t5dHhhZWZMQWk5dzdQNWFNbmEEAAAAByRtYXRjaDAFAAAAAnR4AwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAA9CdXJuVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwCQAAAAAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIFAAAAC2RBcHBBZGRyZXNzAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABJSZWlzc3VlVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwCQAAAAAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIFAAAAC2RBcHBBZGRyZXNzAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABlTZXRBc3NldFNjcmlwdFRyYW5zYWN0aW9uBAAAAAJ0eAUAAAAHJG1hdGNoMAkAAAAAAAACCAUAAAACdHgAAAAGc2VuZGVyBQAAAAtkQXBwQWRkcmVzcwMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwCQAAAAAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIFAAAAC2RBcHBBZGRyZXNzAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABNUcmFuc2ZlclRyYW5zYWN0aW9uBAAAAAJ0eAUAAAAHJG1hdGNoMAkAAAAAAAACCAUAAAACdHgAAAAGc2VuZGVyBQAAAAtkQXBwQWRkcmVzcwf56Ssf"

				src, err := base64.StdEncoding.DecodeString(scriptB)
				if err != nil {
					return nil, err
				}

				scriptInfo := proto.ScriptInfo{
					Version:    5,
					Bytes:      src,
					Base64:     "",
					Complexity: 0,
				}

				return &proto.FullAssetInfo{
					AssetInfo:       assetInfo,
					Name:            "CatCoin",
					Description:     "",
					ScriptInfo:      scriptInfo,
					SponsorshipCost: uint64(0),
				}, nil
			}
			return nil, nil
		},
	}
}

var thisAddress proto.WavesAddress
var tx *proto.InvokeScriptWithProofs
var inv rideObject
var id []byte

func WrappedStateFunc() types.SmartState {
	return &wrappedSt
}

var envDappFromDapp = &mockRideEnvironment{
	setInvocationFunc: func(invocation rideObject) {
		inv = invocation
	},
	schemeFunc: func() byte {
		return proto.MainNetScheme
	},
	stateFunc: WrappedStateFunc,
	txIDFunc: func() rideType {
		return rideBytes(id)
	},
	thisFunc: func() rideType {
		return rideAddress(thisAddress)
	},
	setNewDAppAddressFunc: func(address proto.WavesAddress) {
		thisAddress = address
		wrappedSt.cle = rideAddress(address)
	},
	transactionFunc: func() rideObject {
		obj, _ := transactionToObject(proto.MainNetScheme, tx)
		return obj
	},
	invocationFunc: func() rideObject {
		return inv
	},
	timestampFunc: func() uint64 {
		return 1564703444249
	},
	blockV5ActivatedFunc: func() bool {
		return true
	},
	rideV6ActivatedFunc: noRideV6,
	validateInternalPaymentsFunc: func() bool {
		return true
	},
	internalPaymentsValidationHeightFunc: func() uint64 {
		return 0
	},
	maxDataEntriesSizeFunc: func() int {
		return proto.MaxDataEntriesScriptActionsSizeInBytesV2
	},
}

func tearDownDappFromDapp() {
	wrappedSt = WrappedState{}
	firstScript = ""
	secondScript = ""
	assetIDIssue = crypto.Digest{}
	addr = proto.WavesAddress{}
	addressCallable = proto.WavesAddress{}
	addrPK = crypto.PublicKey{}
	addressCallablePK = crypto.PublicKey{}

	thisAddress = proto.WavesAddress{}
	tx = nil
	id = nil
}

func AddExternalPayments(externalPayments proto.ScriptPayments, callerPK crypto.PublicKey) error {
	caller, err := proto.NewAddressFromPublicKey(envDappFromDapp.scheme(), callerPK)
	if err != nil {
		return err
	}
	recipient := proto.NewRecipientFromAddress(wrappedSt.callee())

	for _, payment := range externalPayments {
		var (
			senderBalance uint64
			err           error
			callerRcp     = proto.NewRecipientFromAddress(caller)
		)
		if payment.Asset.Present {
			senderBalance, err = wrappedSt.NewestAssetBalance(callerRcp, payment.Asset.ID)
		} else {
			senderBalance, err = wrappedSt.NewestWavesBalance(callerRcp)
		}
		if err != nil {
			return err
		}
		if senderBalance < payment.Amount {
			return errors.New("not enough money for tx attached payments")
		}

		searchBalance, searchAddr, err := wrappedSt.diff.findBalance(recipient, payment.Asset)
		if err != nil {
			return err
		}
		err = wrappedSt.diff.changeBalance(searchBalance, searchAddr, int64(payment.Amount), payment.Asset, recipient)
		if err != nil {
			return err
		}

		senderSearchBalance, senderSearchAddr, err := wrappedSt.diff.findBalance(callerRcp, payment.Asset)
		if err != nil {
			return err
		}

		err = wrappedSt.diff.changeBalance(senderSearchBalance, senderSearchAddr, -int64(payment.Amount), payment.Asset, callerRcp)
		if err != nil {
			return err
		}
	}
	return nil
}

func AddAssetToSender(senderAddress proto.WavesAddress, amount int64, asset proto.OptionalAsset) error {
	senderRecipient := proto.NewRecipientFromAddress(senderAddress)

	searchBalance, searchAddr, err := wrappedSt.diff.findBalance(senderRecipient, asset)
	if err != nil {
		return err
	}
	err = wrappedSt.diff.changeBalance(searchBalance, searchAddr, amount, asset, senderRecipient)
	if err != nil {
		return err
	}

	return nil
}

func TestInvokeDAppFromDAppAllActions(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func test() = {
		let res = invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "testActions",[], [AttachedPayment(unit, 1234), AttachedPayment(unit, 1234)])
		if res == 17
	        then
	        [
	         IntegerEntry("key", 1)
	        ]
	        else
	         throw("Bad returned value")
	}
	*/

	/* script 2
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func testActions() = {
		  let asset = Issue("CatCoin", "", 1, 0, true, unit, 0)
		  let assetId = asset.calculateAssetId()

		  ([
		    ScriptTransfer(Address(base58'3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv'), 1, unit),
	        Lease(Address(base58'3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv'), 10),
		    BinaryEntry("bin", base58''),
		    BooleanEntry("bool", true),
		    IntegerEntry("int", 1),
		    StringEntry("str", ""),
		    DeleteEntry("str"),
		    asset,
		    Reissue(assetId, 10, false),
		    Burn(assetId, 5)
		  ], 17)
		}
	*/

	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)

	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	recipientCallable := proto.NewRecipientFromAddress(addressCallable)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}
	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEdGVzdAAAAAAEAAAAA3JlcwkAA/wAAAAECQEAAAAHQWRkcmVzcwAAAAEBAAAAGgFXSbIqC+dSm+dDCCL8KamODy9oLyPQygrLAgAAAAt0ZXN0QWN0aW9ucwUAAAADbmlsCQAETAAAAAIJAQAAAA9BdHRhY2hlZFBheW1lbnQAAAACBQAAAAR1bml0AAAAAAAAAATSCQAETAAAAAIJAQAAAA9BdHRhY2hlZFBheW1lbnQAAAACBQAAAAR1bml0AAAAAAAAAATSBQAAAANuaWwDCQAAAAAAAAIFAAAAA3JlcwAAAAAAAAAAEQkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADa2V5AAAAAAAAAAABBQAAAANuaWwJAAACAAAAAQIAAAASQmFkIHJldHVybmVkIHZhbHVlAAAAAGGSphc="
	secondScript = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAALdGVzdEFjdGlvbnMAAAAABAAAAAVhc3NldAkABEMAAAAHAgAAAAdDYXRDb2luAgAAAAAAAAAAAAAAAAEAAAAAAAAAAAAGBQAAAAR1bml0AAAAAAAAAAAABAAAAAdhc3NldElkCQAEOAAAAAEFAAAABWFzc2V0CQAFFAAAAAIJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwkBAAAAB0FkZHJlc3MAAAABAQAAABoBV5hs3CAFUz6eTef/H4C7v1yCbCqvykvRuQAAAAAAAAAAAQUAAAAEdW5pdAkABEwAAAACCQAERAAAAAIJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVeYbNwgBVM+nk3n/x+Au79cgmwqr8pL0bkAAAAAAAAAAAoJAARMAAAAAgkBAAAAC0JpbmFyeUVudHJ5AAAAAgIAAAADYmluAQAAAAAJAARMAAAAAgkBAAAADEJvb2xlYW5FbnRyeQAAAAICAAAABGJvb2wGCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAANpbnQAAAAAAAAAAAEJAARMAAAAAgkBAAAAC1N0cmluZ0VudHJ5AAAAAgIAAAADc3RyAgAAAAAJAARMAAAAAgkBAAAAC0RlbGV0ZUVudHJ5AAAAAQIAAAADc3RyCQAETAAAAAIFAAAABWFzc2V0CQAETAAAAAIJAQAAAAdSZWlzc3VlAAAAAwUAAAAHYXNzZXRJZAAAAAAAAAAACgcJAARMAAAAAgkBAAAABEJ1cm4AAAACBQAAAAdhc3NldElkAAAAAAAAAAAFBQAAAANuaWwAAAAAAAAAABEAAAAAeF27eQ=="

	id = bytes.Repeat([]byte{0}, 32)

	expectedIssueWrites := []*proto.IssueScriptAction{
		{Sender: &addressCallablePK, Name: "CatCoin", Description: "", Quantity: 1, Decimals: 0, Reissuable: true, Script: nil, Nonce: 0},
	}
	expectedReissueWrites := []*proto.ReissueScriptAction{
		{Sender: &addressCallablePK, Quantity: 10, Reissuable: false},
	}
	expectedBurnWrites := []*proto.BurnScriptAction{
		{Sender: &addressCallablePK, Quantity: 5},
	}

	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.BinaryDataEntry{Key: "bin", Value: []byte("")}, Sender: &addressCallablePK},
		{Entry: &proto.BooleanDataEntry{Key: "bool", Value: true}, Sender: &addressCallablePK},
		{Entry: &proto.IntegerDataEntry{Key: "int", Value: 1}, Sender: &addressCallablePK},
		{Entry: &proto.StringDataEntry{Key: "str", Value: ""}, Sender: &addressCallablePK},
		{Entry: &proto.DeleteDataEntry{Key: "str"}, Sender: &addressCallablePK},
		{Entry: &proto.IntegerDataEntry{Key: "key", Value: 1}},
	}

	expectedTransferWrites := []*proto.TransferScriptAction{
		{Sender: &addressCallablePK, Recipient: recipient, Amount: 1, Asset: proto.OptionalAsset{}},
	}

	expectedAttachedPaymentActions := []*proto.AttachedPaymentScriptAction{
		{Sender: &addrPK, Recipient: recipientCallable, Amount: 1234, Asset: proto.OptionalAsset{}},
		{Sender: &addrPK, Recipient: recipientCallable, Amount: 1234, Asset: proto.OptionalAsset{}},
	}

	expectedLeaseWrites := []*proto.LeaseScriptAction{
		{Sender: &addressCallablePK, Recipient: recipient, Amount: 10, Nonce: 0},
	}

	smartState := smartStateDappFromDapp

	thisAddress = addr
	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	pid, ok := env.txID().(rideBytes)
	require.True(t, ok)
	d, err := crypto.NewDigestFromBytes(pid)
	require.NoError(t, err)
	expectedIssueWrites[0].ID = proto.GenerateIssueScriptActionID(expectedIssueWrites[0].Name, expectedIssueWrites[0].Description, int64(expectedIssueWrites[0].Decimals), expectedIssueWrites[0].Quantity, expectedIssueWrites[0].Reissuable, expectedIssueWrites[0].Nonce, d)
	expectedReissueWrites[0].AssetID = expectedIssueWrites[0].ID
	expectedBurnWrites[0].AssetID = expectedIssueWrites[0].ID
	assetIDIssue = expectedIssueWrites[0].ID

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "test", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedAttachedPaymentActions, ap)
	expectedLeaseWrites[0].ID = sr.Leases[0].ID
	assetExp := proto.NewOptionalAssetWaves()

	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  expectedDataEntryWrites,
		Transfers:    expectedTransferWrites,
		Issues:       expectedIssueWrites,
		Reissues:     expectedReissueWrites,
		Burns:        expectedBurnWrites,
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       expectedLeaseWrites,
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}

	assert.Equal(t, expectedActionsResult, sr)

	fullBalanceExpected := &proto.FullWavesBalance{
		Regular:    7533,
		Generating: 0,
		Available:  7533,
		Effective:  7543,
		LeaseIn:    10,
		LeaseOut:   0,
	}
	fullBalance, err := wrappedSt.NewestFullWavesBalance(recipient)

	require.NoError(t, err)
	assert.Equal(t, fullBalance, fullBalanceExpected)

	fullBalanceCallableExpected := &proto.FullWavesBalance{
		Regular:    2467,
		Generating: 0,
		Available:  2457,
		Effective:  2457,
		LeaseIn:    0,
		LeaseOut:   10,
	}
	fullBalanceCallable, err := wrappedSt.NewestFullWavesBalance(recipientCallable)
	require.NoError(t, err)
	assert.Equal(t, fullBalanceCallable, fullBalanceCallableExpected)

	expectedDiffResult := initWrappedState(smartState(), env).diff
	balance := diffBalance{regular: 7533, leaseIn: 10, asset: assetExp, effectiveHistory: []int64{7543}}
	expectedDiffResult.balances[balanceDiffKey{addr, assetExp}] = balance

	balanceSender := diffBalance{regular: 0, leaseOut: 0, asset: assetExp}
	expectedDiffResult.balances[balanceDiffKey{senderAddress, assetExp}] = balanceSender

	balanceCallable := diffBalance{regular: 2467, leaseOut: 10, asset: assetExp, effectiveHistory: []int64{2467, 2457}}
	expectedDiffResult.balances[balanceDiffKey{addressCallable, assetExp}] = balanceCallable

	assetFromIssue := *proto.NewOptionalAssetFromDigest(sr.Issues[0].ID)
	balanceCallableAsset := diffBalance{regular: 6, leaseOut: 0, asset: assetFromIssue} // +1 after Issue. + 10 after Reissue. -5 after Burn. = 6
	expectedDiffResult.balances[balanceDiffKey{addressCallable, assetFromIssue}] = balanceCallableAsset

	intEntry := proto.IntegerDataEntry{Key: "int", Value: 1}
	expectedDiffResult.dataEntries.diffInteger[integerDataEntryKey{intEntry.Key, addressCallable}] = intEntry

	boolEntry := proto.BooleanDataEntry{Key: "bool", Value: true}
	expectedDiffResult.dataEntries.diffBool[booleanDataEntryKey{boolEntry.Key, addressCallable}] = boolEntry

	stringEntry := proto.StringDataEntry{Key: "str", Value: ""}
	expectedDiffResult.dataEntries.diffString[stringDataEntryKey{stringEntry.Key, addressCallable}] = stringEntry

	binaryEntry := proto.BinaryDataEntry{Key: "bin", Value: []byte("")}
	expectedDiffResult.dataEntries.diffBinary[binaryDataEntryKey{binaryEntry.Key, addressCallable}] = binaryEntry

	deleteEntry := proto.DeleteDataEntry{Key: "str"}
	expectedDiffResult.dataEntries.diffDelete[deleteDataEntryKey{deleteEntry.Key, addressCallable}] = deleteEntry

	newAsset := diffNewAssetInfo{dAppIssuer: addressCallable, name: "CatCoin", description: "", quantity: 6, decimals: 0, reissuable: false, script: nil, nonce: 0}
	expectedDiffResult.newAssetsInfo[assetIDIssue] = newAsset

	lease := lease{Recipient: recipient, Sender: recipientCallable, leasedAmount: 10}
	expectedDiffResult.leases[expectedLeaseWrites[0].ID] = lease

	assert.Equal(t, expectedDiffResult.newAssetsInfo, wrappedSt.diff.newAssetsInfo)
	assert.Equal(t, expectedDiffResult.oldAssetsInfo, wrappedSt.diff.oldAssetsInfo)
	assert.Equal(t, expectedDiffResult.dataEntries, wrappedSt.diff.dataEntries)
	assert.Equal(t, expectedDiffResult.balances, wrappedSt.diff.balances)
	assert.Equal(t, expectedDiffResult.sponsorships, wrappedSt.diff.sponsorships)
	assert.Equal(t, expectedDiffResult.leases, wrappedSt.diff.leases)

	tearDownDappFromDapp()
}

func TestInvokeDAppFromDAppScript1(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5#-}
	{-# CONTENT_TYPE DAPP #-}
	{-#SCRIPT_TYPE ACCOUNT#-}

	 @Callable(i)
	 func bar() = {
	   ([IntegerEntry("bar", 1)], "return")
	 }

	 @Callable(i)
	 func foo() = {
	  let r = Invoke(this, "bar", [], [])
	  if r == "return"
	  then
	   let data = getIntegerValue(this, "bar")
	   if data == 1
	   then
	    [
	     IntegerEntry("key", 1)
	    ]
	   else
	    throw("Bad state")
	  else
	   throw("Bad returned value")
	 }
	*/
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        nil,
		FeeAsset:        proto.OptionalAsset{},
		Fee:             900000,
		Timestamp:       1564703444249,
	}

	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAYIAhIAEgAAAAAAAAAAAgAAAAFpAQAAAANiYXIAAAAACQAFFAAAAAIJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAICAAAAA2JhcgAAAAAAAAAAAQUAAAADbmlsAgAAAAZyZXR1cm4AAAABaQEAAAADZm9vAAAAAAQAAAABcgkAA/wAAAAEBQAAAAR0aGlzAgAAAANiYXIFAAAAA25pbAUAAAADbmlsAwkAAAAAAAACBQAAAAFyAgAAAAZyZXR1cm4EAAAABGRhdGEJAQAAABFAZXh0ck5hdGl2ZSgxMDUwKQAAAAIFAAAABHRoaXMCAAAAA2JhcgMJAAAAAAAAAgUAAAAEZGF0YQAAAAAAAAAAAQkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADa2V5AAAAAAAAAAABBQAAAANuaWwJAAACAAAAAQIAAAAJQmFkIHN0YXRlCQAAAgAAAAECAAAAEkJhZCByZXR1cm5lZCB2YWx1ZQAAAADz23Fz"

	id = bytes.Repeat([]byte{0}, 32)

	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "bar", Value: 1}, Sender: &addrPK},
		{Entry: &proto.IntegerDataEntry{Key: "key", Value: 1}},
	}

	smartState := smartStateDappFromDapp

	thisAddress = addr
	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "foo", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  expectedDataEntryWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedActionsResult, sr)

	expectedDiffResult := initWrappedState(smartState(), env).diff

	intEntry := proto.IntegerDataEntry{Key: "bar", Value: 1}
	expectedDiffResult.dataEntries.diffInteger[integerDataEntryKey{intEntry.Key, addr}] = intEntry

	assert.Equal(t, expectedDiffResult.dataEntries, wrappedSt.diff.dataEntries)

	tearDownDappFromDapp()
}

func TestInvokeDAppFromDAppScript2(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-#SCRIPT_TYPE ACCOUNT#-}

	 @Callable(i)
	 func foo() = {
	  let b1 = wavesBalance(this)
	  let ob1 = wavesBalance(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'))
	  if b1 == b1 && ob1 == ob1
	  then
	    let r = Invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "bar", [this.bytes], [AttachedPayment(unit, 17)])
	    if r == 17
	    then
	     let data = getIntegerValue(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "bar")
	     let b2 = wavesBalance(this)
	     let ob2 = wavesBalance(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'))
	     if data == 1
	     then
	      if ob1.regular+14 == ob2.regular && b1.regular == b2.regular+14
	      then
	       [
	        IntegerEntry("key", 1)
	       ]
	      else
	       throw("Balance check failed")
	    else
	     throw("Bad state")
	   else
	    throw("Bad returned value")
	  else
	   throw("Imposible")
	 }
	*/

	/*
		script2

		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-#SCRIPT_TYPE ACCOUNT#-}

		@Callable(i)
		func bar(a: ByteVector) = {
		   ([IntegerEntry("bar", 1), ScriptTransfer(Address(a), 3, unit)], 17)
		}
	*/
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)

	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	recipientCallable := proto.NewRecipientFromAddress(addressCallable)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}

	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAADZm9vAAAAAAQAAAACYjEJAAPvAAAAAQUAAAAEdGhpcwQAAAADb2IxCQAD7wAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssDAwkAAAAAAAACBQAAAAJiMQUAAAACYjEJAAAAAAAAAgUAAAADb2IxBQAAAANvYjEHBAAAAAFyCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAA2JhcgkABEwAAAACCAUAAAAEdGhpcwAAAAVieXRlcwUAAAADbmlsCQAETAAAAAIJAQAAAA9BdHRhY2hlZFBheW1lbnQAAAACBQAAAAR1bml0AAAAAAAAAAARBQAAAANuaWwDCQAAAAAAAAIFAAAAAXIAAAAAAAAAABEEAAAABGRhdGEJAQAAABFAZXh0ck5hdGl2ZSgxMDUwKQAAAAIJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAA2JhcgQAAAACYjIJAAPvAAAAAQUAAAAEdGhpcwQAAAADb2IyCQAD7wAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssDCQAAAAAAAAIFAAAABGRhdGEAAAAAAAAAAAEDAwkAAAAAAAACCQAAZAAAAAIIBQAAAANvYjEAAAAHcmVndWxhcgAAAAAAAAAADggFAAAAA29iMgAAAAdyZWd1bGFyCQAAAAAAAAIIBQAAAAJiMQAAAAdyZWd1bGFyCQAAZAAAAAIIBQAAAAJiMgAAAAdyZWd1bGFyAAAAAAAAAAAOBwkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADa2V5AAAAAAAAAAABBQAAAANuaWwJAAACAAAAAQIAAAAUQmFsYW5jZSBjaGVjayBmYWlsZWQJAAACAAAAAQIAAAAJQmFkIHN0YXRlCQAAAgAAAAECAAAAEkJhZCByZXR1cm5lZCB2YWx1ZQkAAAIAAAABAgAAAAlJbXBvc2libGUAAAAAjMpPSg=="
	secondScript = "AAIFAAAAAAAAAAcIAhIDCgECAAAAAAAAAAEAAAABaQEAAAADYmFyAAAAAQAAAAFhCQAFFAAAAAIJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAICAAAAA2JhcgAAAAAAAAAAAQkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCQEAAAAHQWRkcmVzcwAAAAEFAAAAAWEAAAAAAAAAAAMFAAAABHVuaXQFAAAAA25pbAAAAAAAAAAAEQAAAAAyrXjp"

	id = bytes.Repeat([]byte{0}, 32)

	expectedTransferWrites := []*proto.TransferScriptAction{
		{Sender: &addressCallablePK, Recipient: recipient, Amount: 3, Asset: proto.OptionalAsset{}},
	}

	expectedAttachedPaymentActions := []*proto.AttachedPaymentScriptAction{
		{Sender: &addrPK, Recipient: recipientCallable, Amount: 17, Asset: proto.OptionalAsset{}},
	}

	expectedDataEntryWrites := []*proto.DataEntryScriptAction{

		{Entry: &proto.IntegerDataEntry{Key: "bar", Value: 1}, Sender: &addressCallablePK},
		{Entry: &proto.IntegerDataEntry{Key: "key", Value: 1}},
	}

	smartState := smartStateDappFromDapp

	thisAddress = addr

	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "foo", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedAttachedPaymentActions, ap)
	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  expectedDataEntryWrites,
		Transfers:    expectedTransferWrites,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedActionsResult, sr)

	expectedDiffResult := initWrappedState(smartState(), env).diff
	assetExp := proto.NewOptionalAssetWaves()

	balanceMain := diffBalance{asset: assetExp, regular: 9986, effectiveHistory: []int64{10000, 9986}}
	balanceSender := diffBalance{regular: 0, leaseOut: 0, asset: assetExp}
	balanceCallable := diffBalance{asset: assetExp, regular: 14, effectiveHistory: []int64{0, 14}}
	intEntry := proto.IntegerDataEntry{Key: "bar", Value: 1}
	expectedDiffResult.dataEntries.diffInteger[integerDataEntryKey{intEntry.Key, addressCallable}] = intEntry
	expectedDiffResult.balances[balanceDiffKey{addressCallable, assetExp}] = balanceCallable
	expectedDiffResult.balances[balanceDiffKey{senderAddress, assetExp}] = balanceSender
	expectedDiffResult.balances[balanceDiffKey{addr, assetExp}] = balanceMain

	assert.Equal(t, expectedDiffResult.dataEntries, wrappedSt.diff.dataEntries)
	assert.Equal(t, expectedDiffResult.balances, wrappedSt.diff.balances)

	tearDownDappFromDapp()
}

func TestInvokeDAppFromDAppScript3(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-#SCRIPT_TYPE ACCOUNT#-}

	 @Callable(i)
	 func foo() = {
	  let b1 = wavesBalance(this)
	  let ob1 = wavesBalance(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'))
	  if b1 == b1 && ob1 == ob1
	  then
	    let r = Invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "bar", [this.bytes], [AttachedPayment(unit, 17)])
	    if r == 17
	    then
	     let data = getIntegerValue(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "bar")
	     let b2 = wavesBalance(this)
	     let ob2 = wavesBalance(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'))
	     if data == 1
	     then
	      if ob1.regular+14 == ob2.regular && b1.regular == b2.regular+14
	      then
	       let r1 = Invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "bar", [this.bytes], [AttachedPayment(unit, 18)])
	       if r1 == r1
	       then
	        let b3 = wavesBalance(this)
	        let ob3 = wavesBalance(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'))
	        if ob2.regular+15 == ob3.regular && b2.regular == b3.regular+15
	        then
	         [
	          IntegerEntry("key", 1)
	         ]
	        else
	         throw("Bad balance after second invoke")
	      else
	       throw("Imposible")
	     else
	      throw("Balance check failed")
	    else
	     throw("Bad state")
	  else
	   throw("Bad returned value")
	   else
	    throw("Imposible")
	 }
	*/

	/*
		script2

		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-#SCRIPT_TYPE ACCOUNT#-}

		@Callable(i)
		func bar(a: ByteVector) = {
		    ([IntegerEntry("bar", 1), ScriptTransfer(Address(a), 3, unit)], 17)
		}
	*/
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)

	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	recipientCallable := proto.NewRecipientFromAddress(addressCallable)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}

	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAADZm9vAAAAAAQAAAACYjEJAAPvAAAAAQUAAAAEdGhpcwQAAAADb2IxCQAD7wAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssDAwkAAAAAAAACBQAAAAJiMQUAAAACYjEJAAAAAAAAAgUAAAADb2IxBQAAAANvYjEHBAAAAAFyCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAA2JhcgkABEwAAAACCAUAAAAEdGhpcwAAAAVieXRlcwUAAAADbmlsCQAETAAAAAIJAQAAAA9BdHRhY2hlZFBheW1lbnQAAAACBQAAAAR1bml0AAAAAAAAAAARBQAAAANuaWwDCQAAAAAAAAIFAAAAAXIAAAAAAAAAABEEAAAABGRhdGEJAQAAABFAZXh0ck5hdGl2ZSgxMDUwKQAAAAIJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAA2JhcgQAAAACYjIJAAPvAAAAAQUAAAAEdGhpcwQAAAADb2IyCQAD7wAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssDCQAAAAAAAAIFAAAABGRhdGEAAAAAAAAAAAEDAwkAAAAAAAACCQAAZAAAAAIIBQAAAANvYjEAAAAHcmVndWxhcgAAAAAAAAAADggFAAAAA29iMgAAAAdyZWd1bGFyCQAAAAAAAAIIBQAAAAJiMQAAAAdyZWd1bGFyCQAAZAAAAAIIBQAAAAJiMgAAAAdyZWd1bGFyAAAAAAAAAAAOBwQAAAACcjEJAAP8AAAABAkBAAAAB0FkZHJlc3MAAAABAQAAABoBV0myKgvnUpvnQwgi/Cmpjg8vaC8j0MoKywIAAAADYmFyCQAETAAAAAIIBQAAAAR0aGlzAAAABWJ5dGVzBQAAAANuaWwJAARMAAAAAgkBAAAAD0F0dGFjaGVkUGF5bWVudAAAAAIFAAAABHVuaXQAAAAAAAAAABIFAAAAA25pbAMJAAAAAAAAAgUAAAACcjEFAAAAAnIxBAAAAAJiMwkAA+8AAAABBQAAAAR0aGlzBAAAAANvYjMJAAPvAAAAAQkBAAAAB0FkZHJlc3MAAAABAQAAABoBV0myKgvnUpvnQwgi/Cmpjg8vaC8j0MoKywMDCQAAAAAAAAIJAABkAAAAAggFAAAAA29iMgAAAAdyZWd1bGFyAAAAAAAAAAAPCAUAAAADb2IzAAAAB3JlZ3VsYXIJAAAAAAAAAggFAAAAAmIyAAAAB3JlZ3VsYXIJAABkAAAAAggFAAAAAmIzAAAAB3JlZ3VsYXIAAAAAAAAAAA8HCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAANrZXkAAAAAAAAAAAEFAAAAA25pbAkAAAIAAAABAgAAAB9CYWQgYmFsYW5jZSBhZnRlciBzZWNvbmQgaW52b2tlCQAAAgAAAAECAAAACUltcG9zaWJsZQkAAAIAAAABAgAAABRCYWxhbmNlIGNoZWNrIGZhaWxlZAkAAAIAAAABAgAAAAlCYWQgc3RhdGUJAAACAAAAAQIAAAASQmFkIHJldHVybmVkIHZhbHVlCQAAAgAAAAECAAAACUltcG9zaWJsZQAAAABDUPNk"
	secondScript = "AAIFAAAAAAAAAAcIAhIDCgECAAAAAAAAAAEAAAABaQEAAAADYmFyAAAAAQAAAAFhCQAFFAAAAAIJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAICAAAAA2JhcgAAAAAAAAAAAQkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCQEAAAAHQWRkcmVzcwAAAAEFAAAAAWEAAAAAAAAAAAMFAAAABHVuaXQFAAAAA25pbAAAAAAAAAAAEQAAAAAyrXjp"

	id = bytes.Repeat([]byte{0}, 32)

	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "bar", Value: 1}, Sender: &addressCallablePK},
		{Entry: &proto.IntegerDataEntry{Key: "bar", Value: 1}, Sender: &addressCallablePK},
		{Entry: &proto.IntegerDataEntry{Key: "key", Value: 1}},
	}

	expectedTransferWrites := []*proto.TransferScriptAction{
		{Sender: &addressCallablePK, Recipient: recipient, Amount: 3, Asset: proto.OptionalAsset{}},
		{Sender: &addressCallablePK, Recipient: recipient, Amount: 3, Asset: proto.OptionalAsset{}},
	}

	expectedAttachedPaymentActions := []*proto.AttachedPaymentScriptAction{
		{Sender: &addrPK, Recipient: recipientCallable, Amount: 17, Asset: proto.OptionalAsset{}},
		{Sender: &addrPK, Recipient: recipientCallable, Amount: 18, Asset: proto.OptionalAsset{}},
	}

	smartState := smartStateDappFromDapp

	thisAddress = addr

	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "foo", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedAttachedPaymentActions, ap)

	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  expectedDataEntryWrites,
		Transfers:    expectedTransferWrites,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedActionsResult, sr)

	expectedDiffResult := initWrappedState(smartState(), env).diff
	assetExp := proto.NewOptionalAssetWaves()

	balanceMain := diffBalance{asset: assetExp, regular: 9971, effectiveHistory: []int64{10000, 9986, 9971}}
	balanceSender := diffBalance{regular: 0, leaseOut: 0, asset: assetExp}
	balanceCallable := diffBalance{asset: assetExp, regular: 29, effectiveHistory: []int64{0, 14, 29}}
	intEntry := proto.IntegerDataEntry{Key: "bar", Value: 1}
	expectedDiffResult.dataEntries.diffInteger[integerDataEntryKey{intEntry.Key, addressCallable}] = intEntry
	expectedDiffResult.balances[balanceDiffKey{addr, assetExp}] = balanceMain
	expectedDiffResult.balances[balanceDiffKey{senderAddress, assetExp}] = balanceSender
	expectedDiffResult.balances[balanceDiffKey{addressCallable, assetExp}] = balanceCallable

	assert.Equal(t, expectedDiffResult.dataEntries, wrappedSt.diff.dataEntries)
	assert.Equal(t, expectedDiffResult.balances, wrappedSt.diff.balances)

	tearDownDappFromDapp()
}

func TestNegativeCycleNewInvokeDAppFromDAppScript4(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-#SCRIPT_TYPE ACCOUNT#-}

	 @Callable(i)
	 func back() = {
	   [IntegerEntry("key", 0), ScriptTransfer(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), 2, unit)]
	 }

	 @Callable(i)
	 func foo() = {
	  let b1 = wavesBalance(this)
	  let ob1 = wavesBalance(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'))
	  if b1 == b1 && ob1 == ob1
	  then
	    let r = Invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "bar", [this.bytes], [AttachedPayment(unit, 17)])
	    if r == 17
	    then
	     let data = getIntegerValue(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "bar")
	     let tdata = getIntegerValue(this, "key")
	     let b2 = wavesBalance(this)
	     let ob2 = wavesBalance(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'))
	     if data == 1 && tdata == 0
	     then
	      if ob1.regular+16 == ob2.regular && b1.regular == b2.regular+16
	      then
	       [
	        IntegerEntry("key", 1)
	       ]
	      else
	       throw("Balance check failed")
	    else
	     throw("Bad state")
	   else
	    throw("Bad returned value")
	  else
	   throw("Imposible")
	 }
	*/

	/*
			script2
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-#SCRIPT_TYPE ACCOUNT#-}

		 @Callable(i)
		 func bar(a: ByteVector) = {
		   let r = Invoke(Address(a), "back", [], [])
		   if r == r
		   then
		    ([IntegerEntry("bar", 1), ScriptTransfer(Address(a), 3, unit)], 17)
		   else
		    throw("Imposible")
		 }


	*/
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)

	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}
	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAYIAhIAEgAAAAAAAAAAAgAAAAFpAQAAAARiYWNrAAAAAAkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADa2V5AAAAAAAAAAAACQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssAAAAAAAAAAAIFAAAABHVuaXQFAAAAA25pbAAAAAFpAQAAAANmb28AAAAABAAAAAJiMQkAA+8AAAABBQAAAAR0aGlzBAAAAANvYjEJAAPvAAAAAQkBAAAAB0FkZHJlc3MAAAABAQAAABoBV0myKgvnUpvnQwgi/Cmpjg8vaC8j0MoKywMDCQAAAAAAAAIFAAAAAmIxBQAAAAJiMQkAAAAAAAACBQAAAANvYjEFAAAAA29iMQcEAAAAAXIJAAP8AAAABAkBAAAAB0FkZHJlc3MAAAABAQAAABoBV0myKgvnUpvnQwgi/Cmpjg8vaC8j0MoKywIAAAADYmFyCQAETAAAAAIIBQAAAAR0aGlzAAAABWJ5dGVzBQAAAANuaWwJAARMAAAAAgkBAAAAD0F0dGFjaGVkUGF5bWVudAAAAAIFAAAABHVuaXQAAAAAAAAAABEFAAAAA25pbAMJAAAAAAAAAgUAAAABcgAAAAAAAAAAEQQAAAAEZGF0YQkBAAAAEUBleHRyTmF0aXZlKDEwNTApAAAAAgkBAAAAB0FkZHJlc3MAAAABAQAAABoBV0myKgvnUpvnQwgi/Cmpjg8vaC8j0MoKywIAAAADYmFyBAAAAAV0ZGF0YQkBAAAAEUBleHRyTmF0aXZlKDEwNTApAAAAAgUAAAAEdGhpcwIAAAADa2V5BAAAAAJiMgkAA+8AAAABBQAAAAR0aGlzBAAAAANvYjIJAAPvAAAAAQkBAAAAB0FkZHJlc3MAAAABAQAAABoBV0myKgvnUpvnQwgi/Cmpjg8vaC8j0MoKywMDCQAAAAAAAAIFAAAABGRhdGEAAAAAAAAAAAEJAAAAAAAAAgUAAAAFdGRhdGEAAAAAAAAAAAAHAwMJAAAAAAAAAgkAAGQAAAACCAUAAAADb2IxAAAAB3JlZ3VsYXIAAAAAAAAAABAIBQAAAANvYjIAAAAHcmVndWxhcgkAAAAAAAACCAUAAAACYjEAAAAHcmVndWxhcgkAAGQAAAACCAUAAAACYjIAAAAHcmVndWxhcgAAAAAAAAAAEAcJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAICAAAAA2tleQAAAAAAAAAAAQUAAAADbmlsCQAAAgAAAAECAAAAFEJhbGFuY2UgY2hlY2sgZmFpbGVkCQAAAgAAAAECAAAACUJhZCBzdGF0ZQkAAAIAAAABAgAAABJCYWQgcmV0dXJuZWQgdmFsdWUJAAACAAAAAQIAAAAJSW1wb3NpYmxlAAAAAOgXYAY="
	secondScript = "AAIFAAAAAAAAAAcIAhIDCgECAAAAAAAAAAEAAAABaQEAAAADYmFyAAAAAQAAAAFhBAAAAAFyCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQUAAAABYQIAAAAEYmFjawUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAAXIFAAAAAXIJAAUUAAAAAgkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADYmFyAAAAAAAAAAABCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMJAQAAAAdBZGRyZXNzAAAAAQUAAAABYQAAAAAAAAAAAwUAAAAEdW5pdAUAAAADbmlsAAAAAAAAAAARCQAAAgAAAAECAAAACUltcG9zaWJsZQAAAACf+Ofn"

	id = bytes.Repeat([]byte{0}, 32)

	smartState := smartStateDappFromDapp

	thisAddress = addr

	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "foo", proto.Arguments{})
	require.Nil(t, res)
	require.Error(t, err)

	tearDownDappFromDapp()
}

func TestReentrantInvokeDAppFromDAppScript5(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-#SCRIPT_TYPE ACCOUNT#-}

	 @Callable(i)
	 func back() = {
	   [ScriptTransfer(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), 2, unit)]
	 }

	 @Callable(i)
	 func foo() = {
	  let b1 = wavesBalance(this)
	  let ob1 = wavesBalance(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'))
	  if b1 == b1 && ob1 == ob1
	  then
	    let r = reentrantInvoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "bar", [this.bytes], [AttachedPayment(unit, 17)])
	    if r == 17
	    then
	     let data = getIntegerValue(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "bar")
	     let b2 = wavesBalance(this)
	     let ob2 = wavesBalance(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'))
	     if data == 1
	     then
	      if ob1.regular+13 == ob2.regular && b1.regular == b2.regular+13
	      then
	       [
	        IntegerEntry("key", 1)
	       ]
	      else
	       throw("Balance check failed")
	    else
	     throw("Bad state")
	   else
	    throw("Bad returned value")
	  else
	   throw("Imposible")
	 }
	*/

	/*
			script2
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-#SCRIPT_TYPE ACCOUNT#-}

		 @Callable(i)
		 func bar(a: ByteVector) = {
		   let r = reentrantInvoke(Address(a), "back", [], [AttachedPayment(unit, 3)])
		   if r == r
		   then
		    ([IntegerEntry("bar", 1), ScriptTransfer(Address(a), 3, unit)], 17)
		   else
		    throw("Imposible")
		 }

	*/
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)

	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	recipientCallable := proto.NewRecipientFromAddress(addressCallable)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}
	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAYIAhIAEgAAAAAAAAAAAgAAAAFpAQAAAARiYWNrAAAAAAkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCQEAAAAHQWRkcmVzcwAAAAEBAAAAGgFXSbIqC+dSm+dDCCL8KamODy9oLyPQygrLAAAAAAAAAAACBQAAAAR1bml0BQAAAANuaWwAAAABaQEAAAADZm9vAAAAAAQAAAACYjEJAAPvAAAAAQUAAAAEdGhpcwQAAAADb2IxCQAD7wAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssDAwkAAAAAAAACBQAAAAJiMQUAAAACYjEJAAAAAAAAAgUAAAADb2IxBQAAAANvYjEHBAAAAAFyCQAD/QAAAAQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAA2JhcgkABEwAAAACCAUAAAAEdGhpcwAAAAVieXRlcwUAAAADbmlsCQAETAAAAAIJAQAAAA9BdHRhY2hlZFBheW1lbnQAAAACBQAAAAR1bml0AAAAAAAAAAARBQAAAANuaWwDCQAAAAAAAAIFAAAAAXIAAAAAAAAAABEEAAAABGRhdGEJAQAAABFAZXh0ck5hdGl2ZSgxMDUwKQAAAAIJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAA2JhcgQAAAACYjIJAAPvAAAAAQUAAAAEdGhpcwQAAAADb2IyCQAD7wAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssDCQAAAAAAAAIFAAAABGRhdGEAAAAAAAAAAAEDAwkAAAAAAAACCQAAZAAAAAIIBQAAAANvYjEAAAAHcmVndWxhcgAAAAAAAAAADQgFAAAAA29iMgAAAAdyZWd1bGFyCQAAAAAAAAIIBQAAAAJiMQAAAAdyZWd1bGFyCQAAZAAAAAIIBQAAAAJiMgAAAAdyZWd1bGFyAAAAAAAAAAANBwkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADa2V5AAAAAAAAAAABBQAAAANuaWwJAAACAAAAAQIAAAAUQmFsYW5jZSBjaGVjayBmYWlsZWQJAAACAAAAAQIAAAAJQmFkIHN0YXRlCQAAAgAAAAECAAAAEkJhZCByZXR1cm5lZCB2YWx1ZQkAAAIAAAABAgAAAAlJbXBvc2libGUAAAAAgjbh2Q=="
	secondScript = "AAIFAAAAAAAAAAcIAhIDCgECAAAAAAAAAAEAAAABaQEAAAADYmFyAAAAAQAAAAFhBAAAAAFyCQAD/QAAAAQJAQAAAAdBZGRyZXNzAAAAAQUAAAABYQIAAAAEYmFjawUAAAADbmlsCQAETAAAAAIJAQAAAA9BdHRhY2hlZFBheW1lbnQAAAACBQAAAAR1bml0AAAAAAAAAAADBQAAAANuaWwDCQAAAAAAAAIFAAAAAXIFAAAAAXIJAAUUAAAAAgkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADYmFyAAAAAAAAAAABCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMJAQAAAAdBZGRyZXNzAAAAAQUAAAABYQAAAAAAAAAAAwUAAAAEdW5pdAUAAAADbmlsAAAAAAAAAAARCQAAAgAAAAECAAAACUltcG9zaWJsZQAAAADwDv0i"

	id = bytes.Repeat([]byte{0}, 32)

	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "bar", Value: 1}, Sender: &addressCallablePK},
		{Entry: &proto.IntegerDataEntry{Key: "key", Value: 1}},
	}

	expectedTransferWrites := []*proto.TransferScriptAction{
		{Sender: &addrPK, Recipient: recipientCallable, Amount: 2, Asset: proto.OptionalAsset{}},
		{Sender: &addressCallablePK, Recipient: recipient, Amount: 3, Asset: proto.OptionalAsset{}},
	}

	expectedAttachedPaymentActions := []*proto.AttachedPaymentScriptAction{
		{Sender: &addrPK, Recipient: recipientCallable, Amount: 17, Asset: proto.OptionalAsset{}},
		{Sender: &addressCallablePK, Recipient: recipient, Amount: 3, Asset: proto.OptionalAsset{}},
	}

	smartState := smartStateDappFromDapp

	thisAddress = addr
	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "foo", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedAttachedPaymentActions, ap)
	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  expectedDataEntryWrites,
		Transfers:    expectedTransferWrites,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedActionsResult, sr)

	expectedDiffResult := initWrappedState(smartState(), env).diff
	assetExp := proto.NewOptionalAssetWaves()

	balanceMain := diffBalance{asset: assetExp, regular: 9987, effectiveHistory: []int64{10000, 9987}}
	balanceSender := diffBalance{asset: assetExp, regular: 0}
	balanceCallable := diffBalance{asset: assetExp, regular: 13, effectiveHistory: []int64{0, 13}}
	intEntry := proto.IntegerDataEntry{Key: "bar", Value: 1}
	expectedDiffResult.dataEntries.diffInteger[integerDataEntryKey{"bar", addressCallable}] = intEntry
	expectedDiffResult.balances[balanceDiffKey{addr, assetExp}] = balanceMain
	expectedDiffResult.balances[balanceDiffKey{senderAddress, assetExp}] = balanceSender
	expectedDiffResult.balances[balanceDiffKey{addressCallable, assetExp}] = balanceCallable

	assert.Equal(t, expectedDiffResult.dataEntries, wrappedSt.diff.dataEntries)
	assert.Equal(t, expectedDiffResult.balances, wrappedSt.diff.balances)

	tearDownDappFromDapp()
}

func TestInvokeDAppFromDAppScript6(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-#SCRIPT_TYPE ACCOUNT#-}

	 @Callable(i)
	 func foo() = {
	  let r = Invoke(this, "foo", [], [])
	  if r == r
	  then
	    [
	    ]
	  else
	   throw("Imposible")
	 }

	*/
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        nil,
		FeeAsset:        proto.OptionalAsset{},
		Fee:             900000,
		Timestamp:       1564703444249,
	}
	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAADZm9vAAAAAAQAAAABcgkAA/wAAAAEBQAAAAR0aGlzAgAAAANmb28FAAAAA25pbAUAAAADbmlsAwkAAAAAAAACBQAAAAFyBQAAAAFyBQAAAANuaWwJAAACAAAAAQIAAAAJSW1wb3NpYmxlAAAAAAWzLtA="

	id = bytes.Repeat([]byte{0}, 32)

	smartState := smartStateDappFromDapp

	thisAddress = addr

	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "foo", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  make([]*proto.DataEntryScriptAction, 0),
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedActionsResult, sr)

	expectedDiffResult := initWrappedState(smartState(), env).diff

	assert.Equal(t, expectedDiffResult.dataEntries, wrappedSt.diff.dataEntries)

	tearDownDappFromDapp()
}

func BenchmarkInvokeDAppFromDAppScript6(b *testing.B) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-#SCRIPT_TYPE ACCOUNT#-}

	 @Callable(i)
	 func foo() = {
	  let r = Invoke(this, "foo", [], [])
	  if r == r
	  then
	    [
	    ]
	  else
	   throw("Imposible")
	 }

	*/
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	if err != nil {
		b.Fatal("Expected no errors, got error ", err)
	}
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	if err != nil {
		b.Fatal("Expected no errors, got error ", err)
	}
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	if err != nil {
		b.Fatal("Expected no errors, got error ", err)
	}
	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	if err != nil {
		b.Fatal("Expected no errors, got error ", err)
	}
	recipient := proto.NewRecipientFromAddress(addr)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        nil,
		FeeAsset:        proto.OptionalAsset{},
		Fee:             900000,
		Timestamp:       1564703444249,
	}
	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAADZm9vAAAAAAQAAAABcgkAA/wAAAAEBQAAAAR0aGlzAgAAAANmb28FAAAAA25pbAUAAAADbmlsAwkAAAAAAAACBQAAAAFyBQAAAAFyBQAAAANuaWwJAAACAAAAAQIAAAAJSW1wb3NpYmxlAAAAAAWzLtA="

	id = bytes.Repeat([]byte{0}, 32)

	smartState := smartStateDappFromDapp

	thisAddress = addr

	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	src, err := base64.StdEncoding.DecodeString(firstScript)
	if err != nil {
		b.Fatal("Expected no errors, got error ", err)
	}

	tree, err := Parse(src)
	if err != nil {
		b.Fatal("Expected no errors, got error ", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CallFunction(env, tree, "foo", proto.Arguments{})
		if err != nil {
			b.Fatal("Expected no errors, got error ", err)
		}
	}
	tearDownDappFromDapp()
}

func TestReentrantInvokeDAppFromDAppScript6(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-#SCRIPT_TYPE ACCOUNT#-}

	 @Callable(i)
	 func foo() = {
	  let r = reentrantInvoke(this, "foo", [], [])
	  if r == r
	  then
	    [
	    ]
	  else
	   throw("Imposible")
	 }

	*/
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        nil,
		FeeAsset:        proto.OptionalAsset{},
		Fee:             900000,
		Timestamp:       1564703444249,
	}
	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAADZm9vAAAAAAQAAAABcgkAA/0AAAAEBQAAAAR0aGlzAgAAAANmb28FAAAAA25pbAUAAAADbmlsAwkAAAAAAAACBQAAAAFyBQAAAAFyBQAAAANuaWwJAAACAAAAAQIAAAAJSW1wb3NpYmxlAAAAALQe43c=\n"

	id = bytes.Repeat([]byte{0}, 32)

	smartState := smartStateDappFromDapp

	thisAddress = addr

	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "foo", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  make([]*proto.DataEntryScriptAction, 0),
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedActionsResult, sr)

	expectedDiffResult := initWrappedState(smartState(), env).diff

	assert.Equal(t, expectedDiffResult.dataEntries, wrappedSt.diff.dataEntries)

	tearDownDappFromDapp()
}

func TestInvokeDAppFromDAppPayments(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let exchangeRate = 5

	@Callable(i)
	func test() = if ((i.payments[0].assetId != unit))
	    then throw("unexpected asset")
	    else {
			let res = Invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "testActions",[(i.payments[0].amount * exchangeRate)], nil)
			if res == 17
	 	    then
		  	[
				ScriptTransfer(i.caller, (i.payments[0].amount * exchangeRate), unit)
		  	]
	    	else
	      		throw("Bad returned value")
	}	*/

	/* script 2
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func testActions(a: Int) = {
	  ([
	    IntegerEntry("int", 1)
	  ], 17)
	}	*/

	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	require.NoError(t, err)
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)
	senderRecipient := proto.NewRecipientFromAddress(senderAddress)
	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}
	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAQAAAAAMZXhjaGFuZ2VSYXRlAAAAAAAAAAAFAAAAAQAAAAFpAQAAAAR0ZXN0AAAAAAMJAQAAAAIhPQAAAAIICQABkQAAAAIIBQAAAAFpAAAACHBheW1lbnRzAAAAAAAAAAAAAAAAB2Fzc2V0SWQFAAAABHVuaXQJAAACAAAAAQIAAAAQdW5leHBlY3RlZCBhc3NldAQAAAADcmVzCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAC3Rlc3RBY3Rpb25zCQAETAAAAAIJAABoAAAAAggJAAGRAAAAAggFAAAAAWkAAAAIcGF5bWVudHMAAAAAAAAAAAAAAAAGYW1vdW50BQAAAAxleGNoYW5nZVJhdGUFAAAAA25pbAUAAAADbmlsAwkAAAAAAAACBQAAAANyZXMAAAAAAAAAABEJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyCQAAaAAAAAIICQABkQAAAAIIBQAAAAFpAAAACHBheW1lbnRzAAAAAAAAAAAAAAAABmFtb3VudAUAAAAMZXhjaGFuZ2VSYXRlBQAAAAR1bml0BQAAAANuaWwJAAACAAAAAQIAAAASQmFkIHJldHVybmVkIHZhbHVlAAAAANOWG8w="
	secondScript = "AAIFAAAAAAAAAAcIAhIDCgEBAAAAAAAAAAEAAAABaQEAAAALdGVzdEFjdGlvbnMAAAABAAAAAWEJAAUUAAAAAgkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADaW50AAAAAAAAAAABBQAAAANuaWwAAAAAAAAAABEAAAAAPWJMug=="

	id = bytes.Repeat([]byte{0}, 32)

	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "int", Value: 1}, Sender: &addressCallablePK},
	}

	expectedTransferWrites := []*proto.TransferScriptAction{
		{Recipient: senderRecipient, Amount: 50000, Asset: proto.OptionalAsset{}},
	}

	smartState := smartStateDappFromDapp

	thisAddress = addr
	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "test", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  expectedDataEntryWrites,
		Transfers:    expectedTransferWrites,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}

	assert.Equal(t, expectedActionsResult, sr)

	expectedDiffResult := initWrappedState(smartState(), env).diff
	assetExp := proto.NewOptionalAssetWaves()

	intEntry := proto.IntegerDataEntry{Key: "int", Value: 1}
	expectedDiffResult.dataEntries.diffInteger[integerDataEntryKey{intEntry.Key, addressCallable}] = intEntry

	balanceMain := diffBalance{asset: assetExp, regular: 10000}
	balanceSender := diffBalance{asset: assetExp, regular: 0}
	expectedDiffResult.balances[balanceDiffKey{addr, assetExp}] = balanceMain
	expectedDiffResult.balances[balanceDiffKey{senderAddress, assetExp}] = balanceSender

	assert.Equal(t, expectedDiffResult.dataEntries, wrappedSt.diff.dataEntries)
	assert.Equal(t, expectedDiffResult.balances, wrappedSt.diff.balances)

	tearDownDappFromDapp()
}

func TestInvokeDAppFromDAppNilResult(t *testing.T) {

	/* script 1
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		let exchangeRate = 5

		@Callable(i)
		func test() = if ((i.payments[0].assetId != unit))
		    then throw("unexpected asset")
		    else {
				let res = Invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "testActions",[(i.payments[0].amount * exchangeRate)], [AttachedPayment(unit, 1)])
				if res == 17
		 	    then
	                nil
		    	else
		      		throw("Bad returned value")
		}
	*/

	/* script 2
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func testActions(a: Int) = {
	  ([
	    IntegerEntry("int", 1)
	  ], 17)
	}
	*/

	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	require.NoError(t, err)
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)
	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	recipientCallable := proto.NewRecipientFromAddress(addressCallable)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}

	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAQAAAAAMZXhjaGFuZ2VSYXRlAAAAAAAAAAAFAAAAAQAAAAFpAQAAAAR0ZXN0AAAAAAMJAQAAAAIhPQAAAAIICQABkQAAAAIIBQAAAAFpAAAACHBheW1lbnRzAAAAAAAAAAAAAAAAB2Fzc2V0SWQFAAAABHVuaXQJAAACAAAAAQIAAAAQdW5leHBlY3RlZCBhc3NldAQAAAADcmVzCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAC3Rlc3RBY3Rpb25zCQAETAAAAAIJAABoAAAAAggJAAGRAAAAAggFAAAAAWkAAAAIcGF5bWVudHMAAAAAAAAAAAAAAAAGYW1vdW50BQAAAAxleGNoYW5nZVJhdGUFAAAAA25pbAkABEwAAAACCQEAAAAPQXR0YWNoZWRQYXltZW50AAAAAgUAAAAEdW5pdAAAAAAAAAAAAQUAAAADbmlsAwkAAAAAAAACBQAAAANyZXMAAAAAAAAAABEFAAAAA25pbAkAAAIAAAABAgAAABJCYWQgcmV0dXJuZWQgdmFsdWUAAAAAbfvo1Q=="
	secondScript = "AAIFAAAAAAAAAAcIAhIDCgEBAAAAAAAAAAEAAAABaQEAAAALdGVzdEFjdGlvbnMAAAABAAAAAWEJAAUUAAAAAgkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADaW50AAAAAAAAAAABBQAAAANuaWwAAAAAAAAAABEAAAAAPWJMug=="

	id = bytes.Repeat([]byte{0}, 32)

	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "int", Value: 1}, Sender: &addressCallablePK},
	}

	expectedAttachedPaymentWrites := []*proto.AttachedPaymentScriptAction{
		{Recipient: recipientCallable, Sender: &addrPK, Amount: 1, Asset: proto.OptionalAsset{}},
	}

	smartState := smartStateDappFromDapp

	thisAddress = addr
	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "test", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedAttachedPaymentWrites, ap)
	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  expectedDataEntryWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}

	assert.Equal(t, expectedActionsResult, sr)

	expectedDiffResult := initWrappedState(smartState(), env).diff
	assetExp := proto.NewOptionalAssetWaves()

	balanceMain := diffBalance{asset: assetExp, regular: 9999}
	balanceSender := diffBalance{asset: assetExp, regular: 0}
	balanceCallable := diffBalance{asset: assetExp, regular: 1}
	expectedDiffResult.balances[balanceDiffKey{addr, assetExp}] = balanceMain
	expectedDiffResult.balances[balanceDiffKey{senderAddress, assetExp}] = balanceSender
	expectedDiffResult.balances[balanceDiffKey{addressCallable, assetExp}] = balanceCallable
	intEntry := proto.IntegerDataEntry{Key: "int", Value: 1}
	expectedDiffResult.dataEntries.diffInteger[integerDataEntryKey{intEntry.Key, addressCallable}] = intEntry

	assert.Equal(t, expectedDiffResult.dataEntries, wrappedSt.diff.dataEntries)
	assert.Equal(t, expectedDiffResult.balances, wrappedSt.diff.balances)

	tearDownDappFromDapp()

}

func TestInvokeDAppFromDAppSmartAssetValidation(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}
	let assetId = base58'13YvHUb3bg7sXgExc6kFcCUKm6WYpJX9rLpHVhiyJNGJ'

	@Callable(i)
	func test() = {
		let res = Invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "call",nil, nil)
		if res == 17
		 	then
	    		nil
		else
		    throw("Bad returned value")
	}
	*/

	/* script 2

	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let assetId = base58'13YvHUb3bg7sXgExc6kFcCUKm6WYpJX9rLpHVhiyJNGJ'

	@Callable(i)
	func call() = {
	  ([
	    Reissue(assetId, 100, false),
	    Burn(assetId, 50),
	    ScriptTransfer(i.caller, 1, assetId)
	  ], 17)
	}
	*/
	/* smart asset script
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE EXPRESSION #-}
	{-# SCRIPT_TYPE ASSET #-}

	let dAppAddress = addressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	match tx {
	    case tx: BurnTransaction =>
	        (tx.sender == dAppAddress)
	    case tx: ReissueTransaction =>
	        (tx.sender == dAppAddress)
	    case tx: SetAssetScriptTransaction =>
	        (tx.sender == dAppAddress)
	    case tx: MassTransferTransaction =>
	        (tx.sender == dAppAddress)
	    case tx: TransferTransaction =>
	        (tx.sender == dAppAddress)
	    case _ =>
	        false
	}
	*/

	assetCat, err := proto.NewOptionalAssetFromString("13YvHUb3bg7sXgExc6kFcCUKm6WYpJX9rLpHVhiyJNGJ")
	require.NoError(t, err)

	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)

	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        nil,
		FeeAsset:        proto.OptionalAsset{},
		Fee:             900000,
		Timestamp:       1564703444249,
	}
	inv, _ = invocationToObject(5, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAQAAAAAHYXNzZXRJZAEAAAAgAKdAj/iZUd+zAOo6DTO6IvheatMyaLi7as+C+lV4EDMAAAABAAAAAWkBAAAABHRlc3QAAAAABAAAAANyZXMJAAP8AAAABAkBAAAAB0FkZHJlc3MAAAABAQAAABoBV0myKgvnUpvnQwgi/Cmpjg8vaC8j0MoKywIAAAAEY2FsbAUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAA3JlcwAAAAAAAAAAEQUAAAADbmlsCQAAAgAAAAECAAAAEkJhZCByZXR1cm5lZCB2YWx1ZQAAAAAnQdRv"
	secondScript = "AAIFAAAAAAAAAAQIAhIAAAAAAQAAAAAHYXNzZXRJZAEAAAAgAKdAj/iZUd+zAOo6DTO6IvheatMyaLi7as+C+lV4EDMAAAABAAAAAWkBAAAABGNhbGwAAAAACQAFFAAAAAIJAARMAAAAAgkBAAAAB1JlaXNzdWUAAAADBQAAAAdhc3NldElkAAAAAAAAAABkBwkABEwAAAACCQEAAAAEQnVybgAAAAIFAAAAB2Fzc2V0SWQAAAAAAAAAADIJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyAAAAAAAAAAABBQAAAAdhc3NldElkBQAAAANuaWwAAAAAAAAAABEAAAAA4X8UHg=="

	id = bytes.Repeat([]byte{0}, 32)

	expectedReissueWrites := []*proto.ReissueScriptAction{
		{Sender: &addressCallablePK, Quantity: 100, Reissuable: false, AssetID: assetCat.ID},
	}
	expectedBurnWrites := []*proto.BurnScriptAction{
		{Sender: &addressCallablePK, Quantity: 50, AssetID: assetCat.ID},
	}

	expectedTransferWrites := []*proto.TransferScriptAction{
		{Sender: &addressCallablePK, Recipient: recipient, Amount: 1, Asset: *assetCat},
	}

	smartState := smartStateDappFromDapp

	thisAddress = addr
	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(addressCallable, 10000, *assetCat)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "test", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  make([]*proto.DataEntryScriptAction, 0),
		Transfers:    expectedTransferWrites,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     expectedReissueWrites,
		Burns:        expectedBurnWrites,
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}

	assert.Equal(t, expectedActionsResult, sr)

	expectedDiffResult := initWrappedState(smartState(), env).diff
	balance := diffBalance{regular: 1, leaseIn: 0, asset: *assetCat}
	expectedDiffResult.balances[balanceDiffKey{addr, *assetCat}] = balance

	balanceCallable := diffBalance{regular: 10049, leaseOut: 0, asset: *assetCat} // the balance was 9999. reissue + 100. burn - 50. = 10049
	expectedDiffResult.balances[balanceDiffKey{addressCallable, *assetCat}] = balanceCallable

	oldAsset := diffOldAssetInfo{diffQuantity: 50}
	expectedDiffResult.oldAssetsInfo[assetIDIssue] = oldAsset

	assert.Equal(t, expectedDiffResult.balances, wrappedSt.diff.balances)
	assert.Equal(t, expectedDiffResult.sponsorships, wrappedSt.diff.sponsorships)
	assert.Equal(t, expectedDiffResult.leases, wrappedSt.diff.leases)

	tearDownDappFromDapp()
}

func TestMixedReentrantInvokeAndInvoke(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-#SCRIPT_TYPE ACCOUNT#-}

	 @Callable(i)
	 func back() = {
	   [IntegerEntry("key", 0), ScriptTransfer(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), 2, unit)]
	 }

	 @Callable(i)
	 func foo() = {
	    let r = reentrantInvoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "bar", [this.bytes], [AttachedPayment(unit, 17)])
	    if r == 17
	    then
	       [
	        IntegerEntry("key", 1)
	       ]
	  else
	   throw("Imposible")
	 }
	*/

	/*
			script2
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-#SCRIPT_TYPE ACCOUNT#-}

		 @Callable(i)
		 func bar(a: ByteVector) = {
		   let r = Invoke(Address(a), "back", [], [])
		   if r == r
		   then
		    ([IntegerEntry("bar", 1), ScriptTransfer(Address(a), 3, unit)], 17)
		   else
		    throw("Imposible")
		 }


	*/
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)

	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	recipientCallable := proto.NewRecipientFromAddress(addressCallable)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}
	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAYIAhIAEgAAAAAAAAAAAgAAAAFpAQAAAARiYWNrAAAAAAkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADa2V5AAAAAAAAAAAACQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssAAAAAAAAAAAIFAAAABHVuaXQFAAAAA25pbAAAAAFpAQAAAANmb28AAAAABAAAAAFyCQAD/QAAAAQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAA2JhcgkABEwAAAACCAUAAAAEdGhpcwAAAAVieXRlcwUAAAADbmlsCQAETAAAAAIJAQAAAA9BdHRhY2hlZFBheW1lbnQAAAACBQAAAAR1bml0AAAAAAAAAAARBQAAAANuaWwDCQAAAAAAAAIFAAAAAXIAAAAAAAAAABEJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAICAAAAA2tleQAAAAAAAAAAAQUAAAADbmlsCQAAAgAAAAECAAAACUltcG9zaWJsZQAAAAB70C6c"
	secondScript = "AAIFAAAAAAAAAAcIAhIDCgECAAAAAAAAAAEAAAABaQEAAAADYmFyAAAAAQAAAAFhBAAAAAFyCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQUAAAABYQIAAAAEYmFjawUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAAXIFAAAAAXIJAAUUAAAAAgkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADYmFyAAAAAAAAAAABCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMJAQAAAAdBZGRyZXNzAAAAAQUAAAABYQAAAAAAAAAAAwUAAAAEdW5pdAUAAAADbmlsAAAAAAAAAAARCQAAAgAAAAECAAAACUltcG9zaWJsZQAAAACf+Ofn"

	id = bytes.Repeat([]byte{0}, 32)

	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "key", Value: 0}, Sender: &addrPK},
		{Entry: &proto.IntegerDataEntry{Key: "bar", Value: 1}, Sender: &addressCallablePK},
		{Entry: &proto.IntegerDataEntry{Key: "key", Value: 1}},
	}

	expectedTransferWrites := []*proto.TransferScriptAction{
		{Sender: &addrPK, Recipient: recipientCallable, Amount: 2, Asset: proto.OptionalAsset{}},
		{Sender: &addressCallablePK, Recipient: recipient, Amount: 3, Asset: proto.OptionalAsset{}},
	}

	expectedAttachedPaymentActions := []*proto.AttachedPaymentScriptAction{
		{Sender: &addrPK, Recipient: recipientCallable, Amount: 17, Asset: proto.OptionalAsset{}},
	}

	smartState := smartStateDappFromDapp

	thisAddress = addr

	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "foo", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedAttachedPaymentActions, ap)
	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  expectedDataEntryWrites,
		Transfers:    expectedTransferWrites,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedActionsResult, sr)

	expectedDiffResult := initWrappedState(smartState(), env).diff
	assetExp := proto.NewOptionalAssetWaves()

	balanceMain := diffBalance{asset: assetExp, regular: 9984}
	balanceSender := diffBalance{asset: assetExp, regular: 0}
	balanceCallable := diffBalance{asset: assetExp, regular: 16}
	intEntry1 := proto.IntegerDataEntry{Key: "key", Value: 0}
	intEntry2 := proto.IntegerDataEntry{Key: "bar", Value: 1}
	expectedDiffResult.dataEntries.diffInteger[integerDataEntryKey{intEntry1.Key, addr}] = intEntry1
	expectedDiffResult.dataEntries.diffInteger[integerDataEntryKey{intEntry2.Key, addressCallable}] = intEntry2
	expectedDiffResult.balances[balanceDiffKey{addr, assetExp}] = balanceMain
	expectedDiffResult.balances[balanceDiffKey{senderAddress, assetExp}] = balanceSender
	expectedDiffResult.balances[balanceDiffKey{addressCallable, assetExp}] = balanceCallable

	assert.Equal(t, expectedDiffResult.dataEntries, wrappedSt.diff.dataEntries)
	assert.Equal(t, expectedDiffResult.balances, wrappedSt.diff.balances)

	tearDownDappFromDapp()
}

func TestExpressionScriptFailInvoke(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE EXPRESSION #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let dapp = Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna')

	match tx {
	    case t: InvokeScriptTransaction => {
	        let result = match invoke(dapp, "foo", [5], [AttachedPayment(unit, 10)]) {
	            case i: Int => i
	            case _ => throw("Wrong result type")
	        }
	        if result == 5 then true else throw("Wrong result '" + result.toString() + "'")
	    }
	    case _ => throw("Wrong tx type")
	}
	*/

	/* script 2
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(inv)
	func foo(amount: Int) = ([
	    IntegerEntry("result", amount),
	    ScriptTransfer(inv.caller, amount, unit)
	], amount)
	*/

	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	require.NoError(t, err)
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)
	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}

	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "BQQAAAAEZGFwcAkBAAAAB0FkZHJlc3MAAAABAQAAABoBV0myKgvnUpvnQwgi/Cmpjg8vaC8j0MoKywQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAF0ludm9rZVNjcmlwdFRyYW5zYWN0aW9uBAAAAAF0BQAAAAckbWF0Y2gwBAAAAAZyZXN1bHQEAAAAByRtYXRjaDEJAAP8AAAABAUAAAAEZGFwcAIAAAADZm9vCQAETAAAAAIAAAAAAAAAAAUFAAAAA25pbAkABEwAAAACCQEAAAAPQXR0YWNoZWRQYXltZW50AAAAAgUAAAAEdW5pdAAAAAAAAAAACgUAAAADbmlsAwkAAAEAAAACBQAAAAckbWF0Y2gxAgAAAANJbnQEAAAAAWkFAAAAByRtYXRjaDEFAAAAAWkJAAACAAAAAQIAAAARV3JvbmcgcmVzdWx0IHR5cGUDCQAAAAAAAAIFAAAABnJlc3VsdAAAAAAAAAAABQYJAAACAAAAAQkAASwAAAACCQABLAAAAAICAAAADldyb25nIHJlc3VsdCAnCQABpAAAAAEFAAAABnJlc3VsdAIAAAABJwkAAAIAAAABAgAAAA1Xcm9uZyB0eCB0eXBlUP0hpw=="
	secondScript = "AAIFAAAAAAAAAAcIAhIDCgEBAAAAAAAAAAEAAAADaW52AQAAAANmb28AAAABAAAABmFtb3VudAkABRQAAAACCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAAZyZXN1bHQFAAAABmFtb3VudAkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAADaW52AAAABmNhbGxlcgUAAAAGYW1vdW50BQAAAAR1bml0BQAAAANuaWwFAAAABmFtb3VudAAAAAD070Yd"

	id = bytes.Repeat([]byte{0}, 32)

	smartState := smartStateDappFromDapp

	thisAddress = addr
	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	_, err = CallVerifier(env, tree)
	require.Error(t, err)
	tearDownDappFromDapp()
}

func TestPaymentsDifferentScriptVersion4(t *testing.T) {
	/* script 1
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		let exchangeRate = 5

		@Callable(i)
		func test() = if ((i.payments[0].assetId != unit))
		    then throw("unexpected asset")
		    else {
				let res = invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "testActions",[(i.payments[0].amount * exchangeRate)], [AttachedPayment(unit, 1)])
				if res == res
		 	    then
	                nil
		    	else
		      		throw("Bad returned value")
		}
	*/

	/* script 2
	{-# STDLIB_VERSION 4 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func testActions(a: Int) = {
	  [
	    IntegerEntry("int", 1)
	  ]
	}
	*/

	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	require.NoError(t, err)
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)
	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}

	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAQAAAAAMZXhjaGFuZ2VSYXRlAAAAAAAAAAAFAAAAAQAAAAFpAQAAAAR0ZXN0AAAAAAMJAQAAAAIhPQAAAAIICQABkQAAAAIIBQAAAAFpAAAACHBheW1lbnRzAAAAAAAAAAAAAAAAB2Fzc2V0SWQFAAAABHVuaXQJAAACAAAAAQIAAAAQdW5leHBlY3RlZCBhc3NldAQAAAADcmVzCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAC3Rlc3RBY3Rpb25zCQAETAAAAAIJAABoAAAAAggJAAGRAAAAAggFAAAAAWkAAAAIcGF5bWVudHMAAAAAAAAAAAAAAAAGYW1vdW50BQAAAAxleGNoYW5nZVJhdGUFAAAAA25pbAkABEwAAAACCQEAAAAPQXR0YWNoZWRQYXltZW50AAAAAgUAAAAEdW5pdAAAAAAAAAAAAQUAAAADbmlsAwkAAAAAAAACBQAAAANyZXMFAAAAA3JlcwUAAAADbmlsCQAAAgAAAAECAAAAEkJhZCByZXR1cm5lZCB2YWx1ZQAAAAAdCpXM"
	secondScript = "AAIEAAAAAAAAAAcIAhIDCgEBAAAAAAAAAAEAAAABaQEAAAALdGVzdEFjdGlvbnMAAAABAAAAAWEJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAICAAAAA2ludAAAAAAAAAAAAQUAAAADbmlsAAAAAM41XKE="

	id = bytes.Repeat([]byte{0}, 32)

	smartState := smartStateDappFromDapp

	thisAddress = addr
	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "test", proto.Arguments{})
	require.Nil(t, res)
	require.Error(t, err)

	tearDownDappFromDapp()
}

func TestPaymentsDifferentScriptVersion3(t *testing.T) {
	/* script 1
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		let exchangeRate = 5

		@Callable(i)
		func test() = if ((i.payments[0].assetId != unit))
		    then throw("unexpected asset")
		    else {
				let res = invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "testActions",[(i.payments[0].amount * exchangeRate)], [AttachedPayment(unit, 1)])
				if res == res
		 	    then
	                nil
		    	else
		      		throw("Bad returned value")
		}
	*/

	/* script 2
	{-# STDLIB_VERSION 3 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func testActions(a: Int) = {

	  WriteSet(
	    [DataEntry("int", 1)]
	  )

	}
	*/

	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	require.NoError(t, err)
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)
	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}

	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAQAAAAAMZXhjaGFuZ2VSYXRlAAAAAAAAAAAFAAAAAQAAAAFpAQAAAAR0ZXN0AAAAAAMJAQAAAAIhPQAAAAIICQABkQAAAAIIBQAAAAFpAAAACHBheW1lbnRzAAAAAAAAAAAAAAAAB2Fzc2V0SWQFAAAABHVuaXQJAAACAAAAAQIAAAAQdW5leHBlY3RlZCBhc3NldAQAAAADcmVzCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAC3Rlc3RBY3Rpb25zCQAETAAAAAIJAABoAAAAAggJAAGRAAAAAggFAAAAAWkAAAAIcGF5bWVudHMAAAAAAAAAAAAAAAAGYW1vdW50BQAAAAxleGNoYW5nZVJhdGUFAAAAA25pbAkABEwAAAACCQEAAAAPQXR0YWNoZWRQYXltZW50AAAAAgUAAAAEdW5pdAAAAAAAAAAAAQUAAAADbmlsAwkAAAAAAAACBQAAAANyZXMFAAAAA3JlcwUAAAADbmlsCQAAAgAAAAECAAAAEkJhZCByZXR1cm5lZCB2YWx1ZQAAAAAdCpXM"
	secondScript = "AAIDAAAAAAAAAAcIARIDCgEBAAAAAAAAAAEAAAABaQEAAAALdGVzdEFjdGlvbnMAAAABAAAAAWEJAQAAAAhXcml0ZVNldAAAAAEJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAAA2ludAAAAAAAAAAAAQUAAAADbmlsAAAAAJvCz7w="

	id = bytes.Repeat([]byte{0}, 32)

	smartState := smartStateDappFromDapp

	thisAddress = addr
	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "test", proto.Arguments{})
	require.Nil(t, res)
	require.Error(t, err)

	tearDownDappFromDapp()
}

func TestActionsLimitInOneInvoke(t *testing.T) {

	/* script 1
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func bar() = {
		let res = invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "foo",[], [AttachedPayment(unit, 500)])
		if res == 17
	        then
	        [

	        ]
	        else
	         throw("Bad returned value")
	}
	*/

	/* script 2.1
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func foo() = {
		([
	    ScriptTransfer(i.caller, 1, unit),
	    ScriptTransfer(i.caller, 2, unit),
	    ScriptTransfer(i.caller, 3, unit),
	    ScriptTransfer(i.caller, 4, unit),
	    ScriptTransfer(i.caller, 5, unit),
	    ScriptTransfer(i.caller, 6, unit),
	    ScriptTransfer(i.caller, 7, unit),
	    ScriptTransfer(i.caller, 8, unit),
	    ScriptTransfer(i.caller, 9, unit),
	    ScriptTransfer(i.caller, 10, unit),
	    ScriptTransfer(i.caller, 11, unit),
	    ScriptTransfer(i.caller, 12, unit),
	    ScriptTransfer(i.caller, 13, unit),
	    ScriptTransfer(i.caller, 14, unit),
	    ScriptTransfer(i.caller, 15, unit),
	    ScriptTransfer(i.caller, 16, unit),
	    ScriptTransfer(i.caller, 17, unit),
	    ScriptTransfer(i.caller, 18, unit),
	    ScriptTransfer(i.caller, 19, unit),
	    ScriptTransfer(i.caller, 20, unit),
	    ScriptTransfer(i.caller, 21, unit),
	    ScriptTransfer(i.caller, 22, unit),
	    ScriptTransfer(i.caller, 23, unit),
	    ScriptTransfer(i.caller, 24, unit),
	    ScriptTransfer(i.caller, 25, unit),
	    ScriptTransfer(i.caller, 26, unit),
	    ScriptTransfer(i.caller, 27, unit),
	    ScriptTransfer(i.caller, 28, unit),
	    ScriptTransfer(i.caller, 29, unit),
	    ScriptTransfer(i.caller, 30, unit)
		], 17)
		}
	*/

	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)

	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}
	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAADYmFyAAAAAAQAAAADcmVzCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAA2ZvbwUAAAADbmlsCQAETAAAAAIJAQAAAA9BdHRhY2hlZFBheW1lbnQAAAACBQAAAAR1bml0AAAAAAAAAAH0BQAAAANuaWwDCQAAAAAAAAIFAAAAA3JlcwAAAAAAAAAAEQUAAAADbmlsCQAAAgAAAAECAAAAEkJhZCByZXR1cm5lZCB2YWx1ZQAAAACzb0+i"
	secondScript = "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"

	smartState := smartStateDappFromDapp

	thisAddress = addr
	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "bar", proto.Arguments{})

	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	_, _, err = proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)

	/* script 2.2
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func foo() = {
		([
	    ScriptTransfer(i.caller, 1, unit),
	    ScriptTransfer(i.caller, 2, unit),
	    ScriptTransfer(i.caller, 3, unit),
	    ScriptTransfer(i.caller, 4, unit),
	    ScriptTransfer(i.caller, 5, unit),
	    ScriptTransfer(i.caller, 6, unit),
	    ScriptTransfer(i.caller, 7, unit),
	    ScriptTransfer(i.caller, 8, unit),
	    ScriptTransfer(i.caller, 9, unit),
	    ScriptTransfer(i.caller, 10, unit),
	    ScriptTransfer(i.caller, 11, unit),
	    ScriptTransfer(i.caller, 12, unit),
	    ScriptTransfer(i.caller, 13, unit),
	    ScriptTransfer(i.caller, 14, unit),
	    ScriptTransfer(i.caller, 15, unit),
	    ScriptTransfer(i.caller, 16, unit),
	    ScriptTransfer(i.caller, 17, unit),
	    ScriptTransfer(i.caller, 18, unit),
	    ScriptTransfer(i.caller, 19, unit),
	    ScriptTransfer(i.caller, 20, unit),
	    ScriptTransfer(i.caller, 21, unit),
	    ScriptTransfer(i.caller, 22, unit),
	    ScriptTransfer(i.caller, 23, unit),
	    ScriptTransfer(i.caller, 24, unit),
	    ScriptTransfer(i.caller, 25, unit),
	    ScriptTransfer(i.caller, 26, unit),
	    ScriptTransfer(i.caller, 27, unit),
	    ScriptTransfer(i.caller, 28, unit),
	    ScriptTransfer(i.caller, 29, unit),
	    ScriptTransfer(i.caller, 30, unit)
		ScriptTransfer(i.caller, 31, unit)
		], 17)
		}
	*/
	secondScript = "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"

	res, err = CallFunction(env, tree, "bar", proto.Arguments{})
	require.Nil(t, res)
	require.Error(t, err)

	tearDownDappFromDapp()
}

func TestActionsLimitInvoke(t *testing.T) {

	/* script 1
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func bar() = {
			let res = invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "foo",[], [AttachedPayment(base58'', 500)])
			if res == 17
		    then
	        let res1 = invoke(Address(base58'3P8eZVKS7a4troGckytxaefLAi9w7P5aMna'), "foo",[], [AttachedPayment(base58'', 500)])
	        if res1 == 17
	          then
	            [
	            ]
	        else
	          throw("impossible")

		  else
		      throw("Bad returned value")
		}
	*/

	/* script 2
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func foo() = {
		([
	    ScriptTransfer(i.caller, 1, unit),
	    ScriptTransfer(i.caller, 2, unit),
	    ScriptTransfer(i.caller, 3, unit),
	    ScriptTransfer(i.caller, 4, unit),
	    ScriptTransfer(i.caller, 5, unit),
	    ScriptTransfer(i.caller, 6, unit),
	    ScriptTransfer(i.caller, 7, unit),
	    ScriptTransfer(i.caller, 8, unit),
	    ScriptTransfer(i.caller, 9, unit),
	    ScriptTransfer(i.caller, 10, unit),
	    ScriptTransfer(i.caller, 11, unit),
	    ScriptTransfer(i.caller, 12, unit),
	    ScriptTransfer(i.caller, 13, unit),
	    ScriptTransfer(i.caller, 14, unit),
	    ScriptTransfer(i.caller, 15, unit),
	    ScriptTransfer(i.caller, 16, unit)
		], 17)
		}
	*/

	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, sender)
	require.NoError(t, err)

	addr, err = proto.NewAddressFromString("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)
	addrPK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addr)
	require.NoError(t, err)

	addressCallable, err = proto.NewAddressFromString("3P8eZVKS7a4troGckytxaefLAi9w7P5aMna")
	require.NoError(t, err)
	addressCallablePK, err = smartStateDappFromDapp().NewestScriptPKByAddr(addressCallable)
	require.NoError(t, err)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})

	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx = &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 10000,
			Asset:  proto.OptionalAsset{},
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       900000,
		Timestamp: 1564703444249,
	}
	inv, _ = invocationToObject(4, proto.MainNetScheme, tx)

	firstScript = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAADYmFyAAAAAAQAAAADcmVzCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAA2ZvbwUAAAADbmlsCQAETAAAAAIJAQAAAA9BdHRhY2hlZFBheW1lbnQAAAACAQAAAAAAAAAAAAAAAfQFAAAAA25pbAMJAAAAAAAAAgUAAAADcmVzAAAAAAAAAAARBAAAAARyZXMxCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVdJsioL51Kb50MIIvwpqY4PL2gvI9DKCssCAAAAA2ZvbwUAAAADbmlsCQAETAAAAAIJAQAAAA9BdHRhY2hlZFBheW1lbnQAAAACAQAAAAAAAAAAAAAAAfQFAAAAA25pbAMJAAAAAAAAAgUAAAAEcmVzMQAAAAAAAAAAEQUAAAADbmlsCQAAAgAAAAECAAAACmltcG9zc2libGUJAAACAAAAAQIAAAASQmFkIHJldHVybmVkIHZhbHVlAAAAAKPxAAQ="
	secondScript = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAADZm9vAAAAAAkABRQAAAACCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAAAAAAAQUAAAAEdW5pdAkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAAAAAAAIFAAAABHVuaXQJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyAAAAAAAAAAADBQAAAAR1bml0CQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAAAAAABAUAAAAEdW5pdAkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAAAAAAAUFAAAABHVuaXQJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyAAAAAAAAAAAGBQAAAAR1bml0CQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAAAAAABwUAAAAEdW5pdAkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAAAAAAAgFAAAABHVuaXQJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyAAAAAAAAAAAJBQAAAAR1bml0CQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAAAAAACgUAAAAEdW5pdAkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAAAAAAAsFAAAABHVuaXQJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyAAAAAAAAAAAMBQAAAAR1bml0CQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAAAAAADQUAAAAEdW5pdAkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAAAAAAA4FAAAABHVuaXQJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyAAAAAAAAAAAPBQAAAAR1bml0CQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAAAAAAEAUAAAAEdW5pdAUAAAADbmlsAAAAAAAAAAARAAAAAEnsJR0="

	smartState := smartStateDappFromDapp

	thisAddress = addr
	env := envDappFromDapp

	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	err = AddAssetToSender(senderAddress, 10000, proto.OptionalAsset{})
	require.NoError(t, err)
	err = AddExternalPayments(tx.Payments, tx.SenderPK)
	require.NoError(t, err)

	src, err := base64.StdEncoding.DecodeString(firstScript)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "bar", proto.Arguments{})
	require.Nil(t, res)
	require.Error(t, err)

	tearDownDappFromDapp()
}

func TestHashScriptFunc(t *testing.T) {
	/*
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-#SCRIPT_TYPE ACCOUNT#-}

		@Callable(i)
		func foo() = {
			let h = hashScriptAtAddress(this)
			if hashScriptAtAddress(i.caller) == unit
			then
				[ BinaryEntry("hash", h.value()) ]
			else
				throw("Unexpected script was found.")
		}
	*/
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	senderPK, err := crypto.NewPublicKeyFromBase58("2v89gsAztdyVq8aEVdNrxUZKtf1HfTAn5umC41idvykp")
	require.NoError(t, err)
	senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, senderPK)
	require.NoError(t, err)
	addrPK, err := crypto.NewPublicKeyFromBase58("2zb2orX2g58YZgXAvdn5ojTuPP8vAU2rsqYQ5L6KCXqz")
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, addrPK)
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        senderPK,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        nil,
		FeeAsset:        proto.OptionalAsset{},
		Fee:             900000,
		Timestamp:       1564703444249,
	}
	inv, _ := invocationToObject(5, proto.MainNetScheme, tx)

	var script string
	var wrappedSt WrappedState

	smartState := func() types.SmartState {
		return &MockSmartState{
			GetByteTreeFunc: func(recipient proto.Recipient) (proto.Script, error) {
				var resScript proto.Script
				var err error
				if recipient.Address.String() == addr.String() {
					resScript, err = base64.StdEncoding.DecodeString(script)
				}
				if recipient.Address.String() == senderAddr.String() {
					return proto.Script{}, nil
				}
				if err != nil {
					return proto.Script{}, err
				}

				return resScript, nil
			},
		}
	}
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		heightFunc: func() rideInt {
			return 368430
		},
		thisFunc: func() rideType {
			return rideAddress(addr)
		},
		invocationFunc: func() rideObject {
			return inv
		},
		timestampFunc: func() uint64 {
			return 1564703444249
		},
		transactionFunc: func() rideObject {
			obj, _ := transactionToObject(proto.MainNetScheme, tx)
			return obj
		},
		stateFunc: func() types.SmartState {
			return &wrappedSt
		},
		rideV6ActivatedFunc: noRideV6,
	}
	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	script = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAADZm9vAAAAAAQAAAABaAkAA/EAAAABBQAAAAR0aGlzAwkAAAAAAAACCQAD8QAAAAEIBQAAAAFpAAAABmNhbGxlcgUAAAAEdW5pdAkABEwAAAACCQEAAAALQmluYXJ5RW50cnkAAAACAgAAAARoYXNoCQEAAAAFdmFsdWUAAAABBQAAAAFoBQAAAANuaWwJAAACAAAAAQIAAAAcVW5leHBlY3RlZCBzY3JpcHQgd2FzIGZvdW5kLgAAAABGhMi8"
	src, err := base64.StdEncoding.DecodeString(script)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "foo", proto.Arguments{})
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	decodedScript, err := base64.StdEncoding.DecodeString(script)
	require.NoError(t, err)
	hash, err := crypto.FastHash(decodedScript)
	require.NoError(t, err)

	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.BinaryDataEntry{Key: "hash", Value: hash.Bytes()}},
	}

	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  expectedDataEntryWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}

	assert.Equal(t, expectedActionsResult, sr)
}

func TestDataStorageUntouchedFunc(t *testing.T) {
	/*
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-#SCRIPT_TYPE ACCOUNT#-}
		@Callable(i)
		func foo() = {
			let check = isDataStorageUntouched(this)
				[ BooleanEntry("virgin", check) ]
		}
	*/
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	senderPK, err := crypto.NewPublicKeyFromBase58("2v89gsAztdyVq8aEVdNrxUZKtf1HfTAn5umC41idvykp")
	require.NoError(t, err)

	addrPK, err := crypto.NewPublicKeyFromBase58("2zb2orX2g58YZgXAvdn5ojTuPP8vAU2rsqYQ5L6KCXqz")
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, addrPK)
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(addr)

	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        senderPK,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        nil,
		FeeAsset:        proto.OptionalAsset{},
		Fee:             900000,
		Timestamp:       1564703444249,
	}
	//inv, _ := invocationToObject(5, proto.MainNetScheme, tx)

	var script string
	var wrappedSt WrappedState

	smartState := func() types.SmartState {
		return &MockSmartState{
			IsStateUntouchedFunc: func(recipient proto.Recipient) (bool, error) {
				if recipient.Address.String() == addr.String() {
					return false, nil
				} else {
					return false, errors.New("unexpected address")
				}
			},
		}
	}
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		heightFunc: func() rideInt {
			return 368430
		},
		thisFunc: func() rideType {
			return rideAddress(addr)
		},
		timestampFunc: func() uint64 {
			return 1564703444249
		},
		transactionFunc: func() rideObject {
			obj, _ := transactionToObject(proto.MainNetScheme, tx)
			return obj
		},
		stateFunc: func() types.SmartState {
			return &wrappedSt
		},
		rideV6ActivatedFunc: noRideV6,
	}
	NewWrappedSt := initWrappedState(smartState(), env)
	wrappedSt = *NewWrappedSt

	script = "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAADZm9vAAAAAAQAAAAFY2hlY2sJAAQeAAAAAQUAAAAEdGhpcwkABEwAAAACCQEAAAAMQm9vbGVhbkVudHJ5AAAAAgIAAAAGdmlyZ2luBQAAAAVjaGVjawUAAAADbmlsAAAAAA8AdTc="
	src, err := base64.StdEncoding.DecodeString(script)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "foo", proto.Arguments{})
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.BooleanDataEntry{Key: "virgin", Value: false}},
	}

	expectedActionsResult := &proto.ScriptResult{
		DataEntries:  expectedDataEntryWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}

	assert.Equal(t, expectedActionsResult, sr)
}

func TestMatchOverwrite(t *testing.T) {
	/*
		{-# STDLIB_VERSION 1 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		match tx {
		    case dt: DataTransaction =>
		    let a = extract(getInteger(dt.sender, "a"))
		    let x = if a == 0 then {
		        match getInteger(dt.sender, "x") {
		            case i: Int => i
		            case _ => 0
		        }
		    } else {
		        0
		    }
		    let xx = match getInteger(dt.data, "x") {
		        case i: Int => i
		        case _ => 0
		    }
		    x + xx == 3
		    case _ => false
		}
	*/
	pk := crypto.PublicKey{}
	sig := crypto.Signature{}
	tx := proto.NewUnsignedData(1, pk, 1400000, 1539113093702)
	tx.Entries = append(tx.Entries, &proto.IntegerDataEntry{Key: "x", Value: 2})
	tx.ID = &crypto.Digest{}
	tx.Proofs = proto.NewProofs()
	tx.Proofs.Proofs = append(tx.Proofs.Proofs, sig[:])

	tv, err := transactionToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)

	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		heightFunc: func() rideInt {
			return 368430
		},
		transactionFunc: func() rideObject {
			return tv
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					switch key {
					case "a":
						return &proto.IntegerDataEntry{Key: "a", Value: 0}, nil
					case "x":
						return &proto.IntegerDataEntry{Key: "x", Value: 1}, nil
					default:
						return nil, errors.New("fail")
					}
				},
			}
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAACZHQFAAAAByRtYXRjaDAEAAAAAWEJAQAAAAdleHRyYWN0AAAAAQkABBoAAAACCAUAAAACZHQAAAAGc2VuZGVyAgAAAAFhBAAAAAF4AwkAAAAAAAACBQAAAAFhAAAAAAAAAAAABAAAAAckbWF0Y2gxCQAEGgAAAAIIBQAAAAJkdAAAAAZzZW5kZXICAAAAAXgDCQAAAQAAAAIFAAAAByRtYXRjaDECAAAAA0ludAQAAAABaQUAAAAHJG1hdGNoMQUAAAABaQAAAAAAAAAAAAAAAAAAAAAAAAQAAAACeHgEAAAAByRtYXRjaDEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAAAXgDCQAAAQAAAAIFAAAAByRtYXRjaDECAAAAA0ludAQAAAABaQUAAAAHJG1hdGNoMQUAAAABaQAAAAAAAAAAAAkAAAAAAAACCQAAZAAAAAIFAAAAAXgFAAAAAnh4AAAAAAAAAAADB2NbtyA="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallVerifier(env, tree)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	assert.True(t, r.Result())
}

func TestFailSript1(t *testing.T) {
	pk, err := crypto.NewPublicKeyFromBase58("5ydncg624xM6LmJKWJ26iZoy7XBdGx9JxcgqKMNhJPaz")
	require.NoError(t, err)
	sig, err := crypto.NewSignatureFromBase58("JR8MP7AFSm5JY5UKYRHtTjJX7sVUEV7rnQaKAvLB7RjV9Ze8Cm1KeYQQiuYBp8gJZrcQqrC6gHiyYheKpheHVgk")
	require.NoError(t, err)
	id, err := crypto.NewDigestFromBase58("Eg5yoFwXcBrq3ik4JvbbhSg429b6HT2qdXTURAUMBTh9")
	require.NoError(t, err)

	tx := proto.NewUnsignedData(1, pk, 1400000, 1539113093702)
	tx.Entries = append(tx.Entries, &proto.IntegerDataEntry{Key: "command", Value: 1})
	tx.Entries = append(tx.Entries, &proto.IntegerDataEntry{Key: "gameState", Value: 1})
	tx.Entries = append(tx.Entries, &proto.StringDataEntry{Key: "player1", Value: ""})
	tx.Entries = append(tx.Entries, &proto.StringDataEntry{Key: "player2", Value: ""})
	tx.ID = &id
	tx.Proofs = proto.NewProofs()
	tx.Proofs.Proofs = append(tx.Proofs.Proofs, sig[:])

	tv, err := transactionToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)

	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		heightFunc: func() rideInt {
			return 368430
		},
		transactionFunc: func() rideObject {
			return tv
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					switch key {
					case "gameState":
						return &proto.IntegerDataEntry{Key: "gameState", Value: 0}, nil
					default:
						return nil, errors.New("fail")
					}
				},
			}
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "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"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallVerifier(env, tree)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	assert.True(t, r.Result())
}

func TestFailSript2(t *testing.T) {
	/* Script:
	{-# STDLIB_VERSION 2 #-}
	{-# CONTENT_TYPE EXPRESSION #-}
	let admin = Address(base58'3PEyLyxu4yGJAEmuVRy3G4FvEBUYV6ykQWF')
	match tx {
	    case tx: MassTransferTransaction|TransferTransaction =>
	        if tx.sender == admin then
	            true
	        else
	            throw("You're not allowed to transfer this asset")
	    case tx: BurnTransaction =>
	        throw("You're not allowed to burn this asset")
	    case tx: ExchangeTransaction =>
	        let amountAsset = match tx.sellOrder.assetPair.amountAsset {
	            case b: ByteVector => b
	            case _ => throw("Incorrect asset pair")
	        }
	        let priceAsset = match tx.sellOrder.assetPair.priceAsset {
	            case b: ByteVector => b
	            case _ => throw("Incorrect asset pair")
	        }
	        let pair1 = toBase58String(amountAsset) + "/" + toBase58String(priceAsset)
	        let pair2 = toBase58String(priceAsset) + "/" + toBase58String(amountAsset)
	        let checkPair1 = match getBoolean(admin, pair1) {
	            case b: Boolean => b
	            case _ => false
	        }
	        let checkPair2 = match getBoolean(admin, pair2) {
	            case b: Boolean => b
	            case _ => false
	        }
	        let status = match getString(admin, "status") {
	            case s: String => s
	            case _ => throw("The contest has not started yet")
	        }
	        if status == "finished" then
	            throw("The contest has already finished")
	        else
	            if status != "started" then
	                throw("The contest has not started yet")
	            else
	                if
	                    if checkPair1 then
	                        true
	                    else
	                        checkPair2
	                then
	                    true
	                else
	                    throw("Incorrect asset pair")
	   case tx: ReissueTransaction|SetAssetScriptTransaction =>
	       true
	   case _ =>
	       false
	}
	*/
	transaction := `{"senderPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy","amount": 100000000,"fee": 1100000,"type": 7,"version": 2,"sellMatcherFee": 1100000,"sender": "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3","feeAssetId": null,"proofs": ["DGxkASjpPaKxu8bAv3PJpF9hJ9KAiLsB7bLBTEZXYcWmmc65pHiq5ymJNAazRM2aoLCeTLXXNda5hR9LZNayB69"],"price": 790000,"id": "5aHKTDvWdVWmo9MPDPoYX83x6hyLJ5ji4eopmoUxELR2","order2": {"version": 2,"id": "CzBrJkpaWz2AHnT3U8baY3eTfRdymuC7dEqiGpas68tD","sender": "3PEjQH31dP2ipvrkouUs12ynKShpBcRQFAT","senderPublicKey": "BVtDAjf1MmUdPW2yRHEBiSP5yy7EnxzKgQWpajQM8FCx","matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy","assetPair": {"amountAsset": "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF","priceAsset": "CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr"},"orderType": "sell","amount": 100000000,"price": 790000,"timestamp": 1557995955609,"expiration": 1560501555609,"matcherFee": 1100000,"signature": "3Aw94WkF4PUeard435jtJTZLESRFMBuxYRYVVf3GrG48aAxLhbvcXdwsrtALLQ3LYbdNdhR1NUUzdMinU8pLiwWc","proofs": ["3Aw94WkF4PUeard435jtJTZLESRFMBuxYRYVVf3GrG48aAxLhbvcXdwsrtALLQ3LYbdNdhR1NUUzdMinU8pLiwWc"]},"order1": {"version": 2,"id": "APLf7qDhU5puSa5h1KChNBobF8VKoy37PcP7BnhoSPvi","sender": "3PEyLyxu4yGJAEmuVRy3G4FvEBUYV6ykQWF","senderPublicKey": "28sBbJ7pHNG4VFrvNN43sNsdWYyrTFVAwd98W892mxBQ","matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy","assetPair": {"amountAsset": "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF","priceAsset": "CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr"},"orderType": "buy","amount": 100000000,"price": 790000,"timestamp": 1557995158094,"expiration": 1560500758093,"matcherFee": 1100000,"signature": "5zUuSSJyv5NU11RPa91fpQaCXR3xvR1ctjQrfxnNREFhMmbXfACzhfFgV18rdvrvm4X3p3iYK3fxS1TXwgSV5m83","proofs": ["5zUuSSJyv5NU11RPa91fpQaCXR3xvR1ctjQrfxnNREFhMmbXfACzhfFgV18rdvrvm4X3p3iYK3fxS1TXwgSV5m83"]},"buyMatcherFee": 1100000,"timestamp": 1557995955923,"height": 1528811}`
	tx := new(proto.ExchangeWithProofs)
	err := json.Unmarshal([]byte(transaction), tx)
	require.NoError(t, err)

	tv, err := transactionToObject(proto.MainNetScheme, tx)
	require.NoError(t, err)

	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		heightFunc: func() rideInt {
			return 368430
		},
		transactionFunc: func() rideObject {
			return tv
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					switch key {
					case "status":
						return &proto.StringDataEntry{Key: "status", Value: "started"}, nil
					default:
						return nil, errors.New("fail")
					}
				},
				RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
					switch key {
					case "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF/CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr":
						return &proto.BooleanDataEntry{Key: "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF/CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr", Value: true}, nil
					default:
						return nil, errors.New("fail")
					}
				},
			}
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "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"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallVerifier(env, tree)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	assert.True(t, r.Result())
}

func TestWhaleDApp(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("36MgbHjSn5L6z6uW9JAx9Pgd8YJcasVaiwe1cRYJhHzf")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5V862rKW36D6f5pZeArLsWFKt8UoGQeN3CanjFAzD9NSNfZjCbdvwp4DEAWWHJpAbqV5NvF68snCJxYQH7YfPDS6")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("5z78SSRPPLJL5MpFZP2XbtEuKCHks3xT7XL1arHx12WU")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3P8Fvy1yDwNHvVrabe4ek5b9dAwxFjDKV7R")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "3P9yVruoCbs4cveU8HpTdFUvzwY59ADaQm3"})
	arguments.Append(&proto.StringArgument{Value: `{"name":"James May","message":"Hello!","isWhale":false,"address":"3P9yVruoCbs4cveU8HpTdFUvzwY59ADaQm3"}`})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "inviteuser",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        nil,
		FeeAsset:        proto.OptionalAsset{},
		Fee:             900000,
		Timestamp:       1564703444249,
	}
	gs := crypto.MustBytesFromBase58("8XgXc3Sh5KyscRs7YwuNy8YrrAS3bX4EeYpqczZf5Sxn")
	gen, err := proto.NewAddressFromString("3P5hx8Lw6nCYgFkQcwHkFZQnwbfF7DfhyyP")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1564703482444,
		Height:              1642207,
		BaseTarget:          80,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender,
	}

	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 368430
		},
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 1642207, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
					switch key {
					case "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF/CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr":
						return &proto.BooleanDataEntry{Key: "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF/CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr", Value: true}, nil
					default:
						return nil, errors.New("fail")
					}
				},
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					switch key {
					case "status":
						return &proto.StringDataEntry{Key: "status", Value: "started"}, nil
					default:
						return nil, errors.New("fail")
					}
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(3, proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "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X2Z1bmRfBQAAAARpdGVtAQAAABZnZXRLZXlJdGVtRnVuZE5lZ2F0aXZlAAAAAQAAAARpdGVtCQABLAAAAAICAAAADm5lZ2F0aXZlX2Z1bmRfBQAAAARpdGVtAQAAABlnZXRLZXlJdGVtQWNjRnVuZFBvc2l0aXZlAAAAAgAAAARpdGVtAAAAB2FjY291bnQJAAEsAAAAAgkAASwAAAACCQEAAAAWZ2V0S2V5SXRlbUZ1bmRQb3NpdGl2ZQAAAAEFAAAABGl0ZW0CAAAAAV8FAAAAB2FjY291bnQBAAAAGWdldEtleUl0ZW1BY2NGdW5kTmVnYXRpdmUAAAACAAAABGl0ZW0AAAAHYWNjb3VudAkAASwAAAACCQABLAAAAAIJAQAAABZnZXRLZXlJdGVtRnVuZE5lZ2F0aXZlAAAAAQUAAAAEaXRlbQIAAAABXwUAAAAHYWNjb3VudAEAAAAXZ2V0S2V5SXRlbUFjY1Jldmlld3NDbnQAAAACAAAABGl0ZW0AAAAHYWNjb3VudAkAASwAAAACCQABLAAAAAIJAAEsAAAAAgIAAAAMcmV2aWV3c19jbnRfBQAAAARpdGVtAgAAAAFfBQAAAAdhY2NvdW50AQAAABNnZXRLZXlJdGVtQWNjUmV2aWV3AAAAAgAAAARpdGVtAAAAB2FjY291bnQJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAAB3Jldmlld18FAAAABGl0ZW0CAAAAAV8FAAAAB2FjY291bnQBAAAAF2dldEtleUl0ZW1BY2NSZXZpZXdUZXh0AAAAAwAAAARpdGVtAAAAB2FjY291bnQAAAADY250CQABLAAAAAIJAAEsAAAAAgkBAAAAE2dldEtleUl0ZW1BY2NSZXZpZXcAAAACBQAAAARpdGVtBQAAAAdhY2NvdW50AgAAAAlfdGV4dF9pZDoFAAAAA2NudAEAAAAXZ2V0S2V5SXRlbUFjY1Jldmlld01vZGUAAAADAAAABGl0ZW0AAAAHYWNjb3VudAAAAANjbnQJAAEsAAAAAgkAASwAAAACCQEAAAATZ2V0S2V5SXRlbUFjY1JldmlldwAAAAIFAAAABGl0ZW0FAAAAB2FjY291bnQCAAAACV9tb2RlX2lkOgUAAAADY250AQAAABdnZXRLZXlJdGVtQWNjUmV2aWV3VGllcgAAAAMAAAAEaXRlbQAAAAdhY2NvdW50AAAAA2NudAkAASwAAAACCQABLAAAAAIJAQAAABNnZXRLZXlJdGVtQWNjUmV2aWV3AAAAAgUAAAAEaXRlbQUAAAAHYWNjb3VudAIAAAAJX3RpZXJfaWQ6BQAAAANjbnQBAAAAG2dldEtleUl0ZW1BY2NWb3RlUmV2aWV3VGV4dAAAAAIAAAAEaXRlbQAAAAdhY2NvdW50CQABLAAAAAIJAQAAABNnZXRLZXlJdGVtQWNjUmV2aWV3AAAAAgUAAAAEaXRlbQUAAAAHYWNjb3VudAIAAAALX3ZvdGVyZXZpZXcBAAAAHGdldEtleUl0ZW1BY2NXaGFsZVJldmlld1RleHQAAAACAAAABGl0ZW0AAAAHYWNjb3VudAkAASwAAAACCQEAAAATZ2V0S2V5SXRlbUFjY1JldmlldwAAAAIFAAAABGl0ZW0FAAAAB2FjY291bnQCAAAADF93aGFsZXJldmlldwEAAAAWZ2V0S2V5SXRlbUJ1eW91dEFtb3VudAAAAAEAAAAEaXRlbQkAASwAAAACAgAAAA5idXlvdXRfYW1vdW50XwUAAAAEaXRlbQEAAAAVZ2V0S2V5SXRlbUFjY1dpbm5pbmdzAAAAAgAAAARpdGVtAAAAB2FjY291bnQJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAACXdpbm5pbmdzXwUAAAAEaXRlbQIAAAABXwUAAAAHYWNjb3VudAEAAAAUZ2V0VmFsdWVXaGl0ZWxpc3RSZWYAAAABAAAAB2FjY291bnQJAQAAAAtnZXRTdHJCeUtleQAAAAEJAQAAABJnZXRLZXlXaGl0ZWxpc3RSZWYAAAABBQAAAAdhY2NvdW50AQAAABdnZXRWYWx1ZVdoaXRlbGlzdFN0YXR1cwAAAAEAAAAHYWNjb3VudAkBAAAAC2dldFN0ckJ5S2V5AAAAAQkBAAAAFWdldEtleVdoaXRlbGlzdFN0YXR1cwAAAAEFAAAAB2FjY291bnQBAAAAD2dldFZhbHVlQmFsYW5jZQAAAAEAAAAHYWNjb3VudAkBAAAADmdldE51bWJlckJ5S2V5AAAAAQkBAAAADWdldEtleUJhbGFuY2UAAAABBQAAAAdhY2NvdW50AQAAABRnZXRWYWx1ZVdoaXRlbGlzdEJpbwAAAAEAAAAHYWNjb3VudAkBAAAAC2dldFN0ckJ5S2V5AAAAAQkBAAAAEmdldEtleVdoaXRlbGlzdEJpbwAAAAEFAAAAB2FjY291bnQBAAAAFmdldFZhbHVlV2hpdGVsaXN0QmxvY2sAAAABAAAAB2FjY291bnQJAQAAAAtnZXRTdHJCeUtleQAAAAEJAQAAABRnZXRLZXlXaGl0ZWxpc3RCbG9jawAAAAEFAAAAB2FjY291bnQBAAAAEmdldFZhbHVlSXRlbUF1dGhvcgAAAAEAAAAEaXRlbQkBAAAAC2dldFN0ckJ5S2V5AAAAAQkBAAAAEGdldEtleUl0ZW1BdXRob3IAAAABBQAAAARpdGVtAQAAABFnZXRWYWx1ZUl0ZW1CbG9jawAAAAEAAAAEaXRlbQkBAAAADmdldE51bWJlckJ5S2V5AAAAAQkBAAAAD2dldEtleUl0ZW1CbG9jawAAAAEFAAAABGl0ZW0BAAAAHGdldFZhbHVlSXRlbVZvdGluZ0V4cGlyYXRpb24AAAABAAAABGl0ZW0JAQAAAA5nZXROdW1iZXJCeUtleQAAAAEJAQAAABpnZXRLZXlJdGVtVm90aW5nRXhwaXJhdGlvbgAAAAEFAAAABGl0ZW0BAAAAEGdldFZhbHVlSXRlbUJhbmsAAAABAAAABGl0ZW0JAQAAAA5nZXROdW1iZXJCeUtleQAAAAEJAQAAAA5nZXRLZXlJdGVtQmFuawAAAAEFAAAABGl0ZW0BAAAAEmdldFZhbHVlSXRlbVN0YXR1cwAAAAEAAAAEaXRlbQkBAAAAC2dldFN0ckJ5S2V5AAAAAQkBAAAAEGdldEtleUl0ZW1TdGF0dXMAAAABBQAAAARpdGVtAQAAABBnZXRWYWx1ZUl0ZW1EYXRhAAAAAQAAAARpdGVtCQEAAAALZ2V0U3RyQnlLZXkAAAABCQEAAAAOZ2V0S2V5SXRlbURhdGEAAAABBQAAAARpdGVtAQAAABtnZXRWYWx1ZUl0ZW1Dcm93ZEV4cGlyYXRpb24AAAABAAAABGl0ZW0JAQAAAA5nZXROdW1iZXJCeUtleQAAAAEJAQAAABlnZXRLZXlJdGVtQ3Jvd2RFeHBpcmF0aW9uAAAAAQUAAAAEaXRlbQEAAAAbZ2V0VmFsdWVJdGVtV2hhbGVFeHBpcmF0aW9uAAAAAQAAAARpdGVtCQEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABCQEAAAAZZ2V0S2V5SXRlbVdoYWxlRXhwaXJhdGlvbgAAAAEFAAAABGl0ZW0BAAAAFGdldFZhbHVlSXRlbU5Db21taXRzAAAAAQAAAARpdGVtCQEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABCQEAAAASZ2V0S2V5SXRlbU5Db21taXRzAAAAAQUAAAAEaXRlbQEAAAAVZ2V0VmFsdWVJdGVtQWNjQ29tbWl0AAAAAgAAAARpdGVtAAAAB2FjY291bnQJAQAAAAtnZXRTdHJCeUtleQAAAAEJAQAAABNnZXRLZXlJdGVtQWNjQ29tbWl0AAAAAgUAAAAEaXRlbQUAAAAHYWNjb3VudAEAAAAVZ2V0VmFsdWVJdGVtQWNjUmV2ZWFsAAAAAgAAAARpdGVtAAAAB2FjY291bnQJAQAAAAtnZXRTdHJCeUtleQAAAAEJAQAAABNnZXRLZXlJdGVtQWNjUmV2ZWFsAAAAAgUAAAAEaXRlbQUAAAAHYWNjb3VudAEAAAAUZ2V0VmFsdWVJdGVtVm90ZXNZZXMAAAABAAAABGl0ZW0JAQAAAA5nZXROdW1iZXJCeUtleQAAAAEJAQAAABJnZXRLZXlJdGVtVm90ZXNZZXMAAAABBQAAAARpdGVtAQAAABNnZXRWYWx1ZUl0ZW1Wb3Rlc05vAAAAAQAAAARpdGVtCQEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABCQEAAAARZ2V0S2V5SXRlbVZvdGVzTm8AAAABBQAAAARpdGVtAQAAABRnZXRWYWx1ZUl0ZW1BY2NGaW5hbAAAAAIAAAAEaXRlbQAAAAdhY2NvdW50CQEAAAALZ2V0U3RyQnlLZXkAAAABCQEAAAASZ2V0S2V5SXRlbUFjY0ZpbmFsAAAAAgUAAAAEaXRlbQUAAAAHYWNjb3VudAEAAAAYZ2V0VmFsdWVJdGVtRnVuZFBvc2l0aXZlAAAAAQAAAARpdGVtCQEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABCQEAAAAWZ2V0S2V5SXRlbUZ1bmRQb3NpdGl2ZQAAAAEFAAAABGl0ZW0BAAAAGGdldFZhbHVlSXRlbUZ1bmROZWdhdGl2ZQAAAAEAAAAEaXRlbQkBAAAADmdldE51bWJlckJ5S2V5AAAAAQkBAAAAFmdldEtleUl0ZW1GdW5kTmVnYXRpdmUAAAABBQAAAARpdGVtAQAAABtnZXRWYWx1ZUl0ZW1BY2NGdW5kUG9zaXRpdmUAAAACAAAABGl0ZW0AAAAHYWNjb3VudAkBAAAADmdldE51bWJlckJ5S2V5AAAAAQkBAAAAGWdldEtleUl0ZW1BY2NGdW5kUG9zaXRpdmUAAAACBQAAAARpdGVtBQAAAAdhY2NvdW50AQAAABtnZXRWYWx1ZUl0ZW1BY2NGdW5kTmVnYXRpdmUAAAACAAAABGl0ZW0AAAAHYWNjb3VudAkBAAAADmdldE51bWJlckJ5S2V5AAAAAQkBAAAAGWdldEtleUl0ZW1BY2NGdW5kTmVnYXRpdmUAAAACBQAAAARpdGVtBQAAAAdhY2NvdW50AQAAABlnZXRWYWx1ZUl0ZW1BY2NSZXZpZXdzQ250AAAAAgAAAARpdGVtAAAAB2FjY291bnQJAQAAAA5nZXROdW1iZXJCeUtleQAAAAEJAQAAABdnZXRLZXlJdGVtQWNjUmV2aWV3c0NudAAAAAIFAAAABGl0ZW0FAAAAB2FjY291bnQBAAAAGWdldFZhbHVlSXRlbUFjY1Jldmlld1RleHQAAAADAAAABGl0ZW0AAAAHYWNjb3VudAAAAANjbnQJAQAAAAtnZXRTdHJCeUtleQAAAAEJAQAAABdnZXRLZXlJdGVtQWNjUmV2aWV3VGV4dAAAAAMFAAAABGl0ZW0FAAAAB2FjY291bnQFAAAAA2NudAEAAAAZZ2V0VmFsdWVJdGVtQWNjUmV2aWV3TW9kZQAAAAMAAAAEaXRlbQAAAAdhY2NvdW50AAAAA2NudAkBAAAAC2dldFN0ckJ5S2V5AAAAAQkBAAAAF2dldEtleUl0ZW1BY2NSZXZpZXdNb2RlAAAAAwUAAAAEaXRlbQUAAAAHYWNjb3VudAUAAAADY250AQAAABlnZXRWYWx1ZUl0ZW1BY2NSZXZpZXdUaWVyAAAAAwAAAARpdGVtAAAAB2FjY291bnQAAAADY250CQEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABCQEAAAAXZ2V0S2V5SXRlbUFjY1Jldmlld1RpZXIAAAADBQAAAARpdGVtBQAAAAdhY2NvdW50BQAAAANjbnQBAAAAGGdldFZhbHVlSXRlbUJ1eW91dEFtb3VudAAAAAEAAAAEaXRlbQkBAAAADmdldE51bWJlckJ5S2V5AAAAAQkBAAAAFmdldEtleUl0ZW1CdXlvdXRBbW91bnQAAAABBQAAAARpdGVtAQAAABdnZXRWYWx1ZUl0ZW1BY2NXaW5uaW5ncwAAAAIAAAAEaXRlbQAAAAdhY2NvdW50CQEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABCQEAAAAVZ2V0S2V5SXRlbUFjY1dpbm5pbmdzAAAAAgUAAAAEaXRlbQUAAAAHYWNjb3VudAAAAAALV0hJVEVMSVNURUQCAAAACnJlZ2lzdGVyZWQAAAAAB0lOVklURUQCAAAAB2ludml0ZWQAAAAABVdIQUxFAgAAAAV3aGFsZQAAAAADTkVXAgAAAANuZXcAAAAABkNPTU1JVAIAAAANdm90aW5nX2NvbW1pdAAAAAAGUkVWRUFMAgAAAA12b3RpbmdfcmV2ZWFsAAAAAAhGRUFUVVJFRAIAAAAIZmVhdHVyZWQAAAAACERFTElTVEVEAgAAAAhkZWxpc3RlZAAAAAAHQ0FTSE9VVAIAAAAHY2FzaG91dAAAAAAGQlVZT1VUAgAAAAZidXlvdXQAAAAACEZJTklTSEVEAgAAAAhmaW5pc2hlZAAAAAAHQ0xBSU1FRAIAAAAHY2xhaW1lZAAAAAAIUE9TSVRJVkUCAAAACHBvc2l0aXZlAAAAAAhORUdBVElWRQIAAAAIbmVnYXRpdmUAAAAAB0dFTkVTSVMCAAAAIzNQOEZ2eTF5RHdOSHZWcmFiZTRlazViOWRBd3hGakRLVjdSAAAAAAZWT1RFUlMAAAAAAAAAAAMAAAAABlFVT1JVTQAAAAAAAAAAAgAAAAAFVElFUlMJAARMAAAAAgkAAGgAAAACAAAAAAAAAAADAAAAAAAF9eEACQAETAAAAAIJAABoAAAAAgAAAAAAAAAACgAAAAAABfXhAAkABEwAAAACCQAAaAAAAAIAAAAAAAAAAGQAAAAAAAX14QAJAARMAAAAAgkAAGgAAAACAAAAAAAAAAEsAAAAAAAF9eEACQAETAAAAAIJAABoAAAAAgAAAAAAAAAD6AAAAAAABfXhAAUAAAADbmlsAAAAAApMSVNUSU5HRkVFCQAAaAAAAAIAAAAAAAAAAAMAAAAAAAX14QAAAAAAB1ZPVEVCRVQJAABoAAAAAgAAAAAAAAAAAQAAAAAABfXhAAAAAAAKTVVMVElQTElFUgAAAAAAAAAAlgAAAA4AAAABaQEAAAAKaW52aXRldXNlcgAAAAIAAAAKbmV3YWNjb3VudAAAAARkYXRhBAAAAAdhY2NvdW50CQACWAAAAAEICAUAAAABaQAAAAZjYWxsZXIAAAAFYnl0ZXMEAAAACW5ld3N0YXR1cwkBAAAAF2dldFZhbHVlV2hpdGVsaXN0U3RhdHVzAAAAAQUAAAAKbmV3YWNjb3VudAQAAAAKY3VycnN0YXR1cwkBAAAAF2dldFZhbHVlV2hpdGVsaXN0U3RhdHVzAAAAAQUAAAAHYWNjb3VudAMDCQAAAAAAAAIFAAAACW5ld3N0YXR1cwUAAAALV0hJVEVMSVNURUQGCQAAAAAAAAIFAAAACW5ld3N0YXR1cwUAAAAFV0hBTEUJAAACAAAAAQIAAAAgVXNlciBoYXMgYWxyZWFkeSBiZWVuIHJlZ2lzdGVyZWQDAwMJAQAAAAIhPQAAAAIFAAAACmN1cnJzdGF0dXMFAAAAC1dISVRFTElTVEVECQEAAAACIT0AAAACBQAAAAdhY2NvdW50BQAAAAdHRU5FU0lTBwkBAAAAAiE9AAAAAgUAAAAKY3VycnN0YXR1cwUAAAAFV0hBTEUHCQAAAgAAAAEJAAEsAAAAAgIAAAAsWW91ciBhY2NvdW50IHNob3VsZCBiZSB3aGl0ZWxpc3RlZC4gc3RhdHVzOiAFAAAACmN1cnJzdGF0dXMJAQAAAAhXcml0ZVNldAAAAAEJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAIJAQAAABJnZXRLZXlXaGl0ZWxpc3RSZWYAAAABBQAAAApuZXdhY2NvdW50BQAAAAdhY2NvdW50CQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACCQEAAAASZ2V0S2V5V2hpdGVsaXN0QmlvAAAAAQUAAAAKbmV3YWNjb3VudAUAAAAEZGF0YQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkBAAAAFWdldEtleVdoaXRlbGlzdFN0YXR1cwAAAAEFAAAACm5ld2FjY291bnQFAAAAB0lOVklURUQFAAAAA25pbAAAAAFpAQAAAAxzaWdudXBieWxpbmsAAAADAAAABGhhc2gAAAAEZGF0YQAAAAR0eXBlBAAAAAdhY2NvdW50CQACWAAAAAEI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FkJhZCB2b3RlIHJlc3VsdCBmb3JtYXQJAQAAAAxTY3JpcHRSZXN1bHQAAAACCQEAAAAIV3JpdGVTZXQAAAABCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACCQEAAAATZ2V0S2V5SXRlbUFjY1JldmVhbAAAAAIFAAAABGl0ZW0FAAAAB2FjY291bnQFAAAABHZvdGUJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAIJAQAAABJnZXRLZXlJdGVtVm90ZXNZZXMAAAABBQAAAARpdGVtCQAAZAAAAAIFAAAABnllc2NudAUAAAAHeWVzbWx0cAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkBAAAAEWdldEtleUl0ZW1Wb3Rlc05vAAAAAQUAAAAEaXRlbQkAAGQAAAACBQAAAAZub3RjbnQFAAAAB25vdG1sdHAJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAIJAQAAABBnZXRLZXlJdGVtU3RhdHVzAAAAAQUAAAAEaXRlbQUAAAAJbmV3c3RhdHVzCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACCQEAAAAbZ2V0S2V5SXRlbUFjY1ZvdGVSZXZpZXdUZXh0AAAAAgUAAAAEaXRlbQUAAAAHYWNjb3VudAUAAAAGcmV2aWV3BQAAAANuaWwJAQAAAAtUcmFuc2ZlclNldAAAAAEJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABBQAAAAdhY2NvdW50BQAAAAdWT1RFQkVUBQAAAAR1bml0BQAAAANuaWwAAAABaQEAAAAOZmluYWxpemV2b3RpbmcAAAACAAAABGl0ZW0AAAAHYWNjb3VudAQAAAAGeWVzY250CQEAAAAUZ2V0VmFsdWVJdGVtVm90ZXNZZXMAAAABBQAAAARpdGVtBAAAAAZub3RjbnQJAQAAABNnZXRWYWx1ZUl0ZW1Wb3Rlc05vAAAAAQUAAAAEaXRlbQQAAAAHYWNjdm90ZQkBAAAAFWdldFZhbHVlSXRlbUFjY1JldmVhbAAAAAIFAAAABGl0ZW0FAAAAB2FjY291bnQEAAAACGlzYXV0aG9yCQAAAAAAAAIFAAAAB2FjY291bnQJAQAAABJnZXRWYWx1ZUl0ZW1BdXRob3IAAAABBQAAAARpdGVtBAAAAAtmaW5hbHN0YXR1cwMJAABmAAAAAgUAAAAGeWVzY250BQAAAAZRVU9SVU0FAAAACEZFQVRVUkVEAwkAAGYAAAACBQAAAAZub3RjbnQFAAAABlFVT1JVTQUAAAAIREVMSVNURUQFAAAABE5PTkUEAAAAFG1sdGlzbm90ZnVsbG1ham9yaXR5AwMJAAAAAAAAAgUAAAAGeWVzY250BQAAAAZWT1RFUlMGCQAAAAAAAAIFAAAABm5vdGNudAUAAAAGVk9URVJTAAAAAAAAAAAAAAAAAAAAAAABBAAAAAhud2lubmVycwMJAAAAAAAAAgUAAAALZmluYWxzdGF0dXMFAAAACEZFQVRVUkVEBQAAAAZ5ZXNjbnQDCQAAAAAAAAIFAAAAC2ZpbmFsc3RhdHVzBQAAAAhERUxJU1RFRAUAAAAGbm90Y250AAAAAAAAAAAABAAAAAhubG9vc2VycwkAAGUAAAACBQAAAAZWT1RFUlMFAAAACG53aW5uZXJzBAAAAA5tbHRhY2Npc3dpbm5lcgMJAAAAAAAAAgUAAAALZmluYWxzdGF0dXMFAAAAB2FjY3ZvdGUAAAAAAAAAAAEAAAAAAAAAAAAEAAAACnZvdGVwcm9maXQDCQAAAAAAAAIFAAAACG53aW5uZXJzAAAAAAAAAAAAAAAAAAAAAAAACQAAaAAAAAIFAAAADm1sdGFjY2lzd2lubmVyCQAAZAAAAAIFAAAAB1ZPVEVCRVQJAABpAAAAAgkAAGgAAAACBQAAABRtbHRpc25vdGZ1bGxtYWpvcml0eQkAAGQAAAACCQAAaAAAAAIFAAAACG5sb29zZXJzBQAAAAdWT1RFQkVUBQAAAApMSVNUSU5HRkVFBQAAAAhud2lubmVycwQAAAAMYXV0aG9ycmV0dXJuCQAAaAAAAAIJAABoAAAAAgkAAGgAAAACBQAAAApMSVNUSU5HRkVFAwUAAAAIaXNhdXRob3IAAAAAAAAAAAEAAAAAAAAAAAADCQAAAAAAAAIFAAAAFG1sdGlzbm90ZnVsbG1ham9yaXR5AAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAABAwkAAAAAAAACBQAAAAtmaW5hbHN0YXR1cwUAAAAIRkVBVFVSRUQAAAAAAAAAAAEAAAAAAAAAAAADCQAAZgAAAAIJAQAAABxnZXRWYWx1ZUl0ZW1Wb3RpbmdFeHBpcmF0aW9uAAAAAQUAAAAEaXRlbQUAAAAGaGVpZ2h0CQAAAgAAAAECAAAAHlRoZSB2b3RpbmcgaGFzbid0IGZpbmlzaGVkIHlldAMJAAAAAAAAAgkBAAAAFGdldFZhbHVlSXRlbUFjY0ZpbmFsAAAAAgUAAAAEaXRlbQUAAAAHYWNjb3VudAUAAAAIRklOSVNIRUQJAAACAAAAAQIAAAAbQWNjb3VudCBoYXMgYWxyZWFkeSBjbGFpbWVkAwMJAAAAAAAAAgUAAAAHYWNjdm90ZQUAAAAETk9ORQkBAAAAASEAAAABBQAAAAhpc2F1dGhvcgcJAAACAAAAAQIAAAAzQWNjb3VudCBoYXNub3Qgdm90ZWQsIGhhc25vdCByZXZlYWwgb3IgaXNub3QgYXV0aG9yAwkAAAAAAAACBQAAAAtmaW5hbHN0YXR1cwUAAAAETk9ORQkAAAIAAAABAgAAABJWb3RpbmcgaGFzIGV4cGlyZWQJAQAAAAxTY3JpcHRSZXN1bHQAAAACCQEAAAAIV3JpdGVTZXQAAAABCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACCQEAAAASZ2V0S2V5SXRlbUFjY0ZpbmFsAAAAAgUAAAAEaXRlbQUAAAAHYWNjb3VudAUAAAAIRklOSVNIRUQFAAAAA25pbAkBAAAAC1RyYW5zZmVyU2V0AAAAAQkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCQEAAAAcQGV4dHJVc2VyKGFkZHJlc3NGcm9tU3RyaW5nKQAAAAEFAAAAB2FjY291bnQJAABkAAAAAgUAAAAKdm90ZXByb2ZpdAUAAAAMYXV0aG9ycmV0dXJuBQAAAAR1bml0BQAAAANuaWwAAAABaQEAAAASY2xvc2VleHBpcmVkdm90aW5nAAAAAgAAAARpdGVtAAAAB2FjY291bnQEAAAAC2ZpbmFsc3RhdHVzAwkAAGYAAAACCQEAAAAUZ2V0VmFsdWVJdGVtVm90ZXNZZXMAAAABBQAAAARpdGVtBQAAAAZRVU9SVU0FAAAACEZFQVRVUkVEAwkAAGYAAAACCQEAAAATZ2V0VmFsdWVJdGVtVm90ZXNObwAAAAEFAAAABGl0ZW0FAAAA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"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "inviteuser", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.StringDataEntry{Key: "wl_ref_3P9yVruoCbs4cveU8HpTdFUvzwY59ADaQm3", Value: "3P8Fvy1yDwNHvVrabe4ek5b9dAwxFjDKV7R"}},
		{Entry: &proto.StringDataEntry{Key: "wl_bio_3P9yVruoCbs4cveU8HpTdFUvzwY59ADaQm3", Value: `{"name":"James May","message":"Hello!","isWhale":false,"address":"3P9yVruoCbs4cveU8HpTdFUvzwY59ADaQm3"}`}},
		{Entry: &proto.StringDataEntry{Key: "wl_sts_3P9yVruoCbs4cveU8HpTdFUvzwY59ADaQm3", Value: "invited"}},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

func TestExchangeDApp(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("46R51i3ATxvYbrLJVWpAG3hZuznXtgEobRW6XSZ9MP6f")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5MriXpPgobRfNHqYx3vSjrZkDdzDrRF6krgvJp1FRvo2qTyk1KB913Nk1H2hWyKPDzL6pV1y8AWREHdQMGStCBuF")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("APg7QwJSx6naBUPnGYM2vvsJxQcpYabcbzkNJoMUXLai")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3PJYvHqNcUsfQyPkvVCYMqYsi1xZKLmKT6k")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6"})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "cancel",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        nil,
		FeeAsset:        proto.OptionalAsset{},
		Fee:             900000,
		Timestamp:       1564703444249,
	}
	gs := crypto.MustBytesFromBase58("AWH9QVEnmN6VjRyEfs93UtAiCkwrNJ2phKYe25KFNCz")
	gen, err := proto.NewAddressFromString("3PMR8zZMswxrVdidk2mZAvRAXtJPSRJjt76")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1566052715248,
		Height:              1665137,
		BaseTarget:          69,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender,
	}

	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 1642207
		},
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 1642207, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
					switch key {
					case "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6":
						v, err := base64.StdEncoding.DecodeString("AAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLw==")
						require.NoError(t, err)
						return &proto.BinaryDataEntry{Key: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6", Value: v}, nil
					default:
						return nil, errors.New("fail")
					}
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(3, proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "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"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "cancel", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	ev, err := base64.StdEncoding.DecodeString("AAAAAAABhqAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLw==")
	require.NoError(t, err)
	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.BinaryDataEntry{Key: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6", Value: ev}},
	}
	ra, err := proto.NewAddressFromString("3P8WrXSDDyNC11dm8XANKeDcJricefgTRyZ")
	require.NoError(t, err)
	rcp := proto.NewRecipientFromAddress(ra)
	asset, err := crypto.NewDigestFromBase58("78tZbyEovK6DLyqfmswMDtxb3bytTX7H5p6hYpGhYtBV")
	require.NoError(t, err)
	expectedTransfers := []*proto.TransferScriptAction{
		{
			Recipient: rcp,
			Amount:    1,
			Asset:     *proto.NewOptionalAssetFromDigest(asset),
		},
	}
	expectedResult := &proto.ScriptResult{
		Transfers:    expectedTransfers,
		DataEntries:  expectedDataWrites,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

func TestBankDApp(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("2VbGtacF9WPYJmcXLU43vAUfRnccNZcywCdWkSsjx71b")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("66YM8VTw4qSECuFRPSVURXvLXrHbuGoCJ3xdLp15fx5jyxmyxpS797kg1xWQMuKwWrbYuyw84bS56SqutwZsA5ih")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("DX4ekUJ3RwVvCd2qSXhcoTKwuP5k8wjvPeZZ298trbpB")
	require.NoError(t, err)
	dapp, err := proto.NewAddressFromString("3P4ub5GDTxMMr9VAoWzvMKofXWLbbpBxqZS")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(dapp)
	call := proto.FunctionCall{
		Default:   false,
		Name:      "buyBack",
		Arguments: proto.Arguments{},
	}
	paymentAsset, err := crypto.NewDigestFromBase58("8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS")
	require.NoError(t, err)
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments: proto.ScriptPayments{proto.ScriptPayment{
			Amount: 213,
			Asset:  *proto.NewOptionalAssetFromDigest(paymentAsset),
		}},
		FeeAsset:  proto.OptionalAsset{},
		Fee:       5000000,
		Timestamp: 1566898524090,
	}
	intEntries := map[string]*proto.IntegerDataEntry{
		"start_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm":           {Key: "start_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm", Value: 1},
		"deposit_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm":         {Key: "deposit_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm", Value: 0},
		"end_of_burndown_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm": {Key: "end_of_burndown_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm", Value: 0},
		"end_of_freeze_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm":   {Key: "end_of_freeze_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm", Value: 0},
		"end_of_grace_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm":    {Key: "end_of_grace_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm", Value: 0},
		"end_of_interest_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm": {Key: "end_of_interest_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm", Value: 0},
		"lend_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm":            {Key: "lend_of_3PEMtQYE48MGe3iTCkP2sq2dW5fFx67QWAm", Value: 213},
		"gracePeriod":        {Key: "gracePeriod", Value: 1440},
		"interestPeriod":     {Key: "interestPeriod", Value: 43200},
		"maxRate":            {Key: "maxRate", Value: 50000},
		"discountPercentile": {Key: "discountPercentile", Value: 66},
		"burndownPeriod":     {Key: "burndownPeriod", Value: 1440000},
	}
	stringEntries := map[string]*proto.StringDataEntry{
		"assetToken": {Key: "assetToken", Value: "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS"},
		"oracle":     {Key: "oracle", Value: "3PPTrTo3AzR56N7ArzbU3Bpq9zYMgcf39Mk"},
		"owner":      {Key: "owner", Value: "3PGaPQp15c1iUyQidFHcEGhdqyHiVZDLeCK"},
	}
	gs := crypto.MustBytesFromBase58("AWH9QVEnmN6VjRyEfs93UtAiCkwrNJ2phKYe25KFNCz")
	gen, err := proto.NewAddressFromString("3PMR8zZMswxrVdidk2mZAvRAXtJPSRJjt76")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1566052715248,
		Height:              1665137,
		BaseTarget:          69,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender,
	}

	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 0
		},
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 1642207, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					v, ok := intEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return v, nil
				},
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					v, ok := stringEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return v, nil
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(dapp)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(3, proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "AAIDAAAAAAAAAC0IARIKCggICAgBAQEBARIICgYIAQEBAQESABIDCgEIEgASAwoBCBIAEgMKAQEAAAAjAAAAAAxkZXBvc2l0VG9rZW4FAAAABHVuaXQAAAAADW9yYWNsZURhdGFLZXkCAAAAC3dhdmVzX2J0Y184AAAAAAR0ZW44CQAAaAAAAAIJAABoAAAAAgAAAAAAAAAAZAAAAAAAAAAD6AAAAAAAAAAD6AAAAAAOZ3JhY2VQZXJpb2RLZXkCAAAAC2dyYWNlUGVyaW9kAAAAABFpbnRlcmVzdFBlcmlvZEtleQIAAAAOaW50ZXJlc3RQZXJpb2QAAAAAEWJ1cm5kb3duUGVyaW9kS2V5AgAAAA5idXJuZG93blBlcmlvZAAAAAAJb3JhY2xlS2V5AgAAAAZvcmFjbGUAAAAAFWRpc2NvdW50UGVyY2VudGlsZUtleQIAAAASZGlzY291bnRQZXJjZW50aWxlAAAAAAptYXhSYXRlS2V5AgAAAAdtYXhSYXRlAAAAAA1hc3NldFRva2VuS2V5AgAAAAphc3NldFRva2VuAAAAAAhvd25lcktleQIAAAAFb3duZXIBAAAAB3N0YXJ0T2YAAAABAAAABnJlbnRlcgkAASwAAAACAgAAAAlzdGFydF9vZl8FAAAABnJlbnRlcgEAAAAMZW5kT2ZHcmFjZU9mAAAAAQAAAAZyZW50ZXIJAAEsAAAAAgIAAAAQZW5kX29mX2dyYWNlX29mXwUAAAAGcmVudGVyAQAAAA9lbmRPZkludGVyZXN0T2YAAAABAAAABnJlbnRlcgkAASwAAAACAgAAABNlbmRfb2ZfaW50ZXJlc3Rfb2ZfBQAAAAZyZW50ZXIBAAAAD2VuZE9mQnVybmRvd25PZgAAAAEAAAAGcmVudGVyCQABLAAAAAICAAAAE2VuZF9vZl9idXJuZG93bl9vZl8FAAAABnJlbnRlcgEAAAAGcmF0ZU9mAAAAAQAAAAZyZW50ZXIJAAEsAAAAAgIAAAAIcmF0ZV9vZl8FAAAABnJlbnRlcgEAAAAJZGVwb3NpdE9mAAAAAQAAAAZyZW50ZXIJAAEsAAAAAgIAAAALZGVwb3NpdF9vZl8FAAAABnJlbnRlcgEAAAAKbGVuZEFtb3VudAAAAAEAAAAGcmVudGVyCQABLAAAAAICAAAACGxlbmRfb2ZfBQAAAAZyZW50ZXIBAAAADHJlZ2lzdGVyZWRUeAAAAAEAAAAEdHhJZAkAASwAAAACAgAAABVyZWdpc3RlcmVkX3JldHVybl9vZl8FAAAABHR4SWQAAAAABW93bmVyCQEAAAAcQGV4dHJVc2VyKGFkZHJlc3NGcm9tU3RyaW5nKQAAAAEJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABB0AAAACBQAAAAR0aGlzBQAAAAhvd25lcktleQIAAAAITm8gb3duZXIAAAAACmFzc2V0VG9rZW4JAAJZAAAAAQkBAAAAE3ZhbHVlT3JFcnJvck1lc3NhZ2UAAAACCQAEHQAAAAIFAAAABHRoaXMFAAAADWFzc2V0VG9rZW5LZXkCAAAACk5vIGFzc2V0SWQAAAAAC2dyYWNlUGVyaW9kCQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAAQaAAAAAgUAAAAEdGhpcwUAAAAOZ3JhY2VQZXJpb2RLZXkCAAAAD05vIGdyYWNlIHBlcmlvZAAAAAAOaW50ZXJlc3RQZXJpb2QJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABBoAAAACBQAAAAR0aGlzBQAAABFpbnRlcmVzdFBlcmlvZEtleQIAAAASTm8gaW50ZXJlc3QgcGVyaW9kAAAAAA5idXJuZG93blBlcmlvZAkBAAAAE3ZhbHVlT3JFcnJvck1lc3NhZ2UAAAACCQAEGgAAAAIFAAAABHRoaXMFAAAAEWJ1cm5kb3duUGVyaW9kS2V5AgAAABJObyBidXJuZG93biBwZXJpb2QAAAAAB21heFJhdGUJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABBoAAAACBQAAAAR0aGlzBQAAAAptYXhSYXRlS2V5AgAAABNObyBvcmFjbGUgbWF4IHZhbHVlAAAAAAZvcmFjbGUJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABB0AAAACBQAAAAR0aGlzBQAAAAlvcmFjbGVLZXkCAAAACU5vIG9yYWNsZQAAAAALb3JhY2xlVmFsdWUJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABBoAAAACCQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAQAAABFhZGRy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AAACBQAAAAR0aGlzCQEAAAAPZW5kT2ZJbnRlcmVzdE9mAAAAAQUAAAAJcmVudGVyU3RyBAAAABNjYW5SZXR1cm5GdWxsQW1vdW50CQAAZwAAAAIFAAAACmVuZE9mR3JhY2UFAAAABmhlaWdodAQAAAAMcmV0dXJuQW1vdW50AwUAAAATY2FuUmV0dXJuRnVsbEFtb3VudAUAAAAOZGVwb3NpdGVkVmFsdWUDCQAAZwAAAAIFAAAABmhlaWdodAUAAAANZW5kT2ZJbnRlcmVzdAkAAAIAAAABAgAAABV5b3VyIGxvYW4gaGFzIGV4cGlyZWQJAABrAAAAAwUAAAAOZGVwb3NpdGVkVmFsdWUJAABlAAAAAgUAAAANZW5kT2ZCdXJuZG93bgUAAAAGaGVpZ2h0CQAAZQAAAAIFAAAADWVuZE9mQnVybmRvd24FAAAACmVuZE9mR3JhY2UEAAAAD3RoZVJlc3RPZkFtb3VudAkAAGUAAAACBQAAAA5kZXBvc2l0ZWRWYWx1ZQUAAAAMcmV0dXJuQW1vdW50CQEAAAAMU2NyaXB0UmVzdWx0AAAAAgkBAAAAB2Nsb3NpbmcAAAABBQAAAAlyZW50ZXJTdHIJAQAAAAtUcmFuc2ZlclNldAAAAAEJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwUAAAAGcmVudGVyBQAAAAxyZXR1cm5BbW91bnQFAAAADGRlcG9zaXRUb2tlbgkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADBQAAAAVvd25lcgUAAAAPdGhlUmVzdE9mQW1vdW50BQAAAAxkZXBvc2l0VG9rZW4FAAAAA25pbAAAAAgAAAABaQEAAAAEaW5pdAAAAAgAAAAFb3duZXIAAAAFdG9rZW4AAAAGb3JhY2xlAAAAB21heFJhdGUAAAAIZGlzY291bnQAAAAFZ3JhY2UAAAAIaW50ZXJlc3QAAAAIYnVybmRvd24DCQAAZgAAAAIFAAAACGludGVyZXN0BQAAAAhidXJuZG93bgkAAAIAAAABAgAAACppbnRlcmVzdCBtdXN0IGJlIGxlc3Mgb3IgZXF1YWwgdG8gYnVybmRvd24DCQEAAAACIT0AAAACCAUAAAABaQAAAAZjYWxsZXIFAAAABHRoaXMJAAACAAAAAQIAAAAZb25seSBkYXBwIGl0c2VsZiBjYW4gaW5pdAkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgUAAAAIb3duZXJLZXkFAAAABW93bmVyCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACBQAAAA1hc3NldFRva2VuS2V5BQAAAAV0b2tlbgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgUAAAAJb3JhY2xlS2V5BQAAAAZvcmFjbGUJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAIFAAAACm1heFJhdGVLZXkFAAAAB21heFJhdGUJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAIFAAAAFWRpc2NvdW50UGVyY2VudGlsZUtleQUAAAAIZGlzY291bnQJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAIFAAAADmdyYWNlUGVyaW9kS2V5BQAAAAVncmFjZQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgUAAAARaW50ZXJlc3RQZXJpb2RLZXkFAAAACGludGVyZXN0CQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACBQAAABFidXJuZG93blBlcmlvZEtleQUAAAAIYnVybmRvd24FAAAAA25pbAAAAAFpAQAAAAx1cGRhdGVQYXJhbXMAAAAGAAAABm9yYWNsZQAAAAdtYXhSYXRlAAAACGRpc2NvdW50AAAABWdyYWNlAAAACGludGVyZXN0AAAACGJ1cm5kb3duAwkAAGYAAAACBQAAAAhpbnRlcmVzdAUAAAAIYnVybmRvd24JAAACAAAAAQIAAAAqaW50ZXJlc3QgbXVzdCBiZSBsZXNzIG9yIGVxdWFsIHRvIGJ1cm5kb3duAwkBAAAAAiE9AAAAAggFAAAAAWkAAAAGY2FsbGVyBQAAAAVvd25lcgkAAAIAAAABAgAAABxvbmx5IG93bmVyIGNhbiB1cGRhdGUgcGFyYW1zCQEAAAAIV3JpdGVTZXQAAAABCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACBQAAAAlvcmFjbGVLZXkFAAAABm9yYWNsZQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgUAAAAKbWF4UmF0ZUtleQUAAAAHbWF4UmF0ZQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgUAAAAVZGlzY291bnRQZXJjZW50aWxlS2V5BQAAAAhkaXNjb3VudAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgUAAAAOZ3JhY2VQZXJpb2RLZXkFAAAABWdyYWNlCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACBQAAABFpbnRlcmVzdFBlcmlvZEtleQUAAAAIaW50ZXJlc3QJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAIFAAAAEWJ1cm5kb3duUGVyaW9kS2V5BQAAAAhidXJuZG93bgUAAAADbmlsAAAAAWkBAAAABmJvcnJvdwAAAAAEAAAABnJlbnRlcgkAAlgAAAABCAgFAAAAAWkAAAAGY2FsbGVyAAAABWJ5dGVzAwkBAAAACmlzTGVuZE9wZW4AAAABBQAAAAZyZW50ZXIJAAACAAAAAQkAASwAAAACBQAAAAZyZW50ZXICAAAAGSBhbHJlYWR5IGhhcyBhbiBvcGVuIGxvYW4EAAAAByRtYXRjaDAIBQAAAAFpAAAAB3BheW1lbnQDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0F0dGFjaGVkUGF5bWVudAQAAAABYQUAAAAHJG1hdGNoMAMJAAAAAAAAAggFAAAAAWEAAAAHYXNzZXRJZAUAAAAMZGVwb3NpdFRva2VuBAAAAA1jdXJyZW50SGVpZ2h0BQAAAAZoZWlnaHQEAAAACmVuZE9mR3JhY2UJAABkAAAAAgUAAAAGaGVpZ2h0BQAAAAtncmFjZVBlcmlvZAQAAAANZW5kT2ZJbnRlcmVzdAkAAGQAAAACBQAAAAplbmRPZkdyYWNlBQAAAA5pbnRlcmVzdFBlcmlvZAQAAAANZW5kT2ZCdXJuZG93bgkAAGQAAAACBQAAAAplbmRPZkdyYWNlBQAAAA5idXJuZG93blBlcmlvZAQAAAANZGVwb3NpdEFtb3VudAgFAAAAAWEAAAAGYW1vdW50BAAAAA9hc3NldFRva2Vuc0xlbnQJAABrAAAAAwUAAAANZGVwb3NpdEFtb3VudAkAAGgAAAACBQAAAARyYXRlBQAAABJkaXNjb3VudFBlcmNlbnRpbGUJAABoAAAAAgUAAAAEdGVuOAAAAAAAAAAAZAMJAABmAAAAAgUAAAAPYXNzZXRU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"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "buyBack", proto.Arguments{})
	require.NoError(t, err)
	_, ok := res.(DAppResult)
	require.True(t, ok)
}

func TestLigaDApp1(t *testing.T) {
	const waves = 100000000

	stringEntries := map[string]*proto.StringDataEntry{"EVENT_INFO": {Key: "EVENT_INFO", Value: "{\"totalTeams\":2,\"tokenQuantity\":1000,\"eventEndsAtBlock\":607279,\"winnerDeclarationInterval\":5,\"payoutInterval\":20,\"eventPublicKey\":\"FpwhtxwCJRg5pvyadHnCmCHsgbLY1JrFgmDpzZanmATc\",\"oraclePublicKey\":\"HCAWcxP3r7Yym8fJR5GzD96ve15ZpGHoFNgVFDy8WKXt\",\"ligaPublicKey\":\"56xTC8QUv2imTCZqZSvWNbjiKbPLMqbfrUk9nNPd1ra6\",\"leaseNodePublicKey\":\"HCAWcxP3r7Yym8fJR5GzD96ve15ZpGHoFNgVFDy8WKXt\"}"}}
	intEntries := map[string]*proto.IntegerDataEntry{
		"4njdbzZQNBSPgU2WWPfcKEnUbFvSKTHQBRdGk2mJJ9ye_BASE_PRICE": {Key: "4njdbzZQNBSPgU2WWPfcKEnUbFvSKTHQBRdGk2mJJ9ye_BASE_PRICE", Value: 1000},
		"FAysFm8ejh8dqDLooe83V7agUNc9CyC2bomy3NDUxGXv_BASE_PRICE": {Key: "FAysFm8ejh8dqDLooe83V7agUNc9CyC2bomy3NDUxGXv_BASE_PRICE", Value: 1000},
	}

	team1, err := crypto.NewDigestFromBase58("4njdbzZQNBSPgU2WWPfcKEnUbFvSKTHQBRdGk2mJJ9ye")
	require.NoError(t, err)
	team2, err := crypto.NewDigestFromBase58("FAysFm8ejh8dqDLooe83V7agUNc9CyC2bomy3NDUxGXv")
	require.NoError(t, err)

	// First transaction
	dapp, err := proto.NewAddressFromString("3MtbWQVSuENX7KQs3VpXmr4wnfJ17ZEJK9t")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(dapp)
	pk, err := crypto.NewPublicKeyFromBase58("FpwhtxwCJRg5pvyadHnCmCHsgbLY1JrFgmDpzZanmATc")
	require.NoError(t, err)

	tx1ID, err := crypto.NewDigestFromBase58("CJq88YkPY5fzveKtjaVDv3xSq5tp5Nw3ttFRWpmDZENf")
	require.NoError(t, err)
	proof1, err := crypto.NewSignatureFromBase58("5faEzRCJ5Ucz9XJE3NcSznQxvJ9rTuoRPgoHRKXyUTyoJdRoB7CJcrW3gBRNM33cawXnnD3tcenVWS6zr8kwrf9x")
	require.NoError(t, err)
	proofs1 := proto.NewProofs()
	proofs1.Proofs = []proto.B58Bytes{proof1[:]}
	sender1, err := crypto.NewPublicKeyFromBase58("56xTC8QUv2imTCZqZSvWNbjiKbPLMqbfrUk9nNPd1ra6")
	require.NoError(t, err)
	call1 := proto.FunctionCall{
		Default:   false,
		Name:      "stage2",
		Arguments: proto.Arguments{},
	}
	tx1 := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &tx1ID,
		Proofs:          proofs1,
		ChainID:         proto.TestNetScheme,
		SenderPK:        sender1,
		ScriptRecipient: recipient,
		FunctionCall:    call1,
		Payments:        nil,
		FeeAsset:        proto.OptionalAsset{},
		Fee:             5000000,
		Timestamp:       1564398503427,
	}

	gs := crypto.MustBytesFromBase58("A9CAzLPzCzweUH4hBmaWHxNeqHDMipiEee8HphivNs4h")
	gen, err := proto.NewAddressFromString("3NB1Yz7fH1bJ2gVDjyJnuyKNTdMFARkKEpV")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1564398502318,
		Height:              607280,
		BaseTarget:          1155,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender1,
	}

	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 1642207
		},
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 607280, nil
				},
				NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
					return 3*waves - 2*waves - 100000 - 1000000 + 5 - 150000, nil
				},
				NewestAssetBalanceFunc: func(account proto.Recipient, asset crypto.Digest) (uint64, error) {
					switch asset {
					case team1:
						return 1000 - 5, nil
					case team2:
						return 1000, nil
					default:
						return 0, errors.Errorf("unexpected asset %q", asset.String())
					}
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				NewestTransactionHeightByIDFunc: func(in1 []byte) (uint64, error) {
					return 607280, nil
				},
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					v, ok := intEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return v, nil
				},
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					v, ok := stringEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return v, nil
				},
				RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
					return nil, errors.New("fail")
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(dapp)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx1)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(3, proto.TestNetScheme, tx1)
			require.NoError(t, err)
			return obj
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "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AAAQQkFMQU5DRV9TTkFQU0hPVAEAAAASZ2V0QmFsYW5jZVNuYXBzaG90AAAAAAkBAAAACGdldEludE9yAAAAAgUAAAAPQkFMQU5DRVNOQVBTSE9UCQEAAAAMd2F2ZXNCYWxhbmNlAAAAAQUAAAAEdGhpcwEAAAASc2V0QmFsYW5jZVNuYXBzaG90AAAAAAkBAAAABnNldEludAAAAAIFAAAAD0JBTEFOQ0VTTkFQU0hPVAkBAAAAEmdldEJhbGFuY2VTbmFwc2hvdAAAAAAAAAAACVBSSVpFUE9PTAIAAAAKUFJJWkVfUE9PTAEAAAAMZ2V0UHJpemVQb29sAAAAAAkBAAAACGdldEludE9yAAAAAgUAAAAJUFJJWkVQT09MCQAAaQAAAAIJAABoAAAAAgkBAAAAEmdldEJhbGFuY2VTbmFwc2hvdAAAAAAJAABlAAAAAgAAAAAAAAAAZAUAAAAObGlnYUNvbW1pc3Npb24AAAAAAAAAAGQBAAAADHNldFByaXplUG9vbAAAAAAJAQAAAAZzZXRJbnQAAAACBQAAAAlQUklaRVBPT0wJAQAAAAxnZXRQcml6ZVBvb2wAAAAAAAAAAAZXSU5ORVICAAAABldJTk5FUgEAAAAJZ2V0V2lubmVyAAAAAAkBAAAACGdldEJ5dGVzAAAAAQUAAAAGV0lOTkVSAQAAAAlzZXRXaW5uZXIAAAABAAAABndpbm5lcgkBAAAACHNldEJ5dGVzAAAAAgUAAAAGV0lOTkVSBQAAAAZ3aW5uZXIAAAAACVRFQU1TTEVGVAIAAAAKVEVBTVNfTEVGVAEAAAAMZ2V0VGVhbXNMZWZ0AAAAAAkBAAAACGdldEludE9yAAAAAgUAAAAJVEVBTVNMRUZUBQAAAAp0b3RhbFRlYW1zAQAAAAxkZWNUZWFtc0xlZnQAAAAACQEAAAAGc2V0SW50AAAAAgUAAAAJVEVBTVNMRUZUCQAAZQAAAAIJAQAAAAxnZXRUZWFtc0xlZnQAAAAAAAAAAAAAAAABAAAAAAtURUFNQ09VTlRFUgIAAAAMVEVBTV9DT1VOVEVSAQAAAA5nZXRUZWFtQ291bnRlcgAAAAAJAQAAAAhnZXRJbnRPcgAAAAIFAAAAC1RFQU1DT1VOVEVSAAAAAAAAAAAAAQAAAA5pbmNUZWFtQ291bnRlcgAAAAAJAQAA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"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "stage2", proto.Arguments{})
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "STAGE", Value: 2}},
		{Entry: &proto.IntegerDataEntry{Key: "BALANCE_SNAPSHOT", Value: 98750005}},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)

	// Second transaction
	stringEntries = map[string]*proto.StringDataEntry{
		"EVENT_INFO": {Key: "EVENT_INFO", Value: "{\"totalTeams\":2,\"tokenQuantity\":1000,\"eventEndsAtBlock\":607279,\"winnerDeclarationInterval\":5,\"payoutInterval\":20,\"eventPublicKey\":\"FpwhtxwCJRg5pvyadHnCmCHsgbLY1JrFgmDpzZanmATc\",\"oraclePublicKey\":\"HCAWcxP3r7Yym8fJR5GzD96ve15ZpGHoFNgVFDy8WKXt\",\"ligaPublicKey\":\"56xTC8QUv2imTCZqZSvWNbjiKbPLMqbfrUk9nNPd1ra6\",\"leaseNodePublicKey\":\"HCAWcxP3r7Yym8fJR5GzD96ve15ZpGHoFNgVFDy8WKXt\"}"},
	}
	intEntries = map[string]*proto.IntegerDataEntry{
		"4njdbzZQNBSPgU2WWPfcKEnUbFvSKTHQBRdGk2mJJ9ye_BASE_PRICE": {Key: "4njdbzZQNBSPgU2WWPfcKEnUbFvSKTHQBRdGk2mJJ9ye_BASE_PRICE", Value: 1000},
		"FAysFm8ejh8dqDLooe83V7agUNc9CyC2bomy3NDUxGXv_BASE_PRICE": {Key: "FAysFm8ejh8dqDLooe83V7agUNc9CyC2bomy3NDUxGXv_BASE_PRICE", Value: 1000},
		"STAGE":            {Key: "STAGE", Value: 2},
		"BALANCE_SNAPSHOT": {Key: "BALANCE_SNAPSHOT", Value: 98750005},
	}
	tx2ID, err := crypto.NewDigestFromBase58("H4DY2btAHzhnzdNo5mx8LB965FBVkKuLMg8hBDq1aJaC")
	require.NoError(t, err)
	proof2, err := crypto.NewSignatureFromBase58("5jbfbEtmFV5LSMpDPP5Z7z4CB8AbzXP6YpwSsz2e88soHivqCMyo4hd73KYnipRjE9cFVc7eirPaTif17M28kydu")
	require.NoError(t, err)
	proofs2 := proto.NewProofs()
	proofs2.Proofs = []proto.B58Bytes{proof2[:]}
	sender2, err := crypto.NewPublicKeyFromBase58("HCAWcxP3r7Yym8fJR5GzD96ve15ZpGHoFNgVFDy8WKXt")
	require.NoError(t, err)
	av, err := base64.StdEncoding.DecodeString("OEpmeyPHnGfKvK5JJ/bJ82VVY6ScsiH6JQpdnT+tCO0=")
	require.NoError(t, err)
	args2 := proto.Arguments{}
	args2.Append(&proto.BinaryArgument{Value: av})
	call2 := proto.FunctionCall{
		Default:   false,
		Name:      "stage31",
		Arguments: args2,
	}
	tx2 := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &tx2ID,
		Proofs:          proofs2,
		ChainID:         proto.TestNetScheme,
		SenderPK:        sender2,
		ScriptRecipient: recipient,
		FunctionCall:    call2,
		Payments:        nil,
		FeeAsset:        proto.OptionalAsset{},
		Fee:             5000000,
		Timestamp:       1564398515618,
	}

	gs = crypto.MustBytesFromBase58("CDhrqb9p6x5V5dav1Kj1ffQjtrhUk6QVjiB6KPhbkeDs")
	gen, err = proto.NewAddressFromString("3MxyKNmnQkVuDCG9AzMpixKCdUWXfMUsxdg")
	require.NoError(t, err)
	blockInfo = &proto.BlockInfo{
		Timestamp:           1564398522337,
		Height:              607281,
		BaseTarget:          1155,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender1,
	}

	env = &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 1642207
		},
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 607281, nil
				},
				NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
					return 3*waves - 2*waves - 100000 - 1000000 + 5 - 150000, nil
				},
				NewestAssetBalanceFunc: func(account proto.Recipient, asset crypto.Digest) (uint64, error) {
					switch asset {
					case team1:
						return 1000 - 5, nil
					case team2:
						return 1000, nil
					default:
						return 0, errors.Errorf("unexpected asset %q", asset.String())
					}
				},
				NewestAssetInfoFunc: func(asset crypto.Digest) (*proto.AssetInfo, error) {
					switch asset {
					case team1:
						return &proto.AssetInfo{
							ID:              team1,
							Quantity:        1000,
							Decimals:        0,
							Issuer:          dapp,
							IssuerPublicKey: pk,
							Reissuable:      false,
							Scripted:        false,
							Sponsored:       false,
						}, nil
					case team2:
						return &proto.AssetInfo{
							ID:              team2,
							Quantity:        1000,
							Decimals:        0,
							Issuer:          dapp,
							IssuerPublicKey: pk,
							Reissuable:      false,
							Scripted:        false,
							Sponsored:       false,
						}, nil
					default:
						return nil, errors.New("fail")
					}
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				NewestTransactionHeightByIDFunc: func(in1 []byte) (uint64, error) {
					switch base58.Encode(in1) {
					case "CJq88YkPY5fzveKtjaVDv3xSq5tp5Nw3ttFRWpmDZENf":
						return 607280, nil
					case "H4DY2btAHzhnzdNo5mx8LB965FBVkKuLMg8hBDq1aJaC":
						return 607281, nil
					default:
						return 0, errors.New("fail")
					}
				},
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					v, ok := intEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return v, nil
				},
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					v, ok := stringEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return v, nil
				},
				RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
					return nil, errors.New("fail")
				},
				RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
					return nil, errors.New("fail")
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(dapp)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx2)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(3, proto.TestNetScheme, tx2)
			require.NoError(t, err)
			return obj
		},
		rideV6ActivatedFunc: noRideV6,
	}

	res, err = CallFunction(env, tree, "stage31", args2)
	require.NoError(t, err)
	r, ok = res.(DAppResult)
	require.True(t, ok)

	sr, ap, err = proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedDataWrites = []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "STAGE", Value: 31}},
		{Entry: &proto.IntegerDataEntry{Key: "PRIZE_POOL", Value: 94800004}},
		{Entry: &proto.BinaryDataEntry{Key: "WINNER", Value: av}},
		{Entry: &proto.IntegerDataEntry{Key: "4njdbzZQNBSPgU2WWPfcKEnUbFvSKTHQBRdGk2mJJ9ye_SOLD", Value: 5}},
	}
	expectedResult = &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

func TestTestingDApp(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("GSUXtkuw1jgVdbkZ3BAVP7kP7VL5RJ2kYyZiaTebW8Xs")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("cvxrRfTStMSYpjybWHG7bBZAGChqWczD1MfbFcapcofSDK7HBuHGm6e3uouD7vuqHEFbqL2k8L4i2vBSFvHcjj1")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("HGT44HrsSSD5cjANV6wtWNB9VKS3y7hhoNXEDWB56Lu9")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3MvqUEAdK8oa1jDS82eqYYVoHTX3S71rRPa")
	require.NoError(t, err)
	av1, err := base64.StdEncoding.DecodeString("ANLvS9cUXfiHVkxyrfhPuJUWnPrALiWiXhaEaqkNb86h")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	arguments.Append(&proto.BinaryArgument{Value: av1})
	arguments.Append(&proto.StringArgument{Value: "10"})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "main",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.TestNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{proto.ScriptPayment{Amount: 1, Asset: proto.OptionalAsset{}}},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1567938316714,
	}
	gs := crypto.MustBytesFromBase58("AWH9QVEnmN6VjRyEfs93UtAiCkwrNJ2phKYe25KFNCz")
	gen, err := proto.NewAddressFromString("3MxTeL8dKLUGh9B1A2aaZxQ8BLL22bDdm6G")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1567938316714,
		Height:              666972,
		BaseTarget:          1550,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender,
	}
	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 1642207
		},
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 666972, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
					switch key {
					case "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6":
						v, err := base64.StdEncoding.DecodeString("AAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLw==")
						require.NoError(t, err)
						return &proto.BinaryDataEntry{Key: "B9spbWQ1rk7YqJUFjW8mLHw6cRcngyh7G9YgRuyFtLv6", Value: v}, nil
					default:
						return nil, errors.New("fail")
					}
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(3, proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "AAIDAAAAAAAAAAAAAAABAAAAAAhudWxsSG9tZQIAAAABMAAAAAIAAAABaQEAAAAHc2V0aG9tZQAAAAEAAAAGaG9tZUlkBAAAAAxzZWxsZXJCYXNlNTgJAAJYAAAAAQgIBQAAAAFpAAAABmNhbGxlcgAAAAVieXRlcwQAAAAFd3JpdGUJAQAAAAhXcml0ZVNldAAAAAEJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAIFAAAADHNlbGxlckJhc2U1OAUAAAAGaG9tZUlkCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACAgAAAApzZXRob21lTG9nCQABLAAAAAIJAAEsAAAAAgUAAAAMc2VsbGVyQmFzZTU4AgAAAAQgLT4gBQAAAAZob21lSWQFAAAAA25pbAkBAAAADFNjcmlwdFJlc3VsdAAAAAIFAAAABXdyaXRlCQEAAAALVHJhbnNmZXJTZXQAAAABBQAAAANuaWwAAAABaQEAAAAEbWFpbgAAAAIAAAAMc2VsbGVyVmVjdG9yAAAABmhvbWVJZAQAAAADcG10CQEAAAAHZXh0cmFjdAAAAAEIBQAAAAFpAAAAB3BheW1lbnQDCQEAAAAJaXNEZWZpbmVkAAAAAQgFAAAAA3BtdAAAAAdhc3NldElkCQAAAgAAAAECAAAAFXBheW1lbnQgaXMgb25seSB3YXZlcwQAAAAKYnVlckJhc2U1OAkAAlgAAAABCAgFAAAAAWkAAAAGY2FsbGVyAAAABWJ5dGVzBAAAAA1zZWxsZXJBZGRyZXNzCQEAAAAUYWRkcmVzc0Zyb21QdWJsaWNLZXkAAAABBQAAAAxzZWxsZXJWZWN0b3IEAAAAAWEJAQAAAAdBZGRyZXNzAAAAAQUAAAAMc2VsbGVyVmVjdG9yBAAAAAJzMQkAAlgAAAABBQAAAAxzZWxsZXJWZWN0b3IEAAAAAnMyCQACWAAAAAEIBQAAAA1zZWxsZXJBZGRyZXNzAAAABWJ5dGVzBAAAAAJzMwkAAlgAAAABCAUAAAABYQAAAAVieXRlcwkBAAAADFNjcmlwdFJlc3VsdAAAAAIJAQAAAAhXcml0ZVNldAAAAAEJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAAB21haW5Mb2cJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIFAAAAAnMxAgAAAAMgLSAFAAAAAnMyAgAAAAMgLSAFAAAAAnMzAgAAAAQgLT4gBQAAAApidWVyQmFzZTU4BQAAAANuaWwJAQAAAAtUcmFuc2ZlclNldAAAAAEFAAAAA25pbAAAAAAonzwX"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "main", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.StringDataEntry{Key: "mainLog", Value: "1FCQFaXp6A3s2po6M3iP3ECkjzjMojE5hNA1s8NyvxzgY - 3N4XM8G5WXzdkLXYDL6X229Entc5Hqgz7DM - 1FCQFaXp6A3s2po6M3iP3ECkjzjMojE5hNA1s8NyvxzgY -> 3NBQxw1ZzTfWbrLjWj2euMwizncrGG4nXJX"}},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

func TestDropElementDApp(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("HsezR6axe1twb8DNV8oo5Kqus5ZTDLLPbCqVvvHzaELS")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("3nRXeLFr9exB4fFCb6wtQKYaJ4jfcY6FziZSjMrx3baYHuQqFmpz23iNwxdNHUojgpdSRbdg33t3PGvYfWSWMLmJ")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("HGT44HrsSSD5cjANV6wtWNB9VKS3y7hhoNXEDWB56Lu9")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3MpG8hTQfgrXcavZYWYaBcT31FUonRAXfYS")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "aaa,bbb,ccc"})
	arguments.Append(&proto.StringArgument{Value: "ccc"})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "dropElementInArray",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.TestNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1573578115548,
	}
	gs := crypto.MustBytesFromBase58("AWH9QVEnmN6VjRyEfs93UtAiCkwrNJ2phKYe25KFNCz")
	gen, err := proto.NewAddressFromString("3MxTeL8dKLUGh9B1A2aaZxQ8BLL22bDdm6G")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1567938316714,
		Height:              762110,
		BaseTarget:          1550,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender,
	}

	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 1642207
		},
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		takeStringFunc: v5takeString,
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 666972, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					return nil, errors.New("fail")
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "AAIDAAAAAAAAAAgIARIECgIICAAAAAEBAAAAFmRyb3BFbGVtZW50SW5Kc29uQXJyYXkAAAACAAAABWFycmF5AAAAB2VsZW1lbnQEAAAADHNwbGl0ZWRBcnJheQkABLUAAAACBQAAAAVhcnJheQUAAAAHZWxlbWVudAMJAAAAAAAAAgkAAS8AAAACCQABkQAAAAIFAAAADHNwbGl0ZWRBcnJheQAAAAAAAAAAAQAAAAAAAAAAAQIAAAABLAkAASwAAAACCQABkQAAAAIFAAAADHNwbGl0ZWRBcnJheQAAAAAAAAAAAAkAATAAAAACCQABkQAAAAIFAAAADHNwbGl0ZWRBcnJheQAAAAAAAAAAAQAAAAAAAAAAAQkAASwAAAACCQEAAAAJZHJvcFJpZ2h0AAAAAgkAAZEAAAACBQAAAAxzcGxpdGVkQXJyYXkAAAAAAAAAAAAAAAAAAAAAAAEJAAGRAAAAAgUAAAAMc3BsaXRlZEFycmF5AAAAAAAAAAABAAAAAQAAAAJ0eAEAAAASZHJvcEVsZW1lbnRJbkFycmF5AAAAAgAAAAVhcnJheQAAAAdlbGVtZW50BAAAAAluZXh0SWRPcHQJAAQaAAAAAgUAAAAEdGhpcwIAAAAGTkVYVElEBAAAAAZuZXh0SWQDCQEAAAAJaXNEZWZpbmVkAAAAAQUAAAAJbmV4dElkT3B0CQEAAAAHZXh0cmFjdAAAAAEFAAAACW5leHRJZE9wdAAAAAAAAAAAAQQAAAASYXJyYXlXaXRob3RFbGVtZW50CQEAAAAWZHJvcEVsZW1lbnRJbkpzb25BcnJheQAAAAIFAAAABWFycmF5BQAAAAdlbGVtZW50CQEAAAAIV3JpdGVTZXQAAAABCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACCQABpAAAAAEFAAAABm5leHRJZAkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACBQAAAAVhcnJheQIAAAADIC0gBQAAAAdlbGVtZW50AgAAAAMgPSAFAAAAEmFycmF5V2l0aG90RWxlbWVudAUAAAADbmlsAAAAANx44LU="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "dropElementInArray", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.StringDataEntry{Key: "1", Value: "aaa,bbb,ccc - ccc = aaa,bbb"}},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

func TestMathDApp(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("BWgVRTzu4tBZ7n4NNLYTJYiJYLFeD9YjLisTVezzPYMw")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("ooa1Ep1enfPki3khQvyGRdhJ1w3TFsH62fRDx9m9wuZaBWG1EqqSwvgzyMt3fPEJRgJvmHpmgBNJV5oCGL6AhPL")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("HGT44HrsSSD5cjANV6wtWNB9VKS3y7hhoNXEDWB56Lu9")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3MvQVj21fwPXbyXsrVDV2Sf639TcWTsaxmC")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	arguments.Append(&proto.IntegerArgument{Value: 92})
	arguments.Append(&proto.IntegerArgument{Value: 1000})
	arguments.Append(&proto.IntegerArgument{Value: 970})
	arguments.Append(&proto.IntegerArgument{Value: 6})
	arguments.Append(&proto.IntegerArgument{Value: 20})
	arguments.Append(&proto.IntegerArgument{Value: 4})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "coxRossRubinsteinCall",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.TestNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             900000,
		Timestamp:       1578475562553,
	}
	gs := crypto.MustBytesFromBase58("AWH9QVEnmN6VjRyEfs93UtAiCkwrNJ2phKYe25KFNCz")
	gen, err := proto.NewAddressFromString("3MxTeL8dKLUGh9B1A2aaZxQ8BLL22bDdm6G")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1567938316714,
		Height:              844761,
		BaseTarget:          1550,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender,
	}

	env := &mockRideEnvironment{
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		heightFunc: func() rideInt {
			return 1642207
		},
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 844761, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		validateInternalPaymentsFunc: func() bool {
			return false
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "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"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallFunction(env, tree, "coxRossRubinsteinCall", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))

	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "deltaT", Value: 6301369}},
		{Entry: &proto.IntegerDataEntry{Key: "sqrtDeltaT", Value: 25102528}},
		{Entry: &proto.IntegerDataEntry{Key: "up", Value: 105148668}},
		{Entry: &proto.IntegerDataEntry{Key: "down", Value: 95103439}},
		{Entry: &proto.IntegerDataEntry{Key: "df", Value: 99622632}},
		{Entry: &proto.IntegerDataEntry{Key: "pUp", Value: 52516065}},
		{Entry: &proto.IntegerDataEntry{Key: "pDown", Value: 47483935}},
		{Entry: &proto.IntegerDataEntry{Key: "firstProjectedPrice", Value: 0}},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

func TestDAppWithInvalidAddress(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("HuonmGzdTvYFbD5q9pn32zo8zYJPyuSbnJBQJ6TtZXNd")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("4FqjzEtyjhhutG2sczrsdwXcCAv8bBfRDyYAt67P6rVcnERd8fa9saMzegyfa5d5LvgXDCpLWV6oYREsu2VJKUCX")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("CneM2DD58Xtnnyee8sWDCafU1vPsoLhVgTvGJtPHaou6")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3N5jpkcHiH5R36y9cYnoXhVHe4pxRkS3peF")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "deposit",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.TestNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{proto.ScriptPayment{Amount: 100000000, Asset: proto.OptionalAsset{}}},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1583951694368,
	}
	gs := crypto.MustBytesFromBase58("AWH9QVEnmN6VjRyEfs93UtAiCkwrNJ2phKYe25KFNCz")
	gen, err := proto.NewAddressFromString("3MxTeL8dKLUGh9B1A2aaZxQ8BLL22bDdm6G")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1567938316714,
		Height:              844761,
		BaseTarget:          1550,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender,
	}

	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 844761
		},
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 844761, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					switch key {
					case "3MwT5r4YSyG4QAiqi8VNZkL9eP9e354DXfE_waves":
						return &proto.IntegerDataEntry{Key: "3MwT5r4YSyG4QAiqi8VNZkL9eP9e354DXfE_waves", Value: 6012000}, nil
					default:
						return nil, errors.New("fail")
					}
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(3, proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "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"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "deposit", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))

	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "3MwT5r4YSyG4QAiqi8VNZkL9eP9e354DXfE_waves", Value: 7012000}},
	}
	a, err := proto.NewAddressFromString("3MwT5r4YSyG4QAiqi8VNZkL9eP9e354DXfE")
	require.NoError(t, err)
	asset, err := proto.NewOptionalAssetFromString("Ea6CdDfumo8ZFecxUSWKAjZpQXmkRC79WB4ktu3KffPn")
	require.NoError(t, err)
	expectedTransfers := []*proto.TransferScriptAction{
		{Recipient: proto.NewRecipientFromAddress(a), Amount: 0, Asset: *asset},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    expectedTransfers,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

func Test8Ball(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("6zUFtrHoWpzVoGcW1eqxQptoYpv3WSMDFjwpU7CtdgDn")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("4x5AEuTj5yhaJQrE8YXUKg9Bc2n5GtdfG7bbrhXqB6wro9AcAtQH4ZgDFMawp5jLVcp3yesJxQ53ALVZTZjkeaWY")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("4KxkQHV5VP5a5tm5ETSEj78r9JfLUPFqZFmnQz1q878Y")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3N27HUMt4ddx2X7foQwZRmpFzg5PSzLrUgU")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	arguments.Append(&proto.StringArgument{Value: "What is my purpose?"})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "tellme",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.TestNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1577191068093,
	}
	gs := crypto.MustBytesFromBase58("AWH9QVEnmN6VjRyEfs93UtAiCkwrNJ2phKYe25KFNCz")
	gen, err := proto.NewAddressFromString("3MxTeL8dKLUGh9B1A2aaZxQ8BLL22bDdm6G")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1567938316714,
		Height:              844761,
		BaseTarget:          1550,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender,
	}
	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 844761
		},
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 844761, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					switch key {
					case "3Mz67eGY4aNdBHJtgbRPVde3KwAeN3ULLHG_q":
						return &proto.StringDataEntry{Key: "3Mz67eGY4aNdBHJtgbRPVde3KwAeN3ULLHG_q", Value: "What is my purpose?"}, nil
					case "3Mz67eGY4aNdBHJtgbRPVde3KwAeN3ULLHG_a":
						return &proto.StringDataEntry{Key: "3Mz67eGY4aNdBHJtgbRPVde3KwAeN3ULLHG_a", Value: "You may rely on it."}, nil
					default:
						return nil, errors.New("fail")
					}
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(3, proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		checkMessageLengthFunc: v3check,
		rideV6ActivatedFunc:    noRideV6,
	}

	code := "AAIDAAAAAAAAAAAAAAAEAAAAAAxhbnN3ZXJzQ291bnQAAAAAAAAAABQAAAAAB2Fuc3dlcnMJAARMAAAAAgIAAAAOSXQgaXMgY2VydGFpbi4JAARMAAAAAgIAAAATSXQgaXMgZGVjaWRlZGx5IHNvLgkABEwAAAACAgAAABBXaXRob3V0IGEgZG91YnQuCQAETAAAAAICAAAAEVllcyAtIGRlZmluaXRlbHkuCQAETAAAAAICAAAAE1lvdSBtYXkgcmVseSBvbiBpdC4JAARMAAAAAgIAAAARQXMgSSBzZWUgaXQsIHllcy4JAARMAAAAAgIAAAAMTW9zdCBsaWtlbHkuCQAETAAAAAICAAAADU91dGxvb2sgZ29vZC4JAARMAAAAAgIAAAAEWWVzLgkABEwAAAACAgAAABNTaWducyBwb2ludCB0byB5ZXMuCQAETAAAAAICAAAAFlJlcGx5IGhhenksIHRyeSBhZ2Fpbi4JAARMAAAAAgIAAAAQQXNrIGFnYWluIGxhdGVyLgkABEwAAAACAgAAABhCZXR0ZXIgbm90IHRlbGwgeW91IG5vdy4JAARMAAAAAgIAAAATQ2Fubm90IHByZWRpY3Qgbm93LgkABEwAAAACAgAAABpDb25jZW50cmF0ZSBhbmQgYXNrIGFnYWluLgkABEwAAAACAgAAABJEb24ndCBjb3VudCBvbiBpdC4JAARMAAAAAgIAAAAPTXkgcmVwbHkgaXMgbm8uCQAETAAAAAICAAAAEk15IHNvdXJjZXMgc2F5IG5vLgkABEwAAAACAgAAABRPdXRsb29rIG5vdCBzbyBnb29kLgkABEwAAAACAgAAAA5WZXJ5IGRvdWJ0ZnVsLgUAAAADbmlsAQAAAAlnZXRBbnN3ZXIAAAACAAAACHF1ZXN0aW9uAAAADnByZXZpb3VzQW5zd2VyBAAAAARoYXNoCQAB9wAAAAEJAAGbAAAAAQkAASwAAAACBQAAAAhxdWVzdGlvbgUAAAAOcHJldmlvdXNBbnN3ZXIEAAAABWluZGV4CQAEsQAAAAEFAAAABGhhc2gJAAGRAAAAAgUAAAAHYW5zd2VycwkAAGoAAAACBQAAAAVpbmRleAUAAAAMYW5zd2Vyc0NvdW50AQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEAAAAHYWRkcmVzcwQAAAAHJG1hdGNoMAkABB0AAAACBQAAAAR0aGlzCQABLAAAAAIFAAAAB2FkZHJlc3MCAAAAAl9hAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAAZTdHJpbmcEAAAAAWEFAAAAByRtYXRjaDAFAAAAAWEFAAAAB2FkZHJlc3MAAAABAAAAAWkBAAAABnRlbGxtZQAAAAEAAAAIcXVlc3Rpb24EAAAADWNhbGxlckFkZHJlc3MJAAJYAAAAAQgIBQAAAAFpAAAABmNhbGxlcgAAAAVieXRlcwQAAAAGYW5zd2VyCQEAAAAJZ2V0QW5zd2VyAAAAAgUAAAAIcXVlc3Rpb24JAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEFAAAADWNhbGxlckFkZHJlc3MJAQAAAAhXcml0ZVNldAAAAAEJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAIJAAEsAAAAAgUAAAANY2FsbGVyQWRkcmVzcwIAAAACX3EFAAAACHF1ZXN0aW9uCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACCQABLAAAAAIFAAAADWNhbGxlckFkZHJlc3MCAAAAAl9hBQAAAAZhbnN3ZXIFAAAAA25pbAAAAACOjDZR"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "tellme", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))

	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.StringDataEntry{Key: "3Mz67eGY4aNdBHJtgbRPVde3KwAeN3ULLHG_q", Value: "What is my purpose?"}},
		{Entry: &proto.StringDataEntry{Key: "3Mz67eGY4aNdBHJtgbRPVde3KwAeN3ULLHG_a", Value: "Yes - definitely."}},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

func TestIntegerEntry(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("AjSkRGMhckj4bhwtLPyeSTeDY6unoDwjs736t2bNvV3D")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("2g1hQJKw1Mzc7Qpw8WzzheDibi34JWATTsV1m39GPGJc1oz1DH82RRFnHkp1QEMg7ccH3K71YFLuK1GrHrrnfEjJ")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("Ccebak7uPmCpdNGrVTxENghcrCLF7m9MXGA2BbMDknoW")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3MouSkYhyvLXkn9wYRcqHUrhcDgNipSGFQN")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	arguments.Append(&proto.IntegerArgument{Value: 1})
	arguments.Append(&proto.StringArgument{Value: "Hi!!! hello!"})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "call",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.StageNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1588047474869,
	}
	gs := crypto.MustBytesFromBase58("AWH9QVEnmN6VjRyEfs93UtAiCkwrNJ2phKYe25KFNCz")
	gen, err := proto.NewAddressFromString("3MxTeL8dKLUGh9B1A2aaZxQ8BLL22bDdm6G")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1567938316714,
		Height:              844761,
		BaseTarget:          1550,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender,
	}
	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 844761
		},
		schemeFunc: func() byte {
			return proto.StageNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 386529, nil
				},
				NewestAssetIsSponsoredFunc: func(asset crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.StageNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		rideV6ActivatedFunc: noRideV6,
	}

	code := "AAIEAAAAAAAAAAgIAhIECgIIAQAAAAAAAAABAAAAAWkBAAAABGNhbGwAAAACAAAAA25vbQAAAANhZ2UEAAAADG93bmVyQWRkcmVzcwkABCUAAAABCAUAAAABaQAAAAZjYWxsZXIJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAIJAAEsAAAAAgUAAAAMb3duZXJBZGRyZXNzAgAAAARfYWdlBQAAAANhZ2UJAARMAAAAAgkBAAAAC1N0cmluZ0VudHJ5AAAAAgkAASwAAAACBQAAAAxvd25lckFkZHJlc3MCAAAABF9ub20FAAAAA25vbQUAAAADbmlsAAAAAHNCMbc="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	_, err = CallFunction(env, tree, "tellme", arguments)
	assert.Error(t, err)
}

func TestAssetInfoV3V4(t *testing.T) {
	pk, err := crypto.NewPublicKeyFromBase58("Ccebak7uPmCpdNGrVTxENghcrCLF7m9MXGA2BbMDknoW")
	require.NoError(t, err)
	issuer, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, pk)
	require.NoError(t, err)
	assetID1, err := crypto.NewDigestFromBase58("4njdbzZQNBSPgU2WWPfcKEnUbFvSKTHQBRdGk2mJJ9ye")
	require.NoError(t, err)
	info := proto.AssetInfo{
		ID:              assetID1,
		Quantity:        1000,
		Decimals:        0,
		Issuer:          issuer,
		IssuerPublicKey: pk,
		Reissuable:      false,
		Scripted:        false,
		Sponsored:       false,
	}
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				NewestAssetBalanceFunc: func(account proto.Recipient, asset crypto.Digest) (uint64, error) {
					return 1000, nil
				},
				NewestAssetInfoFunc: func(asset crypto.Digest) (*proto.AssetInfo, error) {
					return &info, nil
				},
				NewestAssetIsSponsoredFunc: func(asset crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullAssetInfoFunc: func(asset crypto.Digest) (*proto.FullAssetInfo, error) {
					return &proto.FullAssetInfo{
						AssetInfo:   info,
						Name:        "ASSET1",
						Description: "DESCRIPTION1",
					}, nil
				},
			}
		},
		rideV6ActivatedFunc: noRideV6,
	}

	codeV3 := "AwQAAAACYWkJAQAAAAdleHRyYWN0AAAAAQkAA+wAAAABAQAAACA4SmZ7I8ecZ8q8rkkn9snzZVVjpJyyIfolCl2dP60I7QkAAAAAAAACCAUAAAACYWkAAAACaWQBAAAAIDhKZnsjx5xnyryuSSf2yfNlVWOknLIh+iUKXZ0/rQjthFBV8Q=="
	srcV3, err := base64.StdEncoding.DecodeString(codeV3)
	require.NoError(t, err)

	treeV3, err := Parse(srcV3)
	require.NoError(t, err)
	assert.NotNil(t, treeV3)

	res, err := CallVerifier(env, treeV3)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	assert.True(t, r.Result())

	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		let ai =  value(assetInfo(base58'4njdbzZQNBSPgU2WWPfcKEnUbFvSKTHQBRdGk2mJJ9ye'))
		ai.name == "ASSET1" && ai.description == "DESCRIPTION1"
	*/
	codeV4 := "BAQAAAACYWkJAQAAAAV2YWx1ZQAAAAEJAAPsAAAAAQEAAAAgOEpmeyPHnGfKvK5JJ/bJ82VVY6ScsiH6JQpdnT+tCO0DCQAAAAAAAAIIBQAAAAJhaQAAAARuYW1lAgAAAAZBU1NFVDEJAAAAAAAAAggFAAAAAmFpAAAAC2Rlc2NyaXB0aW9uAgAAAAxERVNDUklQVElPTjEHchuBRQ=="
	srcV4, err := base64.StdEncoding.DecodeString(codeV4)
	require.NoError(t, err)

	treeV4, err := Parse(srcV4)
	require.NoError(t, err)
	assert.NotNil(t, treeV3)

	res, err = CallVerifier(env, treeV4)
	require.NoError(t, err)
	r, ok = res.(ScriptResult)
	require.True(t, ok)
	assert.True(t, r.Result())
}

func TestJSONParsing(t *testing.T) {
	env := &mockRideEnvironment{
		takeStringFunc:      v5takeString,
		rideV6ActivatedFunc: noRideV6,
	}

	code := "AwoBAAAADmdldFZhbHVlU3RyaW5nAAAAAQAAAARqc29uCQABLwAAAAIJAAEwAAAAAgUAAAAEanNvbgAAAAAAAAAAAQkBAAAABXZhbHVlAAAAAQkABLMAAAACCQABMAAAAAIFAAAABGpzb24AAAAAAAAAAAECAAAAASIKAQAAAAhnZXRWYWx1ZQAAAAIAAAAEanNvbgAAAANrZXkEAAAACGtleUluZGV4CQEAAAAFdmFsdWUAAAABCQAEswAAAAIFAAAABGpzb24JAAEsAAAAAgkAASwAAAACAgAAAAEiBQAAAANrZXkCAAAAAiI6BAAAAARkYXRhCQABMAAAAAIFAAAABGpzb24JAABkAAAAAgkAAGQAAAACBQAAAAhrZXlJbmRleAkAATEAAAABBQAAAANrZXkAAAAAAAAAAAMJAQAAAA5nZXRWYWx1ZVN0cmluZwAAAAEFAAAABGRhdGEEAAAACWFkZHJlc3NlcwIAAAFgeyJ0aXRsZSI6Ikjhu6NwIMSR4buTbmcgbXVhIGLDoW4gxJHhuqV0IChyZWFsLWVzdGF0ZSBjb250cmFjdCkiLCJ0aW1lc3RhbXAiOjE1OTE2MDg5NDQzNTQsImhhc2giOiJkOGYwOWFjYmRlYTIwMTc5MTUyY2Q5N2RiNDNmNmJjZjhjYjYxMTE1YmE3YzNmZWU3NDk4MWU0ZjRiNTBlNGEwIiwiY3JlYXRvciI6IiIsImFkZHJlc3MxIjoiM015Yjg1REd2N3hqNFhaRlpBTDRHSHVHRG1aU0czQ0NVdlciLCJhZGRyZXNzMiI6IiIsImFkZHJlc3MzIjoiIiwiYWRkcmVzczQiOiIiLCJhZGRyZXNzNSI6IiIsImFkZHJlc3M2IjoiIiwiaXBmcyI6IlFtVEtCbUg5aW4yRU50NkFRcnZwUHpvYWFtMnozcWRFZUhRU1k5M3JkOEpqSFkifQkAAAAAAAACCQEAAAAIZ2V0VmFsdWUAAAACBQAAAAlhZGRyZXNzZXMCAAAACGFkZHJlc3MxAgAAACMzTXliODVER3Y3eGo0WFpGWkFMNEdIdUdEbVpTRzNDQ1V2V6k+k0o="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallVerifier(env, tree)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	assert.True(t, r.Result())
}

func TestDAppWithFullIssue(t *testing.T) {
	code := "AAIEAAAAAAAAAAcIAhIDCgEIAAAAAAAAAAEAAAABaQEAAAAFaXNzdWUAAAABAAAABG5hbWUJAARMAAAAAgkABEMAAAAHBQAAAARuYW1lAgAAAAtkZXNjcmlwdGlvbgAAAAAAAAGGoAAAAAAAAAAAAgYFAAAABHVuaXQAAAAAAAAAAAAFAAAAA25pbAAAAABNz7Zz"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	id := bytes.Repeat([]byte{0}, 32)
	env := &mockRideEnvironment{
		txIDFunc: func() rideType {
			return rideBytes(id)
		},
		rideV6ActivatedFunc: noRideV6,
	}
	res, err := CallFunction(env, tree, "issue", proto.Arguments{&proto.StringArgument{Value: "xxx"}})
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)
	assert.Equal(t, 1, len(r.ScriptActions()))
	a := r.ScriptActions()[0]
	issue, ok := a.(*proto.IssueScriptAction)
	assert.True(t, ok)
	assert.Equal(t, "xxx", issue.Name)
}

func TestDAppWithSimpleIssue(t *testing.T) {
	code := "AAIEAAAAAAAAAAcIAhIDCgEIAAAAAAAAAAEAAAABaQEAAAAFaXNzdWUAAAABAAAABG5hbWUJAARMAAAAAgkABEIAAAAFBQAAAARuYW1lAgAAAAtkZXNjcmlwdGlvbgAAAAAAAAGGoAAAAAAAAAAAAgYFAAAAA25pbAAAAAAOKB/n"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	id := bytes.Repeat([]byte{0}, 32)
	env := &mockRideEnvironment{
		txIDFunc: func() rideType {
			return rideBytes(id)
		},
		rideV6ActivatedFunc: noRideV6,
	}
	res, err := CallFunction(env, tree, "issue", proto.Arguments{&proto.StringArgument{Value: "xxx"}})
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)
	assert.Equal(t, 1, len(r.ScriptActions()))
	a := r.ScriptActions()[0]
	issue, ok := a.(*proto.IssueScriptAction)
	assert.True(t, ok)
	assert.Equal(t, "xxx", issue.Name)
}

func TestBadType(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("E3veSYzwJJEYEjgYAWV5vTi4TjsqbGuFDnCxSmrHmQXB")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5Hr1wmxEpHV4YHxB3yEFM1mMhRcKq9BQJLz4fmcHKN9BdmAcwaMLUqP7XPkTdAvc6wNJjgeug5u1A1cujvLKLKUc")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("FCaP4jLhLawzEqbwAQGAVvPQBv2h3LdERCx7fckDvnzr")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3MrFYvH6tMTTg1wxC7CdWDtHgovrTQjyaXs")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	arguments.Append(&proto.IntegerArgument{Value: 1})
	arguments.Append(&proto.IntegerArgument{Value: 3})
	arguments.Append(&proto.IntegerArgument{Value: 1})
	call := proto.FunctionCall{
		Default:   false,
		Name:      "initDraw",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.TestNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{proto.ScriptPayment{Amount: 13500000, Asset: proto.OptionalAsset{}}},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1565026317983,
	}
	gs := crypto.MustBytesFromBase58("AWH9QVEnmN6VjRyEfs93UtAiCkwrNJ2phKYe25KFNCz")
	gen, err := proto.NewAddressFromString("3MxTeL8dKLUGh9B1A2aaZxQ8BLL22bDdm6G")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1567938316714,
		Height:              617907,
		BaseTarget:          1550,
		GenerationSignature: gs,
		Generator:           gen,
		GeneratorPublicKey:  sender,
	}
	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 617907
		},
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 617907, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(3, proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		checkMessageLengthFunc: v3check,
		rideV6ActivatedFunc:    noRideV6,
	}

	code := "AAIDAAAAAAAAAAAAAAAaAAAAAAlSU0FQVUJMSUMJAAJbAAAAAQIAAAGPYmFzZTY0Ok1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBbXB1WGNJL280cElCNXl3djlET09HYXBUQlV3UlZsTS82K0g2aEZlbE9YdGtyd1kvWUl0bVB4RURwejdyQWVyUVBRZTl0RFBFYUF2L0dubEV6dHliT0ZYZ3U5RHpEZThZb01SRDF2YWtnb0Fjb2dtYlk1OFFENktNajVIa29Wai95VE5JYzlzemo1cWhJbHJBZG1iM0tMTDZoUVU3eTgrSmo2OUJXVlBzYVFna3NwU2RlWXRiMXRIUWM3dDk1bjdPWjU2cjJBN0czK2JRZjZuU01rUGtBaElyRXBiQ201OG9pR0JjemRUZC9McUZTVm90WnNiTDdZaDZTSExmbkhlRCtRZ2NmSnJuYW04T0hNR0pFSlRSWGpJTGVIR2psUkNQOG9WcGlvSHJ5MVMyeFB4NXNWekltMk1NK0N6WWVuQUdsbzBqMjZhdEJoaVVMb1R1bHdEM3BRSURBUUFCAAAAAAZTRVJWRVIJAQAAABxAZXh0clVzZXIoYWRkcmVzc0Zyb21TdHJpbmcpAAAAAQIAAAAjM05DaUcyOExtV3lUaWdXRzEzRTVRbnZkSEJzWkZZWFNTMmoAAAAAB1dBVkVMRVQJAABoAAAAAgkAAGgAAAACAAAAAAAAAABkAAAAAAAAAAPoAAAAAAAAAAPoAAAAABBTRVNTSU9OSURGSVhTSVpFAAAAAAAAAAAsAAAAAA5SQU5EQ1lDTEVQUklDRQkAAGkAAAACCQAAaAAAAAIAAAAAAAAAAAUFAAAAB1dBVkVMRVQAAAAAAAAAA+gAAAAAEE1BWFJBTkRTUEVSQ1lDTEUAAAAAAAAAAA4AAAAACVNUQVRFSU5JVAIAAAAESU5JVAAAAAAIREFUQURPTkUCAAAABVJFQURZAAAAAA1TVEFURUZJTklTSEVEAgAAAAhGSU5JU0hFRAAAAAAISWR4U3RhdGUAAAAAAAAAAAAAAAAAD0lkeE9yZ2FuaXplclB1YgAAAAAAAAAAAQAAAAALSWR4UmFuZEZyb20AAAAAAAAAAAIAAAAACUlkeFJhbmRUbwAAAAAAAAAAAwAAAAANSWR4UmFuZHND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AA9jYWxsZXJQdWJsaWNLZXkEAAAADXJhbmRzQ291bnRTdHIJAAGkAAAAAQUAAAAKcmFuZHNDb3VudAQAAAAJaW5pdFN0YXRlCQEAAAASZm9ybWF0U3RhdGVEYXRhU3RyAAAACgUAAAAJU1RBVEVJTklUBQAAABFvcmdhbml6ZXJQdWJLZXk1OAkAAaQAAAABBQAAAAhyYW5kRnJvbQkAAaQAAAABBQAAAAZyYW5kVG8FAAAADXJhbmRzQ291bnRTdHIFAAAADXJhbmRzQ291bnRTdHIJAAGkAAAAAQUAAAANZGF0YUtleXNDb3VudAIAAAAEbnVsbAIAAAABMAIAAAAACQEAAAAMU2NyaXB0UmVzdWx0AAAAAgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgUAAAAJc2Vzc2lvbklkBQAAAAlpbml0U3RhdGUFAAAAA25pbAkBAAAAC1RyYW5zZmVyU2V0AAAAAQkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADBQAAAAZTRVJWRVIFAAAACHJhbmRzUG10BQAAAAR1bml0BQAAAANuaWwAAAABaQEAAAAFcmVhZHkAAAABAAAACXNlc3Npb25JZAQAAAAOZHJhd1BhcmFtc0xpc3QJAQAAABNleHRyYWN0R2FtZURhdGFMaXN0AAAAAQUAAAAJc2Vzc2lvbklkBAAAAAlkcmF3U3RhdGUJAAGRAAAAAgUAAAAOZHJhd1BhcmFtc0xpc3QFAAAACElkeFN0YXRlBAAAABFvcmdhbml6ZXJQdWJLZXk1OAkAAZEAAAACBQAAAA5kcmF3UGFyYW1zTGlzdAUAAAAPSWR4T3JnYW5pemVyUHViBAAAAA1yYW5kc0NvdW50U3RyCQABkQAAAAIFAAAADmRyYXdQYXJhbXNMaXN0BQAAAA1JZHhSYW5kc0NvdW50BAAAABNyZW1haW5SYW5kc0NvdW50U3RyCQABkQAAAAIFAAAADmRyYXdQYXJhbXNMaXN0BQAAABNJZHhSZW1haW5SYW5kc0NvdW50BAAAAAdmcm9tU3RyCQABkQAAAAIFAAAADmRyYXdQYXJhbXNMaXN0BQAAAAtJZHhSYW5kRnJvbQQAAAAFdG9TdHIJAAGRAAAAAgUAAAAOZHJhd1BhcmFtc0xpc3QFAAAACUlkeFJhbmRUbwQAAAANZGF0YUtleXNDb3VudAkAAZEAAAACBQAAAA5kcmF3UGFyYW1zTGlzdAUAAAAQSWR4RGF0YUtleXNDb3VudAQAAAAPb3JnYW5pemVyUHViS2V5CQACWQAAAAEFAAAAEW9yZ2FuaXplclB1YktleTU4AwkBAAAAAiE9AAAAAgUAAAAJZHJhd1N0YXRlBQAAAAlTVEFURUlOSVQJAAACAAAAAQkAASwAAAACBQAAAAlzZXNzaW9uSWQCAAAAPiBkcmF3OiBtb3ZpbmcgaW50byBSRUFEWSBzdGF0ZSBpcyBhbGxvd2VkIG9ubHkgZnJvbSBJTklUIHN0YXRlAwkBAAAAAiE9AAAAAgUAAAAPb3JnYW5pemVyUHViS2V5CAUAAAABaQAAAA9jYWxsZXJQdWJsaWNLZXkJAAACAAAAAQkAASwAAAACBQAAAAlzZXNzaW9uSWQCAAAAO2RyYXc6IG1vdmluZyBpbnRvIFJFQURZIHN0YXRlIGlzIGFsbG93ZWQgZm9yIG9yZ2FuaXplciBvbmx5BAAAAApyZWFkeVN0YXRlCQEAAAASZm9ybWF0U3RhdGVEYXRhU3RyAAAACgUAAAAIREFUQURPTkUFAAAAEW9yZ2FuaXplclB1YktleTU4BQAAAAdmcm9tU3RyBQAAAAV0b1N0cgUAAAANcmFuZHNDb3VudFN0cgUAAAATcmVtYWluUmFuZHNDb3VudFN0cgUAAAANZGF0YUtleXNDb3VudAkAAlgAAAABCAUAAAABaQAAAA10cmFuc2FjdGlvbklkAgAAAAEwAgAAAAAJAQAAAAhXcml0ZVNldAAAAAEJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAIFAAAACXNlc3Npb25JZAUAAAAKcmVhZHlTdGF0ZQUAAAADbmlsAAAAAWkBAAAABnJhbmRvbQAAAAIAAAAJc2Vzc2lvbklkAAAAB3JzYVNpZ24EAAAADmRyYXdQYXJhbXNMaXN0CQEAAAATZXh0cmFjdEdhbWVEYXRhTGlzdAAAAAEFAAAACXNlc3Npb25JZAQAAAAJZHJhd1N0YXRlCQABkQAAAAIFAAAADmRyYXdQYXJhbXNMaXN0BQAAAAhJZHhTdGF0ZQQAAAARb3JnYW5pemVyUHViS2V5NTgJAAGRAAAAAgUAAAAOZHJhd1BhcmFtc0xpc3QFAAAAD0lkeE9yZ2FuaXplclB1YgQAAAANcmFuZHNDb3VudFN0cgkAAZEAAAACBQAAAA5kcmF3UGFyYW1zTGlzdAUAAAANSWR4UmFuZHNDb3VudAQAAAAQcmVtYWluUmFuZHNDb3VudAkBAAAADXBhcnNlSW50VmFsdWUAAAABCQABkQAAAAIFAAAADmRyYXdQYXJhbXNMaXN0BQAAABNJZHhSZW1haW5SYW5kc0NvdW50BAAAAA9sYXN0T2Zmc2V0Qnl0ZXMJAQAAAA1wYXJzZUludFZhbHVlAAAAAQkAAZEAAAACBQAAAA5kcmF3UGFyYW1zTGlzdAUAAAANSWR4TGFzdE9mZnNldAQAAAAMY3VyclJhbmRzU3RyCQABkQAAAAIFAAAADmRyYXdQYXJhbXNMaXN0BQAAAAxJZHhDdXJyUmFuZHMEAAAAB2Zyb21TdHIJAAGRAAAAAgUAAAAOZHJhd1BhcmFtc0xpc3QFAAAAC0lkeFJhbmRGcm9tBAAAAAV0b1N0cgkAAZEAAAACBQAAAA5kcmF3UGFyYW1zTGlzdAUAAAAJSWR4UmFuZFRvBAAAAAxkYXRhRG9uZVR4SWQJAAGRAAAAAgUAAAAOZHJhd1BhcmFtc0xpc3QFAAAAD0lkeERhdGFEb25lVHhJZAQAAAANZGF0YUtleXNDb3VudAkAAZEAAAACBQAAAA5kcmF3UGFyYW1zTGlzdAUAAAAQSWR4RGF0YUtleXNDb3VudAQAAAAEZnJvbQkBAAAADXBhcnNlSW50VmFsdWUAAAABBQAAAAdmcm9tU3RyBAAAAAJ0bwkBAAAADXBhcnNlSW50VmFsdWUAAAABBQAAAAV0b1N0cgQAAAAPb3JnYW5pemVyUHViS2V5CQACWQAAAAEFAAAAEW9yZ2FuaXplclB1YktleTU4AwkBAAAAAiE9AAAAAgUAAAAJZHJhd1N0YXRlBQAAAAhEQVRBRE9ORQkAAAIAAAABCQABLAAAAAIFAAAACXNlc3Npb25JZAIAAAA7IGRyYXc6IGl0IG11c3QgYmUgaW4gUkVBRFkgc3RhdGUgdG8gZ2VuZXJhdGUgcmFuZG9tIG51bWJlcnMDCQEAAAABIQAAAAEJAAH4AAAABAUAAAAGU0hBMjU2CQAAywAAAAIJAAGbAAAAAQUAAAAJc2Vzc2lvbklkCQABmwAAAAEFAAAADGRhdGFEb25lVHhJZAUAAAAHcnNhU2lnbgUAAAAJUlNBUFVCTElDCQAAAgAAAAECAAAAFUludmFsaWQgUlNBIHNpZ25hdHVyZQQAAAALcmFuZEdlbkluZm8JAQAAAAxnZW5lcmF0ZVJhbmQAAAAHBQAAAAlzZXNzaW9uSWQFAAAABGZyb20FAAAAAnRvBQAAAAdyc2FTaWduBQAAAAxjdXJyUmFuZHNTdHIFAAAAEHJlbWFpblJhbmRzQ291bnQFAAAAD2xhc3RPZmZzZXRCeXRlcwQAAAALbmV3UmFuZHNTdHIJAAGRAAAAAgUAAAALcmFuZEdlbkluZm8AAAAAAAAAAAAEAAAAFm5ld1JlbWFpblJhbmRzQ291bnRTdHIJAAGRAAAAAgUAAAALcmFuZEdlbkluZm8AAAAAAAAAAAEEAAAADm5ld09mZnNldEJ5dGVzCQABkQAAAAIFAAAAC3JhbmRHZW5JbmZvAAAAAAAAAAACBAAAAAhuZXdTdGF0ZQMJAAAAAAAAAgUAAAAWbmV3UmVtYWluUmFuZHNDb3VudFN0cgIAAAABMAUAAAANU1RBVEVGSU5JU0hFRAUAAAAIREFUQURPTkUJAQAAAAhXcml0ZVNldAAAAAEJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAIFAAAACXNlc3Npb25JZAkBAAAAEmZvcm1hdFN0YXRlRGF0YVN0cgAAAAoFAAAACG5ld1N0YXRlBQAAABFvcmdhbml6ZXJQdWJLZXk1OAUAAAAHZnJvbVN0cgUAAAAFdG9TdHIFAAAADXJhbmRzQ291bnRTdHIFAAAAFm5ld1JlbWFpblJhbmRzQ291bnRTdHIFAAAADWRhdGFLZXlzQ291bnQFAAAADGRhdGFEb25lVHhJZAUAAAAObmV3T2Zmc2V0Qnl0ZXMFAAAAC25ld1JhbmRzU3RyBQAAAANuaWwAAAABAAAAAnR4AQAAAAZ2ZXJpZnkAAAAABAAAAAckbWF0Y2gwBQAAAAJ0eAMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAPRGF0YVRyYW5zYWN0aW9uBAAAAANkdHgFAAAAByRtYXRjaDAEAAAABWRhdGEwCQABkQAAAAIIBQAAAANkdHgAAAAEZGF0YQAAAAAAAAAAAAQAAAAJc2Vzc2lvbklkCQABLwAAAAIIBQAAAAVkYXRhMAAAAANrZXkFAAAAEFNFU1NJT05JREZJWFNJWkUEAAAADmRyYXdQYXJhbXNMaXN0CQEAAAATZXh0cmFjdEdhbWVEYXRhTGlzdAAAAAEFAAAACXNlc3Npb25JZAQAAAAJZHJhd1N0YXRlCQABkQAAAAIFAAAADmRyYXdQYXJhbXNMaXN0BQAAAAhJZHhTdGF0ZQQAAAARb3JnYW5pemVyUHViS2V5NTgJAAGRAAAAAgUAAAAOZHJhd1BhcmFtc0xpc3QFAAAAD0lkeE9yZ2FuaXplclB1YgQAAAANZGF0YUtleXNDb3VudAkBAAAADXBhcnNlSW50VmFsdWUAAAABCQABkQAAAAIFAAAADmRyYXdQYXJhbXNMaXN0BQAAABBJZHhEYXRhS2V5c0NvdW50BAAAAA9vcmdhbml6ZXJQdWJLZXkJAAJZAAAAAQUAAAARb3JnYW5pemVyUHViS2V5NTgEAAAAEGRhdGFFbnRyaWVzQ291bnQJAAGQAAAAAQgFAAAAA2R0eAAAAARkYXRhBAAAAAhzaWdWYWxpZAkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAAFAAAAD29yZ2FuaXplclB1YktleQQAAAANZGF0YVNpemVWYWxpZAkAAAAAAAACBQAAABBkYXRhRW50cmllc0NvdW50AAAAAAAAAAAFBAAAAAlrZXlzVmFsaWQDAwMDCQEAAAAOdmFsaWRhdGVEdHhLZXkAAAADBQAAAAlzZXNzaW9uSWQFAAAADWRhdGFLZXlzQ291bnQFAAAABWRhdGEwCQEAAAAOdmFsaWRhdGVEdHhLZXkAAAADBQAAAAlzZXNzaW9uSWQFAAAADWRhdGFLZXlzQ291bnQJAAGRAAAAAggFAAAAA2R0eAAAAARkYXRhAAAAAAAAAAABBwkBAAAADnZhbGlkYXRlRHR4S2V5AAAAAwUAAAAJc2Vzc2lvbklkBQAAAA1kYXRhS2V5c0NvdW50CQABkQAAAAIIBQAAAANkdHgAAAAEZGF0YQAAAAAAAAAAAgcJAQAAAA52YWxpZGF0ZUR0eEtleQAAAAMFAAAACXNlc3Npb25JZAUAAAANZGF0YUtleXNDb3VudAkAAZEAAAACCAUAAAADZHR4AAAABGRhdGEAAAAAAAAAAAMHCQEAAAAOdmFsaWRhdGVEdHhLZXkAAAADBQAAAAlzZXNzaW9uSWQFAAAADWRhdGFLZXlzQ291bnQJAAGRAAAAAggFAAAAA2R0eAAAAARkYXRhAAAAAAAAAAAEBwMDAwkAAAAAAAACBQAAAAlkcmF3U3RhdGUFAAAACVNUQVRFSU5JVAUAAAAIc2lnVmFsaWQHBQAAAA1kYXRhU2l6ZVZhbGlkBwUAAAAJa2V5c1ZhbGlkBwMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAUU2V0U2NyaXB0VHJhbnNhY3Rpb24EAAAABHNzdHgFAAAAByRtYXRjaDAGAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABNUcmFuc2ZlclRyYW5zYWN0aW9uBAAAAAN0dHgFAAAAByRtYXRjaDAGBz5YAVg="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "initDraw", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))

	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.StringDataEntry{Key: "E3veSYzwJJEYEjgYAWV5vTi4TjsqbGuFDnCxSmrHmQXB", Value: "INIT_FCaP4jLhLawzEqbwAQGAVvPQBv2h3LdERCx7fckDvnzr_1_3_1_1_5_null_0_-"}},
	}
	a, err := proto.NewAddressFromString("3NCiG28LmWyTigWG13E5QnvdHBsZFYXSS2j")
	require.NoError(t, err)
	expectedTransfers := []*proto.TransferScriptAction{
		{Recipient: proto.NewRecipientFromAddress(a), Amount: 13000000, Asset: proto.OptionalAsset{}},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    expectedTransfers,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

func TestNoDeclaration(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("DwkmSLEjhpbR3jiuKx1dbVfTP71QBf7VNcN5B8WaqLuM")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("4YRDY7okPK8hRCJs66Ut9bQS7M6pXkL4iSVtganecx5Q7N747UDNtZQrEndMxKrDU7gb6fAukK2Am25pjR7wUmJk")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("65f71CjreUhgfNxbxHkJ1ESdiSVxf3eNX8eLbqCfvReU")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3P5sWCrmDJzbHFWU8rQqkJ9LZ46SeByaSJi")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "settle",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1606205563973,
	}
	genPK := crypto.MustPublicKeyFromBase58("BDjPpGYcC8ANJSPX7xgprPpp9nioWK6Qpw9PjbekXxav")
	gs := crypto.MustBytesFromBase58("5uvam2VyTJHLaeD8chY1KUcThgs1HpXJqKpCh1NeL2PtcAu41hirDFDz6J2SDfaAPGDbGrEh11ncFMpx5T7ZXVk83cLHy9qWReU9hyzhKct94r8H7bQdKD6HTm3AnME1eMx")
	genAddr, err := proto.NewAddressFromString("3PMj3yGPBEa1Sx9X4TSBFeJCMMaE3wvKR4N")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1606205508343,
		Height:              2342971,
		BaseTarget:          64,
		GenerationSignature: gs,
		Generator:           genAddr,
		GeneratorPublicKey:  genPK,
	}
	stringEntries := map[string]string{
		"BEARId":           "GsxdrWu1tNGbNubEBYRxy1zcHVB4sWJpcmr9Ni2cHBpB",
		"BULLId":           "FpK8CfKqcgpM9xACRkLXpMqp9wXCRBiRNU8xphdWddg",
		"oracle":           "HGVrLrtmJhSigh8z7HZvZPThVzQpT5YsqPDaQia6EreW",
		"poolToken":        "4VDwPimjMR31ofr8qoRZ6nvhTJq7Rf21cZp1c425dUAR",
		"__dbg__bear":      "GsxdrWu1tNGbNubEBYRxy1zcHVB4sWJpcmr9Ni2cHBpB",
		"__dbg__bull":      "FpK8CfKqcgpM9xACRkLXpMqp9wXCRBiRNU8xphdWddg",
		"headPointer":      "qnH16p6PAbWMyJN4pZbnb2PKeM7EwMBpb5C5sLWugmi",
		"mainTokenId":      "DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p",
		"tailPointer":      "qnH16p6PAbWMyJN4pZbnb2PKeM7EwMBpb5C5sLWugmi",
		"defoAssetName":    "EUR",
		"__dbg__requested": "GsxdrWu1tNGbNubEBYRxy1zcHVB4sWJpcmr9Ni2cHBpB",
		"T2ETwL8zFkL2JXCGTuKtsFYJdj3nXh6oWy2X6RZbgeT":  "ISSUE|10000000|GsxdrWu1tNGbNubEBYRxy1zcHVB4sWJpcmr9Ni2cHBpB|68892|3PHRtM6a6VYHD1ehSwDXsuSkdWefEoPRS34|372322|374562|BLPbSmZdSeGchffcMqEyaNtiuC8Bmk7stQFK54m1PGKG",
		"qnH16p6PAbWMyJN4pZbnb2PKeM7EwMBpb5C5sLWugmi":  "UNPOOL|19000000||68978|3PJCXW3XZWr2tTpT5u52cPXcMWVP9AHBC9h|93519731|97723055|",
		"3AgFbqXgKjcDzmEt26yFawUccpKMH2x6TGghXpttTRzL": "POOL|100000000||68828|3PJCXW3XZWr2tTpT5u52cPXcMWVP9AHBC9h|0|9223372036854775807|",
		"BLPbSmZdSeGchffcMqEyaNtiuC8Bmk7stQFK54m1PGKG": "ISSUE|10000000|GsxdrWu1tNGbNubEBYRxy1zcHVB4sWJpcmr9Ni2cHBpB|68894|3PHRtM6a6VYHD1ehSwDXsuSkdWefEoPRS34|368965|377919|qnH16p6PAbWMyJN4pZbnb2PKeM7EwMBpb5C5sLWugmi",
	}
	intEntries := map[string]int64{
		"poolUp":                17030457,
		"poolDwn":               0,
		"leverage":              50,
		"queueSize":             1,
		"bearCollateral":        39381951,
		"bullCollateral":        39381953,
		"bearCirculation":       1082818,
		"bullCirculation":       17363790,
		"feesAccumulated":       20000,
		"poolMainTokenValue":    51216096,
		"poolTokenCirculation":  19875987,
		"lastSettlementPriceId": 68978,
	}
	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 2342971
		},
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 2342971, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					v, ok := stringEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return &proto.StringDataEntry{Key: key, Value: v}, nil
				},
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					v, ok := intEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return &proto.IntegerDataEntry{Key: key, Value: v}, nil
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(4, proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		checkMessageLengthFunc: v3check,
		rideV6ActivatedFunc:    noRideV6,
	}

	code := "AAIEAAAAAAAAADgIAhINCgsICAgICAgICAgBARIDCgEBEgASAwoBARIDCgEIEgQKAggBEgASAwoBARIAEgMKAQESAAAAAGwBAAAAAlNFAAAAAgAAAAFrAAAAAXYJAQAAAAtTdHJpbmdFbnRyeQAAAAIFAAAAAWsFAAAAAXYBAAAAAklFAAAAAgAAAAFrAAAAAXYJAQAAAAxJbnRlZ2VyRW50cnkAAAACBQAAAAFrBQAAAAF2AQAAAAVkZWJ1ZwAAAAIAAAABawAAAAF2CQEAAAACU0UAAAACCQABLAAAAAICAAAAB19fZGJnX18FAAAAAWsFAAAAAXYAAAAABHRlbjYAAAAAAAAPQkAAAAAABHRlbjgAAAAAAAX14QAAAAAAA01BWAB//////////wAAAAARY29uZmlnUHJvdmlkZXJLZXkCAAAADmNvbmZpZ1Byb3ZpZGVyAAAAAA5jb25maWdQcm92aWRlcgQAAAAHJG1hdGNoMAkABB0AAAACBQAAAAR0aGlzBQAAABFjb25maWdQcm92aWRlcktleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAGU3RyaW5nBAAAAAFzBQAAAAckbWF0Y2gwCQEAAAARQGV4dHJOYXRpdmUoMTA2MikAAAABBQAAAAFzBQAAAAR0aGlzAQAAAAZsb2NhbEkAAAACAAAAAWsAAAABZQkBAAAAE3ZhbHVlT3JFcnJvck1lc3NhZ2UAAAACCQAEGgAAAAIFAAAABHRoaXMFAAAAAWsFAAAAAWUBAAAABmxvY2FsUwAAAAIAAAABawAAAAFlCQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAAQdAAAAAgUAAAAEdGhpcwUAAAABawUAAAABZQEAAAAFY29uZkkAAAACAAAAAWsAAAABZQkBAAAAE3ZhbHVlT3JFcnJvck1lc3NhZ2UAAAACCQAEGgAAAAIFAAAADmNvbmZpZ1Byb3ZpZGVyBQAAAAFrBQAAAAFlAQAAAAVjb25mUwAAAAIAAAABawAAAAFlCQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAAQdAAAAAgUAAAAOY29uZmlnUHJvdmlkZXIFAAAAAWsFAAAAAWUAAAAABUJVTExLAgAAAAZCVUxMSWQAAAAABUJFQVJLAgAAAAZCRUFSSWQAAAAABVVTRE5LAgAAAAttYWluVG9rZW5JZAAAAAAIQlVMTENPTEsCAAAADmJ1bGxDb2xsYXRlcmFsAAAAAAhCRUFSQ09MSwIAAAAOYmVhckNvbGxhdGVyYWwAAAAACUJVTExDSVJDSwIAAAAPYnVsbENpcmN1bGF0aW9uAAAAAAlCRUFSQ0lSQ0sCAAAAD2JlYXJDaXJjdWxhdGlvbgAAAAAISVNTUEVSQ0sCAAAAD2lzc3VlUGVyY2VudGlsZQAAAAAIUkVEUEVSQ0sCAAAAEHJlZGVlbVBlcmNlbnRpbGUAAAAAB01JTklTU0sCAAAACG1pbklzc3VlAAAAAAdNSU5SRURLAgAAAAltaW5SZWRlZW0AAAAACE1JTlBPT0xLAgAAAAdtaW5Qb29sAAAAAAdGRUVBQ0NLAgAAAA9mZWVzQWNjdW11bGF0ZWQAAAAABldMSVNUSwIAAAAOaXNzdWVXaGl0ZUxpc3QAAAAACFJFQlBFUkNLAgAAABNyZWJhbGFuY2VQZXJjZW50aWxlAAAAAAdSRUJJRFhLAgAAABVsYXN0U2V0dGxlbWVudFByaWNlSWQAAAAABUhFQURLAgAAAAtoZWFkUG9pbnRlcgAAAAAFVEFJTEsCAAAAC3RhaWxQb2ludGVyAAAAAAZRU0laRUsCAAAACXF1ZXVlU2l6ZQAAAAAJUE9PTFVTRE5LAgAAABJwb29sTWFpblRva2VuVmFsdWUAAAAAB1BPT0xVUEsCAAAABnBvb2xVcAAAAAAIUE9PTERXTksCAAAAB3Bvb2xEd24AAAAACVBPT0xDSVJDSwIAAAAUcG9vbFRva2VuQ2lyY3VsYXRpb24AAAAABVBPT0xLAgAAAAlwb29sVG9rZW4AAAAACEFTU05BTUVLAgAAAA1kZWZvQXNzZXROYW1lAAAAAARMRVZLAgAAAAhsZXZlcmFnZQAAAAAJV0FWRVNGRUVLAgAAABF3YXZlc1BhY2VtYWtlckZlZQAAAAAIVVNETkZFRUsCAAAAEHVzZG5QYWNlbWFrZXJGZWUAAAAAC29yYWNsZVBLS2V5AgAAAAZvcmFjbGUBAAAAEWxhc3RQcmljZUluZGV4S2V5AAAAAQAAAAdhc3NldElkAwkAAAAAAAACBQAAAAdhc3NldElkAgAAAAACAAAAC3ByaWNlX2luZGV4CQABLAAAAAICAAAAEiVzJXNfX2lkeEN1cnJlbnRfXwUAAAAHYXNzZXRJZAEAAAAQcHJpY2VJbmRleFByZWZpeAAAAAEAAAAHYXNzZXRJZAMJAAAAAAAAAgUAAAAHYXNzZXRJZAIAAAAAAgAAAAxwcmljZV9pbmRleF8JAAEsAAAAAgkAASwAAAACAgAAABQlcyVzJWRfX2lkeDJIZWlnaHRfXwUAAAAHYXNzZXRJZAIAAAACX18BAAAAEXByaWNlSGVpZ2h0UHJlZml4AAAAAQAAAAdhc3NldElkAwkAAAAAAAACBQAAAAdhc3NldElkAgAAAAACAAAABnByaWNlXwkAASwAAAACCQABLAAAAAICAAAAFyVzJXMlZF9fcHJpY2VCeUhlaWdodF9fBQAAAAdhc3NldElkAgAAAAJfXwAAAAAKbWluVXNkbkZlZQkBAAAAC3ZhbHVlT3JFbHNlAAAAAgkABBoAAAACBQAAAA5jb25maWdQcm92aWRlcgUAAAAIVVNETkZFRUsAAAAAAAAAAAAAAAAAC21pbldhdmVzRmVlCQEAAAALdmFsdWVPckVsc2UAAAACCQAEGgAAAAIFAAAADmNvbmZpZ1Byb3ZpZGVyBQAAAAlXQVZFU0ZFRUsAAAAAAAAAAAAAAAAACWFzc2V0TmFtZQkBAAAAC3ZhbHVlT3JFbHNlAAAAAgkABB0AAAACBQAAAAR0aGlzBQAAAAhBU1NOQU1FSwIAAAAAAAAAAAdidWxsQ29sCQEAAAAGbG9jYWxJAAAAAgUAAAAIQlVMTENPTEsCAAAABG5vIDAAAAAAB2JlYXJDb2wJAQAAAAZsb2NhbEkAAAACBQAAAAhCRUFSQ09MSwIAAAAEbm8gMQAAAAAIYnVsbENpcmMJAQAAAAZsb2NhbEkAAAACBQAAAAlCVUxMQ0lSQ0sCAAAABG5vIDIAAAAACGJlYXJDaXJjCQEAAAAGbG9jYWxJAAAAAgUAAAAJQkVBUkNJUkNLAgAAAARubyAzAAAAAARCVUxMCQEAAAAGbG9jYWxTAAAAAgUAAAAFQlVMTEsCAAAABW5vIDE0AAAAAARCRUFSCQEAAAAGbG9jYWxTAAAAAgUAAAAFQkVBUksCAAAABW5vIDE1AAAAAAltYWluVG9rZW4JAQAAAAZsb2NhbFMAAAACBQAAAAVVU0ROSwIAAAAFbm8gMTYAAAAAD2lzc3VlUGVyY2VudGlsZQkBAAAABWNvbmZJAAAAAgUAAAAISVNTUEVSQ0sCAAAABG5vIDQAAAAAEHJlZGVlbVBlcmNlbnRpbGUJAQAAAAVjb25mSQAAAAIFAAAACFJFRFBFUkNLAgAAAARubyA1AAAAAAhtaW5Jc3N1ZQkBAAAABWNvbmZJAAAAAgUAAAAHTUlOSVNTSwIAAAAEbm8gNgAAAAAJbWluUmVkZWVtCQEAAAAFY29uZkkAAAACBQAAAAdNSU5SRURLAgAAAARubyA3AAAAAAdtaW5Qb29sCQEAAAAFY29uZkkAAAACBQAAAAhNSU5QT09MSwIAAAAEbm8gOAAAAAATcmViYWxhbmNlUGVyY2VudGlsZQkBAAAAC3ZhbHVlT3JFbHNlAAAAAgkABBoAAAACBQAAAA5jb25maWdQcm92aWRlcgkAASwAAAACCQABLAAAAAIJAAQlAAAAAQUAAAAEdGhpcwIAAAABXwUAAAAIUkVCUEVSQ0sAAAAAAAAAAAAAAAAACXdoaXRlbGlzdAkBAAAABWNvbmZTAAAAAgUAAAAGV0xJU1RLAgAAAARubyA5AQAAAAdhbGxvd2VkAAAAAQAAAAFhAwkAAAAAAAACBQAAAAl3aGl0ZWxpc3QCAAAAAAYJAQAAAAlpc0RlZmluZWQAAAABCQAEswAAAAIFAAAACXdoaXRlbGlzdAkABCUAAAABBQAAAAFhAAAAAAhwb29sTWFpbgkBAAAABmxvY2FsSQAAAAIFAAAACVBPT0xVU0ROSwIAAAACbm8AAAAABnBvb2xVcAkBAAAABmxvY2FsSQAAAAIFAAAAB1BPT0xVUEsCAAAABW5vIDEwAAAAAAdwb29sRHduCQEAAAAGbG9jYWxJAAAAAgUAAAAIUE9PTERXTksCAAAABW5vIDExAAAAAAlwb29sVG9rZW4JAQAAAAZsb2NhbFMAAAACBQAAAAVQT09MSwIAAAAFbm8gMTIAAAAAFHBvb2xUb2tlbkNpcmN1bGF0aW9uCQEAAAAGbG9jYWxJAAAAAgUAAAAJUE9PTENJUkNLAgAAAAVubyAxMwAAAAAQcG9vbEJ1bGxFeHBvc3VyZQkAAGsAAAADBQAAAAdidWxsQ29sBQAAAAZwb29sVXAFAAAACGJ1bGxDaXJjAAAAABBwb29sQmVhckV4cG9zdXJlCQAAawAAAAMFAAAAB2JlYXJDb2wFAAAAB3Bvb2xEd24FAAAACGJlYXJDaXJjAAAAAAlwb29sVmFsdWUJAABkAAAAAgkAAGQAAAACBQAAAAhwb29sTWFpbgUAAAAQcG9vbEJ1bGxFeHBvc3VyZQUAAAAQcG9vbEJlYXJFeHBvc3VyZQAAAAAGb3JhY2xlCQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAQAAABRhZGRyZXNzRnJvbVB1YmxpY0tleQAAAAEJAAJZAAAAAQkBAAAAE3ZhbHVlT3JFcnJvck1lc3NhZ2UAAAACCQAEHQAAAAIFAAAABHRoaXMFAAAAC29yYWNsZVBLS2V5AgAAAA5ubyBvcmFjbGVQS0tleQIAAAASYmFkIG9yYWNsZSBhZGRyZXNzAAAAABRyZWJhbGFuY2VkUHJpY2VJbmRleAkBAAAAE3ZhbHVlT3JFcnJvck1lc3NhZ2UAAAACCQAEGgAAAAIFAAAABHRoaXMFAAAAB1JFQklEWEsCAAAAF25vIGxhc3QgcmViYWxhbmNlIHByaWNlAAAAABBvcmFjbGVQcmljZUluZGV4CQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAAQaAAAAAgUAAAAGb3JhY2xlCQEAAAARbGFzdFByaWNlSW5kZXhLZXkAAAABBQAAAAlhc3NldE5hbWUJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAAE2JhZCBvcmFjbGUgZGF0YSBhdCAJAAQlAAAAAQUAAAAGb3JhY2xlAgAAABA6IG5vIGludGVnZXIgYXQgCQEAAAARbGFzdFByaWNlSW5kZXhLZXkAAAABBQAAAAlhc3NldE5hbWUAAAAACGxldmVyYWdlCQEAAAALdmFsdWVPckVsc2UAAAACCQAEGgAAAAIFAAAABHRoaXMFAAAABExFVksAAAAAAAAAAAMBAAAADWhlaWdodEJ5SW5kZXgAAAACAAAACWFzc2V0TmFtZQAAAApwcmljZUluZGV4CQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAAQaAAAAAgUAAAAGb3JhY2xlCQABLAAAAAIJAQAAABBwcmljZUluZGV4UHJlZml4AAAAAQUAAAAJYXNzZXROYW1lCQABpAAAAAEFAAAACnByaWNlSW5kZXgJAAEsAAAAAgIAAAAcbm8gZGF0YSBmb3IgaGVpZ2h0IGF0IGluZGV4IAkAAaQAAAABBQAAAApwcmljZUluZGV4AQAAAA1wcmljZUJ5SGVpZ2h0AAAAAgAAAAlhc3NldE5hbWUAAAALcHJpY2VIZWlnaHQJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABBoAAAACBQAAAAZvcmFjbGUJAAEsAAAAAgkBAAAAEXByaWNlSGVpZ2h0UHJlZml4AAAAAQUAAAAJYXNzZXROYW1lCQABpAAAAAEFAAAAC3ByaWNlSGVpZ2h0CQABLAAAAAICAAAAE25vIGRhdGEgZm9yIGhlaWdodCAJAAGkAAAAAQUAAAALcHJpY2VIZWlnaHQBAAAADHByaWNlQnlJbmRleAAAAAIAAAAJYXNzZXROYW1lAAAACnByaWNlSW5kZXgJAQAAAA1wcmljZUJ5SGVpZ2h0AAAAAgUAAAAJYXNzZXROYW1lCQEAAAANaGVpZ2h0QnlJbmRleAAAAAIFAAAACWFzc2V0TmFtZQUAAAAKcHJpY2VJbmRleAAAAAAJcXVldWVTaXplCQEAAAALdmFsdWVPckVsc2UAAAACCQAEGgAAAAIFAAAABHRoaXMFAAAABlFTSVpFSwAAAAAAAAAAAAAAAAALaGVhZFBvaW50ZXIJAQAAAAt2YWx1ZU9yRWxzZQAAAAIJAAQdAAAAAgUAAAAEdGhpcwUAAAAFSEVBREsCAAAAAAAAAAALdGFpbFBvaW50ZXIJAQAAAAt2YWx1ZU9yRWxzZQAAAAIJAAQdAAAAAgUAAAAEdGhpcwUAAAAFVEFJTEsCAAAAAAAAAAAPZmVlc0FjY3VtdWxhdGVkCQEAAAALdmFsdWVPckVsc2UAAAACCQAEGgAAAAIFAAAABHRoaXMFAAAAB0ZFRUFDQ0sAAAAAAAAAAAAAAAAABUlTU1VFAgAAAAVJU1NVRQAAAAAGUkVERUVNAgAAAAZSRURFRU0AAAAABFBPT0wCAAAABFBPT0wAAAAABlVOUE9PTAIAAAAGVU5QT09MAAAAAApmZWVBZGRyS2V5AgAAAApmZWVBZGRyZXNzAAAAAA5zdGFraW5nQWRkcktleQIAAAAOc3Rha2luZ0FkZHJlc3MAAAAAD2RhZW1vblB1YktleUtleQIAAAAPZGFlbW9uUHVibGljS2V5AAAAAApmZWVBZGRyZXNzCQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAAQmAAAAAQkBAAAABWNvbmZTAAAAAgUAAAAKZmVlQWRkcktleQIAAAANbm8gZmVlQWRkcmVzcwIAAAAOYmFkIGZlZUFkZHJlc3MAAAAADnN0YWtpbmdBZGRyZXNzCQEAAAAFY29uZlMAAAACBQAAAA5zdGFraW5nQWRkcktleQIAAAARbm8gc3Rha2luZ0FkZHJlc3MAAAAAD2RhZW1vblB1YmxpY0tleQkAAlkAAAABCQEAAAAFY29uZlMAAAACBQAAAA9kYWVtb25QdWJLZXlLZXkCAAAAEm5vIGRhZW1vblB1YmxpY0tleQAAAAAKcnBkQWRkcmVzcwkABCYAAAABAgAAACMzUE5pa002eXA0TnFjU1U4Z3V4UXRtUjVvbnIyRDRlOHlUSgAAAAAQcHViS2V5QWRtaW5zTGlzdAkABEwAAAACAgAAACwySEhxVjhXOURKYXlWNVI2dEJEMlNiOHNycGhwb2JvRGk3cjF0MWFQaXVtQwkABEwAAAACAgAAACw1WlhlODJSUkFTVTdxc2hYTTJKOUpOWWhxSjlHV1lqalZxMmd3VVY1TmF6OQkABEwAAAACAgAAACw1V1JYRlNqd2NUYk5mS2NKczhacVhtU1NXWXNTVkpVdE12TXFaajVoSDROYwUAAAADbmlsAQAAAAxidWlsZE5ld0l0ZW0AAAAHAAAABmFjdGlvbgAAAANhbXQAAAAFdG9rZW4AAAAKcHJpY2VJbmRleAAAAAdpbnZva2VyAAAACW1pblBheW91dAAAAAltYXhQYXlvdXQJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAEsAAAAAgUAAAAGYWN0aW9uAgAAAAF8CQABpAAAAAEFAAAAA2FtdAIAAAABfAUAAAAFdG9rZW4CAAAAAXwJAAGkAAAAAQUAAAAKcHJpY2VJbmRleAIAAAABfAUAAAAHaW52b2tlcgIAAAABfAkAAaQAAAABBQAAAAltaW5QYXlvdXQCAAAAAXwJAAGkAAAAAQUAAAAJbWF4UGF5b3V0AgAA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AAARc2V0dGxlZFByaWNlSW5kZXgAAAAAAAAAAAEEAAAADHNldHRsZWRQcmljZQkBAAAADHByaWNlQnlJbmRleAAAAAIFAAAACWFzc2V0TmFtZQUAAAARc2V0dGxlZFByaWNlSW5kZXgEAAAACW5leHRQcmljZQkBAAAADHByaWNlQnlJbmRleAAAAAIFAAAACWFzc2V0TmFtZQUAAAATdW5zZXR0bGVkUHJpY2VJbmRleAQAAAAGbWluVm9sAwkAAGYAAAACBQAAAAdiZWFyQ29sBQAAAAdidWxsQ29sBQAAAAdidWxsQ29sBQAAAAdiZWFyQ29sBAAAAAZyZWRpc3QJAQAAAAJMVgAAAAQFAAAABm1pblZvbAUAAAAMc2V0dGxlZFByaWNlBQAAAAluZXh0UHJpY2UFAAAACGxldmVyYWdlBAAAABNwcmljZVVwR29vZEZvckJ1bGxzCQAAAAAAAAIFAAAACWFzc2V0TmFtZQIAAAAABAAAAAtwcmljZUdvZXNVcAkAAGYAAAACBQAAAAluZXh0UHJpY2UFAAAADHNldHRsZWRQcmljZQQAAAAJYnVsbHNFYXJuCQAAAAAAAAIFAAAAE3ByaWNlVXBHb29kRm9yQnVsbHMFAAAAC3ByaWNlR29lc1VwBAAAAApuZXdCdWxsQ29sAwUAAAAJYnVsbHNFYXJuCQAAZAAAAAIFAAAAB2J1bGxDb2wFAAAABnJlZGlzdAkAAGUAAAACBQAAAAdidWxsQ29sBQAAAAZyZWRpc3QEAAAACm5ld0JlYXJDb2wDBQAAAAlidWxsc0Vhcm4JAABlAAAAAgUAAAAHYmVhckNvbAUAAAAGcmVkaXN0CQAAZAAAAAIFAAAAB2JlYXJDb2wFAAAABnJlZGlzdAQAAAANJHQwMTk1MjgxOTcxNAkBAAAAC3Bvb2xTdXBwb3J0AAAABwUAAAAKbmV3QnVsbENvbAUAAAAKbmV3QmVhckNvbAUAAAAIYnVsbENpcmMFAAAACGJlYXJDaXJjBQAAAAhwb29sTWFpbgUAAAAGcG9vbFVwBQAAAAdwb29sRHduBAAAAAp1cGRCdWxsQ29sCAUAAAANJHQwMTk1MjgxOTcxNAAAAAJfMQQAAAAKdXBkQmVhckNvbAgFAAAADSR0MDE5NTI4MTk3MTQAAAACXzIEAAAAC3VwZEJ1bGxDaXJjCAUAAAANJHQwMTk1MjgxOTcxNAAAAAJfMwQAAAALdXBkQmVhckNpcmMIBQAAAA0kdDAxOTUyODE5NzE0AAAAAl80BAAAAAt1cGRQb29sTWFpbggFAAAADSR0MDE5NTI4MTk3MTQAAAACXzUEAAAACXVwZFBvb2xVcAgFAAAADSR0MDE5NTI4MTk3MTQAAAACXzYEAAAACnVwZFBvb2xEd24IBQAAAA0kdDAxOTUyODE5NzE0AAAAAl83CQAETAAAAAIJAQAAAAJJRQAAAAIFAAAACEJVTExDT0xLBQAAAAp1cGRCdWxsQ29sCQAETAAAAAIJAQAAAAJJRQAAAAIFAAAACEJFQVJDT0xLBQAAAAp1cGRCZWFyQ29sCQAETAAAAAIJAQAAAAJJRQAAAAIFAAAACUJVTExDSVJDSwUAAAALdXBkQnVsbENpcmMJAARMAAAAAgkBAAAAAklFAAAAAgUAAAAJQkVBUkNJUkNLBQAAAAt1cGRCZWFyQ2lyYwkABEwAAAACCQEAAAACSUUAAAACBQAAAAlQT09MVVNETksFAAAAC3VwZFBvb2xNYWluCQAETAAAAAIJAQAAAAJJRQAAAAIFAAAAB1BPT0xVUEsFAAAACXVwZFBvb2xVcAkABEwAAAACCQEAAAACSUUAAAACBQAAAAhQT09MRFdOSwUAAAAKdXBkUG9vbER3bgkABEwAAAACCQEAAAACSUUAAAACBQAAAAdSRUJJRFhLBQAAABN1bnNldHRsZWRQcmljZUluZGV4BQAAAANuaWwBAAAAB2NhbGNNYXgAAAACAAAAA21pbgAAAANhdmcDCQAAZgAAAAIFAAAAA21pbgUAAAADYXZnCQAAAgAAAAEJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAAGXByaWNlIHRvbyBvbGQ6IG1pblBheW91dCAJAAGkAAAAAQUAAAADbWluAgAAAAkgPiBhdmcgPSAJAAGkAAAAAQUAAAADYXZnCQAAZQAAAAIJAABkAAAAAgUAAAADYXZnBQAAAANhdmcFAAAAA21pbgEAAAAUcmVxdWVzdElzc3VlSW50ZXJuYWwAAAADAAAAA2ludgAAAAd0b2tlbklkAAAACW1pblBheW91dAMDCQEAAAACIT0AAAACBQAAAAd0b2tlbklkBQAAAARCVUxMCQEAAAACIT0AAAACBQAAAAd0b2tlbklkBQAAAARCRUFSBwkAAAIAAAABAgAAAA1iYWQgdG9rZW4gcmVxAwkAAAAAAAACCAUAAAADaW52AAAABmNhbGxlcgUAAAAEdGhpcwkAAAIAAAABAgAAAAhjYW4ndCBkbwMJAQAAAAEhAAAAAQkBAAAAB2FsbG93ZWQAAAABCAUAAAADaW52AAAABmNhbGxlcgkAAAIAAAABAgAAABdvbmx5IHdoaXRlbGlzdGVkIGNhbiBkbwQAAAAMZXJyb3JNZXNzYWdlCQABLAAAAAIJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAAGWJhZCB0b2tlbiByZXEsIG9ubHkgQlVMTCgFAAAABEJVTEwCAAAACikgb3IgQkVBUigFAAAABEJFQVICAAAACSkgYWxsb3dlZAMJAQAAAAIhPQAAAAIICQABkQAAAAIIBQAAAANpbnYAAAAIcGF5bWVudHMAAAAAAAAAAAAAAAAHYXNzZXRJZAkAAlkAAAABBQAAAAltYWluVG9rZW4JAAACAAAAAQIAAAANYmFkIHRva2VuIGF0dAQAAAADYW10CAkAAZEAAAACCAUAAAADaW52AAAACHBheW1lbnRzAAAAAAAAAAAAAAAABmFtb3VudAQAAAANJHQwMjA3MjAyMDg5OAMJAAAAAAAAAgUAAAAHdG9rZW5JZAUAAAAEQlVMTAkABRQAAAACBQAAAAdidWxsQ29sBQAAAAhidWxsQ2lyYwMJAAAAAAAAAgUAAAAHdG9rZW5JZAUAAAAEQkVBUgkABRQAAAACBQAAAAdiZWFyQ29sBQAAAAhiZWFyQ2lyYwkAAAIAAAABBQAAAAxlcnJvck1lc3NhZ2UEAAAAA2NvbAgFAAAADSR0MDIwNzIwMjA4OTgAAAACXzEEAAAABGNpcmMIBQAAAA0kdDAyMDcyMDIwODk4AAAAAl8yBAAAAANlc3QJAABrAAAAAwUAAAADYW10BQAAAARjaXJjBQAAAANjb2wEAAAADSR0MDIwOTQwMjEwMzQDCQAAAAAAAAIFAAAACW1pblBheW91dAAAAAAAAAAAAAkABRQAAAACAAAAAAAAAAAABQAAAANNQVgJAAUUAAAAAgUAAAAJbWluUGF5b3V0CQEAAAAHY2FsY01heAAAAAIFAAAACW1pblBheW91dAUAAAADZXN0BAAAAARtaW5QCAUAAAANJHQwMjA5NDAyMTAzNAAAAAJfMQQAAAAEbWF4UAgFAAAADSR0MDIwOTQwMjEwMzQAAAACXzIDCQAAZgAAAAIFAAAACG1pbklzc3VlBQAAAANhbXQJAAACAAAAAQkAASwAAAACCQABLAAAAAICAAAAKkF0dGFjaGVkIHBheW1lbnQgdG9vIHNtYWxsLiBNaW4gcmVxdWlyZWQ6IAkAAaQAAAABCQAAaQAAAAIFAAAACG1pbklzc3VlAAAAAAAAD0JAAgAAAAUgVVNETgQAAAAKbWF4QWxsb3dlZAkBAAAACG1heElzc3VlAAAAAQUAAAAHdG9rZW5JZAMDCQAAAAAAAAIFAAAACXdoaXRlbGlzdAIAAAAACQAAZgAAAAIICQABkQAAAAIIBQAAAANpbnYAAAAIcGF5bWVudHMAAAAAAAAAAAAAAAAGYW1vdW50BQAAAAptYXhBbGxvd2VkBwkAAAIAAAABCQABLAAAAAIJAAEsAAAAAgIAAABEdHJ5aW5nIHRvIGlzc3VlIG1vcmUgdGhhbiBwb29sIGNhbiBoYW5kbGUuIE1heCBhdHRhY2htZW50IGFsbG93ZWQgPSAJAAGkAAAAAQkAAGkAAAACBQAAAAptYXhBbGxvd2VkAAAAAAAAD0JAAgAAAAUgVVNETgkABE4AAAACCQEAAAAHZW5xdWV1ZQAAAAgJAAJYAAAAAQgFAAAAA2ludgAAAA10cmFuc2FjdGlvbklkBQAAAAVJU1NVRQUAAAADYW10BQAAAAd0b2tlbklkCQAAZAAAAAIFAAAAEG9yYWNsZVByaWNlSW5kZXgAAAAAAAAAAAEJAAQlAAAAAQgFAAAAA2ludgAAAAZjYWxsZXIFAAAABG1pblAFAAAABG1heFAJAARMAAAAAgkBAAAABWRlYnVnAAAAAgIAAAAJcmVxdWVzdGVkBQAAAAd0b2tlbklkCQAETAAAAAIJAQAAAAVkZWJ1ZwAAAAICAAAABGJ1bGwFAAAABEJVTEwJAARMAAAAAgkBAAAABWRlYnVnAAAAAgIAAAAEYmVhcgUAAAAEQkVBUgUAAAADbmlsAQAAABVyZXF1ZXN0UmVkZWVtSW50ZXJuYWwAAAACAAAAA2ludgAAAAltaW5QYXlvdXQEAAAAA2FtdAgJAAGRAAAAAggFAAAAA2ludgAAAAhwYXltZW50cwAAAAAAAAAAAAAAAAZhbW91bnQEAAAAB3Rva2VuSWQJAAJYAAAAAQkBAAAAE3ZhbHVlT3JFcnJvck1lc3NhZ2UAAAACCAkAAZEAAAACCAUAAAADaW52AAAACHBheW1lbnRzAAAAAAAAAAAAAAAAB2Fzc2V0SWQCAAAADWJhZCB0b2tlbiBhdHQDAwkBAAAAAiE9AAAAAgUAAAAHdG9rZW5JZAUAAAAEQlVMTAkBAAAAAiE9AAAAAgUAAAAHdG9rZW5JZAUAAAAEQkVBUgcJAAACAAAAAQIAAAANYmFkIHRva2VuIHJlcQQAAAANJHQwMjIwNzEyMjIxMAMJAAAAAAAAAgUAAAAHdG9rZW5JZAUAAAAEQlVMTAkABRQAAAACBQAAAAdidWxsQ29sBQAAAAhidWxsQ2lyYwMJAAAAAAAAAgUAAAAHdG9rZW5JZAUAAAAEQkVBUgkABRQAAAACBQAAAAdiZWFyQ29sBQAAAAhiZWFyQ2lyYwkAAAIAAAABAgAAAA1iYWQgdG9rZW4gcmVxBAAAAANjb2wIBQAAAA0kdDAyMjA3MTIyMjEwAAAAAl8xBAAAAARjaXJjCAUAAAANJHQwMjIwNzEyMjIxMAAAAAJfMgQAAAADZXN0CQAAawAAAAMFAAAAA2FtdAUAAAADY29sBQAAAARjaXJjBAAAAA0kdDAyMjI1ODIyMzUyAwkAAAAAAAACBQAAAAltaW5QYXlvdXQAAAAAAAAAAAAJAAUUAAAAAgAAAAAAAAAAAAUAAAADTUFYCQAFFAAAAAIFAAAACW1pblBheW91dAkBAAAAB2NhbGNNYXgAAAACBQAAAAltaW5QYXlvdXQFAAAAA2VzdAQAAAAEbWluUAgFAAAADSR0MDIyMjU4MjIzNTIAAAACXzEEAAAABG1heFAIBQAAAA0kdDAyMjI1ODIyMzUyAAAAAl8yAwkAAAAAAAACCQEAAAAVdmFsaWRhdGVSZXF1ZXN0UmVkZWVtAAAAAQUAAAADaW52BQAAAAR1bml0CQEAAAAHZW5xdWV1ZQAAAAgJAAJYAAAAAQgFAAAAA2ludgAAAA10cmFuc2FjdGlvbklkBQAAAAZSRURFRU0FAAAAA2FtdAUAAAAHdG9rZW5JZAkAAGQAAAACBQAAABBvcmFjbGVQcmljZUluZGV4AAAAAAAAAAABCQAEJQAAAAEIBQAAAANpbnYAAAAGY2FsbGVyBQAAAARtaW5QBQAAAARtYXhQCQAAAgAAAAECAAAADmRvZXNuJ3QgaGFwcGVuAQAAABNyZXF1ZXN0UG9vbEludGVybmFsAAAAAgAAAANpbnYAAAAJbWluUGF5b3V0AwkBAAAAASEAAAABCQEAAAAHYWxsb3dlZAAAAAEIBQAAAANpbnYAAAAGY2FsbGVyCQAAAgAAAAECAAAAF29ubHkgd2hpdGVsaXN0ZWQgY2FuIGRvBAAAAAplcnJNZXNzYWdlCQABLAAAAAIJAAEsAAAAAgIAAAAcbWFpbiB0b2tlbiBtdXN0IGJlIGF0dGFjaGVkKAUAAAAJbWFpblRva2VuAgAAAAEpBAAAAANwbXQJAAGRAAAAAggFAAAAA2ludgAAAAhwYXltZW50cwAAAAAAAAAAAAMJAQAAAAIhPQAAAAIIBQAAAANwbXQAAAAHYXNzZXRJZAkAAlkAAAABBQAAAAltYWluVG9rZW4JAAACAAAAAQUAAAAKZXJyTWVzc2FnZQMJAABmAAAAAgUAAAAHbWluUG9vbAgFAAAAA3BtdAAAAAZhbW91bnQJAAACAAAAAQkAASwAAAACCQABLAAAAAIJAAEsAAAAAgIAAAAOcG9vbCBhdCBsZWFzdCAJAAGkAAAAAQUAAAAHbWluUG9vbAIAAAABIAUAAAAJbWFpblRva2VuBAAAAAhlc3RpbWF0ZQkAAGsAAAADBQAAABRwb29sVG9rZW5DaXJjdWxhdGlvbggFAAAAA3BtdAAAAAZhbW91bnQFAAAACXBvb2xWYWx1ZQQAAAANJHQwMjMyMDAyMzI5OQMJAAAAAAAAAgUAAAAJbWluUGF5b3V0AAAAAAAAAAAACQAFFAAAAAIAAAAAAAAAAAAFAAAAA01BWAkABRQAAAACBQAAAAltaW5QYXlvdXQJAQAAAAdjYWxjTWF4AAAAAgUAAAAJbWluUGF5b3V0BQAAAAhlc3RpbWF0ZQQAAAAEbWluUAgFAAAADSR0MDIzMjAwMjMyOTkAAAACXzEEAAAABG1heFAIBQAAAA0kdDAyMzIwMDIzMjk5AAAAAl8yCQEAAAAHZW5xdWV1ZQAAAAgJAAJYAAAAAQgFAAAAA2ludgAAAA10cmFuc2FjdGlvbklkBQAAAARQT09MCAkAAZEAAAACCAUAAAADaW52AAAACHBheW1lbnRzAAAAAAAAAAAAAAAABmFtb3VudAIAAAAACQAAZAAAAAIFAAAAEG9yYWNsZVByaWNlSW5kZXgAAAAAAAAAAAEJAAQlAAAAAQgFAAAAA2ludgAAAAZjYWxsZXIFAAAABG1pblAFAAAABG1heFABAAAAFXJlcXVlc3RVbnBvb2xJbnRlcm5hbAAAAAIAAAADaW52AAAACW1pblBheW91dAQAAAAKZXJyTWVzc2FnZQkAASwAAAACCQABLAAAAAICAAAAGG9ubHkgcG9vbCB0b2tlbiBhbGxvd2VkKAUAAAAJcG9vbFRva2VuAgAAAAEpBAAAAANwbXQJAAGRAAAAAggFAAAAA2ludgAAAAhwYXltZW50cwAAAAAAAAAAAAMJAQAAAAIhPQAAAAIIBQAAAANwbXQAAAAHYXNzZXRJZAkAAlkAAAABBQAAAAlwb29sVG9rZW4JAAACAAAAAQUAAAAKZXJyTWVzc2FnZQQAAAAIZXN0aW1hdGUJAABrAAAAAwUAAAAJcG9vbFZhbHVlCAUAAAADcG10AAAABmFtb3VudAUAAAAUcG9vbFRva2VuQ2lyY3VsYXRpb24DCQAAZgAAAAIFAAAAB21pblBvb2wFAAAACGVzdGltYXRlCQAAAgAAAAEJAAEsAAAAAgkAASwAAAACCQABLAAAAAICAAAAE3VucG9vbCBhdCBsZWFzdCBmb3IJAAGkAAAAAQUAAAAHbWluUG9vbAIAAAABIAUAAAAJbWFpblRva2VuBAAAAA0kdDAyMzk1NjI0MDU1AwkAAAAAAAACBQAAAAltaW5QYXlvdXQAAAAAAAAAAAAJAAUUAAAAAgAAAAAAAAAAAAUAAAADTUFYCQAFFAAAAAIFAAAACW1pblBheW91dAkBAAAAB2NhbGNNYXgAAAACBQAAAAltaW5QYXlvdXQFAAAACGVzdGltYXRlBAAAAARtaW5QCAUAAAANJHQwMjM5NTYyNDA1NQAAAAJfMQQAAAAEbWF4UAgFAAAADSR0MDIzOTU2MjQwNTUAAAACXzIJAQAAAAdlbnF1ZXVlAAAACAkAAlgAAAABCAUAAAADaW52AAAADXRyYW5zYWN0aW9uSWQFAAAABlVOUE9PTAgJAAGRAAAAAggFAAAAA2ludgAAAAhwYXltZW50cwAAAAAAAAAAAAAAAAZhbW91bnQCAAAAAAkAAGQAAAACBQAAABBvcmFjbGVQcmljZUluZGV4AAAAAAAAAAABCQAEJQAAAAEIBQAAAANpbnYAAAAGY2FsbGVyBQAAAARtaW5QBQAAAARtYXhQAAAACwAAAANpbnYBAAAABGluaXQAAAALAAAABmNvbmZpZwAAAAhvcmFjbGVQSwAAAAZuYW1ldXAAAAAHbmFtZWR3bgAAAAZkZXNjVXAAAAAHZGVzY0R3bgAAAAhwb29sTmFtZQAAAAhwb29sRGVzYwAAAA1kZWZvQXNzZXROYW1lAAAABWRlbm9tAAAAA2xldgMJAQAAAAlpc0RlZmluZWQAAAABCQAEHQAAAAIFAAAABHRoaXMFAAAABUJVTExLCQAAAgAAAAECAAAAE2FscmVhZHkgaW5pdGlhbGl6ZWQEAAAAE3RvdGFsT3duZWRNYWluVG9rZW4ICQABkQAAAAIIBQAAAANpbnYAAAAIcGF5bWVudHMAAAAAAAAAAAAAAAAGYW1vdW50BAAAAAVidWxscwkAAGkAAAACBQAAABN0b3RhbE93bmVkTWFpblRva2VuAAAAAAAAAAADBAAAAAViZWFycwUAAAAFYnVsbHMEAAAABXBvb2xzCQAAZQAAAAIJAABlAAAAAgUAAAATdG90YWxPd25lZE1haW5Ub2tlbgUAAAAFYnVsbHMFAAAABWJlYXJzAwMDCQAAAAAAAAIFAAAABWJlYXJzAAAAAAAAAAAABgkAAAAAAAACBQAAAAVidWxscwAAAAAAAAAAAAYJAAAAAAAAAgUAAAAFcG9vbHMAAAAAAAAAAAAJAAACAAAAAQIAAAATY2FuJ3QgaW5pdCBiYWxhbmNlcwQAAAAXb3JhY2xlQ3VycmVudFByaWNlSW5kZXgJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABBoAAAACCQEAAAATdmFsdWVPckVycm9yTWVzc2FnZQAAAAIJAQAAABRhZGRyZXNzRnJvbVB1YmxpY0tleQAAAAEJAAJZAAAAAQUAAAAIb3JhY2xlUEsCAAAAEmJhZCBvcmFjbGUgYWRkcmVzcwkBAAAAEWxhc3RQcmljZUluZGV4S2V5AAAAAQUAAAANZGVmb0Fzc2V0TmFtZQIAAAAiY2FuJ3QgZmluZCBsYXN0IG9yYWNsZSBwcmljZSBpbmRleAQAAAAEYnVsbAkABEIAAAAFBQAAAAZuYW1ldXAFAAAABmRlc2NVcAkAAGgAAAACCQAAaAAAAAIAAAAAAAAAAGQFAAAABHRlbjYFAAAABHRlbjYAAAAAAAAAAAYGBAAAAARiZWFyCQAEQgAAAAUFAAAAB25hbWVkd24FAAAAB2Rlc2NEd24JAABoAAAAAgkAAGgAAAACAAAAAAAAAABkBQAAAAR0ZW42BQAAAAR0ZW42AAAAAAAAAAAGBgQAAAAEcG9vbAkABEIAAAAFBQAAAAhwb29sTmFtZQUAAAAIcG9vbERlc2MJAABoAAAAAgkAAGgAAAACAAAAAAAAAABkBQAAAAR0ZW42BQAAAAR0ZW42AAAAAAAAAAAGBgQAAAAEYnVpZAkABDgAAAABBQAAAARidWxsBAAAAARiZWlkCQAEOAAAAAEFAAAABGJlYXIEAAAABHBvaWQJAAQ4AAAAAQUAAAAEcG9vbAkABEwAAAACBQAAAARidWxsCQAETAAAAAIFAAAABGJlYXIJAARMAAAAAgUAAAAEcG9vbAkABEwAAAACCQEAAAACU0UAAAACBQAAAAVCVUxMSwkAAlgAAAABBQAAAARidWlkCQAETAAAAAIJAQAAAAJTRQAAAAIFAAAABUJFQVJLCQACWAAAAAEFAAAABGJlaWQJAARMAAAAAgkBAAAAAlNFAAAAAgUAAAAFVVNETksJAAJYAAAAAQkBAAAABXZhbHVlAAAAAQgJAAGRAAAAAggFAAAAA2ludgAAAAhwYXltZW50cwAAAAAAAAAAAAAAAAdhc3NldElkCQAETAAAAAIJAQAAAAJTRQAAAAIFAAAABVBPT0xLCQACWAAAAAEFAAAABHBvaWQJAARMAAAAAgkBAAAAAlNFAAAAAgUAAAAIQVNTTkFNRUsFAAAADWRlZm9Bc3NldE5hbWUJAARMAAAAAgkBAAAAAlNFAAAAAgUAAAALb3JhY2xlUEtLZXkFAAAACG9yYWNsZVBLCQAETAAAAAIJAQAAAAJJRQAAAAIFAAAAB1JFQklEWEsFAAAAF29yYWNsZUN1cnJlbnRQcmljZUluZGV4CQAETAAAAAIJAQAAAAJJRQAAAAIFAAAACEJVTExDT0xLBQAAAAVidWxscwkABEwAAAACCQEAAAACSUUAAAACBQAAAAhCRUFSQ09MSwUAAAAFYmVhcnMJAARMAAAAAgkBAAAAAklFAAAAAgUAAAAJQlVMTENJUkNLCQAAaQAAAAIFAAAABWJ1bGxzBQAAAAVkZW5vbQkABEwAAAACCQEAAAACSUUAAAACBQAAAAlCRUFSQ0lSQ0sJAABpAAAAAgUAAAAFYmVhcnMFAAAABWRlbm9tCQAETAAAAAIJAQAAAAJJRQAAAAIFAAAACVBPT0xDSVJDSwkAAGkAAAACBQAAAAVwb29scwUAAAAFZGVub20JAARMAAAAAgkBAAAAAklFAAAAAgUAAAAIUE9PTERXTksAAAAAAAAAAAAJAARMAAAAAgkBAAAAAklFAAAAAgUAAAAHUE9PTFVQSwAAAAAAAAAAAAkABEwAAAACCQEAAAACSUUAAAACBQAAAAlQT09MVVNETksFAAAABXBvb2xzCQAETAAAAAIJAQAAAAJTRQAAAAIFAAAAEWNvbmZpZ1Byb3ZpZGVyS2V5BQAAAAZjb25maWcJAARMAAAAAgkBAAAAAklFAAAAAgUAAAAETEVWSwUAAAADbGV2CQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAANpbnYAAAAGY2FsbGVyCQAAaQAAAAIFAAAABWJ1bGxzBQAAAAVkZW5vbQUAAAAEYnVpZAkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAADaW52AAAABmNhbGxlcgkAAGkAAAACBQAAAAViZWFycwUAAAAFZGVub20FAAAABGJlaWQJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAA2ludgAAAAZjYWxsZXIJAABpAAAAAgUAAAAFcG9vbHMFAAAABWRlbm9tBQAAAARwb2lkBQAAAANuaWwAAAABaQEAAAALd2l0aGRyYXdGZWUAAAABAAAABmFtb3VudAMJAABmAAAAAgUAAAAGYW1vdW50BQAAAA9mZWVzQWNjdW11bGF0ZWQJAAACAAAAAQkAASwAAAACAgAAABV0b28gbXVjaC4gYXZhaWxhYmxlOiAJAAGkAAAAAQUAAAAPZmVlc0FjY3VtdWxhdGVkCQAETAAAAAIJAQAAAAJJRQAAAAIFAAAAB0ZFRUFDQ0sJAABlAAAAAgUAAAAPZmVlc0FjY3VtdWxhdGVkBQAAAAZhbW91bnQJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwUAAAAKZmVlQWRkcmVzcwUAAAAGYW1vdW50CQACWQAAAAEFAAAACW1haW5Ub2tlbgUAAAADbmlsAAAAA2ludgEAAAANcmVxdWVzdFJlZGVlbQAAAAAJAQAAABVyZXF1ZXN0UmVkZWVtSW50ZXJuYWwAAAACBQAAAANpbnYAAAAAAAAAAAAAAAADaW52AQAAAA9yZXF1ZXN0UmVkZWVtU2wAAAABAAAAAnNsAwkBAAAADXZhbGlkYXRlUE1GZWUAAAACBQAAAANpbnYFAAAAAnNsCQEAAAAVcmVxdWVzdFJlZGVlbUludGVybmFsAAAAAgUAAAADaW52BQAAAAJzbAkBAAAABXRocm93AAAAAAAAAANpbnYBAAAADHJlcXVlc3RJc3N1ZQAAAAEAAAAHdG9rZW5JZAkBAAAAFHJlcXVlc3RJc3N1ZUludGVybmFsAAAAAwUAAAADaW52BQAAAAd0b2tlbklkAAAAAAAAAAAAAAAAA2ludgEAAAAOcmVxdWVzdElzc3VlU2wAAAACAAAAB3Rva2VuSWQAAAACc2wDCQEAAAANdmFsaWRhdGVQTUZlZQAAAAIFAAAAA2ludgUAAAACc2wJAQAAABRyZXF1ZXN0SXNzdWVJbnRlcm5hbAAAAAMFAAAAA2ludgUAAAAHdG9rZW5JZAUAAAACc2wJAQAAAAV0aHJvdwAAAAAAAAADaW52AQAAAAtyZXF1ZXN0UG9vbAAAAAAJAQAAABNyZXF1ZXN0UG9vbEludGVybmFsAAAAAgUAAAADaW52AAAAAAAAAAAAAAAAA2ludgEAAAANcmVxdWVzdFBvb2xTbAAAAAEAAAACc2wDCQEAAAANdmFsaWRhdGVQTUZlZQAAAAIFAAAAA2ludgUAAAACc2wJAQAAABNyZXF1ZXN0UG9vbEludGVybmFsAAAAAgUAAAADaW52BQAAAAJzbAkBAAAABXRocm93AAAAAAAAAANpbnYBAAAADXJlcXVlc3RVbnBvb2wAAAAACQEAAAAVcmVxdWVzdFVucG9vbEludGVybmFsAAAAAgUAAAADaW52AAAAAAAAAAAAAAAAA2ludgEAAAAPcmVxdWVzdFVucG9vbFNsAAAAAQAAAAJzbAMJAQAAAA12YWxpZGF0ZVBNRmVlAAAAAgUAAAADaW52BQAAAAJzbAkBAAAAFXJlcXVlc3RVbnBvb2xJbnRlcm5hbAAAAAIFAAAAA2ludgUAAAACc2wJAQAAAAV0aHJvdwAAAAAAAAADaW52AQAAAAZzZXR0bGUAAAAABAAAAApxdWV1ZUVtcHR5CQAAAAAAAAIFAAAAC2hlYWRQb2ludGVyAgAAAAAEAAAADGNhblJlYmFsYW5jZQkAAGYAAAACBQAAABBvcmFjbGVQcmljZUluZGV4BQAAABRyZWJhbGFuY2VkUHJpY2VJbmRleAMFAAAACnF1ZXVlRW1wdHkDBQAAAAxjYW5SZWJhbGFuY2UJAQAAAAlyZWJhbGFuY2UAAAAACQAAAgAAAAECAAAAF1tPS10gYWxsIGRvbmUsIGNhcnJ5IG9uBAAAAARkYXRhCQAEtQAAAAIJAQAAABN2YWx1ZU9yRXJyb3JNZXNzYWdlAAAAAgkABB0AAAACBQAAAAR0aGlzBQAAAAtoZWFkUG9pbnRlcgkAASwAAAACAgAAABpiYWQgaGVhZCBwb2ludGVyKHNldHRsZSk6IAUAAAALaGVhZFBvaW50ZXICAAAAAXwEAAAACnByaWNlSW5kZXgJAQAAAA1wYXJzZUludFZhbHVlAAAAAQkAAZEAAAACBQAAAARkYXRhAAAAAAAAAAADAwkAAGYAAAACBQAAAApwcmljZUluZGV4BQAAABRyZWJhbGFuY2VkUHJpY2VJbmRleAMFAAAADGNhblJlYmFsYW5jZQkBAAAACXJlYmFsYW5jZQAAAAAJAAACAAAAAQIAAAARW09LXSBuZWVkIHRvIHdhaXQDCQAAAAAAAAIFAAAACnByaWNlSW5kZXgFAAAAFHJlYmFsYW5jZWRQcmljZUluZGV4CQEAAAAHZGVxdWV1ZQAAAAAJAAACAAAAAQIAAAAwY29ycnVwdCBkYXRhLCBmdXR1cmUgcHJpY2UgaWQgYWxyZWFkeSByZWJhbGFuY2VkAAAAAQAAAAJ0eAEAAAAGdmVyaWZ5AAAAAAQAAAAHaW5pdGlhbAMJAQAAAAEhAAAAAQkBAAAACWlzRGVmaW5lZAAAAAEJAAQdAAAAAgUAAAAEdGhpcwUAAAAFQlVMTEsJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAACAUAAAACdHgAAAAPc2VuZGVyUHVibGljS2V5BwQAAAALYWRtaW5BY3Rpb24JAABmAAAAAgkAAGQAAAACCQAAZAAAAAIDCQAB9AAAAAMIBQAAAAJ0eAAAAAlib2R5Qnl0ZXMJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAAkAAlkAAAABCQABkQAAAAIFAAAAEHB1YktleUFkbWluc0xpc3QAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAADCQAB9AAAAAMIBQAAAAJ0eAAAAAlib2R5Qnl0ZXMJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAQkAAlkAAAABCQABkQAAAAIFAAAAEHB1YktleUFkbWluc0xpc3QAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAADCQAB9AAAAAMIBQAAAAJ0eAAAAAlib2R5Qnl0ZXMJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAgkAAlkAAAABCQABkQAAAAIFAAAAEHB1YktleUFkbWluc0xpc3QAAAAAAAAAAAIAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAEEAAAADXN0YWtpbmdBY3Rpb24EAAAAByRtYXRjaDAFAAAAAnR4AwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABdJbnZva2VTY3JpcHRUcmFuc2FjdGlvbgQAAAACdHgFAAAAByRtYXRjaDAEAAAAD3NpZ25lZENvcnJlY3RseQkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAAFAAAAD2RhZW1vblB1YmxpY0tleQQAAAALZmVlc0NvcnJlY3QDCQAAAAAAAAIIBQAAAAJ0eAAAAApmZWVBc3NldElkBQAAAAR1bml0CQAAZwAAAAIJAABoAAAAAgAAAAAAAAAD6AAAAAAAAAAD6AgFAAAAAnR4AAAAA2ZlZQcEAAAAC2RhcHBDb3JyZWN0CQAAAAAAAAIIBQAAAAJ0eAAAAARkQXBwBQAAAApycGRBZGRyZXNzBAAAAAZ1bmxvY2sJAAAAAAAAAggFAAAAAnR4AAAACGZ1bmN0aW9uAgAAAA51bmxvY2tOZXV0cmlubwQAAAAEbG9jawMDCQAAAAAAAAIIBQAAAAJ0eAAAAAhmdW5jdGlvbgIAAAAObG9ja05ldXRyaW5vU1AJAAAAAAAAAgkAAZEAAAACCAUAAAACdHgAAAAEYXJncwAAAAAAAAAAAAUAAAAOc3Rha2luZ0FkZHJlc3MHCQAAZwAAAAIICQAD7wAAAAEFAAAABHRoaXMAAAAJYXZhaWxhYmxlBQAAAAR0ZW44BwQAAAALZnVuY0NvcnJlY3QDBQAAAARsb2NrBgUAAAAGdW5sb2NrAwMDBQAAAA9zaWduZWRDb3JyZWN0bHkFAAAAC2ZlZXNDb3JyZWN0BwUAAAALZGFwcENvcnJlY3QHBQAAAAtmdW5jQ29ycmVjdAcHAwMFAAAAB2luaXRpYWwGBQAAAAthZG1pbkFjdGlvbgYFAAAADXN0YWtpbmdBY3Rpb24lxvcZ"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "settle", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))

	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.StringDataEntry{Key: "tailPointer", Value: ""}},
		{Entry: &proto.StringDataEntry{Key: "headPointer", Value: ""}},
		{Entry: &proto.IntegerDataEntry{Key: "queueSize", Value: 0}},
	}
	expectedTransfers := []*proto.TransferScriptAction{
		{
			Recipient: proto.NewRecipientFromAddress(proto.MustAddressFromString("3PJCXW3XZWr2tTpT5u52cPXcMWVP9AHBC9h")),
			Amount:    19000000,
			Asset:     *proto.NewOptionalAssetFromDigest(crypto.MustDigestFromBase58("4VDwPimjMR31ofr8qoRZ6nvhTJq7Rf21cZp1c425dUAR")),
		},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    expectedTransfers,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

func mustBytesFromBase64(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func mustIntFromString(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestZeroReissue(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("BbVsHJpTJKhuMTco6MTV984fR1dHMLiwUuDjepbsYi45")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("3fNyidPXzaXhdJdhzvKuNbZ3D5tSQGe9Txuy8gZamhhEdwzpVQwrmupvqDm6hHfnoFA1XkTQ773iCpQTef68P71t")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("289xpUrYrKbLjaKkqH3XNhfecukcYRaDRT3JDrvkvQRU")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3MYx35iDGa74cDGfXLUGrrBKAtyeLpMTNC8")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "replenishment",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{proto.ScriptPayment{Amount: 10000, Asset: proto.OptionalAsset{}}, proto.ScriptPayment{Amount: 10000, Asset: proto.OptionalAsset{}}},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             1000000,
		Timestamp:       1599205729886,
	}
	genPK := crypto.MustPublicKeyFromBase58("BeaUPHHepTBnfjt9jxv53S4PnQELfxs8XgJX6hNqu3vu")
	gs := crypto.MustBytesFromBase58("72945rtzGJUfCPq5TXioePGMZLNHgu444phMBpCegcL1ettVKiS88W2gr296g5DBzLe1AHs9ZRgbuPf7dLP5s4ZV8YtXhUVm2YLEoueSVQAfBEdAbHEk8wH2PjP8CL1B49S")
	genAddr, err := proto.NewAddressFromString("3Mmfx11MzJQq9s45ewtTbVC6fzqzDhzAFJ7")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1599205725997,
		Height:              451323,
		BaseTarget:          1216,
		GenerationSignature: gs,
		Generator:           genAddr,
		GeneratorPublicKey:  genPK,
	}
	booleanEntries := map[string]bool{
		"status": true,
	}
	stringEntries := map[string]string{
		"owner":         "3N3AeA5FWm7EHheHoik8BBEA3GXXJosnVY8",
		"version":       "0.0.2",
		"assetIdTokenA": "WAVES",
		"assetIdTokenB": "WAVES",
	}
	intEntries := map[string]int64{
		"comission":          0,
		"amountTokenA":       10000,
		"amountTokenB":       10000,
		"exchange_count":     0,
		"share_token_supply": 10000,
	}
	binaryEntries := map[string][]byte{
		"share_token_id": mustBytesFromBase64("Q3Uk9ZN5g5+xynU7VGPXUg1eVga04VYXnnZ0q+M1dxQ="),
	}
	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 451323
		},
		schemeFunc: func() byte {
			return proto.StageNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 451323, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
					v, ok := booleanEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return &proto.BooleanDataEntry{Key: key, Value: v}, nil
				},
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					v, ok := stringEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return &proto.StringDataEntry{Key: key, Value: v}, nil
				},
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					v, ok := intEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return &proto.IntegerDataEntry{Key: key, Value: v}, nil
				},
				RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
					v, ok := binaryEntries[key]
					if !ok {
						return nil, errors.New("fail")
					}
					return &proto.BinaryDataEntry{Key: key, Value: v}, nil
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.StageNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(4, proto.StageNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		checkMessageLengthFunc: v3check,
		rideV6ActivatedFunc:    noRideV6,
	}

	code := "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ABN0b2tlblJlY2VpdmVBbW91bnRBCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAAxhbW91bnRUb2tlbkIFAAAAE3Rva2VuUmVjZWl2ZUFtb3VudEIJAARMAAAAAgkBAAAAC1N0cmluZ0VudHJ5AAAAAgIAAAANYXNzZXRJZFRva2VuQQUAAAATYXNzZXRJZFRva2VuU3RyaW5nQQkABEwAAAACCQEAAAALU3RyaW5nRW50cnkAAAACAgAAAA1hc3NldElkVG9rZW5CBQAAABNhc3NldElkVG9rZW5TdHJpbmdCCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAA5leGNoYW5nZV9jb3VudAAAAAAAAAAAAAkABEwAAAACCQEAAAAMQm9vbGVhbkVudHJ5AAAAAgIAAAAGc3RhdHVzBgkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAAJY29taXNzaW9uBQAAAAljb21pc3Npb24JAARMAAAAAgkBAAAAC1N0cmluZ0VudHJ5AAAAAgIAAAAHdmVyc2lvbgUAAAAHdmVyc2lvbgkABEwAAAACCQEAAAALU3RyaW5nRW50cnkAAAACAgAAAAVvd25lcgkAAlgAAAABBQAAAAVvd25lcgkABEwAAAACCQAEQgAAAAUFAAAADnNoYXJlVG9rZW5OYW1lBQAAAA5zaGFyZVRva2VuTmFtZQUAAAAXc2hhcmVUb2tlbkluaXRpYWxBbW91bnQAAAAAAAAAAAAGCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAApjb250ZXh0T2JqAAAABmNhbGxlcgUAAAAXc2hhcmVUb2tlbkluaXRpYWxBbW91bnQFAAAAEXNoYXJlVG9rZW5Bc3NldElkCQAETAAAAAIJAQAAAAtCaW5hcnlFbnRyeQAAAAICAAAADnNoYXJlX3Rva2VuX2lkBQAAABFzaGFyZVRva2VuQXNzZXRJZAkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAASc2hhcmVfdG9rZW5fc3VwcGx5BQAAABdzaGFyZVRva2VuSW5pdGlhbEFtb3VudAUAAAADbmlsAAAACmNvbnRleHRPYmoBAAAADXJlcGxlbmlzaG1lbnQAAAAABAAAAAhwYXltZW50QQkBAAAABXZhbHVlAAAAAQkAAZEAAAACCAUAAAAKY29udGV4dE9iagAAAAhwYXltZW50cwAAAAAAAAAAAAQAAAAIcGF5bWVudEIJAQAAAAV2YWx1ZQAAAAEJAAGRAAAAAggFAAAACmNvbnRleHRPYmoAAAAIcGF5bWVudHMAAAAAAAAAAAEEAAAAEGFzc2V0SWRSZWNlaXZlZEEIBQAAAAhwYXltZW50QQAAAAdhc3NldElkBAAAABN0b2tlblJlY2VpdmVBbW91bnRBCAUAAAAIcGF5bWVudEEAAAAGYW1vdW50BAAAABBhc3NldElkUmVjZWl2ZWRCCAUAAAAIcGF5bWVudEIAAAAHYXNzZXRJZAQAAAATdG9rZW5SZWNlaXZlQW1vdW50QggFAAAACHBheW1lbnRCAAAABmFtb3VudAQAAAARZEFwcFRva2Vuc0Ftb3VudEEEAAAAByRtYXRjaDAFAAAADWFzc2V0SWRUb2tlbkEDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABFVuaXQEAAAADWFzc2V0SWRUb2tlbkEFAAAAByRtYXRjaDAICQAD7wAAAAEFAAAABHRoaXMAAAAJYXZhaWxhYmxlAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAApCeXRlVmVjdG9yBAAAAA1hc3NldElkVG9rZW5BBQAAAAckbWF0Y2gwCQAD8AAAAAIFAAAABHRoaXMFAAAADWFzc2V0SWRUb2tlbkEJAAACAAAAAQIAAAALTWF0Y2ggZXJyb3IEAAAAEWRBcHBUb2tlbnNBbW91bnRCBAAAAAckbWF0Y2gwBQAAAA1hc3NldElkVG9rZW5CAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAARVbml0BAAAAA1hc3NldElkVG9rZW5CBQAAAAckbWF0Y2gwCAkAA+8AAAABBQAAAAR0aGlzAAAACWF2YWlsYWJsZQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAKQnl0ZVZlY3RvcgQAAAANYXNzZXRJZFRva2VuQgUAAAAHJG1hdGNoMAkAA/AAAAACBQAAAAR0aGlzBQAAAA1hc3NldElkVG9rZW5CCQAAAgAAAAECAAAAC01hdGNoIGVycm9yAwMJAQAAAAIhPQAAAAIFAAAAEGFzc2V0SWRSZWNlaXZlZEEFAAAADWFzc2V0SWRUb2tlbkEGCQEAAAACIT0AAAACBQAAABBhc3NldElkUmVjZWl2ZWRCBQAAAA1hc3NldElkVG9rZW5CCQAAAgAAAAECAAAAEGluY29ycmVjdCBhc3NldHMEAAAACnRva2VuUmF0aW8JAABrAAAAAwkAAGgAAAACBQAAABN0b2tlblJlY2VpdmVBbW91bnRBBQAAABFkQXBwVG9rZW5zQW1vdW50QgAAAAAAAAAD6AkAAGgAAAACBQAAABN0b2tlblJlY2VpdmVBbW91bnRCBQAAABFkQXBwVG9rZW5zQW1vdW50QQQAAAAQdG9rZW5TaGFyZVN1cHBseQkBAAAAEUBleHRyTmF0aXZlKDEwNTApAAAAAgUAAAAEdGhpcwIAAAASc2hhcmVfdG9rZW5fc3VwcGx5AwkBAAAAASEAAAABAwkAAGYAAAACBQAAAAp0b2tlblJhdGlvAAAAAAAAAAPnCQAAZgAAAAIAAAAAAAAAA+kFAAAACnRva2VuUmF0aW8HCQAAAgAAAAECAAAAEGluY29ycmVjdCBhc3NldHMEAAAAFXNoYXJlVG9rZW5Ub1BheUFtb3VudAkAAGkAAAACCQAAaAAAAAIFAAAAE3Rva2VuUmVjZWl2ZUFtb3VudEEFAAAAEHRva2VuU2hhcmVTdXBwbHkFAAAAEWRBcHBUb2tlbnNBbW91bnRBCQAETAAAAAIJAQAAAAdSZWlzc3VlAAAAAwUAAAARYXNzZXRJZFRva2VuU2hhcmUFAAAAFXNoYXJlVG9rZW5Ub1BheUFtb3VudAYJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAACmNvbnRleHRPYmoAAAAGY2FsbGVyBQAAABVzaGFyZVRva2VuVG9QYXlBbW91bnQFAAAAEWFzc2V0SWRUb2tlblNoYXJlCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAABJzaGFyZV90b2tlbl9zdXBwbHkJAABkAAAAAgUAAAAQdG9rZW5TaGFyZVN1cHBseQUAAAAVc2hhcmVUb2tlblRvUGF5QW1vdW50BQAAAANuaWwAAAAApj0Y9A=="
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "replenishment", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))

	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "share_token_supply", Value: 10000}},
	}
	expectedTransfers := []*proto.TransferScriptAction{
		{
			Recipient: proto.NewRecipientFromAddress(proto.MustAddressFromString("3MSNMcqyweiM9cWpvf4Fn8GAWeuPstxj2hK")),
			Amount:    0,
			Asset:     *proto.NewOptionalAssetFromDigest(crypto.MustDigestFromBase58("5YKvHw6nEbVckCje1khnM5XZPRufzSxU29kV2hZXc9co")),
		},
	}
	expectedReissues := []*proto.ReissueScriptAction{
		{
			AssetID:    crypto.MustDigestFromBase58("5YKvHw6nEbVckCje1khnM5XZPRufzSxU29kV2hZXc9co"),
			Quantity:   0,
			Reissuable: true,
		},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    expectedTransfers,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     expectedReissues,
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

type jsonDataProvider struct {
	strings  map[string]string
	ints     map[string]int
	bools    map[string]bool
	binaries map[string][]byte
}

func newJsonDataProvider(s string) *jsonDataProvider {
	strs := make(map[string]string)
	ints := make(map[string]int)
	bools := make(map[string]bool)
	binaries := make(map[string][]byte)
	var data []struct {
		Entry struct {
			Key         string  `json:"key"`
			BoolValue   *bool   `json:"boolValue"`
			StringVale  *string `json:"stringValue"`
			IntValue    *string `json:"intValue"`
			BinaryValue *string `json:"binaryValue"`
		} `json:"entry"`
	}
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		panic(err)
	}
	for _, d := range data {
		key := d.Entry.Key
		switch {
		case d.Entry.BoolValue != nil:
			bools[key] = *d.Entry.BoolValue
		case d.Entry.StringVale != nil:
			strs[key] = *d.Entry.StringVale
		case d.Entry.IntValue != nil:
			ints[key] = mustIntFromString(*d.Entry.IntValue)
		case d.Entry.BinaryValue != nil:
			binaries[key] = mustBytesFromBase64(*d.Entry.BinaryValue)
		}
	}
	return &jsonDataProvider{
		strings:  strs,
		ints:     ints,
		bools:    bools,
		binaries: binaries,
	}
}

func (p *jsonDataProvider) getBool(k string) (*proto.BooleanDataEntry, error) {
	v, ok := p.bools[k]
	if !ok {
		return nil, errors.Errorf("bool value not found by key '%s'", k)
	}
	return &proto.BooleanDataEntry{Key: k, Value: v}, nil
}

func (p *jsonDataProvider) getString(k string) (*proto.StringDataEntry, error) {
	v, ok := p.strings[k]
	if !ok {
		return nil, errors.Errorf("string value not found by key '%s'", k)
	}
	return &proto.StringDataEntry{Key: k, Value: v}, nil
}

func (p *jsonDataProvider) getInt(k string) (*proto.IntegerDataEntry, error) {
	v, ok := p.ints[k]
	if !ok {
		return nil, errors.Errorf("int value not found by key '%s'", k)
	}
	return &proto.IntegerDataEntry{Key: k, Value: int64(v)}, nil
}

func (p *jsonDataProvider) getBinary(k string) (*proto.BinaryDataEntry, error) {
	v, ok := p.binaries[k]
	if !ok {
		return nil, errors.Errorf("binary value not found by key '%s'", k)
	}
	return &proto.BinaryDataEntry{Key: k, Value: v}, nil
}

func TestStageNet2(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("9vt45R9y63Xwcseat59BchUjfJGHSuN5LeTK6Pd6cFUM")
	require.NoError(t, err)
	proof, err := crypto.NewSignatureFromBase58("5umzUJtYh4KQADV8TsgphYrtafyve1ahfUVVmygVSbCeJXCYz7ivXipJbkB4hqqKvvt48ocU5oi3TT198UnuPZzD")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	proofs.Proofs = []proto.B58Bytes{proof[:]}
	sender, err := crypto.NewPublicKeyFromBase58("9pe2JtkhuJAwRcvSJFBYJQVxNPfRSeq39XUCw6E4Rk8c")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{&proto.StringArgument{Value: "listed_2metSrd6Gn7VDVB61LF7DkZfxP3sx7Ag9Evx9JomcdTb"}, &proto.IntegerArgument{Value: 500000}}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "purchaseToken",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{proto.ScriptPayment{Amount: 10000, Asset: proto.OptionalAsset{}}, proto.ScriptPayment{Amount: 10000, Asset: proto.OptionalAsset{}}},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             1300000,
		Timestamp:       1599565088614,
	}
	genPK := crypto.MustPublicKeyFromBase58("289xpUrYrKbLjaKkqH3XNhfecukcYRaDRT3JDrvkvQRU")
	gs := crypto.MustBytesFromBase58("FHujZ5xfeqkp7H3yKhUmSd2Fc7vwTcLPcrCBiYmVnX2aeCoe6EwGXzuJMhGhsjRaSA3SGtG6kGEMDDTnMVyqMW3kRC1r1xqsnpoR1tN7vwmeALGEY8xQnwrkae8znRrHMJB")
	genAddr, err := proto.NewAddressFromString("3MSNMcqyweiM9cWpvf4Fn8GAWeuPstxj2hK")
	require.NoError(t, err)
	blockInfo := &proto.BlockInfo{
		Timestamp:           1599565042591,
		Height:              457303,
		BaseTarget:          1313,
		GenerationSignature: gs,
		Generator:           genAddr,
		GeneratorPublicKey:  genPK,
	}
	dp := newJsonDataProvider(`[{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"aa","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"dappAddress","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"total_amount","intValue":"600900000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"masterAddress","stringValue":"3MkT3qvGwdLrSs2Cfx3E29ffaM5GYrEZegz"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"assetId_2020-10-3","stringValue":"Eo7N1sjexrfu6mx5LrG3suovSaXaBNnmYfvqJsMzSYE8"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"assetId_2020-10-5","stringValue":"J2j4PRKXuUKUZCP345EXAHYF2gRg15JsYQYtFT4GNPda"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_total_amount_2020-10-3","intValue":"400000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_total_amount_2020-10-5","intValue":"100000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_Qysv1EeAG3svSgY4rXeXYVd5UDWLijge5GTSMJBZWAE","stringValue":"2021-01-31"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_fnWceyvSknkwSvwg3a8viP4BbqZbJ9Xw4bKAuXfgpCf","stringValue":"2020-10-01"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_3ZwqiyJ71v2RL9ynFfhbhrL6exVvpBXq4tMZsM8BMjS2","stringValue":"2020-11-23"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_5WUifJaLLAQwmZdBujmsDRRjd4j75PTqAPFNex3cD1BE","stringValue":"2020-11-11"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_6pooGSU35S9beXySXnfB2Pd8graz6JRvZr6pk9tFRkVX","stringValue":"2020-10-01"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_9QbcTW1TnEG9UtMXj7Qn6QGonY2sbQnDuhADJHRUfYkR","stringValue":"2020-10-01"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_AeRfbghRJkE9De7wpBZBSSunmgrZ1WXAqzp6HEW3thes","stringValue":"2021-01-31"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_BCJ5nmSeoT7o7PGbqPXeFGLmTbrbhGvdYGqFSTGPLQak","stringValue":"2021-01-31"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_DevmCm3b6ciwmcoGtf7amdsbobmSjEQFZdsbS7No6ye4","stringValue":"2020-10-01"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"limit_GHTzwH5nGskQJc6LH3Z9q2rE5dKA1UkhW44ZToKcTU6J","stringValue":"2020-09-30"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"unitPrice_Qysv1EeAG3svSgY4rXeXYVd5UDWLijge5GTSMJBZWAE","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"unitPrice_fnWceyvSknkwSvwg3a8viP4BbqZbJ9Xw4bKAuXfgpCf","intValue":"30"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"unitPrice_3ZwqiyJ71v2RL9ynFfhbhrL6exVvpBXq4tMZsM8BMjS2","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"unitPrice_5WUifJaLLAQwmZdBujmsDRRjd4j75PTqAPFNex3cD1BE","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"unitPrice_6pooGSU35S9beXySXnfB2Pd8graz6JRvZr6pk9tFRkVX","intValue":"30"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"unitPrice_9QbcTW1TnEG9UtMXj7Qn6QGonY2sbQnDuhADJHRUfYkR","intValue":"30"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"unitPrice_AeRfbghRJkE9De7wpBZBSSunmgrZ1WXAqzp6HEW3thes","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"unitPrice_BCJ5nmSeoT7o7PGbqPXeFGLmTbrbhGvdYGqFSTGPLQak","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"unitPrice_DevmCm3b6ciwmcoGtf7amdsbobmSjEQFZdsbS7No6ye4","intValue":"30"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"unitPrice_GHTzwH5nGskQJc6LH3Z9q2rE5dKA1UkhW44ZToKcTU6J","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_kq1zuYsA6epnS1KeduHLUYVfShfMdjzS88xYutzWwRR_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"owned_5ypDJF3LJAYdHBDEQLbmKcu9NzcumLHL3QZpW3DkuHJ4_limit","stringValue":"2020-10-3"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"owned_5ypDJF3LJAYdHBDEQLbmKcu9NzcumLHL3QZpW3DkuHJ4_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"owned_ExF6Be3WrUyW4aeS4qBnfcH2aEDQXyScDY8u3cG87M8n_limit","stringValue":"2020-10-5"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"owned_ExF6Be3WrUyW4aeS4qBnfcH2aEDQXyScDY8u3cG87M8n_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_29i7ZQAhzWzMV8Dfjqt1jyp8y3GHDBFnV4qWhqLmoZvy_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_2metSrd6Gn7VDVB61LF7DkZfxP3sx7Ag9Evx9JomcdTb_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_5ypDJF3LJAYdHBDEQLbmKcu9NzcumLHL3QZpW3DkuHJ4_limit","stringValue":"2020-10-3"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_5ypDJF3LJAYdHBDEQLbmKcu9NzcumLHL3QZpW3DkuHJ4_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_7nZycAMeNvuiivEJdD1X8U6YF62P4BJb8TNS9QkSMtDS_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_93pNpaap3RT9bVGjGPkFHmJtyXphiUMY68VB5WCEif9G_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_AT3LBUYgVcf7SjNLJLXRsTvgw9Lb94wupN9YSkge3Axy_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_Bk5dKUfzPRCkY4ZMN3rG5xaSACueB3XvKoM5KQTJN1qQ_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_CPNhFYsDBJs8a4KXtGCCu7uGc45QeYNxwkFCGMoqeifW_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_EHuZ1jhXoXeYNY244PC5pzh2fgpDJ1oSoSntMr4yWvGW_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_ExF6Be3WrUyW4aeS4qBnfcH2aEDQXyScDY8u3cG87M8n_limit","stringValue":"2020-10-5"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_ExF6Be3WrUyW4aeS4qBnfcH2aEDQXyScDY8u3cG87M8n_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_GF8HGGqPLDfAjZvkLGhamioZouV6uH6vS92mZv1zA8hu_owner","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_kq1zuYsA6epnS1KeduHLUYVfShfMdjzS88xYutzWwRR_amount","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"owned_5ypDJF3LJAYdHBDEQLbmKcu9NzcumLHL3QZpW3DkuHJ4_amount","intValue":"400000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"owned_ExF6Be3WrUyW4aeS4qBnfcH2aEDQXyScDY8u3cG87M8n_amount","intValue":"100000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_29i7ZQAhzWzMV8Dfjqt1jyp8y3GHDBFnV4qWhqLmoZvy_amount","intValue":"100000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_2metSrd6Gn7VDVB61LF7DkZfxP3sx7Ag9Evx9JomcdTb_amount","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_5ypDJF3LJAYdHBDEQLbmKcu9NzcumLHL3QZpW3DkuHJ4_amount","intValue":"400000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_7nZycAMeNvuiivEJdD1X8U6YF62P4BJb8TNS9QkSMtDS_amount","intValue":"100000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_93pNpaap3RT9bVGjGPkFHmJtyXphiUMY68VB5WCEif9G_amount","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_AT3LBUYgVcf7SjNLJLXRsTvgw9Lb94wupN9YSkge3Axy_amount","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_Bk5dKUfzPRCkY4ZMN3rG5xaSACueB3XvKoM5KQTJN1qQ_amount","intValue":"100000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_CPNhFYsDBJs8a4KXtGCCu7uGc45QeYNxwkFCGMoqeifW_amount","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_EHuZ1jhXoXeYNY244PC5pzh2fgpDJ1oSoSntMr4yWvGW_amount","intValue":"100000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_ExF6Be3WrUyW4aeS4qBnfcH2aEDQXyScDY8u3cG87M8n_amount","intValue":"100000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_GF8HGGqPLDfAjZvkLGhamioZouV6uH6vS92mZv1zA8hu_amount","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_kq1zuYsA6epnS1KeduHLUYVfShfMdjzS88xYutzWwRR_assetId","stringValue":"GHTzwH5nGskQJc6LH3Z9q2rE5dKA1UkhW44ZToKcTU6J"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_29i7ZQAhzWzMV8Dfjqt1jyp8y3GHDBFnV4qWhqLmoZvy_assetId","stringValue":"9QbcTW1TnEG9UtMXj7Qn6QGonY2sbQnDuhADJHRUfYkR"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_2metSrd6Gn7VDVB61LF7DkZfxP3sx7Ag9Evx9JomcdTb_assetId","stringValue":"3ZwqiyJ71v2RL9ynFfhbhrL6exVvpBXq4tMZsM8BMjS2"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_7nZycAMeNvuiivEJdD1X8U6YF62P4BJb8TNS9QkSMtDS_assetId","stringValue":"fnWceyvSknkwSvwg3a8viP4BbqZbJ9Xw4bKAuXfgpCf"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_93pNpaap3RT9bVGjGPkFHmJtyXphiUMY68VB5WCEif9G_assetId","stringValue":"Qysv1EeAG3svSgY4rXeXYVd5UDWLijge5GTSMJBZWAE"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_AT3LBUYgVcf7SjNLJLXRsTvgw9Lb94wupN9YSkge3Axy_assetId","stringValue":"AeRfbghRJkE9De7wpBZBSSunmgrZ1WXAqzp6HEW3thes"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_Bk5dKUfzPRCkY4ZMN3rG5xaSACueB3XvKoM5KQTJN1qQ_assetId","stringValue":"DevmCm3b6ciwmcoGtf7amdsbobmSjEQFZdsbS7No6ye4"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_CPNhFYsDBJs8a4KXtGCCu7uGc45QeYNxwkFCGMoqeifW_assetId","stringValue":"5WUifJaLLAQwmZdBujmsDRRjd4j75PTqAPFNex3cD1BE"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_EHuZ1jhXoXeYNY244PC5pzh2fgpDJ1oSoSntMr4yWvGW_assetId","stringValue":"6pooGSU35S9beXySXnfB2Pd8graz6JRvZr6pk9tFRkVX"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_GF8HGGqPLDfAjZvkLGhamioZouV6uH6vS92mZv1zA8hu_assetId","stringValue":"BCJ5nmSeoT7o7PGbqPXeFGLmTbrbhGvdYGqFSTGPLQak"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_kq1zuYsA6epnS1KeduHLUYVfShfMdjzS88xYutzWwRR_unitPrice","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_29i7ZQAhzWzMV8Dfjqt1jyp8y3GHDBFnV4qWhqLmoZvy_unitPrice","intValue":"30"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_2metSrd6Gn7VDVB61LF7DkZfxP3sx7Ag9Evx9JomcdTb_unitPrice","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_5ypDJF3LJAYdHBDEQLbmKcu9NzcumLHL3QZpW3DkuHJ4_unitPrice","intValue":"300"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_7nZycAMeNvuiivEJdD1X8U6YF62P4BJb8TNS9QkSMtDS_unitPrice","intValue":"30"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_93pNpaap3RT9bVGjGPkFHmJtyXphiUMY68VB5WCEif9G_unitPrice","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_AT3LBUYgVcf7SjNLJLXRsTvgw9Lb94wupN9YSkge3Axy_unitPrice","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_Bk5dKUfzPRCkY4ZMN3rG5xaSACueB3XvKoM5KQTJN1qQ_unitPrice","intValue":"30"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_CPNhFYsDBJs8a4KXtGCCu7uGc45QeYNxwkFCGMoqeifW_unitPrice","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_EHuZ1jhXoXeYNY244PC5pzh2fgpDJ1oSoSntMr4yWvGW_unitPrice","intValue":"30"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_ExF6Be3WrUyW4aeS4qBnfcH2aEDQXyScDY8u3cG87M8n_unitPrice","intValue":"300"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_GF8HGGqPLDfAjZvkLGhamioZouV6uH6vS92mZv1zA8hu_unitPrice","intValue":"20"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"asset_total_amount_Qysv1EeAG3svSgY4rXeXYVd5UDWLijge5GTSMJBZWAE","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"asset_total_amount_fnWceyvSknkwSvwg3a8viP4BbqZbJ9Xw4bKAuXfgpCf","intValue":"100000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_kq1zuYsA6epnS1KeduHLUYVfShfMdjzS88xYutzWwRR_description","stringValue":"みんな電力公式"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"asset_total_amount_3ZwqiyJ71v2RL9ynFfhbhrL6exVvpBXq4tMZsM8BMjS2","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"asset_total_amount_5WUifJaLLAQwmZdBujmsDRRjd4j75PTqAPFNex3cD1BE","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"asset_total_amount_6pooGSU35S9beXySXnfB2Pd8graz6JRvZr6pk9tFRkVX","intValue":"100000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"asset_total_amount_9QbcTW1TnEG9UtMXj7Qn6QGonY2sbQnDuhADJHRUfYkR","intValue":"100000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"asset_total_amount_AeRfbghRJkE9De7wpBZBSSunmgrZ1WXAqzp6HEW3thes","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"asset_total_amount_BCJ5nmSeoT7o7PGbqPXeFGLmTbrbhGvdYGqFSTGPLQak","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"asset_total_amount_DevmCm3b6ciwmcoGtf7amdsbobmSjEQFZdsbS7No6ye4","intValue":"100000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"asset_total_amount_GHTzwH5nGskQJc6LH3Z9q2rE5dKA1UkhW44ZToKcTU6J","intValue":"100000000"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_29i7ZQAhzWzMV8Dfjqt1jyp8y3GHDBFnV4qWhqLmoZvy_description","stringValue":"みんな電力公式"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_2metSrd6Gn7VDVB61LF7DkZfxP3sx7Ag9Evx9JomcdTb_description","stringValue":"みんな電力公式"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_5ypDJF3LJAYdHBDEQLbmKcu9NzcumLHL3QZpW3DkuHJ4_description","stringValue":"みんな電力公式"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_7nZycAMeNvuiivEJdD1X8U6YF62P4BJb8TNS9QkSMtDS_description","stringValue":"みんな電力公式"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_93pNpaap3RT9bVGjGPkFHmJtyXphiUMY68VB5WCEif9G_description","stringValue":"みんな電力公式"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_AT3LBUYgVcf7SjNLJLXRsTvgw9Lb94wupN9YSkge3Axy_description","stringValue":"みんな電力公式"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_Bk5dKUfzPRCkY4ZMN3rG5xaSACueB3XvKoM5KQTJN1qQ_description","stringValue":"みんな電力公式"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_CPNhFYsDBJs8a4KXtGCCu7uGc45QeYNxwkFCGMoqeifW_description","stringValue":"みんな電力公式"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_EHuZ1jhXoXeYNY244PC5pzh2fgpDJ1oSoSntMr4yWvGW_description","stringValue":"みんな電力公式"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_ExF6Be3WrUyW4aeS4qBnfcH2aEDQXyScDY8u3cG87M8n_description","stringValue":"みんな電力公式"}},{"address":"AVNR0DAPNgBsCKRIvPT9wVRvDtiQBUHm8sU=","entry":{"key":"listed_GF8HGGqPLDfAjZvkLGhamioZouV6uH6vS92mZv1zA8hu_description","stringValue":"みんな電力公式"}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"dappAddress","stringValue":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC"}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"masterAddress","stringValue":"3MSvD3m1R8Z3v8SAztrt1afp28vRdsMwxAu"}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"asset_total_amount","intValue":"100000"}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MgvX2f2ExVwTMkAk6dua8yE2iRmuBV4heT","stringValue":"{\"name\":\"ママママママママっまm\",\"description\":\"retail user\"}"}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MktJgV2eTmcCqtyQaeqiiHkQ1eY3EH5Tdb","stringValue":"{\"name\":\"ママママママママっまm\",\"description\":\"retail user\"}"}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MRiqDCpFntSEud3Co8bdQygjSwB515zyS5_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MSvD3m1R8Z3v8SAztrt1afp28vRdsMwxAu_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MUJS7P4W3XyP2pnAJUGqkstSAiU4Ac2YdA_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MY34vVDzBnYxE34Ug4K1Y1GyRyVSgcfnpC_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MZyZAgAJmXmJs5gDihnMvZ7HCLxe6zVVpU_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MfF8z9y9nUUuHTeKiGFGoWXnUrRPbEcNiD_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MgvX2f2ExVwTMkAk6dua8yE2iRmuBV4heT_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3Mh5b5UttYteWjd5Mku43kajZFKX9z5WNxZ_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MjBN2kiRB6JmoEVEC42ZNMX9ibx5iZ9Mih_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MkT3qvGwdLrSs2Cfx3E29ffaM5GYrEZegz_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MktJgV2eTmcCqtyQaeqiiHkQ1eY3EH5Tdb_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3Mm9VfS5424Vn4oNKv1DSh7Htk6FhQReEuP_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3MmNtj9n49UgGapeh1Sg8Nd8jfQGDbqRTkx_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3P35F9e1QdcHkBMbYtovuMUmsxxCqo9DF1d_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3PLXmyBua1pAH4y3aHjMqJrcJEcyrWMP1EB_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3PMoTnMU6U4hx8km23iZJ6Akis6JKhcxhUn_active","boolValue":true}},{"address":"AVPewZB4PhL4dXSM6B1zWwYdeqFAtf/yW7I=","entry":{"key":"3PNVubsGCrnMHLXvg1gYidcP3G7HUC5fAuZ_active","boolValue":true}}]`)
	env := &mockRideEnvironment{
		heightFunc: func() rideInt {
			return 451323
		},
		schemeFunc: func() byte {
			return proto.StageNetScheme
		},
		blockFunc: func() rideObject {
			return blockInfoToObject(blockInfo)
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				AddingBlockHeightFunc: func() (uint64, error) {
					return 451323, nil
				},
				NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
					return false, nil
				},
				NewestFullWavesBalanceFunc: func(account proto.Recipient) (*proto.FullWavesBalance, error) {
					return &proto.FullWavesBalance{Available: 5000000000}, nil
				},
				RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
					return dp.getBool(key)
				},
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					return dp.getString(key)
				},
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					return dp.getInt(key)
				},
				RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
					return dp.getBinary(key)
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.StageNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(4, proto.StageNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		checkMessageLengthFunc: v3check,
		rideV6ActivatedFunc:    noRideV6,
	}

	code := "AAIEAAAAAAAAACsIAhIGCgQIAQEIEgUKAwEICBIDCgEIEgQKAggBEgMKAQgSAwoBCBIDCgEIAAAAGwAAAAAFYWRtaW4BAAAAIAj8cpFuriBeYVtrdATBbLEGiGP0beK8x+YDq9aX3ZNqAAAAAAtkYXBwQWRkcmVzcwkAAlgAAAABCAUAAAAEdGhpcwAAAAVieXRlcwAAAAAETk9ORQIAAAALbm8gZXhpc3RpbmcAAAAABExJU1QJAQAAABFAZXh0ck5hdGl2ZSgxMDYyKQAAAAECAAAAIzNNa3RKZ1YyZVRtY0NxdHlRYWVxaWlIa1ExZVkzRUg1VGRiAQAAAAtmZXRjaFN0cmluZwAAAAIAAAAFYWxpYXMAAAADa2V5BAAAAAckbWF0Y2gwCQAEHQAAAAIFAAAABWFsaWFzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABlN0cmluZwQAAAABYQUAAAAHJG1hdGNoMAUAAAABYQUAAAAETk9ORQEAAAAMZmV0Y2hJbnRlZ2VyAAAAAgAAAAVhbGlhcwAAAANrZXkEAAAAByRtYXRjaDAJAAQaAAAAAgUAAAAFYWxpYXMFAAAAA2tleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAFhBQAAAAckbWF0Y2gwBQAAAAFhAAAAAAAAAAAAAQAAAAlnZXRNYXN0ZXIAAAAACQEAAAALZmV0Y2hTdHJpbmcAAAACBQAAAARMSVNUAgAAAA1tYXN0ZXJBZGRyZXNzAQAAABBnZXRBY2NvdW50U3RhdHVzAAAAAQAAAAdhZGRyZXNzCQEAAAARQGV4dHJOYXRpdmUoMTA1MSkAAAACBQAAAARMSVNUCQABLAAAAAIFAAAAB2FkZHJlc3MCAAAAB19hY3RpdmUBAAAAFmdldEFzc2V0VG90YWxBbW91bnRLZXkAAAABAAAAB2Fzc2V0SWQJAAEsAAAAAgIAAAATYXNzZXRfdG90YWxfYW1vdW50XwUAAAAHYXNzZXRJZAEAAAAYZ2V0QXNzZXRUb3RhbEFtb3VudFZhbHVlAAAAAQAAAAdhc3NldElkCQEAAAAMZmV0Y2hJbnRlZ2VyAAAAAgUAAAAEdGhpcwkBAAAAFmdldEFzc2V0VG90YWxBbW91bnRLZXkAAAABBQAAAAdhc3NldElkAQAAAAtnZXRMaW1pdEtleQAAAAEAAAAHYXNzZXRJZAkAASwAAAACAgAAAAZsaW1pdF8FAAAAB2Fzc2V0SWQBAAAADWdldExpbWl0VmFsdWUAAAABAAAAB2Fzc2V0SWQJAQAAAAtmZXRjaFN0cmluZwAAAAIFAAAABHRoaXMJAQAAAAtnZXRMaW1pdEtleQAAAAEFAAAAB2Fzc2V0SWQBAAAACWdldElzc3VlcgAAAAEAAAAHYXNzZXRJZAQAAAAHJG1hdGNoMAkAA+wAAAABCQACWQAAAAEFAAAAB2Fzc2V0SWQDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABUFzc2V0BAAAAAFhBQAAAAckbWF0Y2gwCAUAAAABYQAAAAZpc3N1ZXIJAAACAAAAAQIAAAAPaW52YWxpZCBhc3NldElkAQAAAA5nZXRUb3RhbEFtb3VudAAAAAAJAQAAAAxmZXRjaEludGVnZXIAAAACBQAAAAR0aGlzAgAAAAx0b3RhbF9hbW91bnQBAAAACmdldExpc3RLZXkAAAACAAAAB2FkZHJlc3MAAAAHYXNzZXRJZAQAAAAKc2VlZFBocmFzZQkAASwAAAACBQAAAAdhZGRyZXNzBQAAAAdhc3NldElkCQABLAAAAAICAAAAB2xpc3RlZF8JAAJYAAAAAQkAAfcAAAABCQABmwAAAAEFAAAACnNlZWRQaHJhc2UBAAAAEGdldExpc3RBbW91bnRLZXkAAAABAAAAB2xpc3RLZXkJAAEsAAAAAgUAAAAHbGlzdEtleQIAAAAHX2Ftb3VudAEAAAAPZ2V0TGlzdEFzc2V0S2V5AAAAAQAAAAdsaXN0S2V5CQABLAAAAAIFAAAAB2xpc3RLZXkCAAAACF9hc3NldElkAQAAAA9nZXRMaXN0T3duZXJLZXkAAAABAAAAB2xpc3RLZXkJAAEsAAAAAgUAAAAHbGlzdEtleQIAAAAGX293bmVyAQAAABNnZXRMaXN0VW5pdFByaWNlS2V5AAAAAQAAAAdsaXN0S2V5CQABLAAAAAIFAAAAB2xpc3RLZXkCAAAACl91bml0UHJpY2UBAAAAFWdldExpc3REZXNjcmlwdGlvbktleQAAAAEAAAAHbGlzdEtleQkAASwAAAACBQAAAAdsaXN0S2V5AgAAAAxfZGVzY3JpcHRpb24BAAAAEmdldExpc3RBbW91bnRWYWx1ZQAAAAEAAAAHbGlzdEtleQkBAAAADGZldGNoSW50ZWdlcgAAAAIFAAAABHRoaXMJAQAAABBnZXRMaXN0QW1vdW50S2V5AAAAAQUAAAAHbGlzdEtleQEAAAARZ2V0TGlzdEFzc2V0VmFsdWUAAAABAAAAB2xpc3RLZXkJAQAAAAtmZXRjaFN0cmluZwAAAAIFAAAABHRoaXMJAQAAAA9nZXRMaXN0QXNzZXRLZXkAAAABBQAAAAdsaXN0S2V5AQAAABFnZXRMaXN0T3duZXJWYWx1ZQAAAAEAAAAHbGlzdEtleQkBAAAAC2ZldGNoU3RyaW5nAAAAAgUAAAAEdGhpcwkBAAAAD2dldExpc3RPd25lcktleQAAAAEFAAAAB2xpc3RLZXkBAAAAFWdldExpc3RVbml0UHJpY2VWYWx1ZQAAAAEAAAAHbGlzdEtleQkBAAAADGZldGNoSW50ZWdlcgAAAAIFAAAABHRoaXMJAQAAABNnZXRMaXN0VW5pdFByaWNlS2V5AAAAAQUAAAAHbGlzdEtleQEAAAAXZ2V0TGlzdERlc2NyaXB0aW9uVmFsdWUAAAABAAAAB2xpc3RLZXkJAQAAAAtmZXRjaFN0cmluZwAAAAIFAAAABHRoaXMJAQAAABVnZXRMaXN0RGVzY3JpcHRpb25LZXkAAAABBQAAAAdsaXN0S2V5AQAAAAp1cGRhdGVMaXN0AAAABQAAAAVvd25lcgAAAAZhbW91bnQAAAAHYXNzZXRJZAAAAAl1bml0UHJpY2UAAAALZGVzY3JpcHRpb24EAAAAB2xpc3RLZXkJAQAAAApnZXRMaXN0S2V5AAAAAgUAAAAFb3duZXIFAAAAB2Fzc2V0SWQJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAIJAQAAABBnZXRMaXN0QW1vdW50S2V5AAAAAQUAAAAHbGlzdEtleQkAAGQAAAACCQEAAAASZ2V0TGlzdEFtb3VudFZhbHVlAAAAAQUAAAAHbGlzdEtleQUAAAAGYW1vdW50CQAETAAAAAIJAQAAAAtTdHJpbmdFbnRyeQAAAAIJAQAAAA9nZXRMaXN0QXNzZXRLZXkAAAABBQAAAAdsaXN0S2V5BQAAAAdhc3NldElkCQAETAAAAAIJAQAAAAtTdHJpbmdFbnRyeQAA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"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "purchaseToken", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))

	expectedDataWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "listed_2metSrd6Gn7VDVB61LF7DkZfxP3sx7Ag9Evx9JomcdTb_amount", Value: 99500000}},
	}
	expectedTransfers := []*proto.TransferScriptAction{
		{
			Recipient: proto.NewRecipientFromAddress(proto.MustAddressFromString("3MgvX2f2ExVwTMkAk6dua8yE2iRmuBV4heT")),
			Amount:    500000,
			Asset:     *proto.NewOptionalAssetFromDigest(crypto.MustDigestFromBase58("3ZwqiyJ71v2RL9ynFfhbhrL6exVvpBXq4tMZsM8BMjS2")),
		},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    expectedTransfers,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
	}
	assert.Equal(t, expectedResult, sr)
}

func TestRecipientAddressToString(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		match tx {
		    case tr: TransferTransaction =>
		        match tr.recipient {
		            case a: Address => toString(a) == "3N61Xs9cTetvoP1uZSrtuRxxJ4A4RCR7a4G"
		            case _ => false
		        }
		    case _ => false
		}
	*/
	s := "BAQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnRyBQAAAAckbWF0Y2gwBAAAAAckbWF0Y2gxCAUAAAACdHIAAAAJcmVjaXBpZW50AwkAAAEAAAACBQAAAAckbWF0Y2gxAgAAAAdBZGRyZXNzBAAAAAFhBQAAAAckbWF0Y2gxCQAAAAAAAAIJAAQlAAAAAQUAAAABYQIAAAAjM042MVhzOWNUZXR2b1AxdVpTcnR1Unh4SjRBNFJDUjdhNEcHBzdCrWM="
	src, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	id := crypto.MustDigestFromBase58("2RW5wedbBi9PTEM9Ao5s5Y7U25FD7PepujC2CS7Qeta1")
	tx := &proto.TransferWithProofs{
		Type:    proto.TransferTransaction,
		Version: 3,
		ID:      &id,
		Proofs:  proto.NewProofs(),
		Transfer: proto.Transfer{
			SenderPK:    crypto.PublicKey{},
			AmountAsset: proto.OptionalAsset{},
			FeeAsset:    proto.OptionalAsset{},
			Timestamp:   0,
			Amount:      0,
			Fee:         0,
			Recipient:   proto.NewRecipientFromAddress(proto.MustAddressFromString("3N61Xs9cTetvoP1uZSrtuRxxJ4A4RCR7a4G")),
			Attachment:  nil,
		},
	}

	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		checkMessageLengthFunc: v3check,
		rideV6ActivatedFunc:    noRideV6,
	}

	res, err := CallVerifier(env, tree)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	assert.True(t, r.Result())
}

func TestScriptPaymentPublicKey(t *testing.T) {
	pk := crypto.MustPublicKeyFromBase58("7gYTeHxHZ2NRQdNpa6DHAxQY4K5LS6bezcsMQcUhYuo1")
	addr := proto.MustAddressFromPublicKey(proto.MainNetScheme, pk)
	asset, err := proto.NewOptionalAssetFromString("5F4PshPwzE8sQeesDPzjJN45CFVnAnqCUHJcmi7kZq22")
	require.NoError(t, err)
	rcp := proto.NewRecipientFromAddress(addr)
	action := &proto.TransferScriptAction{
		Recipient: rcp,
		Amount:    12345,
		Asset:     *asset,
	}
	id := crypto.MustDigestFromBase58("9vt45R9y63Xwcseat59BchUjfJGHSuN5LeTK6Pd6cFUM")
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &id,
		ChainID:         proto.MainNetScheme,
		SenderPK:        pk,
		ScriptRecipient: rcp,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             1300000,
		Timestamp:       1599565088614,
	}

	tr, _ := proto.NewFullScriptTransfer(action, addr, pk, tx.ID, tx.Timestamp)
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		transactionFunc: func() rideObject {
			return scriptTransferToObject(tr)
		},
		checkMessageLengthFunc: v3check,
		rideV6ActivatedFunc:    noRideV6,
	}

	code := "AQQAAAAGc2VuZGVyCQACWAAAAAEICQEAAAAUYWRkcmVzc0Zyb21QdWJsaWNLZXkAAAABCAUAAAACdHgAAAAPc2VuZGVyUHVibGljS2V5AAAABWJ5dGVzCQAAAAAAAAICAAAAIzNQNjFiNnRlMmZ2akw3YWdLSHFOY0NrcHV0Z1lzNjV4dzVSBQAAAAZzZW5kZXJlKXM0"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallVerifier(env, tree)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	require.True(t, r.Result())
}

func TestInvalidAssetInTransferScriptAction(t *testing.T) {
	txID, err := crypto.NewDigestFromBase58("AUpiEr49Jo43Q9zXKkNN23rstiq87hguvhfQqV8ov9uQ")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	sender, err := crypto.NewPublicKeyFromBase58("Hjd6p3ArqjnQAsejFwu7JcQciVVx9RaQhtMfGBCAi76z")
	require.NoError(t, err)
	address, err := proto.NewAddressFromString("3P8FF73N7ZvvNJ34vnJ3h9Tfmh7oQCnRz8E")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(address)
	arguments := proto.Arguments{}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "swapRKMTToWAVES",
		Arguments: arguments,
	}
	asset, err := proto.NewOptionalAssetFromString("2fCdmsn6maErwtLuzxoUrCBkh2vx5SvXtMKAJtN4YBgd")
	require.NoError(t, err)
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        sender,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{proto.ScriptPayment{Amount: 1000, Asset: *asset}},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1609698441420,
	}
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		thisFunc: func() rideType {
			return rideAddress(address)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			obj, err := invocationToObject(3, proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		checkMessageLengthFunc: v3check,
		rideV6ActivatedFunc:    noRideV6,
	}

	code := "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"
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)
	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "swapRKMTToWAVES", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))

	expectedTransfers := []*proto.TransferScriptAction{
		{
			Recipient: proto.NewRecipientFromAddress(proto.MustAddressFromString("3P8FF73N7ZvvNJ34vnJ3h9Tfmh7oQCnRz8E")),
			Amount:    0,
			Asset:     proto.NewOptionalAssetWaves(),
		},
	}
	expectedResult := &proto.ScriptResult{
		DataEntries:  make([]*proto.DataEntryScriptAction, 0),
		Transfers:    expectedTransfers,
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
		ErrorMsg:     proto.ScriptErrorMessage{},
	}
	assert.Equal(t, expectedResult, sr)
}

func TestOriginCaller(t *testing.T) {
	/*
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		let contract = base58'3PGZyyPg7Mx91yaNT8k3MWxSQzuzusMUyzX'

		@Callable(i)
		func call() = {
		  strict res = invoke(Address(contract),  "call", [], [])
		  match (res) {
		      case b:Boolean => if b then ([], res) else throw("fail!!!")
		      case _ => throw("not a boolean")
		    }
		}
	*/
	code1 := "AAIFAAAAAAAAAAQIAhIAAAAAAQAAAAAIY29udHJhY3QBAAAAGgFXoJWHaFIS+neTXowyvvYUIY9fLjbMmBsgAAAAAQAAAAFpAQAAAARjYWxsAAAAAAQAAAADcmVzCQAD/AAAAAQJAQAAAAdBZGRyZXNzAAAAAQUAAAAIY29udHJhY3QCAAAABGNhbGwFAAAAA25pbAUAAAADbmlsAwkAAAAAAAACBQAAAANyZXMFAAAAA3JlcwQAAAAHJG1hdGNoMAUAAAADcmVzAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAAdCb29sZWFuBAAAAAFiBQAAAAckbWF0Y2gwAwUAAAABYgkABRQAAAACBQAAAANuaWwFAAAAA3JlcwkAAAIAAAABAgAAAAdmYWlsISEhCQAAAgAAAAECAAAADW5vdCBhIGJvb2xlYW4JAAACAAAAAQIAAAAkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuAAAAAFMoVsA="
	src1, err := base64.StdEncoding.DecodeString(code1)
	require.NoError(t, err)

	/*
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func call() = {
		  ([BinaryEntry("origin-caller-address", i.originCaller.bytes), BinaryEntry("origin-caller-pk", i.originCallerPublicKey)], true)
		}
	*/
	code2 := "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAAJAAUUAAAAAgkABEwAAAACCQEAAAALQmluYXJ5RW50cnkAAAACAgAAABVvcmlnaW4tY2FsbGVyLWFkZHJlc3MICAUAAAABaQAAAAxvcmlnaW5DYWxsZXIAAAAFYnl0ZXMJAARMAAAAAgkBAAAAC0JpbmFyeUVudHJ5AAAAAgIAAAAQb3JpZ2luLWNhbGxlci1wawgFAAAAAWkAAAAVb3JpZ2luQ2FsbGVyUHVibGljS2V5BQAAAANuaWwGAAAAAAd0XdI="
	src2, err := base64.StdEncoding.DecodeString(code2)
	require.NoError(t, err)

	txID, err := crypto.NewDigestFromBase58("BuCo8EEM2VbvjJbC6VyBVa64m2fNmdSoKLSxmoshnbmv")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	senderPK, err := crypto.NewPublicKeyFromBase58("EY3etWLNnrLg4znKsncuJFXVUHiP61PYpuZTAED98QUS")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, senderPK)
	require.NoError(t, err)
	dApp1, err := proto.NewAddressFromString("3PH75p2rmMKCV2nyW4TsAdFgFtmc61mJaqA")
	require.NoError(t, err)
	dApp1PK, err := crypto.NewPublicKeyFromBase58("3GtkwhnMmG1yeozW51o4dJ1x3BDToPaLBXyBWKGdAc2e")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(dApp1)
	arguments := proto.Arguments{}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "call",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        senderPK,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1624967106278,
	}
	testInv, err := invocationToObject(5, proto.MainNetScheme, tx)
	require.NoError(t, err)
	testDAppAddress := dApp1
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		thisFunc: func() rideType {
			return rideAddress(testDAppAddress)
		},
		rideV6ActivatedFunc: noRideV6,
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			return testInv
		},
		checkMessageLengthFunc: v3check,
		setInvocationFunc: func(inv rideObject) {
			testInv = inv
		},
		setNewDAppAddressFunc: func(address proto.WavesAddress) {
			testDAppAddress = address
		},
		maxDataEntriesSizeFunc: func() int {
			return proto.MaxDataEntriesScriptActionsSizeInBytesV2
		},
		validateInternalPaymentsFunc: func() bool {
			return true
		},
		blockV5ActivatedFunc: func() bool {
			return true
		},
	}

	mockState := &MockSmartState{
		GetByteTreeFunc: func(recipient proto.Recipient) (proto.Script, error) {
			switch recipient.Address.String() {
			case "3PGZyyPg7Mx91yaNT8k3MWxSQzuzusMUyzX":
				return src2, nil
			case "3PH75p2rmMKCV2nyW4TsAdFgFtmc61mJaqA":
				return src1, nil
			default:
				return nil, errors.Errorf("unexpected address %s", recipient.String())
			}
		},
		NewestScriptPKByAddrFunc: func(addr proto.WavesAddress) (crypto.PublicKey, error) {
			switch addr {
			case senderAddress:
				return senderPK, nil
			case dApp1:
				return dApp1PK, nil
			}
			return crypto.PublicKey{}, errors.Errorf("unexpected address %s", addr.String())
		},
		NewestRecipientToAddressFunc: func(recipient proto.Recipient) (*proto.WavesAddress, error) {
			return recipient.Address, nil
		},
	}
	testState := initWrappedState(mockState, env)
	env.stateFunc = func() types.SmartState {
		return testState
	}

	tree, err := Parse(src1)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "call", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))

	expectedDataWrites := []*proto.DataEntryScriptAction{
		{
			Sender: &dApp1PK,
			Entry:  &proto.BinaryDataEntry{Key: "origin-caller-address", Value: senderAddress.Bytes()},
		},
		{
			Sender: &dApp1PK,
			Entry:  &proto.BinaryDataEntry{Key: "origin-caller-pk", Value: senderPK.Bytes()},
		},
	}

	expectedResult := &proto.ScriptResult{
		DataEntries:  expectedDataWrites,
		Transfers:    make([]*proto.TransferScriptAction, 0),
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
		ErrorMsg:     proto.ScriptErrorMessage{},
	}
	assert.Equal(t, expectedResult, sr)
}

func TestInternalPaymentsValidationFailure(t *testing.T) {
	issuerPK, err := crypto.NewPublicKeyFromBase58("Hjd6p3ArqjnQAsejFwu7JcQciVVx9RaQhtMfGBCAi76z")
	require.NoError(t, err)
	issuerAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, issuerPK)
	require.NoError(t, err)
	asset, err := crypto.NewDigestFromBase58("2fCdmsn6maErwtLuzxoUrCBkh2vx5SvXtMKAJtN4YBgd")
	require.NoError(t, err)
	/*
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		let contract = base58'3PGZyyPg7Mx91yaNT8k3MWxSQzuzusMUyzX'
		let asset = base58'2fCdmsn6maErwtLuzxoUrCBkh2vx5SvXtMKAJtN4YBgd'

		@Callable(i)
		func call() = {
			strict res = invoke(Address(contract),  "call", [], [AttachedPayment(asset, 50)])
			match (res) {
				case b:Boolean => if b then ([], res) else throw("fail!!!")
				case _ => throw("not a boolean")
			}
		}
	*/
	code1 := "AAIFAAAAAAAAAAQIAhIAAAAAAgAAAAAIY29udHJhY3QBAAAAGgFXoJWHaFIS+neTXowyvvYUIY9fLjbMmBsgAAAAAAVhc3NldAEAAAAgGKTpjSxBVaOkcUeRe5pz6g1WuOiEc9KXeZMyi74aesoAAAABAAAAAWkBAAAABGNhbGwAAAAABAAAAANyZXMJAAP8AAAABAkBAAAAB0FkZHJlc3MAAAABBQAAAAhjb250cmFjdAIAAAAEY2FsbAUAAAADbmlsCQAETAAAAAIJAQAAAA9BdHRhY2hlZFBheW1lbnQAAAACBQAAAAVhc3NldAAAAAAAAAAAMgUAAAADbmlsAwkAAAAAAAACBQAAAANyZXMFAAAAA3JlcwQAAAAHJG1hdGNoMAUAAAADcmVzAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAAdCb29sZWFuBAAAAAFiBQAAAAckbWF0Y2gwAwUAAAABYgkABRQAAAACBQAAAANuaWwFAAAAA3JlcwkAAAIAAAABAgAAAAdmYWlsISEhCQAAAgAAAAECAAAADW5vdCBhIGJvb2xlYW4JAAACAAAAAQIAAAAkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuAAAAAOq4bsI="
	src1, err := base64.StdEncoding.DecodeString(code1)
	require.NoError(t, err)

	/* On 3PGZyyPg7Mx91yaNT8k3MWxSQzuzusMUyzX
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}
	let asset = base58'2fCdmsn6maErwtLuzxoUrCBkh2vx5SvXtMKAJtN4YBgd'
	@Callable(i)
	func call() = ([ScriptTransfer(i.caller, 50, asset)], true)
	*/
	code2 := "AAIFAAAAAAAAAAQIAhIAAAAAAQAAAAAFYXNzZXQBAAAAIBik6Y0sQVWjpHFHkXuac+oNVrjohHPSl3mTMou+GnrKAAAAAQAAAAFpAQAAAARjYWxsAAAAAAkABRQAAAACCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAAAAAAMgUAAAAFYXNzZXQFAAAAA25pbAYAAAAAHQNJXQ=="
	src2, err := base64.StdEncoding.DecodeString(code2)
	require.NoError(t, err)

	txID, err := crypto.NewDigestFromBase58("BuCo8EEM2VbvjJbC6VyBVa64m2fNmdSoKLSxmoshnbmv")
	require.NoError(t, err)
	proofs := proto.NewProofs()
	senderPK, err := crypto.NewPublicKeyFromBase58("EY3etWLNnrLg4znKsncuJFXVUHiP61PYpuZTAED98QUS")
	require.NoError(t, err)
	senderAddress, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, senderPK)
	require.NoError(t, err)
	dApp1, err := proto.NewAddressFromString("3PH75p2rmMKCV2nyW4TsAdFgFtmc61mJaqA")
	require.NoError(t, err)
	dApp1PK, err := crypto.NewPublicKeyFromBase58("3GtkwhnMmG1yeozW51o4dJ1x3BDToPaLBXyBWKGdAc2e")
	require.NoError(t, err)
	dApp2, err := proto.NewAddressFromString("3PGZyyPg7Mx91yaNT8k3MWxSQzuzusMUyzX")
	require.NoError(t, err)
	dApp2PK, err := crypto.NewPublicKeyFromBase58("EmRAgwaLuMrvnkeorjU9UmmGnRMXMu5ctEqkYRxnG2za")
	require.NoError(t, err)
	recipient := proto.NewRecipientFromAddress(dApp1)
	arguments := proto.Arguments{}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "call",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &txID,
		Proofs:          proofs,
		ChainID:         proto.MainNetScheme,
		SenderPK:        senderPK,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1624967106278,
	}
	var testState *WrappedState

	testInv, err := invocationToObject(5, proto.MainNetScheme, tx)
	require.NoError(t, err)
	testDAppAddress := dApp1
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.MainNetScheme
		},
		thisFunc: func() rideType {
			return rideAddress(testDAppAddress)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.MainNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			return testInv
		},
		checkMessageLengthFunc: v3check,
		setInvocationFunc: func(inv rideObject) {
			testInv = inv
		},
		blockV5ActivatedFunc: func() bool {
			return true
		},
		rideV6ActivatedFunc: noRideV6,
		setNewDAppAddressFunc: func(address proto.WavesAddress) {
			testDAppAddress = address
			testState.cle = rideAddress(address)
		},
		validateInternalPaymentsFunc: func() bool {
			return true
		},
		maxDataEntriesSizeFunc: func() int {
			return proto.MaxDataEntriesScriptActionsSizeInBytesV2
		},
	}

	mockState := &MockSmartState{
		GetByteTreeFunc: func(recipient proto.Recipient) (proto.Script, error) {
			switch recipient.Address.String() {
			case "3PH75p2rmMKCV2nyW4TsAdFgFtmc61mJaqA":
				return src1, nil
			case "3PGZyyPg7Mx91yaNT8k3MWxSQzuzusMUyzX":
				return src2, nil
			default:
				return nil, errors.Errorf("unexpected address %s", recipient.String())
			}
		},
		NewestScriptPKByAddrFunc: func(addr proto.WavesAddress) (crypto.PublicKey, error) {
			switch addr {
			case senderAddress:
				return senderPK, nil
			case dApp1:
				return dApp1PK, nil
			case dApp2:
				return dApp2PK, nil
			}
			return crypto.PublicKey{}, errors.Errorf("unexpected address %s", addr.String())
		},
		NewestRecipientToAddressFunc: func(recipient proto.Recipient) (*proto.WavesAddress, error) {
			return recipient.Address, nil
		},
		NewestAssetInfoFunc: func(assetID crypto.Digest) (*proto.AssetInfo, error) {
			if assetID == asset {
				return &proto.AssetInfo{
					ID:              assetID,
					Quantity:        1000000,
					Decimals:        2,
					Issuer:          issuerAddress,
					IssuerPublicKey: issuerPK,
					Reissuable:      false,
					Scripted:        false,
					Sponsored:       false,
				}, nil
			}
			return nil, errors.Errorf("unexpected asset '%s'", assetID.String())
		},
		NewestAssetBalanceFunc: func(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
			if assetID == asset {
				switch account.String() {
				case "3PH75p2rmMKCV2nyW4TsAdFgFtmc61mJaqA":
					return 0, nil
				case "3PGZyyPg7Mx91yaNT8k3MWxSQzuzusMUyzX":
					return 0, nil
				default:
					return 0, errors.New("unexpected account")
				}
			}
			return 0, errors.Errorf("unexpected asset '%s'", assetID.String())
		},
	}
	testState = initWrappedState(mockState, env)
	env.stateFunc = func() types.SmartState {
		return testState
	}

	tree, err := Parse(src1)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "call", arguments)
	// Expecting validation error for the switched on internal payments validation
	require.Nil(t, res)
	require.Error(t, err)

	// Turning off internal payments validation
	env.validateInternalPaymentsFunc = func() bool {
		return false
	}
	testInv, err = invocationToObject(5, proto.MainNetScheme, tx)
	require.NoError(t, err)
	testDAppAddress = dApp1
	testState = initWrappedState(mockState, env)
	env.stateFunc = func() types.SmartState {
		return testState
	}

	tree, err = Parse(src1)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err = CallFunction(env, tree, "call", arguments)
	// No error is expected in this case
	require.NoError(t, err)
	require.IsType(t, DAppResult{}, res)
}

func TestAliasesInInvokes(t *testing.T) {
	_, dApp1PK, dApp1 := makeAddressAndPK(t, "DAPP1")    // 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz
	_, dApp2PK, dApp2 := makeAddressAndPK(t, "DAPP2")    // 3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1
	_, senderPK, sender := makeAddressAndPK(t, "SENDER") // 3N8CkZAyS4XcDoJTJoKNuNk2xmNKmQj7myW

	caller := proto.NewAlias(proto.TestNetScheme, "caller")
	callee := proto.NewAlias(proto.TestNetScheme, "callee")
	/* On dApp1 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let callee = Alias("callee")

	@Callable(i)
	func call() = {
		strict res = invoke(callee,  "call", [], [])
		match (res) {
			case b:Boolean => if b then ([], res) else throw("fail!!!")
			case _ => throw("not a boolean")
		}
	}
	*/
	code1 := "AAIFAAAAAAAAAAQIAhIAAAAAAQAAAAAGY2FsbGVlCQEAAAAFQWxpYXMAAAABAgAAAAZjYWxsZWUAAAABAAAAAWkBAAAABGNhbGwAAAAABAAAAANyZXMJAAP8AAAABAUAAAAGY2FsbGVlAgAAAARjYWxsBQAAAANuaWwFAAAAA25pbAMJAAAAAAAAAgUAAAADcmVzBQAAAANyZXMEAAAAByRtYXRjaDAFAAAAA3JlcwMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAHQm9vbGVhbgQAAAABYgUAAAAHJG1hdGNoMAMFAAAAAWIJAAUUAAAAAgUAAAADbmlsBQAAAANyZXMJAAACAAAAAQIAAAAHZmFpbCEhIQkAAAIAAAABAgAAAA1ub3QgYSBib29sZWFuCQAAAgAAAAECAAAAJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgAAAAATG5XV"
	src1, err := base64.StdEncoding.DecodeString(code1)
	require.NoError(t, err)

	/* On dApp2 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func call() = ([ScriptTransfer(i.caller, 100000000, unit)], true)
	*/
	code2 := "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAAJAAUUAAAAAgkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAAAX14QAFAAAABHVuaXQFAAAAA25pbAYAAAAAvdgXFg=="
	src2, err := base64.StdEncoding.DecodeString(code2)
	require.NoError(t, err)

	recipient := proto.NewRecipientFromAlias(*caller)
	arguments := proto.Arguments{}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "call",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              makeRandomTxID(t),
		Proofs:          proto.NewProofs(),
		ChainID:         proto.TestNetScheme,
		SenderPK:        senderPK,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1624967106278,
	}
	testInv, err := invocationToObject(5, proto.TestNetScheme, tx)
	require.NoError(t, err)
	testDAppAddress := dApp1
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		thisFunc: func() rideType {
			return rideAddress(testDAppAddress)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			return testInv
		},
		blockV5ActivatedFunc: func() bool {
			return true
		},
		rideV6ActivatedFunc:    noRideV6,
		checkMessageLengthFunc: v3check,
		setInvocationFunc: func(inv rideObject) {
			testInv = inv
		},
		validateInternalPaymentsFunc: func() bool {
			return true
		},
		maxDataEntriesSizeFunc: func() int {
			return proto.MaxDataEntriesScriptActionsSizeInBytesV2
		},
	}

	mockState := &MockSmartState{
		GetByteTreeFunc: func(recipient proto.Recipient) (proto.Script, error) {
			switch recipient.String() {
			case dApp1.String():
				return src1, nil
			case dApp2.String():
				return src2, nil
			case "alias:T:callee":
				return src2, nil
			default:
				return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
			}
		},
		NewestScriptPKByAddrFunc: func(addr proto.WavesAddress) (crypto.PublicKey, error) {
			switch addr {
			case sender:
				return senderPK, nil
			case dApp1:
				return dApp1PK, nil
			case dApp2:
				return dApp2PK, nil
			default:
				return crypto.PublicKey{}, errors.Errorf("unexpected address %s", addr.String())
			}
		},
		NewestRecipientToAddressFunc: func(recipient proto.Recipient) (*proto.WavesAddress, error) {
			switch recipient.String() {
			case dApp1.String():
				return &dApp1, nil
			case dApp2.String():
				return &dApp2, nil
			case "alias:T:callee":
				return &dApp2, nil
			default:
				return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
			}
		},
		NewestAddrByAliasFunc: func(alias proto.Alias) (proto.WavesAddress, error) {
			switch alias.String() {
			case caller.String():
				return dApp1, nil
			case callee.String():
				return dApp2, nil
			default:
				return proto.WavesAddress{}, errors.Errorf("unexpected alias '%s'", alias.String())
			}
		},
		NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
			switch account.String() {
			case dApp1.String():
				return 0, nil
			case caller.String():
				return 0, nil
			case dApp2.String():
				return 100_000_000_000, nil
			case callee.String():
				return 100_000_000_000, nil
			default:
				return 0, errors.Errorf("unexpected account '%s'", account.String())
			}
		},
	}
	testState := initWrappedState(mockState, env)
	env.stateFunc = func() types.SmartState {
		return testState
	}
	env.setNewDAppAddressFunc = func(address proto.WavesAddress) {
		testDAppAddress = address
		testState.cle = rideAddress(address) // We have to update wrapped state's `cle`
	}

	tree, err := Parse(src1)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "call", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedResult := &proto.ScriptResult{
		DataEntries:  make([]*proto.DataEntryScriptAction, 0),
		Transfers:    []*proto.TransferScriptAction{{Sender: &dApp2PK, Recipient: proto.NewRecipientFromAddress(dApp1), Amount: 100_000_000}},
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
		ErrorMsg:     proto.ScriptErrorMessage{},
	}
	assert.Equal(t, expectedResult, sr)
}

// makeAddressAndPK creates keys and an address on TestNet from given string as seed
func makeAddressAndPK(t *testing.T, s string) (crypto.SecretKey, crypto.PublicKey, proto.WavesAddress) {
	sk, pk, err := crypto.GenerateKeyPair([]byte(s))
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, pk)
	require.NoError(t, err)
	return sk, pk, addr
}

func makeRandomTxID(t *testing.T) *crypto.Digest {
	b := make([]byte, crypto.DigestSize)
	_, err := rand.Read(b)
	require.NoError(t, err)
	d, err := crypto.NewDigestFromBytes(b)
	require.NoError(t, err)
	return &d
}

func TestIssueAndTransferInInvoke(t *testing.T) {
	_, dApp1PK, dApp1 := makeAddressAndPK(t, "DAPP1")    // 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz
	_, dApp2PK, dApp2 := makeAddressAndPK(t, "DAPP2")    // 3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1
	_, dApp3PK, dApp3 := makeAddressAndPK(t, "DAPP3")    // 3N186hYM5PFwGdkVUsLJaBvpPEECrSj5CJh
	_, senderPK, sender := makeAddressAndPK(t, "SENDER") // 3N8CkZAyS4XcDoJTJoKNuNk2xmNKmQj7myW

	/* On dApp1 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let farm = Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1')
	let callee = Address(base58'3N186hYM5PFwGdkVUsLJaBvpPEECrSj5CJh')

	@Callable(i)
	func call() = {
		strict res1 = invoke(farm, "farm", [], []) # Receiving a freashly issued NFT asset
		match (res1) {
			case b1: ByteVector => {
				strict res2 = invoke(callee,  "call", [b1], [AttachedPayment(b1, 1)]) # Use the asset to pay for second call and get it back
				match (res2) {
					case b2: Boolean => if b2 then ([], res2) else throw("fail!!!")
					case _ => throw("not a Boolean")
				}
			}
			case _ => throw("not a ByteVector")
		}
	}
	*/
	code1 := "AAIFAAAAAAAAAAQIAhIAAAAAAgAAAAAEZmFybQkBAAAAB0FkZHJlc3MAAAABAQAAABoBVMByBn03y+jAvm4M5s8/31mxeRh33VavrgAAAAAGY2FsbGVlCQEAAAAHQWRkcmVzcwAAAAEBAAAAGgFUeu8lmsRjc2kucGmTq6Am5fkIjxQl3OMuAAAAAQAAAAFpAQAAAARjYWxsAAAAAAQAAAAEcmVzMQkAA/wAAAAEBQAAAARmYXJtAgAAAARmYXJtBQAAAANuaWwFAAAAA25pbAMJAAAAAAAAAgUAAAAEcmVzMQUAAAAEcmVzMQQAAAAHJG1hdGNoMAUAAAAEcmVzMQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAKQnl0ZVZlY3RvcgQAAAACYjEFAAAAByRtYXRjaDAEAAAABHJlczIJAAP8AAAABAUAAAAGY2FsbGVlAgAAAARjYWxsCQAETAAAAAIFAAAAAmIxBQAAAANuaWwJAARMAAAAAgkBAAAAD0F0dGFjaGVkUGF5bWVudAAAAAIFAAAAAmIxAAAAAAAAAAABBQAAAANuaWwDCQAAAAAAAAIFAAAABHJlczIFAAAABHJlczIEAAAAByRtYXRjaDEFAAAABHJlczIDCQAAAQAAAAIFAAAAByRtYXRjaDECAAAAB0Jvb2xlYW4EAAAAAmIyBQAAAAckbWF0Y2gxAwUAAAACYjIJAAUUAAAAAgUAAAADbmlsBQAAAARyZXMyCQAAAgAAAAECAAAAB2ZhaWwhISEJAAACAAAAAQIAAAANbm90IGEgQm9vbGVhbgkAAAIAAAABAgAAACRTdHJpY3QgdmFsdWUgaXMgbm90IGVxdWFsIHRvIGl0c2VsZi4JAAACAAAAAQIAAAAQbm90IGEgQnl0ZVZlY3RvcgkAAAIAAAABAgAAACRTdHJpY3QgdmFsdWUgaXMgbm90IGVxdWFsIHRvIGl0c2VsZi4AAAAAcrJ1zA=="
	src1, err := base64.StdEncoding.DecodeString(code1)
	require.NoError(t, err)

	/* On dApp2 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func farm() = {
	    let issue = Issue("TEST_ASSET", "ASSET FOR INTEGRATION TESTING", 1, 0, false, unit, 0)
	    let assetId = calculateAssetId(issue)
	    ([issue, ScriptTransfer(i.caller, 1, assetId)], assetId)
	}
	*/
	code2 := "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEZmFybQAAAAAEAAAABWlzc3VlCQAEQwAAAAcCAAAAClRFU1RfQVNTRVQCAAAAHUFTU0VUIEZPUiBJTlRFR1JBVElPTiBURVNUSU5HAAAAAAAAAAABAAAAAAAAAAAABwUAAAAEdW5pdAAAAAAAAAAAAAQAAAAHYXNzZXRJZAkABDgAAAABBQAAAAVpc3N1ZQkABRQAAAACCQAETAAAAAIFAAAABWlzc3VlCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAAAAAAAQUAAAAHYXNzZXRJZAUAAAADbmlsBQAAAAdhc3NldElkAAAAALylLbk="
	src2, err := base64.StdEncoding.DecodeString(code2)
	require.NoError(t, err)
	nft, err := crypto.NewDigestFromBase58("7tEQngNz2bMxwr2vUdP6GkcY4s25EuhNk1aWJoqZusYD")
	require.NoError(t, err)

	/* On dApp3 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func call(id: ByteVector) = ([ScriptTransfer(i.caller, 1, id)], true)
	*/
	code3 := "AAIFAAAAAAAAAAcIAhIDCgECAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAEAAAACaWQJAAUUAAAAAgkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAAAAAAAEFAAAAAmlkBQAAAANuaWwGAAAAAMcyoF8="
	src3, err := base64.StdEncoding.DecodeString(code3)
	require.NoError(t, err)

	recipient := proto.NewRecipientFromAddress(dApp1)
	arguments := proto.Arguments{}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "call",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &crypto.Digest{},
		Proofs:          proto.NewProofs(),
		ChainID:         proto.TestNetScheme,
		SenderPK:        senderPK,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1624967106278,
	}
	testInv, err := invocationToObject(5, proto.TestNetScheme, tx)
	require.NoError(t, err)
	testDAppAddress := dApp1
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		thisFunc: func() rideType {
			return rideAddress(testDAppAddress)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			return testInv
		},
		checkMessageLengthFunc: v3check,
		setInvocationFunc: func(inv rideObject) {
			testInv = inv
		},
		blockV5ActivatedFunc: func() bool {
			return true
		},
		rideV6ActivatedFunc: noRideV6,
		validateInternalPaymentsFunc: func() bool {
			return true
		},
		txIDFunc: func() rideType {
			return rideBytes(tx.ID.Bytes())
		},
		maxDataEntriesSizeFunc: func() int {
			return proto.MaxDataEntriesScriptActionsSizeInBytesV2
		},
	}

	mockState := &MockSmartState{
		GetByteTreeFunc: func(recipient proto.Recipient) (proto.Script, error) {
			switch recipient.String() {
			case dApp1.String():
				return src1, nil
			case dApp2.String():
				return src2, nil
			case dApp3.String():
				return src3, nil
			default:
				return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
			}
		},
		NewestScriptPKByAddrFunc: func(addr proto.WavesAddress) (crypto.PublicKey, error) {
			switch addr {
			case sender:
				return senderPK, nil
			case dApp1:
				return dApp1PK, nil
			case dApp2:
				return dApp2PK, nil
			case dApp3:
				return dApp3PK, nil
			default:
				return crypto.PublicKey{}, errors.Errorf("unexpected address %s", addr.String())
			}
		},
		NewestRecipientToAddressFunc: func(recipient proto.Recipient) (*proto.WavesAddress, error) {
			switch recipient.String() {
			case dApp1.String():
				return &dApp1, nil
			case dApp2.String():
				return &dApp2, nil
			case dApp3.String():
				return &dApp3, nil
			default:
				return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
			}
		},
		NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
			return 0, nil
		},
		NewestAssetBalanceFunc: func(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
			if assetID == nft {
				return 0, nil
			}
			return 0, errors.Errorf("unxepected asset '%s'", assetID.String())
		},
		NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
			switch assetID {
			case nft:
				return false, nil
			default:
				return false, errors.Errorf("unexpected asset '%s'", assetID.String())
			}
		},
	}
	testState := initWrappedState(mockState, env)
	env.stateFunc = func() types.SmartState {
		return testState
	}
	env.setNewDAppAddressFunc = func(address proto.WavesAddress) {
		testDAppAddress = address
		testState.cle = rideAddress(address) // We have to update wrapped state's `cle`
	}

	tree, err := Parse(src1)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "call", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	nftOA := proto.NewOptionalAssetFromDigest(nft)
	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []*proto.AttachedPaymentScriptAction{{Sender: &dApp1PK, Recipient: proto.NewRecipientFromAddress(dApp3), Amount: 1, Asset: *nftOA}}, ap)
	expectedResult := &proto.ScriptResult{
		DataEntries:  make([]*proto.DataEntryScriptAction, 0),
		Transfers:    []*proto.TransferScriptAction{{Sender: &dApp2PK, Recipient: proto.NewRecipientFromAddress(dApp1), Amount: 1, Asset: *nftOA}, {Sender: &dApp3PK, Recipient: proto.NewRecipientFromAddress(dApp1), Amount: 1, Asset: *nftOA}},
		Issues:       []*proto.IssueScriptAction{{Sender: &dApp2PK, ID: nft, Name: "TEST_ASSET", Description: "ASSET FOR INTEGRATION TESTING", Quantity: 1, Decimals: 0, Reissuable: false}},
		Reissues:     make([]*proto.ReissueScriptAction, 0),
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
		ErrorMsg:     proto.ScriptErrorMessage{},
	}
	assert.Equal(t, expectedResult, sr)
}

func TestTransferUnavailableFundsInInvoke(t *testing.T) {
	_, dApp1PK, dApp1 := makeAddressAndPK(t, "DAPP1")    // 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz
	_, dApp2PK, dApp2 := makeAddressAndPK(t, "DAPP2")    // 3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1
	_, senderPK, sender := makeAddressAndPK(t, "SENDER") // 3N8CkZAyS4XcDoJTJoKNuNk2xmNKmQj7myW

	/* On dApp1 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let dApp = Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1')

	@Callable(i)
	func call() = {
	  strict r1 = invoke(dApp, "loan", [100], nil)
	  let balance = wavesBalance(this)
	  strict r2 = invoke(dApp, "back", [], [AttachedPayment(unit, 100)])
	  [IntegerEntry("balance", balance.available)]
	}
	*/
	code1 := "AAIFAAAAAAAAAAQIAhIAAAAAAQAAAAAEZEFwcAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVMByBn03y+jAvm4M5s8/31mxeRh33VavrgAAAAEAAAABaQEAAAAEY2FsbAAAAAAEAAAAAnIxCQAD/AAAAAQFAAAABGRBcHACAAAABGxvYW4JAARMAAAAAgAAAAAAAAAAZAUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAAnIxBQAAAAJyMQQAAAAHYmFsYW5jZQkAA+8AAAABBQAAAAR0aGlzBAAAAAJyMgkAA/wAAAAEBQAAAARkQXBwAgAAAARiYWNrBQAAAANuaWwJAARMAAAAAgkBAAAAD0F0dGFjaGVkUGF5bWVudAAAAAIFAAAABHVuaXQAAAAAAAAAAGQFAAAAA25pbAMJAAAAAAAAAgUAAAACcjIFAAAAAnIyCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAAdiYWxhbmNlCAUAAAAHYmFsYW5jZQAAAAlhdmFpbGFibGUFAAAAA25pbAkAAAIAAAABAgAAACRTdHJpY3QgdmFsdWUgaXMgbm90IGVxdWFsIHRvIGl0c2VsZi4JAAACAAAAAQIAAAAkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuAAAAAALjV2o="
	src1, err := base64.StdEncoding.DecodeString(code1)
	require.NoError(t, err)

	/* On dApp2 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func loan(a: Int) =
	{
	  [ScriptTransfer(i.caller, a, unit)]
	}

	@Callable(i)
	func back() = []
	*/
	code2 := "AAIFAAAAAAAAABsIAhIDCgEBEgAaBwoCYTESAWkaBwoCYTISAWEAAAAAAAAAAgAAAAJhMQEAAAAEbG9hbgAAAAEAAAACYTIJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAmExAAAABmNhbGxlcgUAAAACYTIFAAAABHVuaXQFAAAAA25pbAAAAAJhMQEAAAAEYmFjawAAAAAFAAAAA25pbAAAAACBSAmD"
	src2, err := base64.StdEncoding.DecodeString(code2)
	require.NoError(t, err)

	recipient := proto.NewRecipientFromAddress(dApp1)
	arguments := proto.Arguments{}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "call",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              &crypto.Digest{},
		Proofs:          proto.NewProofs(),
		ChainID:         proto.TestNetScheme,
		SenderPK:        senderPK,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1624967106278,
	}
	testInv, err := invocationToObject(5, proto.TestNetScheme, tx)
	require.NoError(t, err)
	testDAppAddress := dApp1
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		thisFunc: func() rideType {
			return rideAddress(testDAppAddress)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			return testInv
		},
		checkMessageLengthFunc: v3check,
		setInvocationFunc: func(inv rideObject) {
			testInv = inv
		},
		validateInternalPaymentsFunc: func() bool {
			return true
		},
		txIDFunc: func() rideType {
			return rideBytes(tx.ID.Bytes())
		},
		maxDataEntriesSizeFunc: func() int {
			return proto.MaxDataEntriesScriptActionsSizeInBytesV2
		},
		blockV5ActivatedFunc: func() bool {
			return true
		},
		rideV6ActivatedFunc: func() bool {
			return true
		},
	}

	mockState := &MockSmartState{
		GetByteTreeFunc: func(recipient proto.Recipient) (proto.Script, error) {
			switch recipient.String() {
			case dApp1.String():
				return src1, nil
			case dApp2.String():
				return src2, nil
			default:
				return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
			}
		},
		NewestScriptPKByAddrFunc: func(addr proto.WavesAddress) (crypto.PublicKey, error) {
			switch addr {
			case sender:
				return senderPK, nil
			case dApp1:
				return dApp1PK, nil
			case dApp2:
				return dApp2PK, nil
			default:
				return crypto.PublicKey{}, errors.Errorf("unexpected address %s", addr.String())
			}
		},
		NewestRecipientToAddressFunc: func(recipient proto.Recipient) (*proto.WavesAddress, error) {
			switch recipient.String() {
			case dApp1.String():
				return &dApp1, nil
			case dApp2.String():
				return &dApp2, nil
			default:
				return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
			}
		},
		NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
			return 0, nil
		},
		NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
			return false, errors.Errorf("unexpected asset '%s'", assetID.String())
		},
	}
	testState := initWrappedState(mockState, env)
	env.stateFunc = func() types.SmartState {
		return testState
	}
	env.setNewDAppAddressFunc = func(address proto.WavesAddress) {
		testDAppAddress = address
		testState.cle = rideAddress(address) // We have to update wrapped state's `cle`
	}

	tree, err := Parse(src1)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "call", arguments)
	require.Nil(t, res)
	require.Error(t, err)
	assert.EqualError(t, err, "invoke: failed to apply actions: failed to pass validation of transfer action: not enough money in the DApp, balance of DApp with address 3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1 is 0 and it tried to transfer asset WAVES to 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz, amount of 100")
	assert.Equal(t, strings.Join(EvaluationErrorCallStack(err), ";"), "failed to evaluate block after declaration of variable 'r1';failed to estimate the condition of if;failed to call system function '0';failed to materialize argument 1;failed to evaluate expression of scope value 'r1';failed to call system function '1020'")
}

func TestBurnAndFailOnTransferInInvokeAfterRideV6(t *testing.T) {
	_, dApp1PK, dApp1 := makeAddressAndPK(t, "DAPP1")    // 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz
	_, dApp2PK, dApp2 := makeAddressAndPK(t, "DAPP2")    // 3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1
	_, senderPK, sender := makeAddressAndPK(t, "SENDER") // 3N8CkZAyS4XcDoJTJoKNuNk2xmNKmQj7myW
	asset, err := crypto.NewDigestFromBase58("HXa5senn3qfi4sKPPLADnTaYnT2foBrhXnMymqFgpVp8")
	require.NoError(t, err)

	/* On dApp1 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let callee = Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1')
	let asset = base58'HXa5senn3qfi4sKPPLADnTaYnT2foBrhXnMymqFgpVp8'

	@Callable(i)
	func call() = {
		strict res = invoke(callee,  "call", [], [AttachedPayment(asset, 1)]) # Send the asset for burning and not transferign back
		match (res) {
			case b: Boolean => if b then ([], res) else throw("fail!!!")
			case _ => throw("not a Boolean")
		}
	}
	*/
	code1 := "AAIFAAAAAAAAAAQIAhIAAAAAAgAAAAAGY2FsbGVlCQEAAAAHQWRkcmVzcwAAAAEBAAAAGgFUwHIGfTfL6MC+bgzmzz/fWbF5GHfdVq+uAAAAAAVhc3NldAEAAAAg9Y/SxOzTV4ajXPwTZa80xxl1ur65XafAcNuNl2uQEiUAAAABAAAAAWkBAAAABGNhbGwAAAAABAAAAANyZXMJAAP8AAAABAUAAAAGY2FsbGVlAgAAAARjYWxsBQAAAANuaWwJAARMAAAAAgkBAAAAD0F0dGFjaGVkUGF5bWVudAAAAAIFAAAABWFzc2V0AAAAAAAAAAABBQAAAANuaWwDCQAAAAAAAAIFAAAAA3JlcwUAAAADcmVzBAAAAAckbWF0Y2gwBQAAAANyZXMDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAB0Jvb2xlYW4EAAAAAWIFAAAAByRtYXRjaDADBQAAAAFiCQAFFAAAAAIFAAAAA25pbAUAAAADcmVzCQAAAgAAAAECAAAAB2ZhaWwhISEJAAACAAAAAQIAAAANbm90IGEgQm9vbGVhbgkAAAIAAAABAgAAACRTdHJpY3QgdmFsdWUgaXMgbm90IGVxdWFsIHRvIGl0c2VsZi4AAAAAX+9VkA=="
	src1, err := base64.StdEncoding.DecodeString(code1)
	require.NoError(t, err)

	/* On dApp2 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func call() = if (size(i.payments) == 1) then {
	    let assetID = value(i.payments[0].assetId)
	    let amount = i.payments[0].amount
	    let burn = Burn(assetID, amount)
	    ([burn, ScriptTransfer(i.caller, amount, assetID)], true)
	} else throw("invalid number of payments")
	*/
	code2 := "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAADCQAAAAAAAAIJAAGQAAAAAQgFAAAAAWkAAAAIcGF5bWVudHMAAAAAAAAAAAEEAAAAB2Fzc2V0SUQJAQAAAAV2YWx1ZQAAAAEICQABkQAAAAIIBQAAAAFpAAAACHBheW1lbnRzAAAAAAAAAAAAAAAAB2Fzc2V0SWQEAAAABmFtb3VudAgJAAGRAAAAAggFAAAAAWkAAAAIcGF5bWVudHMAAAAAAAAAAAAAAAAGYW1vdW50BAAAAARidXJuCQEAAAAEQnVybgAAAAIFAAAAB2Fzc2V0SUQFAAAABmFtb3VudAkABRQAAAACCQAETAAAAAIFAAAABGJ1cm4JAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyBQAAAAZhbW91bnQFAAAAB2Fzc2V0SUQFAAAAA25pbAYJAAACAAAAAQIAAAAaaW52YWxpZCBudW1iZXIgb2YgcGF5bWVudHMAAAAAe7xLlQ=="
	src2, err := base64.StdEncoding.DecodeString(code2)
	require.NoError(t, err)

	recipient := proto.NewRecipientFromAddress(dApp1)
	arguments := proto.Arguments{}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "call",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              makeRandomTxID(t),
		Proofs:          proto.NewProofs(),
		ChainID:         proto.TestNetScheme,
		SenderPK:        senderPK,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1624967106278,
	}
	testInv, err := invocationToObject(5, proto.TestNetScheme, tx)
	require.NoError(t, err)
	testDAppAddress := dApp1
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		thisFunc: func() rideType {
			return rideAddress(testDAppAddress)
		},
		blockV5ActivatedFunc: func() bool {
			return true
		},
		rideV6ActivatedFunc: noRideV6,
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			return testInv
		},
		checkMessageLengthFunc: v3check,
		setInvocationFunc: func(inv rideObject) {
			testInv = inv
		},
		validateInternalPaymentsFunc: func() bool {
			return true
		},
		maxDataEntriesSizeFunc: func() int {
			return proto.MaxDataEntriesScriptActionsSizeInBytesV2
		},
	}

	mockState := &MockSmartState{
		GetByteTreeFunc: func(recipient proto.Recipient) (proto.Script, error) {
			switch recipient.String() {
			case dApp1.String():
				return src1, nil
			case dApp2.String():
				return src2, nil
			default:
				return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
			}
		},
		NewestScriptPKByAddrFunc: func(addr proto.WavesAddress) (crypto.PublicKey, error) {
			switch addr {
			case sender:
				return senderPK, nil
			case dApp1:
				return dApp1PK, nil
			case dApp2:
				return dApp2PK, nil
			default:
				return crypto.PublicKey{}, errors.Errorf("unexpected address %s", addr.String())
			}
		},
		NewestRecipientToAddressFunc: func(recipient proto.Recipient) (*proto.WavesAddress, error) {
			switch recipient.String() {
			case dApp1.String():
				return &dApp1, nil
			case dApp2.String():
				return &dApp2, nil
			default:
				return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
			}
		},
		NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
			return 0, nil
		},
		NewestAssetBalanceFunc: func(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
			if assetID != asset {
				return 0, errors.Errorf("unxepected asset '%s'", assetID.String())
			}
			switch {
			case account.Address.Equal(dApp1):
				return 1, nil
			case account.Address.Equal(dApp2):
				return 0, nil
			default:
				return 0, errors.Errorf("unexpected account '%s'", account.String())
			}
		},
		NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
			switch assetID {
			case asset:
				return false, nil
			default:
				return false, errors.Errorf("unexpected asset '%s'", assetID.String())
			}
		},
		NewestAssetInfoFunc: func(assetID crypto.Digest) (*proto.AssetInfo, error) {
			if assetID != asset {
				return nil, errors.Errorf("unexpected asset '%s'", assetID.String())
			}
			return &proto.AssetInfo{
				ID:              assetID,
				Quantity:        10,
				Decimals:        2,
				Issuer:          dApp1,
				IssuerPublicKey: dApp1PK,
				Reissuable:      false,
				Scripted:        false,
				Sponsored:       false,
			}, nil
		},
	}
	testState := initWrappedState(mockState, env)
	env.stateFunc = func() types.SmartState {
		return testState
	}
	env.setNewDAppAddressFunc = func(address proto.WavesAddress) {
		testDAppAddress = address
		testState.cle = rideAddress(address) // We have to update wrapped state's `cle`
	}

	tree, err := Parse(src1)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "call", arguments)
	require.Nil(t, res)
	require.Error(t, err)
}

func TestReissueInInvoke(t *testing.T) {
	_, dApp1PK, dApp1 := makeAddressAndPK(t, "DAPP1")    // 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz
	_, dApp2PK, dApp2 := makeAddressAndPK(t, "DAPP2")    // 3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1
	_, senderPK, sender := makeAddressAndPK(t, "SENDER") // 3N8CkZAyS4XcDoJTJoKNuNk2xmNKmQj7myW
	asset, err := crypto.NewDigestFromBase58("HXa5senn3qfi4sKPPLADnTaYnT2foBrhXnMymqFgpVp8")
	require.NoError(t, err)

	/* On dApp1 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let callee = Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1')
	let asset = base58'HXa5senn3qfi4sKPPLADnTaYnT2foBrhXnMymqFgpVp8'

	@Callable(i)
	func call() = {
		strict res = invoke(callee,  "call", [], []) # Requesting transfer of a freshly issued asset
		match (res) {
			case b: Boolean => if b then ([ScriptTransfer(i.caller, 1, asset)], res) else throw("fail!!!")
			case _ => throw("not a Boolean")
		}
	}
	*/
	code1 := "AAIFAAAAAAAAAAQIAhIAAAAAAgAAAAAGY2FsbGVlCQEAAAAHQWRkcmVzcwAAAAEBAAAAGgFUwHIGfTfL6MC+bgzmzz/fWbF5GHfdVq+uAAAAAAVhc3NldAEAAAAg9Y/SxOzTV4ajXPwTZa80xxl1ur65XafAcNuNl2uQEiUAAAABAAAAAWkBAAAABGNhbGwAAAAABAAAAANyZXMJAAP8AAAABAUAAAAGY2FsbGVlAgAAAARjYWxsBQAAAANuaWwFAAAAA25pbAMJAAAAAAAAAgUAAAADcmVzBQAAAANyZXMEAAAAByRtYXRjaDAFAAAAA3JlcwMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAHQm9vbGVhbgQAAAABYgUAAAAHJG1hdGNoMAMFAAAAAWIJAAUUAAAAAgkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAAAAAAAEFAAAABWFzc2V0BQAAAANuaWwFAAAAA3JlcwkAAAIAAAABAgAAAAdmYWlsISEhCQAAAgAAAAECAAAADW5vdCBhIEJvb2xlYW4JAAACAAAAAQIAAAAkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuAAAAAOyIF7Y="
	src1, err := base64.StdEncoding.DecodeString(code1)
	require.NoError(t, err)

	/* On dApp2 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let asset = base58'HXa5senn3qfi4sKPPLADnTaYnT2foBrhXnMymqFgpVp8'

	@Callable(i)
	func call() = ([Reissue(asset, 1, true), ScriptTransfer(i.caller, 1, asset)], true)
	*/
	code2 := "AAIFAAAAAAAAAAQIAhIAAAAAAQAAAAAFYXNzZXQBAAAAIPWP0sTs01eGo1z8E2WvNMcZdbq+uV2nwHDbjZdrkBIlAAAAAQAAAAFpAQAAAARjYWxsAAAAAAkABRQAAAACCQAETAAAAAIJAQAAAAdSZWlzc3VlAAAAAwUAAAAFYXNzZXQAAAAAAAAAAAEGCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAAAAAAAQUAAAAFYXNzZXQFAAAAA25pbAYAAAAAUOFniw=="
	src2, err := base64.StdEncoding.DecodeString(code2)
	require.NoError(t, err)

	recipient := proto.NewRecipientFromAddress(dApp1)
	arguments := proto.Arguments{}
	call := proto.FunctionCall{
		Default:   false,
		Name:      "call",
		Arguments: arguments,
	}
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              makeRandomTxID(t),
		Proofs:          proto.NewProofs(),
		ChainID:         proto.TestNetScheme,
		SenderPK:        senderPK,
		ScriptRecipient: recipient,
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1624967106278,
	}
	testInv, err := invocationToObject(5, proto.TestNetScheme, tx)
	require.NoError(t, err)
	testDAppAddress := dApp1
	env := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		thisFunc: func() rideType {
			return rideAddress(testDAppAddress)
		},
		transactionFunc: func() rideObject {
			obj, err := transactionToObject(proto.TestNetScheme, tx)
			require.NoError(t, err)
			return obj
		},
		invocationFunc: func() rideObject {
			return testInv
		},
		checkMessageLengthFunc: v3check,
		setInvocationFunc: func(inv rideObject) {
			testInv = inv
		},
		validateInternalPaymentsFunc: func() bool {
			return true
		},
		maxDataEntriesSizeFunc: func() int {
			return proto.MaxDataEntriesScriptActionsSizeInBytesV2
		},
		blockV5ActivatedFunc: func() bool {
			return true
		},
		rideV6ActivatedFunc: noRideV6,
	}

	mockState := &MockSmartState{
		GetByteTreeFunc: func(recipient proto.Recipient) (proto.Script, error) {
			switch recipient.String() {
			case dApp1.String():
				return src1, nil
			case dApp2.String():
				return src2, nil
			default:
				return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
			}
		},
		NewestScriptPKByAddrFunc: func(addr proto.WavesAddress) (crypto.PublicKey, error) {
			switch addr {
			case sender:
				return senderPK, nil
			case dApp1:
				return dApp1PK, nil
			case dApp2:
				return dApp2PK, nil
			default:
				return crypto.PublicKey{}, errors.Errorf("unexpected address %s", addr.String())
			}
		},
		NewestRecipientToAddressFunc: func(recipient proto.Recipient) (*proto.WavesAddress, error) {
			switch recipient.String() {
			case dApp1.String():
				return &dApp1, nil
			case dApp2.String():
				return &dApp2, nil
			default:
				return nil, errors.Errorf("unexpected recipient '%s'", recipient.String())
			}
		},
		NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
			return 0, nil
		},
		NewestAssetBalanceFunc: func(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
			if assetID != asset {
				return 0, errors.Errorf("unxepected asset '%s'", assetID.String())
			}
			switch {
			case account.Address.Equal(dApp1):
				return 0, nil
			case account.Address.Equal(dApp2):
				return 0, nil
			default:
				return 0, errors.Errorf("unxepected account '%s'", account.String())
			}
		},
		NewestAssetIsSponsoredFunc: func(assetID crypto.Digest) (bool, error) {
			switch assetID {
			case asset:
				return false, nil
			default:
				return false, errors.Errorf("unexpected asset '%s'", assetID.String())
			}
		},
		NewestAssetInfoFunc: func(assetID crypto.Digest) (*proto.AssetInfo, error) {
			if assetID != asset {
				return nil, errors.Errorf("unexpected asset '%s'", assetID.String())
			}
			return &proto.AssetInfo{
				ID:              assetID,
				Quantity:        10,
				Decimals:        0,
				Issuer:          dApp2,
				IssuerPublicKey: dApp2PK,
				Reissuable:      true,
			}, nil
		},
	}
	testState := initWrappedState(mockState, env)
	env.stateFunc = func() types.SmartState {
		return testState
	}
	env.setNewDAppAddressFunc = func(address proto.WavesAddress) {
		testDAppAddress = address
		testState.cle = rideAddress(address) // We have to update wrapped state's `cle`
	}

	tree, err := Parse(src1)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	res, err := CallFunction(env, tree, "call", arguments)
	require.NoError(t, err)
	r, ok := res.(DAppResult)
	require.True(t, ok)

	optionalAsset := proto.NewOptionalAssetFromDigest(asset)
	sr, ap, err := proto.NewScriptResult(r.actions, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(ap))
	expectedResult := &proto.ScriptResult{
		DataEntries:  make([]*proto.DataEntryScriptAction, 0),
		Transfers:    []*proto.TransferScriptAction{{Sender: &dApp2PK, Recipient: proto.NewRecipientFromAddress(dApp1), Amount: 1, Asset: *optionalAsset}, {Recipient: proto.NewRecipientFromAddress(sender), Amount: 1, Asset: *optionalAsset}},
		Issues:       make([]*proto.IssueScriptAction, 0),
		Reissues:     []*proto.ReissueScriptAction{{Sender: &dApp2PK, AssetID: asset, Quantity: 1, Reissuable: true}},
		Burns:        make([]*proto.BurnScriptAction, 0),
		Sponsorships: make([]*proto.SponsorshipScriptAction, 0),
		Leases:       make([]*proto.LeaseScriptAction, 0),
		LeaseCancels: make([]*proto.LeaseCancelScriptAction, 0),
		ErrorMsg:     proto.ScriptErrorMessage{},
	}
	assert.Equal(t, expectedResult, sr)
}
