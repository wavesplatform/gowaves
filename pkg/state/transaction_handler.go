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
type txCountFeeFunc func(proto.Transaction, *feeDistribution) error

type txHandleFuncs struct {
	check      txCheckFunc
	perform    txPerformFunc
	createDiff txCreateDiffFunc
	countFee   txCountFeeFunc
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
			tc.checkGenesis, nil, td.createDiffGenesis, nil,
		},
		proto.TransactionTypeInfo{Type: proto.PaymentTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkPayment, nil, td.createDiffPayment, tf.minerFeePayment,
		},
		proto.TransactionTypeInfo{Type: proto.TransferTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkTransferWithSig, nil, td.createDiffTransferWithSig, tf.minerFeeTransferWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.TransferTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkTransferWithProofs, nil, td.createDiffTransferWithProofs, tf.minerFeeTransferWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.IssueTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkIssueWithSig, tp.performIssueWithSig, td.createDiffIssueWithSig, tf.minerFeeIssueWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.IssueTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkIssueWithProofs, tp.performIssueWithProofs, td.createDiffIssueWithProofs, tf.minerFeeIssueWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.ReissueTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkReissueWithSig, tp.performReissueWithSig, td.createDiffReissueWithSig, tf.minerFeeReissueWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.ReissueTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkReissueWithProofs, tp.performReissueWithProofs, td.createDiffReissueWithProofs, tf.minerFeeReissueWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.BurnTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkBurnWithSig, tp.performBurnWithSig, td.createDiffBurnWithSig, tf.minerFeeBurnWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.BurnTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkBurnWithProofs, tp.performBurnWithProofs, td.createDiffBurnWithProofs, tf.minerFeeBurnWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.ExchangeTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkExchangeWithSig, tp.performExchange, td.createDiffExchange, tf.minerFeeExchange,
		},
		proto.TransactionTypeInfo{Type: proto.ExchangeTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkExchangeWithProofs, tp.performExchange, td.createDiffExchange, tf.minerFeeExchange,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkLeaseWithSig, tp.performLeaseWithSig, td.createDiffLeaseWithSig, tf.minerFeeLeaseWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkLeaseWithProofs, tp.performLeaseWithProofs, td.createDiffLeaseWithProofs, tf.minerFeeLeaseWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseCancelTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkLeaseCancelWithSig, tp.performLeaseCancelWithSig, td.createDiffLeaseCancelWithSig, tf.minerFeeLeaseCancelWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseCancelTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkLeaseCancelWithProofs, tp.performLeaseCancelWithProofs, td.createDiffLeaseCancelWithProofs, tf.minerFeeLeaseCancelWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.CreateAliasTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkCreateAliasWithSig, tp.performCreateAliasWithSig, td.createDiffCreateAliasWithSig, tf.minerFeeCreateAliasWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.CreateAliasTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkCreateAliasWithProofs, tp.performCreateAliasWithProofs, td.createDiffCreateAliasWithProofs, tf.minerFeeCreateAliasWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.MassTransferTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkMassTransferWithProofs, nil, td.createDiffMassTransferWithProofs, tf.minerFeeMassTransferWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.DataTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkDataWithProofs, tp.performDataWithProofs, td.createDiffDataWithProofs, tf.minerFeeDataWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.SponsorshipTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkSponsorshipWithProofs, tp.performSponsorshipWithProofs, td.createDiffSponsorshipWithProofs, tf.minerFeeSponsorshipWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.SetScriptTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkSetScriptWithProofs, tp.performSetScriptWithProofs, td.createDiffSetScriptWithProofs, tf.minerFeeSetScriptWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.SetAssetScriptTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkSetAssetScriptWithProofs, tp.performSetAssetScriptWithProofs, td.createDiffSetAssetScriptWithProofs, tf.minerFeeSetAssetScriptWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.InvokeScriptTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkInvokeScriptWithProofs, tp.performInvokeScriptWithProofs, td.createDiffInvokeScriptWithProofs, tf.minerFeeInvokeScriptWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.InvokeExpressionTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkInvokeExpressionWithProofs, tp.performInvokeExpressionWithProofs, td.createDiffInvokeExpressionWithProofs, tf.minerFeeInvokeExpressionWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.UpdateAssetInfoTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkUpdateAssetInfoWithProofs, tp.performUpdateAssetInfoWithProofs, td.createDiffUpdateAssetInfoWithProofs, tf.minerFeeUpdateAssetInfoWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.EthereumMetamaskTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkEthereumTransactionWithProofs, tp.performEthereumTransactionWithProofs, td.createDiffEthereumTransactionWithProofs, tf.minerFeeEthereumTxWithProofs,
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
