package fourbyte

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"
)

var (
	errBadBool = errors.New("abi: improperly encoded boolean value")
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

// ReadInteger reads the integer based on its kind and returns the appropriate value.
func ReadInteger(typ Type, b []byte) interface{} {
	if typ.T == UintTy {
		switch typ.Size {
		case 8:
			return b[len(b)-1]
		case 16:
			return binary.BigEndian.Uint16(b[len(b)-2:])
		case 32:
			return binary.BigEndian.Uint32(b[len(b)-4:])
		case 64:
			return binary.BigEndian.Uint64(b[len(b)-8:])
		default:
			// the only case left for unsigned integer is uint256.
			return new(big.Int).SetBytes(b)
		}
	}
	switch typ.Size {
	case 8:
		return int8(b[len(b)-1])
	case 16:
		return int16(binary.BigEndian.Uint16(b[len(b)-2:]))
	case 32:
		return int32(binary.BigEndian.Uint32(b[len(b)-4:]))
	case 64:
		return int64(binary.BigEndian.Uint64(b[len(b)-8:]))
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
		return ret
	}
}

// forEachUnpack iteratively unpack elements.
func forEachUnpack(t Type, output []byte, start, size int) (interface{}, error) {
	if size < 0 {
		return nil, fmt.Errorf("cannot marshal input to array, size is negative (%d)", size)
	}
	if start+32*size > len(output) {
		return nil, fmt.Errorf(
			"abi: cannot marshal in to go array: offset %d would go over slice boundary (len=%d)",
			len(output), start+32*size,
		)
	}
	if t.T != SliceTy {
		return nil, fmt.Errorf("abi: invalid type in slice unpacking stage")

	}

	// this value will become our slice or our array, depending on the type
	refSlice := reflect.MakeSlice(t.GetType(), size, size)

	// Arrays have packed elements, resulting in longer unpack steps.
	// Slices have just 32 bytes per element (pointing to the contents).
	elemSize := getTypeSize(*t.Elem)

	for i, j := start, 0; j < size; i, j = i+elemSize, j+1 {
		inter, err := toGoType(i, *t.Elem, output)
		if err != nil {
			return nil, err
		}

		// append the item to our reflect slice
		refSlice.Index(j).Set(reflect.ValueOf(inter))
	}

	// return the interface
	return refSlice.Interface(), nil
}

func forTupleUnpack(t Type, output []byte) (interface{}, error) {
	retval := reflect.New(t.GetType()).Elem()
	virtualArgs := 0
	for index, elem := range t.TupleElems {
		marshalledValue, err := toGoType((index+virtualArgs)*32, *elem, output)
		if elem.T == TupleTy && !isDynamicType(*elem) {
			// If we have a static tuple, like (uint256, bool, uint256), these are
			// coded as just like uint256,bool,uint256
			virtualArgs += getTypeSize(*elem)/32 - 1
		}
		if err != nil {
			return nil, err
		}
		retval.Field(index).Set(reflect.ValueOf(marshalledValue))
	}
	return retval.Interface(), nil
}

// lengthPrefixPointsTo interprets a 32 byte slice as an offset and then determines which indices to look to decode the type.
func lengthPrefixPointsTo(index int, output []byte) (start int, length int, err error) {
	// TODO(nickeskov): I have no idea how it works, but we should...

	bigOffsetEnd := big.NewInt(0).SetBytes(output[index : index+32])
	bigOffsetEnd.Add(bigOffsetEnd, Big32)
	outputLength := big.NewInt(int64(len(output)))

	if bigOffsetEnd.Cmp(outputLength) > 0 {
		return 0, 0, fmt.Errorf(
			"abi: cannot marshal in to go slice: offset %v would go over slice boundary (len=%v)",
			bigOffsetEnd, outputLength,
		)
	}

	if bigOffsetEnd.BitLen() > 63 {
		return 0, 0, fmt.Errorf("abi offset larger than int64: %v", bigOffsetEnd)
	}

	offsetEnd := int(bigOffsetEnd.Uint64())
	lengthBig := big.NewInt(0).SetBytes(output[offsetEnd-32 : offsetEnd])

	totalSize := big.NewInt(0)
	totalSize.Add(totalSize, bigOffsetEnd)
	totalSize.Add(totalSize, lengthBig)
	if totalSize.BitLen() > 63 {
		return 0, 0, fmt.Errorf("abi: length larger than int64: %v", totalSize)
	}

	if totalSize.Cmp(outputLength) > 0 {
		return 0, 0, fmt.Errorf(
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
		return 0, fmt.Errorf(
			"abi: cannot marshal in to go slice: offset %v would go over slice boundary (len=%v)",
			offset, outputLen,
		)
	}
	if offset.BitLen() > 63 {
		return 0, fmt.Errorf("abi offset larger than int64: %v", offset)
	}
	return int(offset.Uint64()), nil
}
