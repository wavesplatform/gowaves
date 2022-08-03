package state

import (
	"encoding/binary"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var assetFeeRecordSize = crypto.DigestSize + 8

type assetFeeMap map[crypto.Digest]uint64

func newAssetFeeMap() assetFeeMap {
	return make(assetFeeMap)
}

func (m assetFeeMap) marshalBinary() []byte {
	res := make([]byte, 4+len(m)*assetFeeRecordSize)
	count := uint32(4)
	for asset, fee := range m {
		copy(res[count:count+crypto.DigestSize], asset[:])
		count += crypto.DigestSize
		binary.BigEndian.PutUint64(res[count:count+8], fee)
		count += 8
	}
	binary.BigEndian.PutUint32(res[:4], count)
	return res
}

func (m assetFeeMap) unmarshalBinary(data []byte) (uint32, error) {
	size := uint32(len(data))
	if size < 4 {
		return 0, errInvalidDataSize
	}
	expected := binary.BigEndian.Uint32(data[:4])
	if size < expected {
		return 0, errInvalidDataSize
	}
	for count := uint32(4); count < expected; {
		if size < count+crypto.DigestSize {
			return 0, errInvalidDataSize
		}
		asset, err := crypto.NewDigestFromBytes(data[count : count+crypto.DigestSize])
		if err != nil {
			return 0, err
		}
		count += crypto.DigestSize
		if size < count+8 {
			return 0, errInvalidDataSize
		}
		fee := binary.BigEndian.Uint64(data[count : count+8])
		m[asset] = fee
		count += 8
	}
	return expected, nil
}

type wavesFeeDistribution struct {
	totalWavesFees        uint64
	currentWavesBlockFees uint64
}

type assetsFeeDistribution struct {
	totalFees        assetFeeMap
	currentBlockFees assetFeeMap
}

func newAssetsFeeDistribution() assetsFeeDistribution {
	return assetsFeeDistribution{
		totalFees:        newAssetFeeMap(),
		currentBlockFees: newAssetFeeMap(),
	}
}

type feeDistribution struct {
	wavesFeeDistribution
	assetsFeeDistribution
}

func newFeeDistribution() feeDistribution {
	return feeDistribution{assetsFeeDistribution: newAssetsFeeDistribution()}
}

func (distr *feeDistribution) marshalBinary() []byte {
	totalFeesBytes := distr.totalFees.marshalBinary()
	currentBlockFeesBytes := distr.currentBlockFees.marshalBinary()
	totalSize := 8 + 8 + len(totalFeesBytes) + len(currentBlockFeesBytes)
	distrBytes := make([]byte, totalSize)
	binary.BigEndian.PutUint64(distrBytes[:8], distr.totalWavesFees)
	binary.BigEndian.PutUint64(distrBytes[8:16], distr.currentWavesBlockFees)
	count := 16
	copy(distrBytes[count:count+len(totalFeesBytes)], totalFeesBytes)
	count += len(totalFeesBytes)
	copy(distrBytes[count:count+len(currentBlockFeesBytes)], currentBlockFeesBytes)
	return distrBytes
}

func (distr *feeDistribution) unmarshalBinary(data []byte) error {
	if len(data) < 16 {
		return errInvalidDataSize
	}
	distr.totalWavesFees = binary.BigEndian.Uint64(data[:8])
	distr.currentWavesBlockFees = binary.BigEndian.Uint64(data[8:16])
	size, err := distr.totalFees.unmarshalBinary(data[16:])
	if err != nil {
		return err
	}
	data = data[16+size:]
	if _, err := distr.currentBlockFees.unmarshalBinary(data); err != nil {
		return err
	}
	return nil
}

type blocksInfo struct {
	hs *historyStorage
}

func newBlocksInfo(hs *historyStorage) *blocksInfo {
	return &blocksInfo{hs}
}

func (i *blocksInfo) feeDistribution(blockID proto.BlockID) (*feeDistribution, error) {
	key := blocksInfoKey{blockID}
	distrBytes, err := i.hs.topEntryData(key.bytes())
	if err != nil {
		return &feeDistribution{}, err
	}
	distr := newFeeDistribution()
	if err := distr.unmarshalBinary(distrBytes); err != nil {
		return &feeDistribution{}, err
	}
	return &distr, nil
}

func (i *blocksInfo) saveFeeDistribution(blockID proto.BlockID, distr *feeDistribution) error {
	key := blocksInfoKey{blockID}
	return i.hs.addNewEntry(feeDistr, key.bytes(), distr.marshalBinary(), blockID)
}
