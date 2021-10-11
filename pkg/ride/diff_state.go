package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type dataEntryKey struct {
	key     string
	address proto.WavesAddress
}

type (
	integerDataEntryKey dataEntryKey
	booleanDataEntryKey dataEntryKey
	stringDataEntryKey  dataEntryKey
	binaryDataEntryKey  dataEntryKey
	deleteDataEntryKey  dataEntryKey
)

type diffDataEntries struct {
	diffInteger map[integerDataEntryKey]proto.IntegerDataEntry // map[key + address.String()]
	diffBool    map[booleanDataEntryKey]proto.BooleanDataEntry
	diffString  map[stringDataEntryKey]proto.StringDataEntry
	diffBinary  map[binaryDataEntryKey]proto.BinaryDataEntry
	diffDelete  map[deleteDataEntryKey]proto.DeleteDataEntry
}

type lease struct {
	Recipient    proto.Recipient
	leasedAmount int64
	Sender       proto.Recipient
}

type diffBalance struct {
	asset            proto.OptionalAsset
	regular          int64
	leaseIn          int64
	leaseOut         int64
	effectiveHistory []int64
}

type diffSponsorship struct {
	MinFee int64
}

type diffNewAssetInfo struct {
	dAppIssuer  proto.WavesAddress
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

// TODO(nickeskov): create keys for diffState

type balanceDiffKey struct {
	address proto.WavesAddress
	assetID proto.AssetID
}

type diffState struct {
	state         types.SmartState
	dataEntries   diffDataEntries
	balances      map[string]diffBalance      // map[address.String() + Digest.String()] or map[address.String()]
	sponsorships  map[string]diffSponsorship  // map[Digest.String()]
	newAssetsInfo map[string]diffNewAssetInfo // map[asset.String()]
	oldAssetsInfo map[string]diffOldAssetInfo // map[asset.String()]
	leases        map[string]lease            // map[lease.String()]
}

func (diffSt *diffState) addBalanceTo(searchAddress string, amount int64) {
	// searchAddress == address.String() + Digest.String(); // see findBalance func
	oldDiffBalance := diffSt.balances[searchAddress]
	oldDiffBalance.regular += amount
	diffSt.balances[searchAddress] = oldDiffBalance
}

func (diffSt *diffState) reissueNewAsset(assetID crypto.Digest, quantity int64, reissuable bool) {
	asset := proto.NewOptionalAssetFromDigest(assetID)
	assetInfo := diffSt.newAssetsInfo[asset.String()]
	assetInfo.reissuable = reissuable
	assetInfo.quantity += quantity
	diffSt.newAssetsInfo[asset.String()] = assetInfo
}

func (diffSt *diffState) burnNewAsset(assetID crypto.Digest, quantity int64) {
	asset := proto.NewOptionalAssetFromDigest(assetID)

	assetInfo := diffSt.newAssetsInfo[asset.String()]
	assetInfo.quantity -= quantity
	diffSt.newAssetsInfo[asset.String()] = assetInfo
}

func (diffSt *diffState) createNewWavesBalance(account proto.Recipient) (*diffBalance, string) {
	wavesAsset := proto.NewOptionalAssetWaves()
	balance := diffBalance{asset: wavesAsset}
	diffSt.balances[account.Address.String()+wavesAsset.String()] = balance
	return &balance, account.Address.String() + wavesAsset.String()
}

func (diffSt *diffState) cancelLease(searchLease lease, senderSearchAddress, recipientSearchAddress string) {
	oldDiffBalanceRecipient := diffSt.balances[recipientSearchAddress]
	oldDiffBalanceRecipient.leaseIn -= searchLease.leasedAmount
	diffSt.balances[recipientSearchAddress] = oldDiffBalanceRecipient

	oldDiffBalanceSender := diffSt.balances[senderSearchAddress]
	oldDiffBalanceSender.leaseOut -= searchLease.leasedAmount
	diffSt.balances[senderSearchAddress] = oldDiffBalanceSender
}

func (diffSt *diffState) findMinGenerating(effectiveHistory []int64, generatingFromState int64) int64 {
	min := generatingFromState
	for _, value := range effectiveHistory {
		if value < min {
			min = value
		}
	}
	return min
}

func (diffSt *diffState) addEffectiveToHistory(searchAddress string, effective int64) error {
	oldDiffBalance, ok := diffSt.balances[searchAddress]
	if !ok {
		return errors.Errorf("cannot find balance to add effective to history")
	}
	oldDiffBalance.effectiveHistory = append(oldDiffBalance.effectiveHistory, effective)
	diffSt.balances[searchAddress] = oldDiffBalance
	return nil
}

func (diffSt *diffState) addNewLease(recipient proto.Recipient, sender proto.Recipient, leasedAmount int64, leaseID crypto.Digest) {
	lease := lease{Recipient: recipient, Sender: sender, leasedAmount: leasedAmount}
	diffSt.leases[leaseID.String()] = lease
}

func (diffSt *diffState) addLeaseInTo(searchAddress string, leasedAmount int64) {
	oldDiffBalance := diffSt.balances[searchAddress]
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
	balance.asset = proto.NewOptionalAssetWaves()
	balance.leaseIn = leasedAmount

	diffSt.balances[address.String()+balance.asset.String()] = balance
	return nil
}

func (diffSt *diffState) addLeaseOutTo(searchAddress string, leasedAmount int64) {
	oldDiffBalance := diffSt.balances[searchAddress]
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
	balance.asset = proto.NewOptionalAssetWaves()
	balance.leaseOut = leasedAmount

	diffSt.balances[address.String()+balance.asset.String()] = balance
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
	asset := *proto.NewOptionalAssetFromDigest(assetID)
	balance.asset = asset
	balance.regular = amount

	diffSt.balances[address.String()+asset.String()] = balance
	return nil
}

func (diffSt *diffState) findLeaseByIDForCancel(leaseID crypto.Digest) (*lease, error) {
	if lease, ok := diffSt.leases[leaseID.String()]; ok {
		return &lease, nil
	}
	leaseFromStore, err := diffSt.state.NewestLeasingInfo(leaseID)
	if err != nil {
		return nil, err
	}
	if leaseFromStore != nil {
		if !leaseFromStore.IsActive {
			return nil, nil
		}
		lease := lease{
			Recipient:    proto.NewRecipientFromAddress(leaseFromStore.Recipient),
			Sender:       proto.NewRecipientFromAddress(leaseFromStore.Sender),
			leasedAmount: int64(leaseFromStore.LeaseAmount),
		}
		return &lease, nil
	}
	return nil, nil
}

func (diffSt *diffState) findIntFromDataEntryByKey(key string, address proto.WavesAddress) *proto.IntegerDataEntry {
	intKey := integerDataEntryKey{key, address}
	if integerEntry, ok := diffSt.dataEntries.diffInteger[intKey]; ok {
		return &integerEntry
	}
	return nil
}

func (diffSt *diffState) findBoolFromDataEntryByKey(key string, address proto.WavesAddress) *proto.BooleanDataEntry {
	boolKey := booleanDataEntryKey{key, address}
	if boolEntry, ok := diffSt.dataEntries.diffBool[boolKey]; ok {
		return &boolEntry
	}
	return nil
}

func (diffSt *diffState) findStringFromDataEntryByKey(key string, address proto.WavesAddress) *proto.StringDataEntry {
	stringKey := stringDataEntryKey{key, address}
	if stringEntry, ok := diffSt.dataEntries.diffString[stringKey]; ok {
		return &stringEntry
	}
	return nil
}

func (diffSt *diffState) findBinaryFromDataEntryByKey(key string, address proto.WavesAddress) *proto.BinaryDataEntry {
	binaryKey := binaryDataEntryKey{key, address}
	if binaryEntry, ok := diffSt.dataEntries.diffBinary[binaryKey]; ok {
		return &binaryEntry
	}
	return nil
}

func (diffSt *diffState) findDeleteFromDataEntryByKey(key string, address proto.WavesAddress) *proto.DeleteDataEntry {
	deleteKey := deleteDataEntryKey{key, address}
	if deleteEntry, ok := diffSt.dataEntries.diffDelete[deleteKey]; ok {
		return &deleteEntry
	}
	return nil
}

func (diffSt *diffState) putDataEntry(entry proto.DataEntry, address proto.WavesAddress) error {
	d := diffSt.dataEntries
	switch entry := entry.(type) {
	case *proto.IntegerDataEntry:
		d.diffInteger[integerDataEntryKey{entry.Key, address}] = *entry
	case *proto.StringDataEntry:
		d.diffString[stringDataEntryKey{entry.Key, address}] = *entry
	case *proto.BooleanDataEntry:
		d.diffBool[booleanDataEntryKey{entry.Key, address}] = *entry
	case *proto.BinaryDataEntry:
		d.diffBinary[binaryDataEntryKey{entry.Key, address}] = *entry
	case *proto.DeleteDataEntry:
		d.diffDelete[deleteDataEntryKey{entry.Key, address}] = *entry
	default:
		return errors.Errorf("unknown DataEntry type (%T)=%v", entry, entry)
	}
	return nil
}

func (diffSt *diffState) findBalance(recipient proto.Recipient, asset proto.OptionalAsset) (*diffBalance, string, error) {
	address, err := diffSt.state.NewestRecipientToAddress(recipient)
	if err != nil {
		return nil, "", errors.Errorf("cannot get address from recipient")
	}

	if balance, ok := diffSt.balances[address.String()+asset.String()]; ok {
		return &balance, address.String() + asset.String(), nil
	}

	return nil, "", nil
}

func (diffSt *diffState) findSponsorship(assetID crypto.Digest) *int64 {
	asset := proto.NewOptionalAssetFromDigest(assetID)
	if sponsorship, ok := diffSt.sponsorships[asset.String()]; ok {
		return &sponsorship.MinFee
	}
	return nil
}

func (diffSt *diffState) findNewAsset(assetID crypto.Digest) *diffNewAssetInfo {
	asset := proto.NewOptionalAssetFromDigest(assetID)
	if newAsset, ok := diffSt.newAssetsInfo[asset.String()]; ok {
		return &newAsset
	}
	return nil
}

func (diffSt *diffState) findOldAsset(assetID crypto.Digest) *diffOldAssetInfo {
	asset := proto.NewOptionalAssetFromDigest(assetID)
	if oldAsset, ok := diffSt.oldAssetsInfo[asset.String()]; ok {
		return &oldAsset
	}
	return nil
}
