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
	checkerInfo         *checkerInfo
}

type transactionPerformer struct {
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
}

func newTransactionPerformer(stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) (*transactionPerformer, error) {
	return &transactionPerformer{stor, settings}, nil
}

func (tp *transactionPerformer) performGenesis(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.Genesis)
	if !ok {
		return nil, errors.New("failed to convert interface to genesis transaction")
	}
	return tp.generateSnapshotForGenesisTx(applicationRes)
}

func (tp *transactionPerformer) performPayment(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.Payment)
	if !ok {
		return nil, errors.New("failed to convert interface to payment transaction")
	}
	return tp.generateSnapshotForPaymentTx(applicationRes)
}

func (tp *transactionPerformer) performTransfer(applicationRes *applicationResult) (TransactionSnapshot, error) {
	return tp.generateSnapshotForTransferTx(applicationRes)
}

func (tp *transactionPerformer) performTransferWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to transfer with sig transaction")
	}
	return tp.performTransfer(applicationRes)
}

func (tp *transactionPerformer) performTransferWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to transfer with proofs transaction")
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

	snapshot, err := tp.generateSnapshotForIssueTx(assetID, txID, tx.SenderPK, *assetInfo, applicationRes)
	if err != nil {
		return nil, err
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

	snapshot, err := tp.generateSnapshotForReissueTx(tx.AssetID, *change, applicationRes)
	if err != nil {
		return nil, err
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

	snapshot, err := tp.generateSnapshotForBurnTx(tx.AssetID, *change, applicationRes)
	if err != nil {
		return nil, err
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

func (tp *transactionPerformer) increaseOrderVolume(order proto.Order, fee uint64, volume uint64, info *performerInfo) error {
	orderID, err := order.GetID()
	if err != nil {
		return err
	}
	return tp.stor.ordersVolumes.increaseFilled(orderID, volume, fee, info.blockID)
}

func (tp *transactionPerformer) performExchange(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return nil, errors.New("failed to convert interface to Exchange transaction")
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return nil, errors.Wrap(err, "no sell order")
	}
	buyOrder, err := tx.GetBuyOrder()
	if err != nil {
		return nil, errors.Wrap(err, "no buy order")
	}
	volume := tx.GetAmount()
	sellFee := tx.GetSellMatcherFee()
	buyFee := tx.GetBuyMatcherFee()

	// snapshot must be generated before the state with orders is changed
	snapshot, err := tp.generateSnapshotForExchangeTx(sellOrder, sellFee, buyOrder, buyFee, volume, applicationRes)
	if err != nil {
		return nil, err
	}

	err = tp.increaseOrderVolume(sellOrder, sellFee, volume, info)
	if err != nil {
		return nil, err
	}
	err = tp.increaseOrderVolume(buyOrder, buyFee, volume, info)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performLease(tx *proto.Lease, txID crypto.Digest, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
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
		Sender:    senderAddr,
		Recipient: recipientAddr,
		Amount:    tx.Amount,
		Height:    info.height,
		Status:    LeaseActive,
	}
	snapshot, err := tp.generateSnapshotForLeaseTx(*l, txID, txID, applicationRes)
	if err != nil {
		return nil, nil
	}

	if err := tp.stor.leases.addLeasing(txID, l, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to add leasing")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performLeaseWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	return tp.performLease(&tx.Lease, *tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) performLeaseWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	return tp.performLease(&tx.Lease, *tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) performLeaseCancel(tx *proto.LeaseCancel, txID *crypto.Digest, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	oldLease, err := tp.stor.leases.newestLeasingInfo(tx.LeaseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receiver leasing info")
	}

	snapshot, err := tp.generateSnapshotForLeaseCancelTx(txID, *oldLease, tx.LeaseID, *oldLease.OriginTransactionID, info.height, applicationRes)
	if err != nil {
		return nil, err
	}
	if err := tp.stor.leases.cancelLeasing(tx.LeaseID, info.blockID, info.height, txID); err != nil {
		return nil, errors.Wrap(err, "failed to cancel leasing")
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

	snapshot, err := tp.generateSnapshotForCreateAliasTx(senderAddr, tx.Alias, applicationRes)
	if err != nil {
		return nil, err
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

func (tp *transactionPerformer) performMassTransferWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.MassTransferWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return tp.generateSnapshotForMassTransferTx(applicationRes)
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

	snapshot, err := tp.generateSnapshotForDataTx(senderAddr, tx.Entries, applicationRes)
	if err != nil {
		return nil, err
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

	snapshot, err := tp.generateSnapshotForSponsorshipTx(tx.AssetID, tx.MinAssetFee, applicationRes)
	if err != nil {
		return nil, err
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
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}

	snapshot, err := tp.generateSnapshotForSetScriptTx(senderAddr, tx.SenderPK, tx.Script, info, applicationRes)
	if err != nil {
		return nil, err
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

	snapshot, err := tp.generateSnapshotForSetAssetScriptTx(tx.AssetID, tx.Script, applicationRes)
	if err != nil {
		return nil, err
	}

	if err := tp.stor.scriptsStorage.setAssetScript(tx.AssetID, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to set asset script")
	}
	return snapshot, nil
}

func addToWavesBalanceDiff(addrWavesBalanceDiff addressWavesBalanceDiff,
	senderAddress proto.WavesAddress,
	recipientAddress proto.WavesAddress,
	amount int64) {
	if _, ok := addrWavesBalanceDiff[senderAddress]; ok {
		prevBalance := addrWavesBalanceDiff[senderAddress]
		prevBalance.balance -= amount
		addrWavesBalanceDiff[senderAddress] = prevBalance
	} else {
		addrWavesBalanceDiff[senderAddress] = balanceDiff{balance: amount}
	}

	if _, ok := addrWavesBalanceDiff[recipientAddress]; ok {
		prevRecipientBalance := addrWavesBalanceDiff[recipientAddress]
		prevRecipientBalance.balance += amount
		addrWavesBalanceDiff[recipientAddress] = prevRecipientBalance
	} else {
		addrWavesBalanceDiff[recipientAddress] = balanceDiff{balance: amount}
	}
}

// subtracts the amount from the sender's balance and add it to the recipient's balane
func addSenderRecipientToAssetBalanceDiff(addrAssetBalanceDiff addressAssetBalanceDiff,
	senderAddress proto.WavesAddress,
	recipientAddress proto.WavesAddress,
	asset proto.AssetID,
	amount int64) {
	keySender := assetBalanceDiffKey{address: senderAddress, asset: asset}
	keyRecipient := assetBalanceDiffKey{address: recipientAddress, asset: asset}

	if _, ok := addrAssetBalanceDiff[keySender]; ok {
		prevSenderBalance := addrAssetBalanceDiff[keySender]
		prevSenderBalance -= amount
		addrAssetBalanceDiff[keySender] = prevSenderBalance
	} else {
		addrAssetBalanceDiff[keySender] = amount
	}

	if _, ok := addrAssetBalanceDiff[keyRecipient]; ok {
		prevRecipientBalance := addrAssetBalanceDiff[keyRecipient]
		prevRecipientBalance += amount
		addrAssetBalanceDiff[keyRecipient] = prevRecipientBalance
	} else {
		addrAssetBalanceDiff[keyRecipient] = amount
	}
}

// adds/subtracts the amount to the sender balance
func addSenderToAssetBalanceDiff(addrAssetBalanceDiff addressAssetBalanceDiff,
	senderAddress proto.WavesAddress,
	asset proto.AssetID,
	amount int64) {
	keySender := assetBalanceDiffKey{address: senderAddress, asset: asset}

	if _, ok := addrAssetBalanceDiff[keySender]; ok {
		prevSenderBalance := addrAssetBalanceDiff[keySender]
		prevSenderBalance += amount
		addrAssetBalanceDiff[keySender] = prevSenderBalance
	} else {
		addrAssetBalanceDiff[keySender] = amount
	}

}

func (tp *transactionPerformer) performInvokeScriptWithProofs(transaction proto.Transaction, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeScriptWithProofs); !ok {
		return nil, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to commit invoke changes")
	}
	txIDBytes, err := transaction.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	txID, err := crypto.NewDigestFromBytes(txIDBytes)
	if err != nil {
		return nil, err
	}

	snapshot, err := tp.generateSnapshotForInvokeScriptTx(txID, info, invocationRes, applicationRes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot for an invoke transaction")
	}

	return snapshot, nil
}

func (tp *transactionPerformer) performInvokeExpressionWithProofs(transaction proto.Transaction, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs); !ok {
		return nil, errors.New("failed to convert interface to InvokeExpressionWithProofs transaction")
	}
	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to commit invoke changes")
	}
	txIDBytes, err := transaction.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	txID, err := crypto.NewDigestFromBytes(txIDBytes)
	if err != nil {
		return nil, err
	}

	return tp.generateSnapshotForInvokeExpressionTx(txID, info, invocationRes, applicationRes)
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
	txIDBytes, err := transaction.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	txID, err := crypto.NewDigestFromBytes(txIDBytes)
	if err != nil {
		return nil, err
	}

	snapshot, err := tp.generateSnapshotForEthereumInvokeScriptTx(txID, info, invocationRes, applicationRes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot for an invoke transaction")
	}

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

	snapshot, err := tp.generateSnapshotForUpdateAssetInfoTx(tx.AssetID, tx.Name, tx.Description, blockHeight, applicationRes)
	if err != nil {
		return nil, err
	}
	if err := tp.stor.assets.updateAssetInfo(tx.AssetID, ch, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to update asset info")
	}
	return snapshot, nil
}
