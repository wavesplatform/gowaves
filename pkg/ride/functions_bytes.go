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

func bytesAndIntArgs(args []rideType) ([]byte, int, error) {
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
	return b, int(i), nil
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

func bytesOrUnitArgAsBytes(args ...rideType) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return nil, errors.Errorf("argument is empty")
	}
	switch arg := args[0].(type) {
	case rideBytes:
		return arg, nil
	case rideUnit:
		return nil, nil
	default:
		return nil, errors.Errorf("toBase58: unexpected argument type '%s'", args[0].instanceOf())
	}
}

func sizeBytes(_ environment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "sizeBytes")
	}
	return rideInt(len(b)), nil
}

func takeBytes(_ environment, args ...rideType) (rideType, error) {
	b, n, err := bytesAndIntArgs(args)
	//TODO: `takeBytes` function must check `n` is less or equal 165947 (DataTxMaxProtoBytes) after activation of RideV6
	if err != nil {
		return nil, errors.Wrap(err, "takeBytes")
	}
	return takeRideBytes(b, n), nil
}

func dropBytes(_ environment, args ...rideType) (rideType, error) {
	b, n, err := bytesAndIntArgs(args)
	//TODO: `dropBytes` function must check `n` is less or equal 165947 (DataTxMaxProtoBytes) after activation of RideV6
	if err != nil {
		return nil, errors.Wrap(err, "dropBytes")
	}
	return dropRideBytes(b, n), nil
}

func concatBytes(env environment, args ...rideType) (rideType, error) {
	b1, b2, err := bytesArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "concatBytes")
	}
	l := len(b1) + len(b2)
	if env == nil {
		return nil, errors.New("concatBytes: empty environment")
	}
	if !env.checkMessageLength(l) {
		return nil, errors.Errorf("concatBytes: invalid result length %d", l)
	}
	out := make([]byte, l)
	copy(out, b1)
	copy(out[len(b1):], b2)
	return rideBytes(out), nil
}

func toBase58(_ environment, args ...rideType) (rideType, error) {
	//TODO: Before activation of RideV4 length of the result string must not be more than 165947 (DataTxMaxProtoBytes) bytes,
	// after no more than 32768 bytes.
	b, err := bytesOrUnitArgAsBytes(args...)
	if err != nil {
		return nil, errors.Wrap(err, "toBase58")
	}
	return rideString(base58.Encode(b)), nil
}

func fromBase58(_ environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase58")
	}
	str := string(s)
	if str == "" {
		return rideBytes{}, nil
	}
	r, err := base58.Decode(str)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase58")
	}
	return rideBytes(r), nil
}

func toBase64(_ environment, args ...rideType) (rideType, error) {
	//TODO: Before activation of RideV4 length of the result string must not be more than 165947 (DataTxMaxProtoBytes) bytes,
	// after no more than 32768 bytes.
	b, err := bytesOrUnitArgAsBytes(args...)
	if err != nil {
		return nil, errors.Wrap(err, "toBase64")
	}
	return rideString(base64.StdEncoding.EncodeToString(b)), nil
}

func fromBase64(_ environment, args ...rideType) (rideType, error) {
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
		return rideBytes(decoded), nil
	}
	return rideBytes(decoded), nil
}

func toBase16(_ environment, args ...rideType) (rideType, error) {
	b, err := bytesOrUnitArgAsBytes(args...)
	if err != nil {
		return nil, errors.Wrap(err, "toBase16")
	}
	return rideString(hex.EncodeToString(b)), nil
}

func fromBase16(_ environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase16")
	}
	str := strings.TrimPrefix(string(s), "base16:")
	decoded, err := hex.DecodeString(str)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase16")
	}
	return rideBytes(decoded), nil
}

func dropRightBytes(_ environment, args ...rideType) (rideType, error) {
	b, n, err := bytesAndIntArgs(args)
	//TODO: `dropRightBytes` function must check `n` is less or equal 165947 (DataTxMaxProtoBytes) after activation of RideV6
	if err != nil {
		return nil, errors.Wrap(err, "dropRightBytes")
	}
	return takeRideBytes(b, len(b)-n), nil
}

func takeRightBytes(_ environment, args ...rideType) (rideType, error) {
	b, n, err := bytesAndIntArgs(args)
	//TODO: `takeRightBytes` function must check `n` is less or equal 165947 (DataTxMaxProtoBytes) after activation of RideV6
	if err != nil {
		return nil, errors.Wrap(err, "takeRightBytes")
	}
	return dropRideBytes(b, len(b)-n), nil
}

func bytesToUTF8String(_ environment, args ...rideType) (rideType, error) {
	//TODO: Before activation of RideV4 length of the result string must not be more than 165947 (DataTxMaxProtoBytes) bytes,
	// after no more than 32768 bytes.
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesToUTF8String")
	}
	if s := string(b); utf8.ValidString(s) {
		return rideString(s), nil
	}
	return nil, errors.Errorf("invalid UTF-8 sequence")
}

func bytesToInt(_ environment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesToInt")
	}
	if l := len(b); l < 8 {
		return nil, errors.Errorf("bytesToInt: %d is too little bytes to make int value", l)
	}
	return rideInt(binary.BigEndian.Uint64(b)), nil
}

func bytesToIntWithOffset(_ environment, args ...rideType) (rideType, error) {
	b, n, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesToLongWithOffset")
	}
	if n < 0 || n > len(b)-8 {
		return nil, errors.Errorf("bytesToLongWithOffset: offset %d is out of bytes array bounds", n)
	}
	return rideInt(binary.BigEndian.Uint64(b[n:])), nil
}

func takeRideBytes(b []byte, n int) rideBytes {
	l := n
	if bl := len(b); l > bl {
		l = bl
	}
	if l < 0 {
		l = 0
	}
	r := make(rideBytes, l)
	copy(r, b[:l])
	return r
}

func dropRideBytes(b []byte, n int) rideBytes {
	l := n
	bl := len(b)
	if l > bl {
		l = bl
	}
	if l < 0 {
		l = 0
	}
	r := make(rideBytes, bl-l)
	copy(r, b[l:])
	return r
}
