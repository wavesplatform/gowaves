package state

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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
		return errors.Errorf("wavesBalanceRecord unmarshalBinary: invalid data size, expected %d, found %d", wavesBalanceRecordSize, len(data))
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

type leaseBalanceRecordForHashes struct {
	addr     *proto.Address
	leaseIn  int64
	leaseOut int64
}

func (lc *leaseBalanceRecordForHashes) less(other stateComponent) bool {
	lc2 := other.(*leaseBalanceRecordForHashes)
	return bytes.Compare(lc.addr[:], lc2.addr[:]) == -1
}

func (lc *leaseBalanceRecordForHashes) writeTo(w io.Writer) error {
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

type wavesRecordForHashes struct {
	addr    *proto.Address
	balance uint64
}

func (wc *wavesRecordForHashes) less(other stateComponent) bool {
	wc2 := other.(*wavesRecordForHashes)
	return bytes.Compare(wc.addr[:], wc2.addr[:]) == -1
}

func (wc *wavesRecordForHashes) writeTo(w io.Writer) error {
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

type assetRecordForHashes struct {
	addr    *proto.Address
	asset   []byte
	balance uint64
}

func (ac *assetRecordForHashes) less(other stateComponent) bool {
	ac2 := other.(*assetRecordForHashes)
	val := bytes.Compare(ac.addr[:], ac2.addr[:])
	if val > 0 {
		return false
	} else if val == 0 {
		return bytes.Compare(ac.asset, ac2.asset) == -1
	}
	return true
}

func (ac *assetRecordForHashes) writeTo(w io.Writer) error {
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

	emptyHash         crypto.Digest
	wavesHashesState  map[proto.BlockID]*stateForHashes
	wavesHashes       map[proto.BlockID]crypto.Digest
	assetsHashesState map[proto.BlockID]*stateForHashes
	assetsHashes      map[proto.BlockID]crypto.Digest
	leaseHashesState  map[proto.BlockID]*stateForHashes
	leaseHashes       map[proto.BlockID]crypto.Digest

	calculateHashes bool
}

func newBalances(db keyvalue.IterableKeyVal, hs *historyStorage, calcHashes bool) (*balances, error) {
	emptyHash, err := crypto.FastHash(nil)
	if err != nil {
		return nil, err
	}
	return &balances{
		db:                db,
		hs:                hs,
		calculateHashes:   calcHashes,
		emptyHash:         emptyHash,
		wavesHashesState:  make(map[proto.BlockID]*stateForHashes),
		wavesHashes:       make(map[proto.BlockID]crypto.Digest),
		assetsHashesState: make(map[proto.BlockID]*stateForHashes),
		assetsHashes:      make(map[proto.BlockID]crypto.Digest),
		leaseHashesState:  make(map[proto.BlockID]*stateForHashes),
		leaseHashes:       make(map[proto.BlockID]crypto.Digest),
	}, nil
}

func (s *balances) wavesHashAt(blockID proto.BlockID) crypto.Digest {
	hash, ok := s.wavesHashes[blockID]
	if !ok {
		return s.emptyHash
	}
	return hash
}

func (s *balances) assetsHashAt(blockID proto.BlockID) crypto.Digest {
	hash, ok := s.assetsHashes[blockID]
	if !ok {
		return s.emptyHash
	}
	return hash
}

func (s *balances) leaseHashAt(blockID proto.BlockID) crypto.Digest {
	hash, ok := s.leaseHashes[blockID]
	if !ok {
		return s.emptyHash
	}
	return hash
}

func (s *balances) cancelAllLeases(blockID proto.BlockID) error {
	iter, err := s.hs.newNewestTopEntryIterator(wavesBalance, true)
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
		recordBytes := keyvalue.SafeValue(iter)
		var r wavesBalanceRecord
		if err := r.unmarshalBinary(recordBytes); err != nil {
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
	iter, err := s.hs.newNewestTopEntryIterator(wavesBalance, true)
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
		recordBytes := keyvalue.SafeValue(iter)
		var r wavesBalanceRecord
		if err := r.unmarshalBinary(recordBytes); err != nil {
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
	iter, err := s.hs.newNewestTopEntryIterator(wavesBalance, true)
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
		recordBytes := keyvalue.SafeValue(iter)
		var r wavesBalanceRecord
		if err := r.unmarshalBinary(recordBytes); err != nil {
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

type assetInfoFn func(crypto.Digest, bool) (*assetInfo, error)

func (s *balances) nftList(addr proto.Address, limit uint64, after []byte, assetInfoById assetInfoFn) ([]crypto.Digest, error) {
	key := assetBalanceKey{address: addr}
	iter, err := s.hs.newTopEntryIteratorByPrefix(key.addressPrefix(), true)
	if err != nil {
		return nil, err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	var k assetBalanceKey
	if after != nil {
		// Iterate until `after_asset_id` asset is found.
		afterID, err := crypto.NewDigestFromBytes(after)
		if err != nil {
			return nil, err
		}
		for iter.Next() {
			keyBytes := keyvalue.SafeKey(iter)
			if err := k.unmarshal(keyBytes); err != nil {
				return nil, err
			}
			assetID, err := crypto.NewDigestFromBytes(k.asset)
			if err != nil {
				return nil, err
			}
			if assetID == afterID {
				break
			}
		}
	}
	var r assetBalanceRecord
	var res []crypto.Digest
	for iter.Next() {
		if uint64(len(res)) >= limit {
			break
		}
		recordBytes := keyvalue.SafeValue(iter)
		if err := r.unmarshalBinary(recordBytes); err != nil {
			return nil, err
		}
		if r.balance == 0 {
			continue
		}
		keyBytes := keyvalue.SafeKey(iter)
		if err := k.unmarshal(keyBytes); err != nil {
			return nil, err
		}
		assetID, err := crypto.NewDigestFromBytes(k.asset)
		if err != nil {
			return nil, err
		}
		assetInfo, err := assetInfoById(assetID, true)
		if err != nil {
			return nil, err
		}
		nft := assetInfo.isNFT()
		if nft {
			res = append(res, assetID)
		}
	}
	return res, nil
}

func (s *balances) wavesAddressesNumber() (uint64, error) {
	iter, err := s.hs.newTopEntryIterator(wavesBalance, true)
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
		recordBytes := keyvalue.SafeValue(iter)
		var r wavesBalanceRecord
		if err := r.unmarshalBinary(recordBytes); err != nil {
			return 0, err
		}
		if r.balance > 0 {
			addressesNumber++
		}
	}
	return addressesNumber, nil
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
	if minBalance == math.MaxUint64 {
		// This is the case when records is empty.
		// This actually means that address has no balance records, i.e. it has 0 balance.
		minBalance = 0
	}
	return minBalance, nil
}

func (s *balances) minEffectiveBalanceInRange(addr proto.Address, startHeight, endHeight uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	records, err := s.hs.entriesDataInHeightRange(key.bytes(), startHeight, endHeight, true)
	if err != nil {
		return 0, err
	}
	return s.minEffectiveBalanceInRangeCommon(records)
}

func (s *balances) newestMinEffectiveBalanceInRange(addr proto.Address, startHeight, endHeight uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	records, err := s.hs.newestEntriesDataInHeightRange(key.bytes(), startHeight, endHeight, true)
	if err != nil {
		return 0, err
	}
	return s.minEffectiveBalanceInRangeCommon(records)
}

func (s *balances) assetBalanceFromRecordBytes(recordBytes []byte) (uint64, error) {
	var record assetBalanceRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, err
	}
	return record.balance, nil
}

func (s *balances) assetBalance(addr proto.Address, asset []byte, filter bool) (uint64, error) {
	key := assetBalanceKey{address: addr, asset: asset}
	recordBytes, err := s.hs.topEntryData(key.bytes(), filter)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// Unknown address, expected behavior is to return 0 and no errors in this case.
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return s.assetBalanceFromRecordBytes(recordBytes)
}

func (s *balances) newestAssetBalance(addr proto.Address, asset []byte, filter bool) (uint64, error) {
	key := assetBalanceKey{address: addr, asset: asset}
	recordBytes, err := s.hs.newestTopEntryData(key.bytes(), filter)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// Unknown address, expected behavior is to return 0 and no errors in this case.
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return s.assetBalanceFromRecordBytes(recordBytes)
}

func (s *balances) newestWavesRecord(key []byte, filter bool) (*wavesBalanceRecord, error) {
	recordBytes, err := s.hs.newestTopEntryData(key, filter)
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

func (s *balances) newestWavesBalance(addr proto.Address, filter bool) (*balanceProfile, error) {
	key := wavesBalanceKey{address: addr}
	r, err := s.newestWavesRecord(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	return &r.balanceProfile, nil
}

func (s *balances) wavesRecord(key []byte, filter bool) (*wavesBalanceRecord, error) {
	recordBytes, err := s.hs.topEntryData(key, filter)
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

func (s *balances) wavesBalance(addr proto.Address, filter bool) (*balanceProfile, error) {
	key := wavesBalanceKey{address: addr}
	r, err := s.wavesRecord(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	return &r.balanceProfile, nil
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
		ac := &assetRecordForHashes{
			addr:    &addr,
			asset:   asset,
			balance: balance,
		}
		if _, ok := s.assetsHashesState[blockID]; !ok {
			s.assetsHashesState[blockID] = newStateForHashes()
		}
		s.assetsHashesState[blockID].set(keyStr, ac)
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
			wc := &wavesRecordForHashes{
				addr:    &addr,
				balance: record.balance,
			}
			if _, ok := s.wavesHashesState[blockID]; !ok {
				s.wavesHashesState[blockID] = newStateForHashes()
			}
			s.wavesHashesState[blockID].set(keyStr, wc)
		}
		if balance.leaseChange {
			lc := &leaseBalanceRecordForHashes{
				addr:     &addr,
				leaseIn:  record.leaseIn,
				leaseOut: record.leaseOut,
			}
			if _, ok := s.leaseHashesState[blockID]; !ok {
				s.leaseHashesState[blockID] = newStateForHashes()
			}
			s.leaseHashesState[blockID].set(keyStr, lc)
		}
	}
	return s.hs.addNewEntry(wavesBalance, keyBytes, recordBytes, blockID)
}

func (s *balances) prepareHashes() error {
	for blockID, st := range s.wavesHashesState {
		res, err := st.hash()
		if err != nil {
			return err
		}
		s.wavesHashes[blockID] = res
	}
	for blockID, st := range s.assetsHashesState {
		res, err := st.hash()
		if err != nil {
			return err
		}
		s.assetsHashes[blockID] = res
	}
	for blockID, st := range s.leaseHashesState {
		res, err := st.hash()
		if err != nil {
			return err
		}
		s.leaseHashes[blockID] = res
	}
	return nil
}

func (s *balances) reset() {
	if !s.calculateHashes {
		return
	}
	s.wavesHashesState = make(map[proto.BlockID]*stateForHashes)
	s.wavesHashes = make(map[proto.BlockID]crypto.Digest)
	s.assetsHashesState = make(map[proto.BlockID]*stateForHashes)
	s.assetsHashes = make(map[proto.BlockID]crypto.Digest)
	s.leaseHashesState = make(map[proto.BlockID]*stateForHashes)
	s.leaseHashes = make(map[proto.BlockID]crypto.Digest)
}
