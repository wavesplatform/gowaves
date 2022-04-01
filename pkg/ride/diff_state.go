package ride

import (
	"fmt"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type dataEntryKey struct {
	key     string
	address proto.WavesAddress
}

func (d *dataEntryKey) String() string {
	return fmt.Sprintf("%s|%s", d.key, d.address.String())
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
	minFee int64
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

type balanceDiffKey struct {
	address proto.WavesAddress
	asset   proto.OptionalAsset
}

func (b *balanceDiffKey) String() string {
	return fmt.Sprintf("%s|%s", b.address.String(), b.asset.String())
}

type diffState struct {
	state         types.SmartState
	data          map[dataEntryKey]proto.DataEntry
	balances      map[balanceDiffKey]diffBalance
	sponsorships  map[crypto.Digest]diffSponsorship
	newAssetsInfo map[crypto.Digest]diffNewAssetInfo
	oldAssetsInfo map[crypto.Digest]diffOldAssetInfo
	leases        map[crypto.Digest]lease
}

func newDiffState(state types.SmartState) diffState {
	return diffState{
		state:         state,
		data:          map[dataEntryKey]proto.DataEntry{},
		balances:      map[balanceDiffKey]diffBalance{},
		sponsorships:  map[crypto.Digest]diffSponsorship{},
		newAssetsInfo: map[crypto.Digest]diffNewAssetInfo{},
		oldAssetsInfo: map[crypto.Digest]diffOldAssetInfo{},
		leases:        map[crypto.Digest]lease{},
	}
}

func (diffSt *diffState) addBalanceTo(balanceKey balanceDiffKey, amount int64) {
	oldDiffBalance := diffSt.balances[balanceKey]
	oldDiffBalance.regular += amount
	diffSt.balances[balanceKey] = oldDiffBalance
}

func (diffSt *diffState) reissueNewAsset(assetID crypto.Digest, quantity int64, reissuable bool) {
	assetInfo := diffSt.newAssetsInfo[assetID]
	assetInfo.reissuable = reissuable
	assetInfo.quantity += quantity
	diffSt.newAssetsInfo[assetID] = assetInfo
}

func (diffSt *diffState) burnNewAsset(assetID crypto.Digest, quantity int64) {
	assetInfo := diffSt.newAssetsInfo[assetID]
	assetInfo.quantity -= quantity
	diffSt.newAssetsInfo[assetID] = assetInfo
}

func (diffSt *diffState) createNewWavesBalance(account proto.Recipient) (*diffBalance, balanceDiffKey) {
	balance := diffBalance{asset: proto.NewOptionalAssetWaves()}
	key := balanceDiffKey{*account.Address, balance.asset}
	diffSt.balances[key] = balance
	return &balance, key
}

func (diffSt *diffState) cancelLease(searchLease lease, senderSearchAddress, recipientSearchBalanceKey balanceDiffKey) {
	oldDiffBalanceRecipient := diffSt.balances[recipientSearchBalanceKey]
	oldDiffBalanceRecipient.leaseIn -= searchLease.leasedAmount
	diffSt.balances[recipientSearchBalanceKey] = oldDiffBalanceRecipient

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

func (diffSt *diffState) addEffectiveToHistory(searchBalanceKey balanceDiffKey, effective int64) error {
	oldDiffBalance, ok := diffSt.balances[searchBalanceKey]
	if !ok {
		return EvaluationFailure.Errorf("cannot find balance to add effective to history, key %q", searchBalanceKey.String())
	}
	oldDiffBalance.effectiveHistory = append(oldDiffBalance.effectiveHistory, effective)
	diffSt.balances[searchBalanceKey] = oldDiffBalance
	return nil
}

func (diffSt *diffState) addNewLease(recipient proto.Recipient, sender proto.Recipient, leasedAmount int64, leaseID crypto.Digest) {
	lease := lease{Recipient: recipient, Sender: sender, leasedAmount: leasedAmount}
	diffSt.leases[leaseID] = lease
}

func (diffSt *diffState) addLeaseInTo(searchBalanceKey balanceDiffKey, leasedAmount int64) {
	oldDiffBalance := diffSt.balances[searchBalanceKey]
	oldDiffBalance.leaseIn += leasedAmount

	diffSt.balances[searchBalanceKey] = oldDiffBalance
}

func (diffSt *diffState) changeLeaseIn(searchBalance *diffBalance, searchBalanceKey balanceDiffKey, leasedAmount int64, account proto.Recipient) error {
	if searchBalance != nil {
		diffSt.addLeaseInTo(searchBalanceKey, leasedAmount)
		return nil
	}
	address, err := diffSt.state.NewestRecipientToAddress(account)
	if err != nil {
		return err
	}

	balance := diffBalance{
		asset:   proto.NewOptionalAssetWaves(),
		leaseIn: leasedAmount,
	}
	key := balanceDiffKey{*address, balance.asset}

	diffSt.balances[key] = balance
	return nil
}

func (diffSt *diffState) addLeaseOutTo(balanceKey balanceDiffKey, leasedAmount int64) {
	oldDiffBalance := diffSt.balances[balanceKey]
	oldDiffBalance.leaseOut += leasedAmount
	diffSt.balances[balanceKey] = oldDiffBalance
}

func (diffSt *diffState) changeLeaseOut(searchBalance *diffBalance, searchBalanceKey balanceDiffKey, leasedAmount int64, account proto.Recipient) error {
	if searchBalance != nil {
		diffSt.addLeaseOutTo(searchBalanceKey, leasedAmount)
		return nil
	}

	address, err := diffSt.state.NewestRecipientToAddress(account)
	if err != nil {
		return err
	}

	balance := diffBalance{
		asset:    proto.NewOptionalAssetWaves(),
		leaseOut: leasedAmount,
	}
	key := balanceDiffKey{*address, balance.asset}

	diffSt.balances[key] = balance
	return nil
}

func (diffSt *diffState) changeBalance(searchBalance *diffBalance, balanceKey balanceDiffKey, amount int64, asset proto.OptionalAsset, account proto.Recipient) error {
	if searchBalance != nil {
		diffSt.addBalanceTo(balanceKey, amount)
		return nil
	}

	address, err := diffSt.state.NewestRecipientToAddress(account)
	if err != nil {
		return err
	}

	balance := diffBalance{
		asset:   asset,
		regular: amount,
	}
	key := balanceDiffKey{*address, balance.asset}

	diffSt.balances[key] = balance
	return nil
}

func (diffSt *diffState) findLeaseByIDForCancel(leaseID crypto.Digest) (*lease, error) {
	if lease, ok := diffSt.leases[leaseID]; ok {
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
	k := dataEntryKey{key, address}
	if e, ok := diffSt.data[k]; ok {
		if te, ok := e.(*proto.IntegerDataEntry); ok {
			return te
		}
	}
	return nil
}

func (diffSt *diffState) findBoolFromDataEntryByKey(key string, address proto.WavesAddress) *proto.BooleanDataEntry {
	k := dataEntryKey{key, address}
	if e, ok := diffSt.data[k]; ok {
		if te, ok := e.(*proto.BooleanDataEntry); ok {
			return te
		}
	}
	return nil
}

func (diffSt *diffState) findStringFromDataEntryByKey(key string, address proto.WavesAddress) *proto.StringDataEntry {
	k := dataEntryKey{key, address}
	if e, ok := diffSt.data[k]; ok {
		if te, ok := e.(*proto.StringDataEntry); ok {
			return te
		}
	}
	return nil
}

func (diffSt *diffState) findBinaryFromDataEntryByKey(key string, address proto.WavesAddress) *proto.BinaryDataEntry {
	k := dataEntryKey{key, address}
	if e, ok := diffSt.data[k]; ok {
		if te, ok := e.(*proto.BinaryDataEntry); ok {
			return te
		}
	}
	return nil
}

func (diffSt *diffState) findDeleteFromDataEntryByKey(key string, address proto.WavesAddress) *proto.DeleteDataEntry {
	k := dataEntryKey{key, address}
	if e, ok := diffSt.data[k]; ok {
		if te, ok := e.(*proto.DeleteDataEntry); ok {
			return te
		}
	}
	return nil
}

func (diffSt *diffState) putDataEntry(entry proto.DataEntry, address proto.WavesAddress) {
	k := dataEntryKey{entry.GetKey(), address}
	diffSt.data[k] = entry
}

func (diffSt *diffState) findBalance(recipient proto.Recipient, asset proto.OptionalAsset) (*diffBalance, balanceDiffKey, error) {
	address, err := diffSt.state.NewestRecipientToAddress(recipient)
	if err != nil {
		return nil, balanceDiffKey{}, EvaluationFailure.Errorf("cannot get address from recipient")
	}
	key := balanceDiffKey{*address, asset}
	if balance, ok := diffSt.balances[key]; ok {
		return &balance, key, nil
	}
	return nil, balanceDiffKey{}, nil
}

func (diffSt *diffState) findSponsorship(assetID crypto.Digest) *int64 {
	if sponsorship, ok := diffSt.sponsorships[assetID]; ok {
		return &sponsorship.minFee
	}
	return nil
}

func (diffSt *diffState) findNewAsset(assetID crypto.Digest) *diffNewAssetInfo {
	if newAsset, ok := diffSt.newAssetsInfo[assetID]; ok {
		return &newAsset
	}
	return nil
}

func (diffSt *diffState) findOldAsset(assetID crypto.Digest) *diffOldAssetInfo {
	if oldAsset, ok := diffSt.oldAssetsInfo[assetID]; ok {
		return &oldAsset
	}
	return nil
}
