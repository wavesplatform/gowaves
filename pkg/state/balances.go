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
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	wavesBalanceRecordSize = 8 + 8 + 8 + 4
	assetBalanceRecordSize = 8 + 4
)

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
	blockNum uint32
}

func (r *wavesBalanceRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, wavesBalanceRecordSize)
	binary.BigEndian.PutUint64(res[:8], r.balance)
	binary.PutVarint(res[8:16], r.leaseIn)
	binary.PutVarint(res[16:24], r.leaseOut)
	binary.BigEndian.PutUint32(res[24:], r.blockNum)
	return res, nil
}

func (r *wavesBalanceRecord) unmarshalBinary(data []byte) error {
	if len(data) != wavesBalanceRecordSize {
		return errors.New("invalid data size")
	}
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
	r.blockNum = binary.BigEndian.Uint32(data[24:])
	return nil
}

type assetBalanceRecord struct {
	balance  uint64
	blockNum uint32
}

func (r *assetBalanceRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, assetBalanceRecordSize)
	binary.BigEndian.PutUint64(res[:8], r.balance)
	binary.BigEndian.PutUint32(res[8:], r.blockNum)
	return res, nil
}

func (r *assetBalanceRecord) unmarshalBinary(data []byte) error {
	if len(data) != assetBalanceRecordSize {
		return errors.New("invalid data size")
	}
	r.balance = binary.BigEndian.Uint64(data[:8])
	r.blockNum = binary.BigEndian.Uint32(data[8:])
	return nil
}

type balances struct {
	db      keyvalue.IterableKeyVal
	stateDB *stateDB
	hs      *historyStorage
}

func newBalances(db keyvalue.IterableKeyVal, stateDB *stateDB, hs *historyStorage) (*balances, error) {
	return &balances{db, stateDB, hs}, nil
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
		key := keyvalue.SafeKey(iter)
		r, err := s.wavesRecord(key, true)
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
		key := keyvalue.SafeKey(iter)
		r, err := s.wavesRecord(key, true)
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
		r, err := s.wavesRecord(k.bytes(), true)
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
		profile, err := s.wavesBalanceImpl(iter.Key(), true)
		if err != nil {
			return 0, err
		}
		if profile.balance > 0 {
			addressesNumber++
		}
	}
	return addressesNumber, nil
}

// minEffectiveBalanceInRange() is used to get min miner's effective balance, so it includes blocks which
// have not been flushed to DB yet (and are currently stored in memory).
func (s *balances) minEffectiveBalanceInRange(addr proto.Address, startHeight, endHeight uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	records, err := s.hs.recordsInHeightRange(wavesBalance, key.bytes(), startHeight, endHeight, true)
	if err != nil {
		return 0, err
	}
	minBalance := uint64(math.MaxUint64)
	for _, recordBytes := range records {
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

func (s *balances) assetBalance(addr proto.Address, asset []byte, filter bool) (uint64, error) {
	key := assetBalanceKey{address: addr, asset: asset}
	recordBytes, err := s.hs.get(assetBalance, key.bytes(), filter)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// Unknown address, expected behavior is to return 0 and no errors in this case.
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var record assetBalanceRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, err
	}
	return record.balance, nil
}

func (s *balances) wavesRecord(key []byte, filter bool) (*wavesBalanceRecord, error) {
	recordBytes, err := s.hs.get(wavesBalance, key, filter)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// Unknown address, expected behavior is to return empty profile and no errors in this case.
		return &wavesBalanceRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	var record wavesBalanceRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *balances) wavesBalanceImpl(key []byte, filter bool) (*balanceProfile, error) {
	r, err := s.wavesRecord(key, filter)
	if err != nil {
		return nil, err
	}
	return &r.balanceProfile, nil
}

func (s *balances) wavesBalance(addr proto.Address, filter bool) (*balanceProfile, error) {
	key := wavesBalanceKey{address: addr}
	return s.wavesBalanceImpl(key.bytes(), filter)
}

func (s *balances) setAssetBalance(addr proto.Address, asset []byte, balance uint64, blockID crypto.Signature) error {
	key := assetBalanceKey{address: addr, asset: asset}
	blockNum, err := s.stateDB.blockIdToNum(blockID)
	if err != nil {
		return err
	}
	record := &assetBalanceRecord{balance, blockNum}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	return s.hs.set(assetBalance, key.bytes(), recordBytes)
}

func (s *balances) setWavesBalanceImpl(key []byte, record *wavesBalanceRecord) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	return s.hs.set(wavesBalance, key, recordBytes)
}

func (s *balances) setWavesBalance(addr proto.Address, profile *balanceProfile, blockID crypto.Signature) error {
	key := wavesBalanceKey{address: addr}
	blockNum, err := s.stateDB.blockIdToNum(blockID)
	if err != nil {
		return err
	}
	record := &wavesBalanceRecord{*profile, blockNum}
	return s.setWavesBalanceImpl(key.bytes(), record)
}
