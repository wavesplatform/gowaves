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
	wavesBalanceRecordSize = 8 + 8 + 8
	assetBalanceRecordSize = 8
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

/* TODO: unused code, need to write tests if it is needed or otherwise remove it.
func (bp *balanceProfile) spendableBalance() uint64 {
	return uint64(int64(bp.balance) - bp.leaseOut)
}
*/

type wavesBalanceRecord struct {
	balanceProfile
}

func (r *wavesBalanceRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, wavesBalanceRecordSize)
	binary.BigEndian.PutUint64(res[:8], r.balance)
	binary.PutVarint(res[8:16], r.leaseIn)
	binary.PutVarint(res[16:24], r.leaseOut)
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
	return nil
}

type assetBalanceRecord struct {
	balance uint64
}

func (r *assetBalanceRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, assetBalanceRecordSize)
	binary.BigEndian.PutUint64(res[:8], r.balance)
	return res, nil
}

func (r *assetBalanceRecord) unmarshalBinary(data []byte) error {
	if len(data) != assetBalanceRecordSize {
		return errors.New("invalid data size")
	}
	r.balance = binary.BigEndian.Uint64(data[:8])
	return nil
}

type balances struct {
	db keyvalue.IterableKeyVal
	hs *historyStorage
}

func newBalances(db keyvalue.IterableKeyVal, hs *historyStorage) (*balances, error) {
	return &balances{db, hs}, nil
}

func (s *balances) cancelAllLeases() error {
	// TODO: this action can not be rolled back now, do we need it?
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
		r, err := s.wavesRecord(key, false, true)
		if err != nil {
			return err
		}
		if r.leaseIn == 0 && r.leaseOut == 0 {
			// Empty lease balance, no need to reset.
			continue
		}
		var k wavesBalanceKey
		if err := k.unmarshal(key); err != nil {
			return err
		}
		log.Printf("Resetting lease balance for %s", k.address.String())
		r.leaseOut = 0
		r.leaseIn = 0
		if err := s.setWavesBalanceImpl(key, r, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *balances) cancelLeaseOverflows() (map[proto.Address]struct{}, error) {
	// TODO: this action can not be rolled back now, do we need it?
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
		r, err := s.wavesRecord(key, false, true)
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
		if err := s.setWavesBalanceImpl(key, r, nil); err != nil {
			return nil, err
		}
	}
	return overflowedAddresses, err
}

func (s *balances) cancelInvalidLeaseIns(correctLeaseIns map[proto.Address]int64) error {
	// TODO: this action can not be rolled back now, do we need it?
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

	log.Printf("Started to cancel invalid leaseIns\n")
	for iter.Next() {
		key := keyvalue.SafeKey(iter)
		r, err := s.wavesRecord(key, false, true)
		if err != nil {
			return err
		}
		var k wavesBalanceKey
		if err := k.unmarshal(key); err != nil {
			return err
		}
		correctLeaseIn := int64(0)
		if leaseIn, ok := correctLeaseIns[k.address]; ok {
			correctLeaseIn = leaseIn
		}
		if r.leaseIn != correctLeaseIn {
			log.Printf("Invalid leaseIn for address %s detected; fixing it: %d ---> %d.", k.address.String(), r.leaseIn, correctLeaseIn)
			r.leaseIn = correctLeaseIn
			if err := s.setWavesBalanceImpl(key, r, nil); err != nil {
				return err
			}
		}
	}
	log.Printf("Finished to cancel invalid leaseIns\n")
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
		profile, err := s.wavesBalanceImpl(iter.Key(), false, true)
		if err != nil {
			return 0, err
		}
		if profile.balance > 0 {
			addressesNumber++
		}
	}
	return addressesNumber, nil
}

func (s *balances) effectiveBalanceBeforeHeight(addr proto.Address, height uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	recordBytes, err := s.hs.freshEntryDataBeforeHeight(key.bytes(), height, true)
	if err != nil {
		return 0, err
	}
	if recordBytes == nil {
		return 0, nil
	}
	var record wavesBalanceRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, err
	}
	return record.effectiveBalance()
}

// minEffectiveBalanceInRange() is used to get min miner's effective balance, so it includes blocks which
// have not been flushed to DB yet (and are currently stored in memory).
func (s *balances) minEffectiveBalanceInRange(addr proto.Address, startHeight, endHeight uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	records, err := s.hs.entriesDataInHeightRange(key.bytes(), startHeight, endHeight, true)
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
		// No balances found at height range, use the latest before startHeight.
		return s.effectiveBalanceBeforeHeight(addr, startHeight)
	}
	return minBalance, nil
}

func (s *balances) assetBalanceImpl(addr proto.Address, asset []byte, newest, filter bool) (uint64, error) {
	var recordBytes []byte
	var err error
	key := assetBalanceKey{address: addr, asset: asset}
	if newest {
		recordBytes, err = s.hs.freshLatestEntryData(key.bytes(), filter)
	} else {
		recordBytes, err = s.hs.latestEntryData(key.bytes(), filter)
	}
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// Unknown address, expected behavior is to return 0 and no errors in this case.
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	var record assetBalanceRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, err
	}
	return record.balance, nil
}

func (s *balances) newestAssetBalance(addr proto.Address, asset []byte, filter bool) (uint64, error) {
	return s.assetBalanceImpl(addr, asset, true, filter)
}

func (s *balances) assetBalance(addr proto.Address, asset []byte, filter bool) (uint64, error) {
	return s.assetBalanceImpl(addr, asset, false, filter)
}

func (s *balances) wavesRecord(key []byte, newest, filter bool) (*wavesBalanceRecord, error) {
	var recordBytes []byte
	var err error
	if newest {
		recordBytes, err = s.hs.freshLatestEntryData(key, filter)
	} else {
		recordBytes, err = s.hs.latestEntryData(key, filter)
	}
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// Unknown address, expected behavior is to return empty profile and no errors in this case.
		return &wavesBalanceRecord{}, nil
	} else if err != nil {
		return nil, err
	}
	var record wavesBalanceRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *balances) wavesBalanceImpl(key []byte, newest, filter bool) (*balanceProfile, error) {
	r, err := s.wavesRecord(key, newest, filter)
	if err != nil {
		return nil, err
	}
	return &r.balanceProfile, nil
}

func (s *balances) newestWavesBalance(addr proto.Address, filter bool) (*balanceProfile, error) {
	key := wavesBalanceKey{address: addr}
	return s.wavesBalanceImpl(key.bytes(), true, filter)
}

func (s *balances) wavesBalance(addr proto.Address, filter bool) (*balanceProfile, error) {
	key := wavesBalanceKey{address: addr}
	return s.wavesBalanceImpl(key.bytes(), false, filter)
}

func (s *balances) setAssetBalance(addr proto.Address, asset []byte, balance uint64, blockID *crypto.Signature) error {
	key := assetBalanceKey{address: addr, asset: asset}
	record := &assetBalanceRecord{balance}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	return s.hs.addNewEntryWithBlockID(assetBalance, key.bytes(), recordBytes, blockID)
}

func (s *balances) setWavesBalanceImpl(key []byte, record *wavesBalanceRecord, blockID *crypto.Signature) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	return s.hs.addNewEntryWithBlockID(wavesBalance, key, recordBytes, blockID)
}

func (s *balances) setWavesBalance(addr proto.Address, profile *balanceProfile, blockID *crypto.Signature) error {
	key := wavesBalanceKey{address: addr}
	record := &wavesBalanceRecord{*profile}
	return s.setWavesBalanceImpl(key.bytes(), record, blockID)
}
