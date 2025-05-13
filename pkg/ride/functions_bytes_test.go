package ride

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestSizeBytes(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector{0, 0, 0, 0, 0, 0, 0, 0}}, false, rideInt(8)},
		{[]rideType{rideByteVector{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, false, rideInt(8)},
		{[]rideType{rideByteVector{0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5}}, false, rideInt(12)},
		{[]rideType{rideByteVector{}}, false, rideInt(0)},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := sizeBytes(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestTakeBytes(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector{1, 2, 3}, rideInt(2)}, false, rideByteVector{1, 2}},
		{[]rideType{rideByteVector{1, 2, 3}, rideInt(4)}, false, rideByteVector{1, 2, 3}},
		{[]rideType{rideByteVector{1, 2, 3}, rideInt(0)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{1, 2, 3}, rideInt(-4)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{}, rideInt(0)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{}, rideInt(3)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := takeBytes(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestDropBytes(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(2)}, false, rideByteVector{3, 4, 5}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(4)}, false, rideByteVector{5}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(5)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(8)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(0)}, false, rideByteVector{1, 2, 3, 4, 5}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(-4)}, false, rideByteVector{1, 2, 3, 4, 5}},
		{[]rideType{rideByteVector{}, rideInt(0)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{}, rideInt(3)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := dropBytes(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestConcatBytes(t *testing.T) {
	te := &mockRideEnvironment{checkMessageLengthFunc: bytesSizeCheckV1V2}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector{1, 2, 3}, rideByteVector{4, 5}}, false, rideByteVector{1, 2, 3, 4, 5}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideByteVector{6}}, false, rideByteVector{1, 2, 3, 4, 5, 6}},
		{[]rideType{rideByteVector{1, 2, 3}, rideByteVector{}}, false, rideByteVector{1, 2, 3}},
		{[]rideType{rideByteVector{}, rideByteVector{1, 2, 3}}, false, rideByteVector{1, 2, 3}},
		{[]rideType{rideByteVector{}, rideByteVector{}}, false, rideByteVector{}},
		{[]rideType{rideByteVector{1, 2, 3}}, true, nil},
		{[]rideType{rideByteVector{1, 2, 4}, rideInt(0)}, true, nil},
		{[]rideType{rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := concatBytes(te, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestToBase58Generic(t *testing.T) {
	var (
		maxDataWithProofsBytesBV      = make([]byte, proto.MaxDataWithProofsBytes+1)
		maxDataWithProofsBytesBVLower = maxDataWithProofsBytesBV[:proto.MaxDataWithProofsBytes]
	)
	var (
		maxDataEntryValueSizeBV      = maxDataWithProofsBytesBV[:proto.MaxDataEntryValueSize+1]
		maxDataEntryValueSizeBVLower = maxDataEntryValueSizeBV[:proto.MaxDataEntryValueSize]
	)
	var (
		overMaxBase58BytesSize = make([]byte, maxBase58BytesToEncode+1)
		maxBase58BytesSize     = overMaxBase58BytesSize[:maxBase58BytesToEncode]
		maxBase58BytesSizeRes  = base58.Encode(maxBase58BytesSize)
	)
	for i, test := range []struct {
		reduceLimit bool
		args        []rideType
		fail        bool
		r           rideType
	}{
		{false, []rideType{rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, rideString("6gVbAXCUdsa14xdsSk2SKaNBXs271V3Mo4zjb2cvCrsM")}, //nolint:lll
		{false, []rideType{rideByteVector{0, 0, 0, 0, 0}}, false, rideString("11111")},
		{false, []rideType{rideByteVector{}}, false, rideString("")},
		{false, []rideType{rideUnit{}}, false, rideString("")},
		{false, []rideType{rideByteVector{}, rideByteVector{}}, true, nil},
		{false, []rideType{rideByteVector{1, 2, 4}, rideInt(0)}, true, nil},
		{false, []rideType{rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}}, true, nil},
		{false, []rideType{rideInt(1), rideString("x")}, true, nil},
		{false, []rideType{}, true, nil},
		//
		{false, []rideType{rideByteVector(maxDataWithProofsBytesBV)}, true, nil},
		{false, []rideType{rideByteVector(maxDataWithProofsBytesBVLower)}, true, nil},
		//
		{true, []rideType{rideByteVector(maxDataEntryValueSizeBV)}, true, nil},
		{true, []rideType{rideByteVector(maxDataEntryValueSizeBVLower)}, true, nil},
		//
		{false, []rideType{rideByteVector(overMaxBase58BytesSize)}, true, nil},
		{true, []rideType{rideByteVector(overMaxBase58BytesSize)}, true, nil},
		//
		{false, []rideType{rideByteVector(maxBase58BytesSize)}, false, rideString(maxBase58BytesSizeRes)},
		{true, []rideType{rideByteVector(maxBase58BytesSize)}, false, rideString(maxBase58BytesSizeRes)},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			r, err := toBase58Generic(test.reduceLimit, test.args...)
			if test.fail {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.r, r)
			}
		})
	}
}

func TestFromBase58(t *testing.T) {
	var (
		overMaxInputRes = make([]byte, maxBase58StringToDecode+1)
		overMaxInput    = base58.Encode(overMaxInputRes)
	)
	var (
		maxInputRes = overMaxInputRes[:maxBase58StringToDecode]
		maxInput    = base58.Encode(maxInputRes)
	)
	for i, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("6gVbAXCUdsa14xdsSk2SKaNBXs271V3Mo4zjb2cvCrsM")}, false, rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]rideType{rideString("11111")}, false, rideByteVector{0, 0, 0, 0, 0}},
		{[]rideType{rideString("")}, false, rideByteVector{}},
		{[]rideType{rideString(""), rideString("")}, true, nil},
		{[]rideType{rideByteVector{1, 2, 4}}, true, nil},
		{[]rideType{rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideString(overMaxInput)}, true, nil},
		{[]rideType{rideString(maxInput)}, false, rideByteVector(maxInputRes)},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			r, err := fromBase58(nil, test.args...)
			if test.fail {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.r, r)
			}
		})
	}
}

func TestToBase64Generic(t *testing.T) {
	const (
		// base64 approximately expands to 4/3 of the original size
		// divide by 4/3
		maxDataWithProofsBytesB64 = proto.MaxDataWithProofsBytes * 3 / 4 // gives 153_600 bytes in base64
		maxDataEntryValueSizeB64  = proto.MaxDataEntryValueSize*3/4 - 2  // gives 32_764 bytes in base64
	)
	var (
		maxDataWithProofsBytesBV   = make([]byte, maxDataWithProofsBytesB64+1)
		maxDataWithProofsBytesBVOK = maxDataWithProofsBytesBV[:maxDataWithProofsBytesB64]
	)
	var (
		maxDataEntryValueSizeBV      = maxDataWithProofsBytesBV[:maxDataEntryValueSizeB64+1]
		maxDataEntryValueSizeBVOK    = maxDataEntryValueSizeBV[:maxDataEntryValueSizeB64]
		maxDataEntryValueSizeBVOKRes = base64.StdEncoding.EncodeToString(maxDataEntryValueSizeBVOK)
	)
	var (
		overMaxInput = make([]byte, maxBase64BytesToEncode+1)
		maxInput     = overMaxInput[:maxBase64BytesToEncode]
		maxInputRes  = base64.StdEncoding.EncodeToString(maxInput)
	)
	for i, test := range []struct {
		reduceLimit bool
		args        []rideType
		fail        bool
		r           rideType
	}{
		{false, []rideType{rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, rideString("VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q=")}, //nolint:lll
		{false, []rideType{rideByteVector{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}}, false, rideString("AQa3b8tH")},
		{false, []rideType{rideByteVector{}}, false, rideString("")},
		{false, []rideType{rideUnit{}}, false, rideString("")},
		{false, []rideType{rideByteVector{}, rideByteVector{}}, true, nil},
		{false, []rideType{rideByteVector{1, 2, 4}, rideInt(0)}, true, nil},
		{false, []rideType{rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}}, true, nil},
		{false, []rideType{rideInt(1), rideString("x")}, true, nil},
		{false, []rideType{}, true, nil},
		//
		{false, []rideType{rideByteVector(maxDataWithProofsBytesBV)}, true, nil},
		{false, []rideType{rideByteVector(maxDataWithProofsBytesBVOK)}, true, nil}, // fails because of huge input
		//
		{true, []rideType{rideByteVector(maxDataEntryValueSizeBV)}, true, nil},
		{true, []rideType{rideByteVector(maxDataEntryValueSizeBVOK)}, false, rideString(maxDataEntryValueSizeBVOKRes)}, //nolint:lll
		// both of these should fail because of huge input
		{false, []rideType{rideByteVector(overMaxInput)}, true, nil},
		{true, []rideType{rideByteVector(overMaxInput)}, true, nil},
		//
		{false, []rideType{rideByteVector(maxInput)}, false, rideString(maxInputRes)},
		{true, []rideType{rideByteVector(maxInput)}, true, nil}, // fails because of huge output
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			r, err := toBase64Generic(test.reduceLimit, test.args...)
			if test.fail {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.r, r)
			}
		})
	}
}

func TestFromBase64(t *testing.T) {
	var (
		overMaxInput    = make([]byte, maxBase64StringToDecode*3/4+1)
		overMaxInputB64 = base64.StdEncoding.EncodeToString(overMaxInput)
	)
	var (
		maxInput    = overMaxInput[:maxBase64StringToDecode*3/4]
		maxInputB64 = base64.StdEncoding.EncodeToString(maxInput)
	)
	for i, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q=")}, false, rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]rideType{rideString("base64:VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q=")}, false, rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]rideType{rideString("AQa3b8tH")}, false, rideByteVector{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}},
		{[]rideType{rideString("base64:AQa3b8tH")}, false, rideByteVector{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}},
		{[]rideType{rideString("")}, false, rideByteVector{}},
		{[]rideType{rideString("base64:")}, false, rideByteVector{}},
		{[]rideType{rideString("base64")}, false, rideByteVector{0x6d, 0xab, 0x1e, 0xeb}},
		{[]rideType{rideString("base64:"), rideString("")}, true, nil},
		{[]rideType{rideByteVector{1, 2, 4}}, true, nil},
		{[]rideType{rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{}, true, nil},
		//
		{[]rideType{rideString(overMaxInputB64)}, true, nil},
		{[]rideType{rideString(maxInputB64)}, false, rideByteVector(maxInput)},
		{[]rideType{rideString("base64:" + maxInputB64)}, true, nil}, // prefix is also included in the length check
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			r, err := fromBase64(nil, test.args...)
			if test.fail {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.r, r)
			}
		})
	}
}

func TestToBase16Generic(t *testing.T) {
	var (
		maxDataEntryValueSizeBV      = make([]byte, proto.MaxDataEntryValueSize/2+1)
		maxDataEntryValueSizeBVOK    = maxDataEntryValueSizeBV[:proto.MaxDataEntryValueSize/2]
		maxDataEntryValueSizeBVOKRes = hex.EncodeToString(maxDataEntryValueSizeBVOK)
	)
	var (
		overMaxInput    = make([]byte, maxBase16BytesToEncode+1)
		overMaxInputRes = hex.EncodeToString(overMaxInput)
		maxInput        = overMaxInput[:maxBase16BytesToEncode]
		maxInputRes     = hex.EncodeToString(maxInput)
	)
	for i, test := range []struct {
		checkLength bool
		args        []rideType
		fail        bool
		r           rideType
	}{
		{false, []rideType{rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, rideString("5468697320697320612073696d706c6520737472696e6720666f722074657374")}, //nolint:lll
		{false, []rideType{rideByteVector{}}, false, rideString("")},
		{false, []rideType{rideUnit{}}, false, rideString("")},
		{false, []rideType{rideByteVector{}, rideByteVector{}}, true, nil},
		{false, []rideType{rideByteVector{1, 2, 4}, rideInt(0)}, true, nil},
		{false, []rideType{rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}}, true, nil},
		{false, []rideType{rideInt(1), rideString("x")}, true, nil},
		{false, []rideType{}, true, nil},
		{false, []rideType{rideByteVector(maxDataEntryValueSizeBV)}, true, nil},
		{false, []rideType{rideByteVector(maxDataEntryValueSizeBVOK)}, false, rideString(maxDataEntryValueSizeBVOKRes)},
		//
		{false, []rideType{rideByteVector(overMaxInput)}, false, rideString(overMaxInputRes)},
		{true, []rideType{rideByteVector(overMaxInput)}, true, nil},
		//
		{false, []rideType{rideByteVector(maxInput)}, false, rideString(maxInputRes)},
		{true, []rideType{rideByteVector(maxInput)}, false, rideString(maxInputRes)},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			r, err := toBase16Generic(test.checkLength, test.args...)
			if test.fail {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.r, r)
			}
		})
	}
}

func TestFromBase16Generic(t *testing.T) {
	var (
		overMaxInput    = make([]byte, maxBase16StringToDecode/2+1)
		overMaxInputRes = hex.EncodeToString(overMaxInput)
		maxInput        = overMaxInput[:maxBase16StringToDecode/2]
		maxInputRes     = hex.EncodeToString(maxInput)
	)
	for i, test := range []struct {
		checkLength bool
		args        []rideType
		fail        bool
		r           rideType
	}{
		{false, []rideType{rideString("5468697320697320612073696d706c6520737472696e6720666f722074657374")}, false, rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},        //nolint:lll
		{false, []rideType{rideString("base16:5468697320697320612073696d706c6520737472696e6720666f722074657374")}, false, rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, //nolint:lll
		{false, []rideType{rideString("")}, false, rideByteVector{}},
		{false, []rideType{rideString("base16:")}, false, rideByteVector{}},
		{false, []rideType{rideString("base16")}, true, nil},
		{false, []rideType{rideString("base16:"), rideString("")}, true, nil},
		{false, []rideType{rideByteVector{1, 2, 4}}, true, nil},
		{false, []rideType{rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}}, true, nil},
		{false, []rideType{rideInt(1), rideString("x")}, true, nil},
		{false, []rideType{}, true, nil},
		//
		{false, []rideType{rideString(overMaxInputRes)}, false, rideByteVector(overMaxInput)},
		{true, []rideType{rideString(overMaxInputRes)}, true, nil},
		//
		{false, []rideType{rideString(maxInputRes)}, false, rideByteVector(maxInput)},
		{true, []rideType{rideString(maxInputRes)}, false, rideByteVector(maxInput)},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			r, err := fromBase16Generic(test.checkLength, test.args...)
			if test.fail {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.r, r)
			}
		})
	}
}

func TestDropRightBytes(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(2)}, false, rideByteVector{1, 2, 3}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(4)}, false, rideByteVector{1}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(5)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(8)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(0)}, false, rideByteVector{1, 2, 3, 4, 5}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(-4)}, false, rideByteVector{1, 2, 3, 4, 5}},
		{[]rideType{rideByteVector{}, rideInt(0)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{}, rideInt(3)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := dropRightBytes(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestTakeRightBytes(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(2)}, false, rideByteVector{4, 5}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(4)}, false, rideByteVector{2, 3, 4, 5}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(5)}, false, rideByteVector{1, 2, 3, 4, 5}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(8)}, false, rideByteVector{1, 2, 3, 4, 5}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(0)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}, rideInt(-4)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{}, rideInt(0)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{}, rideInt(3)}, false, rideByteVector{}},
		{[]rideType{rideByteVector{1, 2, 3, 4, 5}}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := takeRightBytes(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBytesToUTF8StringGeneric(t *testing.T) {
	broken, err := base64.StdEncoding.DecodeString("As7ayhU0UVXXiQ==")
	require.NoError(t, err)
	var (
		maxDataWithProofsBytesBV   = bytes.Repeat([]byte{'f'}, proto.MaxDataWithProofsBytes+1)
		maxDataWithProofsBytesBVOK = maxDataWithProofsBytesBV[:proto.MaxDataWithProofsBytes]
	)
	var (
		maxDataEntryValueSizeBV   = maxDataWithProofsBytesBV[:proto.MaxDataEntryValueSize+1]
		maxDataEntryValueSizeBVOK = maxDataEntryValueSizeBV[:proto.MaxDataEntryValueSize]
	)
	for i, test := range []struct {
		reduceLimit bool
		args        []rideType
		fail        bool
		r           rideType
	}{
		{false, []rideType{rideByteVector("blah-blah-blah")}, false, rideString("blah-blah-blah")},
		{false, []rideType{rideByteVector("")}, false, rideString("")},
		{false, []rideType{rideByteVector{}}, false, rideString("")},
		{false, []rideType{rideByteVector(broken)}, true, nil},
		{false, []rideType{rideString("blah-blah-blah")}, true, nil},
		{false, []rideType{rideByteVector{0, 0, 0, 0, 0, 0, 0, 1}, rideInt(1)}, true, nil},
		{false, []rideType{}, true, nil},
		{false, []rideType{rideByteVector(maxDataWithProofsBytesBV)}, true, nil},
		{false, []rideType{rideByteVector(maxDataWithProofsBytesBVOK)}, false, rideString(maxDataWithProofsBytesBVOK)},
		{true, []rideType{rideByteVector(maxDataEntryValueSizeBV)}, true, nil},
		{true, []rideType{rideByteVector(maxDataEntryValueSizeBVOK)}, false, rideString(maxDataEntryValueSizeBVOK)},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			r, fErr := bytesToUTF8StringGeneric(test.reduceLimit, test.args...)
			if test.fail {
				assert.Error(t, fErr)
			} else {
				require.NoError(t, fErr)
				assert.Equal(t, test.r, r)
			}
		})
	}
}

func TestBytesToInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector{0, 0, 0, 0, 0, 0, 0, 0}}, false, rideInt(0)},
		{[]rideType{rideByteVector{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, false, rideInt(math.MaxInt64)},
		{[]rideType{rideByteVector{0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5}}, false, rideInt(1)},
		{[]rideType{rideByteVector{}}, true, nil},
		{[]rideType{rideByteVector{0, 0, 0, 0}}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := bytesToInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBytesToIntWithOffset(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector{0, 0, 0, 0, 0, 0, 0, 0}, rideInt(0)}, false, rideInt(0)},
		{[]rideType{rideByteVector{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, rideInt(0)}, false, rideInt(math.MaxInt64)},
		{[]rideType{rideByteVector{0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0}, rideInt(2)}, false, rideInt(0)},
		{[]rideType{rideByteVector{0xff, 0xff, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, rideInt(2)}, false, rideInt(math.MaxInt64)},
		{[]rideType{rideByteVector{}, rideInt(0)}, true, nil},
		{[]rideType{rideByteVector{0, 0, 0, 0, 0, 0, 0, 1}, rideInt(1)}, true, nil},
		{[]rideType{rideByteVector{0, 0, 0, 0, 0, 0, 0, 0}}, true, nil},
		{[]rideType{rideByteVector{0, 0, 0, 0, 0, 0, 0, 0}, rideString("x")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := bytesToIntWithOffset(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}
