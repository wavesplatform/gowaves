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
	checkerData         txCheckerData
}

func newPerformerInfo(height proto.Height, stateActionsCounter *proto.StateActionsCounter,
	blockID proto.BlockID, currentMinerAddress proto.WavesAddress,
	checkerData txCheckerData) *performerInfo {
	return &performerInfo{height, blockID,
		currentMinerAddress, stateActionsCounter,
		checkerData} // all fields must be initialized
}

type transactionPerformer struct {
	stor              *blockchainEntitiesStorage
	settings          *settings.BlockchainSettings
	snapshotGenerator *snapshotGenerator // initialized in appendTx
	snapshotApplier   SnapshotApplier    // initialized in appendTx
}

func newTransactionPerformer(stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) (*transactionPerformer, error) {
	return &transactionPerformer{stor: stor, settings: settings}, nil
}

func (tp *transactionPerformer) performGenesis(
	transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult,
	balanceChanges txDiff) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.Genesis)
	if !ok {
		return nil, errors.New("failed to convert interface to genesis transaction")
	}
	snapshot, err := tp.snapshotGenerator.generateSnapshotForGenesisTx(balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performPayment(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.Payment)
	if !ok {
		return nil, errors.New("failed to convert interface to payment transaction")
	}
	snapshot, err := tp.snapshotGenerator.generateSnapshotForPaymentTx(balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performTransfer(balanceChanges txDiff) (TransactionSnapshot, error) {
	snapshot, err := tp.snapshotGenerator.generateSnapshotForTransferTx(balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performTransferWithSig(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to transfer with sig transaction")
	}
	return tp.performTransfer(balanceChanges)
}

func (tp *transactionPerformer) performTransferWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to transfer with proofs transaction")
	}
	return tp.performTransfer(balanceChanges)
}

func (tp *transactionPerformer) performIssue(tx *proto.Issue, txID crypto.Digest,
	assetID crypto.Digest, info *performerInfo,
	balanceChanges txDiff, scriptInformation *scriptInformation) (TransactionSnapshot, error) {
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
		*assetInfo, balanceChanges, scriptInformation)

	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performIssueWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	txID, err := tx.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return nil, err
	}
	return tp.performIssue(&tx.Issue, assetID, assetID, info, balanceChanges, nil)
}

func (tp *transactionPerformer) performIssueWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithProofs transaction")
	}
	txID, err := tx.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return nil, err
	}
	var se *scriptEstimation
	var scriptInfo *scriptInformation
	if se = info.checkerData.scriptEstimation; se.isPresent() { // script estimation is present and not nil
		scriptInfo = &scriptInformation{
			script:     tx.Script,
			complexity: se.estimation.Verifier,
		}
	}

	return tp.performIssue(&tx.Issue, assetID, assetID, info, balanceChanges, scriptInfo)
}

func (tp *transactionPerformer) performReissue(tx *proto.Reissue, _ *performerInfo,
	balanceChanges txDiff) (TransactionSnapshot, error) {
	// Modify asset.
	change := &assetReissueChange{
		reissuable: tx.Reissuable,
		diff:       int64(tx.Quantity),
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForReissueTx(tx.AssetID, *change, balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performReissueWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to ReissueWithSig transaction")
	}
	return tp.performReissue(&tx.Reissue, info, balanceChanges)
}

func (tp *transactionPerformer) performReissueWithProofs(transaction proto.Transaction,
	info *performerInfo, _ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to ReissueWithProofs transaction")
	}
	return tp.performReissue(&tx.Reissue, info, balanceChanges)
}

func (tp *transactionPerformer) performBurn(tx *proto.Burn, _ *performerInfo,
	balanceChanges txDiff) (TransactionSnapshot, error) {
	// Modify asset.
	change := &assetBurnChange{
		diff: int64(tx.Amount),
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForBurnTx(tx.AssetID, *change, balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performBurnWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to BurnWithSig transaction")
	}
	return tp.performBurn(&tx.Burn, info, balanceChanges)
}

func (tp *transactionPerformer) performBurnWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to BurnWithProofs transaction")
	}
	return tp.performBurn(&tx.Burn, info, balanceChanges)
}

func (tp *transactionPerformer) performExchange(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
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
	snapshot, err := tp.snapshotGenerator.generateSnapshotForExchangeTx(sellOrder,
		sellFee, buyOrder, buyFee, volume, balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performLease(tx *proto.Lease, txID crypto.Digest, info *performerInfo,
	balanceChanges txDiff) (TransactionSnapshot, error) {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}
	var recipientAddr proto.WavesAddress
	if addr := tx.Recipient.Address(); addr == nil {
		recipientAddr, err = tp.stor.aliases.newestAddrByAlias(tx.Recipient.Alias().Alias)
		if err != nil {
			return nil, errors.Errorf("invalid alias: %v", err)
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
	snapshot, err := tp.snapshotGenerator.generateSnapshotForLeaseTx(*l, txID, txID, balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performLeaseWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	return tp.performLease(&tx.Lease, *tx.ID, info, balanceChanges)
}

func (tp *transactionPerformer) performLeaseWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	return tp.performLease(&tx.Lease, *tx.ID, info, balanceChanges)
}

func (tp *transactionPerformer) performLeaseCancel(tx *proto.LeaseCancel, txID *crypto.Digest, info *performerInfo,
	balanceChanges txDiff) (TransactionSnapshot, error) {
	oldLease, err := tp.stor.leases.newestLeasingInfo(tx.LeaseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receiver leasing info")
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForLeaseCancelTx(txID,
		*oldLease, tx.LeaseID, *oldLease.OriginTransactionID,
		info.height, balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performLeaseCancelWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseCancelWithSig transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, tx.ID, info, balanceChanges)
}

func (tp *transactionPerformer) performLeaseCancelWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseCancelWithProofs transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, tx.ID, info, balanceChanges)
}

func (tp *transactionPerformer) performCreateAlias(tx *proto.CreateAlias,
	_ *performerInfo, balanceChanges txDiff) (TransactionSnapshot, error) {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForCreateAliasTx(senderAddr, tx.Alias, balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performCreateAliasWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasWithSig transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info, balanceChanges)
}

func (tp *transactionPerformer) performCreateAliasWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info, balanceChanges)
}

func (tp *transactionPerformer) performMassTransferWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.MassTransferWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	snapshot, err := tp.snapshotGenerator.generateSnapshotForMassTransferTx(balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performDataWithProofs(transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.DataWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to DataWithProofs transaction")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForDataTx(senderAddr, tx.Entries, balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performSponsorshipWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForSponsorshipTx(tx.AssetID, tx.MinAssetFee, balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performSetScriptWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SetScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SetScriptWithProofs transaction")
	}

	se := info.checkerData.scriptEstimation
	if !se.isPresent() {
		return nil, errors.New("script estimations must be set for SetScriptWithProofs tx")
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForSetScriptTx(tx.SenderPK,
		tx.Script, *se, balanceChanges)

	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performSetAssetScriptWithProofs(transaction proto.Transaction,
	info *performerInfo, _ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SetAssetScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SetAssetScriptWithProofs transaction")
	}

	se := info.checkerData.scriptEstimation
	if !se.isPresent() {
		return nil, errors.New("script estimations must be set for SetAssetScriptWithProofs tx")
	}
	snapshot, err := tp.snapshotGenerator.generateSnapshotForSetAssetScriptTx(tx.AssetID,
		tx.Script, se.estimation.Verifier, tx.SenderPK, balanceChanges)
	if err != nil {
		return nil, err
	}

	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performInvokeScriptWithProofs(transaction proto.Transaction,
	info *performerInfo,
	invocationRes *invocationResult,
	balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.InvokeScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}

	se := info.checkerData.scriptEstimation
	if se.isPresent() {
		// script estimation is present an not nil

		// we've pulled up an old script which estimation had been done by an old estimator
		// in txChecker we've estimated script with a new estimator
		// this is the place where we have to store new estimation
		scriptAddr, err := recipientToAddress(tx.ScriptRecipient, tp.stor.aliases)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get sender for InvokeScriptWithProofs")
		}
		// update callable and summary complexity, verifier complexity remains the same
		// TODO this might a problem in the future with importing snapshots,
		// TODO because snapshots don't contain the information about the callables
		if scErr := tp.stor.scriptsComplexity.updateCallableComplexitiesForAddr(scriptAddr, *se, info.blockID); scErr != nil {
			return nil, errors.Wrapf(scErr, "failed to save complexity for addr %q in tx %q",
				scriptAddr.String(), tx.ID.String(),
			)
		}
	}

	txIDBytes, err := transaction.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v", err)
	}
	txID, err := crypto.NewDigestFromBytes(txIDBytes)
	if err != nil {
		return nil, err
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForInvokeScriptTx(txID, info,
		invocationRes, balanceChanges, tx.SenderPK, se, &tx.ScriptRecipient)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performInvokeExpressionWithProofs(transaction proto.Transaction,
	info *performerInfo, invocationRes *invocationResult,
	balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to InvokeExpressionWithProofs transaction")
	}

	txIDBytes, err := transaction.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v", err)
	}
	txID, err := crypto.NewDigestFromBytes(txIDBytes)
	if err != nil {
		return nil, err
	}

	snapshot, err := tp.snapshotGenerator.generateSnapshotForInvokeExpressionTx(txID, info, invocationRes,
		balanceChanges, tx.SenderPK)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performEthereumTransactionWithProofs(transaction proto.Transaction, info *performerInfo,
	invocationRes *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	ethTx, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return nil, errors.New("failed to convert interface to EthereumTransaction transaction")
	}

	txIDBytes, err := transaction.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v", err)
	}
	txID, err := crypto.NewDigestFromBytes(txIDBytes)
	if err != nil {
		return nil, err
	}
	scriptAddr, err := ethTx.WavesAddressTo(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, err
	}
	var si scriptBasicInfoRecord
	si, err = tp.stor.scriptsStorage.newestScriptBasicInfoByAddressID(scriptAddr.ID())
	if err != nil {
		return nil,
			errors.Wrapf(err, "failed to get script's public key on address '%s'", scriptAddr.String())
	}
	scriptPK := si.PK
	snapshot, err := tp.snapshotGenerator.generateSnapshotForEthereumInvokeScriptTx(txID,
		info, invocationRes, balanceChanges, scriptPK)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}

func (tp *transactionPerformer) performUpdateAssetInfoWithProofs(transaction proto.Transaction,
	info *performerInfo, _ *invocationResult, balanceChanges txDiff) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.UpdateAssetInfoWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to UpdateAssetInfoWithProofs transaction")
	}
	blockHeight := info.height + 1

	snapshot, err := tp.snapshotGenerator.generateSnapshotForUpdateAssetInfoTx(tx.AssetID,
		tx.Name, tx.Description, blockHeight, balanceChanges)
	if err != nil {
		return nil, err
	}
	return snapshot, snapshot.Apply(tp.snapshotApplier)
}
