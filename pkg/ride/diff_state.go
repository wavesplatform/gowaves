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
	assetID    crypto.Digest
	regular    int64
	generating int64
	available  int64
	effective  int64
	leaseIn    int64
	leaseOut   int64

}

type diffSponsorship struct {
	MinFee int64
}

type diffNewAssetInfo struct {
	dAppIssuer  proto.Address
	name        string
	description string
	quantity    int64
	decimals    int32
	reissuable  bool
	script      []byte
	nonce       int64
}

type diffOldAssetInfo struct {
	diffQuantity int64
}

type diffState struct {
	state         types.SmartState
	dataEntries   diffDataEntries
	balances      map[string]diffBalance      // map[address.String() + Digest.String()] or map[address.String()] only
	sponsorships  map[string]diffSponsorship  // map[Digest.String()]
	newAssetsInfo map[string]diffNewAssetInfo // map[assetID.String()]
	oldAssetsInfo map[string]diffOldAssetInfo // map[assetID.String()]
}

func (diffSt *diffState) addBalanceTo(searchAddress string, amount int64) {
	oldDiffBalance := diffSt.balances[searchAddress]
	oldDiffBalance.regular += amount
	oldDiffBalance.available += amount
	oldDiffBalance.generating += amount
	oldDiffBalance.effective += amount
	diffSt.balances[searchAddress] = oldDiffBalance
}

func (diffSt *diffState) reissueNewAsset(assetID crypto.Digest, quantity int64, reissuable bool) {

	asset := diffSt.newAssetsInfo[assetID.String()]
	asset.reissuable = reissuable
	asset.quantity += quantity
	diffSt.newAssetsInfo[assetID.String()] = asset
}

func (diffSt *diffState) burnNewAsset(assetID crypto.Digest, quantity int64) {

	asset := diffSt.newAssetsInfo[assetID.String()]
	asset.quantity -= quantity
	diffSt.newAssetsInfo[assetID.String()] = asset
}

func (diffSt *diffState) addLeaseInTo(searchAddress string, leasedAmount int64) {
	oldDiffBalance := diffSt.balances[searchAddress]
	oldDiffBalance.available += leasedAmount
	// oldDiffBalance.generating += leasedAmount // should we sumarize
	oldDiffBalance.effective += leasedAmount
	oldDiffBalance.leaseIn += leasedAmount

	diffSt.balances[searchAddress] = oldDiffBalance
}

func (diffSt *diffState) changeLeaseIn(searchBalance *diffBalance, searchAddress string, leasedAmount int64, account proto.Recipient) error {
	if searchBalance != nil {
		diffSt.addLeaseInTo(searchAddress, leasedAmount)
		return nil
	}
	address, err := diffSt.state.NewestRecipientToAddress(account)
	if err != nil {
		return err
	}

	var balance diffBalance
	balance.assetID = crypto.Digest{}
	balance.available = leasedAmount
	balance.effective = leasedAmount
	// balance.generating = leasedAmount // should we sumarize
	balance.leaseIn = leasedAmount

	diffSt.balances[address.String()+balance.assetID.String()] = balance
	return nil
}

func (diffSt *diffState) addLeaseOutTo(searchAddress string, leasedAmount int64) {
	oldDiffBalance := diffSt.balances[searchAddress]
	oldDiffBalance.available -= leasedAmount
	oldDiffBalance.generating -= leasedAmount
	oldDiffBalance.effective -= leasedAmount
	oldDiffBalance.leaseOut += leasedAmount
	diffSt.balances[searchAddress] = oldDiffBalance
}

func (diffSt *diffState) changeLeaseOut(searchBalance *diffBalance, searchAddress string, leasedAmount int64, account proto.Recipient) error {
	if searchBalance != nil {
		diffSt.addLeaseOutTo(searchAddress, leasedAmount)
		return nil
	}

	address, err := diffSt.state.NewestRecipientToAddress(account)
	if err != nil {
		return err
	}

	var balance diffBalance
	balance.assetID = crypto.Digest{}
	balance.available = -leasedAmount
	balance.generating = -leasedAmount
	balance.effective = -leasedAmount
	balance.leaseOut = leasedAmount

	diffSt.balances[address.String()+balance.assetID.String()] = balance
	return nil
}

func (diffSt *diffState) changeBalance(searchBalance *diffBalance, searchAddress string, amount int64, assetID crypto.Digest, account proto.Recipient) error {
	if searchBalance != nil {
		diffSt.addBalanceTo(searchAddress, amount)
		return nil
	}

	address, err := diffSt.state.NewestRecipientToAddress(account)
	if err != nil {
		return err
	}

	var balance diffBalance
	balance.assetID = assetID
	balance.regular = amount
	balance.available = amount
	balance.generating = amount
	balance.effective = amount

	diffSt.balances[address.String()+assetID.String()] = balance
	return nil
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

func (diffSt *diffState) findBalance(recipient proto.Recipient, asset []byte) (*diffBalance, string, error) {
	address, err := diffSt.state.NewestRecipientToAddress(recipient)
	if err != nil {
		return nil, "", errors.Errorf("cannot get address from recipient")
	}
	if asset == nil {
		if balance, ok := diffSt.balances[address.String()]; ok {
			return &balance, address.String(), nil
		}
		emptyAsset := crypto.Digest{}
		if balance, ok := diffSt.balances[address.String()+emptyAsset.String()]; ok {
			return &balance, address.String(), nil
		}
		return nil, "", nil
	}

	assetID, err := crypto.NewDigestFromBytes(asset)
	if err != nil {
		return nil, "", err
	}
	if balance, ok := diffSt.balances[address.String()+assetID.String()]; ok {
		return &balance, address.String() + assetID.String(), nil
	}

	return nil, "", nil
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
