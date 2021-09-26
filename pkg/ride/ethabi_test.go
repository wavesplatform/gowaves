package ride

import (
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"math/big"
	"testing"
)

func TestEthABIDataTypeToRideType(t *testing.T) {
	hugeInt, ok := new(big.Int).SetString("123454323456434285767546723400991456870502323864587234659828639850098161345465903596567", 10)
	require.True(t, ok)

	tests := []struct {
		inputDataType    ethabi.DataType
		expectedRideType RideType
	}{
		{ethabi.Int(5345345), RideInt(5345345)},
		{ethabi.BigInt{V: hugeInt}, RideBigInt{V: hugeInt}},
		{ethabi.Bool(true), RideBoolean(true)},
		{ethabi.Bool(false), RideBoolean(false)},
		{ethabi.Bytes("#This is Test bytes!"), RideBytes("#This is Test bytes!")},
		{ethabi.String("This is @ Test string!"), RideString("This is @ Test string!")},
		{
			inputDataType: ethabi.List{
				ethabi.Int(453),
				ethabi.Bool(true),
				ethabi.String("the best test string ever!"),
				ethabi.Bytes("command and conquer!"),
				ethabi.BigInt{V: big.NewInt(1232347)},
				ethabi.List{
					ethabi.Bytes("one more"),
					ethabi.Bool(false),
				},
			},
			expectedRideType: RideList{
				RideInt(453),
				RideBoolean(true),
				RideString("the best test string ever!"),
				RideBytes("command and conquer!"),
				RideBigInt{V: big.NewInt(1232347)},
				RideList{
					RideBytes("one more"),
					RideBoolean(false),
				},
			},
		},
	}
	for _, tc := range tests {
		actualRideType, err := EthABIDataTypeToRideType(tc.inputDataType)
		require.NoError(t, err)
		require.Equal(t, tc.expectedRideType, actualRideType)
	}
}
