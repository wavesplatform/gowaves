package ride

import (
	"encoding/base64"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestToBase58(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, rideString("6gVbAXCUdsa14xdsSk2SKaNBXs271V3Mo4zjb2cvCrsM")},
		{[]rideType{rideByteVector{0, 0, 0, 0, 0}}, false, rideString("11111")},
		{[]rideType{rideByteVector{}}, false, rideString("")},
		{[]rideType{rideUnit{}}, false, rideString("")},
		{[]rideType{rideByteVector{}, rideByteVector{}}, true, nil},
		{[]rideType{rideByteVector{1, 2, 4}, rideInt(0)}, true, nil},
		{[]rideType{rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := toBase58(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestFromBase58(t *testing.T) {
	for _, test := range []struct {
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
	} {
		r, err := fromBase58(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestToBase64(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, rideString("VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q=")},
		{[]rideType{rideByteVector{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}}, false, rideString("AQa3b8tH")},
		{[]rideType{rideByteVector{}}, false, rideString("")},
		{[]rideType{rideUnit{}}, false, rideString("")},
		{[]rideType{rideByteVector{}, rideByteVector{}}, true, nil},
		{[]rideType{rideByteVector{1, 2, 4}, rideInt(0)}, true, nil},
		{[]rideType{rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := toBase64(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestFromBase64(t *testing.T) {
	for _, test := range []struct {
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
	} {
		r, err := fromBase64(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestToBase16(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, rideString("5468697320697320612073696d706c6520737472696e6720666f722074657374")},
		{[]rideType{rideByteVector{}}, false, rideString("")},
		{[]rideType{rideUnit{}}, false, rideString("")},
		{[]rideType{rideByteVector{}, rideByteVector{}}, true, nil},
		{[]rideType{rideByteVector{1, 2, 4}, rideInt(0)}, true, nil},
		{[]rideType{rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := toBase16(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestFromBase16(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("5468697320697320612073696d706c6520737472696e6720666f722074657374")}, false, rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]rideType{rideString("base16:5468697320697320612073696d706c6520737472696e6720666f722074657374")}, false, rideByteVector{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]rideType{rideString("")}, false, rideByteVector{}},
		{[]rideType{rideString("base16:")}, false, rideByteVector{}},
		{[]rideType{rideString("base16")}, true, nil},
		{[]rideType{rideString("base16:"), rideString("")}, true, nil},
		{[]rideType{rideByteVector{1, 2, 4}}, true, nil},
		{[]rideType{rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}, rideByteVector{1, 2, 3}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := fromBase16(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
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

func TestBytesToUTF8String(t *testing.T) {
	broken, err := base64.StdEncoding.DecodeString("As7ayhU0UVXXiQ==")
	require.NoError(t, err)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideByteVector("blah-blah-blah")}, false, rideString("blah-blah-blah")},
		{[]rideType{rideByteVector("")}, false, rideString("")},
		{[]rideType{rideByteVector{}}, false, rideString("")},
		{[]rideType{rideByteVector(broken)}, true, nil},
		{[]rideType{rideString("blah-blah-blah")}, true, nil},
		{[]rideType{rideByteVector{0, 0, 0, 0, 0, 0, 0, 1}, rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := bytesToUTF8String(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
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
