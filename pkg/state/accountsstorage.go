package state

import (
	"encoding/binary"
	"log"
	"math"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

const (
	recordSize = crypto.SignatureSize + 8
)

func idToKey(id []byte) ([]byte, error) {
	sig, err := crypto.NewSignatureFromBytes(id)
	if err != nil {
		return nil, err
	}
	key := blockIdKey{blockID: sig}
	return key.bytes(), nil
}

func filterHistory(db keyvalue.KeyValue, historyKey []byte) ([]byte, error) {
	fmt, err := newHistoryFormatter(recordSize, crypto.SignatureSize)
	if err != nil {
		return nil, err
	}
	history, err := db.Get(historyKey)
	if err != nil {
		return nil, err
	}
	prevSize := len(history)
	history, err = fmt.filter(history, db, idToKey)
	if err != nil {
		return nil, err
	}
	if len(history) != prevSize {
		// Some records were removed, so we need to update the DB.
		if err := db.Put(historyKey, history); err != nil {
			return nil, err
		}
	}
	return history, nil
}

type accountsStorage struct {
	genesis crypto.Signature

	db        keyvalue.IterableKeyVal
	dbBatch   keyvalue.Batch
	localStor *localStorage

	// rw is used to get height by ID.
	rw          *blockReadWriter
	rollbackMax int

	// fmt is used for operations on balances history.
	fmt *historyFormatter
}

var Empty = []byte{}

func newAccountsStorage(
	genesis crypto.Signature,
	db keyvalue.IterableKeyVal,
	dbBatch keyvalue.Batch,
	rw *blockReadWriter,
) (*accountsStorage, error) {
	has, err := db.Has([]byte{dbHeightKeyPrefix})
	if err != nil {
		return nil, err
	}
	if !has {
		heightBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(heightBuf, 0)
		if err := db.Put([]byte{dbHeightKeyPrefix}, heightBuf); err != nil {
			return nil, err
		}
	}
	localStor, err := newLocalStorage(db, filterHistory)
	if err != nil {
		return nil, err
	}
	fmt, err := newHistoryFormatter(recordSize, crypto.SignatureSize)
	if err != nil {
		return nil, err
	}
	return &accountsStorage{
		genesis:   genesis,
		db:        db,
		dbBatch:   dbBatch,
		rw:        rw,
		localStor: localStor,
		fmt:       fmt,
	}, nil
}

func (s *accountsStorage) setRollbackMax(rollbackMax int) {
	s.rollbackMax = rollbackMax
}

func (s *accountsStorage) idToHeight(id []byte) (uint64, error) {
	sig, err := crypto.NewSignatureFromBytes(id)
	if err != nil {
		return 0, err
	}
	return s.rw.heightByBlockID(sig)
}

func (s *accountsStorage) setHeight(height uint64, directly bool) error {
	dbHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(dbHeightBytes, height)
	if directly {
		if err := s.db.Put([]byte{dbHeightKeyPrefix}, dbHeightBytes); err != nil {
			return err
		}
	} else {
		s.dbBatch.Put([]byte{dbHeightKeyPrefix}, dbHeightBytes)
	}
	return nil
}

func (s *accountsStorage) getHeight() (uint64, error) {
	dbHeightBytes, err := s.db.Get([]byte{dbHeightKeyPrefix})
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(dbHeightBytes), nil
}

func (s *accountsStorage) cutHistory(historyKey []byte, history []byte) ([]byte, error) {
	prevSize := len(history)
	currentHeight, err := s.getHeight()
	if err != nil {
		return nil, err
	}

	history, err = s.fmt.cut(history, s.idToHeight, currentHeight, s.genesis[:])
	if err != nil {
		return nil, err
	}
	if len(history) != prevSize {
		// Some records were removed, so we need to update the DB.
		if err := s.db.Put(historyKey, history); err != nil {
			return nil, err
		}
	}
	return history, nil
}

func (s *accountsStorage) addressesNumber() (uint64, error) {
	iter, err := s.db.NewKeyIterator([]byte{balanceKeyPrefix})
	if err != nil {
		return 0, err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			log.Fatalf("Iterator error: %v", err)
		}
	}()

	addressesNumber := uint64(0)
	for iter.Next() {
		balance, err := s.accountBalance(iter.Key())
		if err != nil {
			return 0, err
		}
		if balance > 0 {
			addressesNumber++
		}
	}
	return addressesNumber, nil
}

// minBalanceInRange() is used to get min miner's effective balance, so it includes blocks which
// have not been flushed to DB yet (and are currently stored in memory).
func (s *accountsStorage) minBalanceInRange(balanceKey []byte, startHeight, endHeight uint64) (uint64, error) {
	history, err := s.localStor.record(balanceKey)
	if err != nil {
		return 0, err
	}
	minBalance := uint64(math.MaxUint64)
	for i := len(history); i >= recordSize; i -= recordSize {
		record := history[i-recordSize : i]
		balanceEnd := len(record) - crypto.SignatureSize
		idBytes := record[balanceEnd:]
		blockID, err := crypto.NewSignatureFromBytes(idBytes)
		if err != nil {
			return 0, err
		}
		// Set height to genesis by default.
		height := uint64(1)
		if blockID != s.genesis {
			// Change height if needed.
			height, err = s.rw.heightByNewBlockID(blockID)
			if err != nil {
				return 0, err
			}
		}
		if height > endHeight {
			continue
		}
		if height < startHeight && minBalance != math.MaxUint64 {
			break
		}
		balance := binary.LittleEndian.Uint64(record[balanceEnd-8 : balanceEnd])
		if balance < minBalance {
			minBalance = balance
		}
	}
	if minBalance == math.MaxUint64 {
		return 0, errors.New("invalid height range or unknown address")
	}
	return minBalance, nil
}

func (s *accountsStorage) accountBalance(balanceKey []byte) (uint64, error) {
	has, err := s.db.Has(balanceKey)
	if err != nil {
		return 0, errors.Errorf("failed to check if balance key exists: %v\n", err)
	}
	if !has {
		// TODO: think about this scenario.
		return 0, nil
	}
	// Delete invalid records.
	history, err := filterHistory(s.db, balanceKey)
	if err != nil {
		return 0, errors.Errorf("failed to filter history: %v\n", err)
	}
	if s.rollbackMax != 0 {
		// Remove records which are too far in the past.
		history, err = s.cutHistory(balanceKey, history)
		if err != nil {
			return 0, errors.Errorf("failed to cut history: %v\n", err)
		}
	}
	if len(history) == 0 {
		// There were no valid records, so the history is empty after filtering.
		return 0, nil
	}
	record, err := s.fmt.getLatest(history)
	if err != nil {
		return 0, err
	}
	balance := binary.LittleEndian.Uint64(record[:len(record)-crypto.SignatureSize])
	return balance, nil
}

func (s *accountsStorage) newHistory(newRecord []byte, key []byte) ([]byte, error) {
	// Get current history.
	history, err := s.localStor.record(key)
	if err != nil {
		return nil, err
	}
	return s.fmt.addRecord(history, newRecord)
}

func (s *accountsStorage) setAccountBalance(balanceKey []byte, balance uint64, blockID crypto.Signature) error {
	// Add block to valid blocks.
	key := blockIdKey{blockID: blockID}
	s.dbBatch.Put(key.bytes(), Empty)
	// Prepare new record.
	balanceBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(balanceBuf, balance)
	newRecord := append(balanceBuf, blockID[:]...)
	// Add it to history.
	history, err := s.newHistory(newRecord, balanceKey)
	if err != nil {
		return err
	}
	if err := s.localStor.setRecord(balanceKey, history); err != nil {
		return err
	}
	return nil
}

func (s *accountsStorage) rollbackBlock(blockID crypto.Signature) error {
	// Decrease DB's height (for sync/recovery).
	height, err := s.getHeight()
	if err != nil {
		return err
	}
	if err := s.setHeight(height-1, true); err != nil {
		return err
	}
	key := blockIdKey{blockID: blockID}
	if err := s.db.Delete(key.bytes()); err != nil {
		return err
	}
	return nil
}

func (s *accountsStorage) updateHeight(heightChange int) error {
	// Increase DB's height (for sync/recovery).
	height, err := s.getHeight()
	if err != nil {
		return err
	}
	if err := s.setHeight(height+uint64(heightChange), false); err != nil {
		return err
	}
	return nil
}

func (s *accountsStorage) reset() {
	s.dbBatch.Reset()
	s.localStor.reset()
}

func (s *accountsStorage) flush() error {
	if err := s.localStor.addToBatch(s.dbBatch); err != nil {
		return err
	}
	if err := s.db.Flush(s.dbBatch); err != nil {
		return err
	}
	s.reset()
	return nil
}
