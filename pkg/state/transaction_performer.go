package state

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type performerInfo struct {
	height              uint64
	blockID             proto.BlockID
	currentMinerAddress proto.WavesAddress
	stateActionsCounter *proto.StateActionsCounter
}

type transactionPerformer struct {
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
}

type assetBalanceDiff struct {
	asset  proto.AssetID
	amount int64
}

type addressWavesBalanceDiff map[proto.WavesAddress]balanceDiff
type addressAssetBalanceDiff map[proto.WavesAddress]assetBalanceDiff

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
			addrAssetBalanceDiff[address] = assetBalanceDiff{asset: asset, amount: diffAmount.balance}
			continue
		}
		address, err := wavesBalanceKey.address.ToWavesAddress(scheme)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to convert address id to waves address")
		}
		addrWavesBalanceDiff[address] = diffAmount
	}
	return addrWavesBalanceDiff, addrAssetBalanceDiff, nil
}

//func (firstAddressBalanceDiff addressWavesBalanceDiff) mergeWithAnotherDiff(secondDiff addressWavesBalanceDiff) error {
//	for address, diffBalance := range secondDiff {
//		if _, ok := firstAddressBalanceDiff[address]; ok {
//			oldBalance := firstAddressBalanceDiff[address]
//			err := oldBalance.addCommon(&diffBalance)
//			if err != nil {
//				return errors.Wrap(err, "failed to merge two balance diffs")
//			}
//		}
//	}
//	return nil
//}

func newTransactionPerformer(stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) (*transactionPerformer, error) {
	return &transactionPerformer{stor, settings}, nil
}

// from txDiff and fees. no validation needed at this point
func (tp *transactionPerformer) constructWavesBalanceSnapshotFromDiff(diff addressWavesBalanceDiff) ([]WavesBalanceSnapshot, error) {
	var wavesBalances []WavesBalanceSnapshot
	// add miner address to the diff

	for wavesAddress, diffAmount := range diff {

		fullBalance, err := tp.stor.balances.wavesBalance(wavesAddress.ID())
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

func (tp *transactionPerformer) constructAssetBalanceSnapshotFromDiff(diff addressAssetBalanceDiff) ([]AssetBalanceSnapshot, error) {
	var assetBalances []AssetBalanceSnapshot
	// add miner address to the diff

	for wavesAddress, diffAmount := range diff {
		balance, err := tp.stor.balances.assetBalance(wavesAddress.ID(), diffAmount.asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to receive sender's waves balance")
		}
		assetInfo, err := tp.stor.assets.assetInfo(diffAmount.asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get newest asset info")
		}
		newBalance := AssetBalanceSnapshot{
			Address: wavesAddress,
			AssetID: diffAmount.asset.Digest(assetInfo.tail),
			Balance: uint64(int64(balance) + diffAmount.amount),
		}
		assetBalances = append(assetBalances, newBalance)
	}
	return assetBalances, nil
}

func (tp *transactionPerformer) constructBalancesSnapshotFromDiff(addrWavesBalanceDiff addressWavesBalanceDiff, addrAssetBalanceDiff addressAssetBalanceDiff) ([]WavesBalanceSnapshot, []AssetBalanceSnapshot, error) {
	wavesBalanceSnapshot, err := tp.constructWavesBalanceSnapshotFromDiff(addrWavesBalanceDiff)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to construct waves balance snapshot")
	}
	if len(addrAssetBalanceDiff) == 0 {
		return wavesBalanceSnapshot, nil, nil
	}

	assetBalanceSnapshot, err := tp.constructAssetBalanceSnapshotFromDiff(addrAssetBalanceDiff)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to construct asset balance snapshot")
	}
	return wavesBalanceSnapshot, assetBalanceSnapshot, nil
}

func (tp *transactionPerformer) performGenesis(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.Genesis)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	var snapshot TransactionSnapshot
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}

	}

	return snapshot, nil
}

func (tp *transactionPerformer) performPayment(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.Payment)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	var snapshot TransactionSnapshot
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}

	}

	return snapshot, nil
}

func (tp *transactionPerformer) performTransfer(applicationRes *applicationResult) (TransactionSnapshot, error) {

	var snapshot TransactionSnapshot
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}

	}

	return snapshot, nil
}

func (tp *transactionPerformer) performTransferWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	return tp.performTransfer(applicationRes)
}

func (tp *transactionPerformer) performTransferWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	return tp.performTransfer(applicationRes)
}

func (tp *transactionPerformer) performIssue(tx *proto.Issue, txID crypto.Digest, assetID crypto.Digest, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	blockHeight := info.height + 1
	// Create new asset.
	assetInfo := &assetInfo{
		assetConstInfo: assetConstInfo{
			tail:                 proto.DigestTail(assetID),
			issuer:               tx.SenderPK,
			decimals:             tx.Decimals,
			issueHeight:          blockHeight,
			issueSequenceInBlock: info.stateActionsCounter.NextIssueActionNumber(),
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(int64(tx.Quantity)),
			name:                     tx.Name,
			description:              tx.Description,
			lastNameDescChangeHeight: blockHeight,
			reissuable:               tx.Reissuable,
		},
	}

	var snapshot TransactionSnapshot
	issueStaticInfoSnapshot := &StaticAssetInfoSnapshot{
		AssetID:             assetID,
		IssuerPublicKey:     tx.SenderPK,
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
		TotalQuantity: *big.NewInt(int64(tx.Quantity)),
	}

	snapshot = append(snapshot, issueStaticInfoSnapshot, assetDescription, assetReissuability)
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		// Remove the just issues snapshot from the diff, because it's not in the storage yet, so can't be processed with constructBalancesSnapshotFromDiff
		var specialAssetSnapshot *AssetBalanceSnapshot
		for addr, diff := range addrAssetBalanceDiff {
			if diff.asset == proto.AssetIDFromDigest(assetID) {
				// remove the element from the array
				delete(addrAssetBalanceDiff, addr)
				specialAssetSnapshot = &AssetBalanceSnapshot{
					Address: addr,
					AssetID: assetID,
					Balance: uint64(diff.amount),
				}
			}
		}

		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
		if specialAssetSnapshot != nil {
			snapshot = append(snapshot, specialAssetSnapshot)
		}

	}

	if err := tp.stor.assets.issueAsset(proto.AssetIDFromDigest(assetID), assetInfo, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to issue asset")
	}

	return snapshot, nil
}

func (tp *transactionPerformer) performIssueWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	txID, err := tx.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return nil, err
	}
	if err := tp.stor.scriptsStorage.setAssetScript(assetID, proto.Script{}, tx.SenderPK, info.blockID); err != nil {
		return nil, err
	}
	return tp.performIssue(&tx.Issue, assetID, assetID, info, applicationRes)
}

func (tp *transactionPerformer) performIssueWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithProofs transaction")
	}
	txID, err := tx.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return nil, err
	}
	if err := tp.stor.scriptsStorage.setAssetScript(assetID, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return nil, err
	}
	return tp.performIssue(&tx.Issue, assetID, assetID, info, applicationRes)
}

func (tp *transactionPerformer) performReissue(tx *proto.Reissue, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	// Modify asset.
	change := &assetReissueChange{
		reissuable: tx.Reissuable,
		diff:       int64(tx.Quantity),
	}

	assetInfo, err := tp.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(tx.AssetID))
	if err != nil {
		return nil, err
	}

	quantityDiff := big.NewInt(change.diff)
	resQuantity := assetInfo.quantity.Add(&assetInfo.quantity, quantityDiff)
	assetReissuability := &AssetVolumeSnapshot{
		AssetID:       tx.AssetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  change.reissuable,
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, assetReissuability)

	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}

	if err := tp.stor.assets.reissueAsset(proto.AssetIDFromDigest(tx.AssetID), change, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to reissue asset")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performReissueWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to ReissueWithSig transaction")
	}
	return tp.performReissue(&tx.Reissue, info, applicationRes)
}

func (tp *transactionPerformer) performReissueWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to ReissueWithProofs transaction")
	}
	return tp.performReissue(&tx.Reissue, info, applicationRes)
}

func (tp *transactionPerformer) performBurn(tx *proto.Burn, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	// Modify asset.
	change := &assetBurnChange{
		diff: int64(tx.Amount),
	}

	assetInfo, err := tp.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(tx.AssetID))
	if err != nil {
		return nil, err
	}
	quantityDiff := big.NewInt(change.diff)
	resQuantity := assetInfo.quantity.Sub(&assetInfo.quantity, quantityDiff)
	assetReissuability := &AssetVolumeSnapshot{
		AssetID:       tx.AssetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  assetInfo.reissuable,
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, assetReissuability)

	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}

	if err := tp.stor.assets.burnAsset(proto.AssetIDFromDigest(tx.AssetID), change, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to burn asset")
	}

	return snapshot, nil
}

func (tp *transactionPerformer) performBurnWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to BurnWithSig transaction")
	}
	return tp.performBurn(&tx.Burn, info, applicationRes)
}

func (tp *transactionPerformer) performBurnWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to BurnWithProofs transaction")
	}
	return tp.performBurn(&tx.Burn, info, applicationRes)
}

func (tp *transactionPerformer) increaseOrderVolume(order proto.Order, tx proto.Exchange, info *performerInfo) (*FilledVolumeFeeSnapshot, error) {
	orderId, err := order.GetID()
	if err != nil {
		return nil, err
	}
	fee := tx.GetBuyMatcherFee()
	if order.GetOrderType() == proto.Sell {
		fee = tx.GetSellMatcherFee()
	}
	volume := tx.GetAmount()

	newestFilledFee, err := tp.stor.ordersVolumes.newestFilledFee(orderId)
	if err != nil {
		return nil, err
	}
	newestFilledAmount, err := tp.stor.ordersVolumes.newestFilledAmount(orderId)
	if err != nil {
		return nil, err
	}
	orderIdDigset, err := crypto.NewDigestFromBytes(orderId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct digest from order id bytes")
	}
	orderSnapshot := &FilledVolumeFeeSnapshot{
		OrderID:      orderIdDigset,
		FilledFee:    newestFilledFee + fee,
		FilledVolume: newestFilledAmount + volume,
	}

	if err := tp.stor.ordersVolumes.increaseFilledFee(orderId, fee, info.blockID); err != nil {
		return nil, err
	}
	if err := tp.stor.ordersVolumes.increaseFilledAmount(orderId, volume, info.blockID); err != nil {
		return nil, err
	}

	return orderSnapshot, nil
}

func (tp *transactionPerformer) performExchange(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return nil, errors.New("failed to convert interface to Exchange transaction")
	}
	so, err := tx.GetSellOrder()
	if err != nil {
		return nil, errors.Wrap(err, "no sell order")
	}
	sellOrderSnapshot, err := tp.increaseOrderVolume(so, tx, info)
	if err != nil {
		return nil, err
	}
	bo, err := tx.GetBuyOrder()
	if err != nil {
		return nil, errors.Wrap(err, "no buy order")
	}
	buyOrderSnapshot, err := tp.increaseOrderVolume(bo, tx, info)
	if err != nil {
		return nil, err
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, sellOrderSnapshot, buyOrderSnapshot)

	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performLease(tx *proto.Lease, id *crypto.Digest, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}
	var recipientAddr proto.WavesAddress
	if addr := tx.Recipient.Address(); addr == nil {
		recipientAddr, err = tp.stor.aliases.newestAddrByAlias(tx.Recipient.Alias().Alias)
		if err != nil {
			return nil, errors.Errorf("invalid alias: %v\n", err)
		}
	} else {
		recipientAddr = *addr
	}
	// Add leasing to lease state.
	l := &leasing{
		Sender:         senderAddr,
		Recipient:      recipientAddr,
		Amount:         tx.Amount,
		Height:         info.height,
		Status:         LeaseActive,
		RecipientAlias: tx.Recipient.Alias(),
	}
	leaseStatusSnapshot := &LeaseStateSnapshot{
		LeaseID:             *id,
		Status:              l.Status,
		Amount:              l.Amount,
		Sender:              l.Sender,
		Recipient:           l.Recipient,
		OriginTransactionID: *id,
		Height:              l.Height,
	}

	senderBalanceProfile, err := tp.stor.balances.wavesBalance(senderAddr.ID())
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive sender's waves balance")
	}
	senderLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		Address:  senderAddr,
		LeaseIn:  uint64(senderBalanceProfile.leaseIn),
		LeaseOut: uint64(senderBalanceProfile.leaseOut + int64(tx.Amount)),
	}

	receiverBalanceProfile, err := tp.stor.balances.wavesBalance(recipientAddr.ID())
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive recipient's waves balance")
	}
	recipientLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		Address:  recipientAddr,
		LeaseIn:  uint64(receiverBalanceProfile.leaseIn + int64(tx.Amount)),
		LeaseOut: uint64(receiverBalanceProfile.leaseOut),
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}

	if err := tp.stor.leases.addLeasing(*id, l, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to add leasing")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performLeaseWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	return tp.performLease(&tx.Lease, tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) performLeaseWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	return tp.performLease(&tx.Lease, tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) performLeaseCancel(tx *proto.LeaseCancel, txID *crypto.Digest, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	leasingInfo, err := tp.stor.leases.leasingInfo(tx.LeaseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receiver leasing info")
	}

	if err := tp.stor.leases.cancelLeasing(tx.LeaseID, info.blockID, info.height, txID); err != nil {
		return nil, errors.Wrap(err, "failed to cancel leasing")
	}
	leaseStatusSnapshot := &LeaseStateSnapshot{
		LeaseID:             tx.LeaseID,
		Status:              LeaseCanceled,
		Amount:              leasingInfo.Amount,
		Sender:              leasingInfo.Sender,
		Recipient:           leasingInfo.Recipient,
		OriginTransactionID: *txID,
		Height:              leasingInfo.Height,
	}

	// TODO check if the balance will be updated immediately after the leasing
	senderBalanceProfile, err := tp.stor.balances.wavesBalance(leasingInfo.Sender.ID())
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive sender's waves balance")
	}
	senderLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		Address:  leasingInfo.Sender,
		LeaseIn:  uint64(senderBalanceProfile.leaseIn),
		LeaseOut: uint64(senderBalanceProfile.leaseOut),
	}

	receiverBalanceProfile, err := tp.stor.balances.wavesBalance(leasingInfo.Recipient.ID())
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive recipient's waves balance")
	}
	recipientLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		Address:  leasingInfo.Recipient,
		LeaseIn:  uint64(receiverBalanceProfile.leaseIn),
		LeaseOut: uint64(receiverBalanceProfile.leaseOut),
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performLeaseCancelWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseCancelWithSig transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) performLeaseCancelWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseCancelWithProofs transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) performCreateAlias(tx *proto.CreateAlias, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}

	aliasSnapshot := &AliasSnapshot{
		Address: senderAddr,
		Alias:   tx.Alias,
	}
	var snapshot TransactionSnapshot
	snapshot = append(snapshot, aliasSnapshot)
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}

	if err := tp.stor.aliases.createAlias(tx.Alias.Alias, senderAddr, info.blockID); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performCreateAliasWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasWithSig transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info, applicationRes)
}

func (tp *transactionPerformer) performCreateAliasWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info, applicationRes)
}

func (tp *transactionPerformer) performDataWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.DataWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to DataWithProofs transaction")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}

	dataEntriesSnapshot := &DataEntriesSnapshot{
		Address:     senderAddr,
		DataEntries: tx.Entries,
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, dataEntriesSnapshot)
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}

	for _, entry := range tx.Entries {
		if err := tp.stor.accountsDataStor.appendEntry(senderAddr, entry, info.blockID); err != nil {
			return nil, err
		}
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performSponsorshipWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}
	sponsorshipSnapshot := &SponsorshipSnapshot{
		AssetID:         tx.AssetID,
		MinSponsoredFee: tx.MinAssetFee,
	}
	var snapshot TransactionSnapshot
	snapshot = append(snapshot, sponsorshipSnapshot)
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}

	if err := tp.stor.sponsoredAssets.sponsorAsset(tx.AssetID, tx.MinAssetFee, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to sponsor asset")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performSetScriptWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SetScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SetScriptWithProofs transaction")
	}
	var snapshot TransactionSnapshot

	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}
	sponsorshipSnapshot := &AccountScriptSnapshot{
		SenderPublicKey:    tx.SenderPK,
		Script:             tx.Script,
		VerifierComplexity: 0, // TODO fix it
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}

	if err := tp.stor.scriptsStorage.setAccountScript(senderAddr, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to set account script")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performSetAssetScriptWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SetAssetScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SetAssetScriptWithProofs transaction")
	}

	var snapshot TransactionSnapshot
	sponsorshipSnapshot := &AssetScriptSnapshot{
		AssetID:    tx.AssetID,
		Script:     tx.Script,
		Complexity: 0, // TDODO fix it
	}
	snapshot = append(snapshot, sponsorshipSnapshot)

	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}

	if err := tp.stor.scriptsStorage.setAssetScript(tx.AssetID, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to set asset script")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performInvokeScriptWithProofs(transaction proto.Transaction, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeScriptWithProofs); !ok {
		return nil, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to commit invoke changes")
	}
	// TODO
	if applicationRes != nil {
		for _, action := range invocationRes.actions {

			switch a := action.(type) {
			case *proto.DataEntryScriptAction:

			case *proto.AttachedPaymentScriptAction:

			case *proto.TransferScriptAction:

			case *proto.SponsorshipScriptAction:

			case *proto.IssueScriptAction:

			case *proto.ReissueScriptAction:

			case *proto.BurnScriptAction:

			case *proto.LeaseScriptAction:

			case *proto.LeaseCancelScriptAction:

			default:
				return nil, errors.Errorf("unknown script action type %T", a)
			}
		}
	}
	return nil, nil
}

func (tp *transactionPerformer) performInvokeExpressionWithProofs(transaction proto.Transaction, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs); !ok {
		return nil, errors.New("failed to convert interface to InvokeExpressionWithProofs transaction")
	}
	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to commit invoke changes")
	}
	// TODO
	return nil, nil
}

func (tp *transactionPerformer) performEthereumTransactionWithProofs(transaction proto.Transaction, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	ethTx, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return nil, errors.New("failed to convert interface to EthereumTransaction transaction")
	}
	if _, ok := ethTx.TxKind.(*proto.EthereumInvokeScriptTxKind); ok {
		if err := tp.stor.commitUncertain(info.blockID); err != nil {
			return nil, errors.Wrap(err, "failed to commit invoke changes")
		}
	}
	var snapshot TransactionSnapshot
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}

	// nothing to do for proto.EthereumTransferWavesTxKind and proto.EthereumTransferAssetsErc20TxKind
	return snapshot, nil
}

func (tp *transactionPerformer) performUpdateAssetInfoWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.UpdateAssetInfoWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to UpdateAssetInfoWithProofs transaction")
	}
	blockHeight := info.height + 1
	ch := &assetInfoChange{
		newName:        tx.Name,
		newDescription: tx.Description,
		newHeight:      blockHeight,
	}

	sponsorshipSnapshot := &AssetDescriptionSnapshot{
		AssetID:          tx.AssetID,
		AssetName:        tx.Name,
		AssetDescription: tx.Description,
		ChangeHeight:     blockHeight,
	}
	var snapshot TransactionSnapshot
	snapshot = append(snapshot, sponsorshipSnapshot)
	if applicationRes != nil {
		addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
		}
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(addrWavesBalanceDiff, addrAssetBalanceDiff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		for _, wb := range wavesBalanceSnapshot {
			snapshot = append(snapshot, &wb)
		}
		for _, ab := range assetBalanceSnapshot {
			snapshot = append(snapshot, &ab)
		}
	}

	if err := tp.stor.assets.updateAssetInfo(tx.AssetID, ch, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to update asset info")
	}
	return snapshot, nil
}
