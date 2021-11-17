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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes{0, 0, 0, 0, 0, 0, 0, 0}}, false, RideInt(8)},
		{[]RideType{RideBytes{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, false, RideInt(8)},
		{[]RideType{RideBytes{0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5}}, false, RideInt(12)},
		{[]RideType{RideBytes{}}, false, RideInt(0)},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes{1, 2, 3}, RideInt(2)}, false, RideBytes{1, 2}},
		{[]RideType{RideBytes{1, 2, 3}, RideInt(4)}, false, RideBytes{1, 2, 3}},
		{[]RideType{RideBytes{1, 2, 3}, RideInt(0)}, false, RideBytes{}},
		{[]RideType{RideBytes{1, 2, 3}, RideInt(-4)}, false, RideBytes{}},
		{[]RideType{RideBytes{}, RideInt(0)}, false, RideBytes{}},
		{[]RideType{RideBytes{}, RideInt(3)}, false, RideBytes{}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(2)}, false, RideBytes{3, 4, 5}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(4)}, false, RideBytes{5}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(5)}, false, RideBytes{}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(8)}, false, RideBytes{}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(0)}, false, RideBytes{1, 2, 3, 4, 5}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(-4)}, false, RideBytes{1, 2, 3, 4, 5}},
		{[]RideType{RideBytes{}, RideInt(0)}, false, RideBytes{}},
		{[]RideType{RideBytes{}, RideInt(3)}, false, RideBytes{}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes{1, 2, 3}, RideBytes{4, 5}}, false, RideBytes{1, 2, 3, 4, 5}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideBytes{6}}, false, RideBytes{1, 2, 3, 4, 5, 6}},
		{[]RideType{RideBytes{1, 2, 3}, RideBytes{}}, false, RideBytes{1, 2, 3}},
		{[]RideType{RideBytes{}, RideBytes{1, 2, 3}}, false, RideBytes{1, 2, 3}},
		{[]RideType{RideBytes{}, RideBytes{}}, false, RideBytes{}},
		{[]RideType{RideBytes{1, 2, 3}}, true, nil},
		{[]RideType{RideBytes{1, 2, 4}, RideInt(0)}, true, nil},
		{[]RideType{RideBytes{1, 2, 3}, RideBytes{1, 2, 3}, RideBytes{1, 2, 3}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := concatBytes(nil, test.args...)
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, RideString("6gVbAXCUdsa14xdsSk2SKaNBXs271V3Mo4zjb2cvCrsM")},
		{[]RideType{RideBytes{0, 0, 0, 0, 0}}, false, RideString("11111")},
		{[]RideType{RideBytes{}}, false, RideString("")},
		{[]RideType{rideUnit{}}, false, RideString("")},
		{[]RideType{RideBytes{}, RideBytes{}}, true, nil},
		{[]RideType{RideBytes{1, 2, 4}, RideInt(0)}, true, nil},
		{[]RideType{RideBytes{1, 2, 3}, RideBytes{1, 2, 3}, RideBytes{1, 2, 3}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("6gVbAXCUdsa14xdsSk2SKaNBXs271V3Mo4zjb2cvCrsM")}, false, RideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]RideType{RideString("11111")}, false, RideBytes{0, 0, 0, 0, 0}},
		{[]RideType{RideString("")}, false, RideBytes{}},
		{[]RideType{RideString(""), RideString("")}, true, nil},
		{[]RideType{RideBytes{1, 2, 4}}, true, nil},
		{[]RideType{RideBytes{1, 2, 3}, RideBytes{1, 2, 3}, RideBytes{1, 2, 3}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, RideString("VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q=")},
		{[]RideType{RideBytes{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}}, false, RideString("AQa3b8tH")},
		{[]RideType{RideBytes{}}, false, RideString("")},
		{[]RideType{rideUnit{}}, false, RideString("")},
		{[]RideType{RideBytes{}, RideBytes{}}, true, nil},
		{[]RideType{RideBytes{1, 2, 4}, RideInt(0)}, true, nil},
		{[]RideType{RideBytes{1, 2, 3}, RideBytes{1, 2, 3}, RideBytes{1, 2, 3}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q=")}, false, RideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]RideType{RideString("base64:VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q=")}, false, RideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]RideType{RideString("AQa3b8tH")}, false, RideBytes{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}},
		{[]RideType{RideString("base64:AQa3b8tH")}, false, RideBytes{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}},
		{[]RideType{RideString("")}, false, RideBytes{}},
		{[]RideType{RideString("base64:")}, false, RideBytes{}},
		{[]RideType{RideString("base64")}, false, RideBytes{0x6d, 0xab, 0x1e, 0xeb}},
		{[]RideType{RideString("base64:"), RideString("")}, true, nil},
		{[]RideType{RideBytes{1, 2, 4}}, true, nil},
		{[]RideType{RideBytes{1, 2, 3}, RideBytes{1, 2, 3}, RideBytes{1, 2, 3}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}}, false, RideString("5468697320697320612073696d706c6520737472696e6720666f722074657374")},
		{[]RideType{RideBytes{}}, false, RideString("")},
		{[]RideType{rideUnit{}}, false, RideString("")},
		{[]RideType{RideBytes{}, RideBytes{}}, true, nil},
		{[]RideType{RideBytes{1, 2, 4}, RideInt(0)}, true, nil},
		{[]RideType{RideBytes{1, 2, 3}, RideBytes{1, 2, 3}, RideBytes{1, 2, 3}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("5468697320697320612073696d706c6520737472696e6720666f722074657374")}, false, RideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]RideType{RideString("base16:5468697320697320612073696d706c6520737472696e6720666f722074657374")}, false, RideBytes{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}},
		{[]RideType{RideString("")}, false, RideBytes{}},
		{[]RideType{RideString("base16:")}, false, RideBytes{}},
		{[]RideType{RideString("base16")}, true, nil},
		{[]RideType{RideString("base16:"), RideString("")}, true, nil},
		{[]RideType{RideBytes{1, 2, 4}}, true, nil},
		{[]RideType{RideBytes{1, 2, 3}, RideBytes{1, 2, 3}, RideBytes{1, 2, 3}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(2)}, false, RideBytes{1, 2, 3}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(4)}, false, RideBytes{1}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(5)}, false, RideBytes{}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(8)}, false, RideBytes{}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(0)}, false, RideBytes{1, 2, 3, 4, 5}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(-4)}, false, RideBytes{1, 2, 3, 4, 5}},
		{[]RideType{RideBytes{}, RideInt(0)}, false, RideBytes{}},
		{[]RideType{RideBytes{}, RideInt(3)}, false, RideBytes{}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(2)}, false, RideBytes{4, 5}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(4)}, false, RideBytes{2, 3, 4, 5}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(5)}, false, RideBytes{1, 2, 3, 4, 5}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(8)}, false, RideBytes{1, 2, 3, 4, 5}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(0)}, false, RideBytes{}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}, RideInt(-4)}, false, RideBytes{}},
		{[]RideType{RideBytes{}, RideInt(0)}, false, RideBytes{}},
		{[]RideType{RideBytes{}, RideInt(3)}, false, RideBytes{}},
		{[]RideType{RideBytes{1, 2, 3, 4, 5}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes("blah-blah-blah")}, false, RideString("blah-blah-blah")},
		{[]RideType{RideBytes("")}, false, RideString("")},
		{[]RideType{RideBytes{}}, false, RideString("")},
		{[]RideType{RideBytes(broken)}, true, nil},
		{[]RideType{RideString("blah-blah-blah")}, true, nil},
		{[]RideType{RideBytes{0, 0, 0, 0, 0, 0, 0, 1}, RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes{0, 0, 0, 0, 0, 0, 0, 0}}, false, RideInt(0)},
		{[]RideType{RideBytes{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, false, RideInt(math.MaxInt64)},
		{[]RideType{RideBytes{0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5}}, false, RideInt(1)},
		{[]RideType{RideBytes{}}, true, nil},
		{[]RideType{RideBytes{0, 0, 0, 0}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideBytes{0, 0, 0, 0, 0, 0, 0, 0}, RideInt(0)}, false, RideInt(0)},
		{[]RideType{RideBytes{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, RideInt(0)}, false, RideInt(math.MaxInt64)},
		{[]RideType{RideBytes{0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0}, RideInt(2)}, false, RideInt(0)},
		{[]RideType{RideBytes{0xff, 0xff, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, RideInt(2)}, false, RideInt(math.MaxInt64)},
		{[]RideType{RideBytes{}, RideInt(0)}, true, nil},
		{[]RideType{RideBytes{0, 0, 0, 0, 0, 0, 0, 1}, RideInt(1)}, true, nil},
		{[]RideType{RideBytes{0, 0, 0, 0, 0, 0, 0, 0}}, true, nil},
		{[]RideType{RideBytes{0, 0, 0, 0, 0, 0, 0, 0}, RideString("x")}, true, nil},
		{[]RideType{}, true, nil},
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
