package state

import (
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type performerInfo struct {
	blockchainHeight    proto.Height
	blockID             proto.BlockID
	currentMinerAddress proto.WavesAddress
	checkerData         txCheckerData
}

func (i *performerInfo) blockHeight() proto.Height { return i.blockchainHeight + 1 }

func newPerformerInfo(
	blockchainHeight proto.Height,
	blockID proto.BlockID,
	currentMinerAddress proto.WavesAddress,
	checkerData txCheckerData,
) *performerInfo {
	return &performerInfo{ // all fields must be initialized
		blockchainHeight,
		blockID,
		currentMinerAddress,
		checkerData,
	}
}

// transactionPerformer ONLY generates tx snapshots
// TODO: rename to snapshot generator
type transactionPerformer struct {
	stor        *blockchainEntitiesStorage
	scheme      proto.Scheme
	internalGen *internalSnapshotGenerator // TODO: remove this field
}

func newTransactionPerformer(
	stor *blockchainEntitiesStorage,
	scheme proto.Scheme,
) *transactionPerformer {
	return &transactionPerformer{
		stor:        stor,
		scheme:      scheme,
		internalGen: newInternalSnapshotGenerator(stor, scheme),
	}
}

func (tp *transactionPerformer) performGenesis(
	transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult,
	balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.Genesis)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to genesis transaction")
	}
	return tp.internalGen.generateSnapshotForGenesisTx(balanceChanges)
}

func (tp *transactionPerformer) performPayment(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.Payment)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to payment transaction")
	}
	return tp.internalGen.generateSnapshotForPaymentTx(balanceChanges)
}

func (tp *transactionPerformer) performTransferWithSig(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to transfer with sig transaction")
	}
	return tp.internalGen.generateSnapshotForTransferTx(balanceChanges)
}

func (tp *transactionPerformer) performTransferWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to transfer with proofs transaction")
	}
	return tp.internalGen.generateSnapshotForTransferTx(balanceChanges)
}

func (tp *transactionPerformer) performIssueWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	txID, err := tx.GetID(tp.scheme)
	if err != nil {
		return txSnapshot{}, errors.Errorf("failed to get transaction ID: %v", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return txSnapshot{}, err
	}
	return tp.internalGen.generateSnapshotForIssueTx(&tx.Issue, assetID, info, balanceChanges, nil, nil)
}

func (tp *transactionPerformer) performIssueWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to IssueWithProofs transaction")
	}
	txID, err := tx.GetID(tp.scheme)
	if err != nil {
		return txSnapshot{}, errors.Errorf("failed to get transaction ID: %v", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return txSnapshot{}, err
	}
	se := info.checkerData.scriptEstimation
	return tp.internalGen.generateSnapshotForIssueTx(&tx.Issue, assetID, info, balanceChanges, se, tx.Script)
}

func (tp *transactionPerformer) performReissueWithSig(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to ReissueWithSig transaction")
	}
	return tp.internalGen.generateSnapshotForReissueTx(tx.AssetID, tx.Reissuable, tx.Quantity, balanceChanges)
}

func (tp *transactionPerformer) performReissueWithProofs(transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to ReissueWithProofs transaction")
	}
	return tp.internalGen.generateSnapshotForReissueTx(tx.AssetID, tx.Reissuable, tx.Quantity, balanceChanges)
}

func (tp *transactionPerformer) performBurnWithSig(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to BurnWithSig transaction")
	}
	return tp.internalGen.generateSnapshotForBurnTx(tx.AssetID, tx.Amount, balanceChanges)
}

func (tp *transactionPerformer) performBurnWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to BurnWithProofs transaction")
	}
	return tp.internalGen.generateSnapshotForBurnTx(tx.AssetID, tx.Amount, balanceChanges)
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
	return tp.internalGen.generateSnapshotForExchangeTx(sellOrder, sellFee, buyOrder, buyFee, volume, balanceChanges)
}

func (tp *transactionPerformer) performLeaseWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	return tp.internalGen.generateSnapshotForLeaseTx(&tx.Lease, tx.ID, info, balanceChanges)
}

func (tp *transactionPerformer) performLeaseWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	return tp.internalGen.generateSnapshotForLeaseTx(&tx.Lease, tx.ID, info, balanceChanges)
}

func (tp *transactionPerformer) performLeaseCancelWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseCancelWithSig transaction")
	}
	return tp.internalGen.generateSnapshotForLeaseCancelTx(
		tx.LeaseCancel.LeaseID,
		tx.ID,
		info.blockHeight(),
		balanceChanges,
	)
}

func (tp *transactionPerformer) performLeaseCancelWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseCancelWithProofs transaction")
	}
	return tp.internalGen.generateSnapshotForLeaseCancelTx(
		tx.LeaseCancel.LeaseID,
		tx.ID,
		info.blockHeight(),
		balanceChanges,
	)
}

func (tp *transactionPerformer) performCreateAliasWithSig(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithSig transaction")
	}
	return tp.internalGen.generateSnapshotForCreateAliasTx(tp.scheme, tx.SenderPK, tx.Alias, balanceChanges)
}

func (tp *transactionPerformer) performCreateAliasWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return tp.internalGen.generateSnapshotForCreateAliasTx(tp.scheme, tx.SenderPK, tx.Alias, balanceChanges)
}

func (tp *transactionPerformer) performMassTransferWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.MassTransferWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return tp.internalGen.generateSnapshotForMassTransferTx(balanceChanges)
}

func (tp *transactionPerformer) performDataWithProofs(transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.DataWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to DataWithProofs transaction")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tp.scheme, tx.SenderPK)
	if err != nil {
		return txSnapshot{}, err
	}
	return tp.internalGen.generateSnapshotForDataTx(senderAddr, tx.Entries, balanceChanges)
}

func (tp *transactionPerformer) performSponsorshipWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}
	return tp.internalGen.generateSnapshotForSponsorshipTx(tx.AssetID, tx.MinAssetFee, balanceChanges)
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

	snapshot, err := tp.internalGen.generateSnapshotForSetScriptTx(tx.SenderPK, tx.Script, *se, balanceChanges)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to generate snapshot for set script tx")
	}
	return snapshot, nil
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

	snapshot, err := tp.internalGen.generateSnapshotForSetAssetScriptTx(tx.AssetID, tx.Script, balanceChanges, *se)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to generate snapshot for set asset script tx")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performInvokeScriptWithProofs(transaction proto.Transaction,
	info *performerInfo,
	_ *invocationResult,
	balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.InvokeScriptWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	se := info.checkerData.scriptEstimation
	return tp.internalGen.generateSnapshotForInvokeScript(tx.ScriptRecipient, balanceChanges, se)
}

func (tp *transactionPerformer) performInvokeExpressionWithProofs(transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult,
	balanceChanges txDiff) (txSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs); !ok {
		return txSnapshot{}, errors.New("failed to convert interface to InvokeExpressionWithProofs transaction")
	}
	return tp.internalGen.generateSnapshotForInvokeExpressionTx(balanceChanges)
}

func (tp *transactionPerformer) performEthereumTransactionWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to EthereumTransaction transaction")
	}
	return tp.internalGen.generateSnapshotForEthereumInvokeScriptTx(balanceChanges)
}

func (tp *transactionPerformer) performUpdateAssetInfoWithProofs(transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.UpdateAssetInfoWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to UpdateAssetInfoWithProofs transaction")
	}
	return tp.internalGen.generateSnapshotForUpdateAssetInfoTx(
		tx.AssetID,
		tx.Name,
		tx.Description,
		balanceChanges,
	)
}
