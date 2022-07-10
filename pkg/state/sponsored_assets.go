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
	id   crypto.Digest
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

type uncertainSponsoredAsset struct {
	assetID   crypto.Digest
	assetCost uint64
}

type sponsoredAssets struct {
	rw       *blockReadWriter
	features featuresState
	hs       *historyStorage
	settings *settings.BlockchainSettings

	uncertainSponsoredAssets map[proto.AssetID]uncertainSponsoredAsset

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
		uncertainSponsoredAssets: make(map[proto.AssetID]uncertainSponsoredAsset),
		hasher:                   newStateHasher(),
		calculateHashes:          calcHashes,
	}
}

func (s *sponsoredAssets) sponsorAsset(assetID crypto.Digest, assetCost uint64, blockID proto.BlockID) error {
	key := sponsorshipKey{assetID: proto.AssetIDFromDigest(assetID)}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	record := &sponsorshipRecord{assetCost}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if s.calculateHashes {
		sr := &sponsorshipRecordForHashes{
			id:   assetID,
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
	s.uncertainSponsoredAssets[proto.AssetIDFromDigest(assetID)] = uncertainSponsoredAsset{
		assetID:   assetID,
		assetCost: assetCost,
	}
}

func (s *sponsoredAssets) newestIsSponsored(assetID proto.AssetID) (bool, error) {
	cost, err := s.newestAssetCost(assetID)
	if err != nil {
		return false, nil
	}
	if cost == 0 {
		// 0 cost means that asset isn't really sponsored anymore.
		return false, nil
	}
	return true, nil
}

func (s *sponsoredAssets) isSponsored(assetID proto.AssetID) (bool, error) {
	key := sponsorshipKey{assetID: assetID}
	if _, err := s.hs.topEntryData(key.bytes()); err != nil {
		// No sponsorship info for this asset at all.
		return false, nil
	}
	cost, err := s.assetCost(assetID)
	if err != nil {
		return false, err
	}
	if cost == 0 {
		// 0 cost means that asset isn't really sponsored anymore.
		return false, nil
	}
	return true, nil
}

func (s *sponsoredAssets) newestAssetCost(assetID proto.AssetID) (uint64, error) {
	if sponsored, ok := s.uncertainSponsoredAssets[assetID]; ok {
		return sponsored.assetCost, nil
	}
	key := sponsorshipKey{assetID: assetID}
	recordBytes, err := s.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return 0, err
	}
	var record sponsorshipRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, errors.Errorf("failed to unmarshal sponsorship record: %v\n", err)
	}
	return record.assetCost, nil
}

func (s *sponsoredAssets) assetCost(assetID proto.AssetID) (uint64, error) {
	key := sponsorshipKey{assetID: assetID}
	recordBytes, err := s.hs.topEntryData(key.bytes())
	if err != nil {
		return 0, err
	}
	var record sponsorshipRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, errors.Errorf("failed to unmarshal sponsorship record: %v\n", err)
	}
	return record.assetCost, nil
}

func (s *sponsoredAssets) sponsoredAssetToWaves(assetID proto.AssetID, assetAmount uint64) (uint64, error) {
	cost, err := s.newestAssetCost(assetID)
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

func (s *sponsoredAssets) wavesToSponsoredAsset(assetID proto.AssetID, wavesAmount uint64) (uint64, error) {
	cost, err := s.newestAssetCost(assetID)
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
	sponsorshipActivated := false
	if s.settings.SponsorshipSingleActivationPeriod {
		sponsorshipActivated = featureActivated
	} else if featureActivated {
		height, err := s.features.newestActivationHeight(int16(settings.FeeSponsorship))
		if err != nil {
			return false, err
		}
		// Sponsorship has double activation period.
		curHeight := s.rw.recentHeight()
		sponsorshipTrueActivationHeight := height + s.settings.ActivationWindowSize(height)
		sponsorshipActivated = curHeight >= sponsorshipTrueActivationHeight
	}
	return sponsorshipActivated, nil
}

func (s *sponsoredAssets) prepareHashes() error {
	return s.hasher.stop()
}

func (s *sponsoredAssets) commitUncertain(blockID proto.BlockID) error {
	for _, sponsored := range s.uncertainSponsoredAssets {
		if err := s.sponsorAsset(sponsored.assetID, sponsored.assetCost, blockID); err != nil {
			return err
		}
	}
	return nil
}

func (s *sponsoredAssets) dropUncertain() {
	s.uncertainSponsoredAssets = make(map[proto.AssetID]uncertainSponsoredAsset)
}

func (s *sponsoredAssets) reset() {
	s.hasher.reset()
}
