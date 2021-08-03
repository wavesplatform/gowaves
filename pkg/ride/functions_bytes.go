package ride

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"unicode/utf8"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
)

const maxBytesLength = 65536

func bytesArg(args []RideType) (RideBytes, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return nil, errors.Errorf("argument 1 is empty")
	}
	b, ok := args[0].(RideBytes)
	if !ok {
		return nil, errors.Errorf("argument 1 is not of type 'ByteVector' but '%s'", args[0].instanceOf())
	}
	return b, nil
}

func bytesAndIntArgs(args []RideType) ([]byte, int, error) {
	if len(args) != 2 {
		return nil, 0, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, 0, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, 0, errors.Errorf("argument 2 is empty")
	}
	b, ok := args[0].(RideBytes)
	if !ok {
		return nil, 0, errors.Errorf("argument 1 is not of type 'ByteVector' but '%s'", args[0].instanceOf())
	}
	i, ok := args[1].(RideInt)
	if !ok {
		return nil, 0, errors.Errorf("argument 2 is not of type 'Int' but '%s'", args[1].instanceOf())
	}
	return b, int(i), nil
}

func bytesArgs2(args []RideType) (RideBytes, RideBytes, error) {
	if len(args) != 2 {
		return nil, nil, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, nil, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, nil, errors.Errorf("argument 2 is empty")
	}
	b1, ok := args[0].(RideBytes)
	if !ok {
		return nil, nil, errors.Errorf("argument 1 is not of type 'ByteVector' but '%s'", args[0].instanceOf())
	}
	b2, ok := args[1].(RideBytes)
	if !ok {
		return nil, nil, errors.Errorf("argument 2 is not of type 'ByteVector' but '%s'", args[1].instanceOf())
	}
	return b1, b2, nil
}

func bytesOrUnitArgAsBytes(args ...RideType) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return nil, errors.Errorf("argument is empty")
	}
	switch arg := args[0].(type) {
	case RideBytes:
		return arg, nil
	case rideUnit:
		return nil, nil
	default:
		return nil, errors.Errorf("toBase58: unexpected argument type '%s'", args[0].instanceOf())
	}
}

func sizeBytes(_ Environment, args ...RideType) (RideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "sizeBytes")
	}
	return RideInt(len(b)), nil
}

func takeBytes(_ Environment, args ...RideType) (RideType, error) {
	b, n, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "takeBytes")
	}
	return takeRideBytes(b, n), nil
}

func dropBytes(_ Environment, args ...RideType) (RideType, error) {
	b, n, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "dropBytes")
	}
	return dropRideBytes(b, n), nil
}

func concatBytes(_ Environment, args ...RideType) (RideType, error) {
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
	copy(out[len(b1):], b2)
	return RideBytes(out), nil
}

func toBase58(_ Environment, args ...RideType) (RideType, error) {
	b, err := bytesOrUnitArgAsBytes(args...)
	if err != nil {
		return nil, errors.Wrap(err, "toBase58")
	}
	return RideString(base58.Encode(b)), nil
}

func fromBase58(_ Environment, args ...RideType) (RideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase58")
	}
	str := string(s)
	if str == "" {
		return RideBytes{}, nil
	}
	r, err := base58.Decode(str)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase58")
	}
	return RideBytes(r), nil
}

func toBase64(_ Environment, args ...RideType) (RideType, error) {
	b, err := bytesOrUnitArgAsBytes(args...)
	if err != nil {
		return nil, errors.Wrap(err, "toBase64")
	}
	return RideString(base64.StdEncoding.EncodeToString(b)), nil
}

func fromBase64(_ Environment, args ...RideType) (RideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase64")
	}
	str := strings.TrimPrefix(string(s), "base64:")
	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(str) // Try no padding.
		if err != nil {
			return nil, errors.Wrap(err, "fromBase64")
		}
		return RideBytes(decoded), nil
	}
	return RideBytes(decoded), nil
}

func toBase16(_ Environment, args ...RideType) (RideType, error) {
	b, err := bytesOrUnitArgAsBytes(args...)
	if err != nil {
		return nil, errors.Wrap(err, "toBase16")
	}
	return RideString(hex.EncodeToString(b)), nil
}

func fromBase16(_ Environment, args ...RideType) (RideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase16")
	}
	str := strings.TrimPrefix(string(s), "base16:")
	decoded, err := hex.DecodeString(str)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase16")
	}
	return RideBytes(decoded), nil
}

func dropRightBytes(_ Environment, args ...RideType) (RideType, error) {
	b, n, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "dropRightBytes")
	}
	return takeRideBytes(b, len(b)-n), nil
}

func takeRightBytes(_ Environment, args ...RideType) (RideType, error) {
	b, n, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "takeRightBytes")
	}
	return dropRideBytes(b, len(b)-n), nil
}

func bytesToUTF8String(_ Environment, args ...RideType) (RideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesToUTF8String")
	}
	if s := string(b); utf8.ValidString(s) {
		return RideString(s), nil
	}
	return nil, errors.Errorf("invalid UTF-8 sequence")
}

func bytesToInt(_ Environment, args ...RideType) (RideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesToInt")
	}
	if l := len(b); l < 8 {
		return nil, errors.Errorf("bytesToInt: %d is too little bytes to make int value", l)
	}
	return RideInt(binary.BigEndian.Uint64(b)), nil
}

func bytesToIntWithOffset(_ Environment, args ...RideType) (RideType, error) {
	b, n, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesToLongWithOffset")
	}
	if n < 0 || n > len(b)-8 {
		return nil, errors.Errorf("bytesToLongWithOffset: offset %d is out of bytes array bounds", n)
	}
	return RideInt(binary.BigEndian.Uint64(b[n:])), nil
}

func takeRideBytes(b []byte, n int) RideBytes {
	l := n
	if bl := len(b); l > bl {
		l = bl
	}
	if l < 0 {
		l = 0
	}
	r := make(RideBytes, l)
	copy(r, b[:l])
	return r
}

func dropRideBytes(b []byte, n int) RideBytes {
	l := n
	bl := len(b)
	if l > bl {
		l = bl
	}
	if l < 0 {
		l = 0
	}
	r := make(RideBytes, bl-l)
	copy(r, b[l:])
	return r
}
