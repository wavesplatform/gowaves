package fourbyte

import (
	"encoding/binary"
	stdErr "errors"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/metamask"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"math/big"
)

var (
	errBadBool = stdErr.New("abi: improperly encoded boolean value")
)

// readBool reads a bool.
func readBool(word []byte) (bool, error) {
	for _, b := range word[:31] {
		if b != 0 {
			return false, errBadBool
		}
	}
	switch word[31] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, errBadBool
	}
}

// readRideInteger reads the integer based on its kind and returns the appropriate value.
func readRideInteger(typ Type, b []byte) ride.RideType {
	if typ.T == UintTy {
		switch typ.Size {
		case 8:
			return ride.RideInt(b[len(b)-1])
		case 16:
			return ride.RideInt(binary.BigEndian.Uint16(b[len(b)-2:]))
		case 32:
			return ride.RideInt(binary.BigEndian.Uint32(b[len(b)-4:]))
		case 64:
			return ride.RideInt(binary.BigEndian.Uint64(b[len(b)-8:]))
		default:
			// the only case left for unsigned integer is uint256.
			return ride.RideBigInt{V: new(big.Int).SetBytes(b)}
		}
	}
	switch typ.Size {
	case 8:
		return ride.RideInt(int8(b[len(b)-1]))
	case 16:
		return ride.RideInt(int16(binary.BigEndian.Uint16(b[len(b)-2:])))
	case 32:
		return ride.RideInt(int32(binary.BigEndian.Uint32(b[len(b)-4:])))
	case 64:
		return ride.RideInt(int64(binary.BigEndian.Uint64(b[len(b)-8:])))
	default:
		// the only case left for integer is int256
		// big.SetBytes can't tell if a number is negative or positive in itself.
		// On EVM, if the returned number > max int256, it is negative.
		// A number is > max int256 if the bit at position 255 is set.
		ret := new(big.Int).SetBytes(b)
		if ret.Bit(255) == 1 {
			ret.Add(MaxUint256, new(big.Int).Neg(ret))
			ret.Add(ret, Big1)
			ret.Neg(ret)
		}
		return ride.RideBigInt{V: ret}
	}
}

func tryAsInt64(rideT ride.RideType) (int64, error) {
	switch rideInt := rideT.(type) {
	case ride.RideInt:
		return int64(rideInt), nil

	case ride.RideBigInt:
		if !rideInt.V.IsInt64() {
			return 0, errors.New(
				"abi: failed to convert BigInt as int64, value too big",
			)
		}
		return rideInt.V.Int64(), nil
	default:
		return 0, errors.Errorf("abi: failed to convert RideType as int64, type is not number")
	}
}

// forEachUnpack iteratively unpack elements.
func forEachUnpackRideList(t Type, output []byte, start, size int) (ride.RideList, error) {
	if size < 0 {
		return nil, errors.Errorf("cannot marshal input to array, size is negative (%d)", size)
	}
	if start+32*size > len(output) {
		return nil, errors.Errorf(
			"abi: cannot marshal in to go array: offset %d would go over slice boundary (len=%d)",
			len(output), start+32*size,
		)
	}
	if t.T != SliceTy {
		return nil, errors.Errorf("abi: invalid type in slice unpacking stage")

	}

	// this value will become our slice or our array, depending on the type
	refSlice := make(ride.RideList, 0, size)

	// Arrays have packed elements, resulting in longer unpack steps.
	// Slices have just 32 bytes per element (pointing to the contents).
	elemSize := getTypeSize(*t.Elem)

	for i, j := start, 0; j < size; i, j = i+elemSize, j+1 {
		inter, err := toRideType(i, *t.Elem, output)
		if err != nil {
			return nil, err
		}

		// append the item to our reflect slice
		refSlice = append(refSlice, inter)
	}

	// return the interface
	return refSlice, nil
}

func extractIndexFromFirstElemOfTuple(index int, t Type, output []byte) (int64, error) {
	if t.T != IntTy && t.T != UintTy {
		return 0, errors.New(
			"abi: failed to convert eth tuple to ride union, first element of eth tuple must be a number",
		)
	}
	rideT, err := toRideType(index, t, output)
	if err != nil {
		return 0, err
	}
	return tryAsInt64(rideT)

}

func forUnionTupleUnpackToRideType(t Type, output []byte) (ride.RideType, error) {
	if t.T != TupleTy {
		return nil, errors.New("abi: type in forTupleUnpack must be TupleTy")
	}
	if len(t.TupleElems) < 2 {
		return nil, errors.New(
			"abi: failed to convert eth tuple to ride union, elements count of eth tuple must greater than 2",
		)
	}
	unionIndex, err := extractIndexFromFirstElemOfTuple(0, t.TupleElems[0], output)
	if err != nil {
		return nil, err
	}
	elems := t.TupleElems[1:]
	if unionIndex >= int64(len(elems)) {
		return nil, errors.Errorf(
			"abi: failed to convert eth tuple to ride union, union index (%d) greater than tuple elems count (%d)",
			unionIndex, len(elems),
		)
	}
	retval := make([]ride.RideType, 0, len(elems))
	virtualArgs := 0
	for index := 1; index < len(elems); index++ {
		elem := elems[index]
		marshalledValue, err := toRideType((index+virtualArgs)*32, elem, output)
		if err != nil {
			return nil, err
		}
		if elem.T == TupleTy && !isDynamicType(elem) {

			virtualArgs += getTypeSize(elem)/32 - 1
		}
		retval = append(retval, marshalledValue)
	}
	return retval[unionIndex], nil
}

type Payment struct {
	AssetID metamask.Address
	Amount  int64
}

var (
	paymentType = Type{
		T: TupleTy,
		TupleElems: []Type{
			{T: BytesTy},
			{Size: 64, T: IntTy},
		},
		TupleRawNames: []string{
			"id",
			"value",
		},
	}
	paymentsType = Type{
		Elem: &paymentType,
		T:    SliceTy,
	}
	paymentsArgument = Argument{
		Name: "payments",
		Type: paymentsType,
	}
)

func unpackPayment(output []byte) (Payment, error) {
	assetIDType := paymentType.TupleElems[0]
	amountType := paymentType.TupleElems[1]

	var (
		assetID metamask.Address
		amount  int64
	)

	assetRideValue, err := toRideType(0, assetIDType, output)
	if err != nil {
		return Payment{}, errors.Wrap(err, "abi: failed to decode payment, failed to parse assetID")
	}
	if assetIDBytes, ok := assetRideValue.(ride.RideBytes); ok {
		assetID.SetBytes(assetIDBytes)
	} else {
		panic("BUG, CREATE REPORT: failed to parse payment, assetRideValue type must be RideBytes type")
	}

	amountRideValue, err := toRideType(1, amountType, output)
	if err != nil {
		return Payment{}, errors.Wrap(err, "abi: failed to decode payment, failed to parse amount")
	}
	if amount, err = tryAsInt64(amountRideValue); err != nil {
		panic("BUG, CREATE REPORT: failed to parse payment, amountRideValue type must be representable as int64")
	}

	payment := Payment{
		AssetID: assetID,
		Amount:  amount,
	}
	return payment, nil
}

func unpackPayments(output []byte) ([]Payment, error) {
	if len(output) == 0 {
		return nil, nil
	}

	begin, size, err := lengthPrefixPointsTo(0, output)
	if err != nil {
		return nil, err
	}
	// nickeskov: jumping to the data section
	output = output[begin:]

	if size < 0 {
		return nil, errors.Errorf("cannot marshal input to array, size is negative (%d)", size)
	}
	if 32*size > len(output) {
		return nil, errors.Errorf(
			"abi: cannot marshal in to go array: offset %d would go over slice boundary (len=%d)",
			len(output), 32*size,
		)
	}

	elemSize := getTypeSize(*paymentsType.Elem)
	payments := make([]Payment, 0, size)
	for i := 0; i < size; i++ {
		payment, err := unpackPayment(output[i*elemSize:])
		if err != nil {
			return nil, errors.Wrap(err, "failed to unpack payment")
		}
		payments = append(payments, payment)
	}
	return payments, nil
}

// lengthPrefixPointsTo interprets a 32 byte slice as an offset and then determines which indices to look to decode the type.
func lengthPrefixPointsTo(index int, output []byte) (start int, length int, err error) {
	// nickeskov: I have no idea how it works, but we should...

	bigOffsetEnd := big.NewInt(0).SetBytes(output[index : index+32])
	bigOffsetEnd.Add(bigOffsetEnd, Big32)
	outputLength := big.NewInt(int64(len(output)))

	if bigOffsetEnd.Cmp(outputLength) > 0 {
		return 0, 0, errors.Errorf(
			"abi: cannot marshal in to go slice: offset %v would go over slice boundary (len=%v)",
			bigOffsetEnd, outputLength,
		)
	}

	if bigOffsetEnd.BitLen() > 63 {
		return 0, 0, errors.Errorf("abi offset larger than int64: %v", bigOffsetEnd)
	}

	offsetEnd := int(bigOffsetEnd.Uint64())
	lengthBig := big.NewInt(0).SetBytes(output[offsetEnd-32 : offsetEnd])

	totalSize := big.NewInt(0)
	totalSize.Add(totalSize, bigOffsetEnd)
	totalSize.Add(totalSize, lengthBig)
	if totalSize.BitLen() > 63 {
		return 0, 0, errors.Errorf("abi: length larger than int64: %v", totalSize)
	}

	if totalSize.Cmp(outputLength) > 0 {
		return 0, 0, errors.Errorf(
			"abi: cannot marshal in to go type: length insufficient %v require %v",
			outputLength, totalSize,
		)
	}
	start = int(bigOffsetEnd.Uint64())
	length = int(lengthBig.Uint64())
	return
}

// tuplePointsTo resolves the location reference for dynamic tuple.
func tuplePointsTo(index int, output []byte) (start int, err error) {
	offset := big.NewInt(0).SetBytes(output[index : index+32])
	outputLen := big.NewInt(int64(len(output)))

	if offset.Cmp(big.NewInt(int64(len(output)))) > 0 {
		return 0, errors.Errorf(
			"abi: cannot marshal in to go slice: offset %v would go over slice boundary (len=%v)",
			offset, outputLen,
		)
	}
	if offset.BitLen() > 63 {
		return 0, errors.Errorf("abi offset larger than int64: %v", offset)
	}
	return int(offset.Uint64()), nil
}

func toRideType(index int, t Type, output []byte) (ride.RideType, error) {
	if index+32 > len(output) {
		return nil, errors.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d",
			len(output), index+32,
		)
	}

	var (
		returnOutput  []byte
		begin, length int
		err           error
	)

	// if we require a length prefix, find the beginning word and size returned.
	if requiresLengthPrefix(t) {
		begin, length, err = lengthPrefixPointsTo(index, output)
		if err != nil {
			return nil, err
		}
	} else {
		returnOutput = output[index : index+32]
	}

	switch t.T {
	case TupleTy:
		if isDynamicType(t) {
			begin, err := tuplePointsTo(index, output)
			if err != nil {
				return nil, err
			}
			return forUnionTupleUnpackToRideType(t, output[begin:])
		}
		return forUnionTupleUnpackToRideType(t, output[index:])
	case SliceTy:
		return forEachUnpackRideList(t, output[begin:], 0, length)
	case StringTy: // variable arrays are written at the end of the return bytes
		return ride.RideString(output[begin : begin+length]), nil
	case IntTy, UintTy:
		return readRideInteger(t, returnOutput), nil
	case BoolTy:
		boolean, err := readBool(returnOutput)
		if err != nil {
			return nil, err
		}
		return ride.RideBoolean(boolean), nil
	case AddressTy:
		address := metamask.BytesToAddress(returnOutput)
		return ride.RideBytes(address.Bytes()), nil
	case BytesTy:
		bytes, err := ride.NewRideBytes(output[begin : begin+length])
		if err != nil {
			return nil, err
		}
		return bytes, nil
	default:
		return nil, errors.Errorf("abi: unknown type %v", t.T)
	}
}
