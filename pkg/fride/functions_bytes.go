package fride

import (
	"github.com/pkg/errors"
)

const maxBytesLength = 65536

func bytesArg(args []rideType) (rideBytes, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return nil, errors.Errorf("argument 1 is empty")
	}
	b, ok := args[0].(rideBytes)
	if !ok {
		return nil, errors.Errorf("argument 1 is not of type 'ByteVector' but '%s'", args[0].instanceOf())
	}
	return b, nil
}

func bytesAndIntArgs(args []rideType) (rideBytes, rideInt, error) {
	if len(args) != 2 {
		return nil, 0, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, 0, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, 0, errors.Errorf("argument 2 is empty")
	}
	b, ok := args[0].(rideBytes)
	if !ok {
		return nil, 0, errors.Errorf("argument 1 is not of type 'ByteVector' but '%s'", args[0].instanceOf())
	}
	i, ok := args[1].(rideInt)
	if !ok {
		return nil, 0, errors.Errorf("argument 2 is not of type 'Int' but '%s'", args[1].instanceOf())
	}
	return b, i, nil
}

func bytesArgs2(args []rideType) (rideBytes, rideBytes, error) {
	if len(args) != 2 {
		return nil, nil, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, nil, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, nil, errors.Errorf("argument 2 is empty")
	}
	b1, ok := args[0].(rideBytes)
	if !ok {
		return nil, nil, errors.Errorf("argument 1 is not of type 'ByteVector' but '%s'", args[0].instanceOf())
	}
	b2, ok := args[1].(rideBytes)
	if !ok {
		return nil, nil, errors.Errorf("argument 2 is not of type 'ByteVector' but '%s'", args[1].instanceOf())
	}
	return b1, b2, nil
}

func sizeBytes(args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "sizeBytes")
	}
	return rideInt(len(b)), nil
}

func takeBytes(args ...rideType) (rideType, error) {
	b, i, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "takeBytes")
	}
	l := int(i)
	if bl := len(b); l > bl {
		l = bl
	}
	if l < 0 {
		l = 0
	}
	out := make([]byte, l)
	copy(out, b[:l])
	return rideBytes(out), nil
}

func dropBytes(args ...rideType) (rideType, error) {
	b, i, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "dropBytes")
	}
	l := int(i)
	bl := len(b)
	if l > bl {
		l = bl
	}
	if l < 0 {
		l = 0
	}
	out := make([]byte, bl-l)
	copy(out, b[l:])
	return rideBytes(out), nil
}

func concatBytes(args ...rideType) (rideType, error) {
	b1, b2, err := bytesArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "concatBytes")
	}
	l := len(b1) + len(b2)
	if l > maxBytesLength {
		return nil, errors.Errorf("concatBytes: length of result (%d) is greater than allowed (%d)", l, maxBytesLength)
	}
	out := make([]byte, l)
	copy(out, b1)
	copy(out[:len(b1)], b2)
	return rideBytes(out), nil
}

func toBase58(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func fromBase58(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func toBase64(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func fromBase64(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func toBase16(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func fromBase16(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func dropRightBytes(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func takeRightBytes(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func bytesToUTF8String(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func bytesToLong(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func bytesToLongWithOffset(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}
