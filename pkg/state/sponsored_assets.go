package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	FeeUnit = 100000

	sponsorshipRecordSize = 8 + 4
)

type sponsorshipRecord struct {
	// Cost in assets equal to FeeUnit Waves.
	assetCost uint64
	blockNum  uint32
}

func (s *sponsorshipRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, sponsorshipRecordSize)
	binary.BigEndian.PutUint64(res[:8], s.assetCost)
	binary.BigEndian.PutUint32(res[8:12], s.blockNum)
	return res, nil
}

func (s *sponsorshipRecord) unmarshalBinary(data []byte) error {
	if len(data) != sponsorshipRecordSize {
		return errors.New("invalid data size")
	}
	s.assetCost = binary.BigEndian.Uint64(data[:8])
	s.blockNum = binary.BigEndian.Uint32(data[8:12])
	return nil
}

type sponsoredAssets struct {
	features *features
	stateDB  *stateDB
	hs       *historyStorage
	settings *settings.BlockchainSettings
}

func newSponsoredAssets(features *features, stateDB *stateDB, hs *historyStorage, settings *settings.BlockchainSettings) (*sponsoredAssets, error) {
	return &sponsoredAssets{features, stateDB, hs, settings}, nil
}

func (s *sponsoredAssets) sponsorAsset(assetID crypto.Digest, assetCost uint64, blockID crypto.Signature) error {
	key := sponsorshipKey{assetID}
	blockNum, err := s.stateDB.blockIdToNum(blockID)
	if err != nil {
		return err
	}
	record := &sponsorshipRecord{assetCost, blockNum}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if err := s.hs.set(sponsorship, key.bytes(), recordBytes); err != nil {
		return err
	}
	return nil
}

func (s *sponsoredAssets) newestIsSponsored(assetID crypto.Digest, filter bool) bool {
	key := sponsorshipKey{assetID}
	if _, err := s.hs.getFresh(sponsorship, key.bytes(), filter); err != nil {
		return false
	}
	return true
}

func (s *sponsoredAssets) isSponsored(assetID crypto.Digest, filter bool) bool {
	key := sponsorshipKey{assetID}
	if _, err := s.hs.get(sponsorship, key.bytes(), filter); err != nil {
		return false
	}
	return true
}

func (s *sponsoredAssets) newestAssetCost(assetID crypto.Digest, filter bool) (uint64, error) {
	key := sponsorshipKey{assetID}
	recordBytes, err := s.hs.getFresh(sponsorship, key.bytes(), filter)
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
	recordBytes, err := s.hs.get(sponsorship, key.bytes(), filter)
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
	return assetAmount / cost * FeeUnit, nil
}

func (s *sponsoredAssets) wavesToSponsoredAsset(assetID crypto.Digest, wavesAmount uint64) (uint64, error) {
	cost, err := s.newestAssetCost(assetID, true)
	if err != nil {
		return 0, err
	}
	return wavesAmount / FeeUnit * cost, nil
}

func (s *sponsoredAssets) sponsoredFeesSwitchHeight() (uint64, error) {
	height, err := s.features.activationHeight(int16(settings.FeeSponsorship))
	if err != nil {
		return 0, err
	}
	return height + s.settings.ActivationWindowSize(height), nil
}
