package proto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// binaryTxVectors returns binary representations of all non-protobuf test vectors that can be parsed back.
func binaryTxVectors(tb testing.TB) [][]byte {
	var out [][]byte
	add := func(cases []simpleTxCase) {
		for _, c := range cases {
			if IsProtobufTx(c.tx) || strings.Contains(c.name, "WithProofs/v1/") {
				continue
			}
			b, err := c.tx.MarshalBinary(benchScheme)
			require.NoError(tb, err)
			out = append(out, b)
		}
	}
	add(simpleTxCases(tb))
	add(complexTxCases(tb))
	binarySeeds, _ := fuzzTxSeeds(tb)
	return append(out, binarySeeds...)
}

// checkCanonical asserts that the re-marshalled bytes of a parsed binary transaction are a prefix of the input.
func checkCanonical(t *testing.T, data []byte, tx Transaction) {
	full, err := tx.MarshalBinary(benchScheme)
	require.NoError(t, err)
	require.LessOrEqual(t, len(full), len(data))
	require.Equal(t, data[:len(full)], full, "non-canonical input accepted")
	body, err := tx.BodyMarshalBinary(benchScheme)
	require.NoError(t, err)
	switch tx.(type) {
	case *IssueWithSig, *TransferWithSig, *ReissueWithSig:
		assert.Equal(t, data[1+crypto.SignatureSize:1+crypto.SignatureSize+len(body)], body)
	case *BurnWithSig, *LeaseWithSig, *LeaseCancelWithSig, *CreateAliasWithSig, *MassTransferWithProofs:
		assert.Equal(t, data[:len(body)], body)
	case *Genesis, *Payment, *ExchangeWithSig, *ExchangeWithProofs:
		// Genesis and Payment IDs are not body hashes; Exchange bodies have their own layouts.
	default:
		assert.Equal(t, data[1:1+len(body)], body)
	}
}

// checkIDFromInput asserts that the ID of a successfully parsed binary transaction equals the ID computed from the
// re-marshalled body.
func checkIDFromInput(t *testing.T, tx Transaction) {
	expected, ok := remarshalledID(t, tx)
	if !ok {
		return
	}
	id, err := tx.GetID(benchScheme)
	require.NoError(t, err)
	require.Equal(t, expected[:], id, "%T", tx)
}

// remarshalledID returns the ID computed from the re-marshalled body, the way GenerateID did before the ID was
// derived from the input bytes. It returns false for transactions whose ID is not a hash of the body.
func remarshalledID(t *testing.T, tx Transaction) (crypto.Digest, bool) {
	if _, ok := tx.(*Genesis); ok {
		return crypto.Digest{}, false
	}
	if _, ok := tx.(*Payment); ok {
		return crypto.Digest{}, false
	}
	if IsProtobufTx(tx) {
		body, err := MarshalTxBody(benchScheme, tx)
		require.NoError(t, err)
		return crypto.MustFastHash(body), true
	}
	switch tt := tx.(type) {
	case *CreateAliasWithSig:
		id, err := tt.id()
		require.NoError(t, err)
		return *id, true
	case *CreateAliasWithProofs:
		id, err := tt.id()
		require.NoError(t, err)
		return *id, true
	default:
		body, err := MarshalTxBody(benchScheme, tx)
		require.NoError(t, err)
		return crypto.MustFastHash(body), true
	}
}

func TestTxIDFromInputCanonical(t *testing.T) {
	for i, data := range binaryTxVectors(t) {
		tx, err := BytesToTransaction(data, benchScheme)
		require.NoError(t, err, "vector %d", i)
		checkCanonical(t, data, tx)
		checkIDFromInput(t, tx)
		// Trailing bytes must not change the ID.
		withTail := append(append([]byte(nil), data...), 0xde, 0xad)
		tx2, err := BytesToTransaction(withTail, benchScheme)
		if err == nil {
			id1, idErr := tx.GetID(benchScheme)
			require.NoError(t, idErr)
			id2, idErr := tx2.GetID(benchScheme)
			require.NoError(t, idErr)
			assert.Equal(t, id1, id2, "vector %d", i)
		}
	}
}

func FuzzTxIDFromInput(f *testing.F) {
	for _, s := range binaryTxVectors(f) {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		tx, err := BytesToTransaction(data, benchScheme)
		if err != nil {
			return
		}
		checkIDFromInput(t, tx)
	})
}

// TestLeaseWithProofsReservedByte checks that a non-zero reserved byte, which the parser ignores, does not leak into
// the ID.
func TestLeaseWithProofsReservedByte(t *testing.T) {
	for _, data := range binaryTxVectors(t) {
		if data[0] != 0 || TransactionType(data[1]) != LeaseTransaction {
			continue
		}
		m := append([]byte(nil), data...)
		m[3] = 0x30
		tx, err := BytesToTransaction(m, benchScheme)
		require.NoError(t, err)
		checkIDFromInput(t, tx)
		orig, err := BytesToTransaction(data, benchScheme)
		require.NoError(t, err)
		id1, err := orig.GetID(benchScheme)
		require.NoError(t, err)
		id2, err := tx.GetID(benchScheme)
		require.NoError(t, err)
		assert.Equal(t, id1, id2)
	}
}

// TestUnmarshalBinaryOverwritesID checks that unmarshalling into a transaction with an ID replaces the ID.
func TestUnmarshalBinaryOverwritesID(t *testing.T) {
	for _, data := range binaryTxVectors(t) {
		tx, err := BytesToTransaction(data, benchScheme)
		require.NoError(t, err)
		if _, ok := tx.(*TransferWithProofs); !ok {
			continue
		}
		stale := crypto.Digest{1, 2, 3}
		again := &TransferWithProofs{ID: &stale}
		require.NoError(t, again.UnmarshalBinary(data, benchScheme))
		expected, err := tx.GetID(benchScheme)
		require.NoError(t, err)
		assert.Equal(t, expected, again.ID.Bytes())
	}
}

func TestIDFromInputBodyBounds(t *testing.T) {
	tx := &TransferWithProofs{Type: TransferTransaction, Version: 2}
	_, err := idFromInputBody(tx, benchScheme, []byte{1, 2, 3}, 1, 3)
	assert.Error(t, err)
	_, err = idFromInputBody(tx, benchScheme, []byte{1, 2, 3}, -1, 1)
	assert.Error(t, err)
	_, err = idFromInputBody(tx, benchScheme, []byte{1, 2, 3}, 0, -1)
	assert.Error(t, err)
	id, err := idFromInputBody(tx, benchScheme, []byte{1, 2, 3}, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, crypto.MustFastHash([]byte{2, 3}), *id)
}

func TestProofsV1UnmarshalBinarySingleBacking(t *testing.T) {
	src := &ProofsV1{Version: proofsVersion, Proofs: []B58Bytes{{1, 2, 3}, {}, {4, 5}, {6}}}
	b, err := src.MarshalBinary()
	require.NoError(t, err)
	var p ProofsV1
	require.NoError(t, p.UnmarshalBinary(b))
	assert.Equal(t, src.Proofs, p.Proofs)
	assert.Equal(t, len(src.Proofs), cap(p.Proofs))
	// Proofs must be copies of the input.
	b[5] = 0xff
	assert.Equal(t, B58Bytes{1, 2, 3}, p.Proofs[0])
	// Appending to a proof must not overwrite the next one.
	p.Proofs[2] = append(p.Proofs[2], 0xaa)
	assert.Equal(t, B58Bytes{6}, p.Proofs[3])

	var empty ProofsV1
	require.NoError(t, empty.UnmarshalBinary([]byte{proofsVersion, 0, 0}))
	assert.Empty(t, empty.Proofs)

	// Existing proofs are kept, as before.
	existing := ProofsV1{Proofs: []B58Bytes{{9}}}
	require.NoError(t, existing.UnmarshalBinary([]byte{proofsVersion, 0, 1, 0, 1, 7}))
	assert.Equal(t, []B58Bytes{{9}, {7}}, existing.Proofs)
}

func BenchmarkProofsV1UnmarshalBinary(b *testing.B) {
	sig := make([]byte, crypto.SignatureSize)
	src := &ProofsV1{Version: proofsVersion, Proofs: []B58Bytes{sig, sig, sig}}
	data, err := src.MarshalBinary()
	require.NoError(b, err)
	b.ReportAllocs()
	for b.Loop() {
		var p ProofsV1
		if uErr := p.UnmarshalBinary(data); uErr != nil {
			b.Fatal(uErr)
		}
	}
}

// TestTxIDFromInputMutations is a deterministic complement to FuzzTxIDFromInput: it mutates every byte of every test
// vector and checks that the ID of every accepted mutation equals the ID of its re-marshalled body.
func TestTxIDFromInputMutations(t *testing.T) {
	for _, data := range binaryTxVectors(t) {
		for i := range data {
			for _, v := range []byte{0, 1, 2, 0x30, 0x80, 0xff} {
				if data[i] == v {
					continue
				}
				m := append([]byte(nil), data...)
				m[i] = v
				tx, err := BytesToTransaction(m, benchScheme)
				if err != nil {
					continue
				}
				checkIDFromInput(t, tx)
			}
		}
	}
}

// TestSetScriptWithProofsEmptyPresentScript checks that a script flag of 1 followed by a zero-length script, which
// is marshalled back as an absent script, does not leak into the ID.
func TestSetScriptWithProofsEmptyPresentScript(t *testing.T) {
	sk, pk, err := crypto.GenerateKeyPair([]byte("set-script-empty"))
	require.NoError(t, err)
	tx := NewUnsignedSetScriptWithProofs(1, pk, nil, 1400000, 0x0102030405060100)
	require.NoError(t, tx.Sign(benchScheme, sk))
	canonical, err := tx.MarshalBinary(benchScheme)
	require.NoError(t, err)
	const flagPos = 1 + 3 + crypto.PublicKeySize
	require.Equal(t, byte(0), canonical[flagPos])
	// Replace the absent script flag with a present flag and a zero length.
	m := make([]byte, 0, len(canonical)+2)
	m = append(m, canonical[:flagPos]...)
	m = append(m, 1, 0, 0)
	m = append(m, canonical[flagPos+1:]...)
	parsed := new(SetScriptWithProofs)
	require.NoError(t, parsed.UnmarshalBinary(m, benchScheme))
	checkIDFromInput(t, parsed)
}
