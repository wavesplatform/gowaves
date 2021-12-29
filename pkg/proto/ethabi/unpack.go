package ethabi

import (
	"encoding/binary"
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

var (
	big1  = big.NewInt(1)
	big32 = big.NewInt(32)
	// maxUint256 is the maximum value that can be represented by an uint256.
	maxUint256 = new(big.Int).Sub(new(big.Int).Lsh(big1, 256), big1)
)

// readBool reads a bool.
func readBool(word []byte) (bool, error) {
	for _, b := range word[:31] {
		if b != 0 {
			return false, errors.New("abi: improperly encoded boolean value")
		}
	}
	switch word[31] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, errors.New("abi: improperly encoded boolean value")
	}
}

// readInteger reads the integer based on its kind and returns the appropriate value.
func readInteger(typ Type, b []byte) DataType {
	if typ.T == UintType {
		switch typ.Size {
		case 8:
			return Int(b[len(b)-1])
		case 16:
			return Int(binary.BigEndian.Uint16(b[len(b)-2:]))
		case 32:
			return Int(binary.BigEndian.Uint32(b[len(b)-4:]))
		case 64:
			return Int(binary.BigEndian.Uint64(b[len(b)-8:]))
		default:
			// the only case left for unsigned integer is uint256.
			return BigInt{V: new(big.Int).SetBytes(b)}
		}
	}
	switch typ.Size {
	case 8:
		return Int(int8(b[len(b)-1]))
	case 16:
		return Int(int16(binary.BigEndian.Uint16(b[len(b)-2:])))
	case 32:
		return Int(int32(binary.BigEndian.Uint32(b[len(b)-4:])))
	case 64:
		return Int(int64(binary.BigEndian.Uint64(b[len(b)-8:])))
	default:
		// the only case left for integer is int256
		// big.SetBytes can't tell if a number is negative or positive in itself.
		// On EVM, if the returned number > max int256, it is negative.
		// A number is > max int256 if the bit at position 255 is set.
		ret := new(big.Int).SetBytes(b)
		if ret.Bit(255) == 1 {
			ret.Add(maxUint256, new(big.Int).Neg(ret))
			ret.Add(ret, big1)
			ret.Neg(ret)
		}
		return BigInt{V: ret}
	}
}

func tryAsInt64(dataT DataType) (int64, error) {
	switch i := dataT.(type) {
	case Int:
		return int64(i), nil
	case BigInt:
		if !i.V.IsInt64() {
			return 0, errors.New("abi: failed to convert BigInt as int64, value too big")
		}
		return i.V.Int64(), nil
	default:
		return 0, errors.Errorf("abi: failed to convert RideType as int64, type is not number")
	}
}

// forEachUnpack iteratively unpack elements.
func forEachUnpackRideList(t Type, output []byte, start, size int) (List, error) {
	if size < 0 {
		return nil, errors.Errorf("cannot marshal input to array, size is negative (%d)", size)
	}
	if start+32*size > len(output) {
		return nil, errors.Errorf(
			"abi: cannot marshal in to go array: offset %d would go over slice boundary (len=%d)",
			len(output), start+32*size,
		)
	}
	if t.T != SliceType {
		return nil, errors.Errorf("abi: invalid type in slice unpacking stage")

	}

	// this value will become our slice or our array, depending on the type
	slice := make(List, 0, size)

	// Arrays have packed elements, resulting in longer unpack steps.
	// Slices have just 32 bytes per element (pointing to the contents).
	elemSize := getTypeSize(*t.Elem)

	for i, j := start, 0; j < size; i, j = i+elemSize, j+1 {
		inter, err := toDataType(i, *t.Elem, output)
		if err != nil {
			return nil, err
		}
		slice = append(slice, inter)
	}
	return slice, nil
}

func extractIndexFromFirstElemOfTuple(index int, t Type, output []byte) (int64, error) {
	if t.T != IntType && t.T != UintType {
		return 0, errors.New(
			"abi: failed to convert eth tuple to ride union, first element of eth tuple must be a number",
		)
	}
	rideT, err := toDataType(index, t, output)
	if err != nil {
		return 0, err
	}
	return tryAsInt64(rideT)

}

func forUnionTupleUnpackToDataType(t Type, output []byte) (DataType, error) {
	if t.T != TupleType {
		return nil, errors.New("abi: type in forTupleUnpack must be TupleTy")
	}
	if len(t.TupleFields) < 2 {
		return nil, errors.New(
			"abi: failed to convert eth tuple to ride union, elements count of eth tuple must greater than 2",
		)
	}
	unionIndex, err := extractIndexFromFirstElemOfTuple(0, t.TupleFields[0].Type, output)
	if err != nil {
		return nil, err
	}
	fields := t.TupleFields[1:]
	if unionIndex >= int64(len(fields)) {
		return nil, errors.Errorf(
			"abi: failed to convert eth tuple to ride union, union index (%d) greater than tuple fields count (%d)",
			unionIndex, len(fields),
		)
	}
	retval := make([]DataType, 0, len(fields))
	virtualArgs := 0
	for index := 1; index < len(fields); index++ {
		field := fields[index]
		marshalledValue, err := toDataType((index+virtualArgs)*32, field.Type, output)
		if err != nil {
			return nil, err
		}
		if field.Type.T == TupleType && !isDynamicType(field.Type) {

			virtualArgs += getTypeSize(field.Type)/32 - 1
		}
		retval = append(retval, marshalledValue)
	}
	return retval[unionIndex], nil
}

// readFixedBytes creates a Bytes with length 1..32 to be read from.
func readFixedBytes(t Type, word []byte) (Bytes, error) {
	// type check
	if t.T != FixedBytesType {
		return nil, errors.Errorf("abi: invalid type in call to make fixed byte array")
	}
	// size check
	if t.Size < 1 || t.Size > 32 {
		return nil, errors.Errorf(
			"abi: invalid type size in call to make fixed byte array, want 0 < size <= 32, actual size=%d",
			t.Size,
		)
	}
	array := word[0:t.Size]
	return array, nil
}

type Payment struct {
	PresentAssetID bool
	AssetID        crypto.Digest
	Amount         int64
}

var (
	paymentType = Type{
		T: TupleType,
		TupleFields: Arguments{
			{Name: "id", Type: Type{T: FixedBytesType, Size: 32}},
			{Name: "value", Type: Type{T: IntType, Size: 64}},
		},
	}
	paymentsType = Type{
		Elem: &paymentType,
		T:    SliceType,
	}
	paymentsArgument = Argument{
		Name: "payments",
		Type: paymentsType,
	}
)

func unpackPayment(output []byte) (Payment, error) {
	assetIDType := paymentType.TupleFields[0].Type
	amountType := paymentType.TupleFields[1].Type

	assetRideValue, err := toDataType(0, assetIDType, output)
	if err != nil {
		return Payment{}, errors.Wrap(err, "failed to decode payment, failed to parse fullAssetID")
	}

	fullAssetIDBytes, ok := assetRideValue.(Bytes)
	if !ok {
		panic("BUG, CREATE REPORT: failed to parse payment, assetRideValue type must be RideBytes type")
	}
	fullAssetID, err := crypto.NewDigestFromBytes(fullAssetIDBytes)
	if err != nil {
		return Payment{}, errors.Wrapf(err, "abi: failed extract asset from bytes")
	}

	amountRideValue, err := toDataType(getTypeSize(assetIDType), amountType, output)
	if err != nil {
		return Payment{}, errors.Wrap(err, "failed to decode payment, failed to parse amount")
	}
	amount, err := tryAsInt64(amountRideValue)
	if err != nil {
		return Payment{}, errors.Wrapf(err, "failed to parse payment, amountRideValue type MUST be representable as int64")
	}

	payment := Payment{
		PresentAssetID: fullAssetID != crypto.Digest{}, // empty digest (32 zeroes) == WAVES asset
		AssetID:        fullAssetID,
		Amount:         amount,
	}
	return payment, nil
}

// unpackPayments unpacks payments from call data without selector
func unpackPayments(paymentsSliceOffset int, output []byte) ([]Payment, error) {
	if len(output) == 0 {
		return nil, errors.Errorf("empty payments bytes")
	}

	begin, size, err := lengthPrefixPointsTo(paymentsSliceOffset, output)
	if err != nil {
		return nil, err
	}
	// jumping to the data section
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
	bigOffsetBytes := output[index : index+32]
	bigOffsetEnd := new(big.Int).SetBytes(bigOffsetBytes)
	bigOffsetEnd.Add(bigOffsetEnd, big32)

	outputLength := new(big.Int).SetUint64(uint64(len(output)))

	if bigOffsetEnd.Cmp(outputLength) > 0 {
		return 0, 0, errors.Errorf(
			"abi: cannot marshal in to go slice: offset %v would go over slice boundary (len=%v)",
			bigOffsetEnd, outputLength,
		)
	}

	if bigOffsetEnd.BitLen() > 63 {
		return 0, 0, errors.Errorf("abi offset larger than int64: %v", bigOffsetEnd)
	}

	offsetEnd := bigOffsetEnd.Uint64()
	lengthBig := new(big.Int).SetBytes(output[offsetEnd-32 : offsetEnd])

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
	offset := new(big.Int).SetBytes(output[index : index+32])
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

func toDataType(index int, t Type, output []byte) (DataType, error) {
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
	case TupleType:
		if isDynamicType(t) {
			begin, err := tuplePointsTo(index, output)
			if err != nil {
				return nil, err
			}
			return forUnionTupleUnpackToDataType(t, output[begin:])
		}
		return forUnionTupleUnpackToDataType(t, output[index:])
	case SliceType:
		return forEachUnpackRideList(t, output[begin:], 0, length)
	case StringType: // variable arrays are written at the end of the return bytes
		return String(output[begin : begin+length]), nil
	case IntType, UintType:
		return readInteger(t, returnOutput), nil
	case BoolType:
		boolean, err := readBool(returnOutput)
		if err != nil {
			return nil, err
		}
		return Bool(boolean), nil
	case AddressType:
		if len(returnOutput) == 0 {
			return nil, errors.Errorf(
				"invalid etherum address size, expected %d, actual %d",
				EthereumAddressSize, len(returnOutput),
			)
		}
		return Bytes(returnOutput[len(returnOutput)-EthereumAddressSize:]), nil
	case BytesType:
		return Bytes(output[begin : begin+length]), nil
	case FixedBytesType:
		fixedBytes, err := readFixedBytes(t, returnOutput)
		if err != nil {
			return nil, err
		}
		return fixedBytes, err
	default:
		return nil, errors.Errorf("abi: unknown type %v", t.T)
	}
}
