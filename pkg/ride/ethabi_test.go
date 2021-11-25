package ride

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
)

func TestEthABIDataTypeToRideType(t *testing.T) {
	hugeInt, ok := new(big.Int).SetString("123454323456434285767546723400991456870502323864587234659828639850098161345465903596567", 10)
	require.True(t, ok)

	tests := []struct {
		inputDataType    ethabi.DataType
		expectedRideType rideType
	}{
		{ethabi.Int(5345345), rideInt(5345345)},
		{ethabi.BigInt{V: hugeInt}, rideBigInt{v: hugeInt}},
		{ethabi.Bool(true), rideBoolean(true)},
		{ethabi.Bool(false), rideBoolean(false)},
		{ethabi.Bytes("#This is Test bytes!"), rideBytes("#This is Test bytes!")},
		{ethabi.String("This is @ Test string!"), rideString("This is @ Test string!")},
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
			expectedRideType: rideList{
				rideInt(453),
				rideBoolean(true),
				rideString("the best test string ever!"),
				rideBytes("command and conquer!"),
				rideBigInt{v: big.NewInt(1232347)},
				rideList{
					rideBytes("one more"),
					rideBoolean(false),
				},
			},
		},
	}
	for _, tc := range tests {
		actualRideType, err := ethABIDataTypeToRideType(tc.inputDataType)
		require.NoError(t, err)
		require.Equal(t, tc.expectedRideType, actualRideType)
	}
}
