package state

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/ccoveille/go-safecast/v2"
	"github.com/minio/minlz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/bytebufferpool"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

// oldWriteTransactionUncompressed and oldTxBytesToReadWriterTxBytes are the replaced implementations of the
// stored transaction record encoding, kept as the byte-equality oracle.
func oldWriteTransactionUncompressed(txBytes []byte) ([]byte, error) {
	var (
		txBytesTotal     = txBytes
		txBytesTotalSize = len(txBytes) + uint32Size
	)
	if cap(txBytesTotal) < txBytesTotalSize {
		txBytesTotal = make([]byte, txBytesTotalSize)
	}
	txBytesTotal = txBytesTotal[:txBytesTotalSize]
	txSizeU32, err := safecast.Convert[uint32](len(txBytes))
	if err != nil {
		return nil, err
	}
	encodedSize, err := encodeSize(txSizeU32, false)
	if err != nil {
		return nil, err
	}
	copy(txBytesTotal[uint32Size:], txBytes)
	binary.BigEndian.PutUint32(txBytesTotal[:uint32Size], encodedSize)
	return txBytesTotal, nil
}

func oldTxBytesToReadWriterTxBytes(txBytes []byte) ([]byte, error) {
	bb := bytebufferpool.Get()
	defer bytebufferpool.Put(bb)
	bb.B = bb.B[:cap(bb.B)]
	var err error
	bb.B, err = minlz.Encode(bb.B, txBytes, minlz.LevelFastest)
	if err != nil {
		return nil, err
	}
	if len(bb.B) >= len(txBytes) {
		return oldWriteTransactionUncompressed(txBytes)
	}
	txSizeU32, err := safecast.Convert[uint32](len(bb.B))
	if err != nil {
		return nil, err
	}
	encodedSize, err := encodeSize(txSizeU32, true)
	if err != nil {
		return nil, err
	}
	txBytesTotal := binary.BigEndian.AppendUint32(txBytes[:0], encodedSize)
	txBytesTotal = append(txBytesTotal, bb.B...)
	return txBytesTotal, nil
}

func randBytes(t testing.TB, n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return b
}

func storageIOTestTransfer(t testing.TB, version byte, attachment []byte, ts uint64) proto.Transaction {
	sender := testGlobal.senderInfo
	tx := proto.NewUnsignedTransferWithProofs(version, sender.pk, proto.NewOptionalAssetWaves(),
		proto.NewOptionalAssetWaves(), ts, 100500, 100000, testGlobal.recipientInfo.rcp, attachment)
	require.NoError(t, tx.Sign(proto.MainNetScheme, sender.sk))
	return tx
}

func storageIOTestData(t testing.TB, version byte, value []byte, ts uint64) proto.Transaction {
	sender := testGlobal.senderInfo
	tx := proto.NewUnsignedDataWithProofs(version, sender.pk, 100000, ts)
	require.NoError(t, tx.AppendEntry(&proto.BinaryDataEntry{Key: "binary", Value: value}))
	require.NoError(t, tx.AppendEntry(&proto.StringDataEntry{Key: "string", Value: "value"}))
	require.NoError(t, tx.AppendEntry(&proto.IntegerDataEntry{Key: "integer", Value: math.MinInt64}))
	require.NoError(t, tx.Sign(proto.MainNetScheme, sender.sk))
	return tx
}

// storageIOTestTxs returns transactions covering compressed and uncompressed records, records shorter and
// longer than the read ahead size; the last record is short, so its read ahead goes past the end of file.
func storageIOTestTxs(t testing.TB, protobuf bool) []proto.Transaction {
	transferVersion, dataVersion := byte(2), byte(1)
	if protobuf {
		transferVersion, dataVersion = 3, 2
	}
	return []proto.Transaction{
		storageIOTestTransfer(t, transferVersion, randBytes(t, 100), 1),                            // uncompressed, short
		storageIOTestTransfer(t, transferVersion, make([]byte, 140), 2),                            // compressed, short
		storageIOTestData(t, dataVersion, randBytes(t, 3*txReadAheadSize), 3),                      // uncompressed, long
		storageIOTestData(t, dataVersion, make([]byte, 30*txReadAheadSize), 4),                     // compressed, long
		storageIOTestData(t, dataVersion, bytes.Repeat(randBytes(t, 64), 3*txReadAheadSize/64), 5), // compressed, long
		storageIOTestTransfer(t, transferVersion, randBytes(t, 10), 6),                             // uncompressed, last
	}
}

func marshalForStorage(t testing.TB, tx proto.Transaction, protobuf bool) []byte {
	var (
		b   []byte
		err error
	)
	if protobuf {
		b, err = tx.MarshalSignedToProtobuf(proto.MainNetScheme)
	} else {
		b, err = tx.MarshalBinary(proto.MainNetScheme)
	}
	require.NoError(t, err)
	return b
}

func storageIOTestStatus(i int) proto.TransactionStatus {
	if i%2 == 1 {
		return proto.TransactionFailed
	}
	return proto.TransactionSucceeded
}

// writeStorageIOTestTxs writes the transactions in a new block and returns the expected stored bytes,
// produced by the old encoder.
func writeStorageIOTestTxs(t *testing.T, to *testStorageObjects, txs []proto.Transaction, protobuf bool) []byte {
	blockID := genRandBlockID(t)
	to.prepareAndStartBlock(t, blockID)
	var (
		want            []byte
		compressedCount int
	)
	for i, tx := range txs {
		n, err := to.rw.writeTransaction(tx, storageIOTestStatus(i))
		require.NoError(t, err)
		rec, err := oldTxBytesToReadWriterTxBytes(marshalForStorage(t, tx, protobuf))
		require.NoError(t, err)
		assert.Equal(t, len(rec), n)
		if _, compressed := decodeSize(binary.BigEndian.Uint32(rec)); compressed {
			compressedCount++
		}
		want = append(want, rec...)
	}
	require.Positive(t, compressedCount)
	require.Less(t, compressedCount, len(txs))
	to.finishBlock(t, blockID)
	to.flush(t)
	return want
}

func TestTransactionStorageRoundTrip(t *testing.T) {
	for _, protobuf := range []bool{false, true} {
		t.Run(fmt.Sprintf("protobuf=%v", protobuf), func(t *testing.T) {
			to := createStorageObjects(t, true)
			if protobuf {
				to.rw.setProtobufActivated()
			}
			txs := storageIOTestTxs(t, protobuf)
			start := to.rw.blockchainLen
			want := writeStorageIOTestTxs(t, to, txs, protobuf)

			// The stored bytes are exactly the bytes of the old encoder.
			got := make([]byte, to.rw.blockchainLen-start)
			n, err := to.rw.blockchain.ReadAt(got, int64(start))
			require.NoError(t, err)
			require.Equal(t, len(got), n)
			assert.Equal(t, want, got)

			// All transactions are read before they are compared, so the decoded transactions must not refer to
			// the pooled read buffers, which are reused by the later reads.
			read := make([]proto.Transaction, len(txs))
			for i, tx := range txs {
				id, idErr := tx.GetID(proto.MainNetScheme)
				require.NoError(t, idErr)
				res, status, rErr := to.rw.readTransaction(id)
				require.NoError(t, rErr)
				assert.Equal(t, storageIOTestStatus(i), status)
				read[i] = res
			}
			for i, tx := range txs {
				id, idErr := tx.GetID(proto.MainNetScheme)
				require.NoError(t, idErr)
				resID, idErr := read[i].GetID(proto.MainNetScheme)
				require.NoError(t, idErr)
				assert.Equal(t, id, resID)
				assert.Equal(t, marshalForStorage(t, tx, protobuf), marshalForStorage(t, read[i], protobuf))
			}
			// The block read path decodes the same records.
			blockTxs, err := unmarshalTransactions(got, len(txs), protobuf, proto.MainNetScheme)
			require.NoError(t, err)
			require.Len(t, blockTxs, len(txs))
			for i, tx := range txs {
				assert.Equal(t, marshalForStorage(t, tx, protobuf), marshalForStorage(t, blockTxs[i], protobuf))
			}
			// Rollback reads all transactions of the block by offsets.
			require.NoError(t, to.rw.rollback(0))
			to.flush(t)
			id, err := txs[0].GetID(proto.MainNetScheme)
			require.NoError(t, err)
			_, _, err = to.rw.readTransaction(id)
			assert.Error(t, err)
		})
	}
}

func TestReadTransactionByOffsetErrors(t *testing.T) {
	to := createStorageObjects(t, true)
	tx := storageIOTestTransfer(t, 2, randBytes(t, 10), 1)
	blockID := genRandBlockID(t)
	to.prepareAndStartBlock(t, blockID)
	n, err := to.rw.writeTransaction(tx, proto.TransactionSucceeded)
	require.NoError(t, err)
	to.finishBlock(t, blockID)
	to.flush(t)
	end := to.rw.blockchainLen
	start := end - uint64(n)

	_, err = to.rw.readTransactionByOffset(start)
	require.NoError(t, err)
	_, err = to.rw.readTransactionByOffset(end) // nothing to read, as with the old implementation
	assert.ErrorIs(t, err, io.EOF)
	_, err = to.rw.readTransactionByOffset(end - 2) // partial size header
	assert.Error(t, err)
	_, err = to.rw.readTransactionByOffset(start + uint32Size) // garbage size, past the end of file
	assert.Error(t, err)
}

func TestDataEntryRecordRoundTrip(t *testing.T) {
	entries := []proto.DataEntry{
		&proto.IntegerDataEntry{Key: "integer", Value: math.MaxInt64},
		&proto.IntegerDataEntry{Key: "negative", Value: math.MinInt64},
		&proto.BooleanDataEntry{Key: "true", Value: true},
		&proto.BooleanDataEntry{Key: "false", Value: false},
		&proto.BinaryDataEntry{Key: "binary", Value: randBytes(t, 100)},
		&proto.BinaryDataEntry{Key: "empty-binary", Value: []byte{}},
		&proto.StringDataEntry{Key: "string", Value: strings.Repeat("s", 1000)},
		&proto.StringDataEntry{Key: "empty-string", Value: ""},
		&proto.DeleteDataEntry{Key: "delete"},
	}
	for _, e := range entries {
		t.Run(e.GetKey(), func(t *testing.T) {
			valueBytes, err := e.MarshalValue()
			require.NoError(t, err)
			want := bytes.Clone(valueBytes)
			record := dataEntryRecord{value: valueBytes}
			recordBytes, err := record.marshalBinary()
			require.NoError(t, err)
			assert.Equal(t, want, recordBytes) // the record bytes are the value bytes
			var r dataEntryRecord
			require.NoError(t, r.unmarshalBinary(recordBytes))
			decoded, err := proto.NewDataEntryFromValueBytes(r.value)
			require.NoError(t, err)
			// The decoded entry must not alias the record bytes.
			for i := range recordBytes {
				recordBytes[i] ^= 0xff
			}
			decoded.SetKey(e.GetKey())
			assert.Equal(t, e, decoded)
		})
	}

	to := createAccountsDataStorage(t, true)
	addr := testGlobal.senderInfo.addr
	to.stor.addBlock(t, blockID0)
	for _, e := range entries {
		require.NoError(t, to.accountsDataStor.appendEntry(addr, e, blockID0))
	}
	check := func(newest bool) {
		for _, e := range entries {
			var (
				res proto.DataEntry
				err error
			)
			if newest {
				res, err = to.accountsDataStor.retrieveNewestEntry(addr, e.GetKey())
			} else {
				res, err = to.accountsDataStor.retrieveEntry(addr, e.GetKey())
			}
			if e.GetValueType() == proto.DataDelete {
				assert.Error(t, err)
				continue
			}
			require.NoError(t, err)
			assert.Equal(t, e, res)
		}
	}
	check(true)
	to.stor.flush(t)
	check(false)
	check(true)
	stored, err := to.accountsDataStor.retrieveEntries(addr)
	require.NoError(t, err)
	assert.ElementsMatch(t, entries[:len(entries)-1], stored)

	intEntry, err := to.accountsDataStor.retrieveIntegerEntry(addr, "integer")
	require.NoError(t, err)
	assert.Equal(t, entries[0], intEntry)
	boolEntry, err := to.accountsDataStor.retrieveNewestBooleanEntry(addr, "true")
	require.NoError(t, err)
	assert.Equal(t, entries[2], boolEntry)
	binEntry, err := to.accountsDataStor.retrieveBinaryEntry(addr, "binary")
	require.NoError(t, err)
	assert.Equal(t, entries[4], binEntry)
	strEntry, err := to.accountsDataStor.retrieveNewestStringEntry(addr, "string")
	require.NoError(t, err)
	assert.Equal(t, entries[6], strEntry)
}
