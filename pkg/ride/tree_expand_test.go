package ride

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	t.Run("expand 2 functions", func(t *testing.T) {
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

func TestExpand(t *testing.T) {
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
		},
		RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
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
	require.Equal(t, false, rs.Result())
}
