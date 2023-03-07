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
		{[]rideType{rideBytes{0, 0, 0, 0, 0, 0, 0, 0}}, false, rideInt(8)},
		{[]rideType{rideBytes{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, false, rideInt(8)},
		{[]rideType{rideBytes{0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5}}, false, rideInt(12)},
		{[]rideType{rideBytes{}}, false, rideInt(0)},
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
		{[]rideType{rideBytes{1, 2, 3}, rideInt(2)}, false, rideBytes{1, 2}},
		{[]rideType{rideBytes{1, 2, 3}, rideInt(4)}, false, rideBytes{1, 2, 3}},
		{[]rideType{rideBytes{1, 2, 3}, rideInt(0)}, false, rideBytes{}},
		{[]rideType{rideBytes{1, 2, 3}, rideInt(-4)}, false, rideBytes{}},
		{[]rideType{rideBytes{}, rideInt(0)}, false, rideBytes{}},
		{[]rideType{rideBytes{}, rideInt(3)}, false, rideBytes{}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}}, true, nil},
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
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(2)}, false, rideBytes{3, 4, 5}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(4)}, false, rideBytes{5}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(5)}, false, rideBytes{}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(8)}, false, rideBytes{}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(0)}, false, rideBytes{1, 2, 3, 4, 5}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(-4)}, false, rideBytes{1, 2, 3, 4, 5}},
		{[]rideType{rideBytes{}, rideInt(0)}, false, rideBytes{}},
		{[]rideType{rideBytes{}, rideInt(3)}, false, rideBytes{}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}}, true, nil},
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
		{[]rideType{rideBytes{1, 2, 3}, rideBytes{4, 5}}, false, rideBytes{1, 2, 3, 4, 5}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideBytes{6}}, false, rideBytes{1, 2, 3, 4, 5, 6}},
		{[]rideType{rideBytes{1, 2, 3}, rideBytes{}}, false, rideBytes{1, 2, 3}},
		{[]rideType{rideBytes{}, rideBytes{1, 2, 3}}, false, rideBytes{1, 2, 3}},
		{[]rideType{rideBytes{}, rideBytes{}}, false, rideBytes{}},
		{[]rideType{rideBytes{1, 2, 3}}, true, nil},
		{[]rideType{rideBytes{1, 2, 4}, rideInt(0)}, true, nil},
		{[]rideType{rideBytes{1, 2, 3}, rideBytes{1, 2, 3}, rideBytes{1, 2, 3}}, true, nil},
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
		{[]rideType{rideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, rideString("6gVbAXCUdsa14xdsSk2SKaNBXs271V3Mo4zjb2cvCrsM")},
		{[]rideType{rideBytes{0, 0, 0, 0, 0}}, false, rideString("11111")},
		{[]rideType{rideBytes{}}, false, rideString("")},
		{[]rideType{rideUnit{}}, false, rideString("")},
		{[]rideType{rideBytes{}, rideBytes{}}, true, nil},
		{[]rideType{rideBytes{1, 2, 4}, rideInt(0)}, true, nil},
		{[]rideType{rideBytes{1, 2, 3}, rideBytes{1, 2, 3}, rideBytes{1, 2, 3}}, true, nil},
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
		{[]rideType{rideString("6gVbAXCUdsa14xdsSk2SKaNBXs271V3Mo4zjb2cvCrsM")}, false, rideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]rideType{rideString("11111")}, false, rideBytes{0, 0, 0, 0, 0}},
		{[]rideType{rideString("")}, false, rideBytes{}},
		{[]rideType{rideString(""), rideString("")}, true, nil},
		{[]rideType{rideBytes{1, 2, 4}}, true, nil},
		{[]rideType{rideBytes{1, 2, 3}, rideBytes{1, 2, 3}, rideBytes{1, 2, 3}}, true, nil},
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
		{[]rideType{rideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, rideString("VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q=")},
		{[]rideType{rideBytes{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}}, false, rideString("AQa3b8tH")},
		{[]rideType{rideBytes{}}, false, rideString("")},
		{[]rideType{rideUnit{}}, false, rideString("")},
		{[]rideType{rideBytes{}, rideBytes{}}, true, nil},
		{[]rideType{rideBytes{1, 2, 4}, rideInt(0)}, true, nil},
		{[]rideType{rideBytes{1, 2, 3}, rideBytes{1, 2, 3}, rideBytes{1, 2, 3}}, true, nil},
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
		{[]rideType{rideString("VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q=")}, false, rideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]rideType{rideString("base64:VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q=")}, false, rideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]rideType{rideString("AQa3b8tH")}, false, rideBytes{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}},
		{[]rideType{rideString("base64:AQa3b8tH")}, false, rideBytes{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}},
		{[]rideType{rideString("")}, false, rideBytes{}},
		{[]rideType{rideString("base64:")}, false, rideBytes{}},
		{[]rideType{rideString("base64")}, false, rideBytes{0x6d, 0xab, 0x1e, 0xeb}},
		{[]rideType{rideString("base64:"), rideString("")}, true, nil},
		{[]rideType{rideBytes{1, 2, 4}}, true, nil},
		{[]rideType{rideBytes{1, 2, 3}, rideBytes{1, 2, 3}, rideBytes{1, 2, 3}}, true, nil},
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
		{[]rideType{rideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, rideString("5468697320697320612073696d706c6520737472696e6720666f722074657374")},
		{[]rideType{rideBytes{}}, false, rideString("")},
		{[]rideType{rideUnit{}}, false, rideString("")},
		{[]rideType{rideBytes{}, rideBytes{}}, true, nil},
		{[]rideType{rideBytes{1, 2, 4}, rideInt(0)}, true, nil},
		{[]rideType{rideBytes{1, 2, 3}, rideBytes{1, 2, 3}, rideBytes{1, 2, 3}}, true, nil},
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
		{[]rideType{rideString("5468697320697320612073696d706c6520737472696e6720666f722074657374")}, false, rideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]rideType{rideString("base16:5468697320697320612073696d706c6520737472696e6720666f722074657374")}, false, rideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]rideType{rideString("")}, false, rideBytes{}},
		{[]rideType{rideString("base16:")}, false, rideBytes{}},
		{[]rideType{rideString("base16")}, true, nil},
		{[]rideType{rideString("base16:"), rideString("")}, true, nil},
		{[]rideType{rideBytes{1, 2, 4}}, true, nil},
		{[]rideType{rideBytes{1, 2, 3}, rideBytes{1, 2, 3}, rideBytes{1, 2, 3}}, true, nil},
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
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(2)}, false, rideBytes{1, 2, 3}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(4)}, false, rideBytes{1}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(5)}, false, rideBytes{}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(8)}, false, rideBytes{}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(0)}, false, rideBytes{1, 2, 3, 4, 5}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(-4)}, false, rideBytes{1, 2, 3, 4, 5}},
		{[]rideType{rideBytes{}, rideInt(0)}, false, rideBytes{}},
		{[]rideType{rideBytes{}, rideInt(3)}, false, rideBytes{}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}}, true, nil},
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
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(2)}, false, rideBytes{4, 5}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(4)}, false, rideBytes{2, 3, 4, 5}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(5)}, false, rideBytes{1, 2, 3, 4, 5}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(8)}, false, rideBytes{1, 2, 3, 4, 5}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(0)}, false, rideBytes{}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}, rideInt(-4)}, false, rideBytes{}},
		{[]rideType{rideBytes{}, rideInt(0)}, false, rideBytes{}},
		{[]rideType{rideBytes{}, rideInt(3)}, false, rideBytes{}},
		{[]rideType{rideBytes{1, 2, 3, 4, 5}}, true, nil},
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
		{[]rideType{rideBytes("blah-blah-blah")}, false, rideString("blah-blah-blah")},
		{[]rideType{rideBytes("")}, false, rideString("")},
		{[]rideType{rideBytes{}}, false, rideString("")},
		{[]rideType{rideBytes(broken)}, true, nil},
		{[]rideType{rideString("blah-blah-blah")}, true, nil},
		{[]rideType{rideBytes{0, 0, 0, 0, 0, 0, 0, 1}, rideInt(1)}, true, nil},
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
		{[]rideType{rideBytes{0, 0, 0, 0, 0, 0, 0, 0}}, false, rideInt(0)},
		{[]rideType{rideBytes{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, false, rideInt(math.MaxInt64)},
		{[]rideType{rideBytes{0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5}}, false, rideInt(1)},
		{[]rideType{rideBytes{}}, true, nil},
		{[]rideType{rideBytes{0, 0, 0, 0}}, true, nil},
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
		{[]rideType{rideBytes{0, 0, 0, 0, 0, 0, 0, 0}, rideInt(0)}, false, rideInt(0)},
		{[]rideType{rideBytes{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, rideInt(0)}, false, rideInt(math.MaxInt64)},
		{[]rideType{rideBytes{0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0}, rideInt(2)}, false, rideInt(0)},
		{[]rideType{rideBytes{0xff, 0xff, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, rideInt(2)}, false, rideInt(math.MaxInt64)},
		{[]rideType{rideBytes{}, rideInt(0)}, true, nil},
		{[]rideType{rideBytes{0, 0, 0, 0, 0, 0, 0, 1}, rideInt(1)}, true, nil},
		{[]rideType{rideBytes{0, 0, 0, 0, 0, 0, 0, 0}}, true, nil},
		{[]rideType{rideBytes{0, 0, 0, 0, 0, 0, 0, 0}, rideString("x")}, true, nil},
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
