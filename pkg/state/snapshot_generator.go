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
}

type addressWavesBalanceDiff map[proto.WavesAddress]balanceDiff

func (wavesDiff addressWavesBalanceDiff) append(
	senderAddress proto.WavesAddress,
	recipientAddress proto.WavesAddress,
	amount int64) {
	if _, ok := wavesDiff[senderAddress]; ok {
		prevBalance := wavesDiff[senderAddress]
		prevBalance.balance -= amount
		wavesDiff[senderAddress] = prevBalance
	} else {
		wavesDiff[senderAddress] = balanceDiff{balance: amount}
	}

	if _, ok := wavesDiff[recipientAddress]; ok {
		prevRecipientBalance := wavesDiff[recipientAddress]
		prevRecipientBalance.balance += amount
		wavesDiff[recipientAddress] = prevRecipientBalance
	} else {
		wavesDiff[recipientAddress] = balanceDiff{balance: amount}
	}
}

type assetBalanceDiffKey struct {
	address proto.WavesAddress
	asset   proto.AssetID
}
type addressAssetBalanceDiff map[assetBalanceDiffKey]int64

func (assetDiff addressAssetBalanceDiff) append(
	senderAddress proto.WavesAddress,
	recipientAddress proto.WavesAddress,
	asset proto.AssetID,
	amount int64) {
	keySender := assetBalanceDiffKey{address: senderAddress, asset: asset}
	keyRecipient := assetBalanceDiffKey{address: recipientAddress, asset: asset}

	if _, ok := assetDiff[keySender]; ok {
		prevSenderBalance := assetDiff[keySender]
		prevSenderBalance -= amount
		assetDiff[keySender] = prevSenderBalance
	} else {
		assetDiff[keySender] = amount
	}

	if _, ok := assetDiff[keyRecipient]; ok {
		prevRecipientBalance := assetDiff[keyRecipient]
		prevRecipientBalance += amount
		assetDiff[keyRecipient] = prevRecipientBalance
	} else {
		assetDiff[keyRecipient] = amount
	}
}

func (assetDiff addressAssetBalanceDiff) appendOnlySender(
	senderAddress proto.WavesAddress,
	asset proto.AssetID,
	amount int64) {
	keySender := assetBalanceDiffKey{address: senderAddress, asset: asset}
	if _, ok := assetDiff[keySender]; ok {
		prevSenderBalance := assetDiff[keySender]
		prevSenderBalance += amount
		assetDiff[keySender] = prevSenderBalance
	} else {
		assetDiff[keySender] = amount
	}
}

func (sg *snapshotGenerator) generateSnapshotForGenesisTx(balanceChanges txDiff) (TransactionSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForPaymentTx(balanceChanges txDiff) (TransactionSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForTransferTx(balanceChanges txDiff) (TransactionSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

type scriptInformation struct {
	script     proto.Script
	complexity int
}

func (sg *snapshotGenerator) generateSnapshotForIssueTx(assetID crypto.Digest, txID crypto.Digest,
	senderPK crypto.PublicKey, assetInfo assetInfo, balanceChanges txDiff,
	scriptInformation *scriptInformation) (TransactionSnapshot, error) {
	var snapshot TransactionSnapshot
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := balanceDiffFromTxDiff(balanceChanges, sg.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	// Remove the just issues snapshot from the diff, because it's not in the storage yet,
	// so can't be processed with generateBalancesAtomicSnapshots.
	var specialAssetSnapshot *AssetBalanceSnapshot
	for key, diffAmount := range addrAssetBalanceDiff {
		if key.asset == proto.AssetIDFromDigest(assetID) {
			// remove the element from the array

			delete(addrAssetBalanceDiff, key)
			specialAssetSnapshot = &AssetBalanceSnapshot{
				Address: key.address,
				AssetID: assetID,
				Balance: uint64(diffAmount),
			}
		}
	}

	issueStaticInfoSnapshot := &StaticAssetInfoSnapshot{
		AssetID:             assetID,
		IssuerPublicKey:     senderPK,
		SourceTransactionID: txID,
		Decimals:            assetInfo.decimals,
		IsNFT:               assetInfo.isNFT(),
	}

	assetDescription := &AssetDescriptionSnapshot{
		AssetID:          assetID,
		AssetName:        assetInfo.name,
		AssetDescription: assetInfo.description,
		ChangeHeight:     assetInfo.lastNameDescChangeHeight,
	}

	assetReissuability := &AssetVolumeSnapshot{
		AssetID:       assetID,
		IsReissuable:  assetInfo.reissuable,
		TotalQuantity: assetInfo.quantity,
	}

	snapshot = append(snapshot, issueStaticInfoSnapshot, assetDescription, assetReissuability)

	if scriptInformation != nil {
		assetScriptSnapshot := &AssetScriptSnapshot{
			AssetID:    assetID,
			Script:     scriptInformation.script,
			Complexity: uint64(scriptInformation.complexity),
		}
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
	change assetReissueChange, balanceChanges txDiff) (TransactionSnapshot, error) {
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
	assetReissuability := &AssetVolumeSnapshot{
		AssetID:       assetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  change.reissuable,
	}
	snapshot = append(snapshot, assetReissuability)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForBurnTx(assetID crypto.Digest, change assetBurnChange,
	balanceChanges txDiff) (TransactionSnapshot, error) {
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
	assetReissuability := &AssetVolumeSnapshot{
		AssetID:       assetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  assetInfo.reissuable,
	}
	snapshot = append(snapshot, assetReissuability)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForExchangeTx(sellOrder proto.Order, sellFee uint64,
	buyOrder proto.Order, buyFee uint64, volume uint64,
	balanceChanges txDiff) (TransactionSnapshot, error) {
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
	originalTxID crypto.Digest, balanceChanges txDiff) (TransactionSnapshot, error) {
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
	cancelHeight uint64, balanceChanges txDiff) (TransactionSnapshot, error) {
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
	leaseStatusSnapshot.Status = LeaseStateStatus{
		Value:               LeaseCanceled,
		CancelHeight:        cancelHeight,
		CancelTransactionID: txID,
	}

	snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForCreateAliasTx(senderAddress proto.WavesAddress, alias proto.Alias,
	balanceChanges txDiff) (TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}
	aliasSnapshot := &AliasSnapshot{
		Address: senderAddress,
		Alias:   alias,
	}
	snapshot = append(snapshot, aliasSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForMassTransferTx(balanceChanges txDiff) (TransactionSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForDataTx(senderAddress proto.WavesAddress, entries []proto.DataEntry,
	balanceChanges txDiff) (TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}
	dataEntriesSnapshot := &DataEntriesSnapshot{
		Address:     senderAddress,
		DataEntries: entries,
	}
	snapshot = append(snapshot, dataEntriesSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForSponsorshipTx(assetID crypto.Digest,
	minAssetFee uint64, balanceChanges txDiff) (TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}
	sponsorshipSnapshot := &SponsorshipSnapshot{
		AssetID:         assetID,
		MinSponsoredFee: minAssetFee,
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForSetScriptTx(senderPK crypto.PublicKey, script proto.Script,
	complexity int, _ *performerInfo, balanceChanges txDiff) (TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}

	sponsorshipSnapshot := &AccountScriptSnapshot{
		SenderPublicKey:    senderPK,
		Script:             script,
		VerifierComplexity: uint64(complexity),
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForSetAssetScriptTx(assetID crypto.Digest, script proto.Script,
	complexity int, senderPK crypto.PublicKey, balanceChanges txDiff) (TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}

	sponsorshipSnapshot := &AssetScriptSnapshot{
		AssetID:    assetID,
		Script:     script,
		Complexity: uint64(complexity),
		SenderPK:   senderPK,
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForInvokeScriptTx(txID crypto.Digest, info *performerInfo,
	invocationRes *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	return sg.generateInvokeSnapshot(txID, info, invocationRes, balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForInvokeExpressionTx(txID crypto.Digest, info *performerInfo,
	invocationRes *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	return sg.generateInvokeSnapshot(txID, info, invocationRes, balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForEthereumInvokeScriptTx(txID crypto.Digest, info *performerInfo,
	invocationRes *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	return sg.generateInvokeSnapshot(txID, info, invocationRes, balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForUpdateAssetInfoTx(assetID crypto.Digest, assetName string,
	assetDescription string, changeHeight proto.Height, balanceChanges txDiff) (TransactionSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}
	sponsorshipSnapshot := &AssetDescriptionSnapshot{
		AssetID:          assetID,
		AssetName:        assetName,
		AssetDescription: assetDescription,
		ChangeHeight:     changeHeight,
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
	return snapshot, nil
}

type SenderDataEntries map[proto.WavesAddress]proto.DataEntries

func (senderDataEntries SenderDataEntries) collectEntryFromAction(
	action proto.DataEntryScriptAction, scheme proto.Scheme) error {
	if senderDataEntries == nil {
		return errors.New("senderDataEntries map is not initialized")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(scheme, *action.Sender)
	if err != nil {
		return err
	}
	if _, ok := senderDataEntries[senderAddr]; ok {
		entries := senderDataEntries[senderAddr]
		entries = append(entries, action.Entry)
		senderDataEntries[senderAddr] = entries
	} else {
		senderDataEntries[senderAddr] = proto.DataEntries{action.Entry}
	}
	return nil
}

func (sg *snapshotGenerator) updateBalanceDiffFromPaymentAction(
	action proto.AttachedPaymentScriptAction,
	wavesBalanceDiff addressWavesBalanceDiff,
	assetBalanceDiff addressAssetBalanceDiff,
) error {
	senderAddress, err := proto.NewAddressFromPublicKey(sg.scheme, *action.Sender)
	if err != nil {
		return errors.Wrap(err, "failed to get an address from a public key")
	}
	recipientAddress, err := recipientToAddress(action.Recipient, sg.stor.aliases)
	if err != nil {
		return errors.Wrap(err, "failed to apply attached payment")
	}
	// No balance validation done below
	if action.Asset.Present { // Update asset balance
		assetBalanceDiff.append(senderAddress, recipientAddress, proto.AssetIDFromDigest(action.Asset.ID), action.Amount)
	} else { // Update Waves balance
		wavesBalanceDiff.append(senderAddress, recipientAddress, action.Amount)
	}
	return nil
}

func (sg *snapshotGenerator) updateBalanceDiffFromTransferAction(
	action proto.TransferScriptAction,
	wavesBalanceDiff addressWavesBalanceDiff,
	assetBalanceDiff addressAssetBalanceDiff,
) error {
	senderAddress, err := proto.NewAddressFromPublicKey(sg.scheme, *action.Sender)
	if err != nil {
		return errors.Wrap(err, "failed to get an address from a public key")
	}
	recipientAddress, err := recipientToAddress(action.Recipient, sg.stor.aliases)
	if err != nil {
		return errors.Wrap(err, "failed to apply attached payment")
	}
	// No balance validation done below
	if action.Asset.Present { // Update asset balance
		assetBalanceDiff.append(senderAddress, recipientAddress, proto.AssetIDFromDigest(action.Asset.ID), action.Amount)
	} else { // Update Waves balance
		wavesBalanceDiff.append(senderAddress, recipientAddress, action.Amount)
	}
	return nil
}

func (sg *snapshotGenerator) atomicSnapshotsFromIssueAction(
	action proto.IssueScriptAction,
	blockHeight uint64,
	info *performerInfo,
	txID crypto.Digest,
	assetBalanceDiff addressAssetBalanceDiff) ([]AtomicSnapshot, error) {
	var atomicSnapshots []AtomicSnapshot
	assetInf := assetInfo{
		assetConstInfo: assetConstInfo{
			tail:        proto.DigestTail(action.ID),
			issuer:      *action.Sender,
			decimals:    uint8(action.Decimals),
			issueHeight: blockHeight,
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(action.Quantity),
			name:                     action.Name,
			description:              action.Description,
			lastNameDescChangeHeight: blockHeight,
			reissuable:               action.Reissuable,
		},
	}

	issueStaticInfoSnapshot := &StaticAssetInfoSnapshot{
		AssetID:             action.ID,
		IssuerPublicKey:     *action.Sender,
		SourceTransactionID: txID,
		Decimals:            assetInf.decimals,
		IsNFT:               assetInf.isNFT(),
	}

	assetDescription := &AssetDescriptionSnapshot{
		AssetID:          action.ID,
		AssetName:        assetInf.name,
		AssetDescription: assetInf.description,
		ChangeHeight:     assetInf.lastNameDescChangeHeight,
	}

	assetReissuability := &AssetVolumeSnapshot{
		AssetID:       action.ID,
		IsReissuable:  assetInf.reissuable,
		TotalQuantity: assetInf.quantity,
	}

	var scriptInfo *scriptInformation
	if se := info.checkerData.scriptEstimations; se.isPresent() {
		// Save complexities to storage, so we won't have to calculate it every time the script is called.
		complexity, ok := se.estimations[se.currentEstimatorVersion]
		if !ok {
			return nil,
				errors.Errorf("failed to calculate asset script complexity by estimator version %d",
					se.currentEstimatorVersion)
		}
		scriptInfo = &scriptInformation{
			script:     action.Script,
			complexity: complexity.Verifier,
		}
	}
	if scriptInfo != nil {
		assetScriptSnapshot := &AssetScriptSnapshot{
			AssetID:    action.ID,
			Script:     scriptInfo.script,
			Complexity: uint64(scriptInfo.complexity),
		}
		atomicSnapshots = append(atomicSnapshots, assetScriptSnapshot)
	}
	atomicSnapshots = append(atomicSnapshots, issueStaticInfoSnapshot, assetDescription, assetReissuability)

	issuerAddress, err := proto.NewAddressFromPublicKey(sg.scheme, *action.Sender)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get an address from a public key")
	}
	assetBalanceDiff.appendOnlySender(issuerAddress, proto.AssetIDFromDigest(action.ID), action.Quantity)
	return atomicSnapshots, nil
}

func (sg *snapshotGenerator) atomicActionsFromReissueAction(
	action proto.ReissueScriptAction,
	assetBalanceDiff addressAssetBalanceDiff) ([]AtomicSnapshot, error) {
	var atomicSnapshots []AtomicSnapshot
	assetInf, err := sg.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(action.AssetID))
	if err != nil {
		return nil, err
	}
	quantityDiff := big.NewInt(action.Quantity)
	resQuantity := assetInf.quantity.Add(&assetInf.quantity, quantityDiff)
	assetReissuability := &AssetVolumeSnapshot{
		AssetID:       action.AssetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  action.Reissuable,
	}
	issueAddress, err := proto.NewAddressFromPublicKey(sg.scheme, *action.Sender)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get an address from a public key")
	}
	assetBalanceDiff.appendOnlySender(issueAddress, proto.AssetIDFromDigest(action.AssetID), action.Quantity)
	atomicSnapshots = append(atomicSnapshots, assetReissuability)
	return atomicSnapshots, nil
}

func (sg *snapshotGenerator) atomicActionsFromBurnAction(
	action proto.BurnScriptAction,
	assetBalanceDiff addressAssetBalanceDiff) ([]AtomicSnapshot, error) {
	var atomicSnapshots []AtomicSnapshot
	var assetInf *assetInfo
	assetInf, err := sg.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(action.AssetID))
	if err != nil {
		return nil, err
	}
	quantityDiff := big.NewInt(action.Quantity)
	resQuantity := assetInf.quantity.Sub(&assetInf.quantity, quantityDiff)
	assetReissuability := &AssetVolumeSnapshot{
		AssetID:       action.AssetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  assetInf.reissuable,
	}
	atomicSnapshots = append(atomicSnapshots, assetReissuability)
	issueAddress, err := proto.NewAddressFromPublicKey(sg.scheme, *action.Sender)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get an address from a public key")
	}
	assetBalanceDiff.appendOnlySender(issueAddress, proto.AssetIDFromDigest(action.AssetID), -action.Quantity)
	return atomicSnapshots, nil
}

func (sg *snapshotGenerator) atomicActionsFromLeaseAction(
	action proto.LeaseScriptAction,
	info *performerInfo,
	txID crypto.Digest) ([]AtomicSnapshot, error) {
	var atomicSnapshots []AtomicSnapshot
	senderAddress, err := proto.NewAddressFromPublicKey(sg.scheme, *action.Sender)
	if err != nil {
		return nil, err
	}
	var recipientAddr proto.WavesAddress
	if addr := action.Recipient.Address(); addr == nil {
		recipientAddr, err = sg.stor.aliases.newestAddrByAlias(action.Recipient.Alias().Alias)
		if err != nil {
			return nil, errors.Errorf("invalid alias: %v", err)
		}
	} else {
		recipientAddr = *addr
	}
	l := &leasing{
		Sender:    senderAddress,
		Recipient: recipientAddr,
		Amount:    uint64(action.Amount),
		Height:    info.height,
		Status:    LeaseActive,
	}
	var amount = int64(l.Amount)
	leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err :=
		sg.generateLeaseAtomicSnapshots(action.ID, *l, txID, senderAddress, recipientAddr, amount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate snapshots for a lease action")
	}
	atomicSnapshots = append(atomicSnapshots,
		leaseStatusSnapshot,
		senderLeaseBalanceSnapshot,
		recipientLeaseBalanceSnapshot)
	return atomicSnapshots, nil
}

func (sg *snapshotGenerator) atomicSnapshotsFromLeaseCancelAction(
	action proto.LeaseCancelScriptAction,
	txID crypto.Digest) ([]AtomicSnapshot, error) {
	var atomicSnapshots []AtomicSnapshot
	// TODO what if the leasing is not in the stor yet? lease and leaseCancel in the same contract?
	leasingInfo, err := sg.stor.leases.leasingInfo(action.LeaseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receiver leasing info")
	}

	var amount = -int64(leasingInfo.Amount)
	leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err :=
		sg.generateLeaseAtomicSnapshots(action.LeaseID, *leasingInfo, txID, leasingInfo.Sender, leasingInfo.Recipient, amount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate snapshots for a lease cancel action")
	}
	atomicSnapshots = append(atomicSnapshots,
		leaseStatusSnapshot,
		senderLeaseBalanceSnapshot,
		recipientLeaseBalanceSnapshot)
	return atomicSnapshots, nil
}

func (sg *snapshotGenerator) collectBalanceAndSnapshotFromAction(
	action proto.ScriptAction,
	dataEntries SenderDataEntries,
	wavesBalanceDiff addressWavesBalanceDiff,
	assetBalanceDiff addressAssetBalanceDiff,
	blockHeight uint64,
	info *performerInfo,
	txID crypto.Digest,
) ([]AtomicSnapshot, error) {
	var atomicSnapshots []AtomicSnapshot
	switch a := action.(type) {
	case *proto.DataEntryScriptAction:
		// snapshots store data entries in a different format, so we convert the actions to this format
		err := dataEntries.collectEntryFromAction(*a, sg.scheme)
		if err != nil {
			return nil, err
		}
	case *proto.AttachedPaymentScriptAction:
		err := sg.updateBalanceDiffFromPaymentAction(*a, wavesBalanceDiff, assetBalanceDiff)
		if err != nil {
			return nil, err
		}
	case *proto.TransferScriptAction:
		err := sg.updateBalanceDiffFromTransferAction(*a, wavesBalanceDiff, assetBalanceDiff)
		if err != nil {
			return nil, err
		}
	case *proto.SponsorshipScriptAction:
		sponsorshipSnapshot := &SponsorshipSnapshot{
			AssetID:         a.AssetID,
			MinSponsoredFee: uint64(a.MinFee),
		}
		atomicSnapshots = append(atomicSnapshots, sponsorshipSnapshot)
	case *proto.IssueScriptAction:
		issueSnapshots, err := sg.atomicSnapshotsFromIssueAction(*a, blockHeight, info, txID, assetBalanceDiff)
		if err != nil {
			return nil, err
		}
		atomicSnapshots = append(atomicSnapshots, issueSnapshots...)

	case *proto.ReissueScriptAction:
		reissueSnapshots, err := sg.atomicActionsFromReissueAction(*a, assetBalanceDiff)
		if err != nil {
			return nil, err
		}
		atomicSnapshots = append(atomicSnapshots, reissueSnapshots...)
	case *proto.BurnScriptAction:
		burnSnapshots, err := sg.atomicActionsFromBurnAction(*a, assetBalanceDiff)
		if err != nil {
			return nil, err
		}
		atomicSnapshots = append(atomicSnapshots, burnSnapshots...)
	case *proto.LeaseScriptAction:
		leaseSnapshots, err := sg.atomicActionsFromLeaseAction(*a, info, txID)
		if err != nil {
			return nil, err
		}
		atomicSnapshots = append(atomicSnapshots, leaseSnapshots...)
	case *proto.LeaseCancelScriptAction:
		leaseSnapshots, err := sg.atomicSnapshotsFromLeaseCancelAction(*a, txID)
		if err != nil {
			return nil, err
		}
		atomicSnapshots = append(atomicSnapshots, leaseSnapshots...)
	default:
		return nil, errors.Errorf("unknown script action type %T", a)
	}
	return atomicSnapshots, nil
}

func (sg *snapshotGenerator) atomicSnapshotsFromScriptActions(
	actions []proto.ScriptAction,
	wavesBalanceDiff addressWavesBalanceDiff,
	assetBalanceDiff addressAssetBalanceDiff,
	blockHeight uint64,
	info *performerInfo,
	txID crypto.Digest) ([]AtomicSnapshot, error) {
	var dataEntries = make(SenderDataEntries)
	var atomicSnapshots []AtomicSnapshot
	for _, action := range actions {
		snapshotsFromAction, err := sg.collectBalanceAndSnapshotFromAction(action, dataEntries,
			wavesBalanceDiff, assetBalanceDiff, blockHeight, info, txID)
		if err != nil {
			return nil, err
		}
		atomicSnapshots = append(atomicSnapshots, snapshotsFromAction...)
	}

	for address, entries := range dataEntries {
		dataEntrySnapshot := &DataEntriesSnapshot{Address: address, DataEntries: entries}
		atomicSnapshots = append(atomicSnapshots, dataEntrySnapshot)
	}
	return atomicSnapshots, nil
}

func (sg *snapshotGenerator) generateInvokeSnapshot(
	txID crypto.Digest,
	info *performerInfo,
	invocationRes *invocationResult,
	balanceChanges txDiff) (TransactionSnapshot, error) {
	blockHeight := info.height + 1

	addrWavesBalanceDiff, addrAssetBalanceDiff, err := balanceDiffFromTxDiff(balanceChanges, sg.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	var snapshot TransactionSnapshot
	if invocationRes != nil {
		var atomicSnapshots []AtomicSnapshot
		atomicSnapshots, err = sg.atomicSnapshotsFromScriptActions(
			invocationRes.actions, addrWavesBalanceDiff,
			addrAssetBalanceDiff, blockHeight, info, txID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate atomic snapshots from script actions")
		}
		snapshot = append(snapshot, atomicSnapshots...)
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

	return snapshot, nil
}

func (sg *snapshotGenerator) generateLeaseAtomicSnapshots(leaseID crypto.Digest,
	l leasing, originalTxID crypto.Digest,
	senderAddress proto.WavesAddress,
	receiverAddress proto.WavesAddress,
	amount int64) (*LeaseStateSnapshot, *LeaseBalanceSnapshot, *LeaseBalanceSnapshot, error) {
	leaseStatusSnapshot := &LeaseStateSnapshot{
		LeaseID: leaseID,
		Status: LeaseStateStatus{
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
	senderLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		Address:  senderAddress,
		LeaseIn:  uint64(senderBalanceProfile.leaseIn),
		LeaseOut: uint64(senderBalanceProfile.leaseOut + amount),
	}

	receiverBalanceProfile, err := sg.stor.balances.newestWavesBalance(receiverAddress.ID())
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to receive recipient's waves balance")
	}
	recipientLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		Address:  receiverAddress,
		LeaseIn:  uint64(receiverBalanceProfile.leaseIn + amount),
		LeaseOut: uint64(receiverBalanceProfile.leaseOut),
	}

	return leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, nil
}

func (sg *snapshotGenerator) generateOrderAtomicSnapshot(orderID []byte,
	volume uint64, fee uint64) (*FilledVolumeFeeSnapshot, error) {
	newestFilledAmount, newestFilledFee, err := sg.stor.ordersVolumes.newestFilled(orderID)
	if err != nil {
		return nil, err
	}
	orderIDDigset, err := crypto.NewDigestFromBytes(orderID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct digest from order id bytes")
	}
	orderSnapshot := &FilledVolumeFeeSnapshot{
		OrderID:      orderIDDigset,
		FilledFee:    newestFilledFee + fee,
		FilledVolume: newestFilledAmount + volume,
	}
	return orderSnapshot, nil
}

func (sg *snapshotGenerator) generateBalancesSnapshot(balanceChanges txDiff) (TransactionSnapshot, error) {
	var transactionSnapshot TransactionSnapshot
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
	addrAssetBalanceDiff addressAssetBalanceDiff) ([]WavesBalanceSnapshot, []AssetBalanceSnapshot, error) {
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
	diff addressWavesBalanceDiff) ([]WavesBalanceSnapshot, error) {
	var wavesBalances []WavesBalanceSnapshot
	// add miner address to the diff

	for wavesAddress, diffAmount := range diff {
		fullBalance, err := sg.stor.balances.newestWavesBalance(wavesAddress.ID())
		if err != nil {
			return nil, errors.Wrap(err, "failed to receive sender's waves balance")
		}
		newBalance := WavesBalanceSnapshot{
			Address: wavesAddress,
			Balance: uint64(int64(fullBalance.balance) + diffAmount.balance),
		}
		wavesBalances = append(wavesBalances, newBalance)
	}
	return wavesBalances, nil
}

func (sg *snapshotGenerator) assetBalanceSnapshotFromBalanceDiff(
	diff addressAssetBalanceDiff) ([]AssetBalanceSnapshot, error) {
	var assetBalances []AssetBalanceSnapshot
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

		newBalance := AssetBalanceSnapshot{
			Address: key.address,
			AssetID: key.asset.Digest(assetInfo.tail),
			Balance: uint64(int64(balance) + diffAmount),
		}
		assetBalances = append(assetBalances, newBalance)
	}
	return assetBalances, nil
}
