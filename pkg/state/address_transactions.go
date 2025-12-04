package state

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/lanrat/extsort"
	"golang.org/x/sync/errgroup"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	txMetaSize = 8 + 1
	// AddressID size + length of block num + transaction offset length.
	addrTxRecordSize = proto.AddressIDSize + blockNumLen + txMetaSize
	maxEmsortMem     = 200 * 1024 * 1024 // 200 MiB.
)

var (
	fileSizeKeyBytes = []byte{txsByAddressesFileSizeKeyPrefix}
)

type txMeta struct {
	offset uint64
	status proto.TransactionStatus
}

func (m *txMeta) bytes() []byte {
	buf := make([]byte, txMetaSize)
	binary.BigEndian.PutUint64(buf, m.offset)
	buf[8] = byte(m.status)
	return buf
}

func (m *txMeta) unmarshal(data []byte) error {
	if len(data) < txMetaSize {
		return errInvalidDataSize
	}
	m.offset = binary.BigEndian.Uint64(data)
	switch s := proto.TransactionStatus(data[8]); s {
	case proto.TransactionSucceeded, proto.TransactionFailed, proto.TransactionElided:
		m.status = s
	default:
		return fmt.Errorf("invalid tx status (%d)", s)
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

func (i *txIter) Transaction() (proto.Transaction, proto.TransactionStatus, error) {
	value, err := i.iter.currentRecord()
	if err != nil {
		return nil, 0, err
	}
	var meta txMeta
	err = meta.unmarshal(value)
	if err != nil {
		return nil, 0, err
	}
	tx, err := i.rw.readTransactionByOffset(meta.offset)
	if err != nil {
		return nil, 0, err
	}
	return tx, meta.status, nil
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
	switch {
	case errors.Is(err, keyvalue.ErrNotFound):
		properFileSize = 0
	case err == nil:
		properFileSize = binary.BigEndian.Uint64(fileSizeBytes)
	default:
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
	ctx                 context.Context
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
	ctx context.Context,
	db keyvalue.IterableKeyVal,
	stateDB *stateDB,
	rw *blockReadWriter,
	params *addressTransactionsParams,
	amend bool,
) (_ *addressTransactions, retErr error) {
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
	defer func() {
		if retErr != nil {
			if fErr := addrTransactionsFile.Close(); fErr != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to close address_transactions file: %w", fErr))
			}
		}
	}()
	if err := manageFile(addrTransactionsFile, db); err != nil {
		return nil, err
	}
	stor, err := newBatchedStorage(db, stateDB, bsParams, params.batchedStorMemLimit, params.batchedStorMaxKeys, amend)
	if err != nil {
		return nil, err
	}
	atx := &addressTransactions{
		ctx:                 ctx,
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
		if pErr := atx.persist(); pErr != nil { // no need to close atx here because all resources will be closed above
			return nil, fmt.Errorf("failed to persist address_transactions file: %w", pErr)
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
	meta := txMeta{info.offset, info.txStatus}
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
	if errors.Is(err, errNotFound) {
		// The first record for this key.
		if adErr := at.stor.addRecordBytes(key, newRecordBytes); adErr != nil {
			return fmt.Errorf("batchedStorage: failed to add record: %w", adErr)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("newestLastRecordByKey() failed: %w", err)
	}
	// Make sure the offset we add is greater than any other offsets
	// by comparing it to the last (= maximum) offset.
	// This makes adding from file to batchedStorage idempotent.
	newOffsetBytes := newRecordBytes[blockNumLen:]
	offset := at.offsetFromBytes(newOffsetBytes)
	lastOffset := at.offsetFromBytes(lastOffsetBytes)
	if lastOffset > at.rw.blockchainLen {
		return fmt.Errorf("invalid offset in storage: %d, max is: %d", lastOffset, at.rw.blockchainLen)
	}
	if offset <= lastOffset {
		return nil
	}
	if adErr := at.stor.addRecordBytes(key, newRecordBytes); adErr != nil {
		return fmt.Errorf("batchedStorage: failed to add record: %w", adErr)
	}
	return nil
}

func (at *addressTransactions) shouldPersist() (bool, error) {
	fileStats, err := os.Stat(at.filePath)
	if err != nil {
		return false, err
	}
	size := fileStats.Size()
	slog.Debug("TransactionsByAddresses file", "size", size, "max", at.params.maxFileSize)
	return size >= at.params.maxFileSize, nil
}

func nopPassBytes(data []byte) ([]byte, error) {
	return data, nil
}

func (at *addressTransactions) persist() error {
	start := time.Now()
	fst, err := os.Stat(at.filePath)
	if err != nil {
		return err
	}
	size := fst.Size()
	slog.Info("Starting to sort TransactionsByAddresses file, will take awhile...", "size", size)
	debug.FreeOSMemory()
	eg, egCtx := errgroup.WithContext(at.ctx)

	conf := extsort.DefaultConfig()
	conf.NumWorkers = runtime.NumCPU() / 2
	conf.TempFilesDir = os.TempDir() // Set dir explicitly to turn off automatic selection.

	inCh := make(chan []byte, conf.SortedChanBuffSize) // Set size of input channel the same as of output.
	eg.Go(func() error {
		defer close(inCh)
		for readPos := int64(0); readPos < size; readPos += addrTxRecordSize {
			rec := make([]byte, addrTxRecordSize)
			if n, rErr := at.addrTransactions.ReadAt(rec, readPos); rErr != nil {
				return rErr
			} else if n != addrTxRecordSize {
				return errors.New("failed to read full record")
			}
			// Filtering optimization: if all blocks are valid,
			// we shouldn't check isValid() on records.
			var (
				isValid = true
				vErr    error
			)
			if at.amend {
				blockNum := binary.BigEndian.Uint32(rec[proto.AddressIDSize : proto.AddressIDSize+4])
				isValid, vErr = at.stateDB.isValidBlock(blockNum)
				if vErr != nil {
					return fmt.Errorf("block validation failed: %w", vErr)
				}
			}
			if !isValid {
				// Invalid record, we should skip it.
				continue
			}
			inCh <- rec
		}
		slog.Info("Finished to send transactions to sorter")
		return nil
	})
	sorter, outCh, errCh := extsort.Generic(inCh, nopPassBytes, nopPassBytes, bytes.Compare, conf)
	eg.Go(func() error {
		sorter.Sort(egCtx)
		slog.Info("Finished to sort transactions")
		return nil
	})
	eg.Go(func() error {
		return <-errCh
	})
	eg.Go(func() error {
		slog.Info("Writing sorted records to database, will take awhile...")
		// Read records from sorter in sorted order and save to batchedStorage.
		for rec := range outCh {
			if hErr := at.handleRecord(rec); err != nil {
				return fmt.Errorf("failed to add record: %w", hErr)
			}
		}
		// Write 0 size to database batch.
		// This way 0 size will be written to database together with new records.
		// If program crashes after batch is flushed but before we truncate the file,
		// next time 0 size will be read and file will be truncated upon next start.
		if sErr := at.saveFileSizeToBatch(at.stor.dbBatch, 0); sErr != nil {
			return fmt.Errorf("failed to write file size to db batch: %w", sErr)
		}
		// Flush batchedStorage.
		if fErr := at.stor.flush(); fErr != nil {
			return fmt.Errorf("batchedStorage(): failed to flush: %w", fErr)
		}
		return nil
	})
	// Wait for goroutines to finish.
	if gErr := eg.Wait(); gErr != nil {
		return gErr
	}
	// Clear batchedStorage.
	at.stor.reset()
	// Clear address transactions file.
	if tErr := at.addrTransactions.Truncate(0); tErr != nil {
		return tErr
	}
	if _, sErr := at.addrTransactions.Seek(0, 0); sErr != nil {
		return sErr
	}
	at.addrTransactionsBuf.Reset(at.addrTransactions)
	took := time.Since(start)
	slog.Info("Successfully finished moving records from file to database", "took", took)
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
