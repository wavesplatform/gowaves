package state

import (
	"encoding/binary"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type txIter struct {
	rw   *blockReadWriter
	iter *recordIterator
	err  error
}

func newTxIter(rw *blockReadWriter, iter *recordIterator) *txIter {
	return &txIter{rw: rw, iter: iter}
}

func (i *txIter) Transaction() (proto.Transaction, error) {
	offsetBytes, err := i.iter.currentRecord()
	if err != nil {
		return nil, err
	}
	offset := binary.BigEndian.Uint64(offsetBytes)
	txBytes, err := i.rw.readTransactionByOffset(offset)
	if err != nil {
		return nil, err
	}
	tx, err := proto.BytesToTransaction(txBytes)
	if err != nil {
		return nil, err
	}
	return tx, nil
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

type addressTransactions struct {
	stateDB *stateDB
	rw      *blockReadWriter
	stor    *batchedStorage
}

func newAddressTransactions(
	db keyvalue.IterableKeyVal,
	dbBatch keyvalue.Batch,
	writeLock *sync.Mutex,
	stateDB *stateDB,
	rw *blockReadWriter,
) *addressTransactions {
	params := &batchedStorParams{
		maxBatchSize: maxTransactionIdsBatchSize,
		recordSize:   rw.offsetLen,
		prefix:       transactionIdsPrefix,
	}
	stor := newBatchedStorage(db, dbBatch, writeLock, stateDB, params)
	return &addressTransactions{stateDB, rw, stor}
}

func (at *addressTransactions) saveTxIdByAddress(addr proto.Address, txID []byte, blockID crypto.Signature) error {
	blockNum, err := at.stateDB.blockIdToNum(blockID)
	if err != nil {
		return err
	}
	key := addr.Bytes()
	offset, err := at.rw.newestTransactionOffsetByID(txID)
	if err != nil {
		return err
	}
	offsetBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(offsetBytes, offset)
	return at.stor.addRecord(key, offsetBytes, blockNum)
}

func (at *addressTransactions) newTransactionsByAddrIterator(addr proto.Address) (*txIter, error) {
	key := addr.Bytes()
	iter, err := at.stor.newBackwardRecordIterator(key)
	if err != nil {
		return nil, err
	}
	return newTxIter(at.rw, iter), nil
}

func (at *addressTransactions) reset() {
	at.stor.reset()
}

func (at *addressTransactions) flush() error {
	if err := at.stor.flush(); err != nil {
		return err
	}
	return nil
}
