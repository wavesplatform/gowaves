package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
)

// ethABIDataTypeToRideType perform conversion of ethabi.DataType to rideType.
// Note that this function doesn't copy ethabi.Bytes and ethabi.BigInt. It only copies a pointer to type.
func ethABIDataTypeToRideType(dataType ethabi.DataType) (rideType rideType, err error) {
	switch t := dataType.(type) {
	case ethabi.Int:
		rideType = rideInt(t)
	case ethabi.BigInt:
		rideType = rideBigInt{v: t.V}
	case ethabi.Bool:
		rideType = rideBoolean(t)
	case ethabi.Bytes:
		rideType = rideBytes(t)
	case ethabi.String:
		rideType = rideString(t)
	case ethabi.List:
		rideList := make(rideList, len(t))
		for i, ethABIElem := range t {
			rideElem, err := ethABIDataTypeToRideType(ethABIElem)
			if err != nil {
				return nil, errors.Wrapf(err,
					"failed to convert ethabi.DataType (%T) to rideType at %d list postition", ethABIElem, i,
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
