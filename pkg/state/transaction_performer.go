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
	checkerData         txCheckerData
}

func newPerformerInfo(height proto.Height,
	blockID proto.BlockID, currentMinerAddress proto.WavesAddress,
	checkerData txCheckerData) *performerInfo {
	return &performerInfo{height, blockID,
		currentMinerAddress,
		checkerData} // all fields must be initialized
}

type transactionPerformer struct {
	stor              *blockchainEntitiesStorage
	settings          *settings.BlockchainSettings
	snapshotGenerator *snapshotGenerator      // initialized in appendTx
	snapshotApplier   extendedSnapshotApplier // initialized in appendTx
}

func newTransactionPerformer(stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) (*transactionPerformer, error) {
	return &transactionPerformer{stor: stor, settings: settings}, nil
}

func (tp *transactionPerformer) performGenesis(
	transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult,
	balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.Genesis)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to genesis transaction")
	}
	snapshot, err := tp.snapshotGenerator.generateSnapshotForGenesisTx(balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performPayment(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.Payment)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to payment transaction")
	}
	snapshot, err := tp.snapshotGenerator.generateSnapshotForPaymentTx(balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performTransfer(balanceChanges txDiff) (txSnapshot, error) {
	snapshot, err := tp.snapshotGenerator.generateSnapshotForTransferTx(balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performTransferWithSig(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to transfer with sig transaction")
	}
	return tp.performTransfer(balanceChanges)
}

func (tp *transactionPerformer) performTransferWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to transfer with proofs transaction")
	}
	return tp.performTransfer(balanceChanges)
}

func (tp *transactionPerformer) performIssue(tx *proto.Issue, txID crypto.Digest,
	assetID crypto.Digest, info *performerInfo,
	balanceChanges txDiff, scriptEstimation *scriptEstimation, script *proto.Script) (txSnapshot, error) {
	blockHeight := info.height + 1
	// Create new asset.
	assetInfo := &assetInfo{
		assetConstInfo: assetConstInfo{
			tail:        proto.DigestTail(assetID),
			issuer:      tx.SenderPK,
			decimals:    tx.Decimals,
			issueHeight: blockHeight,
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(int64(tx.Quantity)),
			name:                     tx.Name,
			description:              tx.Description,
			lastNameDescChangeHeight: blockHeight,
			reissuable:               tx.Reissuable,
		},
	}
	snapshot, err := tp.snapshotGenerator.generateSnapshotForIssueTx(assetID, txID, tx.SenderPK,
		*assetInfo, balanceChanges, scriptEstimation, script)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performIssueWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	txID, err := tx.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return txSnapshot{}, errors.Errorf("failed to get transaction ID: %v", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return txSnapshot{}, err
	}
	return tp.performIssue(&tx.Issue, assetID, assetID, info, balanceChanges, nil, nil)
}

func (tp *transactionPerformer) performIssueWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to IssueWithProofs transaction")
	}
	txID, err := tx.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return txSnapshot{}, errors.Errorf("failed to get transaction ID: %v", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return txSnapshot{}, err
	}
	return tp.performIssue(&tx.Issue, assetID, assetID, info,
		balanceChanges, info.checkerData.scriptEstimation, &tx.Script)
}

func (tp *transactionPerformer) performReissue(tx *proto.Reissue, _ *performerInfo,
	balanceChanges txDiff) (txSnapshot, error) {
	// Modify asset.
	change := &assetReissueChange{
		reissuable: tx.Reissuable,
		diff:       int64(tx.Quantity),
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForReissueTx(tx.AssetID, *change, balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performReissueWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to ReissueWithSig transaction")
	}
	return tp.performReissue(&tx.Reissue, info, balanceChanges)
}

func (tp *transactionPerformer) performReissueWithProofs(transaction proto.Transaction,
	info *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to ReissueWithProofs transaction")
	}
	return tp.performReissue(&tx.Reissue, info, balanceChanges)
}

func (tp *transactionPerformer) performBurn(tx *proto.Burn, _ *performerInfo,
	balanceChanges txDiff) (txSnapshot, error) {
	// Modify asset.
	change := &assetBurnChange{
		diff: int64(tx.Amount),
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForBurnTx(tx.AssetID, *change, balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performBurnWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to BurnWithSig transaction")
	}
	return tp.performBurn(&tx.Burn, info, balanceChanges)
}

func (tp *transactionPerformer) performBurnWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to BurnWithProofs transaction")
	}
	return tp.performBurn(&tx.Burn, info, balanceChanges)
}

func (tp *transactionPerformer) performExchange(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to Exchange transaction")
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "no sell order")
	}
	buyOrder, err := tx.GetBuyOrder()
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "no buy order")
	}
	volume := tx.GetAmount()
	sellFee := tx.GetSellMatcherFee()
	buyFee := tx.GetBuyMatcherFee()

	// snapshot must be generated before the state with orders is changed
	snapshot, err := tp.snapshotGenerator.generateSnapshotForExchangeTx(sellOrder,
		sellFee, buyOrder, buyFee, volume, balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performLease(tx *proto.Lease, txID crypto.Digest, info *performerInfo,
	balanceChanges txDiff) (txSnapshot, error) {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txSnapshot{}, err
	}
	var recipientAddr proto.WavesAddress
	if addr := tx.Recipient.Address(); addr == nil {
		recipientAddr, err = tp.stor.aliases.newestAddrByAlias(tx.Recipient.Alias().Alias)
		if err != nil {
			return txSnapshot{}, errors.Errorf("invalid alias: %v", err)
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
		Status:    proto.LeaseActive,
	}
	snapshot, err := tp.snapshotGenerator.generateSnapshotForLeaseTx(*l, txID, txID, balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performLeaseWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	return tp.performLease(&tx.Lease, *tx.ID, info, balanceChanges)
}

func (tp *transactionPerformer) performLeaseWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	return tp.performLease(&tx.Lease, *tx.ID, info, balanceChanges)
}

func (tp *transactionPerformer) performLeaseCancel(tx *proto.LeaseCancel, txID *crypto.Digest, info *performerInfo,
	balanceChanges txDiff) (txSnapshot, error) {
	oldLease, err := tp.stor.leases.newestLeasingInfo(tx.LeaseID)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to receiver leasing info")
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForLeaseCancelTx(txID,
		*oldLease, tx.LeaseID, *oldLease.OriginTransactionID,
		info.height, balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performLeaseCancelWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseCancelWithSig transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, tx.ID, info, balanceChanges)
}

func (tp *transactionPerformer) performLeaseCancelWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseCancelWithProofs transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, tx.ID, info, balanceChanges)
}

func (tp *transactionPerformer) performCreateAlias(tx *proto.CreateAlias,
	_ *performerInfo, balanceChanges txDiff) (txSnapshot, error) {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txSnapshot{}, err
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForCreateAliasTx(senderAddr, tx.Alias, balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performCreateAliasWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithSig transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info, balanceChanges)
}

func (tp *transactionPerformer) performCreateAliasWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info, balanceChanges)
}

func (tp *transactionPerformer) performMassTransferWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.MassTransferWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	snapshot, err := tp.snapshotGenerator.generateSnapshotForMassTransferTx(balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performDataWithProofs(transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.DataWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to DataWithProofs transaction")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txSnapshot{}, err
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForDataTx(senderAddr, tx.Entries, balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performSponsorshipWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForSponsorshipTx(tx.AssetID, tx.MinAssetFee, balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performSetScriptWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.SetScriptWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to SetScriptWithProofs transaction")
	}

	se := info.checkerData.scriptEstimation
	if !se.isPresent() {
		return txSnapshot{}, errors.New("script estimations must be set for SetScriptWithProofs tx")
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForSetScriptTx(tx.SenderPK,
		tx.Script, *se, balanceChanges)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to generate snapshot for set script tx")
	}

	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performSetAssetScriptWithProofs(transaction proto.Transaction,
	info *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.SetAssetScriptWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to SetAssetScriptWithProofs transaction")
	}

	se := info.checkerData.scriptEstimation
	if !se.isPresent() {
		return txSnapshot{}, errors.New("script estimations must be set for SetAssetScriptWithProofs tx")
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForSetAssetScriptTx(tx.AssetID, tx.Script, balanceChanges, *se)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performInvokeScriptWithProofs(transaction proto.Transaction,
	info *performerInfo,
	invocationRes *invocationResult,
	balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.InvokeScriptWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}

	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return txSnapshot{}, errors.Wrapf(err, "failed to commit invoke changes for tx %q", tx.ID.String())
	}
	se := info.checkerData.scriptEstimation
	if !se.isPresent() { // nothing to do, no estimation to save
		return txSnapshot{}, nil
	}
	// script estimation is present an not nil

	// we've pulled up an old script which estimation had been done by an old estimator
	// in txChecker we've estimated script with a new estimator
	// this is the place where we have to store new estimation
	scriptAddr, err := recipientToAddress(tx.ScriptRecipient, tp.stor.aliases)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to get sender for InvokeScriptWithProofs")
	}
	// update callable and summary complexity, verifier complexity remains the same
	if scErr := tp.stor.scriptsComplexity.updateCallableComplexitiesForAddr(scriptAddr, *se, info.blockID); scErr != nil {
		return txSnapshot{}, errors.Wrapf(scErr, "failed to save complexity for addr %q in tx %q",
			scriptAddr.String(), tx.ID.String(),
		)
	}
	// TODO: ignored
	_ = invocationRes
	_ = balanceChanges
	return txSnapshot{}, nil
}

func (tp *transactionPerformer) performInvokeExpressionWithProofs(transaction proto.Transaction,
	info *performerInfo, invocationRes *invocationResult,
	balanceChanges txDiff) (txSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs); !ok {
		return txSnapshot{}, errors.New("failed to convert interface to InvokeExpressionWithProofs transaction")
	}
	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to commit invoke changes")
	}
	// TODO: ignored
	_ = invocationRes
	_ = balanceChanges
	return txSnapshot{}, nil
}

func (tp *transactionPerformer) performEthereumTransactionWithProofs(transaction proto.Transaction, info *performerInfo,
	invocationRes *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	ethTx, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to EthereumTransaction transaction")
	}

	if _, ok := ethTx.TxKind.(*proto.EthereumInvokeScriptTxKind); ok {
		if err := tp.stor.commitUncertain(info.blockID); err != nil {
			return txSnapshot{}, errors.Wrap(err, "failed to commit invoke changes")
		}
	}
	// TODO: ignored
	_ = invocationRes
	_ = balanceChanges
	// nothing to do for proto.EthereumTransferWavesTxKind and proto.EthereumTransferAssetsErc20TxKind
	return txSnapshot{}, nil
}

func (tp *transactionPerformer) performUpdateAssetInfoWithProofs(transaction proto.Transaction,
	info *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.UpdateAssetInfoWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to UpdateAssetInfoWithProofs transaction")
	}
	blockHeight := info.height + 1

	snapshot, err := tp.snapshotGenerator.generateSnapshotForUpdateAssetInfoTx(tx.AssetID,
		tx.Name, tx.Description, blockHeight, balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}
