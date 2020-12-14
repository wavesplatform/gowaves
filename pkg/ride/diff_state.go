package ride

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type diffDataEntries struct {
	diffInteger []proto.IntegerDataEntry
	diffBool    []proto.BooleanDataEntry
	diffString  []proto.StringDataEntry
	diffBinary  []proto.BinaryDataEntry
}

type diffBalance struct {
	address proto.Address
	assetID   crypto.Digest
	amount    int64
}

type diffWavesBalance struct {
	address  proto.Address
	regular    int64
	generating int64
	available  int64
	effective  int64
}

type diffSponsorship struct {
	assetID crypto.Digest
	MinFee  int64
}

type diffNewAssetInfo struct {
	dAppIssuer  proto.Address
	assetID     crypto.Digest
	name        string
	description string
	quantity    int64
	decimals    int32
	reissuable  bool
	script      []byte
	nonce       int64
}

type diffOldAssetInfo struct {
	dAppIssuer   proto.Address
	assetID      crypto.Digest
	diffQuantity int64
}

type diffState struct {
	state    	  types.SmartState
	dataEntries   diffDataEntries
	balances      []diffBalance
	wavesBalances []diffWavesBalance
	sponsorships  []diffSponsorship
	newAssetsInfo []diffNewAssetInfo
	oldAssetsInfo []diffOldAssetInfo
}

func (diffSt *diffState) findIntFromDataEntryByKey(key string) *proto.IntegerDataEntry {
	for _, intDataEntry := range diffSt.dataEntries.diffInteger {
		if key == intDataEntry.Key {
			return &intDataEntry
		}
	}
	return nil
}

func (diffSt *diffState) findBoolFromDataEntryByKey(key string) *proto.BooleanDataEntry {
	for _, boolDataEntry := range diffSt.dataEntries.diffBool {
		if key == boolDataEntry.Key {
			return &boolDataEntry
		}
	}
	return nil
}

func (diffSt *diffState) findStringFromDataEntryByKey(key string) *proto.StringDataEntry {
	for _, stringDataEntry := range diffSt.dataEntries.diffString {
		if key == stringDataEntry.Key {
			return &stringDataEntry
		}
	}
	return nil
}

func (diffSt *diffState) findBinaryFromDataEntryByKey(key string) *proto.BinaryDataEntry {
	for _, binaryDataEntry := range diffSt.dataEntries.diffBinary {
		if key == binaryDataEntry.Key {
			return &binaryDataEntry
		}
	}
	return nil
}

func (diffSt *diffState) findWavesBalance(recipient proto.Recipient) (*diffWavesBalance, error) {
	address, err := diffSt.state.NewestRecipientToAddress(recipient)
	if err != nil {
		return nil, errors.Errorf("cannot get address from recipient")
	}
	for _, v := range diffSt.wavesBalances {
		if v.address == *address {
			return &v, nil
		}
	}
	return nil, nil
}

func (diffSt *diffState) findBalance(recipient proto.Recipient, asset []byte) (*diffBalance, error) {
	address, err := diffSt.state.NewestRecipientToAddress(recipient)
	if err != nil {
		return nil, errors.Errorf("cannot get address from recipient")
	}
	for _, v := range diffSt.balances {
		if v.address == *address && bytes.Equal(v.assetID.Bytes(), asset) {
			return &v, nil
		}
	}
	return nil, nil
}

func (diffSt *diffState) findSponsorship(assetID crypto.Digest) *int64 {
	for _, v := range diffSt.sponsorships {
		if assetID == v.assetID {
			return &v.MinFee
		}
	}
	return nil
}

func (diffSt *diffState) findNewAsset(assetID crypto.Digest) *diffNewAssetInfo {
	for _, v := range diffSt.newAssetsInfo {
		if assetID == v.assetID {
			return &v
		}
	}
	return nil
}

func (diffSt *diffState) findOldAsset(assetID crypto.Digest) *diffOldAssetInfo {
	for _, v := range diffSt.oldAssetsInfo {
		if assetID == v.assetID {
			return &v
		}
	}
	return nil
}
