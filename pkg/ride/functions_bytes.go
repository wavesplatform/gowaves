package ride

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"unicode/utf8"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	maxBase64StringToDecode = 44 * 1024 // 44 KiB
	maxBase58StringToDecode = 100
	maxBase16StringToDecode = 32 * 1024 // 32 KiB
)
const (
	maxBase64BytesToEncode = 32 * 1024 // 32 KiB
	maxBase58BytesToEncode = 64
	maxBase16BytesToEncode = 8 * 1024 // 8 KiB
)

// dataTxMaxProtoBytes depends on DataTransaction.MaxProtoBytes.
// But it SHOULD be equal proto.MaxDataWithProofsProtoBytes. But for unknown reason, it is not.
const dataTxMaxProtoBytes = 165947

func bytesArg(args []rideType) (rideByteVector, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return nil, errors.Errorf("argument 1 is empty")
	}
	b, ok := args[0].(rideByteVector)
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
	b, ok := args[0].(rideByteVector)
	if !ok {
		return nil, 0, errors.Errorf("argument 1 is not of type 'ByteVector' but '%s'", args[0].instanceOf())
	}
	i, ok := args[1].(rideInt)
	if !ok {
		return nil, 0, errors.Errorf("argument 2 is not of type 'Int' but '%s'", args[1].instanceOf())
	}
	return b, int(i), nil
}

func bytesArgs2(args []rideType) (rideByteVector, rideByteVector, error) {
	if len(args) != 2 {
		return nil, nil, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, nil, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, nil, errors.Errorf("argument 2 is empty")
	}
	b1, ok := args[0].(rideByteVector)
	if !ok {
		return nil, nil, errors.Errorf("argument 1 is not of type 'ByteVector' but '%s'", args[0].instanceOf())
	}
	b2, ok := args[1].(rideByteVector)
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
	case rideByteVector:
		return arg, nil
	case rideUnit:
		return nil, nil
	default:
		return nil, errors.Errorf("unexpected argument type '%s'", args[0].instanceOf())
	}
}

func sizeBytes(_ environment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "sizeBytes")
	}
	return rideInt(len(b)), nil
}

func checkBytesNumberLimit(checkLimits bool, n int, fName, rideFName string) error {
	return checkTakeDropNumberLimit("ByteVector", dataTxMaxProtoBytes, checkLimits, n, fName, rideFName)
}

func takeBytesGeneric(checkLimits bool, args ...rideType) (rideType, error) {
	b, n, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "takeBytes")
	}
	if lErr := checkBytesNumberLimit(checkLimits, n, "takeBytes", "take"); lErr != nil {
		return nil, lErr
	}
	return takeRideBytes(b, n), nil
}

func takeBytes(_ environment, args ...rideType) (rideType, error) {
	return takeBytesGeneric(false, args...)
}

func takeBytesV6(_ environment, args ...rideType) (rideType, error) {
	return takeBytesGeneric(true, args...)
}

func dropBytesGeneric(checkLimits bool, args ...rideType) (rideType, error) {
	b, n, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "dropBytes")
	}
	if lErr := checkBytesNumberLimit(checkLimits, n, "dropBytes", "drop"); lErr != nil {
		return nil, lErr
	}
	return dropRideBytes(b, n), nil
}

func dropBytes(_ environment, args ...rideType) (rideType, error) {
	return dropBytesGeneric(false, args...)
}

func dropBytesV6(_ environment, args ...rideType) (rideType, error) {
	return dropBytesGeneric(true, args...)
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
	return rideByteVector(out), nil
}

func checkByteStringLength(reduceLimit bool, s string) error {
	limit := proto.MaxDataWithProofsBytes
	if reduceLimit {
		limit = proto.MaxDataEntryValueSize
	}
	if size := len(s); size > limit { // utf8 bytes length
		return RuntimeError.Errorf("string size=%d exceeds %d bytes", size, limit)
	}
	return nil
}

func toBase58Generic(reduceLimit bool, args ...rideType) (rideType, error) {
	b, err := bytesOrUnitArgAsBytes(args...)
	if err != nil {
		return nil, errors.Wrap(err, "toBase58")
	}
	if l := len(b); l > maxBase58BytesToEncode {
		return nil, RuntimeError.Errorf("toBase58: input is too long (%d), limit is %d", l, maxBase58BytesToEncode)
	}
	s := base58.Encode(b)
	if lErr := checkByteStringLength(reduceLimit, s); lErr != nil {
		return nil, errors.Wrap(lErr, "toBase58")
	}
	return rideString(s), nil
}

func toBase58(_ environment, args ...rideType) (rideType, error) {
	return toBase58Generic(false, args...)
}

func toBase58V4(_ environment, args ...rideType) (rideType, error) {
	return toBase58Generic(true, args...)
}

func fromBase58(_ environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase58")
	}
	if l := len(s); l > maxBase58StringToDecode {
		return nil, RuntimeError.Errorf("fromBase58: input is too long (%d), limit is %d", l, maxBase58StringToDecode)
	}
	str := string(s)
	if str == "" {
		return rideByteVector{}, nil
	}
	r, err := base58.Decode(str)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase58")
	}
	return rideByteVector(r), nil
}

func toBase64Generic(reduceLimit bool, args ...rideType) (rideType, error) {
	b, err := bytesOrUnitArgAsBytes(args...)
	if err != nil {
		return nil, errors.Wrap(err, "toBase64")
	}
	if l := len(b); l > maxBase64BytesToEncode {
		return nil, RuntimeError.Errorf("toBase64: input is too long (%d), limit is %d", l, maxBase64BytesToEncode)
	}
	s := base64.StdEncoding.EncodeToString(b)
	if lErr := checkByteStringLength(reduceLimit, s); lErr != nil {
		return nil, errors.Wrap(lErr, "toBase64")
	}
	return rideString(s), nil
}

func toBase64(_ environment, args ...rideType) (rideType, error) {
	return toBase64Generic(false, args...)
}

func toBase64V4(_ environment, args ...rideType) (rideType, error) {
	return toBase64Generic(true, args...)
}

func fromBase64(_ environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase64")
	}
	if l := len(s); l > maxBase64StringToDecode {
		return nil, RuntimeError.Errorf("fromBase64: input is too long (%d), limit is %d", l, maxBase64StringToDecode)
	}
	str := strings.TrimPrefix(string(s), "base64:")
	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(str) // Try no padding.
		if err != nil {
			return nil, errors.Wrap(err, "fromBase64")
		}
		return rideByteVector(decoded), nil
	}
	return rideByteVector(decoded), nil
}

func toBase16Generic(checkLength bool, args ...rideType) (rideType, error) {
	b, err := bytesOrUnitArgAsBytes(args...)
	if err != nil {
		return nil, errors.Wrap(err, "toBase16")
	}
	if l := len(b); checkLength && l > maxBase16BytesToEncode {
		return nil, RuntimeError.Errorf("toBase16: input is too long (%d), limit is %d", l, maxBase16BytesToEncode)
	}
	s := hex.EncodeToString(b)
	if lErr := checkByteStringLength(true, s); lErr != nil {
		return nil, errors.Wrap(lErr, "toBase16")
	}
	return rideString(s), nil
}

func toBase16(_ environment, args ...rideType) (rideType, error) {
	return toBase16Generic(false, args...)
}

func toBase16V4(_ environment, args ...rideType) (rideType, error) {
	return toBase16Generic(true, args...)
}

func fromBase16Generic(checkLength bool, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase16")
	}
	if l := len(s); checkLength && l > maxBase16StringToDecode {
		return nil, RuntimeError.Errorf("fromBase16: input is too long (%d), limit is %d", l, maxBase16StringToDecode)
	}
	str := strings.TrimPrefix(string(s), "base16:")
	decoded, err := hex.DecodeString(str)
	if err != nil {
		return nil, errors.Wrap(err, "fromBase16")
	}
	return rideByteVector(decoded), nil
}

func fromBase16(_ environment, args ...rideType) (rideType, error) {
	return fromBase16Generic(false, args...)
}

func fromBase16V4(_ environment, args ...rideType) (rideType, error) {
	return fromBase16Generic(true, args...)
}

func dropRightBytesGeneric(checkLimits bool, args ...rideType) (rideType, error) {
	b, n, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "dropRightBytes")
	}
	if lErr := checkBytesNumberLimit(checkLimits, n, "dropRightBytes", "dropRight"); lErr != nil {
		return nil, lErr
	}
	return takeRideBytes(b, len(b)-n), nil
}

func dropRightBytes(_ environment, args ...rideType) (rideType, error) {
	return dropRightBytesGeneric(false, args...)
}

func dropRightBytesV6(_ environment, args ...rideType) (rideType, error) {
	return dropRightBytesGeneric(true, args...)
}

func takeRightBytesGeneric(checkLimits bool, args ...rideType) (rideType, error) {
	b, n, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "takeRightBytes")
	}
	if lErr := checkBytesNumberLimit(checkLimits, n, "takeRightBytes", "takeRight"); lErr != nil {
		return nil, lErr
	}
	return dropRideBytes(b, len(b)-n), nil
}

func takeRightBytes(_ environment, args ...rideType) (rideType, error) {
	return takeRightBytesGeneric(false, args...)
}

func takeRightBytesV6(_ environment, args ...rideType) (rideType, error) {
	return takeRightBytesGeneric(true, args...)
}

func bytesToUTF8StringGeneric(reduceLimit bool, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesToUTF8String")
	}
	s := string(b)
	if !utf8.ValidString(s) {
		return nil, errors.Errorf("invalid UTF-8 sequence")
	}
	if lErr := checkByteStringLength(reduceLimit, s); lErr != nil {
		return nil, errors.Wrap(lErr, "bytesToUTF8String")
	}
	return rideString(s), nil
}

func bytesToUTF8String(_ environment, args ...rideType) (rideType, error) {
	return bytesToUTF8StringGeneric(false, args...)
}

func bytesToUTF8StringV4(_ environment, args ...rideType) (rideType, error) {
	return bytesToUTF8StringGeneric(true, args...)
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

func takeRideBytes(b []byte, n int) rideByteVector {
	l := n
	if bl := len(b); l > bl {
		l = bl
	}
	if l < 0 {
		l = 0
	}
	r := make(rideByteVector, l)
	copy(r, b[:l])
	return r
}

func dropRideBytes(b []byte, n int) rideByteVector {
	l := n
	bl := len(b)
	if l > bl {
		l = bl
	}
	if l < 0 {
		l = 0
	}
	r := make(rideByteVector, bl-l)
	copy(r, b[l:])
	return r
}
