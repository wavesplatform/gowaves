package proto

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

// Regression tests for panics on malformed input in deserializers.

func TestProofsV1UnmarshalBinaryMalformed(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"Empty", nil},
		{"OnlyVersion", []byte{1}},
		{"WrongVersion", []byte{2, 0, 0}},
		{"MissingProofLength", []byte{1, 0, 1}},
		{"HalfOfProofLength", []byte{1, 0, 1, 0}},
		{"TruncatedProof", []byte{1, 0, 1, 0, 5, 1, 2}},
		{"MissingSecondProofLength", []byte{1, 0, 2, 0, 1, 7}},
		{"TruncatedSecondProof", []byte{1, 0, 2, 0, 1, 7, 0, 2, 1}},
		{"HugeProofLength", []byte{1, 0, 1, 0xff, 0xff, 1, 2, 3}},
		{"TooManyProofs", []byte{1, 0xff, 0xff}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p ProofsV1
			assert.NotPanics(t, func() {
				assert.Error(t, p.UnmarshalBinary(tc.data))
			})
		})
	}
	t.Run("Valid", func(t *testing.T) {
		var p ProofsV1
		require.NoError(t, p.UnmarshalBinary([]byte{1, 0, 2, 0, 1, 7, 0, 2, 8, 9}))
		assert.Equal(t, []B58Bytes{{7}, {8, 9}}, p.Proofs)
	})
}

// dataBody builds the body of DataWithProofs transaction (without the leading zero byte) with the given raw entries
// and the given number of tail bytes.
func dataBody(count uint16, entries []byte, tail int) []byte {
	buf := []byte{byte(DataTransaction), 1}
	buf = append(buf, make([]byte, crypto.PublicKeySize)...)
	buf = binary.BigEndian.AppendUint16(buf, count)
	buf = append(buf, entries...)
	return append(buf, make([]byte, tail)...)
}

func TestDataWithProofsBodyUnmarshalBinaryMalformed(t *testing.T) {
	intEntry := []byte{0, 1, 'k', byte(DataInteger), 0, 0, 0, 0, 0, 0, 0, 1}
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"Empty", nil},
		{"OneByte", []byte{byte(DataTransaction)}},
		{"TwoBytes", []byte{byte(DataTransaction), 1}},
		{"UnknownValueType", dataBody(1, []byte{0, 1, 'k', 9, 0, 0, 0, 0, 0, 0, 0, 0}, 16)},
		{"DeleteValueType", dataBody(1, []byte{0, 1, 'k', byte(DataDelete)}, 16)},
		{"TruncatedTimestampAndFee", dataBody(1, intEntry, 4)},
		{"NoTimestampAndFee", dataBody(1, intEntry, 0)},
		{"MoreEntriesThanData", dataBody(3, intEntry, 16)},
		{"HugeKeyLength", dataBody(1, []byte{0xff, 0xff, 'k'}, 16)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var tx DataWithProofs
			assert.NotPanics(t, func() {
				assert.Error(t, tx.bodyUnmarshalBinary(tc.data))
			})
			// The same input wrapped as a full transaction.
			assert.NotPanics(t, func() {
				_, err := BytesToTransaction(append([]byte{0}, tc.data...), TestNetScheme)
				assert.Error(t, err)
			})
		})
	}
	t.Run("Valid", func(t *testing.T) {
		var tx DataWithProofs
		require.NoError(t, tx.bodyUnmarshalBinary(dataBody(1, intEntry, 16)))
		require.Len(t, tx.Entries, 1)
		assert.Equal(t, &IntegerDataEntry{Key: "k", Value: 1}, tx.Entries[0])
	})
}

func TestNewDataEntryFromValueBytesMalformed(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"Empty", nil},
		{"UnknownType", []byte{9}},
		{"TruncatedInteger", []byte{byte(DataInteger), 1, 2}},
		{"TruncatedBoolean", []byte{byte(DataBoolean)}},
		{"TruncatedBinary", []byte{byte(DataBinary), 0, 5, 1}},
		{"TruncatedString", []byte{byte(DataString), 0, 5, 'a'}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				e, err := NewDataEntryFromValueBytes(tc.data)
				assert.Error(t, err)
				assert.Nil(t, e)
			})
		})
	}
}

func TestArgumentsUnmarshalBinaryMalformed(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"Empty", nil},
		{"TruncatedCount", []byte{0, 0, 1}},
		{"MissingArgument", []byte{0, 0, 0, 1}},
		{"MissingSecondArgument", []byte{0, 0, 0, 2, byte(ArgumentValueTrue)}},
		{"UnknownArgumentType", []byte{0, 0, 0, 1, 0x77}},
		{"HugeCount", []byte{0xff, 0xff, 0xff, 0xff, byte(ArgumentValueTrue)}},
		{"TruncatedInteger", []byte{0, 0, 0, 1, byte(ArgumentInteger), 1}},
		{"TruncatedBinary", []byte{0, 0, 0, 1, byte(ArgumentBinary), 0, 0, 0, 9, 1}},
		{"TruncatedString", []byte{0, 0, 0, 1, byte(ArgumentString), 0xff, 0xff, 0xff, 0xff, 'a'}},
		{"TruncatedList", []byte{0, 0, 0, 1, byte(ArgumentList), 0, 0, 0, 1}},
		{"ListWithUnknownType", []byte{0, 0, 0, 1, byte(ArgumentList), 0, 0, 0, 1, 0x77}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var a Arguments
			assert.NotPanics(t, func() {
				assert.Error(t, a.UnmarshalBinary(tc.data))
			})
		})
	}
	t.Run("Valid", func(t *testing.T) {
		var a Arguments
		data := []byte{0, 0, 0, 2, byte(ArgumentValueTrue), byte(ArgumentList), 0, 0, 0, 1, byte(ArgumentValueFalse)}
		require.NoError(t, a.UnmarshalBinary(data))
		expected := Arguments{
			&BooleanArgument{Value: true},
			&ListArgument{Items: Arguments{&BooleanArgument{Value: false}}},
		}
		assert.Equal(t, expected, a)
	})
}

func TestFunctionCallUnmarshalBinaryMalformed(t *testing.T) {
	head := []byte{1, tokenFunctionCall, tokenUserFunction, 0, 0, 0, 1, 'f'}
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"MissingArguments", head},
		{"UnknownArgumentType", append(append([]byte{}, head...), 0, 0, 0, 1, 0x77)},
		{"MissingArgument", append(append([]byte{}, head...), 0, 0, 0, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var fc FunctionCall
			assert.NotPanics(t, func() {
				assert.Error(t, fc.UnmarshalBinary(tc.data))
			})
		})
	}
}

func TestProtobufConverterSignedTransactionMalformed(t *testing.T) {
	for _, tc := range []struct {
		name string
		stx  *g.SignedTransaction
	}{
		{"Nil", nil},
		{"NilVariant", &g.SignedTransaction{}},
		{"NilWavesTransaction", &g.SignedTransaction{
			Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: nil},
		}},
		{"NilTransactionData", &g.SignedTransaction{
			Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: &g.Transaction{}},
		}},
		{"InvalidEthereumTransaction", &g.SignedTransaction{
			Transaction: &g.SignedTransaction_EthereumTransaction{EthereumTransaction: []byte{1, 2, 3}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := ProtobufConverter{FallbackChainID: TestNetScheme}
			assert.NotPanics(t, func() {
				tx, err := c.SignedTransaction(tc.stx)
				assert.Error(t, err)
				assert.Nil(t, tx)
			})
		})
	}
	t.Run("SignedTxFromProtobuf", func(t *testing.T) {
		for _, stx := range []*g.SignedTransaction{
			{},
			{Proofs: [][]byte{{1, 2, 3}}},
		} {
			data, err := stx.MarshalVTStrict()
			require.NoError(t, err)
			assert.NotPanics(t, func() {
				_, err = SignedTxFromProtobuf(data)
				assert.Error(t, err)
			})
		}
	})
}

// fuzzTxSeeds returns binary and protobuf representations of the benchmark fixtures.
func fuzzTxSeeds(tb testing.TB) ([][]byte, [][]byte) {
	var binarySeeds, protobufSeeds [][]byte
	for _, f := range benchTxFixtures(tb) {
		if !IsProtobufTx(f.tx) {
			b, err := f.tx.MarshalBinary(benchScheme)
			require.NoError(tb, err)
			binarySeeds = append(binarySeeds, b)
		}
		pb, err := f.tx.MarshalSignedToProtobuf(benchScheme)
		require.NoError(tb, err)
		protobufSeeds = append(protobufSeeds, pb)
	}
	return binarySeeds, protobufSeeds
}

func FuzzProofsUnmarshal(f *testing.F) {
	binarySeeds, _ := fuzzTxSeeds(f)
	for _, s := range binarySeeds {
		// Proofs are the tail of a binary transaction, so every suffix of the seed that starts with the proofs
		// version byte is a good seed.
		for i := len(s) - 1; i >= 0 && i > len(s)-200; i-- {
			if s[i] == proofsVersion {
				f.Add(s[i:])
			}
		}
	}
	f.Add([]byte{1, 0, 1, 0, 5, 1, 2})
	f.Fuzz(func(t *testing.T, data []byte) {
		var p ProofsV1
		if err := p.UnmarshalBinary(data); err != nil {
			return
		}
		b, err := p.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, data[:len(b)], b)
	})
}

func FuzzBytesToTransaction(f *testing.F) {
	binarySeeds, _ := fuzzTxSeeds(f)
	for _, s := range binarySeeds {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = BytesToTransaction(data, benchScheme)
	})
}

func FuzzSignedTxFromProtobuf(f *testing.F) {
	_, protobufSeeds := fuzzTxSeeds(f)
	for _, s := range protobufSeeds {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = SignedTxFromProtobuf(data)
	})
}

// TestBytesToTransactionTruncated checks that every prefix of every valid binary transaction is rejected with an
// error, without panics.
func TestBytesToTransactionTruncated(t *testing.T) {
	binarySeeds, _ := fuzzTxSeeds(t)
	for _, s := range binarySeeds {
		for i := range len(s) {
			assert.NotPanics(t, func() {
				_, err := BytesToTransaction(s[:i], benchScheme)
				assert.Error(t, err, "prefix of length %d of %d", i, len(s))
			})
		}
		_, err := BytesToTransaction(s, benchScheme)
		require.NoError(t, err)
	}
}
