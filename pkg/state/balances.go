package state

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/common"
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
	val, err := common.AddInt(int64(bp.balance), bp.leaseIn)
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
	addr     *proto.WavesAddress
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
	addr    *proto.WavesAddress
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
	addr    *proto.WavesAddress
	asset   crypto.Digest
	balance uint64
}

func (ac *assetRecordForHashes) less(other stateComponent) bool {
	ac2 := other.(*assetRecordForHashes)
	val := bytes.Compare(ac.addr[:], ac2.addr[:])
	if val > 0 {
		return false
	} else if val == 0 {
		return bytes.Compare(ac.asset[:], ac2.asset[:]) == -1
	}
	return true
}

func (ac *assetRecordForHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(ac.addr[:]); err != nil {
		return err
	}
	if _, err := w.Write(ac.asset[:]); err != nil {
		return err
	}
	balanceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(balanceBytes, ac.balance)
	if _, err := w.Write(balanceBytes); err != nil {
		return err
	}
	return nil
}

type assetInfoGetter interface {
	assetInfo(assetID proto.AssetID) (*assetInfo, error)
	newestConstInfo(assetID proto.AssetID) (*assetConstInfo, error)
}

type balances struct {
	db keyvalue.IterableKeyVal
	hs *historyStorage

	assets assetInfoGetter

	emptyHash         crypto.Digest
	wavesHashesState  map[proto.BlockID]*stateForHashes
	wavesHashes       map[proto.BlockID]crypto.Digest
	assetsHashesState map[proto.BlockID]*stateForHashes
	assetsHashes      map[proto.BlockID]crypto.Digest
	leaseHashesState  map[proto.BlockID]*stateForHashes
	leaseHashes       map[proto.BlockID]crypto.Digest

	calculateHashes bool
	scheme          proto.Scheme
}

func newBalances(db keyvalue.IterableKeyVal, hs *historyStorage, assets assetInfoGetter, scheme proto.Scheme, calcHashes bool) (*balances, error) {
	emptyHash, err := crypto.FastHash(nil)
	if err != nil {
		return nil, err
	}
	return &balances{
		db:                db,
		hs:                hs,
		assets:            assets,
		calculateHashes:   calcHashes,
		scheme:            scheme,
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

func (s *balances) generateZeroLeaseBalanceSnapshotsForAllLeases() ([]proto.LeaseBalanceSnapshot, error) {
	iter, err := s.hs.newNewestTopEntryIterator(wavesBalance)
	if err != nil {
		return nil, err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	var zeroLeaseBalanceSnapshots []proto.LeaseBalanceSnapshot
	for iter.Next() {
		key := keyvalue.SafeKey(iter)
		recordBytes := keyvalue.SafeValue(iter)
		var r wavesBalanceRecord
		if err := r.unmarshalBinary(recordBytes); err != nil {
			return nil, err
		}
		if r.leaseIn == 0 && r.leaseOut == 0 {
			// Empty lease balance, no need to reset.
			continue
		}
		var k wavesBalanceKey
		if err := k.unmarshal(key); err != nil {
			return nil, err
		}
		addr, err := k.address.ToWavesAddress(s.scheme)
		if err != nil {
			return nil, err
		}
		zap.S().Infof("Resetting lease balance for %s", addr.String())
		zeroLeaseBalanceSnapshots = append(zeroLeaseBalanceSnapshots, proto.LeaseBalanceSnapshot{
			Address:  addr,
			LeaseIn:  0,
			LeaseOut: 0,
		})
	}
	return zeroLeaseBalanceSnapshots, nil
}

func (s *balances) generateLeaseBalanceSnapshotsForLeaseOverflows() (
	[]proto.LeaseBalanceSnapshot, map[proto.WavesAddress]struct{}, error,
) {
	iter, err := s.hs.newNewestTopEntryIterator(wavesBalance)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	var leaseBalanceSnapshots []proto.LeaseBalanceSnapshot
	overflowedAddresses := make(map[proto.WavesAddress]struct{})
	for iter.Next() {
		key := keyvalue.SafeKey(iter)
		recordBytes := keyvalue.SafeValue(iter)
		var r wavesBalanceRecord
		if err := r.unmarshalBinary(recordBytes); err != nil {
			return nil, nil, err
		}
		if int64(r.balance) < r.leaseOut {
			var k wavesBalanceKey
			if err := k.unmarshal(key); err != nil {
				return nil, nil, err
			}
			wavesAddr, err := k.address.ToWavesAddress(s.scheme)
			if err != nil {
				return nil, nil, err
			}
			zap.S().Infof("Resolving lease overflow for address %s: %d ---> %d",
				wavesAddr.String(), r.leaseOut, 0,
			)
			overflowedAddresses[wavesAddr] = struct{}{}
			leaseBalanceSnapshots = append(leaseBalanceSnapshots, proto.LeaseBalanceSnapshot{
				Address:  wavesAddr,
				LeaseIn:  uint64(r.leaseIn),
				LeaseOut: 0,
			})
		}
	}
	return leaseBalanceSnapshots, overflowedAddresses, err
}

func (s *balances) generateCorrectingLeaseBalanceSnapshotsForInvalidLeaseIns(
	correctLeaseIns map[proto.WavesAddress]int64,
) ([]proto.LeaseBalanceSnapshot, error) {
	iter, err := s.hs.newNewestTopEntryIterator(wavesBalance)
	if err != nil {
		return nil, err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	var correctLeaseBalanceSnapshots []proto.LeaseBalanceSnapshot
	zap.S().Infof("Started to cancel invalid leaseIns")
	for iter.Next() {
		key := keyvalue.SafeKey(iter)
		recordBytes := keyvalue.SafeValue(iter)
		var r wavesBalanceRecord
		if err := r.unmarshalBinary(recordBytes); err != nil {
			return nil, err
		}
		var k wavesBalanceKey
		if err := k.unmarshal(key); err != nil {
			return nil, err
		}
		correctLeaseIn := int64(0)
		wavesAddress, err := k.address.ToWavesAddress(s.scheme)
		if err != nil {
			return nil, err
		}
		if leaseIn, ok := correctLeaseIns[wavesAddress]; ok {
			correctLeaseIn = leaseIn
		}
		if r.leaseIn != correctLeaseIn {
			zap.S().Infof("Invalid leaseIn for address %s detected; fixing it: %d ---> %d.",
				wavesAddress.String(), r.leaseIn, correctLeaseIn,
			)
			correctLeaseBalanceSnapshots = append(correctLeaseBalanceSnapshots, proto.LeaseBalanceSnapshot{
				Address:  wavesAddress,
				LeaseIn:  uint64(correctLeaseIn),
				LeaseOut: uint64(r.leaseOut),
			})
		}
	}
	zap.S().Infof("Finished to cancel invalid leaseIns")
	return correctLeaseBalanceSnapshots, nil
}

func (s *balances) generateLeaseBalanceSnapshotsWithProvidedChanges(
	changes map[proto.WavesAddress]balanceDiff,
) ([]proto.LeaseBalanceSnapshot, error) {
	zap.S().Infof("Updating balances for cancelled leases")
	leaseBalanceSnapshots := make([]proto.LeaseBalanceSnapshot, 0, len(changes))
	for a, bd := range changes {
		k := wavesBalanceKey{address: a.ID()}
		r, err := s.newestWavesRecord(k.bytes())
		if err != nil {
			return nil, err
		}
		profile := r.balanceProfile
		newProfile, err := bd.applyTo(profile)
		if err != nil {
			return nil, err
		}
		leaseBalanceSnapshots = append(leaseBalanceSnapshots, proto.LeaseBalanceSnapshot{
			Address:  a,
			LeaseIn:  uint64(newProfile.leaseIn),
			LeaseOut: uint64(newProfile.leaseOut),
		})
		zap.S().Infof("Balance of %s changed from (B: %d, LIn: %d, LOut: %d) to (B: %d, lIn: %d, lOut: %d)",
			a.String(), profile.balance, profile.leaseIn, profile.leaseOut,
			newProfile.balance, newProfile.leaseIn, newProfile.leaseOut)
	}
	zap.S().Infof("Finished to update balances")
	return leaseBalanceSnapshots, nil
}

type reducedFeaturesState interface {
	isActivatedAtHeight(featureID int16, height uint64) bool
	activationHeight(featureID int16) (uint64, error)
}

// nftList returns list of NFTs for the given address.
// Since activation of feature #15 this method returns only tokens that are issued
// as NFT (amount: 1, decimal places: 0, reissuable: false) after activation of feature #13.
// Before activation of feature #15 the method returned all the assets that are issued as NFT.
func (s *balances) nftList(
	addr proto.AddressID,
	limit uint64,
	afterAssetID *proto.AssetID, // optional parameter
	height proto.Height,
	feats reducedFeaturesState,
) ([]crypto.Digest, error) {
	blockV5Activated := feats.isActivatedAtHeight(int16(settings.BlockV5), height)
	reducedNFTFeeActivationHeight := uint64(math.MaxUint64) // init with max value if feature is not activated
	if feats.isActivatedAtHeight(int16(settings.ReducedNFTFee), height) {
		var err error
		reducedNFTFeeActivationHeight, err = feats.activationHeight(int16(settings.ReducedNFTFee))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get activation height for ReducedNFTFee feature")
		}
	}

	key := assetBalanceKey{address: addr}
	iter, err := s.hs.newTopEntryIteratorByPrefix(key.addressPrefix())
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
	if afterAssetID != nil {
		// Iterate until `afterAssetID` asset is found.
		target := *afterAssetID
		for iter.Next() {
			keyBytes := keyvalue.SafeKey(iter)
			if err := k.unmarshal(keyBytes); err != nil {
				return nil, err
			}
			if k.asset == target {
				break
			}
		}
	}
	return collectNFTs(iter, s.assets, k, limit, blockV5Activated, reducedNFTFeeActivationHeight)
}

func collectNFTs(
	iter *topEntryIterator,
	assets assetInfoGetter,
	k assetBalanceKey,
	limit uint64,
	blockV5Activated bool,
	reducedNFTFeeActivationHeight uint64,
) ([]crypto.Digest, error) {
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
		ai, aiErr := assets.assetInfo(k.asset)
		if aiErr != nil {
			return nil, aiErr
		}
		if blockV5Activated && ai.IssueHeight < reducedNFTFeeActivationHeight {
			continue // after feature 15 activation we return only NFTs which are issued after feature 13 activation
		}
		nft := ai.IsNFT
		if nft {
			res = append(res, proto.ReconstructDigest(k.asset, ai.Tail))
		}
	}
	return res, nil
}

func (s *balances) wavesAddressesNumber() (uint64, error) {
	iter, err := s.hs.newTopEntryIterator(wavesBalance)
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

func (s *balances) minEffectiveBalanceInRange(addr proto.AddressID, startHeight, endHeight uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	records, err := s.hs.entriesDataInHeightRange(key.bytes(), startHeight, endHeight)
	if err != nil {
		return 0, err
	}
	return s.minEffectiveBalanceInRangeCommon(records)
}

func (s *balances) newestMinEffectiveBalanceInRange(addr proto.AddressID, startHeight, endHeight uint64) (uint64, error) {
	key := wavesBalanceKey{address: addr}
	records, err := s.hs.newestEntriesDataInHeightRange(key.bytes(), startHeight, endHeight)
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

func (s *balances) assetBalance(addr proto.AddressID, assetID proto.AssetID) (uint64, error) {
	key := assetBalanceKey{address: addr, asset: assetID}
	recordBytes, err := s.hs.topEntryData(key.bytes())
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// Unknown address, expected behavior is to return 0 and no errors in this case.
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return s.assetBalanceFromRecordBytes(recordBytes)
}

func (s *balances) newestAssetBalance(addr proto.AddressID, asset proto.AssetID) (uint64, error) {
	key := assetBalanceKey{address: addr, asset: asset}
	recordBytes, err := s.hs.newestTopEntryData(key.bytes())
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// Unknown address, expected behavior is to return 0 and no errors in this case.
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return s.assetBalanceFromRecordBytes(recordBytes)
}

func (s *balances) newestWavesRecord(key []byte) (wavesBalanceRecord, error) {
	recordBytes, err := s.hs.newestTopEntryData(key)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// Unknown address, expected behavior is to return empty profile and no errors in this case.
		return wavesBalanceRecord{}, nil
	} else if err != nil {
		return wavesBalanceRecord{}, err
	}
	var record wavesBalanceRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return wavesBalanceRecord{}, err
	}
	return record, nil
}

// newestWavesBalance returns newest waves balanceProfile.
func (s *balances) newestWavesBalance(addr proto.AddressID) (balanceProfile, error) {
	key := wavesBalanceKey{address: addr}
	r, err := s.newestWavesRecord(key.bytes())
	if err != nil {
		return balanceProfile{}, err
	}
	return r.balanceProfile, nil
}

func (s *balances) wavesRecord(key []byte) (wavesBalanceRecord, error) {
	recordBytes, err := s.hs.topEntryData(key)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// Unknown address, expected behavior is to return empty profile and no errors in this case.
		return wavesBalanceRecord{}, nil
	} else if err != nil {
		return wavesBalanceRecord{}, err
	}
	var record wavesBalanceRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return wavesBalanceRecord{}, errors.Wrap(err, "failed to unmarshal data to %T")
	}
	return record, nil
}

// wavesBalance returns stored waves balanceProfile.
// IMPORTANT NOTE: this method returns saved on disk data, for the newest data use newestWavesBalance.
func (s *balances) wavesBalance(addr proto.AddressID) (balanceProfile, error) {
	key := wavesBalanceKey{address: addr}
	r, err := s.wavesRecord(key.bytes())
	if err != nil {
		return balanceProfile{}, err
	}
	return r.balanceProfile, nil
}

func (s *balances) calculateStateHashesAssetBalance(addr proto.AddressID, assetID proto.AssetID,
	balance uint64, blockID proto.BlockID, keyStr string) error {
	info, err := s.assets.newestConstInfo(assetID)
	if err != nil {
		return err
	}
	wavesAddress, err := addr.ToWavesAddress(s.scheme)
	if err != nil {
		return err
	}
	fullAssetID := proto.ReconstructDigest(assetID, info.Tail)
	ac := &assetRecordForHashes{
		addr:    &wavesAddress,
		asset:   fullAssetID,
		balance: balance,
	}
	if _, ok := s.assetsHashesState[blockID]; !ok {
		s.assetsHashesState[blockID] = newStateForHashes()
	}
	s.assetsHashesState[blockID].set(keyStr, ac)
	return nil
}

func (s *balances) setAssetBalance(addr proto.AddressID, assetID proto.AssetID, balance uint64, blockID proto.BlockID) error {
	key := assetBalanceKey{address: addr, asset: assetID}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	record := assetBalanceRecord{balance}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if s.calculateHashes {
		shErr := s.calculateStateHashesAssetBalance(addr, assetID, balance, blockID, keyStr)
		if shErr != nil {
			return shErr
		}
	}
	return s.hs.addNewEntry(assetBalance, keyBytes, recordBytes, blockID)
}

func (s *balances) calculateStateHashesWavesBalance(addr proto.AddressID, balance wavesValue,
	blockID proto.BlockID, keyStr string, record wavesBalanceRecord) error {
	wavesAddress, err := addr.ToWavesAddress(s.scheme)
	if err != nil {
		return err
	}
	if balance.balanceChange {
		wc := &wavesRecordForHashes{
			addr:    &wavesAddress,
			balance: record.balance,
		}
		if _, ok := s.wavesHashesState[blockID]; !ok {
			s.wavesHashesState[blockID] = newStateForHashes()
		}
		s.wavesHashesState[blockID].set(keyStr, wc)
	}
	if balance.leaseChange {
		lc := &leaseBalanceRecordForHashes{
			addr:     &wavesAddress,
			leaseIn:  record.leaseIn,
			leaseOut: record.leaseOut,
		}
		if _, ok := s.leaseHashesState[blockID]; !ok {
			s.leaseHashesState[blockID] = newStateForHashes()
		}
		s.leaseHashesState[blockID].set(keyStr, lc)
	}
	return nil
}

func (s *balances) setWavesBalance(addr proto.AddressID, balance wavesValue, blockID proto.BlockID) error {
	key := wavesBalanceKey{address: addr}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	record := wavesBalanceRecord{balance.profile}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if s.calculateHashes {
		shErr := s.calculateStateHashesWavesBalance(addr, balance, blockID, keyStr, record)
		if shErr != nil {
			return shErr
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
