package state

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

type cborFastRecord interface {
	appendCBOR(dst []byte) []byte
	decodeCBOR(data []byte) bool
	marshalBinary() ([]byte, error)
	unmarshalBinary(data []byte) error
}

// cborRandUint64 returns a random value with a random bit length, so all integer head sizes are covered.
func cborRandUint64(r *rand.Rand) uint64 {
	bits := r.IntN(65)
	if bits == 0 {
		return 0
	}
	if bits == 64 {
		return r.Uint64()
	}
	return r.Uint64() & (1<<bits - 1)
}

func cborRandInt64(r *rand.Rand) int64 {
	v := int64(cborRandUint64(r) >> 1)
	if r.IntN(2) == 0 {
		return -v - int64(r.IntN(2)) // Covers math.MinInt64 together with the edge cases.
	}
	return v
}

func cborRandBytes(r *rand.Rand, dst []byte) {
	if r.IntN(4) == 0 {
		return // Keep zero bytes.
	}
	for i := range dst {
		dst[i] = byte(r.Uint32())
	}
}

func cborRandDigest(r *rand.Rand) *crypto.Digest {
	if r.IntN(3) == 0 {
		return nil
	}
	d := new(crypto.Digest)
	cborRandBytes(r, d[:])
	return d
}

func cborRandRecords(r *rand.Rand) []cborFastRecord {
	ai := &assetConstInfo{
		Decimals:             uint8(cborRandUint64(r)),
		IssueHeight:          cborRandUint64(r),
		IsNFT:                r.IntN(2) == 0,
		IssueSequenceInBlock: uint32(cborRandUint64(r)),
	}
	cborRandBytes(r, ai.Tail[:])
	cborRandBytes(r, ai.Issuer[:])
	l := &leasing{
		Amount:              cborRandUint64(r),
		OriginHeight:        cborRandUint64(r),
		Status:              LeaseStatus(cborRandUint64(r)),
		OriginTransactionID: cborRandDigest(r),
		CancelHeight:        cborRandUint64(r),
		CancelTransactionID: cborRandDigest(r),
	}
	cborRandBytes(r, l.SenderPK[:])
	cborRandBytes(r, l.RecipientAddr[:])
	sr := &scriptBasicInfoRecord{
		ScriptLen:      uint32(cborRandUint64(r)),
		LibraryVersion: ast.LibraryVersion(cborRandUint64(r)),
		HasVerifier:    r.IntN(2) == 0,
		IsDApp:         r.IntN(2) == 0,
	}
	cborRandBytes(r, sr.PK[:])
	return []cborFastRecord{
		&wavesBalanceRecord{balanceProfile{
			Balance: cborRandUint64(r), LeaseIn: cborRandInt64(r), LeaseOut: cborRandInt64(r),
			Deposit: cborRandUint64(r),
		}},
		&assetBalanceRecord{Balance: cborRandUint64(r)},
		ai, l, sr,
	}
}

func cborEdgeRecords() []cborFastRecord {
	maxDigest := crypto.Digest{}
	for i := range maxDigest {
		maxDigest[i] = 0xff
	}
	var maxPK crypto.PublicKey
	copy(maxPK[:], maxDigest[:])
	var maxAddr proto.WavesAddress
	copy(maxAddr[:], maxDigest[:])
	var maxTail [proto.AssetIDTailSize]byte
	copy(maxTail[:], maxDigest[:])
	return []cborFastRecord{
		&wavesBalanceRecord{},
		&wavesBalanceRecord{balanceProfile{
			Balance: math.MaxUint64, LeaseIn: math.MaxInt64, LeaseOut: math.MaxInt64, Deposit: math.MaxUint64,
		}},
		&wavesBalanceRecord{balanceProfile{LeaseIn: math.MinInt64, LeaseOut: -1}},
		&wavesBalanceRecord{balanceProfile{Balance: 23, LeaseIn: -24, LeaseOut: -25, Deposit: 24}},
		&wavesBalanceRecord{balanceProfile{Balance: 255, LeaseIn: 256, LeaseOut: -256, Deposit: 65535}},
		&wavesBalanceRecord{balanceProfile{Balance: 65536, LeaseIn: math.MaxUint32, LeaseOut: -math.MaxUint32 - 1}},
		&assetBalanceRecord{},
		&assetBalanceRecord{Balance: math.MaxInt64},
		&assetBalanceRecord{Balance: math.MaxUint64},
		&assetConstInfo{},
		&assetConstInfo{Tail: maxTail, Issuer: maxPK, Decimals: math.MaxUint8, IssueHeight: math.MaxUint64,
			IsNFT: true, IssueSequenceInBlock: math.MaxUint32},
		&leasing{},
		&leasing{OriginTransactionID: &crypto.Digest{}, CancelTransactionID: &crypto.Digest{}},
		&leasing{SenderPK: maxPK, RecipientAddr: maxAddr, Amount: math.MaxUint64, OriginHeight: math.MaxUint64,
			Status: math.MaxUint8, OriginTransactionID: &maxDigest, CancelHeight: math.MaxUint64,
			CancelTransactionID: &maxDigest},
		&scriptBasicInfoRecord{},
		&scriptBasicInfoRecord{PK: maxPK, ScriptLen: math.MaxUint32, LibraryVersion: math.MaxUint8,
			HasVerifier: true, IsDApp: true},
	}
}

// newEmptyLike returns a new zero record of the same type as rec.
func newEmptyLike(t testing.TB, rec cborFastRecord) cborFastRecord {
	switch rec.(type) {
	case *wavesBalanceRecord:
		return &wavesBalanceRecord{}
	case *assetBalanceRecord:
		return &assetBalanceRecord{}
	case *assetConstInfo:
		return &assetConstInfo{}
	case *leasing:
		return &leasing{}
	case *scriptBasicInfoRecord:
		return &scriptBasicInfoRecord{}
	default:
		t.Fatalf("unexpected record type %T", rec)
		return nil
	}
}

func checkCBORFastRecord(t *testing.T, rec cborFastRecord) {
	expected, err := cbor.Marshal(rec)
	require.NoError(t, err)
	actual, err := rec.marshalBinary()
	require.NoError(t, err)
	require.Equal(t, expected, actual, "%T %+v", rec, rec)
	assert.Equal(t, len(actual), cap(actual))
	prefix := []byte{1, 2, 3}
	assert.Equal(t, append(prefix, expected...), rec.appendCBOR(slices.Clip(prefix)))

	fast := newEmptyLike(t, rec)
	require.True(t, fast.decodeCBOR(actual), "fast decoder must accept canonical bytes of %T", rec)
	assert.Equal(t, rec, fast)
	slow := newEmptyLike(t, rec)
	require.NoError(t, cbor.Unmarshal(actual, slow))
	assert.Equal(t, slow, fast)
}

func TestCBORFastEdgeValues(t *testing.T) {
	for _, rec := range cborEdgeRecords() {
		checkCBORFastRecord(t, rec)
	}
}

func TestCBORFastRandomValues(t *testing.T) {
	r := rand.New(rand.NewPCG(20261126, 11))
	for range 2000 {
		for _, rec := range cborRandRecords(r) {
			checkCBORFastRecord(t, rec)
		}
	}
}

func TestCBORFastDecodeNonShortestIntegers(t *testing.T) {
	// map(1) {0: 1} with the value written as a uint64 head.
	data := []byte{0xa1, 0x00, 0x1b, 0, 0, 0, 0, 0, 0, 0, 0x01}
	var rec assetBalanceRecord
	require.True(t, rec.decodeCBOR(data))
	assert.Equal(t, uint64(1), rec.Balance)
	var slow assetBalanceRecord
	require.NoError(t, cbor.Unmarshal(data, &slow))
	assert.Equal(t, slow, rec)
}

func TestCBORFastDecodeFallback(t *testing.T) {
	valid, err := (&leasing{Amount: 5, OriginTransactionID: &crypto.Digest{1}}).marshalBinary()
	require.NoError(t, err)
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"trailing byte", append(slices.Clip(valid), 0x00)},
		{"truncated", valid[:len(valid)-1]},
		{"not a map", []byte{0x80}},
		{"indefinite map", []byte{0xbf, 0xff}},
		{"too many entries", []byte{0xa9}},
		{"unknown key", []byte{0xa1, 0x06, 0x00}},
		{"descending keys", []byte{0xa2, 0x02, 0x01, 0x00, 0x58, 0x20}},
		{"duplicate keys", []byte{0xa2, 0x02, 0x01, 0x02, 0x02}},
		{"negative key", []byte{0xa1, 0x20, 0x00}},
		{"status overflow", []byte{0xa1, 0x04, 0x19, 0x01, 0x00}},
		{"null digest", []byte{0xa1, 0x05, 0xf6}},
		{"short digest", []byte{0xa1, 0x05, 0x41, 0x00}},
		{"tagged amount", []byte{0xa1, 0x02, 0xc2, 0x41, 0x01}},
		{"reserved head", []byte{0xa1, 0x02, 0x1c}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var fast leasing
			assert.False(t, fast.decodeCBOR(tc.data))
			assert.Equal(t, leasing{}, fast, "failed fast decoding must not change the record")
			var rec, slow leasing
			fastErr := rec.unmarshalBinary(tc.data)
			slowErr := cbor.Unmarshal(tc.data, &slow)
			if slowErr != nil {
				assert.EqualError(t, fastErr, slowErr.Error())
			} else {
				require.NoError(t, fastErr)
				assert.Equal(t, slow, rec)
			}
		})
	}
}

func TestCBORFastDecodeKeepsAbsentFields(t *testing.T) {
	digest := crypto.Digest{7}
	prev := leasing{Amount: 1, CancelHeight: 10, OriginTransactionID: &digest}
	data, err := (&leasing{Amount: 2, OriginTransactionID: &crypto.Digest{9}}).marshalBinary()
	require.NoError(t, err)

	fast := prev
	require.NoError(t, fast.unmarshalBinary(data))
	slow := leasing{Amount: 1, CancelHeight: 10, OriginTransactionID: &crypto.Digest{7}}
	require.NoError(t, cbor.Unmarshal(data, &slow))
	assert.Equal(t, slow, fast)
	assert.Equal(t, uint64(10), fast.CancelHeight)
	assert.Same(t, &digest, fast.OriginTransactionID, "existing digest pointer is reused as by cbor.Unmarshal")
	assert.Equal(t, crypto.Digest{9}, digest)
}

func cborFuzzTargets() []func() cborFastRecord {
	return []func() cborFastRecord{
		func() cborFastRecord { return &wavesBalanceRecord{} },
		func() cborFastRecord { return &assetBalanceRecord{} },
		func() cborFastRecord { return &assetConstInfo{} },
		func() cborFastRecord { return &leasing{} },
		func() cborFastRecord { return &scriptBasicInfoRecord{} },
	}
}

func FuzzCBORFastDecode(f *testing.F) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 20 {
		for _, rec := range cborRandRecords(r) {
			data, err := rec.marshalBinary()
			require.NoError(f, err)
			f.Add(data)
		}
	}
	for _, rec := range cborEdgeRecords() {
		data, err := rec.marshalBinary()
		require.NoError(f, err)
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, newRec := range cborFuzzTargets() {
			fast, slow := newRec(), newRec()
			err := fast.unmarshalBinary(data)
			slowErr := cbor.Unmarshal(data, slow)
			if slowErr != nil {
				require.EqualError(t, err, slowErr.Error())
				continue
			}
			require.NoError(t, err)
			require.Equal(t, slow, fast)
			// Canonical bytes must round trip through the fast path.
			canonical, mErr := fast.marshalBinary()
			require.NoError(t, mErr)
			again := newRec()
			require.True(t, again.decodeCBOR(canonical))
			require.Equal(t, fast, again)
		}
	})
}
