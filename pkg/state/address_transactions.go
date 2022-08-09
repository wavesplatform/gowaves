package state

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/pkg/errors"
	"github.com/starius/emsort"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	// AddressID size + length of block num + transaction offset length.
	addrTxRecordSize = proto.AddressIDSize + blockNumLen + txMetaSize

	maxEmsortMem = 200 * 1024 * 1024 // 200 MiB.

	txMetaSize = 8 + 1
)

var (
	fileSizeKeyBytes = []byte{txsByAddressesFileSizeKeyPrefix}
)

type txMeta struct {
	offset uint64
	failed bool
}

func (m *txMeta) bytes() []byte {
	buf := make([]byte, txMetaSize)
	binary.BigEndian.PutUint64(buf, m.offset)
	buf[8] = 0
	if m.failed {
		buf[8] = 1
	}
	return buf
}

func (m *txMeta) unmarshal(data []byte) error {
	if len(data) < txMetaSize {
		return errInvalidDataSize
	}
	m.offset = binary.BigEndian.Uint64(data)
	if data[8] == 1 {
		m.failed = true
	}
	return nil
}

type txIter struct {
	rw   *blockReadWriter
	iter *recordIterator
	err  error
}

func newTxIter(rw *blockReadWriter, iter *recordIterator) *txIter {
	return &txIter{rw: rw, iter: iter}
}

func (i *txIter) Transaction() (proto.Transaction, bool, error) {
	value, err := i.iter.currentRecord()
	if err != nil {
		return nil, false, err
	}
	var meta txMeta
	err = meta.unmarshal(value)
	if err != nil {
		return nil, false, err
	}
	tx, err := i.rw.readTransactionByOffset(meta.offset)
	if err != nil {
		return nil, false, err
	}
	return tx, meta.failed, nil
}

func (i *txIter) Next() bool {
	return i.iter.next()
}

func (i *txIter) Error() error {
	if err := i.iter.error(); err != nil {
		return err
	}
	return i.err
}

func (i *txIter) Release() {
	i.iter.release()
}

func manageFile(file *os.File, db keyvalue.IterableKeyVal) error {
	var properFileSize uint64
	fileSizeBytes, err := db.Get(fileSizeKeyBytes)
	if err == keyvalue.ErrNotFound {
		properFileSize = 0
	} else if err == nil {
		properFileSize = binary.BigEndian.Uint64(fileSizeBytes)
	} else {
		return err
	}

	fileStats, err := os.Stat(file.Name())
	if err != nil {
		return err
	}
	size := uint64(fileStats.Size())

	if size < properFileSize {
		return errors.New("data loss: file size is less than it should be")
	} else if size == properFileSize {
		return nil
	}
	if err := file.Truncate(int64(properFileSize)); err != nil {
		return err
	}
	if _, err := file.Seek(int64(properFileSize), 0); err != nil {
		return err
	}
	return nil
}

type addressTransactionsParams struct {
	dir                 string // Directory for address_transactions file.
	batchedStorMemLimit int    // Maximum size of batchedStor db batch.
	batchedStorMaxKeys  int    // Maximum number of keys per flush().
	maxFileSize         int64  // Maximum size of address_transactions file.
	providesData        bool   // True if transaction iterators can be used.
}

type addressTransactions struct {
	stateDB             *stateDB
	rw                  *blockReadWriter
	stor                *batchedStorage
	amend               bool
	filePath            string
	addrTransactions    *os.File
	addrTransactionsBuf *bufio.Writer

	params *addressTransactionsParams
}

func newAddressTransactions(
	db keyvalue.IterableKeyVal,
	stateDB *stateDB,
	rw *blockReadWriter,
	params *addressTransactionsParams,
	amend bool,
) (*addressTransactions, error) {
	bsParams := &batchedStorParams{
		maxBatchSize: maxTransactionIdsBatchSize,
		recordSize:   txMetaSize,
		prefix:       transactionIdsPrefix,
	}
	filePath := filepath.Join(filepath.Clean(params.dir), "address_transactions")
	addrTransactionsFile, _, err := openOrCreateForAppending(filePath)
	if err != nil {
		return nil, err
	}
	if err := manageFile(addrTransactionsFile, db); err != nil {
		return nil, err
	}
	stor, err := newBatchedStorage(db, stateDB, bsParams, params.batchedStorMemLimit, params.batchedStorMaxKeys, amend)
	if err != nil {
		return nil, err
	}
	atx := &addressTransactions{
		stateDB:             stateDB,
		rw:                  rw,
		stor:                stor,
		filePath:            filePath,
		addrTransactions:    addrTransactionsFile,
		addrTransactionsBuf: bufio.NewWriter(addrTransactionsFile),
		params:              params,
		amend:               amend,
	}
	if params.providesData {
		if err := atx.persist(); err != nil {
			return nil, errors.Wrap(err, "failed to persist")
		}
	}
	return atx, nil
}

func (at *addressTransactions) saveTxIdByAddress(addr proto.Address, txID []byte, blockID proto.BlockID) error {
	if at.rw.offsetLen != 8 {
		return errors.New("unsupported meta length")
	}
	newRecord := make([]byte, addrTxRecordSize)
	blockNum, err := at.stateDB.newestBlockIdToNum(blockID)
	if err != nil {
		return err
	}
	copy(newRecord[:proto.AddressIDSize], addr.ID().Bytes())
	pos := proto.AddressIDSize
	info, err := at.rw.newestTransactionInfoByID(txID)
	if err != nil {
		return err
	}
	meta := txMeta{info.offset, info.failed}
	binary.BigEndian.PutUint32(newRecord[pos:], blockNum)
	pos += blockNumLen
	copy(newRecord[pos:], meta.bytes())
	if at.params.providesData {
		return at.stor.addRecordBytes(newRecord[:proto.AddressIDSize], newRecord[proto.AddressIDSize:])
	}
	if _, err := at.addrTransactionsBuf.Write(newRecord); err != nil {
		return err
	}
	return nil
}

func (at *addressTransactions) newTransactionsByAddrIterator(addr proto.Address) (*txIter, error) {
	if !at.params.providesData {
		return nil, errors.New("state does not provide transactions by addresses now")
	}
	key := addr.ID().Bytes()
	iter, err := at.stor.newBackwardRecordIterator(key)
	if err != nil {
		return nil, err
	}
	return newTxIter(at.rw, iter), nil
}

func (at *addressTransactions) startProvidingData() error {
	if at.params.providesData {
		// Already provides.
		return nil
	}
	if err := at.persist(); err != nil {
		return err
	}
	at.params.providesData = true
	return nil
}

func (at *addressTransactions) offsetFromBytes(offsetBytes []byte) uint64 {
	return binary.BigEndian.Uint64(offsetBytes)
}

func (at *addressTransactions) handleRecord(record []byte) error {
	key := record[:proto.AddressIDSize]
	newRecordBytes := record[proto.AddressIDSize:]
	lastOffsetBytes, err := at.stor.newestLastRecordByKey(key)
	if err == errNotFound {
		// The first record for this key.
		if err := at.stor.addRecordBytes(key, newRecordBytes); err != nil {
			return errors.Wrap(err, "batchedStorage: failed to add record")
		}
		return nil
	} else if err != nil {
		return errors.Wrap(err, "newestLastRecordByKey() failed")
	}
	// Make sure the offset we add is greater than any other offsets
	// by comparing it to the last (= maximum) offset.
	// This makes adding from file to batchedStorage idempotent.
	newOffsetBytes := newRecordBytes[blockNumLen:]
	offset := at.offsetFromBytes(newOffsetBytes)
	lastOffset := at.offsetFromBytes(lastOffsetBytes)
	if lastOffset > at.rw.blockchainLen {
		return errors.Errorf("invalid offset in storage: %d, max is: %d", lastOffset, at.rw.blockchainLen)
	}
	if offset <= lastOffset {
		return nil
	}
	if err := at.stor.addRecordBytes(key, newRecordBytes); err != nil {
		return errors.Wrap(err, "batchedStorage: failed to add record")
	}
	return nil
}

func (at *addressTransactions) shouldPersist() (bool, error) {
	fileStats, err := os.Stat(at.filePath)
	if err != nil {
		return false, err
	}
	size := fileStats.Size()
	zap.S().Debugf("TransactionsByAddresses file size: %d; max is %d", size, at.params.maxFileSize)
	return size >= at.params.maxFileSize, nil
}

func (at *addressTransactions) persist() error {
	fileStats, err := os.Stat(at.filePath)
	if err != nil {
		return err
	}
	size := fileStats.Size()
	zap.S().Info("Starting to sort TransactionsByAddresses file, will take awhile...")
	debug.FreeOSMemory()
	// Create file for emsort and set emsort over it.
	tempFile, err := os.CreateTemp(os.TempDir(), "emsort")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file for emsort")
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			zap.S().Warnf("Failed to remove temporary file: %v", err)
		}
	}(tempFile.Name())
	sort, err := emsort.NewFixedSize(addrTxRecordSize, maxEmsortMem, tempFile)
	if err != nil {
		return errors.Wrap(err, "emsort.NewFixedSize() failed")
	}

	// Read records from file and append to emsort.
	for readPos := int64(0); readPos < size; readPos += addrTxRecordSize {
		record := make([]byte, addrTxRecordSize)
		if n, err := at.addrTransactions.ReadAt(record, readPos); err != nil {
			return err
		} else if n != addrTxRecordSize {
			return errors.New("failed to read full record")
		}
		// Filtering optimization: if all blocks are valid,
		// we shouldn't check isValid() on records.
		isValid := true
		if at.amend {
			blockNum := binary.BigEndian.Uint32(record[proto.AddressIDSize : proto.AddressIDSize+4])
			isValid, err = at.stateDB.isValidBlock(blockNum)
			if err != nil {
				return errors.Wrap(err, "isValidBlock() failed")
			}
		}
		if !isValid {
			// Invalid record, we should skip it.
			continue
		}
		if err := sort.Push(record); err != nil {
			return errors.Wrap(err, "emsort.Push() failed")
		}
	}
	// Tell emsort that we have finished appending records.
	if err := sort.StopWriting(); err != nil {
		return errors.Wrap(err, "emsort.StopWriting() failed")
	}
	zap.S().Info("Finished to sort TransactionsByAddresses file")
	debug.FreeOSMemory()
	zap.S().Info("Writing sorted records to database, will take awhile...")
	// Read records from emsort in sorted order and save to batchedStorage.
	for {
		record, err := sort.Pop()
		if err == io.EOF {
			// All records were read.
			break
		} else if err != nil {
			return errors.Wrap(err, "emsort.Pop() failed")
		}
		if err := at.handleRecord(record); err != nil {
			return errors.Wrap(err, "failed to add record")
		}
	}
	// Write 0 size to database batch.
	// This way 0 size will be written to database together with new records.
	// If program crashes after batch is flushed but before we truncate the file,
	// next time 0 size will be read and file will be truncated upon next start.
	if err := at.saveFileSizeToBatch(at.stor.dbBatch, 0); err != nil {
		return errors.Wrap(err, "failed to write file size to db batch")
	}
	// Flush batchedStorage.
	if err := at.stor.flush(); err != nil {
		return errors.Wrap(err, "batchedStorage(): failed to flush")
	}
	// Clear batchedStorage.
	at.stor.reset()
	// Clear address transactions file.
	if err := at.addrTransactions.Truncate(0); err != nil {
		return err
	}
	if _, err := at.addrTransactions.Seek(0, 0); err != nil {
		return err
	}
	at.addrTransactionsBuf.Reset(at.addrTransactions)
	zap.S().Info("Successfully finished moving records from file to database")
	debug.FreeOSMemory()
	return nil
}

func (at *addressTransactions) saveFileSizeToBatch(batch keyvalue.Batch, size uint64) error {
	fileSizeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(fileSizeBytes, size)
	batch.Put(fileSizeKeyBytes, fileSizeBytes)
	return nil
}

func (at *addressTransactions) reset() {
	if at.params.providesData {
		at.stor.reset()
	} else {
		at.addrTransactionsBuf.Reset(at.addrTransactions)
	}
}

func (at *addressTransactions) flush() error {
	if at.params.providesData {
		return at.stor.flush()
	}
	if err := at.addrTransactionsBuf.Flush(); err != nil {
		return err
	}
	if err := at.addrTransactions.Sync(); err != nil {
		return err
	}
	fileStats, err := os.Stat(at.filePath)
	if err != nil {
		return err
	}
	size := uint64(fileStats.Size())
	if err := at.saveFileSizeToBatch(at.stateDB.dbBatch, size); err != nil {
		return err
	}
	return nil
}

func (at *addressTransactions) providesData() bool {
	return at.params.providesData
}

func (at *addressTransactions) close() error {
	return at.addrTransactions.Close()
}
