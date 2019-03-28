package state

import (
	"encoding/binary"
	"log"
	"math"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/state/history"
)

const (
	balancesRecordSize = crypto.SignatureSize + 8
)

type blockInfo interface {
	IsValidBlock(blockID crypto.Signature) (bool, error)
}

type heightInfo interface {
	Height() (uint64, error)
	BlockIDToHeight(blockID crypto.Signature) (uint64, error)
	RollbackMax() uint64
}

type heightInfoExt interface {
	heightInfo
	NewBlockIDToHeight(blockID crypto.Signature) (uint64, error)
}

type balances struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	// Local storage for history, is moved to batch after all the changes are made.
	// The motivation for this is inability to read from DB batch.
	localStor map[string][]byte

	hInfo heightInfoExt
	// fmt is used for operations on balances history.
	fmt *history.HistoryFormatter
}

func newBalances(
	db keyvalue.IterableKeyVal,
	dbBatch keyvalue.Batch,
	hInfo heightInfoExt,
	bInfo blockInfo,
) (*balances, error) {
	fmt, err := history.NewHistoryFormatter(balancesRecordSize, crypto.SignatureSize, hInfo, bInfo)
	if err != nil {
		return nil, err
	}
	return &balances{
		db:        db,
		dbBatch:   dbBatch,
		localStor: make(map[string][]byte),
		hInfo:     hInfo,
		fmt:       fmt,
	}, nil
}

func (s *balances) addressesNumber() (uint64, error) {
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
func (s *balances) minBalanceInRange(balanceKey []byte, startHeight, endHeight uint64) (uint64, error) {
	history, err := fullHistory(balanceKey, s.db, s.localStor, s.fmt)
	if err != nil {
		return 0, err
	}
	minBalance := uint64(math.MaxUint64)
	for i := len(history); i >= balancesRecordSize; i -= balancesRecordSize {
		record := history[i-balancesRecordSize : i]
		idBytes, err := s.fmt.GetID(record)
		if err != nil {
			return 0, err
		}
		blockID, err := crypto.NewSignatureFromBytes(idBytes)
		if err != nil {
			return 0, err
		}
		height, err := s.hInfo.NewBlockIDToHeight(blockID)
		if err != nil {
			return 0, err
		}
		if height > endHeight {
			continue
		}
		if height < startHeight && minBalance != math.MaxUint64 {
			break
		}
		balanceEnd := len(record) - crypto.SignatureSize
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

func (s *balances) accountBalance(balanceKey []byte) (uint64, error) {
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
		return 0, err
	}
	history, err = s.fmt.Normalize(history)
	if err != nil {
		return 0, err
	}
	if len(history) == 0 {
		// There were no valid records, so the history is empty after filtering.
		return 0, nil
	}
	record, err := s.fmt.GetLatest(history)
	if err != nil {
		return 0, err
	}
	balance := binary.LittleEndian.Uint64(record[:len(record)-crypto.SignatureSize])
	return balance, nil
}

func (s *balances) setAccountBalance(balanceKey []byte, balance uint64, blockID crypto.Signature) error {
	// Prepare new record.
	balanceBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(balanceBuf, balance)
	newRecord := append(balanceBuf, blockID[:]...)
	// Add it to history.
	history, _ := s.localStor[string(balanceKey)]
	history, err := s.fmt.AddRecord(history, newRecord)
	if err != nil {
		return err
	}
	s.localStor[string(balanceKey)] = history
	return nil
}

func (s *balances) reset() {
	s.localStor = make(map[string][]byte)
}

func (s *balances) flush() error {
	if err := addHistoryToBatch(s.db, s.dbBatch, s.localStor, s.fmt); err != nil {
		return err
	}
	return nil
}
