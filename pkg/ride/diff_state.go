package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type diffDataEntries struct {
	diffInteger map[string]proto.IntegerDataEntry // map[key + address.String()]
	diffBool    map[string]proto.BooleanDataEntry
	diffString  map[string]proto.StringDataEntry
	diffBinary  map[string]proto.BinaryDataEntry
	diffDDelete map[string]proto.DeleteDataEntry
}

type diffBalance struct {
	assetID crypto.Digest
	amount  int64
}

type diffWavesBalance struct {
	regular    int64
	generating int64
	available  int64
	effective  int64
}

type diffSponsorship struct {
	MinFee int64
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
	assetID      crypto.Digest
	diffQuantity int64
}

type diffState struct {
	state         types.SmartState
	dataEntries   diffDataEntries
	balances      map[string]diffBalance      // map[address.String() + Digest.String()]
	wavesBalances map[string]diffWavesBalance // map[address.String()]
	sponsorships  map[string]diffSponsorship  // map[Digest.String()]
	newAssetsInfo map[string]diffNewAssetInfo // map[address.String()]
	oldAssetsInfo map[string]diffOldAssetInfo // map[address.String()]
}

func (diffSt *diffState) findIntFromDataEntryByKey(key string, address string) *proto.IntegerDataEntry {
	if integerEntry, ok := diffSt.dataEntries.diffInteger[key+address]; ok {
		return &integerEntry
	}
	return nil
}

func (diffSt *diffState) findBoolFromDataEntryByKey(key string, address string) *proto.BooleanDataEntry {
	if boolEntry, ok := diffSt.dataEntries.diffBool[key+address]; ok {
		return &boolEntry
	}
	return nil
}

func (diffSt *diffState) findStringFromDataEntryByKey(key string, address string) *proto.StringDataEntry {
	if stringEntry, ok := diffSt.dataEntries.diffString[key+address]; ok {
		return &stringEntry
	}
	return nil
}

func (diffSt *diffState) findBinaryFromDataEntryByKey(key string, address string) *proto.BinaryDataEntry {
	if binaryEntry, ok := diffSt.dataEntries.diffBinary[key+address]; ok {
		return &binaryEntry
	}
	return nil
}

func (diffSt *diffState) findDeleteFromDataEntryByKey(key string, address string) *proto.DeleteDataEntry {
	if deleteEntry, ok := diffSt.dataEntries.diffDDelete[key+address]; ok {
		return &deleteEntry
	}
	return nil
}

func (diffSt *diffState) findWavesBalance(recipient proto.Recipient) (*diffWavesBalance, error) {
	address, err := diffSt.state.NewestRecipientToAddress(recipient)
	if err != nil {
		return nil, errors.Errorf("cannot get address from recipient")
	}
	if wavesBalance, ok := diffSt.wavesBalances[address.String()]; ok {
		return &wavesBalance, nil
	}
	return nil, nil
}

func (diffSt *diffState) findBalance(recipient proto.Recipient, asset []byte) (*diffBalance, error) {
	address, err := diffSt.state.NewestRecipientToAddress(recipient)
	if err != nil {
		return nil, errors.Errorf("cannot get address from recipient")
	}
	assetID, err := crypto.NewDigestFromBytes(asset)
	if err != nil {
		return nil, err
	}
	if balance, ok := diffSt.balances[address.String()+assetID.String()]; ok {
		return &balance, nil
	}
	return nil, nil
}

func (diffSt *diffState) findSponsorship(assetID crypto.Digest) *int64 {
	if sponsorship, ok := diffSt.sponsorships[assetID.String()]; ok {
		return &sponsorship.MinFee
	}
	return nil
}

func (diffSt *diffState) findNewAsset(assetID crypto.Digest) *diffNewAssetInfo {
	if newAsset, ok := diffSt.newAssetsInfo[assetID.String()]; ok {
		return &newAsset
	}
	return nil
}

func (diffSt *diffState) findOldAsset(assetID crypto.Digest) *diffOldAssetInfo {
	if oldAsset, ok := diffSt.oldAssetsInfo[assetID.String()]; ok {
		return &oldAsset
	}
	return nil
}
