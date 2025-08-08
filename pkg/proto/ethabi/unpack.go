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
	for _, b := range word[:abiSlotSize-1] {
		if b != 0 {
			return false, errors.New("abi: improperly encoded boolean value")
		}
	}
	switch word[abiSlotSize-1] {
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

// forEachUnpackRideList iteratively unpack elements.
func forEachUnpackRideList(t Type, output []byte, start, size int) (_ List, slotsReadTotal int, err error) {
	if t.T != SliceType { // here we can only handle slice type
		return nil, 0, errors.Errorf("abi: invalid type in slice unpacking stage")
	}
	if size < 0 { // size is positive
		return nil, 0, errors.Errorf("cannot marshal input to array, size is negative (%d)", size)
	}
	if start+abiSlotSize*size > len(output) { // check that we can read size slots
		return nil, 0, errors.Errorf(
			"abi: cannot marshal in to go array: offset %d would go over slice boundary (len=%d)",
			len(output), start+abiSlotSize*size,
		)
	}

	// this value will become our slice or our array, depending on the type
	slice := make(List, 0, size)

	// Arrays have packed elements, resulting in longer unpack steps.
	// Slices have just 32 bytes per element (pointing to the contents).
	elemSize := getTypeSize(*t.Elem)

	for i, j := start, 0; j < size; i, j = i+elemSize, j+1 {
		inter, slotsRead, err := toDataType(i, *t.Elem, output)
		if err != nil {
			return nil, 0, err
		}
		slice = append(slice, inter)
		slotsReadTotal += slotsRead
	}
	return slice, slotsReadTotal, nil
}

func extractIndexFromFirstElemOfTuple(index int, t Type, output []byte) (_ int64, slotsRead int, _ error) {
	if t.T != IntType && t.T != UintType {
		return 0, 0, errors.New(
			"abi: failed to convert eth tuple to ride union, first element of eth tuple must be a number",
		)
	}
	rideT, slotsRead, err := toDataType(index, t, output)
	if err != nil {
		return 0, 0, err
	}
	idx, err := tryAsInt64(rideT)
	if err != nil {
		return 0, 0, err
	}
	return idx, slotsRead, nil

}

func forUnionTupleUnpackToDataType(t Type, output []byte) (_ DataType, slotsReadTotal int, _ error) {
	if t.T != TupleType { // here we can only handle slice type
		return nil, 0, errors.New("abi: type in forTupleUnpack must be TupleTy")
	}
	if len(t.TupleFields) < 2 { // tuple is reasonable
		return nil, 0, errors.New(
			"abi: failed to convert eth tuple to ride union, elements count of eth tuple must greater than 2",
		)
	}
	// first slot is an index with necessary and present value
	unionIndex, slotsRead, err := extractIndexFromFirstElemOfTuple(0, t.TupleFields[0].Type, output)
	if err != nil {
		return nil, 0, err
	}
	slotsReadTotal += slotsRead

	fields := t.TupleFields[1:]           // first slot is an index, other slots are slots with field values
	if unionIndex >= int64(len(fields)) { // check that index is correct, i.e. we don't violate fields slice boundaries
		return nil, 0, errors.Errorf(
			"abi: failed to convert eth tuple to ride union, union index (%d) greater than tuple fields count (%d)",
			unionIndex, len(fields),
		)
	}
	retval := make([]DataType, 0, len(fields))
	virtualArgs := 0
	for index := 1; index < len(fields); index++ { // start with 1 because we've already read first slot
		field := fields[index]
		marshalledValue, slotsRead, err := toDataType((index+virtualArgs)*abiSlotSize, field.Type, output)
		if err != nil {
			return nil, 0, err
		}
		if field.Type.T == TupleType && !isDynamicType(field.Type) {
			virtualArgs += getTypeSize(field.Type)/abiSlotSize - 1
		}
		retval = append(retval, marshalledValue)
		slotsReadTotal += slotsRead
	}
	return retval[unionIndex], slotsReadTotal, nil
}

// readFixedBytes creates a Bytes with length 1..32 to be read from.
func readFixedBytes(t Type, word []byte) (Bytes, error) {
	// type check
	if t.T != FixedBytesType {
		return nil, errors.Errorf("abi: invalid type in call to make fixed byte array")
	}
	// size check
	if t.Size < 1 || t.Size > abiSlotSize {
		return nil, errors.Errorf(
			"abi: invalid type size in call to make fixed byte array, want 0 < size <= %d, actual size=%d",
			abiSlotSize, t.Size,
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

func unpackPayment(output []byte) (_ Payment, slotsReadTotal int, _ error) {
	assetIDType := paymentType.TupleFields[0].Type
	amountType := paymentType.TupleFields[1].Type

	assetRideValue, slotsRead, err := toDataType(0, assetIDType, output)
	if err != nil {
		return Payment{}, 0, errors.Wrap(err, "failed to decode payment, failed to parse fullAssetID")
	}
	slotsReadTotal += slotsRead

	fullAssetIDBytes, ok := assetRideValue.(Bytes)
	if !ok {
		panic("BUG, CREATE REPORT: failed to parse payment, assetRideValue type must be RideBytes type")
	}
	fullAssetID, err := crypto.NewDigestFromBytes(fullAssetIDBytes)
	if err != nil {
		return Payment{}, 0, errors.Wrapf(err, "abi: failed extract asset from bytes")
	}

	amountRideValue, slotsRead, err := toDataType(getTypeSize(assetIDType), amountType, output)
	if err != nil {
		return Payment{}, 0, errors.Wrap(err, "failed to decode payment, failed to parse amount")
	}
	slotsReadTotal += slotsRead

	amount, err := tryAsInt64(amountRideValue)
	if err != nil {
		return Payment{}, 0, errors.Wrapf(err,
			"failed to parse payment, amountRideValue type MUST be representable as int64")
	}
	if amount < 0 {
		return Payment{}, 0, errors.New("negative payment amount")
	}

	payment := Payment{
		PresentAssetID: fullAssetID != crypto.Digest{}, // empty digest (32 zeroes) == WAVES asset
		AssetID:        fullAssetID,
		Amount:         amount,
	}
	return payment, slotsReadTotal, nil
}

// unpackPayments unpacks payments from call data without selector
func unpackPayments(paymentsSliceIndex int, output []byte) (_ []Payment, slotsReadTotal int, _ error) {
	if len(output) == 0 {
		return nil, 0, errors.Errorf("empty payments bytes")
	}

	begin, size, slotsRead, err := lengthPrefixPointsTo(paymentsSliceIndex, output)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to read offset and length")
	}
	output = output[begin:]     // jumping to the data section
	slotsReadTotal += slotsRead // count offset and length slots

	if size < 0 { // size is positive
		return nil, 0, errors.Errorf("cannot marshal input to array, size is negative (%d)", size)
	}
	if abiSlotSize*size > len(output) { // check that we don't violate slice boundaries
		return nil, 0, errors.Errorf(
			"abi: cannot marshal in to go array: offset %d would go over slice boundary (len=%d)",
			len(output), abiSlotSize*size,
		)
	}

	elemSize := getTypeSize(*paymentsType.Elem) // we know that elem size here is fixed (and equal 32)
	payments := make([]Payment, 0, size)
	for i := range size {
		payment, slotsRead, err := unpackPayment(output[i*elemSize:])
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed to unpack payment")
		}
		payments = append(payments, payment)
		slotsReadTotal += slotsRead
	}
	return payments, slotsReadTotal, nil
}

// lengthPrefixPointsTo interprets a 32 byte slice as an offset and then determines which indices to look to decode the type.
func lengthPrefixPointsTo(index int, output []byte) (start, length, slotsReadTotal int, err error) {
	// read offset bytes
	bigOffsetBytes := output[index : index+abiSlotSize]
	bigOffsetEnd := new(big.Int).SetBytes(bigOffsetBytes)
	// validate offset bytes
	bigOffsetEnd.Add(bigOffsetEnd, big32)

	outputLength := new(big.Int).SetUint64(uint64(len(output)))

	if bigOffsetEnd.Cmp(outputLength) > 0 {
		return 0, 0, 0, errors.Errorf(
			"abi: cannot marshal in to go slice: offset %v would go over slice boundary (len=%v)",
			bigOffsetEnd, outputLength,
		)
	}

	if bigOffsetEnd.BitLen() > 63 {
		return 0, 0, 0, errors.Errorf("abi offset larger than int64: %v", bigOffsetEnd)
	}
	offsetEnd := bigOffsetEnd.Uint64()

	// read length
	bigLengthBytes := output[offsetEnd-abiSlotSize : offsetEnd]
	lengthBig := new(big.Int).SetBytes(bigLengthBytes)

	//validate length
	totalSize := big.NewInt(0)             // init with sero
	totalSize.Add(totalSize, bigOffsetEnd) // add offset
	totalSize.Add(totalSize, lengthBig)    // add length
	if totalSize.BitLen() > 63 {           // compare whether it's int64 or not
		return 0, 0, 0, errors.Errorf("abi: length larger than int64: %v", totalSize)
	}

	if totalSize.Cmp(outputLength) > 0 { // now compare it with output length, check bounds os slice
		return 0, 0, 0, errors.Errorf(
			"abi: cannot marshal in to go type: length insufficient %v require %v",
			outputLength, totalSize,
		)
	}
	start = int(bigOffsetEnd.Uint64()) // count first slot
	length = int(lengthBig.Uint64())   // count second slot
	return start, length, 2, nil
}

// tuplePointsTo resolves the location reference for dynamic tuple.
func tuplePointsTo(index int, output []byte) (start int, slotsReadTotal int, err error) {
	offset := new(big.Int).SetBytes(output[index : index+abiSlotSize]) // read exactly one slot
	outputLen := big.NewInt(int64(len(output)))

	if offset.Cmp(outputLen) > 0 {
		return 0, 0, errors.Errorf(
			"abi: cannot marshal in to go slice: offset %v would go over slice boundary (len=%v)",
			offset, outputLen,
		)
	}
	if offset.BitLen() > 63 {
		return 0, 0, errors.Errorf("abi offset larger than int64: %v", offset)
	}
	return int(offset.Uint64()), 1, nil
}

func slotsSizeForBytes(b []byte) int {
	l := len(b)
	size := l / abiSlotSize
	if l%abiSlotSize != 0 { // doesn't fit in slots
		size += 1 // add slot
	}
	return size // slots size for data
}

func toDataType(index int, t Type, output []byte) (_ DataType, slotsReadTotal int, _ error) {
	if l := len(output); index+abiSlotSize > l { // check that we can read at least one slot
		return nil, 0, errors.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d",
			l, index+abiSlotSize,
		)
	}

	switch t.T {
	case TupleType:
		if isDynamicType(t) {
			begin, slotsRead, err := tuplePointsTo(index, output) // read offset for dynamic tuple
			if err != nil {
				return nil, 0, err
			}
			slotsReadTotal += slotsRead // count offset slot
			union, slotsRead, err := forUnionTupleUnpackToDataType(t, output[begin:])
			if err != nil {
				return nil, 0, err
			}
			slotsReadTotal += slotsRead // count data slots
			return union, slotsReadTotal, nil
		}
		return forUnionTupleUnpackToDataType(t, output[index:])
	case SliceType:
		begin, length, slotsRead, err := lengthPrefixPointsTo(index, output)
		if err != nil {
			return nil, 0, err
		}
		slotsReadTotal += slotsRead // count offset and length slots
		list, slotsRead, err := forEachUnpackRideList(t, output[begin:], 0, length)
		if err != nil {
			return nil, 0, err
		}
		slotsReadTotal += slotsRead // count data slots
		return list, slotsReadTotal, err
	case StringType: // variable arrays are written at the end of the return bytes
		begin, length, slotsRead, err := lengthPrefixPointsTo(index, output)
		if err != nil {
			return nil, 0, err
		}
		slotsReadTotal += slotsRead // count offset and length slots
		s := output[begin : begin+length]
		slotsReadTotal += slotsSizeForBytes(s) // count data slots
		return String(s), slotsReadTotal, nil
	case BytesType:
		begin, length, slotsRead, err := lengthPrefixPointsTo(index, output)
		if err != nil {
			return nil, 0, err
		}
		slotsReadTotal += slotsRead // count offset and length slots
		s := output[begin : begin+length]
		slotsReadTotal += slotsSizeForBytes(s) // count data slots
		return Bytes(output[begin : begin+length]), slotsReadTotal, nil
	case IntType, UintType:
		slot := output[index : index+abiSlotSize] // read exactly one slot for simple types
		slotsReadTotal += 1
		return readInteger(t, slot), slotsReadTotal, nil
	case BoolType:
		slot := output[index : index+abiSlotSize] // read exactly one slot for simple types
		slotsReadTotal += 1
		boolean, err := readBool(slot)
		if err != nil {
			return nil, 0, err
		}
		return Bool(boolean), slotsReadTotal, nil
	case AddressType:
		slot := output[index : index+abiSlotSize] // read exactly one slot for simple types
		slotsReadTotal += 1
		if len(slot) == 0 {
			return nil, 0, errors.Errorf(
				"invalid ethereum address size, expected %d, actual %d",
				EthereumAddressSize, len(slot),
			)
		}
		return Bytes(slot[len(slot)-EthereumAddressSize:]), slotsReadTotal, nil
	case FixedBytesType:
		slot := output[index : index+abiSlotSize] // read exactly one slot for simple types
		slotsReadTotal += 1
		fixedBytes, err := readFixedBytes(t, slot)
		if err != nil {
			return nil, 0, err
		}
		return fixedBytes, slotsReadTotal, err
	default:
		return nil, 0, errors.Errorf("abi: unknown type %v", t.T)
	}
}
