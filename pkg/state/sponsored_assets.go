package state

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	sponsorshipRecordSize = 8
)

type sponsorshipRecordForHashes struct {
	id   *crypto.Digest
	cost uint64
}

func (sr *sponsorshipRecordForHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(sr.id[:]); err != nil {
		return err
	}
	costBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(costBytes, sr.cost)
	if _, err := w.Write(costBytes); err != nil {
		return err
	}
	return nil
}

func (sr *sponsorshipRecordForHashes) less(other stateComponent) bool {
	sr2 := other.(*sponsorshipRecordForHashes)
	return bytes.Compare(sr.id[:], sr2.id[:]) == -1
}

type sponsorshipRecord struct {
	// Cost in assets equal to FeeUnit Waves.
	assetCost uint64
}

func (s *sponsorshipRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, sponsorshipRecordSize)
	binary.BigEndian.PutUint64(res[:8], s.assetCost)
	return res, nil
}

func (s *sponsorshipRecord) unmarshalBinary(data []byte) error {
	if len(data) != sponsorshipRecordSize {
		return errInvalidDataSize
	}
	s.assetCost = binary.BigEndian.Uint64(data[:8])
	return nil
}

type sponsoredAssets struct {
	rw       *blockReadWriter
	features *features
	hs       *historyStorage
	settings *settings.BlockchainSettings

	uncertainSponsoredAssets map[crypto.Digest]uint64

	calculateHashes bool
	hasher          *stateHasher
}

func newSponsoredAssets(
	rw *blockReadWriter,
	features *features,
	hs *historyStorage,
	settings *settings.BlockchainSettings,
	calcHashes bool,
) *sponsoredAssets {
	return &sponsoredAssets{
		rw:                       rw,
		features:                 features,
		hs:                       hs,
		settings:                 settings,
		uncertainSponsoredAssets: make(map[crypto.Digest]uint64),
		hasher:                   newStateHasher(),
		calculateHashes:          calcHashes,
	}
}

func (s *sponsoredAssets) sponsorAsset(assetID crypto.Digest, assetCost uint64, blockID proto.BlockID) error {
	key := sponsorshipKey{assetID}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	record := &sponsorshipRecord{assetCost}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if s.calculateHashes {
		sr := &sponsorshipRecordForHashes{
			id:   &assetID,
			cost: assetCost,
		}
		if err := s.hasher.push(keyStr, sr, blockID); err != nil {
			return err
		}
	}
	if err := s.hs.addNewEntry(sponsorship, keyBytes, recordBytes, blockID); err != nil {
		return err
	}
	return nil
}

func (s *sponsoredAssets) sponsorAssetUncertain(assetID crypto.Digest, assetCost uint64) {
	s.uncertainSponsoredAssets[assetID] = assetCost
}

func (s *sponsoredAssets) newestIsSponsored(assetID crypto.Digest, filter bool) (bool, error) {
	cost, err := s.newestAssetCost(assetID, filter)
	if err != nil {
		return false, nil
	}
	if cost == 0 {
		// 0 cost means that asset isn't really sponsored anymore.
		return false, nil
	}
	return true, nil
}

func (s *sponsoredAssets) isSponsored(assetID crypto.Digest, filter bool) (bool, error) {
	key := sponsorshipKey{assetID}
	if _, err := s.hs.topEntryData(key.bytes(), filter); err != nil {
		// No sponsorship info for this asset at all.
		return false, nil
	}
	cost, err := s.assetCost(assetID, filter)
	if err != nil {
		return false, err
	}
	if cost == 0 {
		// 0 cost means that asset isn't really sponsored anymore.
		return false, nil
	}
	return true, nil
}

func (s *sponsoredAssets) newestAssetCost(assetID crypto.Digest, filter bool) (uint64, error) {
	if cost, ok := s.uncertainSponsoredAssets[assetID]; ok {
		return cost, nil
	}
	key := sponsorshipKey{assetID}
	recordBytes, err := s.hs.newestTopEntryData(key.bytes(), filter)
	if err != nil {
		return 0, err
	}
	var record sponsorshipRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, errors.Errorf("failed to unmarshal sponsorship record: %v\n", err)
	}
	return record.assetCost, nil
}

func (s *sponsoredAssets) assetCost(assetID crypto.Digest, filter bool) (uint64, error) {
	key := sponsorshipKey{assetID}
	recordBytes, err := s.hs.topEntryData(key.bytes(), filter)
	if err != nil {
		return 0, err
	}
	var record sponsorshipRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, errors.Errorf("failed to unmarshal sponsorship record: %v\n", err)
	}
	return record.assetCost, nil
}

func (s *sponsoredAssets) sponsoredAssetToWaves(assetID crypto.Digest, assetAmount uint64) (uint64, error) {
	cost, err := s.newestAssetCost(assetID, true)
	if err != nil {
		return 0, err
	}
	if cost == 0 {
		return 0, errors.New("0 asset cost")
	}
	var wavesAmount big.Int
	wavesAmount.SetUint64(assetAmount)
	var unit big.Int
	unit.SetUint64(FeeUnit)
	wavesAmount.Mul(&wavesAmount, &unit)
	var costBig big.Int
	costBig.SetUint64(cost)
	wavesAmount.Quo(&wavesAmount, &costBig)
	if !wavesAmount.IsInt64() {
		return 0, errors.New("waves amount exceeds MaxInt64")
	}
	return wavesAmount.Uint64(), nil
}

func (s *sponsoredAssets) wavesToSponsoredAsset(assetID crypto.Digest, wavesAmount uint64) (uint64, error) {
	cost, err := s.newestAssetCost(assetID, true)
	if err != nil {
		return 0, err
	}
	if cost == 0 || wavesAmount == 0 {
		return 0, nil
	}
	var assetAmount big.Int
	assetAmount.SetUint64(wavesAmount)
	var costBig big.Int
	costBig.SetUint64(cost)
	assetAmount.Mul(&assetAmount, &costBig)
	var unit big.Int
	unit.SetUint64(FeeUnit)
	assetAmount.Quo(&assetAmount, &unit)
	if !assetAmount.IsInt64() {
		return 0, errors.New("asset amount exceeds MaxInt64")
	}
	return assetAmount.Uint64(), nil
}

func (s *sponsoredAssets) isSponsorshipActivated() (bool, error) {
	featureActivated, err := s.features.newestIsActivated(int16(settings.FeeSponsorship))
	if err != nil {
		return false, err
	}
	if !featureActivated {
		// Not activated at all.
		return false, nil
	}
	height, err := s.features.newestActivationHeight(int16(settings.FeeSponsorship))
	if err != nil {
		return false, err
	}
	// Sponsorship has double activation period.
	curHeight := s.rw.recentHeight()
	sponsorshipTrueActivationHeight := height + s.settings.ActivationWindowSize(height)
	return curHeight >= sponsorshipTrueActivationHeight, nil
}

func (s *sponsoredAssets) prepareHashes() error {
	return s.hasher.stop()
}

func (s *sponsoredAssets) commitUncertain(blockID proto.BlockID) error {
	for assetID, cost := range s.uncertainSponsoredAssets {
		if err := s.sponsorAsset(assetID, cost, blockID); err != nil {
			return err
		}
	}
	return nil
}

func (s *sponsoredAssets) dropUncertain() {
	s.uncertainSponsoredAssets = make(map[crypto.Digest]uint64)
}

func (s *sponsoredAssets) reset() {
	s.hasher.reset()
}
