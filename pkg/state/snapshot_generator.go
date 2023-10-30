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

func (sg *snapshotGenerator) generateSnapshotForIssueTx(assetID crypto.Digest, txID crypto.Digest,
	senderPK crypto.PublicKey, assetInfo assetInfo, balanceChanges txDiff,
	scriptEstimation *scriptEstimation, script *proto.Script) (proto.TransactionSnapshot, error) {
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

	if script == nil {
		assetScriptSnapshot := &proto.AssetScriptSnapshot{
			AssetID: assetID,
			Script:  proto.Script{},
		}
		snapshot = append(snapshot, assetScriptSnapshot)
	} else {
		assetScriptSnapshot := &proto.AssetScriptSnapshot{
			AssetID: assetID,
			Script:  *script,
		}
		if sg.IsFullNodeMode && scriptEstimation.isPresent() {
			internalComplexitySnapshot := InternalAssetScriptComplexitySnapshot{
				Estimation: scriptEstimation.estimation, AssetID: assetID,
				ScriptIsEmpty: scriptEstimation.scriptIsEmpty}
			snapshot = append(snapshot, &internalComplexitySnapshot)
		}
		snapshot = append(snapshot, assetScriptSnapshot)
	}
	wavesBalancesSnapshot, assetBalancesSnapshot, leaseBalancesSnapshot, err :=
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
	}
	for i := range wavesBalancesSnapshot {
		snapshot = append(snapshot, &wavesBalancesSnapshot[i])
	}
	for i := range leaseBalancesSnapshot {
		snapshot = append(snapshot, &leaseBalancesSnapshot[i])
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
	leaseStatusSnapshot := &proto.LeaseStateSnapshot{
		LeaseID: leaseID,
		Status: proto.LeaseStateStatus{
			Value: lease.Status,
		},
		Amount:              lease.Amount,
		Sender:              lease.Sender,
		Recipient:           lease.Recipient,
		OriginTransactionID: &originalTxID,
		Height:              lease.Height,
	}
	snapshot = append(snapshot, leaseStatusSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForLeaseCancelTx(txID *crypto.Digest, oldLease leasing,
	leaseID crypto.Digest,
	cancelHeight uint64, balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	var err error
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	leaseStatusSnapshot := &proto.LeaseStateSnapshot{
		LeaseID: leaseID,
		Status: proto.LeaseStateStatus{
			Value:               proto.LeaseCanceled,
			CancelHeight:        cancelHeight,
			CancelTransactionID: txID,
		},
		Amount:              oldLease.Amount,
		Sender:              oldLease.Sender,
		Recipient:           oldLease.Recipient,
		OriginTransactionID: oldLease.OriginTransactionID,
		Height:              oldLease.Height,
	}

	snapshot = append(snapshot, leaseStatusSnapshot)
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

func assetFoundInMap(id proto.AssetID, assetsMap map[proto.AssetID]assetInfo) bool {
	for assetIDinMap := range assetsMap {
		if id == assetIDinMap {
			return true
		}
	}
	return false
}

func generateSnapshotsFromAssetsUncertain(assetsUncertain map[proto.AssetID]assetInfo,
	txID crypto.Digest) []proto.AtomicSnapshot {
	var atomicSnapshots []proto.AtomicSnapshot
	for assetID, info := range assetsUncertain {
		infoCpy := info // prevent implicit memory aliasing in for loop
		fullAssetID := proto.ReconstructDigest(assetID, infoCpy.tail)
		issueStaticInfoSnapshot := &proto.StaticAssetInfoSnapshot{
			AssetID:             fullAssetID,
			IssuerPublicKey:     infoCpy.issuer,
			SourceTransactionID: txID,
			Decimals:            infoCpy.decimals,
			IsNFT:               infoCpy.isNFT(),
		}

		assetDescription := &proto.AssetDescriptionSnapshot{
			AssetID:          fullAssetID,
			AssetName:        infoCpy.name,
			AssetDescription: infoCpy.description,
			ChangeHeight:     infoCpy.lastNameDescChangeHeight,
		}

		assetReissuability := &proto.AssetVolumeSnapshot{
			AssetID:       fullAssetID,
			IsReissuable:  infoCpy.reissuable,
			TotalQuantity: infoCpy.quantity,
		}

		atomicSnapshots = append(atomicSnapshots, issueStaticInfoSnapshot, assetDescription, assetReissuability)
	}
	return atomicSnapshots
}

func generateSnapshotsFromDataEntryUncertain(dataEntriesUncertain map[entryId]uncertainAccountsDataStorageEntry,
	scheme proto.Scheme) ([]proto.AtomicSnapshot, error) {
	var atomicSnapshots []proto.AtomicSnapshot
	dataEntries := make(map[proto.WavesAddress]proto.DataEntries)
	for entryID, entry := range dataEntriesUncertain {
		address, errCnvrt := entryID.addrID.ToWavesAddress(scheme)
		if errCnvrt != nil {
			return nil, errors.Wrap(errCnvrt, "failed to convert address id to waves address")
		}
		if _, ok := dataEntries[address]; ok {
			entries := dataEntries[address]
			entries = append(entries, entry.dataEntry)
			dataEntries[address] = entries
		} else {
			dataEntries[address] = proto.DataEntries{entry.dataEntry}
		}
	}
	for address, entries := range dataEntries {
		dataEntrySnapshot := &proto.DataEntriesSnapshot{Address: address, DataEntries: entries}
		atomicSnapshots = append(atomicSnapshots, dataEntrySnapshot)
	}
	return atomicSnapshots, nil
}

func generateSnapshotsFromAssetsScriptsUncertain(
	assetScriptsUncertain map[proto.AssetID]assetScriptRecordWithAssetIDTail) []proto.AtomicSnapshot {
	var atomicSnapshots []proto.AtomicSnapshot
	for assetID, r := range assetScriptsUncertain {
		digest := proto.ReconstructDigest(assetID, r.assetIDTail)
		assetScrptSnapshot := &proto.AssetScriptSnapshot{
			AssetID: digest,
			Script:  proto.Script{},
		}
		atomicSnapshots = append(atomicSnapshots, assetScrptSnapshot)
	}
	return atomicSnapshots
}

func generateSnapshotsFromLeasingsUncertain(
	leasesUncertain map[crypto.Digest]*leasing) []proto.AtomicSnapshot {
	var atomicSnapshots []proto.AtomicSnapshot
	for id, leasing := range leasesUncertain {
		leaseStatusSnapshot := &proto.LeaseStateSnapshot{
			LeaseID: id,
			Status: proto.LeaseStateStatus{
				Value: leasing.Status,
			},
			Amount:              leasing.Amount,
			Sender:              leasing.Sender,
			Recipient:           leasing.Recipient,
			OriginTransactionID: leasing.OriginTransactionID,
			Height:              leasing.Height,
		}
		if leasing.Status == proto.LeaseCanceled {
			leaseStatusSnapshot.Status.Value = proto.LeaseCanceled
			leaseStatusSnapshot.Status.CancelTransactionID = leasing.CancelTransactionID
			leaseStatusSnapshot.Status.CancelHeight = leasing.CancelHeight
		}
		atomicSnapshots = append(atomicSnapshots, leaseStatusSnapshot)
	}
	return atomicSnapshots
}

func generateSnapshotsFromSponsoredAssetsUncertain(
	sponsoredAssetsUncertain map[proto.AssetID]uncertainSponsoredAsset) []proto.AtomicSnapshot {
	var atomicSnapshots []proto.AtomicSnapshot
	for _, sponsored := range sponsoredAssetsUncertain {
		sponsorshipSnapshot := proto.SponsorshipSnapshot{
			AssetID:         sponsored.assetID,
			MinSponsoredFee: sponsored.assetCost,
		}
		atomicSnapshots = append(atomicSnapshots, sponsorshipSnapshot)
	}
	return atomicSnapshots
}

func (sg *snapshotGenerator) generateSnapshotForInvoke(txID crypto.Digest,
	balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	var snapshot proto.TransactionSnapshot
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := balanceDiffFromTxDiff(balanceChanges, sg.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	// Remove the just issues snapshot from the diff, because it's not in the storage yet,
	// so can't be processed with generateBalancesAtomicSnapshots.
	var specialAssetsSnapshots []proto.AssetBalanceSnapshot
	for key, diffAmount := range addrAssetBalanceDiff {
		if assetFoundInMap(key.asset, sg.stor.assets.uncertainAssetInfo) {
			// remove the element from the array
			delete(addrAssetBalanceDiff, key)
			fullAssetID := proto.ReconstructDigest(key.asset, sg.stor.assets.uncertainAssetInfo[key.asset].tail)
			specialAssetSnapshot := proto.AssetBalanceSnapshot{
				Address: key.address,
				AssetID: fullAssetID,
				Balance: uint64(diffAmount),
			}
			specialAssetsSnapshots = append(specialAssetsSnapshots, specialAssetSnapshot)
		}
	}

	assetsUncertain := sg.stor.assets.uncertainAssetInfo
	dataEntriesUncertain := sg.stor.accountsDataStor.uncertainEntries
	assetScriptsUncertain := sg.stor.scriptsStorage.uncertainAssetScriptsCopy()
	leasesUncertain := sg.stor.leases.uncertainLeases
	sponsoredAssetsUncertain := sg.stor.sponsoredAssets.uncertainSponsoredAssets

	assetsSnapshots := generateSnapshotsFromAssetsUncertain(assetsUncertain, txID)
	snapshot = append(snapshot, assetsSnapshots...)

	dataEntriesSnapshots, err := generateSnapshotsFromDataEntryUncertain(dataEntriesUncertain, sg.scheme)
	if err != nil {
		return nil, err
	}
	snapshot = append(snapshot, dataEntriesSnapshots...)

	assetsScriptsSnapshots := generateSnapshotsFromAssetsScriptsUncertain(assetScriptsUncertain)
	snapshot = append(snapshot, assetsScriptsSnapshots...)

	leasingSnapshots := generateSnapshotsFromLeasingsUncertain(leasesUncertain)
	snapshot = append(snapshot, leasingSnapshots...)

	sponsoredAssetsSnapshots := generateSnapshotsFromSponsoredAssetsUncertain(sponsoredAssetsUncertain)
	snapshot = append(snapshot, sponsoredAssetsSnapshots...)

	wavesBalancesSnapshot, assetBalancesSnapshot, leaseBalancesSnapshot, err :=
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
	}
	for i := range wavesBalancesSnapshot {
		snapshot = append(snapshot, &wavesBalancesSnapshot[i])
	}
	for i := range leaseBalancesSnapshot {
		snapshot = append(snapshot, &leaseBalancesSnapshot[i])
	}
	for i := range assetBalancesSnapshot {
		snapshot = append(snapshot, &assetBalancesSnapshot[i])
	}
	for i := range specialAssetsSnapshots {
		snapshot = append(snapshot, &specialAssetsSnapshots[i])
	}
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForInvokeExpressionTx(txID crypto.Digest,
	balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	return sg.generateSnapshotForInvoke(txID, balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForEthereumInvokeScriptTx(txID crypto.Digest,
	balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	return sg.generateSnapshotForInvoke(txID, balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForUpdateAssetInfoTx(assetID crypto.Digest, assetName string,
	assetDescription string, changeHeight proto.Height, balanceChanges txDiff) (proto.TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}
	assetDescriptionSnapshot := &proto.AssetDescriptionSnapshot{
		AssetID:          assetID,
		AssetName:        assetName,
		AssetDescription: assetDescription,
		ChangeHeight:     changeHeight,
	}
	snapshot = append(snapshot, assetDescriptionSnapshot)
	return snapshot, nil
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
	wavesBalancesSnapshot, assetBalancesSnapshot, leaseBalancesSnapshot, err :=
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
	}
	for i := range wavesBalancesSnapshot {
		transactionSnapshot = append(transactionSnapshot, &wavesBalancesSnapshot[i])
	}
	for i := range leaseBalancesSnapshot {
		transactionSnapshot = append(transactionSnapshot, &leaseBalancesSnapshot[i])
	}
	for i := range assetBalancesSnapshot {
		transactionSnapshot = append(transactionSnapshot, &assetBalancesSnapshot[i])
	}
	return transactionSnapshot, nil
}

func (sg *snapshotGenerator) generateBalancesAtomicSnapshots(
	addrWavesBalanceDiff addressWavesBalanceDiff,
	addrAssetBalanceDiff addressAssetBalanceDiff) (
	[]proto.WavesBalanceSnapshot,
	[]proto.AssetBalanceSnapshot,
	[]proto.LeaseBalanceSnapshot, error) {
	wavesBalanceSnapshot, leaseBalanceSnapshot, err := sg.wavesBalanceSnapshotFromBalanceDiff(addrWavesBalanceDiff)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to construct waves balance snapshot")
	}
	if len(addrAssetBalanceDiff) == 0 {
		return wavesBalanceSnapshot, nil, leaseBalanceSnapshot, nil
	}

	assetBalanceSnapshot, err := sg.assetBalanceSnapshotFromBalanceDiff(addrAssetBalanceDiff)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to construct asset balance snapshot")
	}
	return wavesBalanceSnapshot, assetBalanceSnapshot, leaseBalanceSnapshot, nil
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
		addrWavesBalanceDiff[address] = diffAmount
	}
	return addrWavesBalanceDiff, addrAssetBalanceDiff, nil
}

// from txDiff and fees. no validation needed at this point.
func (sg *snapshotGenerator) wavesBalanceSnapshotFromBalanceDiff(
	diff addressWavesBalanceDiff) ([]proto.WavesBalanceSnapshot, []proto.LeaseBalanceSnapshot, error) {
	var wavesBalances []proto.WavesBalanceSnapshot
	var leaseBalances []proto.LeaseBalanceSnapshot
	// add miner address to the diff

	for wavesAddress, diffAmount := range diff {
		fullBalance, err := sg.stor.balances.newestWavesBalance(wavesAddress.ID())
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to receive sender's waves balance")
		}
		if diffAmount.balance != 0 {
			newBalance := proto.WavesBalanceSnapshot{
				Address: wavesAddress,
				Balance: uint64(int64(fullBalance.balance) + diffAmount.balance),
			}
			wavesBalances = append(wavesBalances, newBalance)
		}
		if diffAmount.leaseIn != 0 || diffAmount.leaseOut != 0 {
			newLeaseBalance := proto.LeaseBalanceSnapshot{
				Address:  wavesAddress,
				LeaseIn:  uint64(fullBalance.leaseIn + diffAmount.leaseIn),
				LeaseOut: uint64(fullBalance.leaseOut + diffAmount.leaseOut),
			}
			leaseBalances = append(leaseBalances, newLeaseBalance)
		}
	}
	return wavesBalances, leaseBalances, nil
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
