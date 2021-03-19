package ride

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"

	//"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func lines(ss ...string) string {
	var s strings.Builder
	for _, v := range ss {
		s.WriteString(v)
		s.WriteString(" ")
	}
	return strings.TrimSpace(s.String())
}

func TestTreeExpand(t *testing.T) {
	source := `AAIDAAAAAAAAAAgIARIECgIIAgAAAAEBAAAAAmYyAAAAAAkBAAAABXZhbHVlAAAAAQkABBoAAAACBQAAAAR0aGlzAgAAAAF4AAAAAQAAAAFpAQAAAAJmMQAAAAIAAAAJc2Vzc2lvbklkAAAAB3JzYVNpZ24EAAAAAXgJAQAAAAJmMgAAAAAJAQAAAAhXcml0ZVNldAAAAAEFAAAAA25pbAAAAADvU/gM`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)

	tree2, _ := Expand(tree)

	require.Equal(t,
		`@i\nfunc f1(sessionId,rsaSign) { let x = { value(getInteger(this,"x")) }; WriteSet(nil) }`,
		DecompileTree(tree2),
	)
}

func TestTreeExpandWithArguments(t *testing.T) {
	source := `AAIDAAAAAAAAAAgIARIECgIIAgAAAAIAAAAAAXoAAAAAAAAAAAUBAAAAAmYyAAAAAQAAAAF2CQEAAAAFdmFsdWUAAAABCQAEGgAAAAIFAAAABHRoaXMFAAAAAXYAAAABAAAAAWkBAAAAAmYxAAAAAgAAAAlzZXNzaW9uSWQAAAAHcnNhU2lnbgQAAAABeAkBAAAAAmYyAAAAAQIAAAABZQkBAAAACFdyaXRlU2V0AAAAAQUAAAADbmlsAAAAAN+I8mI=`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)

	tree2, _ := Expand(tree)

	require.Equal(t,
		lines(
			`let z = { 5 };`,
			`@i\nfunc f1(sessionId,rsaSign) { let x = { let v = { "e" }; value(getInteger(this,v)) }; WriteSet(nil) }`,
		),
		DecompileTree(tree2),
	)
}

/**
{-# STDLIB_VERSION 3 #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# CONTENT_TYPE DAPP #-}
func f2() = {
    5
}

@Callable(i)
func f1 () = {
    WriteSet([DataEntry("key", f2())])
}

*/
func TestTreeExpandAsArgument(t *testing.T) {
	source := `AAIDAAAAAAAAAAQIARIAAAAAAQEAAAACZjIAAAAAAAAAAAAAAAAFAAAAAQAAAAFpAQAAAAJmMQAAAAAJAQAAAAhXcml0ZVNldAAAAAEJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAAA2tleQkBAAAAAmYyAAAAAAUAAAADbmlsAAAAABmdzZY=`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)

	tree2, _ := Expand(tree)

	require.Equal(t,
		`@i\nfunc f1() { WriteSet(1100(DataEntry("key",5),nil)) }`,
		DecompileTree(tree2),
	)
}

/**
{-# STDLIB_VERSION 3 #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# CONTENT_TYPE DAPP #-}
func call(v: Int) = {
    func f2() = {
        10
    }
    f2()
}

func f2() = {
    5
}

@Callable(i)
func callback () = {
    let x = call(0)
    WriteSet([DataEntry("key", f2())])
}
*/
func TestTreeExpandWithNamesIntersection(t *testing.T) {
	source := `AAIDAAAAAAAAAAQIARIAAAAAAgEAAAAEY2FsbAAAAAEAAAABdgoBAAAAAmYyAAAAAAAAAAAAAAAACgkBAAAAAmYyAAAAAAEAAAACZjIAAAAAAAAAAAAAAAAFAAAAAQAAAAFpAQAAAAhjYWxsYmFjawAAAAAEAAAAAXgJAQAAAARjYWxsAAAAAQAAAAAAAAAAAAkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAADa2V5CQEAAAACZjIAAAAABQAAAANuaWwAAAAA/C/YQQ==`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)

	tree2, _ := Expand(tree)

	require.Equal(t,
		`@i\nfunc callback() { let x = { let v = { 0 }; 10 }; WriteSet(1100(DataEntry("key",5),nil)) }`,
		DecompileTree(tree2),
	)
}

func TestTreeExpand11(t *testing.T) {
	t.Run("expand with variable and func name collision", func(t *testing.T) {
		/**
		  {-# STDLIB_VERSION 3 #-}
		  {-# SCRIPT_TYPE ACCOUNT #-}
		  {-# CONTENT_TYPE EXPRESSION #-}
		  func inc(v: Int) = v + 1
		  func call(inc: Int) = {
		      inc(inc)
		  }
		  call(2) == 3
		*/
		source := `AwoBAAAAA2luYwAAAAEAAAABdgkAAGQAAAACBQAAAAF2AAAAAAAAAAABCgEAAAAEY2FsbAAAAAEAAAADaW5jCQEAAAADaW5jAAAAAQUAAAADaW5jCQAAAAAAAAIJAQAAAARjYWxsAAAAAQAAAAAAAAAAAgAAAAAAAAAAAxgTXMY=`
		src, err := base64.StdEncoding.DecodeString(source)
		require.NoError(t, err)

		tree, err := Parse(src)
		require.NoError(t, err)

		tree2, _ := Expand(tree)

		require.Equal(t,
			`(let inc = { 2 }; let v = { inc }; (v + 1) == 3)`,
			DecompileTree(tree2),
		)
		rs, err := CallTreeVerifier(nil, tree2)
		require.NoError(t, err)
		require.Equal(t, true, rs.Result())
	})
	t.Run("expand 2 funcs", func(t *testing.T) {
		/**
		{-# STDLIB_VERSION 3 #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		func inc() = true
		func call() = {
			inc()
		}
		call()
		*/
		source := `AwoBAAAAA2luYwAAAAAGCgEAAAAEY2FsbAAAAAAJAQAAAANpbmMAAAAACQEAAAAEY2FsbAAAAAByJ2Mb`
		src, err := base64.StdEncoding.DecodeString(source)
		require.NoError(t, err)

		tree, err := Parse(src)
		require.NoError(t, err)

		tree2, _ := Expand(tree)

		require.Equal(t,
			`true`,
			DecompileTree(tree2),
		)
		rs, err := CallTreeVerifier(nil, tree2)
		require.NoError(t, err)
		require.Equal(t, true, rs.Result())
	})
}

/**
{-# STDLIB_VERSION 3 #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# CONTENT_TYPE EXPRESSION #-}
func f2() = {
    5
}
f2() == f2()
*/
func TestTreeExpandExpression(t *testing.T) {
	source := `AwoBAAAAAmYyAAAAAAAAAAAAAAAABQkAAAAAAAACCQEAAAACZjIAAAAACQEAAAACZjIAAAAAIckc5A==`
	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)

	tree2, _ := Expand(tree)

	require.Equal(t,
		`(5 == 5)`,
		DecompileTree(tree2),
	)
}

func TestExpandScope(t *testing.T) {
	m := newExpandScope().
		add("inc", &FunctionDeclarationNode{Name: "inc"}).
		add("call", &FunctionDeclarationNode{Name: "call"})

	require.NotNil(t, m.get1("call"))
}

func TestExpandSmthWrote(t *testing.T) {
	// file_5.ride
	//source := `AAIDAAAAAAAAAAQIARIAAAAAIgEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABAAAAA2tleQQAAAAHJG1hdGNoMAkABBoAAAACBQAAAAR0aGlzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAA0ludAQAAAABYQUAAAAHJG1hdGNoMAUAAAABYQAAAAAAAAAAAAEAAAAOZ2V0U3RyaW5nQnlLZXkAAAABAAAAA2tleQQAAAAHJG1hdGNoMAkABB0AAAACBQAAAAR0aGlzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABlN0cmluZwQAAAABYQUAAAAHJG1hdGNoMAUAAAABYQIAAAAAAQAAAAxnZXRCb29sQnlLZXkAAAABAAAAA2tleQQAAAAHJG1hdGNoMAkABBsAAAACBQAAAAR0aGlzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAB0Jvb2xlYW4EAAAAAWEFAAAAByRtYXRjaDAFAAAAAWEHAQAAABhnZXROdW1iZXJCeUFkZHJlc3NBbmRLZXkAAAACAAAAB2FkZHJlc3MAAAADa2V5BAAAAAckbWF0Y2gwCQAEGgAAAAIFAAAAB2FkZHJlc3MFAAAAA2tleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAFhBQAAAAckbWF0Y2gwBQAAAAFhAAAAAAAAAAAAAQAAABZjb252ZXJ0SnNvbkFycmF5VG9MaXN0AAAAAQAAAAlqc29uQXJyYXkJAAS1AAAAAgUAAAAJanNvbkFycmF5AgAAAAEsAAAAAAVCTE9DSwIAAAAFYmxvY2sAAAAACUNPTkZJUk1UWAIAAAAKY29uZmlybV90eAAAAAAKT3JhY2xlc0tleQIAAAAHb3JhY2xlcwAAAAAJQWRtaW5zS2V5AgAAAAZhZG1pbnMAAAAAD1ZvdGVJbnRlcnZhbEtleQIAAAANdm90ZV9pbnRlcnZhbAAAAAAUQ29lZmZpY2llbnRPcmFjbGVLZXkCAAAAEmNvZWZmaWNpZW50X29yYWNsZQAAAAATQ29lZmZpY2llbnRBZG1pbktleQIAAAARY29lZmZpY2llbnRfYWRtaW4AAAAADlByaWNlT2Zmc2V0S2V5AgAAAAxwcmljZV9vZmZzZXQAAAAACFByaWNlS2V5AgAAAAVwcmljZQAAAAAMSXNCbG9ja2VkS2V5AgAAAAppc19ibG9ja2VkAAAAABJCbG9ja0V4cGlyZVZvdGVLZXkCAAAAEXZvdGVfZXhwaXJlX2Jsb2NrAAAAAA1QcmljZUluZGV4S2V5AgAAAAtwcmljZV9pbmRleAAAAAAQTGFzdENvbmZpcm1UeEtleQIAAAAPbGFzdF9jb25maXJtX3R4AQAAAA9nZXRBZG1pblZvdGVLZXkAAAABAAAABW93bmVyCQABLAAAAAICAAAAC2FkbWluX3ZvdGVfBQAAAAVvd25lcgEAAAAYZ2V0QWRtaW5Wb3RlQXJndW1lbnRzS2V5AAAAAQAAAAVvd25lcgkAASwAAAACCQABLAAAAAICAAAADHZvdGVfYWN0aW9uXwUAAAAFb3duZXICAAAACl9hcmd1bWVudHMBAAAAFWdldEJsYWNrU3dhcm1QcmljZUtleQAAAAEAAAAFYmxvY2sJAAEsAAAAAgkAASwAAAACAgAAABFibGFja19zd2FybV9wcmljZQIAAAABXwkAAaQAAAABBQAAAAVibG9jawEAAAASZ2V0UHJpY2VIaXN0b3J5S2V5AAAAAQAAAAVibG9jawkAASwAAAACCQABLAAAAAIFAAAACFByaWNlS2V5AgAAAAFfCQABpAAAAAEFAAAABWJsb2NrAQAAABhnZXRIZWlnaHRQcmljZUJ5SW5kZXhLZXkAAAABAAAABWluZGV4CQABLAAAAAIJAAEsAAAAAgUAAAANUHJpY2VJbmRleEtleQIAAAABXwkAAaQAAAABBQAAAAVpbmRleAEAAAAYZ2V0T3JhY2xlUHJvdmlkZVByaWNlS2V5AAAAAQAAAAZoZWlnaHQJAAEsAAAAAgIAAAAGcHJpY2VfCQABpAAAAAEFAAAABmhlaWdodAAAAAAFcHJpY2UJAQAAAA5nZXROdW1iZXJCeUtleQAAAAEFAAAACFByaWNlS2V5AAAAAApwcmljZUluZGV4CQEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABBQAAAA1QcmljZUluZGV4S2V5AAAAAAlpc0Jsb2NrZWQJAQAAAAxnZXRCb29sQnlLZXkAAAABBQAAAAxJc0Jsb2NrZWRLZXkAAAAAFGJmdENvZWZmaWNpZW50T3JhY2xlCQEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABBQAAABRDb2VmZmljaWVudE9yYWNsZUtleQAAAAAScGVyY2VudFByaWNlT2Zmc2V0CQEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABBQAAAA5QcmljZU9mZnNldEtleQAAAAAHb3JhY2xlcwkBAAAADmdldFN0cmluZ0J5S2V5AAAAAQUAAAAKT3JhY2xlc0tleQAAAAALb3JhY2xlc0xpc3QJAQAAABZjb252ZXJ0SnNvbkFycmF5VG9MaXN0AAAAAQUAAAAHb3JhY2xlcwAAAAANbGFzdENvbmZpcm1UeAkBAAAADmdldFN0cmluZ0J5S2V5AAAAAQUAAAAQTGFzdENvbmZpcm1UeEtleQEAAAAWZ2V0T3JhY2xlUHJvdmlkZUhlaWdodAAAAAIAAAAFb3duZXIAAAAGaGVpZ2h0CQEAAAAYZ2V0TnVtYmVyQnlBZGRyZXNzQW5kS2V5AAAAAgkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABBQAAAAVvd25lcgkBAAAAGGdldE9yYWNsZVByb3ZpZGVQcmljZUtleQAAAAEFAAAABmhlaWdodAEAAAAPZ2V0UHJpY2VIaXN0b3J5AAAAAQAAAAZoZWlnaHQJAQAAAA5nZXROdW1iZXJCeUtleQAAAAEJAQAAABJnZXRQcmljZUhpc3RvcnlLZXkAAAABBQAAAAZoZWlnaHQAAAABAAAAAWkBAAAAFGZpbmFsaXplQ3VycmVudFByaWNlAAAAAAQAAAAGcHJpY2VzCQAETAAAAAIJAQAAABZnZXRPcmFjbGVQcm92aWRlSGVpZ2h0AAAAAgkAAZEAAAACBQAAAAtvcmFjbGVzTGlzdAAAAAAAAAAAAAUAAAAGaGVpZ2h0CQAETAAAAAIJAQAAABZnZXRPcmFjbGVQcm92aWRlSGVpZ2h0AAAAAgkAAZEAAAACBQAAAAtvcmFjbGVzTGlzdAAAAAAAAAAAAQUAAAAGaGVpZ2h0CQAETAAAAAIJAQAAABZnZXRPcmFjbGVQcm92aWRlSGVpZ2h0AAAAAgkAAZEAAAACBQAAAAtvcmFjbGVzTGlzdAAAAAAAAAAAAgUAAAAGaGVpZ2h0CQAETAAAAAIJAQAAABZnZXRPcmFjbGVQcm92aWRlSGVpZ2h0AAAAAgkAAZEAAAAC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`
	source := `AAIDAAAAAAAAAAQIARIAAAAAAwEAAAAOZ2V0TnVtYmVyQnlLZXkAAAABAAAAA2tleQMJAAAAAAAAAgUAAAADa2V5AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAEBAAAAEmdldFByaWNlSGlzdG9yeUtleQAAAAEAAAAFYmxvY2sJAAEsAAAAAgIAAAAGcHJpY2VfCQABpAAAAAEFAAAABWJsb2NrAQAAAA9nZXRQcmljZUhpc3RvcnkAAAABAAAABmhlaWdodAkBAAAADmdldE51bWJlckJ5S2V5AAAAAQkBAAAAEmdldFByaWNlSGlzdG9yeUtleQAAAAEFAAAABmhlaWdodAAAAAEAAAABaQEAAAAUZmluYWxpemVDdXJyZW50UHJpY2UAAAAAAwkBAAAAAiE9AAAAAgkBAAAAD2dldFByaWNlSGlzdG9yeQAAAAEFAAAABmhlaWdodAAAAAAAAAAAAAkAAAIAAAABAgAAAA93YWl0IG5leHQgYmxvY2sJAQAAAAhXcml0ZVNldAAAAAEFAAAAA25pbAAAAACFlzmA`

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	tree = MustExpand(tree)
	t.Log(DecompileTree(tree))
	require.NoError(t, err)
	require.NotNil(t, tree)

	script, err := CompileDapp("", tree)
	require.NoError(t, err)
	require.NotNil(t, script)
	this := []byte{1, 83, 0, 150, 158, 207, 181, 8, 55, 66, 81, 31, 197, 85, 116, 80, 81, 99, 170, 84, 137, 245, 151, 194, 97, 213}

	state := &MockSmartState{
		//NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
		//	return byte_helpers.TransferWithProofs.Transaction, nil
		//}
		// 1050
		RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			// 2
			if bytes.Equal([]byte{1, 83, 0, 150, 158, 207, 181, 8, 55, 66, 81, 31, 197, 85, 116, 80, 81, 99, 170, 84, 137, 245, 151, 194, 97, 213}, account.Address.Bytes()) && key == "coefficient_oracle" {
				return &proto.IntegerDataEntry{Key: key, Value: 3}, nil
			}
			// 11
			if bytes.Equal([]byte{1, 83, 116, 45, 101, 110, 53, 200, 52, 21, 10, 84, 172, 243, 171, 35, 86, 210, 136, 52, 119, 25, 63, 230, 32, 147}, account.Address.Bytes()) && key == "price_209553" {
				return &proto.IntegerDataEntry{Key: key, Value: 60}, nil
			}
			// 17
			if bytes.Equal([]byte{1, 83, 15, 138, 8, 31, 66, 12, 76, 206, 150, 15, 215, 66, 227, 143, 47, 204, 196, 97, 159, 62, 62, 71, 220, 58}, account.Address.Bytes()) && key == "price_209553" {
				return &proto.IntegerDataEntry{Key: key, Value: 60}, nil
			}
			// 23
			if bytes.Equal([]byte{1, 83, 59, 25, 51, 179, 38, 169, 228, 134, 63, 30, 65, 161, 51, 193, 50, 252, 107, 192, 198, 211, 1, 181, 85, 155}, account.Address.Bytes()) && key == "price_209553" {
				return &proto.IntegerDataEntry{Key: key, Value: 60}, nil
			}
			// 29
			if bytes.Equal([]byte{1, 83, 136, 55, 96, 43, 245, 23, 100, 121, 143, 9, 41, 146, 104, 231, 155, 80, 89, 107, 191, 124, 84, 104, 99, 235}, account.Address.Bytes()) && key == "price_209553" {
				return nil, errors.New("not found")
			}

			panic(fmt.Sprintf("RetrieveNewestIntegerEntryFunc %+v %s", account.Address.Bytes(), key))
			//if !bytes.Equal(account.Address.Bytes(), this) {
			//	panic("unknown account " + account.String())
			//}
			//
			//switch key {
			//case "coefficient_oracle":
			//	return &proto.IntegerDataEntry{Key: key, Value: 3}, nil
			//case "price_209553":
			//	return &proto.IntegerDataEntry{Key: key, Value: 60}, nil
			//default:
			//	panic("unknown key " + key)
			//}
		},
		RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
			//case "is_blocked":
			return nil, errors.New(key + " not found")
		},
		// 1053
		RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
			if !bytes.Equal(account.Address.Bytes(), this) {
				panic("not equal bytes")
			}
			switch key {
			// 4
			case "oracles":
				return &proto.StringDataEntry{Key: key, Value: "3MbAmZFN3uQ1j2SMj28K32esXKSre2uVVf8,3MRzeHJTxhcAw3FhPbwPSR2ZxA7M8hA5AzV,3MVxxrC79QE2tZufp3pdoWWpnNPpZw3Vw2A,3Mczj2UD9swFgFCyqpfPAacJpn2UTu43vVY,3MYzuVPkN2gaLa5RDUesuUQEq8wWh7Y71GR"}, nil
			default:
				panic("unknown key " + key)
			}
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
		checkMessageLengthFunc: func(in1 int) bool {
			return true
		},
		thisFunc: func() rideType {
			return rideBytes(this)
		},
		invocationFunc: func() rideObject {
			return nil
		},
		heightFunc: func() rideInt {
			return rideInt(209553)
		},
	}

	rsT, err := CallTreeFunction("", env, tree, "finalizeCurrentPrice", nil)
	require.NoError(t, err)
	_ = rsT

	rs, err := script.Invoke(env, "finalizeCurrentPrice", nil)
	require.NoError(t, err)
	//require.Equal(t, 2, len(rs.Calls()))
	require.Equal(t, true, rs.Result())
}

func TestExpandSmthWrote3(t *testing.T) {
	source := `AwoBAAAAC0lzUGxheWVyV2luAAAAAgAAAAxwbGF5ZXJDaG9pY2UAAAAHcmFuZFN0cgQAAAABcwkAATEAAAABBQAAAAxwbGF5ZXJDaG9pY2UDAwMDAwkAAGcAAAACBQAAAAFzAAAAAAAAAAABCQAAAAAAAAIJAAEvAAAAAgkAATAAAAACBQAAAAxwbGF5ZXJDaG9pY2UAAAAAAAAAAAAAAAAAAAAAAAEFAAAAB3JhbmRTdHIHBgMJAABnAAAAAgUAAAABcwAAAAAAAAAAAgkAAAAAAAACCQABLwAAAAIJAAEwAAAAAgUAAAAMcGxheWVyQ2hvaWNlAAAAAAAAAAABAAAAAAAAAAABBQAAAAdyYW5kU3RyBwYDCQAAZwAAAAIFAAAAAXMAAAAAAAAAAAMJAAAAAAAAAgkAAS8AAAACCQABMAAAAAIFAAAADHBsYXllckNob2ljZQAAAAAAAAAAAgAAAAAAAAAAAQUAAAAHcmFuZFN0cgcGAwkAAGcAAAACBQAAAAFzAAAAAAAAAAAECQAAAAAAAAIJAAEvAAAAAgkAATAAAAACBQAAAAxwbGF5ZXJDaG9pY2UAAAAAAAAAAAMAAAAAAAAAAAEFAAAAB3JhbmRTdHIHBgMJAAAAAAAAAgUAAAABcwAAAAAAAAAABQkAAAAAAAACCQABLwAAAAIJAAEwAAAAAgUAAAAMcGxheWVyQ2hvaWNlAAAAAAAAAAAEAAAAAAAAAAABBQAAAAdyYW5kU3RyBwkBAAAAC0lzUGxheWVyV2luAAAAAgIAAAACYWECAAAAAmJinpP4rw==`

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	tree = MustExpand(tree)
	t.Log(DecompileTree(tree))
	//require.NoError(t, err)
	//require.NotNil(t, tree)
	//
	script, err := CompileTree("", tree)
	require.NoError(t, err)

	//rsT, err := CallTreeVerifier(nil, tree)
	//require.NoError(t, err)
	//_ = rsT
	//
	//rs, err := script.Invoke(defaultEnv, "addBuyBondOrder", []rideType{rideInt(1398601), rideString("")})
	rs, err := script.Verify(defaultEnv)
	require.NoError(t, err)
	////require.Equal(t, 2, len(rs.Calls()))
	require.Equal(t, true, rs.Result())
}

/*
{-# STDLIB_VERSION 3 #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# CONTENT_TYPE EXPRESSION #-}
func xx(height: Int) = {
    height
}

xx(height) == 1
*/
func TestWithHeight(t *testing.T) {
	source := `AwoBAAAAAnh4AAAAAQAAAAZoZWlnaHQFAAAABmhlaWdodAkAAAAAAAACCQEAAAACeHgAAAABBQAAAAZoZWlnaHQAAAAAAAAAAAHS0qks`

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	tree = MustExpand(tree)
	t.Log(DecompileTree(tree))
	//require.NoError(t, err)
	//require.NotNil(t, tree)
	//
	script, err := CompileTree("", tree)
	require.NoError(t, err)

	//rsT, err := CallTreeVerifier(nil, tree)
	//require.NoError(t, err)
	//_ = rsT
	//
	//rs, err := script.Invoke(defaultEnv, "addBuyBondOrder", []rideType{rideInt(1398601), rideString("")})
	rs, err := script.Verify(defaultEnv)
	require.NoError(t, err)
	////require.Equal(t, 2, len(rs.Calls()))
	require.Equal(t, true, rs.Result())
}

/*
{-# STDLIB_VERSION 3 #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# CONTENT_TYPE EXPRESSION #-}


*/
func TestWithHeight2(t *testing.T) {
	source := `AAIEAAAAAAAAAAgIAhIECgIIAQAAAB8AAAAABWFkbWluAQAAACCnVErqp8ibQ55WtRdh6dN0wQFoQ9ny+0uTALD786XdOAAAAAAHYXNzZXRJZAkAAlkAAAABAgAAACwzVGFmM1RBSldEeUJzQXN5ZGRiVTFwbjVkV2VjQ3doajRra0UxakZKb0RDawAAAAAETk9ORQIAAAALbm8gZXhpc3RpbmcAAAAABExJU1QJAQAAABFAZXh0ck5hdGl2ZSgxMDYyKQAAAAECAAAAIzNNa3RKZ1YyZVRtY0NxdHlRYWVxaWlIa1ExZVkzRUg1VGRiAQAAAAtmZXRjaFN0cmluZwAAAAIAAAAFYWxpYXMAAAADa2V5BAAAAAckbWF0Y2gwCQAEHQAAAAIFAAAABWFsaWFzBQAAAANrZXkDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABlN0cmluZwQAAAABYQUAAAAHJG1hdGNoMAUAAAABYQUAAAAETk9ORQEAAAAMZmV0Y2hJbnRlZ2VyAAAAAgAAAAVhbGlhcwAAAANrZXkEAAAAByRtYXRjaDAJAAQaAAAAAgUAAAAFYWxpYXMFAAAAA2tleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAADSW50BAAAAAFhBQAAAAckbWF0Y2gwBQAAAAFhAAAAAAAAAAAAAQAAAAdnZXREYXBwAAAAAAkBAAAAC2ZldGNoU3RyaW5nAAAAAgUAAAAEdGhpcwIAAAALZGFwcEFkZHJlc3MBAAAACWdldE1hc3RlcgAAAAAJAQAAAAtmZXRjaFN0cmluZwAAAAIFAAAABHRoaXMCAAAADW1hc3RlckFkZHJlc3MBAAAAEGdldEFjY291bnRTdGF0dXMAAAABAAAAB2FkZHJlc3MJAQAAABFAZXh0ck5hdGl2ZSgxMDUxKQAAAAIFAAAABExJU1QJAAEsAAAAAgUAAAAHYWRkcmVzcwIAAAAHX2FjdGl2ZQEAAAAWZ2V0TGltaXRUb3RhbEFtb3VudEtleQAAAAEAAAAFbGltaXQJAAEsAAAAAgIAAAATbGltaXRfdG90YWxfYW1vdW50XwUAAAAFbGltaXQBAAAAGGdldExpbWl0VG90YWxBbW91bnRWYWx1ZQAAAAEAAAAFbGltaXQJAQAAAAxmZXRjaEludGVnZXIAAAACBQAAAAR0aGlzCQEAAAAWZ2V0TGltaXRUb3RhbEFtb3VudEtleQAAAAEFAAAABWxpbWl0AQAAAA5nZXRUb3RhbEFtb3VudAAAAAAJAQAAAAxmZXRjaEludGVnZXIAAAACBQAAAAR0aGlzAgAAAAx0b3RhbF9hbW91bnQBAAAACmdldExpc3RLZXkAAAACAAAAB2FkZHJlc3MAAAAJbGltaXREYXRlBAAAAApzZWVkUGhyYXNlCQABLAAAAAIFAAAAB2FkZHJlc3MFAAAACWxpbWl0RGF0ZQkAASwAAAACAgAAAAdsaXN0ZWRfCQACWAAAAAEJAAH3AAAAAQkAAZsAAAABBQAAAApzZWVkUGhyYXNlAQAAABBnZXRMaXN0QW1vdW50S2V5AAAAAQAAAAdsaXN0S2V5CQABLAAAAAIFAAAAB2xpc3RLZXkCAAAAB19hbW91bnQBAAAAD2dldExpc3RMaW1pdEtleQAAAAEAAAAHbGlzdEtleQkAASwAAAACBQAAAAdsaXN0S2V5AgAAAAZfbGltaXQBAAAAD2dldExpc3RPd25lcktleQAAAAEAAAAHbGlzdEtleQkAASwAAAACBQAAAAdsaXN0S2V5AgAAAAZfb3duZXIBAAAAE2dldExpc3RVbml0UHJpY2VLZXkAAAABAAAAB2xpc3RLZXkJAAEsAAAAAgUAAAAHbGlzdEtleQIAAAAKX3VuaXRQcmljZQEAAAAVZ2V0TGlzdERlc2NyaXB0aW9uS2V5AAAAAQAAAAdsaXN0S2V5CQABLAAAAAIFAAAAB2xpc3RLZXkCAAAADF9kZXNjcmlwdGlvbgEAAAASZ2V0TGlzdEFtb3VudFZhbHVlAAAAAQAAAAdsaXN0S2V5CQEAAAAMZmV0Y2hJbnRlZ2VyAAAAAgUAAAAEdGhpcwkBAAAAEGdldExpc3RBbW91bnRLZXkAAAABBQAAAAdsaXN0S2V5AQAAABFnZXRMaXN0TGltaXRWYWx1ZQAAAAEAAAAHbGlzdEtleQkBAAAAC2ZldGNoU3RyaW5nAAAAAgUAAAAEdGhpcwkBAAAAD2dldExpc3RMaW1pdEtleQAAAAEFAAAAB2xpc3RLZXkBAAAAEWdldExpc3RPd25lclZhbHVlAAAAAQAAAAdsaXN0S2V5CQEAAAALZmV0Y2hTdHJpbmcAAAACBQAAAAR0aGlzCQEAAAAPZ2V0TGlzdE93bmVyS2V5AAAAAQUAAAAHbGlzdEtleQEAAAAVZ2V0TGlzdFVuaXRQcmljZVZhbHVlAAAAAQAAAAdsaXN0S2V5CQEAAAAMZmV0Y2hJbnRlZ2VyAAAAAgUAAAAEdGhpcwkBAAAAE2dldExpc3RVbml0UHJpY2VLZXkAAAABBQAAAAdsaXN0S2V5AQAAABdnZXRMaXN0RGVzY3JpcHRpb25WYWx1ZQAAAAEAAAAHbGlzdEtleQkBAAAAC2ZldGNoU3RyaW5nAAAAAgUAAAAEdGhpcwkBAAAAFWdldExpc3REZXNjcmlwdGlvbktleQAAAAEFAAAAB2xpc3RLZXkBAAAAC2dldE93bmVkS2V5AAAAAgAAAAVvd25lcgAAAAVsaW1pdAQAAAAKc2VlZFBocmFzZQkAASwAAAACBQAAAAVvd25lcgUAAAAFbGltaXQJAAEsAAAAAgIAAAAGb3duZWRfCQACWAAAAAEJAAH3AAAAAQkAAZsAAAABBQAAAApzZWVkUGhyYXNlAQAAABFnZXRPd25lZEFtb3VudEtleQAAAAEAAAAIb3duZWRLZXkJAAEsAAAAAgUAAAAIb3duZWRLZXkCAAAAB19hbW91bnQBAAAAEGdldE93bmVkTGltaXRLZXkAAAABAAAACG93bmVkS2V5CQABLAAAAAIFAAAACG93bmVkS2V5AgAAAAZfbGltaXQBAAAAEGdldE93bmVkT3duZXJLZXkAAAABAAAACG93bmVkS2V5CQABLAAAAAIFAAAACG93bmVkS2V5AgAAAAZfb3duZXIBAAAAE2dldE93bmVkQW1vdW50VmFsdWUAAAABAAAACG93bmVkS2V5CQEAAAAMZmV0Y2hJbnRlZ2VyAAAAAgUAAAAEdGhpcwkBAAAAEWdldE93bmVkQW1vdW50S2V5AAAAAQUAAAAIb3duZWRLZXkBAAAAEmdldE93bmVkTGltaXRWYWx1ZQAAAAEAAAAIb3duZWRLZXkJAQAAAAtmZXRjaFN0cmluZwAAAAIFAAAABHRoaXMJAQAAABBnZXRPd25lZExpbWl0S2V5AAAAAQUAAAAIb3duZWRLZXkBAAAAC3VwZGF0ZU93bmVkAAAAAwAAAAVvd25lcgAAAAZhbW91bnQAAAAFbGltaXQEAAAACG93bmVkS2V5CQEAAAALZ2V0T3duZWRLZXkAAAACBQAAAAVvd25lcgUAAAAFbGltaXQJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAIJAQAAABFnZXRPd25lZEFtb3VudEtleQAAAAEFAAAACG93bmVkS2V5CQAAZAAAAAIJAQAAABNnZXRPd25lZEFtb3VudFZhbHVlAAAAAQUAAAAIb3duZWRLZXkFAAAABmFtb3VudAkABEwAAAACCQEAAAALU3RyaW5nRW50cnkAAAACCQEAAAAQZ2V0T3duZWRMaW1pdEtleQAAAAEFAAAACG93bmVkS2V5BQAAAAVsaW1pdAkABEwAAAACCQEAAAALU3RyaW5nRW50cnkAAAACCQEAAAAQZ2V0T3duZWRPd25lcktleQAAAAEFAAAACG93bmVkS2V5BQAAAAVvd25lcgUAAAADbmlsAQAAAA10cmFuc2ZlclRva2VuAAAABAAAAAZzZW5kZXIAAAAJcmVjaXBpZW50AAAABmFtb3VudAAAAAVsaW1pdAkABE4AAAACCQEAAAALdXBkYXRlT3duZWQAAAADBQAAAAZzZW5kZXIJAQAAAAEtAAAAAQUAAAAGYW1vdW50BQAAAAVsaW1pdAkBAAAAC3VwZGF0ZU93bmVkAAAAAwUAAAAJcmVjaXBpZW50BQAAAAZhbW91bnQFAAAABWxpbWl0AAAAAQAAAAFpAQAAAAhwdXJjaGFzZQAAAAIAAAADa2V5AAAABmFtb3VudAQAAAAHaW52b2tlcgIAAAAjM01qQk4ya2lSQjZKbW9FVkVDNDJaTk1YOWlieDVpWjlNaWgEAAAACHN1cHBsaWVyCQEAAAAPZ2V0TGlzdE93bmVyS2V5AAAAAQUAAAADa2V5BAAAAAVsaW1pdAkBAAAAEWdldExpc3RMaW1pdFZhbHVlAAAAAQUAAAADa2V5CQAETgAAAAIJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwkBAAAAEUBleHRyTmF0aXZlKDEwNjIpAAAAAQUAAAAHaW52b2tlcgUAAAAGYW1vdW50BQAAAAdhc3NldElkCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACCQEAAAAQZ2V0TGlzdEFtb3VudEtleQAAAAEFAAAAA2tleQkAAGUAAAACCQEAAAASZ2V0TGlzdEFtb3VudFZhbHVlAAAAAQUAAAADa2V5BQAAAAZhbW91bnQFAAAAA25pbAkBAAAADXRyYW5zZmVyVG9rZW4AAAAECQEAAAAHZ2V0RGFwcAAAAAAFAAAAB2ludm9rZXIFAAAABmFtb3VudAUAAAAFbGltaXQAAAAAWzckVA==`

	src, err := base64.StdEncoding.DecodeString(source)
	require.NoError(t, err)

	tree, err := Parse(src)
	tree = MustExpand(tree)
	t.Log(DecompileTree(tree))
	//require.NoError(t, err)
	//require.NotNil(t, tree)
	//
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
		RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {

			panic("key " + key)
		},
	}

	var defaultEnv = &MockRideEnvironment{
		transactionFunc: testTransferObject,
		stateFunc: func() types.SmartState {
			return defaultState
		},
		schemeFunc: func() byte {
			return 'S'
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

	script, err := CompileTree("", tree)
	require.NoError(t, err)

	//rsT, err := CallTreeVerifier(nil, tree)
	//require.NoError(t, err)
	//_ = rsT
	//
	//rs, err := script.Invoke(defaultEnv, "addBuyBondOrder", []rideType{rideInt(1398601), rideString("")})
	rs, err := script.Invoke(defaultEnv, "purchase", []rideType{rideString("listed_J5ALPF3Z2fbtNs3mP1nFRNvG4AJ9LgDdwdnK6bKzF22s"), rideInt(5000)})

	for _, r := range rs.Calls() {
		t.Log(r)
	}
	require.NoError(t, err)
	require.Equal(t, true, rs.Result())
}
