package state

import (
	"bytes"
	"encoding/binary"
	"io"
	"log/slog"
	"math"
	"sort"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/logging"
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

// effectiveBalanceUnchecked returns effective balance without checking for account challenging.
// The function MUST be used ONLY in the context where account challenging IS CHECKED.
func (bp *balanceProfile) effectiveBalanceUnchecked() (uint64, error) {
	val, err := common.AddInt(int64(bp.balance), bp.leaseIn)
	if err != nil {
		return 0, err
	}
	return uint64(val - bp.leaseOut), nil
}

type challengedChecker func(proto.AddressID, proto.Height) (bool, error)

func (bp *balanceProfile) effectiveBalance(
	challengedCheck challengedChecker, // Function to check if the account is challenged.
	addrID proto.AddressID, // Address ID of the current balanceProfile.
	currentHeight proto.Height, // Current height.
) (uint64, error) {
	challenged, err := challengedCheck(addrID, currentHeight)
	if err != nil {
		return 0, err
	}
	if challenged {
		return 0, nil // Challenged account has 0 effective balance.
	}
	return bp.effectiveBalanceUnchecked()
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

type heights []proto.Height

func (h heights) Len() int { return len(h) }

func (h heights) Less(i, j int) bool { return h[i] < h[j] }

func (h heights) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h heights) Last() proto.Height {
	if len(h) == 0 {
		return 0
	}
	return h[len(h)-1]
}

type challengedAddressRecord struct {
	Heights heights `cbor:"0,keyasint"`
}

func (c *challengedAddressRecord) marshalBinary() ([]byte, error) { return cbor.Marshal(c) }

func (c *challengedAddressRecord) unmarshalBinary(data []byte) error { return cbor.Unmarshal(data, c) }

func (c *challengedAddressRecord) appendHeight(height proto.Height) {
	prevLast := c.Heights.Last()
	c.Heights = append(c.Heights, height)
	if prevLast > height { // Heights are not sorted in ascending order.
		sort.Sort(c.Heights)
	}
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
	sets            *settings.BlockchainSettings
}

func newBalances(
	db keyvalue.IterableKeyVal,
	hs *historyStorage,
	assets assetInfoGetter,
	sets *settings.BlockchainSettings,
	calcHashes bool,
) (*balances, error) {
	emptyHash, err := crypto.FastHash(nil)
	if err != nil {
		return nil, err
	}
	return &balances{
		db:                db,
		hs:                hs,
		assets:            assets,
		calculateHashes:   calcHashes,
		sets:              sets,
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
		if itErr := iter.Error(); itErr != nil {
			slog.Error("Iterator error", logging.Error(itErr))
			panic(itErr)
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
		addr, waErr := k.address.ToWavesAddress(s.sets.AddressSchemeCharacter)
		if waErr != nil {
			return nil, waErr
		}
		slog.Info("Resetting lease balance", "address", addr.String())
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
		if itErr := iter.Error(); itErr != nil {
			slog.Error("Iterator error", logging.Error(itErr))
			panic(itErr)
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
			wavesAddr, waErr := k.address.ToWavesAddress(s.sets.AddressSchemeCharacter)
			if waErr != nil {
				return nil, nil, waErr
			}
			slog.Info("Resolving lease overflow", "address", wavesAddr.String(), "old", r.leaseOut, "new", 0)
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
		if itErr := iter.Error(); itErr != nil {
			slog.Error("Iterator error", logging.Error(itErr))
			panic(itErr)
		}
	}()

	var correctLeaseBalanceSnapshots []proto.LeaseBalanceSnapshot
	slog.Info("Started to cancel invalid leaseIns")
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
		wavesAddress, waErr := k.address.ToWavesAddress(s.sets.AddressSchemeCharacter)
		if waErr != nil {
			return nil, waErr
		}
		if leaseIn, ok := correctLeaseIns[wavesAddress]; ok {
			correctLeaseIn = leaseIn
		}
		if r.leaseIn != correctLeaseIn {
			slog.Info("Invalid leaseIn detected; fixing it", "address", wavesAddress.String(),
				"invalid", r.leaseIn, "correct", correctLeaseIn)
			correctLeaseBalanceSnapshots = append(correctLeaseBalanceSnapshots, proto.LeaseBalanceSnapshot{
				Address:  wavesAddress,
				LeaseIn:  uint64(correctLeaseIn),
				LeaseOut: uint64(r.leaseOut),
			})
		}
	}
	slog.Info("Finished to cancel invalid leaseIns")
	return correctLeaseBalanceSnapshots, nil
}

func (s *balances) generateLeaseBalanceSnapshotsWithProvidedChanges(
	changes map[proto.WavesAddress]balanceDiff,
) ([]proto.LeaseBalanceSnapshot, error) {
	slog.Info("Updating balances for cancelled leases")
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
		slog.Info("Balance of changed", "address", a.String(),
			"oldBalance", profile.balance, "oldLIn", profile.leaseIn, "oldLOut", profile.leaseOut,
			"newBalance", newProfile.balance, "newLIn", newProfile.leaseIn, "newLOut", newProfile.leaseOut)
	}
	slog.Info("Finished to update balances")
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
		if itErr := iter.Error(); itErr != nil {
			slog.Error("Iterator error", logging.Error(itErr))
			panic(itErr)
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
		if itErr := iter.Error(); itErr != nil {
			slog.Error("Iterator error", logging.Error(itErr))
			panic(itErr)
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

func minEffectiveBalanceInRangeCommon(records [][]byte) (uint64, error) {
	minBalance := uint64(math.MaxUint64)
	for _, recordBytes := range records {
		var record wavesBalanceRecord
		if err := record.unmarshalBinary(recordBytes); err != nil {
			return 0, err
		}
		effectiveBal, err := record.effectiveBalanceUnchecked()
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

func (s *balances) generatingBalance(addr proto.AddressID, height proto.Height) (uint64, error) {
	startHeight, endHeight := s.sets.RangeForGeneratingBalanceByHeight(height)
	gb, err := s.minEffectiveBalanceInRange(addr, startHeight, endHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get min effective balance with startHeight %d and endHeight %d",
			startHeight, endHeight)
	}
	return gb, nil
}

func (s *balances) newestGeneratingBalance(addr proto.AddressID, height proto.Height) (uint64, error) {
	startHeight, endHeight := s.sets.RangeForGeneratingBalanceByHeight(height)
	gb, err := s.newestMinEffectiveBalanceInRange(addr, startHeight, endHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get newest min effective balance with startHeight %d and endHeight %d",
			startHeight, endHeight)
	}
	return gb, nil
}

func (s *balances) storeChallenge(
	challenger, challenged proto.AddressID,
	challengedBlockHeight proto.Height, // Height of the block with the challenged header.
	blockID proto.BlockID,
) error {
	// Check if challenger and challenged addresses are the same. Self-challenge is not allowed.
	if challenger.Equal(challenged) {
		return errors.New("challenger and challenged addresses are the same")
	}
	if err := s.storeChallengeHeightForAddr(challenged, challengedBlockHeight, blockID); err != nil {
		return errors.Wrapf(err, "failed to store challenge height for challenged at height %d",
			challengedBlockHeight,
		)
	}
	return nil
}

// storeChallengeHeightForAddr stores the height of the block at which the address was challenged.
func (s *balances) storeChallengeHeightForAddr(
	challenged proto.AddressID,
	challengedBlockHeight proto.Height,
	blockID proto.BlockID,
) error {
	key := challengedAddressKey{address: challenged}
	keyBytes := key.bytes()
	recordBytes, err := s.hs.newestTopEntryData(keyBytes)
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) { // No record found, create new one.
			r := challengedAddressRecord{Heights: []proto.Height{challengedBlockHeight}}
			data, mErr := r.marshalBinary()
			if mErr != nil {
				return errors.Wrap(mErr, "failed to marshal record to binary data")
			}
			return s.hs.addNewEntry(challengedAddress, keyBytes, data, blockID)
		}
		return err
	}
	var r challengedAddressRecord
	if uErr := r.unmarshalBinary(recordBytes); uErr != nil {
		return errors.Wrap(uErr, "failed to unmarshal record from binary data")
	}
	r.appendHeight(challengedBlockHeight) // Append new height to the list.
	recordBytes, err = r.marshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to marshal record to binary data")
	}
	return s.hs.addNewEntry(challengedAddress, keyBytes, recordBytes, blockID)
}

type entryDataGetter func(key []byte) ([]byte, error)

// isChallengedAddressInRangeCommon checks if the address was challenged in the given range of heights.
// startHeight and endHeight are inclusive.
func isChallengedAddressInRangeCommon(
	getEntryData entryDataGetter,
	addr proto.AddressID,
	startHeight, endHeight proto.Height,
) (bool, error) {
	key := challengedAddressKey{address: addr}
	recordBytes, err := getEntryData(key.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return false, nil
		}
		return false, err
	}
	var r challengedAddressRecord
	if ubErr := r.unmarshalBinary(recordBytes); ubErr != nil {
		return false, errors.Wrapf(ubErr, "failed to unmarshal entry data to %T", r)
	}
	// assume that heights are sorted in ascending order
	for i := len(r.Heights) - 1; i >= 0; i-- { // iterate in reverse order
		h := r.Heights[i]
		if h < startHeight { // fast path: if h < startHeight, then all other heights are also less than startHeight
			return false, nil
		}
		if startHeight <= h && h <= endHeight {
			return true, nil
		}
	}
	return false, nil
}

func (s *balances) isChallengedAddressInRange(addr proto.AddressID, startHeight, endHeight proto.Height) (bool, error) {
	return isChallengedAddressInRangeCommon(s.hs.topEntryData, addr, startHeight, endHeight)
}

func (s *balances) isChallengedAddress(addr proto.AddressID, height proto.Height) (bool, error) {
	startHeight, endHeight := height, height // we're checking only one height, so the heights are the same
	return s.isChallengedAddressInRange(addr, startHeight, endHeight)
}

func (s *balances) newestIsChallengedAddressInRange(
	addr proto.AddressID,
	startHeight, endHeight proto.Height,
) (bool, error) {
	return isChallengedAddressInRangeCommon(s.hs.newestTopEntryData, addr, startHeight, endHeight)
}

func (s *balances) newestIsChallengedAddress(addr proto.AddressID, height proto.Height) (bool, error) {
	startHeight, endHeight := height, height // we're checking only one height, so the heights are the same
	return s.newestIsChallengedAddressInRange(addr, startHeight, endHeight)
}

// minEffectiveBalanceInRange returns minimal effective balance in range [startHeight, endHeight].
//
// IMPORTANT NOTE: this method returns saved on disk data, for the newest data use newestMinEffectiveBalanceInRange.
//
// For getting the generating balance, use generatingBalance.
//
// If address is not challenged, then we can get the minimal effective balance.
// Though if startHeight == endHeight and addr was a challenger at this height, then we still should return
// its effective balance without a challenger bonus.
// This is because the bonus is applied only to the generating balance at the given height.
func (s *balances) minEffectiveBalanceInRange(addr proto.AddressID, startHeight, endHeight uint64) (uint64, error) {
	isChallengedAddr, err := s.isChallengedAddressInRange(addr, startHeight, endHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to check if address is challenged")
	}
	if isChallengedAddr { // Address is challenged, return 0 intentionally.
		return 0, nil
	}
	key := wavesBalanceKey{address: addr}
	records, err := s.hs.entriesDataInHeightRange(key.bytes(), startHeight, endHeight)
	if err != nil {
		return 0, err
	}
	return minEffectiveBalanceInRangeCommon(records)
}

// newestMinEffectiveBalanceInRange returns minimal effective balance in range [startHeight, endHeight].
//
// For getting the newest generating balance, use newestGeneratingBalance.
//
// If address is not challenged, then we can get the minimal effective balance.
// Though if startHeight == endHeight and addr was a challenger at this height, then we still should return
// its effective balance without a challenger bonus.
// This is because the bonus is applied only to the generating balance at the given height.
func (s *balances) newestMinEffectiveBalanceInRange(addr proto.AddressID, startHeight, endHeight uint64) (uint64, error) {
	isChallengedAddr, err := s.newestIsChallengedAddressInRange(addr, startHeight, endHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to check if address is challenged")
	}
	if isChallengedAddr { // Address is challenged, return 0 intentionally.
		return 0, nil
	}
	key := wavesBalanceKey{address: addr}
	records, err := s.hs.newestEntriesDataInHeightRange(key.bytes(), startHeight, endHeight)
	if err != nil {
		return 0, err
	}
	return minEffectiveBalanceInRangeCommon(records)
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
	if errors.Is(err, keyvalue.ErrNotFound) || errors.Is(err, errEmptyHist) {
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
	if errors.Is(err, keyvalue.ErrNotFound) || errors.Is(err, errEmptyHist) {
		// Unknown address, expected behavior is to return 0 and no errors in this case.
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return s.assetBalanceFromRecordBytes(recordBytes)
}

func (s *balances) newestWavesRecord(key []byte) (wavesBalanceRecord, error) {
	recordBytes, err := s.hs.newestTopEntryData(key)
	if errors.Is(err, keyvalue.ErrNotFound) || errors.Is(err, errEmptyHist) {
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
	if errors.Is(err, keyvalue.ErrNotFound) || errors.Is(err, errEmptyHist) {
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
	wavesAddress, err := addr.ToWavesAddress(s.sets.AddressSchemeCharacter)
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
	wavesAddress, err := addr.ToWavesAddress(s.sets.AddressSchemeCharacter)
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
