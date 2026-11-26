package state

import (
	"encoding/binary"
	"math"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

// Hand-written CBOR encoders and decoders for the hot state records.
//
// The encoders produce exactly the bytes that cbor.Marshal (fxamacker/cbor default options) produces for the
// record structs: a definite-length map with integer keys written in the field declaration order, shortest-form
// integer heads, omitempty fields skipped when they are zero (or nil) and byte arrays, public keys and digests
// written as byte strings.
//
// The decoders accept only the layout the encoders produce, with any integer head length and with keys in
// ascending order. Anything else (unknown keys, tags, indefinite lengths, wrong byte string sizes, out of range
// values, trailing bytes) is not decoded by the fast path, and the caller falls back to cbor.Unmarshal, so
// decoding results and errors stay the same. As cbor.Unmarshal does, the decoders set only the fields present
// in the input and write a decoded digest into an already allocated pointer.

const (
	cborMajorUint  byte = 0 << 5
	cborMajorNeg   byte = 1 << 5
	cborMajorBytes byte = 2 << 5
	cborMajorMap   byte = 5 << 5
	cborMajorMask  byte = 7 << 5
	cborInfoMask   byte = 0x1f
	cborFalse      byte = 0xf4
	cborTrue       byte = 0xf5

	cborInfoUint8  = 24
	cborInfoUint16 = 25
	cborInfoUint32 = 26
	cborInfoUint64 = 27

	cborUint8Size  = 1
	cborUint16Size = 2
	cborUint32Size = 4
	cborUint64Size = 8

	// cborRecordBufSize is the size of a stack buffer large enough for any of the records below.
	cborRecordBufSize = 192
)

// cborAppendHead appends a CBOR data item head of the given major type with the argument n in the shortest form.
func cborAppendHead(dst []byte, major byte, n uint64) []byte {
	switch {
	case n < cborInfoUint8:
		return append(dst, major|byte(n))
	case n <= math.MaxUint8:
		return append(dst, major|cborInfoUint8, byte(n))
	case n <= math.MaxUint16:
		return binary.BigEndian.AppendUint16(append(dst, major|cborInfoUint16), uint16(n))
	case n <= math.MaxUint32:
		return binary.BigEndian.AppendUint32(append(dst, major|cborInfoUint32), uint32(n))
	default:
		return binary.BigEndian.AppendUint64(append(dst, major|cborInfoUint64), n)
	}
}

// cborAppendKey appends a small non-negative integer map key.
func cborAppendKey(dst []byte, key byte) []byte {
	return cborAppendHead(dst, cborMajorUint, uint64(key))
}

// cborAppendUint appends the key and the unsigned integer v in the shortest form, as cbor.Marshal does.
func cborAppendUint(dst []byte, key byte, v uint64) []byte {
	return cborAppendHead(cborAppendKey(dst, key), cborMajorUint, v)
}

// cborAppendInt appends the key and the signed integer v: a non-negative value as an unsigned integer, a negative
// one as a negative integer with the argument -1-v, both in the shortest form, as cbor.Marshal does.
func cborAppendInt(dst []byte, key byte, v int64) []byte {
	dst = cborAppendKey(dst, key)
	if v >= 0 {
		return cborAppendHead(dst, cborMajorUint, uint64(v))
	}
	return cborAppendHead(dst, cborMajorNeg, uint64(^v)) //nolint:gosec // ^v = -1 - v >= 0 for negative v.
}

// cborAppendBool appends the key and the simple value true or false.
func cborAppendBool(dst []byte, key byte, v bool) []byte {
	dst = cborAppendKey(dst, key)
	if v {
		return append(dst, cborTrue)
	}
	return append(dst, cborFalse)
}

// cborAppendBytes appends the key and b as a definite-length byte string. It is used only for fixed-size byte
// arrays (public keys, digests, addresses), which cbor.Marshal encodes as byte strings; nil slices, which
// cbor.Marshal encodes as null, must not be passed.
func cborAppendBytes(dst []byte, key byte, b []byte) []byte {
	dst = cborAppendHead(cborAppendKey(dst, key), cborMajorBytes, uint64(len(b)))
	return append(dst, b...)
}

// cborReader is a minimal decoder of the canonical record layout. Once any read fails, ok becomes false and all
// subsequent reads fail.
type cborReader struct {
	b       []byte
	ok      bool
	prevKey int // The last read key, -1 before the first key.
}

func newCBORReader(data []byte) cborReader {
	return cborReader{b: data, ok: true, prevKey: -1}
}

func (r *cborReader) fail() {
	r.ok = false
	r.b = nil
}

// head reads a data item head of the given major type and returns its argument.
func (r *cborReader) head(major byte) uint64 {
	if !r.ok || len(r.b) == 0 || r.b[0]&cborMajorMask != major {
		r.fail()
		return 0
	}
	info := r.b[0] & cborInfoMask
	arg := r.b[1:]
	var v uint64
	var size int
	switch {
	case info < cborInfoUint8:
		r.b = arg
		return uint64(info)
	case info == cborInfoUint8 && len(arg) >= cborUint8Size:
		v, size = uint64(arg[0]), cborUint8Size
	case info == cborInfoUint16 && len(arg) >= cborUint16Size:
		v, size = uint64(binary.BigEndian.Uint16(arg)), cborUint16Size
	case info == cborInfoUint32 && len(arg) >= cborUint32Size:
		v, size = uint64(binary.BigEndian.Uint32(arg)), cborUint32Size
	case info == cborInfoUint64 && len(arg) >= cborUint64Size:
		v, size = binary.BigEndian.Uint64(arg), cborUint64Size
	default: // Truncated input, reserved values and indefinite lengths.
		r.fail()
		return 0
	}
	r.b = arg[size:]
	return v
}

// mapLen reads a definite-length map head and checks that it has no more than maxLen entries.
func (r *cborReader) mapLen(maxLen uint64) uint64 {
	n := r.head(cborMajorMap)
	if n > maxLen {
		r.fail()
		return 0
	}
	return n
}

// cborInvalidKey is returned by cborReader.key on failure, it matches no record field.
const cborInvalidKey byte = math.MaxUint8

// key reads the next map key. Keys must be small non-negative integers in strictly ascending order.
func (r *cborReader) key() byte {
	k := r.head(cborMajorUint)
	if !r.ok || k >= uint64(cborInvalidKey) || int(k) <= r.prevKey {
		r.fail()
		return cborInvalidKey
	}
	r.prevKey = int(k)
	return byte(k)
}

func (r *cborReader) uint64() uint64 {
	return r.head(cborMajorUint)
}

func (r *cborReader) uint32() uint32 {
	v := r.head(cborMajorUint)
	if v > math.MaxUint32 {
		r.fail()
		return 0
	}
	return uint32(v)
}

func (r *cborReader) uint8() uint8 {
	v := r.head(cborMajorUint)
	if v > math.MaxUint8 {
		r.fail()
		return 0
	}
	return uint8(v)
}

func (r *cborReader) int64() int64 {
	if !r.ok || len(r.b) == 0 {
		r.fail()
		return 0
	}
	if r.b[0]&cborMajorMask == cborMajorNeg {
		n := r.head(cborMajorNeg)
		if n > math.MaxInt64 {
			r.fail()
			return 0
		}
		return -1 - int64(n)
	}
	v := r.head(cborMajorUint)
	if v > math.MaxInt64 {
		r.fail()
		return 0
	}
	return int64(v)
}

func (r *cborReader) bool() bool {
	if !r.ok || len(r.b) == 0 {
		r.fail()
		return false
	}
	switch r.b[0] {
	case cborTrue:
		r.b = r.b[1:]
		return true
	case cborFalse:
		r.b = r.b[1:]
		return false
	default:
		r.fail()
		return false
	}
}

// bytes reads a byte string of exactly len(dst) bytes into dst.
func (r *cborReader) bytes(dst []byte) {
	n := r.head(cborMajorBytes)
	if !r.ok || n != uint64(len(dst)) || len(r.b) < len(dst) {
		r.fail()
		return
	}
	copy(dst, r.b)
	r.b = r.b[len(dst):]
}

// done reports whether all reads succeeded and the whole input was consumed.
func (r *cborReader) done() bool {
	return r.ok && len(r.b) == 0
}

func boolToCount(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

// balanceProfile / wavesBalanceRecord: map(4) {0: Balance, 1: LeaseIn, 2: LeaseOut, 3: Deposit}.
const (
	balanceProfileKeyBalance  byte = 0
	balanceProfileKeyLeaseIn  byte = 1
	balanceProfileKeyLeaseOut byte = 2
	balanceProfileKeyDeposit  byte = 3
)

const balanceProfileFields = 4

// appendCBOR appends exactly the bytes cbor.Marshal produces for the record. The result fits in
// cborRecordBufSize bytes, so the callers encode into a stack buffer and clone the result into one exactly
// sized allocation (cborClone).
func (bp *balanceProfile) appendCBOR(dst []byte) []byte {
	dst = cborAppendHead(dst, cborMajorMap, balanceProfileFields)
	dst = cborAppendUint(dst, balanceProfileKeyBalance, bp.Balance)
	dst = cborAppendInt(dst, balanceProfileKeyLeaseIn, bp.LeaseIn)
	dst = cborAppendInt(dst, balanceProfileKeyLeaseOut, bp.LeaseOut)
	return cborAppendUint(dst, balanceProfileKeyDeposit, bp.Deposit)
}

func (bp *balanceProfile) decodeCBOR(data []byte) bool {
	r := newCBORReader(data)
	v := *bp
	for range r.mapLen(balanceProfileFields) {
		switch r.key() {
		case balanceProfileKeyBalance:
			v.Balance = r.uint64()
		case balanceProfileKeyLeaseIn:
			v.LeaseIn = r.int64()
		case balanceProfileKeyLeaseOut:
			v.LeaseOut = r.int64()
		case balanceProfileKeyDeposit:
			v.Deposit = r.uint64()
		default:
			r.fail()
		}
	}
	if !r.done() {
		return false
	}
	*bp = v
	return true
}

// assetBalanceRecord: map(1) {0: Balance}.
const (
	assetBalanceRecordKeyBalance byte = 0
)

// appendCBOR appends exactly the bytes cbor.Marshal produces for the record. The result fits in
// cborRecordBufSize bytes.
func (r *assetBalanceRecord) appendCBOR(dst []byte) []byte {
	dst = cborAppendHead(dst, cborMajorMap, 1)
	return cborAppendUint(dst, assetBalanceRecordKeyBalance, r.Balance)
}

func (r *assetBalanceRecord) decodeCBOR(data []byte) bool {
	rd := newCBORReader(data)
	v := *r
	for range rd.mapLen(1) {
		switch rd.key() {
		case assetBalanceRecordKeyBalance:
			v.Balance = rd.uint64()
		default:
			rd.fail()
		}
	}
	if !rd.done() {
		return false
	}
	*r = v
	return true
}

// assetConstInfo: {0: Tail, 1: Issuer, 2: Decimals?, 3: IssueHeight?, 4: IsNFT?, 5: IssueSequenceInBlock?}.
const (
	assetConstInfoKeyTail                 byte = 0
	assetConstInfoKeyIssuer               byte = 1
	assetConstInfoKeyDecimals             byte = 2
	assetConstInfoKeyIssueHeight          byte = 3
	assetConstInfoKeyIsNFT                byte = 4
	assetConstInfoKeyIssueSequenceInBlock byte = 5
)

const (
	assetConstInfoFields         = 6
	assetConstInfoRequiredFields = 2 // Fields without omitempty.
)

// appendCBOR appends exactly the bytes cbor.Marshal produces for the record: the map length counts only the
// written fields, the omitempty fields are skipped when they are zero. The result fits in cborRecordBufSize bytes.
func (ai *assetConstInfo) appendCBOR(dst []byte) []byte {
	n := assetConstInfoRequiredFields + boolToCount(ai.Decimals != 0) + boolToCount(ai.IssueHeight != 0) +
		boolToCount(ai.IsNFT) +
		boolToCount(ai.IssueSequenceInBlock != 0)
	dst = cborAppendHead(dst, cborMajorMap, n)
	dst = cborAppendBytes(dst, assetConstInfoKeyTail, ai.Tail[:])
	dst = cborAppendBytes(dst, assetConstInfoKeyIssuer, ai.Issuer[:])
	if ai.Decimals != 0 {
		dst = cborAppendUint(dst, assetConstInfoKeyDecimals, uint64(ai.Decimals))
	}
	if ai.IssueHeight != 0 {
		dst = cborAppendUint(dst, assetConstInfoKeyIssueHeight, ai.IssueHeight)
	}
	if ai.IsNFT {
		dst = cborAppendBool(dst, assetConstInfoKeyIsNFT, ai.IsNFT)
	}
	if ai.IssueSequenceInBlock != 0 {
		dst = cborAppendUint(dst, assetConstInfoKeyIssueSequenceInBlock, uint64(ai.IssueSequenceInBlock))
	}
	return dst
}

func (ai *assetConstInfo) decodeCBOR(data []byte) bool {
	r := newCBORReader(data)
	v := *ai
	for range r.mapLen(assetConstInfoFields) {
		switch r.key() {
		case assetConstInfoKeyTail:
			r.bytes(v.Tail[:])
		case assetConstInfoKeyIssuer:
			r.bytes(v.Issuer[:])
		case assetConstInfoKeyDecimals:
			v.Decimals = r.uint8()
		case assetConstInfoKeyIssueHeight:
			v.IssueHeight = r.uint64()
		case assetConstInfoKeyIsNFT:
			v.IsNFT = r.bool()
		case assetConstInfoKeyIssueSequenceInBlock:
			v.IssueSequenceInBlock = r.uint32()
		default:
			r.fail()
		}
	}
	if !r.done() {
		return false
	}
	*ai = v
	return true
}

// leasing: {0: SenderPK, 1: RecipientAddr, 2: Amount, 3: OriginHeight?, 4: Status, 5: OriginTransactionID?,
// 7: CancelHeight?, 8: CancelTransactionID?}.
const (
	leasingKeySenderPK            byte = 0
	leasingKeyRecipientAddr       byte = 1
	leasingKeyAmount              byte = 2
	leasingKeyOriginHeight        byte = 3
	leasingKeyStatus              byte = 4
	leasingKeyOriginTransactionID byte = 5
	leasingKeyCancelHeight        byte = 7
	leasingKeyCancelTransactionID byte = 8
)

const (
	leasingFields         = 8
	leasingRequiredFields = 4 // Fields without omitempty.
)

// appendCBOR appends exactly the bytes cbor.Marshal produces for the record: the map length counts only the
// written fields, the omitempty fields are skipped when they are zero and the digest pointers when they are nil.
// The result fits in cborRecordBufSize bytes.
func (l *leasing) appendCBOR(dst []byte) []byte {
	n := leasingRequiredFields + boolToCount(l.OriginHeight != 0) + boolToCount(l.OriginTransactionID != nil) +
		boolToCount(l.CancelHeight != 0) + boolToCount(l.CancelTransactionID != nil)
	dst = cborAppendHead(dst, cborMajorMap, n)
	dst = cborAppendBytes(dst, leasingKeySenderPK, l.SenderPK[:])
	dst = cborAppendBytes(dst, leasingKeyRecipientAddr, l.RecipientAddr[:])
	dst = cborAppendUint(dst, leasingKeyAmount, l.Amount)
	if l.OriginHeight != 0 {
		dst = cborAppendUint(dst, leasingKeyOriginHeight, l.OriginHeight)
	}
	dst = cborAppendUint(dst, leasingKeyStatus, uint64(l.Status))
	if l.OriginTransactionID != nil {
		dst = cborAppendBytes(dst, leasingKeyOriginTransactionID, l.OriginTransactionID[:])
	}
	if l.CancelHeight != 0 {
		dst = cborAppendUint(dst, leasingKeyCancelHeight, l.CancelHeight)
	}
	if l.CancelTransactionID != nil {
		dst = cborAppendBytes(dst, leasingKeyCancelTransactionID, l.CancelTransactionID[:])
	}
	return dst
}

func (l *leasing) decodeCBOR(data []byte) bool {
	r := newCBORReader(data)
	v := *l
	var originTxID, cancelTxID crypto.Digest
	var hasOriginTxID, hasCancelTxID bool
	for range r.mapLen(leasingFields) {
		switch r.key() {
		case leasingKeySenderPK:
			r.bytes(v.SenderPK[:])
		case leasingKeyRecipientAddr:
			r.bytes(v.RecipientAddr[:])
		case leasingKeyAmount:
			v.Amount = r.uint64()
		case leasingKeyOriginHeight:
			v.OriginHeight = r.uint64()
		case leasingKeyStatus:
			v.Status = LeaseStatus(r.uint8())
		case leasingKeyOriginTransactionID:
			r.bytes(originTxID[:])
			hasOriginTxID = true
		case leasingKeyCancelHeight:
			v.CancelHeight = r.uint64()
		case leasingKeyCancelTransactionID:
			r.bytes(cancelTxID[:])
			hasCancelTxID = true
		default:
			r.fail()
		}
	}
	if !r.done() {
		return false
	}
	if hasOriginTxID {
		setDigest(&v.OriginTransactionID, &originTxID)
	}
	if hasCancelTxID {
		setDigest(&v.CancelTransactionID, &cancelTxID)
	}
	*l = v
	return true
}

// setDigest stores d into the existing pointer or into a newly allocated one, as cbor.Unmarshal does.
func setDigest(p **crypto.Digest, d *crypto.Digest) {
	if *p == nil {
		*p = new(crypto.Digest)
	}
	**p = *d
}

// scriptBasicInfoRecord: {0: PK, 1: ScriptLen?, 2: LibraryVersion?, 3: HasVerifier?, 4: IsDApp?}.
const (
	scriptBasicInfoRecordKeyPK             byte = 0
	scriptBasicInfoRecordKeyScriptLen      byte = 1
	scriptBasicInfoRecordKeyLibraryVersion byte = 2
	scriptBasicInfoRecordKeyHasVerifier    byte = 3
	scriptBasicInfoRecordKeyIsDApp         byte = 4
)

const (
	scriptBasicInfoRecordFields         = 5
	scriptBasicInfoRecordRequiredFields = 1 // Fields without omitempty.
)

// appendCBOR appends exactly the bytes cbor.Marshal produces for the record: the map length counts only the
// written fields, the omitempty fields are skipped when they are zero. The result fits in cborRecordBufSize bytes.
func (r *scriptBasicInfoRecord) appendCBOR(dst []byte) []byte {
	n := scriptBasicInfoRecordRequiredFields + boolToCount(r.ScriptLen != 0) + boolToCount(r.LibraryVersion != 0) +
		boolToCount(r.HasVerifier) +
		boolToCount(r.IsDApp)
	dst = cborAppendHead(dst, cborMajorMap, n)
	dst = cborAppendBytes(dst, scriptBasicInfoRecordKeyPK, r.PK[:])
	if r.ScriptLen != 0 {
		dst = cborAppendUint(dst, scriptBasicInfoRecordKeyScriptLen, uint64(r.ScriptLen))
	}
	if r.LibraryVersion != 0 {
		dst = cborAppendUint(dst, scriptBasicInfoRecordKeyLibraryVersion, uint64(r.LibraryVersion))
	}
	if r.HasVerifier {
		dst = cborAppendBool(dst, scriptBasicInfoRecordKeyHasVerifier, r.HasVerifier)
	}
	if r.IsDApp {
		dst = cborAppendBool(dst, scriptBasicInfoRecordKeyIsDApp, r.IsDApp)
	}
	return dst
}

func (r *scriptBasicInfoRecord) decodeCBOR(data []byte) bool {
	rd := newCBORReader(data)
	v := *r
	for range rd.mapLen(scriptBasicInfoRecordFields) {
		switch rd.key() {
		case scriptBasicInfoRecordKeyPK:
			rd.bytes(v.PK[:])
		case scriptBasicInfoRecordKeyScriptLen:
			v.ScriptLen = rd.uint32()
		case scriptBasicInfoRecordKeyLibraryVersion:
			v.LibraryVersion = ast.LibraryVersion(rd.uint8())
		case scriptBasicInfoRecordKeyHasVerifier:
			v.HasVerifier = rd.bool()
		case scriptBasicInfoRecordKeyIsDApp:
			v.IsDApp = rd.bool()
		default:
			rd.fail()
		}
	}
	if !rd.done() {
		return false
	}
	*r = v
	return true
}

// cborClone returns an exactly sized copy of a record encoded into a stack buffer.
func cborClone(b []byte) []byte {
	res := make([]byte, len(b))
	copy(res, b)
	return res
}
