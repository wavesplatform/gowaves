package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type txCheckFunc func(proto.Transaction, *checkerInfo) ([]crypto.Digest, error)
type txPerformFunc func(proto.Transaction, *performerInfo) error
type txCreateDiffFunc func(proto.Transaction, *differInfo) (txBalanceChanges, error)

//TODO: Consider not using txCreateFeeDiffFunc function but extract 2 special cases for Invoke and Exchange transactions
type txCreateFeeDiffFunc func(proto.Transaction, *differInfo) (txBalanceChanges, error)
type txCountFeeFunc func(proto.Transaction, *feeDistribution) error

type txHandleFuncs struct {
	check         txCheckFunc
	perform       txPerformFunc
	createDiff    txCreateDiffFunc
	createFeeDiff txCreateFeeDiffFunc
	countFee      txCountFeeFunc
}

type handles map[proto.TransactionTypeInfo]txHandleFuncs

type transactionHandler struct {
	tc *transactionChecker
	tp *transactionPerformer
	td *transactionDiffer
	tf *transactionFeeCounter

	funcs handles
}

// TODO: see TODO on GetTypeInfo() in proto/transactions.go.
func buildHandles(tc *transactionChecker, tp *transactionPerformer, td *transactionDiffer, tf *transactionFeeCounter) handles {
	return handles{
		proto.TransactionTypeInfo{Type: proto.GenesisTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkGenesis, nil, td.createDiffGenesis, td.createFeeDiffGenesis, nil,
		},
		proto.TransactionTypeInfo{Type: proto.PaymentTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkPayment, nil, td.createDiffPayment, td.createFeeDiffPayment, tf.minerFeePayment,
		},
		proto.TransactionTypeInfo{Type: proto.TransferTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkTransferWithSig, nil, td.createDiffTransferWithSig, td.createFeeDiffTransferWithSig, tf.minerFeeTransferWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.TransferTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkTransferWithProofs, nil, td.createDiffTransferWithProofs, td.createFeeDiffTransferWithProofs, tf.minerFeeTransferWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.IssueTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkIssueWithSig, tp.performIssueWithSig, td.createDiffIssueWithSig, td.createFeeDiffIssueWithSig, tf.minerFeeIssueWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.IssueTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkIssueWithProofs, tp.performIssueWithProofs, td.createDiffIssueWithProofs, td.createFeeDiffIssueWithProofs, tf.minerFeeIssueWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.ReissueTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkReissueWithSig, tp.performReissueWithSig, td.createDiffReissueWithSig, td.createFeeDiffReissueWithSig, tf.minerFeeReissueWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.ReissueTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkReissueWithProofs, tp.performReissueWithProofs, td.createDiffReissueWithProofs, td.createFeeDiffReissueWithProofs, tf.minerFeeReissueWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.BurnTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkBurnWithSig, tp.performBurnWithSig, td.createDiffBurnWithSig, td.createFeeDiffBurnWithSig, tf.minerFeeBurnWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.BurnTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkBurnWithProofs, tp.performBurnWithProofs, td.createDiffBurnWithProofs, td.createFeeDiffBurnWithProofs, tf.minerFeeBurnWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.ExchangeTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkExchangeWithSig, tp.performExchange, td.createDiffExchange, td.createFeeDiffExchange, tf.minerFeeExchange,
		},
		proto.TransactionTypeInfo{Type: proto.ExchangeTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkExchangeWithProofs, tp.performExchange, td.createDiffExchange, td.createFeeDiffExchange, tf.minerFeeExchange,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkLeaseWithSig, tp.performLeaseWithSig, td.createDiffLeaseWithSig, td.createFeeDiffLeaseWithSig, tf.minerFeeLeaseWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkLeaseWithProofs, tp.performLeaseWithProofs, td.createDiffLeaseWithProofs, td.createFeeDiffLeaseWithProofs, tf.minerFeeLeaseWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseCancelTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkLeaseCancelWithSig, tp.performLeaseCancelWithSig, td.createDiffLeaseCancelWithSig, td.createFeeDiffLeaseCancelWithSig, tf.minerFeeLeaseCancelWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseCancelTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkLeaseCancelWithProofs, tp.performLeaseCancelWithProofs, td.createDiffLeaseCancelWithProofs, td.createFeeDiffLeaseCancelWithProofs, tf.minerFeeLeaseCancelWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.CreateAliasTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkCreateAliasWithSig, tp.performCreateAliasWithSig, td.createDiffCreateAliasWithSig, td.createDiffCreateAliasWithSig, tf.minerFeeCreateAliasWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.CreateAliasTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkCreateAliasWithProofs, tp.performCreateAliasWithProofs, td.createDiffCreateAliasWithProofs, td.createDiffCreateAliasWithProofs, tf.minerFeeCreateAliasWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.MassTransferTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkMassTransferWithProofs, nil, td.createDiffMassTransferWithProofs, td.createFeeDiffMassTransferWithProofs, tf.minerFeeMassTransferWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.DataTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkDataWithProofs, tp.performDataWithProofs, td.createDiffDataWithProofs, td.createDiffDataWithProofs, tf.minerFeeDataWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.SponsorshipTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkSponsorshipWithProofs, tp.performSponsorshipWithProofs, td.createDiffSponsorshipWithProofs, td.createDiffSponsorshipWithProofs, tf.minerFeeSponsorshipWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.SetScriptTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkSetScriptWithProofs, tp.performSetScriptWithProofs, td.createDiffSetScriptWithProofs, td.createDiffSetScriptWithProofs, tf.minerFeeSetScriptWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.SetAssetScriptTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkSetAssetScriptWithProofs, tp.performSetAssetScriptWithProofs, td.createDiffSetAssetScriptWithProofs, td.createDiffSetAssetScriptWithProofs, tf.minerFeeSetAssetScriptWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.InvokeScriptTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkInvokeScriptWithProofs, nil, td.createDiffInvokeScriptWithProofs, td.createFeeDiffInvokeScriptWithProofs, tf.minerFeeInvokeScriptWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.UpdateAssetInfoTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkUpdateAssetInfoWithProofs, tp.performUpdateAssetInfoWithProofs, td.createDiffUpdateAssetInfoWithProofs, td.createDiffUpdateAssetInfoWithProofs, tf.minerFeeUpdateAssetInfoWithProofs,
		},
	}
}

func newTransactionHandler(
	genesis proto.BlockID,
	stor *blockchainEntitiesStorage,
	settings *settings.BlockchainSettings,
) (*transactionHandler, error) {
	tc, err := newTransactionChecker(genesis, stor, settings)
	if err != nil {
		return nil, err
	}
	tp, err := newTransactionPerformer(stor, settings)
	if err != nil {
		return nil, err
	}
	td, err := newTransactionDiffer(stor, settings)
	if err != nil {
		return nil, err
	}
	tf, err := newTransactionFeeCounter(stor)
	if err != nil {
		return nil, err
	}
	return &transactionHandler{tc: tc, tp: tp, td: td, tf: tf, funcs: buildHandles(tc, tp, td, tf)}, nil
}

func (h *transactionHandler) checkTx(tx proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tv := tx.GetTypeInfo()
	funcs, ok := h.funcs[tv]
	if !ok {
		return nil, errors.Errorf("No function handler implemented for tx struct type %T\n", tx)
	}
	if funcs.check == nil {
		// No check func for this combination of transaction type and version.
		return nil, nil
	}
	return funcs.check(tx, info)
}

func (h *transactionHandler) performTx(tx proto.Transaction, info *performerInfo) error {
	tv := tx.GetTypeInfo()
	funcs, ok := h.funcs[tv]
	if !ok {
		return errors.Errorf("No function handler implemented for tx struct type %T\n", tx)
	}
	if funcs.perform == nil {
		// No perform func for this combination of transaction type and version.
		return nil
	}
	return funcs.perform(tx, info)
}

func (h *transactionHandler) createDiffTx(tx proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tv := tx.GetTypeInfo()
	funcs, ok := h.funcs[tv]
	if !ok {
		return txBalanceChanges{}, errors.Errorf("No function handler implemented for tx struct type %T\n", tx)
	}
	if funcs.createDiff == nil {
		// No createDiff func for this combination of transaction type and version.
		return txBalanceChanges{}, nil
	}
	return funcs.createDiff(tx, info)
}

func (h *transactionHandler) createFeeDiffTx(tx proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	ti := tx.GetTypeInfo()
	handlers, ok := h.funcs[ti]
	if !ok {
		return txBalanceChanges{}, errors.Errorf("No function handler implemented for tx struct type %T\n", tx)
	}
	if handlers.createDiff == nil {
		// No createDiff func for this combination of transaction type and version.
		return txBalanceChanges{}, nil
	}
	return handlers.createFeeDiff(tx, info)
}

func (h *transactionHandler) minerFeeTx(tx proto.Transaction, distr *feeDistribution) error {
	tv := tx.GetTypeInfo()
	funcs, ok := h.funcs[tv]
	if !ok {
		return errors.Errorf("No function handler implemented for tx struct type %T\n", tx)
	}
	if funcs.countFee == nil {
		// No countFee func for this combination of transaction type and version.
		return nil
	}
	return funcs.countFee(tx, distr)
}
