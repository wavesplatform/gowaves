package ride

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

var (
	v5takeString = takeRideString
	noRideV6     = func() bool {
		return false
	}
)

func TestAddressFromString(t *testing.T) {
	te := &mockRideEnvironment{
		schemeFunc: func() byte {
			return 'W'
		},
		rideV6ActivatedFunc: noRideV6,
	}
	ma, err := proto.NewAddressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3")
	require.NoError(t, err)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString(ma.String())}, false, rideAddress(ma)},
		{[]rideType{rideString("3MpV2xvvcWUcv8FLDKJ9ZRrQpEyF8nFwRUM")}, false, rideUnit{}},
		{[]rideType{rideString("fake address")}, false, rideUnit{}},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideInt(12345)}, true, nil},
		{[]rideType{rideString("dsfjsadfl"), rideInt(12345)}, true, nil},
	} {
		r, err := addressFromString(te, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestAddressValueFromString(t *testing.T) {
	te := &mockRideEnvironment{schemeFunc: func() byte {
		return 'W'
	}}
	ma, err := proto.NewAddressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3")
	require.NoError(t, err)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString(ma.String())}, false, rideAddress(ma)},
		{[]rideType{rideString("3MpV2xvvcWUcv8FLDKJ9ZRrQpEyF8nFwRUM")}, true, nil},
		{[]rideType{rideString("fake address")}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideInt(12345)}, true, nil},
		{[]rideType{rideString("dsfjsadfl"), rideInt(12345)}, true, nil},
	} {
		r, err := addressValueFromString(te, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestTransactionByID(t *testing.T) {
	t.SkipNow()
}

func TestTransactionHeightByID(t *testing.T) {
	t.SkipNow()
}

func TestAssetBalanceV3(t *testing.T) {
	te := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				NewestAssetBalanceFunc: func(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
					return 42, nil
				},
				NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
					return 21, nil
				},
			}
		},
	}
	testCases := []struct {
		expectedBalance rideType
		assetID         rideType
		expectErr       bool
	}{
		{expectedBalance: rideInt(21), assetID: rideUnit{}, expectErr: false},
		{expectedBalance: rideInt(42), assetID: make(rideBytes, crypto.DigestSize), expectErr: false},
		{expectedBalance: rideInt(0), assetID: rideBytes(nil), expectErr: false},
		{expectedBalance: rideInt(0), assetID: rideBytes([]byte{}), expectErr: false},
		{expectedBalance: rideInt(0), assetID: make(rideBytes, 7), expectErr: false},
		{expectedBalance: rideInt(0), assetID: make(rideBytes, 33), expectErr: false},
		{expectedBalance: nil, assetID: rideInt(0), expectErr: true},
	}
	for _, tc := range testCases {
		balance, err := assetBalanceV3(te, rideAddress{}, tc.assetID)
		if tc.expectErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, tc.expectedBalance, balance)
	}
}

func TestAssetBalanceV4(t *testing.T) {
	te := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				NewestAssetBalanceFunc: func(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
					return 42, nil
				},
				NewestWavesBalanceFunc: func(account proto.Recipient) (uint64, error) {
					return 21, nil
				},
			}
		},
	}
	testCases := []struct {
		expectedBalance rideType
		assetID         rideType
		expectErr       bool
	}{
		{expectedBalance: rideInt(42), assetID: make(rideBytes, crypto.DigestSize), expectErr: false},
		{expectedBalance: rideInt(0), assetID: make(rideBytes, 7), expectErr: false},
		{expectedBalance: rideInt(0), assetID: make(rideBytes, 33), expectErr: false},
		{expectedBalance: rideInt(0), assetID: rideBytes(nil), expectErr: false},
		{expectedBalance: rideInt(0), assetID: rideBytes([]byte{}), expectErr: false},
		{expectedBalance: nil, assetID: rideInt(0), expectErr: true},
	}
	for _, tc := range testCases {
		balance, err := assetBalanceV4(te, rideAddress{}, tc.assetID)
		if tc.expectErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, tc.expectedBalance, balance)
	}
}

func TestIntFromState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	correctAlias := proto.NewAlias('T', "good")
	incorrectAddress := proto.MustAddressFromString("3N3isZTp6tchjYox99bpxFkqxxySKY6FQsi")
	incorrectAlias := proto.NewAlias('T', "bad")
	correctAddressRecipient := proto.NewRecipientFromAddress(correctAddress)
	correctAliasRecipient := proto.NewRecipientFromAlias(*correctAlias)
	incorrectAddressRecipient := proto.NewRecipientFromAddress(incorrectAddress)
	incorrectAliasRecipient := proto.NewRecipientFromAlias(*incorrectAlias)
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					if (account.Eq(correctAddressRecipient) || account.Eq(correctAliasRecipient)) && key == "key" {
						return &proto.IntegerDataEntry{Key: "key", Value: 100500}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("key")}, false, rideInt(100500)},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("key")}, false, rideInt(100500)},
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("xxx")}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("xxx")}, false, rideUnit{}},
		{[]rideType{recipientToObject(incorrectAddressRecipient), rideString("key")}, false, rideUnit{}},
		{[]rideType{recipientToObject(incorrectAliasRecipient), rideString("key")}, false, rideUnit{}},
		{[]rideType{}, false, rideUnit{}},
		{[]rideType{rideUnit{}}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAddressRecipient)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient)}, false, rideUnit{}},
		{[]rideType{rideString("xxx"), rideInt(12345)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAddressRecipient), rideInt(12345)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient), rideInt(12345)}, false, rideUnit{}},
	} {
		r, err := intFromState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBytesFromState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	correctAlias := proto.NewAlias('T', "good")
	incorrectAddress := proto.MustAddressFromString("3N3isZTp6tchjYox99bpxFkqxxySKY6FQsi")
	incorrectAlias := proto.NewAlias('T', "bad")
	correctAddressRecipient := proto.NewRecipientFromAddress(correctAddress)
	correctAliasRecipient := proto.NewRecipientFromAlias(*correctAlias)
	incorrectAddressRecipient := proto.NewRecipientFromAddress(incorrectAddress)
	incorrectAliasRecipient := proto.NewRecipientFromAlias(*incorrectAlias)
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
					if (account.Eq(correctAddressRecipient) || account.Eq(correctAliasRecipient)) && key == "key" {
						return &proto.BinaryDataEntry{Key: "key", Value: []byte("value")}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("key")}, false, rideBytes("value")},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("key")}, false, rideBytes("value")},
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("xxx")}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("xxx")}, false, rideUnit{}},
		{[]rideType{recipientToObject(incorrectAddressRecipient), rideString("key")}, false, rideUnit{}},
		{[]rideType{recipientToObject(incorrectAliasRecipient), rideString("key")}, false, rideUnit{}},
		{[]rideType{}, false, rideUnit{}},
		{[]rideType{rideUnit{}}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAddressRecipient)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient)}, false, rideUnit{}},
		{[]rideType{rideString("xxx"), rideInt(12345)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAddressRecipient), rideInt(12345)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient), rideInt(12345)}, false, rideUnit{}},
	} {
		r, err := bytesFromState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestStringFromState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	correctAlias := proto.NewAlias('T', "good")
	incorrectAddress := proto.MustAddressFromString("3N3isZTp6tchjYox99bpxFkqxxySKY6FQsi")
	incorrectAlias := proto.NewAlias('T', "bad")
	correctAddressRecipient := proto.NewRecipientFromAddress(correctAddress)
	correctAliasRecipient := proto.NewRecipientFromAlias(*correctAlias)
	incorrectAddressRecipient := proto.NewRecipientFromAddress(incorrectAddress)
	incorrectAliasRecipient := proto.NewRecipientFromAlias(*incorrectAlias)
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					if (account.Eq(correctAddressRecipient) || account.Eq(correctAliasRecipient)) && key == "key" {
						return &proto.StringDataEntry{Key: "key", Value: "value"}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("key")}, false, rideString("value")},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("key")}, false, rideString("value")},
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("xxx")}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("xxx")}, false, rideUnit{}},
		{[]rideType{recipientToObject(incorrectAddressRecipient), rideString("key")}, false, rideUnit{}},
		{[]rideType{recipientToObject(incorrectAliasRecipient), rideString("key")}, false, rideUnit{}},
		{[]rideType{}, false, rideUnit{}},
		{[]rideType{rideUnit{}}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAddressRecipient)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient)}, false, rideUnit{}},
		{[]rideType{rideString("xxx"), rideInt(12345)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAddressRecipient), rideInt(12345)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient), rideInt(12345)}, false, rideUnit{}},
	} {
		r, err := stringFromState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBooleanFromState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	correctAlias := proto.NewAlias('T', "good")
	incorrectAddress := proto.MustAddressFromString("3N3isZTp6tchjYox99bpxFkqxxySKY6FQsi")
	incorrectAlias := proto.NewAlias('T', "bad")
	correctAddressRecipient := proto.NewRecipientFromAddress(correctAddress)
	correctAliasRecipient := proto.NewRecipientFromAlias(*correctAlias)
	incorrectAddressRecipient := proto.NewRecipientFromAddress(incorrectAddress)
	incorrectAliasRecipient := proto.NewRecipientFromAlias(*incorrectAlias)
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
					if (account.Eq(correctAddressRecipient) || account.Eq(correctAliasRecipient)) && key == "key" {
						return &proto.BooleanDataEntry{Key: "key", Value: true}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("key")}, false, rideBoolean(true)},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("key")}, false, rideBoolean(true)},
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("xxx")}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("xxx")}, false, rideUnit{}},
		{[]rideType{recipientToObject(incorrectAddressRecipient), rideString("key")}, false, rideUnit{}},
		{[]rideType{recipientToObject(incorrectAliasRecipient), rideString("key")}, false, rideUnit{}},
		{[]rideType{}, false, rideUnit{}},
		{[]rideType{rideUnit{}}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAddressRecipient)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient)}, false, rideUnit{}},
		{[]rideType{rideString("xxx"), rideInt(12345)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAddressRecipient), rideInt(12345)}, false, rideUnit{}},
		{[]rideType{recipientToObject(correctAliasRecipient), rideInt(12345)}, false, rideUnit{}},
	} {
		r, err := booleanFromState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestIntFromSelfState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					if *account.Address() == correctAddress && key == "key" {
						return &proto.IntegerDataEntry{Key: "key", Value: 100500}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(correctAddress)
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("key")}, false, rideInt(100500)},
		{[]rideType{rideString("xxx")}, false, rideUnit{}},
		{[]rideType{}, false, rideUnit{}},
		{[]rideType{rideUnit{}}, false, rideUnit{}},
		{[]rideType{rideString("xxx"), rideInt(12345)}, false, rideUnit{}},
	} {
		r, err := intFromSelfState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBytesFromSelfState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
					if *account.Address() == correctAddress && key == "key" {
						return &proto.BinaryDataEntry{Key: "key", Value: []byte("value")}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(correctAddress)
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("key")}, false, rideBytes("value")},
		{[]rideType{rideString("xxx")}, false, rideUnit{}},
		{[]rideType{}, false, rideUnit{}},
		{[]rideType{rideUnit{}}, false, rideUnit{}},
		{[]rideType{rideString("xxx"), rideInt(12345)}, false, rideUnit{}},
	} {
		r, err := bytesFromSelfState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestStringFromSelfState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					if *account.Address() == correctAddress && key == "key" {
						return &proto.StringDataEntry{Key: "key", Value: "value"}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(correctAddress)
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("key")}, false, rideString("value")},
		{[]rideType{rideString("xxx")}, false, rideUnit{}},
		{[]rideType{}, false, rideUnit{}},
		{[]rideType{rideUnit{}}, false, rideUnit{}},
		{[]rideType{rideString("xxx"), rideInt(12345)}, false, rideUnit{}},
	} {
		r, err := stringFromSelfState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBooleanFromSelfState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
					if *account.Address() == correctAddress && key == "key" {
						return &proto.BooleanDataEntry{Key: "key", Value: true}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(correctAddress)
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("key")}, false, rideBoolean(true)},
		{[]rideType{rideString("xxx")}, false, rideUnit{}},
		{[]rideType{}, false, rideUnit{}},
		{[]rideType{rideUnit{}}, false, rideUnit{}},
		{[]rideType{rideString("xxx"), rideInt(12345)}, false, rideUnit{}},
	} {
		r, err := booleanFromSelfState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestAddressFromRecipient(t *testing.T) {
	addr, err := proto.NewAddressFromString("3N9WtaPoD1tMrDZRG26wA142Byd35tLhnLU")
	require.NoError(t, err)
	s := &MockSmartState{
		NewestAddrByAliasFunc: func(alias proto.Alias) (proto.WavesAddress, error) {
			if alias.Alias == "correct" {
				return addr, nil
			}
			return proto.WavesAddress{}, errors.New("unexpected test address")
		},
	}
	alias := proto.NewAlias('T', "correct")
	e := &mockRideEnvironment{
		schemeFunc: func() byte {
			return 'T'
		},
		stateFunc: func() types.SmartState {
			return s
		},
		validateInternalPaymentsFunc: func() bool {
			return false
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideAddress(addr)}, false, rideAddress(addr)},
		{[]rideType{rideAlias(*alias)}, false, rideAddress(addr)},
		{[]rideType{recipientToObject(proto.NewRecipientFromAddress(addr))}, false, rideAddress(addr)},
		{[]rideType{recipientToObject(proto.NewRecipientFromAlias(*alias))}, false, rideAddress(addr)},
		{[]rideType{}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideString("xxx"), rideInt(12345)}, true, nil},
	} {
		r, err := addressFromRecipient(e, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestSigVerify(t *testing.T) {
	msg, err := hex.DecodeString("135212a9cf00d0a05220be7323bfa4a5ba7fc5465514007702121a9c92e46bd473062f00841af83cb7bc4b2cd58dc4d5b151244cc8293e795796835ed36822c6e09893ec991b38ada4b21a06e691afa887db4e9d7b1d2afc65ba8d2f5e6926ff53d2d44d55fa095f3fad62545c714f0f3f59e4bfe91af8")
	require.NoError(t, err)
	sig, err := hex.DecodeString("d971ec27c5bfc384804c8d8d6a2de9edc3d957b25e488e954a71ef4c4a87f5fb09cfdf6bd26cffc49d03048e8edb0c918061be158d737c2e11cc7210263efb85")
	require.NoError(t, err)
	bad, err := hex.DecodeString("44164f23a95ed2662c5b1487e8fd688be9032efa23dd2ef29b018d33f65d0043df75f3ac1d44b4bda50e8b07e0b49e2898bec80adbf7604e72ef6565bd2f8189")
	require.NoError(t, err)
	pk, err := hex.DecodeString("ba9e7203ca62efbaa49098ec408bdf8a3dfed5a7fa7c200ece40aade905e535f")
	require.NoError(t, err)
	big := bytes.Repeat([]byte{0xCA, 0xFE, 0xBA, 0xBE, 0xDE, 0xAD, 0xBE, 0xEF}, 19201)
	for _, test := range []struct {
		args  []rideType
		check func(int) bool
		fail  bool
		r     rideType
	}{
		{[]rideType{rideBytes(msg), rideBytes(sig), rideBytes(pk)}, bytesSizeCheckV1V2, false, rideBoolean(true)},
		{[]rideType{rideBytes(msg), rideBytes(bad), rideBytes(pk)}, bytesSizeCheckV1V2, false, rideBoolean(false)},
		{[]rideType{rideBytes(msg), rideBytes(sig), rideBytes(pk[:10])}, bytesSizeCheckV1V2, false, rideBoolean(false)},
		{[]rideType{rideString("MESSAGE"), rideBytes(sig), rideBytes(pk)}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{rideBytes(big), rideBytes(sig), rideBytes(pk)}, bytesSizeCheckV1V2, false, rideBoolean(false)},
		{[]rideType{rideBytes(big), rideBytes(sig), rideBytes(pk)}, bytesSizeCheckV3V6, true, nil},
		{[]rideType{rideBytes(msg), rideString("SIGNATURE"), rideBytes(pk)}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{rideBytes(msg), rideBytes(sig), rideString("PUBLIC KEY")}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{rideUnit{}}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{rideInt(12345)}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{rideString("dsfjsadfl"), rideInt(12345)}, bytesSizeCheckV1V2, true, nil},
	} {
		te := &mockRideEnvironment{
			checkMessageLengthFunc: test.check,
			libVersionFunc: func() (ast.LibraryVersion, error) {
				return ast.LibV3, nil
			},
		}
		r, err := sigVerify(te, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestKeccak256(t *testing.T) {
	data, err := hex.DecodeString("64617461")
	require.NoError(t, err)
	digest1, err := hex.DecodeString("8f54f1c2d0eb5771cd5bf67a6689fcd6eed9444d91a39e5ef32a9b4ae5ca14ff")
	require.NoError(t, err)
	digest2, err := hex.DecodeString("64e604787cbf194841e7b68d7cd28786f6c9a0a3ab9f8b0a0e87cb4387ab0107")
	require.NoError(t, err)
	digest3, err := hex.DecodeString("fe0a57a797d6cb60a92548f2b43bd5e425212f55e0b7adb772ddabd85d21943e")
	require.NoError(t, err)
	big := bytes.Repeat([]byte{0xCA, 0xFE, 0xBA, 0xBE, 0xDE, 0xAD, 0xBE, 0xEF}, 19201)
	for _, test := range []struct {
		args  []rideType
		check func(int) bool
		fail  bool
		r     rideType
	}{
		{[]rideType{rideBytes(data)}, bytesSizeCheckV1V2, false, rideBytes(digest1)},
		{[]rideType{rideString("123")}, bytesSizeCheckV1V2, false, rideBytes(digest2)},
		{[]rideType{rideBytes(big)}, bytesSizeCheckV1V2, false, rideBytes(digest3)},
		{[]rideType{rideBytes(big)}, bytesSizeCheckV3V6, true, nil},
		{[]rideType{rideUnit{}}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{rideInt(12345)}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{rideString("dsfjsadfl"), rideInt(12345)}, bytesSizeCheckV1V2, true, nil},
	} {
		r, err := keccak256(&mockRideEnvironment{checkMessageLengthFunc: test.check}, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBlake2b256(t *testing.T) {
	data, err := hex.DecodeString("64617461")
	require.NoError(t, err)
	digest1, err := hex.DecodeString("a035872d6af8639ede962dfe7536b0c150b590f3234a922fb7064cd11971b58e")
	require.NoError(t, err)
	digest2, err := hex.DecodeString("f5d67bae73b0e10d0dfd3043b3f4f100ada014c5c37bd5ce97813b13f5ab2bcf")
	require.NoError(t, err)
	digest3, err := hex.DecodeString("336bccfd826a5bf6a5c2c07a289e39b05cb68447c379fb1acdaf9afd3b3d8c67")
	require.NoError(t, err)
	big := bytes.Repeat([]byte{0xCA, 0xFE, 0xBA, 0xBE, 0xDE, 0xAD, 0xBE, 0xEF}, 19201)
	for _, test := range []struct {
		args  []rideType
		check func(int) bool
		fail  bool
		r     rideType
	}{
		{[]rideType{rideBytes(data)}, bytesSizeCheckV1V2, false, rideBytes(digest1)},
		{[]rideType{rideString("123")}, bytesSizeCheckV1V2, false, rideBytes(digest2)},
		{[]rideType{rideBytes(big)}, bytesSizeCheckV1V2, false, rideBytes(digest3)},
		{[]rideType{rideBytes(big)}, bytesSizeCheckV3V6, true, nil},
		{[]rideType{rideUnit{}}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{rideInt(12345)}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{rideString("dsfjsadfl"), rideInt(12345)}, bytesSizeCheckV1V2, true, nil},
	} {
		r, err := blake2b256(&mockRideEnvironment{checkMessageLengthFunc: test.check}, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestSha256(t *testing.T) {
	data1, err := hex.DecodeString("64617461")
	require.NoError(t, err)
	digest1, err := hex.DecodeString("3a6eb0790f39ac87c94f3856b2dd2c5d110e6811602261a9a923d3bb23adc8b7")
	require.NoError(t, err)
	digest2, err := hex.DecodeString("A665A45920422F9D417E4867EFDC4FB8A04A1F3FFF1FA07E998E86F7F7A27AE3")
	require.NoError(t, err)
	digest3, err := hex.DecodeString("956731b38f852244d2d20f8ae618f1f916a6d0694062f90f7a2d9eec9c2ece4e")
	require.NoError(t, err)
	big := bytes.Repeat([]byte{0xCA, 0xFE, 0xBA, 0xBE, 0xDE, 0xAD, 0xBE, 0xEF}, 19201)
	for _, test := range []struct {
		args  []rideType
		check func(int) bool
		fail  bool
		r     rideType
	}{
		{[]rideType{rideBytes(data1)}, bytesSizeCheckV1V2, false, rideBytes(digest1)},
		{[]rideType{rideString("123")}, bytesSizeCheckV1V2, false, rideBytes(digest2)},
		{[]rideType{rideBytes(big)}, bytesSizeCheckV1V2, false, rideBytes(digest3)},
		{[]rideType{rideBytes(big)}, bytesSizeCheckV3V6, true, nil},
		{[]rideType{rideUnit{}}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{rideInt(12345)}, bytesSizeCheckV1V2, true, nil},
		{[]rideType{rideString("dsfjsadfl"), rideInt(12345)}, bytesSizeCheckV1V2, true, nil},
	} {
		r, err := sha256(&mockRideEnvironment{checkMessageLengthFunc: test.check}, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestAddressFromPublicKey(t *testing.T) {
	t.SkipNow()
}

func TestWavesBalanceV3(t *testing.T) {
	t.SkipNow()
}

func TestWavesBalanceV4(t *testing.T) {
	t.SkipNow()
}

func TestAssetInfoV3(t *testing.T) {
	t.SkipNow()
}

func TestAssetInfoV4(t *testing.T) {
	t.SkipNow()
}

func TestBlockInfoByHeight(t *testing.T) {
	t.SkipNow()
}

func TestTransferByID(t *testing.T) {
	t.SkipNow()
}

func TestAddressToString(t *testing.T) {
	addr, err := proto.NewAddressFromString("3P2HNUd5VUPLMQkJmctTPEeeHumiPN2GkTb")
	require.NoError(t, err)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideAddress(addr)}, false, rideString("3P2HNUd5VUPLMQkJmctTPEeeHumiPN2GkTb")},
		{[]rideType{rideAddress(addr), rideString("xxx")}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideString("x")}, true, nil},
	} {
		r, err := addressToString(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestRSAVerify(t *testing.T) {
	pk, err := base64.StdEncoding.DecodeString("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAkDg8m0bCDX7fTbBlHZm+BZIHVOfC2I4klRbjSqwFi/eCdfhGjYRYvu/frpSO0LIm0beKOUvwat6DY4dEhNt2PW3UeQvT2udRQ9VBcpwaJlLreCr837sn4fa9UG9FQFaGofSww1O9eBBjwMXeZr1jOzR9RBIwoL1TQkIkZGaDXRltEaMxtNnzotPfF3vGIZZuZX4CjiitHaSC0zlmQrEL3BDqqoLwo3jq8U3Zz8XUMyQElwufGRbZqdFCeiIs/EoHiJm8q8CVExRoxB0H/vE2uDFK/OXLGTgfwnDlrCa/qGt9Zsb8raUSz9IIHx72XB+kOXTt/GOuW7x2dJvTJIqKTwIDAQAB")
	require.NoError(t, err)
	for i, test := range []struct {
		msg string
		alg rideType
		sig string
		ok  bool
	}{
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", newNoAlg(nil), "SjNvKuuJ8AnBjX8dIx3ums231M5AsVTIPrdonwvcH2lWqAOip8Bv3+hoYjt5jxPwtHxYylEJpJVXyL7q/uaxO8TATok1n/5gPd7ZzvuhuIpABe8Ot/MjcGmeI1Xdz6R6Mb+9QtSugXmy5zHqcqs4kpqQQfGSOwENktxPXqHZFKps9aR5rX945vjGbUV62EKeo76ItOdXMV+ZCN8M1denJTpEtl+Q29uEjaaCvsdwNPIR4JYqb56IjevhAt8kTXpfIypTvEKaeoMpbZaZDbIxtii2Qu+/6+HX4Mog4Bvid/FSj3qSIoPWs6UgqKnNLpMLoc3S2Foh7ZhedSDUvIH4eg==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", newMd5(nil), "Ab0sqqZApwpKOr/remFI5YxSpYEQfowygO31vDdlfCyFqPVg9zxgR6Vh0dMlZodD5cejEP91Jo1yPM4pB4BdyhAVe5EtbmT+ofDy5O2X3LGJbpGOMRyRL7Y2yr4kjDfJ3E7I+55OrThYgsv3taIliAgMV+3ZIqW9QGy4uxSLJaYbvSiLs5t26RHsm1f8pafT2QGZHDfn1KKRhCeYqtEcJIYbO92mXLUQQqFe4OCy4EayqhzEQblibAYJ14CHLfSrnabbRhvacy1RWkcchzYY3nJvyHznyNyBaYiGPgjVgeE2ZgPcIFwHEsCF7zLBzpS3gdbHk0OmhgI7LX9N5f2G0A==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", newSha1(nil), "IzCKTx0UY7t1+GZovIdDKRxe3NUvobJ7fRzcnC5rVrUdY6hZaL5Djg5M7tKG1C19BjmgzgQEZc4oSMXU1BbNJUsggXZ7XWNSi8QAZ3bvXoN2qzF2DsFoxqb6lb6nAU2Vh+oazE0tXSfVjEiN3i7q6LoZPfSdsY8Cc6WdvIQqTqYRB1H25AWVO7I3IniR/qG+5S66yD3fzIRwo/XsFLuHIkoT4Yhj2VXwnrogXvoIG1opNAGtO/ddWxSb3Ac7zJlmLdSPMZjr6SUYH+g+eKM8H3d8fU8hLuLd/0R3JKvClbGyRZI+IfszLoMlowyt3A4hjfP8EXXDXhVX0VyBNHryoA==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", newSha224(nil), "d6m/5WuSQbU2vOlFf74AS9zNRZEyuYBJ+CrLBSuqjIQdj74ewZtB32lfmBJxGQtABrPIl8cdlRE4sTugSc6Jcd8IpwNouNVeCRrwH90IlASOxt+3GlnNwSY2OTB7JOfn7zjLF2wbSMzBq0/qT+VmmpDFkcw7ibRAR8fYmBIQjHL9vH7WWILRJ+sF/JF2SUUkm1+dEEjq6Z6Xi0STDHcyTmBbq0ZFVOt8QRqxUVmIXq27laYjYpwtn+yQok7CT9ci3AyWYUbL4U+G+tMHEIlwBp13ItGOpkxprNKYnozdsuJvJM1XKSCN4fGKFJIlRgpRy06O6kZrIxAkQj2lDLLz+Q==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", newSha256(nil), "ajH3CIH9T/nfrtwK3OPlPqz4CG6cz/cZXxQ/EIYJSUYVsGFft7edg/VhWC/vvIINFeJXues5z5VoRkw79p9akFnd8yjLv1O2X4tkp4v4l0raQZmVwJ/+Be8GfFkNi0vMcYCRBZqHaVMAeEdiXfOS3df20SZyN4IAOyOZhY4JB2phAPZDFjqK/wU1hDL1JXl1v7xAkUeMSk+Sbpmw9XqaI/ntZ4t+VDwWAqs+aVKs65X5OKXMDLSNZZLocR6uul55n74DrmHn7VojYy4LQGDKMCAu9N/nome2vvZRmETXOZUX9zHGXuuQGGNuG+r+BiMDRTHVRIogGbjfMzWQMBwLgw==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", newSha384(nil), "UvoU7qoOFUmKB1P+mX2ddbPILfY0+9eLk3wtahkCPrWsnI4Bwf9yihi88erJNKyWbdhlYP7dVCcYBHOxCyDVuyoLSERimLrwoRD7aFKcwQdtQqIFInbxCenPOMS1QofjVAE0x1Vy+r6n9uh8hzKsDAP59zX2QE53BVZm0hXtRYykKxrm1hWxZdsQ90nncZ4gxb9Gp9M2TRiw1NFaRWungbbV5py64akqC9bJlLKBm5OXWkIrmoEubNJpJORo5IYS5c0Mi4f6nVn9l3UTCKP0lbjTc9LJPt8/UTASiQseaN8KfJTvRwHJkOOVIT0FFk96nBfo+lH1nCO8UW7m8n9Xvg==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", newSha512(nil), "cSV8v78EUUxnV9Z69jmsffjGfmtY5xVQt2W5i2MHZSIM9MQWhPdPTRGT4FmgfeyJZLn2AFNfBA61eR40PeSOyuSLGgrUuUERZEoYxdyl/9KQ7D9NT5K8JRBTtHowm5/zD7qhCPR+bJ4NiD9pRxTZb7MvmBRdJ0jeKRZYTBXTS6FULjxaEGB09Xr/gPQ7i0yGWjqYj52LzkLOErnTTPzTvhQssOmFU1mrQxFOqPFo++YYd48OLIMP0p4q3Swbxx+Em1PpisDRKW5i58UhIEPdveGyGgd3BDgTBAQ8rSkUIPQFgVtgDpgLJaTFvuT1E6v5xNzhS52mi7PhhMgeX1KIVg==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", newSha3224(nil), "L3jASa1P4HJ1XpnpZ3+ZfGxUEA20ApIXiiBWBUU9AoBkJIDx9WP1IjEQOR+4nkguqSvw7SXggH4YYzePwyOxiE1kZLM7U20tXZp/oJ/TqZVrcaMtiHpxWZBYZvTHCnTRjktflXy6Mxr6HVDuaVJaXVLrX6tPcqdw8/e/Cs7vcPdZdVCBGY4/LlQ46HUZQrEOApdCwcER8l3Bz2v7toTLjAnIGEbINuJ7+ye4zksw42WZG8eK2EvjOO8EylPbtWNmoqsED9O81y/HvDAY8419U9XUd/HOd7weKGNOYGZ+S3Rh0bPr7GvKQS5GvGWSxFPq3zmKyzBF7rXqBvv5vzBQ6Q==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", newSha3256(nil), "GsYnxcmQOOAZthDfPjKvU1z+F7SUKGRfpNiWNpjoj6Vf6vdbP8fk9votEvVyXWd13lHZgv2lgaPG5Bd/I8Yt+/H8GPhcr/M7H0/eiZ/1yWag7O0SDdQnOAYINVGaogjuI9GdmSt33BkrPaXWjt+Li1UggT4Zgj8M2uEFvkwkpM1XDHXZChM8wHi8RNHOOfbqcPomm9qai2B1kSlw6eVjaZEEJ3SKuMdvzcsEP1P/P3pOz3/7j4uSXR9T87U0nlY8n1QXBkfMc5LggnoX5XlEvTF7jT8vsSNBYXgpBcQ3farQxdAt+qXhnFj4dttZjPMvFQHDCUxgW4zcubLmcB8/rg==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", newSha3384(nil), "X7rTLJTY/ohbnOLG8hF9QqAPDzi5KCNxn1J3vQrslvTSCaNsQeI/CsVvmlusCfOqx5dI+X9cqQWLedHpxiMCbY3d+8OHKuIBd1Bs6oQuTNCnCVs9p/cyxiP2ZTbdZo5nACMW0F6DGnkLXGA1IPEBpKHTFCjhwZY+KHIwadLbtYOjqH0FfAuXytEA21IDgZIRvh0GdgbDQmzt88EPwcoUxSv+UQ99/5FMsedhrgS/fMmupmAG+DnX82xKGSRNtFe73gokrPEsXK0ldWsJhnIcTUCHXalFvYQo4HrYE8g3XBpuLC7iqHtngtk5dIZyv7nA7oT/H79OsXYXxCp8bMMs4A==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", newSha3512(nil), "bEJA5Ktjst5WLugaWh81QG31PzpJkpFkLkguiAkhEZKFWS/QRsK9Um6MHliLYqzVc3w/EvKVZkfCqLuANwHai2nuYplUwQYyBTdmIb/LuxIvuW0fL3ehajblDyQ2WhQrMBbiPgmgl6DeyeTFPqBSJSkIgT63A/J2yEUWN8iBXeqy80I8ulpHAT6NBfY/ThqSlpJbLuSN761LOkJhM3s2YxUg2O2ZZ/6DT4EnVN51vqioHfPqRxtWHCiTSV+/vXHD7UdiSwYsQC9432FtDpgsN5Fn0ndASUaMpsrpg5EgUk+rak4WwfgG3SZ1MRwBuE4iG9dk4w6tek48L32+sgqSpQ==", true},

		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newNoAlg(nil), "UpuBmo7cfUhjIcaN4A0kwciZWZCp3dYqsxLT4uZkJ8t+yqxDkr5BIiGTG7lbSHEqGZd6aIYWgpoOfvGUt5bgISYWysriFjMHI6FH0ObNPjj+ORyrPAzT1KTPzq5UkwC18VhmK1ZwTGtPfVPTjUagH5YRYHFD0c8uztt4QUIU3GB78l3ScjvYNpdiCsZAxcNFFF/wTfhALMr6KQwYGiWYAQCqzfErK70uqV6F9tZYs1JsZpN3y3OCAboZBzg1QvwBfzhttVwhmGNQrgYaMZmHFwyxzDz5abD/w3bpn2N7OGRApFQPXZLd74nI5H3xJS/9zW45cyv+qdPnMC5sP64epQ==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newMd5(nil), "hXYw1IaK6N0WVIOtzBOpZzaEQi/GW6CQaLW7mDYd1B1EnclE7Yd2wCVvmBs/DYQl+qtL4K4EnR0eQoI54L7S7m/0obN7tRz16f0ObLpGmra5JNlTifJRwLfz8ABoqecm271YOD1cDOScGcoEjC9ZTNJnBMCkHuAxsosk4WrxuOwrQ8cmBIpKq0rG88oHVMNlC8jT/d9ThIE5xxoLZF7Wek6mOhiB8vXhawXtd47SS4JSnAZg5oCuW+CHrlUy/CVy/IS7fvwAa/U/Sodg4pbHX/UKPSPBUCTeUIUDfiYyOBMbcL9WdgcdGFrHh7lnmzd+9reBDRk0aStl4klpe1WFDQ==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newSha1(nil), "KcGWAnsvh2uZbmeedd4dsq+MznQwmZEQ3VO3/HyW4+RMGfBemv0LYCjxMHqs6ztag7aJm/7kL+Rq+9YUol9KsnTx8HuwdDeBtzPBf4HrVKfcvxO20KRDmufq1B6Xy6QLN9dWSDnxjTxl0TFO9s/kbG9fdat84LP5Tl3EfEVA2Nm+lz97dt+foocz8iWWYVnd7g8yVkTB8iW8LPveW/mJvG1q5Agb4mfZIkqkptWtsbsfENBW7je3e/X1b4weJVGTuGN7CYImgMCzUpWpuhHcHs67EqMdFlc01i4w26oDD6WhxwTl+zgu7nA6/cjW+9qFhgPwDJFZM8hg7s4QtpsX1Q==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newSha224(nil), "g9n05IPksL54sQVEx1hd29mhFQ/Qb4ecNIZtm7Sk1c4O8CI0CwXfRRNL59Gk+V9oYk/14jCmgpdC0QdUjpjlEjgV7c7SjgIw7AEWP22+sLlBXpNI5uZ9stGep8aKm3fAeBjmEV3xmfvSuxxvJNC2gy0I4jtkGugrVpxul/euEhzFwWqUbbSG8fRizEsn8rUBLdsHMC9sH0rNq28UmuREgHiljNYK3G+PFMYOsgD/2u8YvgDy1vu59LOKX/2gNDmxELaPv4GZie0OmitEP4y5oufF0O4MZMtWEK1FACQvZoaZVVOPhZPwOaswauvGO7SIFSRzPLGQjORlsr+G4ZuTIg==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newSha256(nil), "DLJzXp0uFISoTFT5n8h914YXEHhAsqSv5UAP52YOOWueJugYchwMFXFP+joFE2KetmF5D7htnEZFAI6j1UShuaTlZzmsrybmLOVOSvk2TYApbj1Rus1ZkRIchjDepvrNFOz6K3uE4PZ47uF6zjX1K5kDN+bD8nWULDZvx5p4P4xWGAF4Y2Aczce6yZRDq6cwTrMA4xCJr21cDlVS7URpdfemDweLIXY9NXU0PcbKc6tkL1LD0ZDRtA/3DAzqy/ae1ObyPE14yC+6+++aXYR5qIOE6CqFb8sd+WLJgbIazgfJ5unIM6kcMMl4UzpeNHhekD4gjfw/r/XWCMsjSq/Hlg==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newSha384(nil), "VvgkOdf62PI9YTUT2VydTdYu9JUdag4UXJiG0gjg3z1xQf+651quqLINB0FC2yN4WL+xZhdlsSuoOVcug0FtM2wxMdWfSpMfSpGerG/u1nsc13MxRdyZLQkNOi8enxowxZGvmNFdOgppQaq9LD9a2ni2rWrQ1Fl+PWAfBIlv23PtQqM9uPJdw+IZTW/5N74TOPWhYMf0sa3oFuTjKr6S76pDKLPxfOzwrXu0oBH1g+CG8wIhxAt13khr2mtJIpu06biEaR/rKp7nBtdAiyDFB1CyNnPozAd0UcJEwXfL1k3+bR4hOknJ6D8BaqRNovICAl1knjf+ZWmt65rVeJX+CQ==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newSha512(nil), "GW4XYgmm8H8e+GWRpTcjouTD2l2oub43iT78fCkraobK+/tzWDAE8nxI05U2/9GHXHC68qLG2SdLyauXJA9YmAQBL/2Yh285YgBa5uSsBaswxuHxf82IxQ73nOj9Ek4zhi8Z32BSc46V2Kn9HFQdI3xMnbAQ1Cz+/uwfA1FeEyH3Q3sVNaE9IZheqFVopIVRV+jcma43fAPNls6ZCavQfv8MAdFsY+8SfhiifjeF+yH3vYKDWX5aG3qfFG15RTavUX2fV66OCLYhG0sGdyuirsn5cpbhVs2G+Pt1hkbs5hF59vTbLiDC+fU2gayTA4odImuyaKl35H5NO8t2h0JnoA==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newSha3224(nil), "PjKn0VMEyzqXwf/zTgbZfrCR3XSfHABwEY+cK+EURwScWxdnsQ8B/KFrmg6U1a5vj5DfdI7x2luHTUi3/UAhKvZiHAxCE+AT4o4QtIXKXn425fikQz8RyrFVAYcMEJIHOzGzaclVQaAuNKMQM44peIHxFVlGRL1ZuFzdlwPWSXTDT/LMIFxrH3IOSNiDnXPZxzjLIoC0TVVZLgNVJmypdLd9TYM5FB6mg/loBd9EuIbOLDVhZXrUuJfAk28ojhdYZWM+CLFh09UbByTZhYLT/6vs8xakA45+84GjAT5VZQOzLK7uR4OAzzMLYXpUTkZHa7x7+nnWjEn2zzVjrBV9SA==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newSha3256(nil), "OXVKJwtSoenRmwizPtpjh3sCNmOpU1tnXUnyzl+PEI1P9Rx20GkxkIXlysFT2WdbPn/HsfGMwGJW7YhrVkDXy4uAQxUxSgQouvfZoqGSPp1NtM8iVJOGyKiepgB3GxRzQsev2G8Ik47eNkEDVQa47ct9j198Wvnkf88yjSkK0KxR057MWAi20ipNLirW4ZHDAf1giv68mniKfKxsPWahOA/7JYkv18sxcsISQqRXM8nGI1UuSLt9ER7kIzyAk2mgPCiVlj0hoPGUytmbiUqvEM4QaJfCpR0wVO4f/fob6jwKkGT6wbtia+5xCD7bESIHH8ISDrdexZ01QyNP2r4enw==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newSha3384(nil), "BagFS/QgaVFTKGKpI+eMh+nMXCpI33y8jmatR6ap4fVPHtWY5+63vku3Q9uzr+4XPDclhNK3rtf+r6duZ0y4GU6M9bJuiYWPEYsq/M/M2BQ0pZVqBzYbCps2vDucaehOWS6ivU4Y9tfq+q1VOgZDZYzh9XiWfBL6pL1eIuPk/RMB11tcD91gpa0hKCD5yRzcHxmF+OVqdnyr9RT79TnR8yQ8Zf7qwBws/bPqMwvEmQsssK67wA+3vTrx8Gqgq1RfYqvIjY2llqrkeohld3O75wHAtbUFMXu8HbI4+fq1Jp3Jr/riVCScIQNv2TyPnPcWO0yfqCj+D86LGYoHoEOXrg==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newSha3512(nil), "cSsxjrYkwfagdcwmA+5emRGspA6132BE/zU/QiG0pXOcaJCFE/DQaz0zPFUv/+D4BBdTx/7T/fUKFA4b3oU9KQ3RvUWaUGruwURsQ10rbmVleQdh8eODSuW38r9Vf2n/qq6VvE/2LBTM8Kamd3/czE/5RAJyCcywFmOKMKkkV96asZlb/bBeBtRSz8ZDpbyGbjm2k/cC5sxuEYgR6X1veH0wmANIsrM04+Dj6AZ4LtpUfG7hNCDUpiONmeO5KpBGvN+3bHwxuNXz311CtpJZcsr5ONvtD4l7vPv7ggQB+C1x9VvZXuJaieyk8Gm5F4oGXXfgmKsve6vAlfonpl4pmg==", true},

		{"Z0gxI00zZzNkMmkjYFxCXDg0K2Yhek9Se0hTRnR3cypSMjQmWUc=", newNoAlg(nil), "SjNvKuuJ8AnBjX8dIx3ums231M5AsVTIPrdonwvcH2lWqAOip8Bv3+hoYjt5jxPwtHxYylEJpJVXyL7q/uaxO8TATok1n/5gPd7ZzvuhuIpABe8Ot/MjcGmeI1Xdz6R6Mb+9QtSugXmy5zHqcqs4kpqQQfGSOwENktxPXqHZFKps9aR5rX945vjGbUV62EKeo76ItOdXMV+ZCN8M1denJTpEtl+Q29uEjaaCvsdwNPIR4JYqb56IjevhAt8kTXpfIypTvEKaeoMpbZaZDbIxtii2Qu+/6+HX4Mog4Bvid/FSj3qSIoPWs6UgqKnNLpMLoc3S2Foh7ZhedSDUvIH4eg==", false},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newSha3512(nil), "hXYw1IaK6N0WVIOtzBOpZzaEQi/GW6CQaLW7mDYd1B1EnclE7Yd2wCVvmBs/DYQl+qtL4K4EnR0eQoI54L7S7m/0obN7tRz16f0ObLpGmra5JNlTifJRwLfz8ABoqecm271YOD1cDOScGcoEjC9ZTNJnBMCkHuAxsosk4WrxuOwrQ8cmBIpKq0rG88oHVMNlC8jT/d9ThIE5xxoLZF7Wek6mOhiB8vXhawXtd47SS4JSnAZg5oCuW+CHrlUy/CVy/IS7fvwAa/U/Sodg4pbHX/UKPSPBUCTeUIUDfiYyOBMbcL9WdgcdGFrHh7lnmzd+9reBDRk0aStl4klpe1WFDQ==", false},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", newSha512(nil), "cSV8v78EUUxnV9Z69jmsffjGfmtY5xVQt2W5i2MHZSIM9MQWhPdPTRGT4FmgfeyJZLn2AFNfBA61eR40PeSOyuSLGgrUuUERZEoYxdyl/9KQ7D9NT5K8JRBTtHowm5/zD7qhCPR+bJ4NiD9pRxTZb7MvmBRdJ0jeKRZYTBXTS6FULjxaEGB09Xr/gPQ7i0yGWjqYj52LzkLOErnTTPzTvhQssOmFU1mrQxFOqPFo++YYd48OLIMP0p4q3Swbxx+Em1PpisDRKW5i58UhIEPdveGyGgd3BDgTBAQ8rSkUIPQFgVtgDpgLJaTFvuT1E6v5xNzhS52mi7PhhMgeX1KIVg==", false},
	} {
		msg, err := base64.StdEncoding.DecodeString(test.msg)
		require.NoError(t, err)
		sig, err := base64.StdEncoding.DecodeString(test.sig)
		require.NoErrorf(t, err, "#%d", i)
		r, err := rsaVerify(nil, test.alg, rideBytes(msg), rideBytes(sig), rideBytes(pk))
		require.NoErrorf(t, err, "#%d", i)
		assert.Equalf(t, rideBoolean(test.ok), r, "#%d", i)
	}
}

func TestCheckMerkleProof(t *testing.T) {
	for _, test := range []struct {
		root   string
		proof  string
		leaf   string
		result bool
	}{
		{"eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk=", "ACCP8jyg8Rv62mE4IMD4FGATnUXEIoCIK0LMoQCjAGpl5AEg16lhBiAz+xB8hwUs8U7dTJeGmJQyWVfXmHqzA+b2YuUBICJEors9RDiMZNeWp2yIlJrpf/a4rZxTvI7yIx3D5pihACAaVrwYIveDbOb3uE+Hj1w+Tl0vornHqPT9pCja/TmfPgAgxGoHWeIYY3RDkfAyYD99LA6OXdiXaB9a86EifTMS728AINbkCaDKCXEc5i61+c3ewBPFoCCYMCyvIrDbmHAThKt4ACAUkUex5ycLaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw==", "AAAdIQ==", true},
		{"eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk=", "ACDdSC04SpOqrUb7PbWs5NaLSSm/k6d1eG0MgFwTDEeJXAAg0iC2Dfqsu4tJUQt+xiDjvHyxUVu664rKruVL8zs6c60AIKLhp/AFQkokTe/NMQnKFL5eTMvDlFejApmJxPY6Rp8XACAWrdgB8DwvPA8D04E9HgUjhKghAn5aqtZnuKcmpLHztQAgd2OG15WYz90r1WipgXwjdq9WhvMIAtvGlm6E3WYY12oAIJXPPVIdbwOTdUJvCgMI4iape2gvR55vsrO2OmJJtZUNASAya23YyBl+EpKytL9+7cPdkeMMWSjk0Bc0GNnqIisofQ==", "AAAc6w==", true},
		{"eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk=", "ASADLSXbJGHQ7MMNaAqIfuLAwkvd7pQNnSQKcRnd3TYA0gAgNqksHYDS1xq5mKOpcWhxdM9KtzAJwVlJ8RECYsm9PMkAIEYOaapf0SZM4wZS8nZ95byib0SgjBLy1XG676X6lvoAASBOVhj3XzjWhqziBwKr/2M6v9VYF026vuWwXieZWMUdSwEgPqfL+ywsEjtOpywTh+k4zz23LGD2KGWHqfJvD8/9WdgBICdQYY0pkNys0gKNdIzTMj3Ou1Ags2EgP237fvxZqR9yACAUkUex5ycLaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw==", "AAAc+w==", true},
		{"eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk=", "ACBlQ+wlERW7AiK0dPotu7wLCCaMcH+X2D9XEU+D8TSNbwEgld8vUreEqWpiFo0nMwUsiP6LPhi8XWpV6Gge/3edo5MBIFCGuyg86lVn9ga7hNacZPBNd6T5gtMk+5OWpO8HthAmASDPIhoSPwQ9YL5aa+S6MjaLNe74dY3/Mq/OrpP7C46/8wAg1FSDEXwBdMgQkmK245kByRV39HfsgpmTdbbYd85GqI0BICdQYY0pkNys0gKNdIzTMj3Ou1Ags2EgP237fvxZqR9yACAUkUex5ycLaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw==", "AAAIVw==", true},
		{"eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk=", "ACBlQ+wlERW7AiK0dPotu7wLCCaMcH+X2D9XEU+D8TSNbwEgld8vUreEqWpiFo0nMwUsiP6LPhi8XWpV6Gge/3edo5MBIFCGuyg86lVn9ga7hNacZPBNd6T5gtMk+5OWpO8HthAmASDPIhoSPwQ9YL5aa+S6MjaLNe74dY3/Mq/OrpP7C46/8wAg1FSDEXwBdMgQkmK245kByRV39HfsgpmTdbbYd85GqI0BICdQYY0pkNys0gKNdIzTMj3Ou1Ags2EgP237fvxZqR9yACAUkUex5ycLaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw==", "AAAdIQ==", false},
		{"AYzKgOs9ARx/ulwB5wBMAAsB//8Aj381Wv8lvRA8gMR/owBwlU8BsQD//7jAnABQ", "ACCtPAMekYsdrprYYtydmNgluQzuW4v8vw2V96ufptzLRAEgkZVHs/yAFKm+dzB6zGol3RqipV9n8J5tkgiA/xGxfIUBIIWgSXngwWlUvpTBVbUM9D2zGEcaLio1PlZNAgkUcpgtASBIvie1RD4kOXIEWFHyWKxGyXR+NAr1r/GX5huq/HOV+gAgHdWZ4xwPTlrgQjIL1M0aOephVd9bOEK4nO08qmyR54oAIJFT7UAb6kacEYQPYORHoMEUwF6hhVbuI3RBPcsMyg9SASCNjzIYs57ugoE56TuTjnSbtkKnJL2c0qxZ/NxEfVAf4w==", "AAASIA==", false},
		{"", "ACBx7RO4K2tuSrrQ+OG3jn8uAT2qKUlxAR1bEz/ucQEsWgAgLFOaa1LHOwhqzFou9Tece3AUeC0izlUraXyfAxnyLGMBIG/cdbO2OvahmTl/38TlRqUKZEhygqlov1KuxYPDLnPhACBUIRPanY7B4wSCGIQr8rifqw1PYIUwJB9Xj/ZFWpSRzwAgTzGXR+KVcknm5jJzJxZocqdtF14Hd8nJliISmI8lrLsAIDwdXWHBoJDzVc31XmVUOPJjgf4oezXhydg8W5nPU5NgACCVh+rJdfzMBUxlzl5N+EJ07X6/REWE8jmB4v319R0L9Q==", "AAAkig==", false},
		{"+YJ4xei2Zj95SyUgDAKivHRtZmtp9k1dex3r3MUO+qE=", "AAA=", "b25l", true},
		{"+YJ4xei2Zj95SyUgDAKivHRtZmtp9k1dex3r3MUO+qE=", "AAAA", "b25l", false},
		{"BvdlUdF6mChCHP/oLvVz2XXq9nBjTIivx8ekUxyv4Jc=", "AA==", "b25l", false},
		{"+YJ4xei2Zj95SyUgDAKivHRtZmtp9k1dex3r3MUO+qE=", "AAE=", "b25l", false},
		{"+YJ4xei2Zj95SyUgDAKivHRtZmtp9k1dex3r3MUO+qE=", "", "b25l", false},
		{"d9bgB9if3XVM9fe+48c4/VSEdfmqTSMTWkpJxUqLsVw=", "ACCTBUdXB60nbh3qIIdmjvj2HMyhJ5RBlNuSzRxDjsXfPQ==", "b25l", true},
		{"d9bgB9if3XVM9fe+48c4/VSEdfmqTSMTWkpJxUqLsVw=", "ASAG92VR0XqYKEIc/+gu9XPZder2cGNMiK/Hx6RTHK/glw==", "dHdv", true},
		{"r25IskaiOuKSJVv+YVyCTHqy+GUCWxnTSSVjP7YjRms=", "ACCTBUdXB60nbh3qIIdmjvj2HMyhJ5RBlNuSzRxDjsXfPQAgqV3r5um7EyOBsrm5cNxe166J2UTv0ecd4YAjwYvhw9gAIBkZnGsnn99LWW5ibWKO9dbcDa95qlCwdMUAZnQ0zsSHACBFc08tWQBIDwhWu2zaUlkBuGP/BOJBwEa0TDaPTTDq1A==", "b25l", true},
		{"r25IskaiOuKSJVv+YVyCTHqy+GUCWxnTSSVjP7YjRms=", "ASAG92VR0XqYKEIc/+gu9XPZder2cGNMiK/Hx6RTHK/glwAgqV3r5um7EyOBsrm5cNxe166J2UTv0ecd4YAjwYvhw9gAIBkZnGsnn99LWW5ibWKO9dbcDa95qlCwdMUAZnQ0zsSHACBFc08tWQBIDwhWu2zaUlkBuGP/BOJBwEa0TDaPTTDq1A==", "dHdv", true},
		{"r25IskaiOuKSJVv+YVyCTHqy+GUCWxnTSSVjP7YjRms=", "ACDFNqa4s+qMMjyHDkd3wALrBuWlReRrEMUhEGINmPxBtgEgd9bgB9if3XVM9fe+48c4/VSEdfmqTSMTWkpJxUqLsVwAIBkZnGsnn99LWW5ibWKO9dbcDa95qlCwdMUAZnQ0zsSHACBFc08tWQBIDwhWu2zaUlkBuGP/BOJBwEa0TDaPTTDq1A==", "dGhyZWU=", true},
		{"r25IskaiOuKSJVv+YVyCTHqy+GUCWxnTSSVjP7YjRms=", "ASD+Q5E4EjGnbX2bOLx4VAUOFsTMAxlunOYRHktspTHaWAEgd9bgB9if3XVM9fe+48c4/VSEdfmqTSMTWkpJxUqLsVwAIBkZnGsnn99LWW5ibWKO9dbcDa95qlCwdMUAZnQ0zsSHACBFc08tWQBIDwhWu2zaUlkBuGP/BOJBwEa0TDaPTTDq1A==", "Zm91cg==", true},
		{"r25IskaiOuKSJVv+YVyCTHqy+GUCWxnTSSVjP7YjRms=", "ACDmklNa/tWyRtlwDn9zniHF5UlBCWg4j9ac0zB7Uyt3jQAgZc5JlfwzRL+/zvy0UBFYn2KH9ucvfM6EfLoDxYHADZ8BIL0hz1FSQpvkYrqjh6T7U8T1pX0qXnoMMatb5C/rlgxOACBFc08tWQBIDwhWu2zaUlkBuGP/BOJBwEa0TDaPTTDq1A==", "Zml2ZQ==", true},
		{"r25IskaiOuKSJVv+YVyCTHqy+GUCWxnTSSVjP7YjRms=", "ASBouwl99wCYVUNxalwv92EdY8VyCLzE6fDBtENWPnQEdgAgZc5JlfwzRL+/zvy0UBFYn2KH9ucvfM6EfLoDxYHADZ8BIL0hz1FSQpvkYrqjh6T7U8T1pX0qXnoMMatb5C/rlgxOACBFc08tWQBIDwhWu2zaUlkBuGP/BOJBwEa0TDaPTTDq1A==", "c2l4", true},
		{"r25IskaiOuKSJVv+YVyCTHqy+GUCWxnTSSVjP7YjRms=", "ACD7UfkUEaMeNZ2K/nNt7gpZ8NClGYsfOkZPIT+jdw5oUAEgCxgJy40nSZ8FFaSvkLG2q9zFSGCMZRVIhIp8Mq6pAHMBIL0hz1FSQpvkYrqjh6T7U8T1pX0qXnoMMatb5C/rlgxOACBFc08tWQBIDwhWu2zaUlkBuGP/BOJBwEa0TDaPTTDq1A==", "c2V2ZW4=", true},
		{"r25IskaiOuKSJVv+YVyCTHqy+GUCWxnTSSVjP7YjRms=", "ASCzQhdi1jyM+eQd3s7/zZYRvpiI4uuHcRSHwsHIvIly+AEgCxgJy40nSZ8FFaSvkLG2q9zFSGCMZRVIhIp8Mq6pAHMBIL0hz1FSQpvkYrqjh6T7U8T1pX0qXnoMMatb5C/rlgxOACBFc08tWQBIDwhWu2zaUlkBuGP/BOJBwEa0TDaPTTDq1A==", "ZWlnaHQ=", true},
		{"r25IskaiOuKSJVv+YVyCTHqy+GUCWxnTSSVjP7YjRms=", "AAAAAAAAASBm6bRiUqWGohBhYktqMWoCLx/rxN/pzAD6Q1PyAj4H9w==", "bmluZQ==", true},
	} {
		root, err := base64.StdEncoding.DecodeString(test.root)
		require.NoError(t, err)
		proof, err := base64.StdEncoding.DecodeString(test.proof)
		require.NoError(t, err)
		leaf, err := base64.StdEncoding.DecodeString(test.leaf)
		require.NoError(t, err)
		r, err := checkMerkleProof(nil, rideBytes(root), rideBytes(proof), rideBytes(leaf))
		require.NoError(t, err)
		assert.Equal(t, rideBoolean(test.result), r)
	}
}

func TestIntValueFromState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	correctAlias := proto.NewAlias('T', "good")
	incorrectAddress := proto.MustAddressFromString("3N3isZTp6tchjYox99bpxFkqxxySKY6FQsi")
	incorrectAlias := proto.NewAlias('T', "bad")
	correctAddressRecipient := proto.NewRecipientFromAddress(correctAddress)
	correctAliasRecipient := proto.NewRecipientFromAlias(*correctAlias)
	incorrectAddressRecipient := proto.NewRecipientFromAddress(incorrectAddress)
	incorrectAliasRecipient := proto.NewRecipientFromAlias(*incorrectAlias)
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					if (account.Eq(correctAddressRecipient) || account.Eq(correctAliasRecipient)) && key == "key" {
						return &proto.IntegerDataEntry{Key: "key", Value: 100500}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("key")}, false, rideInt(100500)},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("key")}, false, rideInt(100500)},
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("xxx")}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("xxx")}, true, nil},
		{[]rideType{recipientToObject(incorrectAddressRecipient), rideString("key")}, true, nil},
		{[]rideType{recipientToObject(incorrectAliasRecipient), rideString("key")}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{recipientToObject(correctAddressRecipient)}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient)}, true, nil},
		{[]rideType{rideString("xxx"), rideInt(12345)}, true, nil},
		{[]rideType{recipientToObject(correctAddressRecipient), rideInt(12345)}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient), rideInt(12345)}, true, nil},
	} {
		r, err := intValueFromState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBytesValueFromState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	correctAlias := proto.NewAlias('T', "good")
	incorrectAddress := proto.MustAddressFromString("3N3isZTp6tchjYox99bpxFkqxxySKY6FQsi")
	incorrectAlias := proto.NewAlias('T', "bad")
	correctAddressRecipient := proto.NewRecipientFromAddress(correctAddress)
	correctAliasRecipient := proto.NewRecipientFromAlias(*correctAlias)
	incorrectAddressRecipient := proto.NewRecipientFromAddress(incorrectAddress)
	incorrectAliasRecipient := proto.NewRecipientFromAlias(*incorrectAlias)
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
					if (account.Eq(correctAddressRecipient) || account.Eq(correctAliasRecipient)) && key == "key" {
						return &proto.BinaryDataEntry{Key: "key", Value: []byte("value")}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("key")}, false, rideBytes("value")},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("key")}, false, rideBytes("value")},
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("xxx")}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("xxx")}, true, nil},
		{[]rideType{recipientToObject(incorrectAddressRecipient), rideString("key")}, true, nil},
		{[]rideType{recipientToObject(incorrectAliasRecipient), rideString("key")}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{recipientToObject(correctAddressRecipient)}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient)}, true, nil},
		{[]rideType{rideString("xxx"), rideInt(12345)}, true, nil},
		{[]rideType{recipientToObject(correctAddressRecipient), rideInt(12345)}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient), rideInt(12345)}, true, nil},
	} {
		r, err := bytesValueFromState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestStringValueFromState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	correctAlias := proto.NewAlias('T', "good")
	incorrectAddress := proto.MustAddressFromString("3N3isZTp6tchjYox99bpxFkqxxySKY6FQsi")
	incorrectAlias := proto.NewAlias('T', "bad")
	correctAddressRecipient := proto.NewRecipientFromAddress(correctAddress)
	correctAliasRecipient := proto.NewRecipientFromAlias(*correctAlias)
	incorrectAddressRecipient := proto.NewRecipientFromAddress(incorrectAddress)
	incorrectAliasRecipient := proto.NewRecipientFromAlias(*incorrectAlias)
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					if (account.Eq(correctAddressRecipient) || account.Eq(correctAliasRecipient)) && key == "key" {
						return &proto.StringDataEntry{Key: "key", Value: "value"}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("key")}, false, rideString("value")},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("key")}, false, rideString("value")},
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("xxx")}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("xxx")}, true, nil},
		{[]rideType{recipientToObject(incorrectAddressRecipient), rideString("key")}, true, nil},
		{[]rideType{recipientToObject(incorrectAliasRecipient), rideString("key")}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{recipientToObject(correctAddressRecipient)}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient)}, true, nil},
		{[]rideType{rideString("xxx"), rideInt(12345)}, true, nil},
		{[]rideType{recipientToObject(correctAddressRecipient), rideInt(12345)}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient), rideInt(12345)}, true, nil},
	} {
		r, err := stringValueFromState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBooleanValueFromState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	correctAlias := proto.NewAlias('T', "good")
	incorrectAddress := proto.MustAddressFromString("3N3isZTp6tchjYox99bpxFkqxxySKY6FQsi")
	incorrectAlias := proto.NewAlias('T', "bad")
	correctAddressRecipient := proto.NewRecipientFromAddress(correctAddress)
	correctAliasRecipient := proto.NewRecipientFromAlias(*correctAlias)
	incorrectAddressRecipient := proto.NewRecipientFromAddress(incorrectAddress)
	incorrectAliasRecipient := proto.NewRecipientFromAlias(*incorrectAlias)
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
					if (account.Eq(correctAddressRecipient) || account.Eq(correctAliasRecipient)) && key == "key" {
						return &proto.BooleanDataEntry{Key: "key", Value: true}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("key")}, false, rideBoolean(true)},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("key")}, false, rideBoolean(true)},
		{[]rideType{recipientToObject(correctAddressRecipient), rideString("xxx")}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient), rideString("xxx")}, true, nil},
		{[]rideType{recipientToObject(incorrectAddressRecipient), rideString("key")}, true, nil},
		{[]rideType{recipientToObject(incorrectAliasRecipient), rideString("key")}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{recipientToObject(correctAddressRecipient)}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient)}, true, nil},
		{[]rideType{rideString("xxx"), rideInt(12345)}, true, nil},
		{[]rideType{recipientToObject(correctAddressRecipient), rideInt(12345)}, true, nil},
		{[]rideType{recipientToObject(correctAliasRecipient), rideInt(12345)}, true, nil},
	} {
		r, err := booleanValueFromState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}
func TestIntValueFromSelfState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestIntegerEntryFunc: func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
					if *account.Address() == correctAddress && key == "key" {
						return &proto.IntegerDataEntry{Key: "key", Value: 100500}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(correctAddress)
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("key")}, false, rideInt(100500)},
		{[]rideType{rideString("xxx")}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideString("xxx"), rideInt(12345)}, true, nil},
	} {
		r, err := intValueFromSelfState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBytesValueFromSelfState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestBinaryEntryFunc: func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
					if *account.Address() == correctAddress && key == "key" {
						return &proto.BinaryDataEntry{Key: "key", Value: []byte("value")}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(correctAddress)
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("key")}, false, rideBytes("value")},
		{[]rideType{rideString("xxx")}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideString("xxx"), rideInt(12345)}, true, nil},
	} {
		r, err := bytesValueFromSelfState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestStringValueFromSelfState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestStringEntryFunc: func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
					if *account.Address() == correctAddress && key == "key" {
						return &proto.StringDataEntry{Key: "key", Value: "value"}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(correctAddress)
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("key")}, false, rideString("value")},
		{[]rideType{rideString("xxx")}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideString("xxx"), rideInt(12345)}, true, nil},
	} {
		r, err := stringValueFromSelfState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBooleanValueFromSelfState(t *testing.T) {
	correctAddress := proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7")
	env := &mockRideEnvironment{
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				RetrieveNewestBooleanEntryFunc: func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
					if *account.Address() == correctAddress && key == "key" {
						return &proto.BooleanDataEntry{Key: "key", Value: true}, nil
					}
					return nil, errors.New("not found")
				},
			}
		},
		thisFunc: func() rideType {
			return rideAddress(correctAddress)
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("key")}, false, rideBoolean(true)},
		{[]rideType{rideString("xxx")}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideString("xxx"), rideInt(12345)}, true, nil},
	} {
		r, err := booleanValueFromSelfState(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestTransferFromProtobuf(t *testing.T) {
	var scheme byte = 'T'
	te := &mockRideEnvironment{schemeFunc: func() byte {
		return 'T'
	}}
	seed, err := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	require.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	ts := uint64(time.Now().UnixNano() / 1000000)
	addr, err := proto.NewAddressFromPublicKey(scheme, pk)
	require.NoError(t, err)
	rcp := proto.NewRecipientFromAddress(addr)
	att := []byte("some attachment")
	tx := proto.NewUnsignedTransferWithProofs(3, pk, proto.OptionalAsset{}, proto.OptionalAsset{}, ts, 1234500000000, 100000, rcp, att)
	err = tx.GenerateID(scheme)
	require.NoError(t, err)
	err = tx.Sign(scheme, sk)
	require.NoError(t, err)
	bts, err := tx.MarshalSignedToProtobuf(scheme)
	require.NoError(t, err)

	for _, test := range []struct {
		args []rideType
		fail bool
		inst rideType
		id   rideType
	}{
		{[]rideType{rideBytes(bts)}, false, rideString("TransferTransaction"), rideBytes(tx.ID.Bytes())},
		{[]rideType{rideUnit{}}, true, nil, nil},
		{[]rideType{}, true, nil, nil},
		{[]rideType{rideString("x")}, true, nil, nil},
	} {
		r, err := transferFromProtobuf(te, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			inst, err := r.get(instanceField)
			require.NoError(t, err)
			assert.Equal(t, test.inst, inst)
			id, err := r.get("id")
			require.NoError(t, err)
			assert.Equal(t, test.id, id)
		}
	}
}

func TestCalculateAssetID(t *testing.T) {
	t.SkipNow()
}

func TestSimplifiedIssue(t *testing.T) {
	t.SkipNow()
}

func TestFullIssue(t *testing.T) {
	t.SkipNow()
}

func TestRebuildMerkleRoot(t *testing.T) {
	leaf, err := base58.Decode("7jsrwD9Xi7TjVoksaV1CDDUWYhFaz7HQmAoWwLEiZa6D")
	require.NoError(t, err)
	root, err := base58.Decode("6tt3obq44UqC4QwLhrKX2KsXV9GRBfhiNvzor2BQfgYZ")
	require.NoError(t, err)
	p1, err := base58.Decode("q1u2PJhro1cwZw5mUuujXm94f245tGS5vbP5yNwLbEv")
	require.NoError(t, err)
	p2, err := base58.Decode("75Aaexax3uEQNg5HAb137jC3TK64RG1S6xrBGvuupWXp")
	require.NoError(t, err)
	p3, err := base58.Decode("GemGCop1arCvTY447FLH8tDQF7knvzNCocNTHqKQBus9")
	require.NoError(t, err)
	r, err := rebuildMerkleRoot(nil, rideList{rideBytes(p1), rideBytes(p2), rideBytes(p3)}, rideBytes(leaf), rideInt(3))
	assert.NoError(t, err)
	assert.Equal(t, "ByteVector", r.instanceOf())
	assert.ElementsMatch(t, root, r)
}

func TestBLS12Groth16Verify(t *testing.T) {
	t.SkipNow()
}

func TestBN256Groth16Verify(t *testing.T) {
	t.SkipNow()
}

func TestECRecover(t *testing.T) {
	t.SkipNow()
}

func TestAddressFromPublicKeyStrict(t *testing.T) {
	te := &mockRideEnvironment{schemeFunc: func() byte {
		return 'T'
	}}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideBytes(guessBytesFromString(t, "qhZIsJQ2+At/RHmPBsLuG3sMSZJfQTDhJOgzPtisRUg="))}, false, rideAddress(proto.MustAddressFromString("3Mp5JgVSHA9iziujC9Kmnf2rCN5SYFE97yC"))},
		{[]rideType{rideBytes(guessBytesFromString(t, "0QoVC6mlNRJUgeAXoJwqxqGrQ/xD96uPDURjUQZnLdfeT3dcBrcwSDhiy8Q3GmRtht93s4FVk6hGtycqzgCMQg=="))}, false, rideAddress(proto.MustAddressFromString("3N2sMJ78BuYwoLHreuwjbk6dZgsnudxecBR"))},
		{[]rideType{rideBytes(guessBytesFromString(t, "yv6+vt6tvu/K/r6+3q2+78r+vr7erb7vyv6+vt6tvu8A/w=="))}, true, nil},
		{[]rideType{rideBytes(guessBytesFromString(t, "yv6+vt6tvu/K/r6+3q2+78r+vr7erb7vyv6+vt6tvu/K/r6+3q2+78r+vr7erb7vyv6+vt6tvu/K/r6+3q2+7/8="))}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideString("x")}, true, nil},
	} {
		r, err := addressFromPublicKeyStrict(te, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			a, ok := r.(rideAddress)
			assert.True(t, ok)
			assert.Equal(t, test.r, a)
		}
	}
}

func TestHashScriptAtAddress(t *testing.T) {
	r1 := proto.NewRecipientFromAddress(proto.MustAddressFromString("3Mp5JgVSHA9iziujC9Kmnf2rCN5SYFE97yC"))
	r2 := proto.NewRecipientFromAlias(*proto.NewAlias('T', "test"))
	r3 := proto.NewRecipientFromAddress(proto.MustAddressFromString("3N2sMJ78BuYwoLHreuwjbk6dZgsnudxecBR"))
	r4 := proto.NewRecipientFromAlias(*proto.NewAlias('T', "empty"))
	r5 := proto.NewRecipientFromAddress(proto.MustAddressFromString("3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7"))
	s1 := []byte("fake script bytes 1")
	d1, err := crypto.FastHash(s1)
	require.NoError(t, err)
	s2 := []byte("fake script bytes 2")
	d2, err := crypto.FastHash(s2)
	require.NoError(t, err)
	te := &mockRideEnvironment{
		schemeFunc: func() byte {
			return 'T'
		},
		stateFunc: func() types.SmartState {
			return &MockSmartState{
				NewestScriptBytesByAccountFunc: func(recipient proto.Recipient) (proto.Script, error) {
					switch {
					case recipient.Eq(r1):
						return s1, nil
					case recipient.Eq(r2):
						return s2, nil
					case recipient.Eq(r3), recipient.Eq(r4):
						return nil, errors.Wrap(keyvalue.ErrNotFound, "blah-blah")
					default:
						return nil, errors.New("other error")
					}
				},
			}
		},
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{recipientToObject(r1)}, false, rideBytes(d1[:])},
		{[]rideType{recipientToObject(r2)}, false, rideBytes(d2[:])},
		{[]rideType{recipientToObject(r3)}, false, rideUnit{}},
		{[]rideType{recipientToObject(r4)}, false, rideUnit{}},
		{[]rideType{recipientToObject(r5)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideString("x")}, true, nil},
	} {
		r, err := hashScriptAtAddress(te, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			switch rr := r.(type) {
			case rideBytes:
				assert.Equal(t, test.r, rr)
			case rideUnit:
				assert.Equal(t, test.r, rr)
			default:
				assert.Fail(t, "unexpected result type")
			}
		}
	}
}
