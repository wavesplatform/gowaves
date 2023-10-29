package state

import (
	"math/big"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type snapshotGenerator struct {
	stor   *blockchainEntitiesStorage
	scheme proto.Scheme

	/* Is IsFullNodeMode is true, then some additional internal fields will be generated */
	IsFullNodeMode bool
}

type addressWavesBalanceDiff map[proto.WavesAddress]balanceDiff

type assetBalanceDiffKey struct {
	address proto.WavesAddress
	asset   proto.AssetID
}
type addressAssetBalanceDiff map[assetBalanceDiffKey]int64

func (sg *snapshotGenerator) generateSnapshotForGenesisTx(balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForPaymentTx(balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForTransferTx(balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

type scriptInformation struct {
	script     proto.Script
	complexity int
}

func (sg *snapshotGenerator) generateSnapshotForIssueTx(assetID crypto.Digest, txID crypto.Digest,
	senderPK crypto.PublicKey, assetInfo assetInfo, balanceChanges txDiff,
	scriptInformation *scriptInformation) (proto.TransactionSnapshot, error) {
	var snapshot proto.TransactionSnapshot
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := balanceDiffFromTxDiff(balanceChanges, sg.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	// Remove the just issues snapshot from the diff, because it's not in the storage yet,
	// so can't be processed with generateBalancesAtomicSnapshots.
	var specialAssetSnapshot *proto.AssetBalanceSnapshot
	for key, diffAmount := range addrAssetBalanceDiff {
		if key.asset == proto.AssetIDFromDigest(assetID) {
			// remove the element from the array

			delete(addrAssetBalanceDiff, key)
			specialAssetSnapshot = &proto.AssetBalanceSnapshot{
				Address: key.address,
				AssetID: assetID,
				Balance: uint64(diffAmount),
			}
		}
	}

	issueStaticInfoSnapshot := &proto.StaticAssetInfoSnapshot{
		AssetID:             assetID,
		IssuerPublicKey:     senderPK,
		SourceTransactionID: txID,
		Decimals:            assetInfo.decimals,
		IsNFT:               assetInfo.isNFT(),
	}

	assetDescription := &proto.AssetDescriptionSnapshot{
		AssetID:          assetID,
		AssetName:        assetInfo.name,
		AssetDescription: assetInfo.description,
		ChangeHeight:     assetInfo.lastNameDescChangeHeight,
	}

	assetReissuability := &proto.AssetVolumeSnapshot{
		AssetID:       assetID,
		IsReissuable:  assetInfo.reissuable,
		TotalQuantity: assetInfo.quantity,
	}

	snapshot = append(snapshot, issueStaticInfoSnapshot, assetDescription, assetReissuability)

	if scriptInformation == nil {
		assetScriptSnapshot := &proto.AssetScriptSnapshot{
			AssetID: assetID,
			Script:  proto.Script{},
		}
		snapshot = append(snapshot, assetScriptSnapshot)
	} else {
		assetScriptSnapshot := &proto.AssetScriptSnapshot{
			AssetID: assetID,
			Script:  scriptInformation.script,
		}
		// TODO: special snapshot for complexity should be generated here
		snapshot = append(snapshot, assetScriptSnapshot)
	}
	wavesBalancesSnapshot, assetBalancesSnapshot, err :=
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
	}

	for i := range wavesBalancesSnapshot {
		snapshot = append(snapshot, &wavesBalancesSnapshot[i])
	}
	for i := range assetBalancesSnapshot {
		snapshot = append(snapshot, &assetBalancesSnapshot[i])
	}
	if specialAssetSnapshot != nil {
		snapshot = append(snapshot, specialAssetSnapshot)
	}

	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForReissueTx(assetID crypto.Digest,
	change assetReissueChange, balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	quantityDiff := big.NewInt(change.diff)
	assetInfo, err := sg.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(assetID))
	if err != nil {
		return nil, err
	}
	resQuantity := assetInfo.quantity.Add(&assetInfo.quantity, quantityDiff)

	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	assetReissuability := &proto.AssetVolumeSnapshot{
		AssetID:       assetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  change.reissuable,
	}
	snapshot = append(snapshot, assetReissuability)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForBurnTx(assetID crypto.Digest, change assetBurnChange,
	balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	quantityDiff := big.NewInt(change.diff)
	assetInfo, err := sg.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(assetID))
	if err != nil {
		return nil, err
	}
	resQuantity := assetInfo.quantity.Sub(&assetInfo.quantity, quantityDiff)

	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	assetReissuability := &proto.AssetVolumeSnapshot{
		AssetID:       assetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  assetInfo.reissuable,
	}
	snapshot = append(snapshot, assetReissuability)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForExchangeTx(sellOrder proto.Order, sellFee uint64,
	buyOrder proto.Order, buyFee uint64, volume uint64,
	balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}

	sellOrderID, err := sellOrder.GetID()
	if err != nil {
		return nil, err
	}
	sellOrderAtomicSnapshot, err := sg.generateOrderAtomicSnapshot(sellOrderID, volume, sellFee)
	if err != nil {
		return nil, err
	}
	buyOrderID, err := buyOrder.GetID()
	if err != nil {
		return nil, err
	}
	buyOrderAtomicSnapshot, err := sg.generateOrderAtomicSnapshot(buyOrderID, volume, buyFee)
	if err != nil {
		return nil, err
	}

	snapshot = append(snapshot, sellOrderAtomicSnapshot, buyOrderAtomicSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForLeaseTx(lease leasing, leaseID crypto.Digest,
	originalTxID crypto.Digest, balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	var err error
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	amount := int64(lease.Amount)
	leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err :=
		sg.generateLeaseAtomicSnapshots(leaseID, lease, originalTxID, lease.Sender, lease.Recipient, amount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate snapshots for a lease transaction")
	}

	snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForLeaseCancelTx(txID *crypto.Digest, oldLease leasing,
	leaseID crypto.Digest, originalTxID crypto.Digest,
	cancelHeight uint64, balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	var err error
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	negativeAmount := -int64(oldLease.Amount)
	leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err :=
		sg.generateLeaseAtomicSnapshots(leaseID, oldLease, originalTxID, oldLease.Sender, oldLease.Recipient, negativeAmount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate snapshots for a lease transaction")
	}
	leaseStatusSnapshot.Status = proto.LeaseStateStatus{
		Value:               proto.LeaseCanceled,
		CancelHeight:        cancelHeight,
		CancelTransactionID: txID,
	}

	snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForCreateAliasTx(senderAddress proto.WavesAddress, alias proto.Alias,
	balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}
	aliasSnapshot := &proto.AliasSnapshot{
		Address: senderAddress,
		Alias:   alias,
	}
	snapshot = append(snapshot, aliasSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForMassTransferTx(
	balanceChanges txDiff,
) (proto.TransactionSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForDataTx(senderAddress proto.WavesAddress, entries []proto.DataEntry,
	balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}
	dataEntriesSnapshot := &proto.DataEntriesSnapshot{
		Address:     senderAddress,
		DataEntries: entries,
	}
	snapshot = append(snapshot, dataEntriesSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForSponsorshipTx(assetID crypto.Digest,
	minAssetFee uint64, balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}
	sponsorshipSnapshot := &proto.SponsorshipSnapshot{
		AssetID:         assetID,
		MinSponsoredFee: minAssetFee,
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForSetScriptTx(senderPK crypto.PublicKey, script proto.Script,
	scriptEstimation scriptEstimation, balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}

	// If the script is empty, it will still be stored in the storage.
	accountScriptSnapshot := &proto.AccountScriptSnapshot{
		SenderPublicKey:    senderPK,
		Script:             script,
		VerifierComplexity: uint64(scriptEstimation.estimation.Verifier),
	}

	snapshot = append(snapshot, accountScriptSnapshot)

	if sg.IsFullNodeMode {
		scriptAddr, cnvrtErr := proto.NewAddressFromPublicKey(sg.scheme, senderPK)
		if cnvrtErr != nil {
			return nil, errors.Wrap(cnvrtErr, "failed to get sender for InvokeScriptWithProofs")
		}
		internalComplexitySnapshot := InternalDAppComplexitySnapshot{
			Estimation: scriptEstimation.estimation, ScriptAddress: scriptAddr, ScriptIsEmpty: scriptEstimation.scriptIsEmpty}
		snapshot = append(snapshot, &internalComplexitySnapshot)
	}

	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForSetAssetScriptTx(assetID crypto.Digest, script proto.Script,
	balanceChanges txDiff, scriptEstimation scriptEstimation) (proto.TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}

	assetScrptSnapshot := &proto.AssetScriptSnapshot{
		AssetID: assetID,
		Script:  script,
	}
	snapshot = append(snapshot, assetScrptSnapshot)
	if sg.IsFullNodeMode {
		internalComplexitySnapshot := InternalAssetScriptComplexitySnapshot{
			Estimation: scriptEstimation.estimation, AssetID: assetID,
			ScriptIsEmpty: scriptEstimation.scriptIsEmpty}
		snapshot = append(snapshot, &internalComplexitySnapshot)
	}
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForUpdateAssetInfoTx(assetID crypto.Digest, assetName string,
	assetDescription string, changeHeight proto.Height, balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}
	sponsorshipSnapshot := &proto.AssetDescriptionSnapshot{
		AssetID:          assetID,
		AssetName:        assetName,
		AssetDescription: assetDescription,
		ChangeHeight:     changeHeight,
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateLeaseAtomicSnapshots(leaseID crypto.Digest,
	l leasing, originalTxID crypto.Digest,
	senderAddress proto.WavesAddress,
	receiverAddress proto.WavesAddress,
	amount int64) (*proto.LeaseStateSnapshot, *proto.LeaseBalanceSnapshot, *proto.LeaseBalanceSnapshot, error) {
	leaseStatusSnapshot := &proto.LeaseStateSnapshot{
		LeaseID: leaseID,
		Status: proto.LeaseStateStatus{
			Value: l.Status,
		},
		Amount:              l.Amount,
		Sender:              l.Sender,
		Recipient:           l.Recipient,
		OriginTransactionID: &originalTxID,
		Height:              l.Height,
	}

	senderBalanceProfile, err := sg.stor.balances.newestWavesBalance(senderAddress.ID())
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to receive sender's waves balance")
	}
	senderLeaseBalanceSnapshot := &proto.LeaseBalanceSnapshot{
		Address:  senderAddress,
		LeaseIn:  uint64(senderBalanceProfile.leaseIn),
		LeaseOut: uint64(senderBalanceProfile.leaseOut + amount),
	}

	receiverBalanceProfile, err := sg.stor.balances.newestWavesBalance(receiverAddress.ID())
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to receive recipient's waves balance")
	}
	recipientLeaseBalanceSnapshot := &proto.LeaseBalanceSnapshot{
		Address:  receiverAddress,
		LeaseIn:  uint64(receiverBalanceProfile.leaseIn + amount),
		LeaseOut: uint64(receiverBalanceProfile.leaseOut),
	}

	return leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, nil
}

func (sg *snapshotGenerator) generateOrderAtomicSnapshot(orderID []byte,
	volume uint64, fee uint64) (*proto.FilledVolumeFeeSnapshot, error) {
	newestFilledAmount, newestFilledFee, err := sg.stor.ordersVolumes.newestFilled(orderID)
	if err != nil {
		return nil, err
	}
	orderIDDigset, err := crypto.NewDigestFromBytes(orderID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct digest from order id bytes")
	}
	// TODO must be added to newest filled amounts and fee
	orderSnapshot := &proto.FilledVolumeFeeSnapshot{
		OrderID:      orderIDDigset,
		FilledFee:    newestFilledFee + fee,
		FilledVolume: newestFilledAmount + volume,
	}
	return orderSnapshot, nil
}

func (sg *snapshotGenerator) generateBalancesSnapshot(balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	var transactionSnapshot proto.TransactionSnapshot
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := balanceDiffFromTxDiff(balanceChanges, sg.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	wavesBalancesSnapshot, assetBalancesSnapshot, err :=
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
	}
	for i := range wavesBalancesSnapshot {
		transactionSnapshot = append(transactionSnapshot, &wavesBalancesSnapshot[i])
	}
	for i := range assetBalancesSnapshot {
		transactionSnapshot = append(transactionSnapshot, &assetBalancesSnapshot[i])
	}
	return transactionSnapshot, nil
}

func (sg *snapshotGenerator) generateBalancesAtomicSnapshots(addrWavesBalanceDiff addressWavesBalanceDiff,
	addrAssetBalanceDiff addressAssetBalanceDiff) ([]proto.WavesBalanceSnapshot, []proto.AssetBalanceSnapshot, error) {
	wavesBalanceSnapshot, err := sg.wavesBalanceSnapshotFromBalanceDiff(addrWavesBalanceDiff)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to construct waves balance snapshot")
	}
	if len(addrAssetBalanceDiff) == 0 {
		return wavesBalanceSnapshot, nil, nil
	}

	assetBalanceSnapshot, err := sg.assetBalanceSnapshotFromBalanceDiff(addrAssetBalanceDiff)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to construct asset balance snapshot")
	}
	return wavesBalanceSnapshot, assetBalanceSnapshot, nil
}

func balanceDiffFromTxDiff(diff txDiff, scheme proto.Scheme) (addressWavesBalanceDiff, addressAssetBalanceDiff, error) {
	addrWavesBalanceDiff := make(addressWavesBalanceDiff)
	addrAssetBalanceDiff := make(addressAssetBalanceDiff)
	for balanceKeyString, diffAmount := range diff {
		// construct address from key
		wavesBalanceKey := &wavesBalanceKey{}
		err := wavesBalanceKey.unmarshal([]byte(balanceKeyString))
		var address proto.WavesAddress
		if err != nil {
			// if the waves balance unmarshal failed, try to marshal into asset balance, and if it fails, then return the error
			assetBalanceKey := &assetBalanceKey{}
			err = assetBalanceKey.unmarshal([]byte(balanceKeyString))
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to convert balance key to asset balance key")
			}
			asset := assetBalanceKey.asset
			address, err = assetBalanceKey.address.ToWavesAddress(scheme)
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to convert address id to waves address")
			}
			assetBalKey := assetBalanceDiffKey{address: address, asset: asset}
			addrAssetBalanceDiff[assetBalKey] = diffAmount.balance
			continue
		}
		address, err = wavesBalanceKey.address.ToWavesAddress(scheme)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to convert address id to waves address")
		}
		// if the waves balance diff is 0, it means it did not change.
		// The reason for the 0 diff to exist is because of how LeaseIn and LeaseOut are handled in transaction differ.
		if diffAmount.balance == 0 {
			continue
		}
		addrWavesBalanceDiff[address] = diffAmount
	}
	return addrWavesBalanceDiff, addrAssetBalanceDiff, nil
}

// from txDiff and fees. no validation needed at this point.
func (sg *snapshotGenerator) wavesBalanceSnapshotFromBalanceDiff(
	diff addressWavesBalanceDiff) ([]proto.WavesBalanceSnapshot, error) {
	var wavesBalances []proto.WavesBalanceSnapshot
	// add miner address to the diff

	for wavesAddress, diffAmount := range diff {
		fullBalance, err := sg.stor.balances.newestWavesBalance(wavesAddress.ID())
		if err != nil {
			return nil, errors.Wrap(err, "failed to receive sender's waves balance")
		}
		newBalance := proto.WavesBalanceSnapshot{
			Address: wavesAddress,
			Balance: uint64(int64(fullBalance.balance) + diffAmount.balance),
		}
		wavesBalances = append(wavesBalances, newBalance)
	}
	return wavesBalances, nil
}

func (sg *snapshotGenerator) assetBalanceSnapshotFromBalanceDiff(
	diff addressAssetBalanceDiff) ([]proto.AssetBalanceSnapshot, error) {
	var assetBalances []proto.AssetBalanceSnapshot
	// add miner address to the diff

	for key, diffAmount := range diff {
		balance, err := sg.stor.balances.newestAssetBalance(key.address.ID(), key.asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to receive sender's waves balance")
		}
		assetInfo, err := sg.stor.assets.newestAssetInfo(key.asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get newest asset info")
		}

		newBalance := proto.AssetBalanceSnapshot{
			Address: key.address,
			AssetID: key.asset.Digest(assetInfo.tail),
			Balance: uint64(int64(balance) + diffAmount),
		}
		assetBalances = append(assetBalances, newBalance)
	}
	return assetBalances, nil
}
