package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

const (
	recordSize = crypto.SignatureSize + 8
)

func filterHistory(db keyvalue.KeyValue, historyKey []byte, history []byte) ([]byte, error) {
	historySize := len(history)
	for i := historySize; i >= recordSize; i -= recordSize {
		record := history[i-recordSize : i]
		idBytes := record[len(record)-crypto.SignatureSize:]
		blockID, err := toBlockID(idBytes)
		if err != nil {
			return nil, err
		}
		key := blockIdKey{blockID: blockID}
		has, err := db.Has(key.bytes())
		if err != nil {
			return nil, err
		}
		if has {
			// Is valid block.
			break
		}
		// Erase invalid (outdated due to rollbacks) record.
		history = history[:i-recordSize]
	}
	if len(history) != historySize {
		// Some records were removed, so we need to update the DB.
		if err := db.PutDirectly(historyKey, history); err != nil {
			return nil, err
		}
	}
	return history, nil
}

type localStor struct {
	db     keyvalue.KeyValue
	waves  map[wavesBalanceKey][]byte
	assets map[assetBalanceKey][]byte
}

func newLocalStor(db keyvalue.KeyValue) (*localStor, error) {
	return &localStor{
		db:     db,
		waves:  make(map[wavesBalanceKey][]byte),
		assets: make(map[assetBalanceKey][]byte),
	}, nil
}

func (s *localStor) retrieveHistoryFromDb(key []byte) ([]byte, error) {
	has, err := s.db.Has(key)
	if err != nil {
		return nil, err
	}
	if !has {
		// New history.
		return nil, nil
	}
	// Get current history.
	history, err := s.db.Get(key)
	if err != nil {
		return nil, err
	}
	// Delete invalid (because of rollback) records.
	history, err = filterHistory(s.db, key, history)
	if err != nil {
		return nil, err
	}
	return history, nil
}

func (s *localStor) getHistory(key []byte) ([]byte, error) {
	size := len(key)
	if size == wavesBalanceKeySize {
		var wavesKey wavesBalanceKey
		copy(wavesKey[:], key)
		if _, ok := s.waves[wavesKey]; !ok {
			history, err := s.retrieveHistoryFromDb(key)
			if err != nil {
				return nil, err
			}
			s.waves[wavesKey] = history
		}
		return s.waves[wavesKey], nil
	} else if size == assetBalanceKeySize {
		var assetKey assetBalanceKey
		copy(assetKey[:], key)
		if _, ok := s.assets[assetKey]; !ok {
			history, err := s.retrieveHistoryFromDb(key)
			if err != nil {
				return nil, err
			}
			s.assets[assetKey] = history
		}
		return s.assets[assetKey], nil
	} else {
		return nil, errors.New("invalid key size")
	}
}

func (s *localStor) setHistory(key []byte, history []byte) error {
	size := len(key)
	if size == wavesBalanceKeySize {
		var wavesKey wavesBalanceKey
		copy(wavesKey[:], key)
		s.waves[wavesKey] = history
	} else if size == assetBalanceKeySize {
		var assetKey assetBalanceKey
		copy(assetKey[:], key)
		s.assets[assetKey] = history
	} else {
		return errors.New("invalid key size")
	}
	return nil
}

func (s *localStor) reset() {
	s.waves = make(map[wavesBalanceKey][]byte)
	s.assets = make(map[assetBalanceKey][]byte)
}

type idToHeight interface {
	heightToBlockID(blockID crypto.Signature) (uint64, error)
}

type accountsStorage struct {
	genesis     crypto.Signature
	db          keyvalue.IterableKeyVal
	idToHeight  idToHeight
	rollbackMax int
	localStor   *localStor
}

var Empty = []byte{}

func toBlockID(bytes []byte) (crypto.Signature, error) {
	var res crypto.Signature
	if len(bytes) != crypto.SignatureSize {
		return res, errors.New("failed to convert bytes to block ID: invalid length of bytes")
	}
	copy(res[:], bytes)
	return res, nil
}

func newAccountsStorage(genesis crypto.Signature, db keyvalue.IterableKeyVal) (*accountsStorage, error) {
	has, err := db.Has([]byte{dbHeightKeyPrefix})
	if err != nil {
		return nil, err
	}
	if !has {
		heightBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(heightBuf, 0)
		if err := db.PutDirectly([]byte{dbHeightKeyPrefix}, heightBuf); err != nil {
			return nil, err
		}
	}
	localStor, err := newLocalStor(db)
	if err != nil {
		return nil, err
	}
	return &accountsStorage{
		genesis:   genesis,
		db:        db,
		localStor: localStor,
	}, nil
}

func (s *accountsStorage) setRollbackMax(rollbackMax int, idToHeight idToHeight) {
	s.rollbackMax = rollbackMax
	s.idToHeight = idToHeight
}

func (s *accountsStorage) setHeight(height uint64, directly bool) error {
	dbHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(dbHeightBytes, height)
	if directly {
		if err := s.db.PutDirectly([]byte{dbHeightKeyPrefix}, dbHeightBytes); err != nil {
			return err
		}
	} else {
		if err := s.db.Put([]byte{dbHeightKeyPrefix}, dbHeightBytes); err != nil {
			return err
		}
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
	historySize := len(history)
	currentHeight, err := s.getHeight()
	if err != nil {
		return nil, err
	}
	firstNeeded := 0
	for i := recordSize; i <= historySize; i += recordSize {
		record := history[i-recordSize : i]
		idBytes := record[len(record)-crypto.SignatureSize:]
		blockID, err := toBlockID(idBytes)
		if err != nil {
			return nil, err
		}
		if blockID != s.genesis {
			blockHeight, err := s.idToHeight.heightToBlockID(blockID)
			if err != nil {
				return nil, err
			}
			if currentHeight-blockHeight > uint64(s.rollbackMax) {
				// 1 record BEFORE rollbackMax blocks is needed.
				firstNeeded = i - recordSize
				continue
			}
			break
		}
	}
	if firstNeeded != 0 {
		history = history[firstNeeded:]
		// Some records were removed, so we need to update the DB.
		if err := s.db.PutDirectly(historyKey, history); err != nil {
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
	iter.Release()
	if err := iter.Error(); err != nil {
		return 0, err
	}
	return addressesNumber, nil
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
	history, err := s.db.Get(balanceKey)
	if err != nil {
		return 0, errors.Errorf("failed to get history for given key: %v\n", err)
	}
	// Delete invalid records.
	history, err = filterHistory(s.db, balanceKey, history)
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
	balanceEnd := len(history) - crypto.SignatureSize
	balance := binary.LittleEndian.Uint64(history[balanceEnd-8 : balanceEnd])
	return balance, nil
}

func (s *accountsStorage) newHistory(newRecord []byte, key []byte, blockID crypto.Signature) ([]byte, error) {
	// Get current history.
	history, err := s.localStor.getHistory(key)
	if err != nil {
		return nil, err
	}
	if len(history) < recordSize {
		// History is empty, new record is the first one.
		return newRecord, nil
	}
	lastRecord := history[len(history)-recordSize:]
	idBytes := lastRecord[len(lastRecord)-crypto.SignatureSize:]
	lastBlockID, err := toBlockID(idBytes)
	if err != nil {
		return nil, err
	}
	if lastBlockID == blockID {
		// If the last record is the same block, rewrite it.
		copy(history[len(history)-recordSize:], newRecord)
	} else {
		// Append new record to the end.
		history = append(history, newRecord...)
	}
	return history, nil
}

func (s *accountsStorage) setAccountBalance(balanceKey []byte, balance uint64, blockID crypto.Signature) error {
	// Add block to valid blocks.
	key := blockIdKey{blockID: blockID}
	if err := s.db.Put(key.bytes(), Empty); err != nil {
		return err
	}
	// Prepare new record.
	balanceBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(balanceBuf, balance)
	newRecord := append(balanceBuf, blockID[:]...)
	// Add it to history.
	history, err := s.newHistory(newRecord, balanceKey, blockID)
	if err != nil {
		return err
	}
	if err := s.localStor.setHistory(balanceKey, history); err != nil {
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

func (s *accountsStorage) addChangesToBatch() error {
	for key, history := range s.localStor.waves {
		if err := s.db.Put(key[:], history); err != nil {
			return err
		}
	}
	for key, history := range s.localStor.assets {
		if err := s.db.Put(key[:], history); err != nil {
			return err
		}
	}
	s.localStor.reset()
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

func (s *accountsStorage) flush() error {
	if err := s.addChangesToBatch(); err != nil {
		return err
	}
	if err := s.db.Flush(); err != nil {
		return err
	}
	return nil
}
