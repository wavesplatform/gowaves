package state

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"go.uber.org/zap"
)

const (
	wavesBalanceRecordSize = 8 + 8 + 8
	assetBalanceRecordSize = 8
)

type wavesValue struct {
	profile       balanceProfile
	leaseChange   bool
	balanceChange bool
}

type balanceProfile struct {
	balance  uint64
	leaseIn  int64
	leaseOut int64
}

func (bp *balanceProfile) effectiveBalance() (uint64, error) {
	val, err := common.AddInt64(int64(bp.balance), bp.leaseIn)
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
}

func (r *wavesBalanceRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, wavesBalanceRecordSize)
	binary.BigEndian.PutUint64(res[:8], r.balance)
	binary.BigEndian.PutUint64(res[8:16], uint64(r.leaseIn))
	binary.BigEndian.PutUint64(res[16:24], uint64(r.leaseOut))
	return res, nil
}

func (r *wavesBalanceRecord) unmarshalBinary(data []byte) error {
	if len(data) != wavesBalanceRecordSize {
		return errInvalidDataSize
	}
	r.balance = binary.BigEndian.Uint64(data[:8])
	r.leaseIn = int64(binary.BigEndian.Uint64(data[8:16]))
	r.leaseOut = int64(binary.BigEndian.Uint64(data[16:24]))
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
		return errInvalidDataSize
	}
	r.balance = binary.BigEndian.Uint64(data[:8])
	return nil
}

type leaseBalanceChangeForHashes struct {
	addr     *proto.Address
	leaseIn  int64
	leaseOut int64
}

func (lc *leaseBalanceChangeForHashes) less(other stateComponent) bool {
	lc2 := other.(*leaseBalanceChangeForHashes)
	return bytes.Compare(lc.addr[:], lc2.addr[:]) == -1
}

func (lc *leaseBalanceChangeForHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(lc.addr[:]); err != nil {
		return err
	}
	leaseInBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(leaseInBytes, uint64(lc.leaseIn))
	if _, err := w.Write(leaseInBytes); err != nil {
		return err
	}
	leaseOutBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(leaseOutBytes, uint64(lc.leaseOut))
	if _, err := w.Write(leaseOutBytes); err != nil {
		return err
	}
	return nil
}

type wavesChangeForHashes struct {
	addr    *proto.Address
	balance uint64
}

func (wc *wavesChangeForHashes) less(other stateComponent) bool {
	wc2 := other.(*wavesChangeForHashes)
	return bytes.Compare(wc.addr[:], wc2.addr[:]) == -1
}

func (wc *wavesChangeForHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(wc.addr[:]); err != nil {
		return err
	}
	balanceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(balanceBytes, wc.balance)
	if _, err := w.Write(balanceBytes); err != nil {
		return err
	}
	return nil
}

type assetChangeForHashes struct {
	addr    *proto.Address
	asset   []byte
	balance uint64
}

func (ac *assetChangeForHashes) less(other stateComponent) bool {
	ac2 := other.(*assetChangeForHashes)
	val := bytes.Compare(ac.addr[:], ac2.addr[:])
	if val > 0 {
		return false
	} else if val == 0 {
		return bytes.Compare(ac.asset, ac2.asset) == -1
	}
	return true
}

func (ac *assetChangeForHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(ac.addr[:]); err != nil {
		return err
	}
	if _, err := w.Write(ac.asset); err != nil {
		return err
	}
	balanceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(balanceBytes, ac.balance)
	if _, err := w.Write(balanceBytes); err != nil {
		return err
	}
	return nil
}

type balances struct {
	db keyvalue.IterableKeyVal
	hs *historyStorage

	wavesHasher  *stateHasher
	assetsHasher *stateHasher
	leaseHasher  *stateHasher

	calculateHashes bool
}

func newBalances(db keyvalue.IterableKeyVal, hs *historyStorage, calcHashes bool) (*balances, error) {
	return &balances{
		db:              db,
		hs:              hs,
		calculateHashes: calcHashes,
		wavesHasher:     newStateHasher(),
		assetsHasher:    newStateHasher(),
		leaseHasher:     newStateHasher(),
	}, nil
}

func (s *balances) cancelAllLeases(blockID proto.BlockID) error {
	iter, err := s.db.NewKeyIterator([]byte{wavesBalanceKeyPrefix})
	if err != nil {
		return err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	for iter.Next() {
		key := keyvalue.SafeKey(iter)
		r, err := s.wavesRecord(key, true)
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
		zap.S().Infof("Resetting lease balance for %s", k.address.String())
		r.leaseOut = 0
		r.leaseIn = 0
		val := &wavesValue{leaseChange: true, profile: r.balanceProfile}
		if err := s.setWavesBalance(k.address, val, blockID); err != nil {
			return err
		}
	}
	return nil
}

func (s *balances) cancelLeaseOverflows(blockID proto.BlockID) (map[proto.Address]struct{}, error) {
	iter, err := s.db.NewKeyIterator([]byte{wavesBalanceKeyPrefix})
	if err != nil {
		return nil, err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
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
			zap.S().Infof("Resolving lease overflow for address %s: %d ---> %d", k.address.String(), r.leaseOut, 0)
			overflowedAddresses[k.address] = empty
			r.leaseOut = 0
			val := &wavesValue{leaseChange: true, profile: r.balanceProfile}
			if err := s.setWavesBalance(k.address, val, blockID); err != nil {
				return nil, err
			}
		}
	}
	return overflowedAddresses, err
}

func (s *balances) cancelInvalidLeaseIns(correctLeaseIns map[proto.Address]int64, blockID proto.BlockID) error {
	iter, err := s.db.NewKeyIterator([]byte{wavesBalanceKeyPrefix})
	if err != nil {
		return err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	zap.S().Infof("Started to cancel invalid leaseIns")
	for iter.Next() {
		key := keyvalue.SafeKey(iter)
		r, err := s.wavesRecord(key, true)
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
			zap.S().Infof("Invalid leaseIn for address %s detected; fixing it: %d ---> %d.", k.address.String(), r.leaseIn, correctLeaseIn)
			r.leaseIn = correctLeaseIn
			val := &wavesValue{leaseChange: true, profile: r.balanceProfile}
			if err := s.setWavesBalance(k.address, val, blockID); err != nil {
				return err
			}
		}
	}
	zap.S().Infof("Finished to cancel invalid leaseIns")
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
			zap.S().Fatalf("Iterator error: %v", err)
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

func (s *balances) effectiveBalanceBeforeHeightCommon(recordBytes []byte) (uint64, error) {
	if recordBytes == nil {
		return 0, nil
	}
	var record wavesBalanceRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, err
	}
	return record.effectiveBalance()
}

func (s *balances) effectiveBalanceBeforeHeightStable(addr proto.Address, height uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	recordBytes, err := s.hs.entryDataBeforeHeight(key.bytes(), height, true)
	if err != nil {
		return 0, err
	}
	return s.effectiveBalanceBeforeHeightCommon(recordBytes)
}

func (s *balances) effectiveBalanceBeforeHeight(addr proto.Address, height uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	recordBytes, err := s.hs.freshEntryDataBeforeHeight(key.bytes(), height, true)
	if err != nil {
		return 0, err
	}
	return s.effectiveBalanceBeforeHeightCommon(recordBytes)
}

func (s *balances) minEffectiveBalanceInRangeCommon(records [][]byte) (uint64, error) {
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
	return minBalance, nil
}

func (s *balances) minEffectiveBalanceInRangeStable(addr proto.Address, startHeight, endHeight uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	records, err := s.hs.entriesDataInHeightRangeStable(key.bytes(), startHeight, endHeight, true)
	if err != nil {
		return 0, err
	}
	minBalance, err := s.minEffectiveBalanceInRangeCommon(records)
	if err != nil {
		return 0, err
	}
	if minBalance == math.MaxUint64 {
		// No balances found at height range, use the latest before startHeight.
		return s.effectiveBalanceBeforeHeightStable(addr, startHeight)
	}
	return minBalance, nil
}

// minEffectiveBalanceInRange() is used to get min miner's effective balance, so it includes blocks which
// have not been flushed to DB yet (and are currently stored in memory).
func (s *balances) minEffectiveBalanceInRange(addr proto.Address, startHeight, endHeight uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	records, err := s.hs.entriesDataInHeightRange(key.bytes(), startHeight, endHeight, true)
	if err != nil {
		return 0, err
	}
	minBalance, err := s.minEffectiveBalanceInRangeCommon(records)
	if err != nil {
		return 0, err
	}
	if minBalance == math.MaxUint64 {
		// No balances found at height range, use the latest before startHeight.
		return s.effectiveBalanceBeforeHeight(addr, startHeight)
	}
	return minBalance, nil
}

func (s *balances) assetBalance(addr proto.Address, asset []byte, filter bool) (uint64, error) {
	key := assetBalanceKey{address: addr, asset: asset}
	recordBytes, err := s.hs.latestEntryData(key.bytes(), filter)
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

func (s *balances) wavesRecord(key []byte, filter bool) (*wavesBalanceRecord, error) {
	recordBytes, err := s.hs.latestEntryData(key, filter)
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

func (s *balances) setAssetBalance(addr proto.Address, asset []byte, balance uint64, blockID proto.BlockID) error {
	key := assetBalanceKey{address: addr, asset: asset}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	record := &assetBalanceRecord{balance}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if s.calculateHashes {
		ac := &assetChangeForHashes{
			addr:    &addr,
			asset:   asset,
			balance: balance,
		}
		if err := s.assetsHasher.push(keyStr, ac, blockID); err != nil {
			return err
		}
	}
	return s.hs.addNewEntry(assetBalance, keyBytes, recordBytes, blockID)
}

func (s *balances) setWavesBalance(addr proto.Address, balance *wavesValue, blockID proto.BlockID) error {
	key := wavesBalanceKey{address: addr}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	record := &wavesBalanceRecord{balance.profile}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if s.calculateHashes {
		if balance.balanceChange {
			wc := &wavesChangeForHashes{
				addr:    &addr,
				balance: record.balance,
			}
			if err := s.wavesHasher.push(keyStr, wc, blockID); err != nil {
				return err
			}
		}
		if balance.leaseChange {
			lc := &leaseBalanceChangeForHashes{
				addr:     &addr,
				leaseIn:  record.leaseIn,
				leaseOut: record.leaseOut,
			}
			if err := s.leaseHasher.push(keyStr, lc, blockID); err != nil {
				return err
			}
		}
	}
	return s.hs.addNewEntry(wavesBalance, keyBytes, recordBytes, blockID)
}

func (s *balances) prepareHashes() error {
	if err := s.wavesHasher.stop(); err != nil {
		return err
	}
	if err := s.assetsHasher.stop(); err != nil {
		return err
	}
	if err := s.leaseHasher.stop(); err != nil {
		return err
	}
	return nil
}

func (s *balances) reset() {
	if !s.calculateHashes {
		return
	}
	s.wavesHasher.reset()
	s.assetsHasher.reset()
	s.leaseHasher.reset()
}
