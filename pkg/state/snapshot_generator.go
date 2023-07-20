package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"math/big"
)

type snapshotGenerator struct {
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
}

type assetBalanceDiffKey struct {
	address proto.WavesAddress
	asset   proto.AssetID
}

type addressWavesBalanceDiff map[proto.WavesAddress]balanceDiff
type addressAssetBalanceDiff map[assetBalanceDiffKey]int64

func (sg *snapshotGenerator) generateSnapshotForGenesisTx(balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForPaymentTx(balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForTransferTx(balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
	return sg.generateBalancesSnapshot(balanceChanges)
}

type scriptInformation struct {
	script     proto.Script
	complexity int
}

func (sg *snapshotGenerator) generateSnapshotForIssueTx(assetID crypto.Digest, txID crypto.Digest, senderPK crypto.PublicKey, assetInfo assetInfo, balanceChanges txDiff, scriptInformation *scriptInformation) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
	var snapshot TransactionSnapshot
	// TODO add asset script snapshot
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(balanceChanges, sg.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	// Remove the just issues snapshot from the diff, because it's not in the storage yet, so can't be processed with generateBalancesAtomicSnapshots
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
		sponsorshipSnapshot := &AssetScriptSnapshot{
			AssetID:    assetID,
			Script:     scriptInformation.script,
			Complexity: uint64(scriptInformation.complexity),
		}
		snapshot = append(snapshot, sponsorshipSnapshot)
	}

	wavesBalancesSnapshot, assetBalancesSnapshot, err := sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
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

func (sg *snapshotGenerator) generateSnapshotForReissueTx(assetID crypto.Digest, change assetReissueChange, balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
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

func (sg *snapshotGenerator) generateSnapshotForBurnTx(assetID crypto.Digest, change assetBurnChange, balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
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

func (sg *snapshotGenerator) generateSnapshotForExchangeTx(sellOrder proto.Order, sellFee uint64, buyOrder proto.Order, buyFee uint64, volume uint64, balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
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

func (sg *snapshotGenerator) generateSnapshotForLeaseTx(lease leasing, leaseID crypto.Digest, originalTxID crypto.Digest, balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
	var err error
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	amount := int64(lease.Amount)
	leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err := sg.generateLeaseAtomicSnapshots(leaseID, lease, originalTxID, lease.Sender, lease.Recipient, amount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate snapshots for a lease transaction")
	}

	snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForLeaseCancelTx(txID *crypto.Digest, oldLease leasing, leaseID crypto.Digest, originalTxID crypto.Digest, cancelHeight uint64, balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
	var err error
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	negativeAmount := -int64(oldLease.Amount)
	leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err := sg.generateLeaseAtomicSnapshots(leaseID, oldLease, originalTxID, oldLease.Sender, oldLease.Recipient, negativeAmount)
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

func (sg *snapshotGenerator) generateSnapshotForCreateAliasTx(senderAddress proto.WavesAddress, alias proto.Alias, balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
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
	if balanceChanges == nil {
		return nil, nil
	}
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForDataTx(senderAddress proto.WavesAddress, entries []proto.DataEntry, balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
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

func (sg *snapshotGenerator) generateSnapshotForSponsorshipTx(assetID crypto.Digest, minAssetFee uint64, balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
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

func (sg *snapshotGenerator) generateSnapshotForSetScriptTx(senderPK crypto.PublicKey, script proto.Script, complexity int, info *performerInfo, balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
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

func (sg *snapshotGenerator) generateSnapshotForSetAssetScriptTx(assetID crypto.Digest, script proto.Script, complexity int, balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return nil, err
	}

	sponsorshipSnapshot := &AssetScriptSnapshot{
		AssetID:    assetID,
		Script:     script,
		Complexity: uint64(complexity),
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForInvokeScriptTx(txID crypto.Digest, info *performerInfo, invocationRes *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	return sg.generateInvokeSnapshot(txID, info, invocationRes, balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForInvokeExpressionTx(txID crypto.Digest, info *performerInfo, invocationRes *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	return sg.generateInvokeSnapshot(txID, info, invocationRes, balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForEthereumInvokeScriptTx(txID crypto.Digest, info *performerInfo, invocationRes *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	return sg.generateInvokeSnapshot(txID, info, invocationRes, balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForUpdateAssetInfoTx(assetID crypto.Digest, assetName string, assetDescription string, changeHeight proto.Height, balanceChanges txDiff) (TransactionSnapshot, error) {
	if balanceChanges == nil {
		return nil, nil
	}

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

// TODO optimize this
func (sg *snapshotGenerator) generateInvokeSnapshot(
	txID crypto.Digest,
	info *performerInfo,
	invocationRes *invocationResult,
	balanceChanges txDiff) (TransactionSnapshot, error) {

	blockHeight := info.height + 1

	addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(balanceChanges, sg.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	var snapshot TransactionSnapshot
	var dataEntries = make(map[proto.WavesAddress]proto.DataEntries)
	if invocationRes != nil {

		for _, action := range invocationRes.actions {

			switch a := action.(type) {
			case *proto.DataEntryScriptAction:
				senderAddr, err := proto.NewAddressFromPublicKey(sg.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, err
				}
				// construct the map first and create the snapshot later for convenience
				if _, ok := dataEntries[senderAddr]; ok {
					entries := dataEntries[senderAddr]
					entries = append(entries, a.Entry)
					dataEntries[senderAddr] = entries
				} else {
					dataEntries[senderAddr] = proto.DataEntries{a.Entry}
				}

			case *proto.AttachedPaymentScriptAction:
				senderAddress, err := proto.NewAddressFromPublicKey(sg.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get an address from a public key")
				}
				recipientAddress, err := recipientToAddress(a.Recipient, sg.stor.aliases)
				if err != nil {
					return nil, errors.Wrap(err, "failed to apply attached payment")
				}
				// No balance validation done below
				if a.Asset.Present { // Update asset balance
					addSenderRecipientToAssetBalanceDiff(addrAssetBalanceDiff, senderAddress, recipientAddress, proto.AssetIDFromDigest(a.Asset.ID), a.Amount)
				} else { // Update Waves balance
					addToWavesBalanceDiff(addrWavesBalanceDiff, senderAddress, recipientAddress, a.Amount)
				}
			case *proto.TransferScriptAction:
				senderAddress, err := proto.NewAddressFromPublicKey(sg.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get an address from a public key")
				}
				recipientAddress, err := recipientToAddress(a.Recipient, sg.stor.aliases)
				if err != nil {
					return nil, errors.Wrap(err, "failed to apply attached payment")
				}
				// No balance validation done below
				if a.Asset.Present { // Update asset balance
					addSenderRecipientToAssetBalanceDiff(addrAssetBalanceDiff, senderAddress, recipientAddress, proto.AssetIDFromDigest(a.Asset.ID), a.Amount)
				} else { // Update Waves balance
					addToWavesBalanceDiff(addrWavesBalanceDiff, senderAddress, recipientAddress, a.Amount)
				}
			case *proto.SponsorshipScriptAction:
				sponsorshipSnapshot := &SponsorshipSnapshot{
					AssetID:         a.AssetID,
					MinSponsoredFee: uint64(a.MinFee),
				}
				snapshot = append(snapshot, sponsorshipSnapshot)
			case *proto.IssueScriptAction:
				assetInfo := assetInfo{
					assetConstInfo: assetConstInfo{
						tail:                 proto.DigestTail(a.ID),
						issuer:               *a.Sender,
						decimals:             uint8(a.Decimals),
						issueHeight:          blockHeight,
						issueSequenceInBlock: info.stateActionsCounter.NextIssueActionNumber(),
					},
					assetChangeableInfo: assetChangeableInfo{
						quantity:                 *big.NewInt(a.Quantity),
						name:                     a.Name,
						description:              a.Description,
						lastNameDescChangeHeight: blockHeight,
						reissuable:               a.Reissuable,
					},
				}
				issuerAddress, err := proto.NewAddressFromPublicKey(sg.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get an address from a public key")
				}

				issueStaticInfoSnapshot := &StaticAssetInfoSnapshot{
					AssetID:             a.ID,
					IssuerPublicKey:     *a.Sender,
					SourceTransactionID: txID,
					Decimals:            assetInfo.decimals,
					IsNFT:               assetInfo.isNFT(),
				}

				assetDescription := &AssetDescriptionSnapshot{
					AssetID:          a.ID,
					AssetName:        assetInfo.name,
					AssetDescription: assetInfo.description,
					ChangeHeight:     assetInfo.lastNameDescChangeHeight,
				}

				assetReissuability := &AssetVolumeSnapshot{
					AssetID:       a.ID,
					IsReissuable:  assetInfo.reissuable,
					TotalQuantity: assetInfo.quantity,
				}
				snapshot = append(snapshot, issueStaticInfoSnapshot, assetDescription, assetReissuability)

				addSenderToAssetBalanceDiff(addrAssetBalanceDiff, issuerAddress, proto.AssetIDFromDigest(a.ID), a.Quantity)

			case *proto.ReissueScriptAction:

				assetInfo, err := sg.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(a.AssetID))
				if err != nil {
					return nil, err
				}
				quantityDiff := big.NewInt(a.Quantity)
				resQuantity := assetInfo.quantity.Add(&assetInfo.quantity, quantityDiff)
				assetReissuability := &AssetVolumeSnapshot{
					AssetID:       a.AssetID,
					TotalQuantity: *resQuantity,
					IsReissuable:  a.Reissuable,
				}

				issueAddress, err := proto.NewAddressFromPublicKey(sg.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get an address from a public key")
				}
				addSenderToAssetBalanceDiff(addrAssetBalanceDiff, issueAddress, proto.AssetIDFromDigest(a.AssetID), a.Quantity)
				snapshot = append(snapshot, assetReissuability)

			case *proto.BurnScriptAction:
				assetInfo, err := sg.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(a.AssetID))
				if err != nil {
					return nil, err
				}
				quantityDiff := big.NewInt(a.Quantity)
				resQuantity := assetInfo.quantity.Sub(&assetInfo.quantity, quantityDiff)
				assetReissuability := &AssetVolumeSnapshot{
					AssetID:       a.AssetID,
					TotalQuantity: *resQuantity,
					IsReissuable:  assetInfo.reissuable,
				}

				issueAddress, err := proto.NewAddressFromPublicKey(sg.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get an address from a public key")
				}
				addSenderToAssetBalanceDiff(addrAssetBalanceDiff, issueAddress, proto.AssetIDFromDigest(a.AssetID), -a.Quantity)
				snapshot = append(snapshot, assetReissuability)
			case *proto.LeaseScriptAction:
				senderAddr, err := proto.NewAddressFromPublicKey(sg.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, err
				}
				var recipientAddr proto.WavesAddress
				if addr := a.Recipient.Address(); addr == nil {
					recipientAddr, err = sg.stor.aliases.newestAddrByAlias(a.Recipient.Alias().Alias)
					if err != nil {
						return nil, errors.Errorf("invalid alias: %v\n", err)
					}
				} else {
					recipientAddr = *addr
				}
				l := &leasing{
					Sender:    senderAddr,
					Recipient: recipientAddr,
					Amount:    uint64(a.Amount),
					Height:    info.height,
					Status:    LeaseActive,
				}
				var amount = int64(l.Amount)
				leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err := sg.generateLeaseAtomicSnapshots(a.ID, *l, txID, senderAddr, recipientAddr, amount)
				if err != nil {
					return nil, errors.Wrap(err, "failed to generate snapshots for a lease action")
				}
				snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
			case *proto.LeaseCancelScriptAction:
				l, err := sg.stor.leases.leasingInfo(a.LeaseID)
				if err != nil {
					return nil, errors.Wrap(err, "failed to receiver leasing info")
				}

				var amount = -int64(l.Amount)
				leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err := sg.generateLeaseAtomicSnapshots(a.LeaseID, *l, txID, l.Sender, l.Recipient, amount)
				if err != nil {
					return nil, errors.Wrap(err, "failed to generate snapshots for a lease cancel action")
				}
				snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
			default:
				return nil, errors.Errorf("unknown script action type %T", a)
			}
		}

		for address, entries := range dataEntries {
			dataEntrySnapshot := &DataEntriesSnapshot{Address: address, DataEntries: entries}
			snapshot = append(snapshot, dataEntrySnapshot)
		}

	}

	wavesBalancesSnapshot, assetBalancesSnapshot, err := sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
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

func (sg *snapshotGenerator) generateLeaseAtomicSnapshots(leaseID crypto.Digest, l leasing, originalTxID crypto.Digest,
	senderAddress proto.WavesAddress, receiverAddress proto.WavesAddress, amount int64) (*LeaseStateSnapshot, *LeaseBalanceSnapshot, *LeaseBalanceSnapshot, error) {
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

func (sg *snapshotGenerator) generateOrderAtomicSnapshot(orderID []byte, volume uint64, fee uint64) (*FilledVolumeFeeSnapshot, error) {
	newestFilledAmount, newestFilledFee, err := sg.stor.ordersVolumes.newestFilled(orderID)
	if err != nil {
		return nil, err
	}
	orderIdDigset, err := crypto.NewDigestFromBytes(orderID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct digest from order id bytes")
	}
	orderSnapshot := &FilledVolumeFeeSnapshot{
		OrderID:      orderIdDigset,
		FilledFee:    newestFilledFee + fee,
		FilledVolume: newestFilledAmount + volume,
	}
	return orderSnapshot, nil
}

func (sg *snapshotGenerator) generateBalancesSnapshot(balanceChanges txDiff) (TransactionSnapshot, error) {
	var transactionSnapshot TransactionSnapshot
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(balanceChanges, sg.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	wavesBalancesSnapshot, assetBalancesSnapshot, err := sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
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

func (sg *snapshotGenerator) generateBalancesAtomicSnapshots(addrWavesBalanceDiff addressWavesBalanceDiff, addrAssetBalanceDiff addressAssetBalanceDiff) ([]WavesBalanceSnapshot, []AssetBalanceSnapshot, error) {
	wavesBalanceSnapshot, err := sg.constructWavesBalanceSnapshotFromDiff(addrWavesBalanceDiff)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to construct waves balance snapshot")
	}
	if len(addrAssetBalanceDiff) == 0 {
		return wavesBalanceSnapshot, nil, nil
	}

	assetBalanceSnapshot, err := sg.constructAssetBalanceSnapshotFromDiff(addrAssetBalanceDiff)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to construct asset balance snapshot")
	}
	return wavesBalanceSnapshot, assetBalanceSnapshot, nil
}

func addressBalanceDiffFromTxDiff(diff txDiff, scheme proto.Scheme) (addressWavesBalanceDiff, addressAssetBalanceDiff, error) {
	addrWavesBalanceDiff := make(addressWavesBalanceDiff)
	addrAssetBalanceDiff := make(addressAssetBalanceDiff)
	for balanceKeyString, diffAmount := range diff {

		// construct address from key
		wavesBalanceKey := &wavesBalanceKey{}
		err := wavesBalanceKey.unmarshal([]byte(balanceKeyString))
		if err != nil {
			assetBalanceKey := &assetBalanceKey{}
			err := assetBalanceKey.unmarshal([]byte(balanceKeyString))
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to convert balance key to asset balance key")
			}
			asset := assetBalanceKey.asset
			address, err := assetBalanceKey.address.ToWavesAddress(scheme)
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to convert address id to waves address")
			}
			assetBalKey := assetBalanceDiffKey{address: address, asset: asset}
			addrAssetBalanceDiff[assetBalKey] = diffAmount.balance
			continue
		}
		address, err := wavesBalanceKey.address.ToWavesAddress(scheme)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to convert address id to waves address")
		}
		// if the waves balance diff is 0, it means it did not change. Though the record might occur when LeaseIn and LeaseOut change,
		// but they are handled separately in snapshots
		if diffAmount.balance == 0 {
			continue
		}
		addrWavesBalanceDiff[address] = diffAmount
	}
	return addrWavesBalanceDiff, addrAssetBalanceDiff, nil
}

// from txDiff and fees. no validation needed at this point.
func (sg *snapshotGenerator) constructWavesBalanceSnapshotFromDiff(diff addressWavesBalanceDiff) ([]WavesBalanceSnapshot, error) {
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

func (sg *snapshotGenerator) constructAssetBalanceSnapshotFromDiff(diff addressAssetBalanceDiff) ([]AssetBalanceSnapshot, error) {
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
