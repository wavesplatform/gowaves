package proto

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// Benchmarks for block, microblock, snapshot and network message serialization.
// Fixtures are checked once before the benchmark loop, so they also work as smoke tests with `-benchtime=1x`.

const (
	benchBlockTxs      = 100
	benchMicroBlockTxs = 10
	benchSnapshotTxs   = 100
)

func benchLiquidSizes() []int {
	return []int{10, 100, 1000}
}

func benchTransfers(tb testing.TB, v byte, kp KeyPair, n int, tsOffset uint64) Transactions {
	asset := NewOptionalAssetFromDigest(benchDigest(1))
	rcp := benchRecipient(tb, "recipient")
	txs := make(Transactions, n)
	for i := range txs {
		tx := NewUnsignedTransferWithProofs(v, kp.Public, *asset, NewOptionalAssetWaves(),
			benchTimestamp+tsOffset+uint64(i), 100_000, 100_000, rcp, Attachment("benchmark attachment"))
		benchSign(tb, tx, kp)
		txs[i] = tx
	}
	return txs
}

func benchConsensus(genSigSize int) NxtConsensus {
	genSig := make([]byte, genSigSize)
	for i := range genSig {
		genSig[i] = byte(i)
	}
	return NxtConsensus{BaseTarget: 123_456, GenSignature: genSig}
}

func benchCreateBlock(tb testing.TB, version BlockVersion, txs Transactions, kp KeyPair) *Block {
	var (
		parent     BlockID
		genSigSize int
		features   []int16
	)
	if version >= ProtobufBlockVersion {
		parent = NewBlockIDFromDigest(benchDigest(7))
		genSigSize = 96
		features = []int16{14, 15}
	} else {
		var sig crypto.Signature
		copy(sig[:], []byte("benchmark parent block signature"))
		parent = NewBlockIDFromSignature(sig)
		genSigSize = crypto.DigestSize
		features = []int16{}
	}
	b, err := CreateBlock(txs, benchTimestamp, parent, kp.Public, benchConsensus(genSigSize), version, features,
		600_000_000, benchScheme, nil, nil)
	require.NoError(tb, err)
	require.NoError(tb, b.Sign(benchScheme, kp.Secret))
	require.NoError(tb, b.GenerateBlockID(benchScheme))
	return b
}

func benchLegacyBlock(tb testing.TB, n int) *Block {
	kp := benchKeyPair(tb, "benchmark generator")
	return benchCreateBlock(tb, RewardBlockVersion, benchTransfers(tb, 2, kp, n, 0), kp)
}

func benchProtobufBlock(tb testing.TB, n int) *Block {
	kp := benchKeyPair(tb, "benchmark generator")
	return benchCreateBlock(tb, ProtobufBlockVersion, benchTransfers(tb, 3, kp, n, 0), kp)
}

func benchMicroBlock(tb testing.TB, ref *Block, n int) *MicroBlock {
	kp := benchKeyPair(tb, "benchmark generator")
	txs := benchTransfers(tb, 3, kp, n, uint64(ref.TransactionCount))
	var totalSig crypto.Signature
	copy(totalSig[:], []byte("benchmark total resulting block signature"))
	mb := &MicroBlock{
		VersionField:          ref.Version,
		Reference:             ref.BlockID(),
		TotalResBlockSigField: totalSig,
		TotalBlockID:          NewBlockIDFromDigest(benchDigest(9)),
		TransactionCount:      uint32(n), // #nosec G115: benchmark sizes are small.
		Transactions:          txs,
		SenderPK:              kp.Public,
	}
	require.NoError(tb, mb.Sign(benchScheme, kp.Secret))
	return mb
}

func BenchmarkBlockMarshalBinary(b *testing.B) {
	blk := benchLegacyBlock(b, benchBlockTxs)
	_, err := blk.MarshalBinary(benchScheme)
	require.NoError(b, err)
	b.ReportAllocs()
	for b.Loop() {
		if _, err = blk.MarshalBinary(benchScheme); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBlockUnmarshalBinary(b *testing.B) {
	blk := benchLegacyBlock(b, benchBlockTxs)
	data, err := blk.MarshalBinary(benchScheme)
	require.NoError(b, err)
	var check Block
	require.NoError(b, check.UnmarshalBinary(data, benchScheme))
	require.Equal(b, blk.BlockID(), check.BlockID())
	b.ReportAllocs()
	for b.Loop() {
		var res Block
		if err = res.UnmarshalBinary(data, benchScheme); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBlockMarshalToProtobuf(b *testing.B) {
	blk := benchProtobufBlock(b, benchBlockTxs)
	_, err := blk.MarshalToProtobuf(benchScheme)
	require.NoError(b, err)
	b.ReportAllocs()
	for b.Loop() {
		if _, err = blk.MarshalToProtobuf(benchScheme); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBlockUnmarshalFromProtobuf(b *testing.B) {
	blk := benchProtobufBlock(b, benchBlockTxs)
	data, err := blk.MarshalToProtobuf(benchScheme)
	require.NoError(b, err)
	var check Block
	require.NoError(b, check.UnmarshalFromProtobuf(data))
	require.Equal(b, blk.TransactionCount, check.TransactionCount)
	b.ReportAllocs()
	for b.Loop() {
		var res Block
		if err = res.UnmarshalFromProtobuf(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBlockVerifyTransactionsRoot(b *testing.B) {
	for _, n := range benchLiquidSizes() {
		b.Run(fmt.Sprintf("txs=%d", n), func(b *testing.B) {
			blk := benchProtobufBlock(b, n)
			ok, err := blk.VerifyTransactionsRoot(benchScheme)
			require.NoError(b, err)
			require.True(b, ok)
			b.ReportAllocs()
			for b.Loop() {
				if _, err = blk.VerifyTransactionsRoot(benchScheme); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkAppendMicroBlock measures appending a microblock of a fixed size to a liquid block of a growing size.
func BenchmarkAppendMicroBlock(b *testing.B) {
	for _, n := range benchLiquidSizes() {
		b.Run(fmt.Sprintf("liquid=%d", n), func(b *testing.B) {
			blk := benchProtobufBlock(b, n)
			mb := benchMicroBlock(b, blk, benchMicroBlockTxs)
			res, err := AppendMicroBlock(blk, mb, benchScheme)
			require.NoError(b, err)
			require.Equal(b, n+benchMicroBlockTxs, res.TransactionCount)
			b.ReportAllocs()
			for b.Loop() {
				if _, err = AppendMicroBlock(blk, mb, benchScheme); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMicroBlockVerifySignature(b *testing.B) {
	mb := benchMicroBlock(b, benchProtobufBlock(b, 1), benchBlockTxs)
	ok, err := mb.VerifySignature(benchScheme)
	require.NoError(b, err)
	require.True(b, ok)
	b.ReportAllocs()
	for b.Loop() {
		if _, err = mb.VerifySignature(benchScheme); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMicroBlockMarshalBinary(b *testing.B) {
	mb := benchMicroBlock(b, benchProtobufBlock(b, 1), benchBlockTxs)
	_, err := mb.MarshalBinary(benchScheme)
	require.NoError(b, err)
	b.ReportAllocs()
	for b.Loop() {
		if _, err = mb.MarshalBinary(benchScheme); err != nil {
			b.Fatal(err)
		}
	}
}

func benchBlockSnapshot(tb testing.TB, n int) BlockSnapshot {
	rcp := benchRecipient(tb, "snapshot address")
	addr := *rcp.Address()
	var bs BlockSnapshot
	for i := range n {
		bs.AppendTxSnapshot([]AtomicSnapshot{
			&TransactionStatusSnapshot{Status: TransactionSucceeded},
			&WavesBalanceSnapshot{Address: addr, Balance: uint64(i) * 1_000},
			&AssetBalanceSnapshot{Address: addr, AssetID: benchDigest(byte(i)), Balance: uint64(i) * 2_000},
			&DataEntriesSnapshot{Address: addr, DataEntries: DataEntries{
				&IntegerDataEntry{Key: fmt.Sprintf("int-%d", i), Value: int64(i)},
				&StringDataEntry{Key: fmt.Sprintf("str-%d", i), Value: "value"},
			}},
		})
	}
	return bs
}

func BenchmarkBlockSnapshotMarshallBinary(b *testing.B) {
	bs := benchBlockSnapshot(b, benchSnapshotTxs)
	_, err := bs.MarshallBinary()
	require.NoError(b, err)
	b.ReportAllocs()
	for b.Loop() {
		if _, err = bs.MarshallBinary(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBlockSnapshotUnmarshalBinary(b *testing.B) {
	bs := benchBlockSnapshot(b, benchSnapshotTxs)
	data, err := bs.MarshallBinary()
	require.NoError(b, err)
	var check BlockSnapshot
	require.NoError(b, check.UnmarshalBinary(data, benchScheme))
	require.Len(b, check.TxSnapshots, benchSnapshotTxs)
	b.ReportAllocs()
	for b.Loop() {
		var res BlockSnapshot
		if err = res.UnmarshalBinary(data, benchScheme); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTransactionMessageMarshalBinary(b *testing.B) {
	kp := benchKeyPair(b, "benchmark sender")
	data, err := benchTransfer(b, 2, kp).MarshalBinary(benchScheme)
	require.NoError(b, err)
	m := &TransactionMessage{Transaction: data}
	_, err = m.MarshalBinary()
	require.NoError(b, err)
	b.ReportAllocs()
	for b.Loop() {
		if _, err = m.MarshalBinary(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPBTransactionMessageMarshalBinary(b *testing.B) {
	kp := benchKeyPair(b, "benchmark sender")
	data, err := benchTransfer(b, 3, kp).MarshalSignedToProtobuf(benchScheme)
	require.NoError(b, err)
	m := &PBTransactionMessage{Transaction: data}
	_, err = m.MarshalBinary()
	require.NoError(b, err)
	b.ReportAllocs()
	for b.Loop() {
		if _, err = m.MarshalBinary(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMicroBlockMessageMarshalBinary(b *testing.B) {
	mb := benchMicroBlock(b, benchProtobufBlock(b, 1), benchBlockTxs)
	data, err := mb.MarshalBinary(benchScheme)
	require.NoError(b, err)
	m := &MicroBlockMessage{Body: data}
	_, err = m.MarshalBinary()
	require.NoError(b, err)
	b.ReportAllocs()
	for b.Loop() {
		if _, err = m.MarshalBinary(); err != nil {
			b.Fatal(err)
		}
	}
}
