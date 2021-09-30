package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
)

// EthABIDataTypeToRideType perform conversion of ethabi.DataType to RideType.
// Note that this function doesn't copy ethabi.Bytes and ethabi.BigInt. It only copies a pointer to type.
func EthABIDataTypeToRideType(dataType ethabi.DataType) (rideType RideType, err error) {
	switch t := dataType.(type) {
	case ethabi.Int:
		rideType = RideInt(t)
	case ethabi.BigInt:
		rideType = RideBigInt{V: t.V}
	case ethabi.Bool:
		rideType = RideBoolean(t)
	case ethabi.Bytes:
		rideType = RideBytes(t)
	case ethabi.String:
		rideType = RideString(t)
	case ethabi.List:
		rideList := make(RideList, len(t))
		for i, ethABIElem := range t {
			rideElem, err := EthABIDataTypeToRideType(ethABIElem)
			if err != nil {
				return nil, errors.Wrapf(err,
					"failed to convert ethabi.DataType (%T) to RideType at %d list postition", ethABIElem, i,
				)
			}
			rideList[i] = rideElem
		}
		rideType = rideList
	default:
		return nil, errors.Errorf(
			"ethabi.DataType (%T) to RIdeType converstion doesn't supported", dataType,
		)
	}
	return rideType, nil
}
