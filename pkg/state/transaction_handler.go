package state

import (
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type txCheckerData struct {
	_                struct{}
	smartAssets      []crypto.Digest
	scriptEstimation *scriptEstimation
}

type scriptEstimation struct {
	currentEstimatorVersion int
	scriptIsEmpty           bool
	estimation              ride.TreeEstimation
}

func (e *scriptEstimation) isPresent() bool { return e != nil }

type txCheckFunc func(proto.Transaction, *checkerInfo) (txCheckerData, error)
type txPerformFunc func(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
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
	tp transactionPerformer
	td *transactionDiffer
	tf *transactionFeeCounter

	sa extendedSnapshotApplier

	funcs handles
}

// TODO: see TODO on GetTypeInfo() in proto/transactions.go.
func buildHandles( //nolint:funlen
	tc *transactionChecker,
	tp transactionPerformer,
	td *transactionDiffer,
	tf *transactionFeeCounter,
) handles {
	return handles{
		proto.TransactionTypeInfo{Type: proto.GenesisTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkGenesis, tp.performGenesis,
			td.createDiffGenesis, nil,
		},
		proto.TransactionTypeInfo{Type: proto.PaymentTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkPayment, tp.performPayment,
			td.createDiffPayment, tf.minerFeePayment,
		},
		proto.TransactionTypeInfo{Type: proto.TransferTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkTransferWithSig, tp.performTransferWithSig,
			td.createDiffTransferWithSig, tf.minerFeeTransferWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.TransferTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkTransferWithProofs, tp.performTransferWithProofs,
			td.createDiffTransferWithProofs, tf.minerFeeTransferWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.IssueTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkIssueWithSig, tp.performIssueWithSig,
			td.createDiffIssueWithSig, tf.minerFeeIssueWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.IssueTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkIssueWithProofs, tp.performIssueWithProofs,
			td.createDiffIssueWithProofs, tf.minerFeeIssueWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.ReissueTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkReissueWithSig, tp.performReissueWithSig,
			td.createDiffReissueWithSig, tf.minerFeeReissueWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.ReissueTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkReissueWithProofs, tp.performReissueWithProofs,
			td.createDiffReissueWithProofs, tf.minerFeeReissueWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.BurnTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkBurnWithSig, tp.performBurnWithSig,
			td.createDiffBurnWithSig, tf.minerFeeBurnWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.BurnTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkBurnWithProofs, tp.performBurnWithProofs,
			td.createDiffBurnWithProofs, tf.minerFeeBurnWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.ExchangeTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkExchangeWithSig, tp.performExchange,
			td.createDiffExchange, tf.minerFeeExchange,
		},
		proto.TransactionTypeInfo{Type: proto.ExchangeTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkExchangeWithProofs, tp.performExchange,
			td.createDiffExchange, tf.minerFeeExchange,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkLeaseWithSig, tp.performLeaseWithSig,
			td.createDiffLeaseWithSig, tf.minerFeeLeaseWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkLeaseWithProofs, tp.performLeaseWithProofs,
			td.createDiffLeaseWithProofs, tf.minerFeeLeaseWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseCancelTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkLeaseCancelWithSig, tp.performLeaseCancelWithSig,
			td.createDiffLeaseCancelWithSig, tf.minerFeeLeaseCancelWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.LeaseCancelTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkLeaseCancelWithProofs, tp.performLeaseCancelWithProofs,
			td.createDiffLeaseCancelWithProofs, tf.minerFeeLeaseCancelWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.CreateAliasTransaction, ProofVersion: proto.Signature}: txHandleFuncs{
			tc.checkCreateAliasWithSig, tp.performCreateAliasWithSig,
			td.createDiffCreateAliasWithSig, tf.minerFeeCreateAliasWithSig,
		},
		proto.TransactionTypeInfo{Type: proto.CreateAliasTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkCreateAliasWithProofs, tp.performCreateAliasWithProofs,
			td.createDiffCreateAliasWithProofs, tf.minerFeeCreateAliasWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.MassTransferTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkMassTransferWithProofs, tp.performMassTransferWithProofs,
			td.createDiffMassTransferWithProofs, tf.minerFeeMassTransferWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.DataTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkDataWithProofs, tp.performDataWithProofs,
			td.createDiffDataWithProofs, tf.minerFeeDataWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.SponsorshipTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkSponsorshipWithProofs, tp.performSponsorshipWithProofs,
			td.createDiffSponsorshipWithProofs, tf.minerFeeSponsorshipWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.SetScriptTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkSetScriptWithProofs, tp.performSetScriptWithProofs,
			td.createDiffSetScriptWithProofs, tf.minerFeeSetScriptWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.SetAssetScriptTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkSetAssetScriptWithProofs, tp.performSetAssetScriptWithProofs,
			td.createDiffSetAssetScriptWithProofs, tf.minerFeeSetAssetScriptWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.InvokeScriptTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkInvokeScriptWithProofs, tp.performInvokeScriptWithProofs,
			td.createDiffInvokeScriptWithProofs, tf.minerFeeInvokeScriptWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.InvokeExpressionTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkInvokeExpressionWithProofs, tp.performInvokeExpressionWithProofs,
			td.createDiffInvokeExpressionWithProofs, tf.minerFeeInvokeExpressionWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.UpdateAssetInfoTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkUpdateAssetInfoWithProofs, tp.performUpdateAssetInfoWithProofs,
			td.createDiffUpdateAssetInfoWithProofs, tf.minerFeeUpdateAssetInfoWithProofs,
		},
		proto.TransactionTypeInfo{Type: proto.EthereumMetamaskTransaction, ProofVersion: proto.Proof}: txHandleFuncs{
			tc.checkEthereumTransactionWithProofs, tp.performEthereumTransactionWithProofs,
			td.createDiffEthereumTransactionWithProofs, tf.minerFeeEthereumTxWithProofs,
		},
	}
}

func newTransactionHandler(
	genesis proto.BlockID,
	stor *blockchainEntitiesStorage,
	settings *settings.BlockchainSettings,
	snapshotApplier extendedSnapshotApplier,
) (*transactionHandler, error) {
	tc, err := newTransactionChecker(genesis, stor, settings)
	if err != nil {
		return nil, err
	}
	sg := newSnapshotGenerator(stor, settings.AddressSchemeCharacter)
	td, err := newTransactionDiffer(stor, settings)
	if err != nil {
		return nil, err
	}
	tf, err := newTransactionFeeCounter(stor)
	if err != nil {
		return nil, err
	}
	return &transactionHandler{
		tc:    tc,
		tp:    sg,
		td:    td,
		tf:    tf,
		sa:    snapshotApplier,
		funcs: buildHandles(tc, sg, td, tf),
	}, nil
}

func (h *transactionHandler) checkTx(tx proto.Transaction, info *checkerInfo) (txCheckerData, error) {
	tv := tx.GetTypeInfo()
	funcs, ok := h.funcs[tv]
	if !ok {
		return txCheckerData{}, errors.Errorf("No function handler implemented for tx struct type %T\n", tx)
	}
	if funcs.check == nil {
		// No check func for this combination of transaction type and version.
		return txCheckerData{}, nil
	}
	return funcs.check(tx, info)
}

func (h *transactionHandler) performTx(
	tx proto.Transaction,
	info *performerInfo,
	validatingUTX bool,
	invocationRes *invocationResult,
	applicationStatus bool,
	balanceChanges txDiff,
) (txSnapshot, error) {
	tv := tx.GetTypeInfo()
	funcs, ok := h.funcs[tv]
	if !ok {
		return txSnapshot{}, errors.Errorf("no function handler implemented for tx struct type %T", tx)
	}
	if funcs.perform == nil {
		// performer function must not be nil
		return txSnapshot{}, errors.Errorf("performer function handler is nil for tx struct type %T", tx)
	}
	var snapshot txSnapshot
	if applicationStatus {
		var err error
		snapshot, err = funcs.perform(tx, info, invocationRes, balanceChanges)
		if err != nil {
			return txSnapshot{}, errors.Wrapf(err, "failed to perform and generate snapshots for tx %q", tx)
		}
		snapshot.regular = append(snapshot.regular,
			&proto.TransactionStatusSnapshot{
				Status: proto.TransactionSucceeded,
			},
		)
	} else {
		// TODO: generate balance atomic snapshots here for failed transactions
		snapshot = txSnapshot{
			regular: []proto.AtomicSnapshot{
				&proto.TransactionStatusSnapshot{Status: proto.TransactionFailed},
			},
			internal: nil,
		}
	}
	if err := snapshot.Apply(h.sa, tx, validatingUTX); err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to apply transaction snapshot")
	}
	return snapshot, nil
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
