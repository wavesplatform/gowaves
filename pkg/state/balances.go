package state

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state/history"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	wavesBalanceRecordSize = 8 + 8 + 8 + crypto.SignatureSize
	assetBalanceRecordSize = 8 + crypto.SignatureSize
)

var empty struct{}

type balanceProfile struct {
	balance  uint64
	leaseIn  int64
	leaseOut int64
}

func (bp *balanceProfile) effectiveBalance() (uint64, error) {
	val, err := util.AddInt64(int64(bp.balance), bp.leaseIn)
	if err != nil {
		return 0, err
	}
	return uint64(val - bp.leaseOut), nil
}

func (bp *balanceProfile) spendableBalance() uint64 {
	return uint64(int64(bp.balance) - bp.leaseOut)
}

type wavesBalanceRecord struct {
	balanceProfile
	blockID crypto.Signature
}

func (r *wavesBalanceRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, 8+8+8+crypto.SignatureSize)
	binary.BigEndian.PutUint64(res[:8], r.balance)
	binary.PutVarint(res[8:16], r.leaseIn)
	binary.PutVarint(res[16:24], r.leaseOut)
	copy(res[24:], r.blockID[:])
	return res, nil
}

func (r *wavesBalanceRecord) unmarshalBinary(data []byte) error {
	r.balance = binary.BigEndian.Uint64(data[:8])
	var err error
	r.leaseIn, err = binary.ReadVarint(bytes.NewReader(data[8:16]))
	if err != nil {
		return err
	}
	r.leaseOut, err = binary.ReadVarint(bytes.NewReader(data[16:24]))
	if err != nil {
		return err
	}
	copy(r.blockID[:], data[24:])
	return nil
}

type assetBalanceRecord struct {
	balance uint64
	blockID crypto.Signature
}

func (r *assetBalanceRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, 8+crypto.SignatureSize)
	binary.BigEndian.PutUint64(res[:8], r.balance)
	copy(res[8:], r.blockID[:])
	return res, nil
}

func (r *assetBalanceRecord) unmarshalBinary(data []byte) error {
	r.balance = binary.BigEndian.Uint64(data[:8])
	copy(r.blockID[:], data[8:])
	return nil
}

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
	// Local storages for history, are moved to batch after all the changes are made.
	// The motivation for this is inability to read from DB batch.
	wavesStor map[string][]byte
	assetStor map[string][]byte

	hInfo heightInfoExt
	// assetFmt is used for operations on assets' balances history.
	assetFmt *history.HistoryFormatter
	// wavesFmt is used for operations on waves' balances history.
	wavesFmt *history.HistoryFormatter
}

func newBalances(
	db keyvalue.IterableKeyVal,
	dbBatch keyvalue.Batch,
	hInfo heightInfoExt,
	bInfo blockInfo,
) (*balances, error) {
	assetFmt, err := history.NewHistoryFormatter(assetBalanceRecordSize, crypto.SignatureSize, hInfo, bInfo)
	if err != nil {
		return nil, err
	}
	wavesFmt, err := history.NewHistoryFormatter(wavesBalanceRecordSize, crypto.SignatureSize, hInfo, bInfo)
	if err != nil {
		return nil, err
	}
	return &balances{
		db:        db,
		dbBatch:   dbBatch,
		wavesStor: make(map[string][]byte),
		assetStor: make(map[string][]byte),
		hInfo:     hInfo,
		assetFmt:  assetFmt,
		wavesFmt:  wavesFmt,
	}, nil
}

func (s *balances) cancelAllLeases() error {
	iter, err := s.db.NewKeyIterator([]byte{wavesBalanceKeyPrefix})
	if err != nil {
		return err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			log.Fatalf("Iterator error: %v", err)
		}
	}()

	for iter.Next() {
		key := iter.Key()
		r, err := s.wavesRecord(key)
		if err != nil {
			return err
		}
		r.leaseOut = 0
		r.leaseIn = 0
		if err := s.setWavesBalanceImpl(key, r); err != nil {
			return err
		}
	}
	return nil
}

func (s *balances) cancelLeaseOverflows() (map[proto.Address]struct{}, error) {
	iter, err := s.db.NewKeyIterator([]byte{wavesBalanceKeyPrefix})
	if err != nil {
		return nil, err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			log.Fatalf("Iterator error: %v", err)
		}
	}()

	overflowedAddresses := make(map[proto.Address]struct{})
	for iter.Next() {
		key := iter.Key()
		r, err := s.wavesRecord(key)
		if err != nil {
			return nil, err
		}
		if int64(r.balance) < r.leaseOut {
			var k wavesBalanceKey
			if err := k.unmarshal(key); err != nil {
				return nil, err
			}
			log.Printf("Resolving lease overflow for address %s: %d ---> %d", k.address.String(), r.leaseOut, 0)
			overflowedAddresses[k.address] = empty
			r.leaseOut = 0
		}
		if err := s.setWavesBalanceImpl(key, r); err != nil {
			return nil, err
		}
	}
	return overflowedAddresses, err
}

func (s *balances) cancelInvalidLeaseIns(correctLeaseIns map[proto.Address]int64) error {
	for addr, leaseIn := range correctLeaseIns {
		k := wavesBalanceKey{addr}
		r, err := s.wavesRecord(k.bytes())
		if err != nil {
			return err
		}
		if r.leaseIn != leaseIn {
			log.Printf("Invalid leaseIn detected; fixing it: %d ---> %d.", r.leaseIn, leaseIn)
			r.leaseIn = leaseIn
			if err := s.setWavesBalanceImpl(k.bytes(), r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *balances) wavesAddressesNumber() (uint64, error) {
	iter, err := s.db.NewKeyIterator([]byte{wavesBalanceKeyPrefix})
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
		key := iter.Key()
		profile, err := s.wavesBalanceImpl(key)
		if err != nil {
			return 0, err
		}
		if profile.balance > 0 {
			addressesNumber++
		}
	}
	return addressesNumber, nil
}

// minBalanceInRange() is used to get min miner's effective balance, so it includes blocks which
// have not been flushed to DB yet (and are currently stored in memory).
func (s *balances) minEffectiveBalanceInRange(addr proto.Address, startHeight, endHeight uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	history, err := fullHistory(key.bytes(), s.db, s.wavesStor, s.wavesFmt)
	if err != nil {
		return 0, err
	}
	minBalance := uint64(math.MaxUint64)
	for i := len(history); i >= wavesBalanceRecordSize; i -= wavesBalanceRecordSize {
		recordBytes := history[i-wavesBalanceRecordSize : i]
		idBytes, err := s.wavesFmt.GetID(recordBytes)
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
		var record wavesBalanceRecord
		if err := record.unmarshalBinary(recordBytes); err != nil {
			return 0, err
		}
		effectiveBal, err := record.effectiveBalance()
		if err != nil {
			return 0, err
		}
		if effectiveBal < minBalance {
			minBalance = effectiveBal
		}
	}
	if minBalance == math.MaxUint64 {
		return 0, errors.New("invalid height range or unknown address")
	}
	return minBalance, nil
}

func (s *balances) lastRecord(key []byte, fmt *history.HistoryFormatter) ([]byte, error) {
	has, err := s.db.Has(key)
	if err != nil {
		return nil, errors.Errorf("failed to check if balance key exists: %v\n", err)
	}
	if !has {
		// TODO: think about this scenario.
		return nil, nil
	}
	history, err := s.db.Get(key)
	if err != nil {
		return nil, err
	}
	history, err = fmt.Normalize(history)
	if err != nil {
		return nil, err
	}
	if len(history) == 0 {
		// There were no valid records, so the history is empty after filtering.
		return nil, nil
	}
	last, err := fmt.GetLatest(history)
	if err != nil {
		return nil, err
	}
	return last, nil
}

func (s *balances) assetBalance(addr proto.Address, asset []byte) (uint64, error) {
	key := assetBalanceKey{address: addr, asset: asset}
	last, err := s.lastRecord(key.bytes(), s.assetFmt)
	if err != nil {
		return 0, err
	}
	if last == nil {
		// No records = unknown address, expected behavior is to return 0 and no errors in this case.
		return 0, nil
	}
	var record assetBalanceRecord
	if err := record.unmarshalBinary(last); err != nil {
		return 0, err
	}
	return record.balance, nil
}

func (s *balances) wavesRecord(key []byte) (*wavesBalanceRecord, error) {
	last, err := s.lastRecord(key, s.wavesFmt)
	if err != nil {
		return nil, err
	}
	if last == nil {
		// No records = unknown address, expected behavior is to return empty profile and no errors in this case.
		return &wavesBalanceRecord{}, nil
	}
	var record wavesBalanceRecord
	if err := record.unmarshalBinary(last); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *balances) wavesBalanceImpl(key []byte) (*balanceProfile, error) {
	r, err := s.wavesRecord(key)
	if err != nil {
		return nil, err
	}
	return &r.balanceProfile, nil
}

func (s *balances) wavesBalance(addr proto.Address) (*balanceProfile, error) {
	key := wavesBalanceKey{address: addr}
	return s.wavesBalanceImpl(key.bytes())
}

func (s *balances) addRecordToLocalStor(key, record []byte, fmt *history.HistoryFormatter, stor map[string][]byte) error {
	history, _ := stor[string(key)]
	history, err := fmt.AddRecord(history, record)
	if err != nil {
		return err
	}
	stor[string(key)] = history
	return nil
}

func (s *balances) setAssetBalance(addr proto.Address, asset []byte, record *assetBalanceRecord) error {
	key := assetBalanceKey{address: addr, asset: asset}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	return s.addRecordToLocalStor(key.bytes(), recordBytes, s.assetFmt, s.assetStor)
}

func (s *balances) setWavesBalanceImpl(key []byte, record *wavesBalanceRecord) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	return s.addRecordToLocalStor(key, recordBytes, s.wavesFmt, s.wavesStor)
}

func (s *balances) setWavesBalance(addr proto.Address, record *wavesBalanceRecord) error {
	key := wavesBalanceKey{address: addr}
	return s.setWavesBalanceImpl(key.bytes(), record)
}

func (s *balances) reset() {
	s.assetStor = make(map[string][]byte)
	s.wavesStor = make(map[string][]byte)
}

func (s *balances) flush() error {
	if err := addHistoryToBatch(s.db, s.dbBatch, s.wavesStor, s.wavesFmt); err != nil {
		return err
	}
	if err := addHistoryToBatch(s.db, s.dbBatch, s.assetStor, s.assetFmt); err != nil {
		return err
	}
	return nil
}
