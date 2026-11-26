package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

// BenchmarkTransactionStorageWrite measures writing of a transaction into the blockchain file buffer and
// of its info into the DB batch.
func BenchmarkTransactionStorageWrite(b *testing.B) {
	for _, bc := range []struct {
		name       string
		attachment []byte
	}{
		{"uncompressed", benchRandBytes(100)},
		{"compressed", make([]byte, 140)},
	} {
		b.Run(bc.name, func(b *testing.B) {
			to := createStorageObjects(b, true)
			tx := benchStorageTransfer(b, bc.attachment)
			blockID := genRandBlockID(b)
			to.prepareAndStartBlock(b, blockID)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := to.rw.writeTransaction(tx, proto.TransactionSucceeded); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkTransactionStorageRead measures reading of a stored transaction by its offset.
func BenchmarkTransactionStorageRead(b *testing.B) {
	for _, bc := range []struct {
		name       string
		attachment []byte
	}{
		{"uncompressed", benchRandBytes(100)},
		{"compressed", make([]byte, 140)},
	} {
		b.Run(bc.name, func(b *testing.B) {
			to := createStorageObjects(b, true)
			tx := benchStorageTransfer(b, bc.attachment)
			blockID := genRandBlockID(b)
			to.prepareAndStartBlock(b, blockID)
			offset := to.rw.blockchainLen
			_, err := to.rw.writeTransaction(tx, proto.TransactionSucceeded)
			require.NoError(b, err)
			// One more transaction, so the read ahead of the first one is not cut by the end of file.
			_, err = to.rw.writeTransaction(benchStorageTransfer(b, benchRandBytes(140)), proto.TransactionSucceeded)
			require.NoError(b, err)
			to.finishBlock(b, blockID)
			to.flush(b)
			b.ReportAllocs()
			for b.Loop() {
				if _, err = to.rw.readTransactionByOffset(offset); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func benchRandBytes(n int) []byte {
	res := make([]byte, n)
	for i := range res {
		res[i] = byte(i*7919 + i>>3*31)
	}
	return res
}

func benchStorageTransfer(b testing.TB, attachment []byte) proto.Transaction {
	sender := testGlobal.senderInfo
	tx := proto.NewUnsignedTransferWithProofs(2, sender.pk, proto.NewOptionalAssetWaves(),
		proto.NewOptionalAssetWaves(), 1, 100500, 100000, testGlobal.recipientInfo.rcp, attachment)
	require.NoError(b, tx.Sign(proto.MainNetScheme, sender.sk))
	return tx
}
