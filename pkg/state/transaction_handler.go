package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type txCheckFunc func(proto.Transaction, *checkerInfo) ([]crypto.Digest, error)
type txPerformFunc func(proto.Transaction, *performerInfo) error
type txCreateDiffFunc func(proto.Transaction, *differInfo) (txDiff, error)
type txCountFeeFunc func(proto.Transaction, *feeDistribution) error

type txHandleFuncs struct {
	check      txCheckFunc
	perform    txPerformFunc
	createDiff txCreateDiffFunc
	countFee   txCountFeeFunc
}

type handles map[proto.TransactionTypeVersion]txHandleFuncs

type transactionHandler struct {
	tc *transactionChecker
	tp *transactionPerformer
	td *transactionDiffer
	tf *transactionFeeCounter

	funcs handles
}

func buildHandles(tc *transactionChecker, tp *transactionPerformer, td *transactionDiffer, tf *transactionFeeCounter) handles {
	return handles{
		proto.TransactionTypeVersion{Type: proto.GenesisTransaction, Version: 1}: txHandleFuncs{
			tc.checkGenesis, nil, td.createDiffGenesis, nil,
		},
		proto.TransactionTypeVersion{Type: proto.PaymentTransaction, Version: 1}: txHandleFuncs{
			tc.checkPayment, nil, td.createDiffPayment, tf.minerFeePayment,
		},
		proto.TransactionTypeVersion{Type: proto.TransferTransaction, Version: 1}: txHandleFuncs{
			tc.checkTransferV1, nil, td.createDiffTransferV1, tf.minerFeeTransferV1,
		},
		proto.TransactionTypeVersion{Type: proto.TransferTransaction, Version: 2}: txHandleFuncs{
			tc.checkTransferV2, nil, td.createDiffTransferV2, tf.minerFeeTransferV2,
		},
		proto.TransactionTypeVersion{Type: proto.IssueTransaction, Version: 1}: txHandleFuncs{
			tc.checkIssueV1, tp.performIssueV1, td.createDiffIssueV1, tf.minerFeeIssueV1,
		},
		proto.TransactionTypeVersion{Type: proto.IssueTransaction, Version: 2}: txHandleFuncs{
			tc.checkIssueV2, tp.performIssueV2, td.createDiffIssueV2, tf.minerFeeIssueV2,
		},
		proto.TransactionTypeVersion{Type: proto.ReissueTransaction, Version: 1}: txHandleFuncs{
			tc.checkReissueV1, tp.performReissueV1, td.createDiffReissueV1, tf.minerFeeReissueV1,
		},
		proto.TransactionTypeVersion{Type: proto.ReissueTransaction, Version: 2}: txHandleFuncs{
			tc.checkReissueV2, tp.performReissueV2, td.createDiffReissueV2, tf.minerFeeReissueV2,
		},
		proto.TransactionTypeVersion{Type: proto.BurnTransaction, Version: 1}: txHandleFuncs{
			tc.checkBurnV1, tp.performBurnV1, td.createDiffBurnV1, tf.minerFeeBurnV1,
		},
		proto.TransactionTypeVersion{Type: proto.BurnTransaction, Version: 2}: txHandleFuncs{
			tc.checkBurnV2, tp.performBurnV2, td.createDiffBurnV2, tf.minerFeeBurnV2,
		},
		proto.TransactionTypeVersion{Type: proto.ExchangeTransaction, Version: 1}: txHandleFuncs{
			tc.checkExchangeV1, tp.performExchange, td.createDiffExchange, tf.minerFeeExchange,
		},
		proto.TransactionTypeVersion{Type: proto.ExchangeTransaction, Version: 2}: txHandleFuncs{
			tc.checkExchangeV2, tp.performExchange, td.createDiffExchange, tf.minerFeeExchange,
		},
		proto.TransactionTypeVersion{Type: proto.LeaseTransaction, Version: 1}: txHandleFuncs{
			tc.checkLeaseV1, tp.performLeaseV1, td.createDiffLeaseV1, tf.minerFeeLeaseV1,
		},
		proto.TransactionTypeVersion{Type: proto.LeaseTransaction, Version: 2}: txHandleFuncs{
			tc.checkLeaseV2, tp.performLeaseV2, td.createDiffLeaseV2, tf.minerFeeLeaseV2,
		},
		proto.TransactionTypeVersion{Type: proto.LeaseCancelTransaction, Version: 1}: txHandleFuncs{
			tc.checkLeaseCancelV1, tp.performLeaseCancelV1, td.createDiffLeaseCancelV1, tf.minerFeeLeaseCancelV1,
		},
		proto.TransactionTypeVersion{Type: proto.LeaseCancelTransaction, Version: 2}: txHandleFuncs{
			tc.checkLeaseCancelV2, tp.performLeaseCancelV2, td.createDiffLeaseCancelV2, tf.minerFeeLeaseCancelV2,
		},
		proto.TransactionTypeVersion{Type: proto.CreateAliasTransaction, Version: 1}: txHandleFuncs{
			tc.checkCreateAliasV1, tp.performCreateAliasV1, td.createDiffCreateAliasV1, tf.minerFeeCreateAliasV1,
		},
		proto.TransactionTypeVersion{Type: proto.CreateAliasTransaction, Version: 2}: txHandleFuncs{
			tc.checkCreateAliasV2, tp.performCreateAliasV2, td.createDiffCreateAliasV2, tf.minerFeeCreateAliasV2,
		},
		proto.TransactionTypeVersion{Type: proto.MassTransferTransaction, Version: 1}: txHandleFuncs{
			tc.checkMassTransferV1, nil, td.createDiffMassTransferV1, tf.minerFeeMassTransferV1,
		},
		proto.TransactionTypeVersion{Type: proto.DataTransaction, Version: 1}: txHandleFuncs{
			tc.checkDataV1, tp.performDataV1, td.createDiffDataV1, tf.minerFeeDataV1,
		},
		proto.TransactionTypeVersion{Type: proto.SponsorshipTransaction, Version: 1}: txHandleFuncs{
			tc.checkSponsorshipV1, tp.performSponsorshipV1, td.createDiffSponsorshipV1, tf.minerFeeSponsorshipV1,
		},
		proto.TransactionTypeVersion{Type: proto.SetScriptTransaction, Version: 1}: txHandleFuncs{
			tc.checkSetScriptV1, tp.performSetScriptV1, td.createDiffSetScriptV1, tf.minerFeeSetScriptV1,
		},
		proto.TransactionTypeVersion{Type: proto.SetAssetScriptTransaction, Version: 1}: txHandleFuncs{
			tc.checkSetAssetScriptV1, tp.performSetAssetScriptV1, td.createDiffSetAssetScriptV1, tf.minerFeeSetAssetScriptV1,
		},
		proto.TransactionTypeVersion{Type: proto.InvokeScriptTransaction, Version: 1}: txHandleFuncs{
			tc.checkInvokeScriptV1, nil, td.createDiffInvokeScriptV1, tf.minerFeeInvokeScriptV1,
		},
	}
}

func newTransactionHandler(
	genesis crypto.Signature,
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
	tv := tx.GetTypeVersion()
	funcs, ok := h.funcs[tv]
	if !ok {
		return nil, errors.Errorf("No function handler implemented for tx type %d and version %v\n", tv.Type, tv.Version)
	}
	if funcs.check == nil {
		// No check func for this combination of transaction type and version.
		return nil, nil
	}
	return funcs.check(tx, info)
}

func (h *transactionHandler) performTx(tx proto.Transaction, info *performerInfo) error {
	tv := tx.GetTypeVersion()
	funcs, ok := h.funcs[tv]
	if !ok {
		return errors.Errorf("No function handler implemented for tx type %d and version %v\n", tv.Type, tv.Version)
	}
	if funcs.perform == nil {
		// No perform func for this combination of transaction type and version.
		return nil
	}
	return funcs.perform(tx, info)
}

func (h *transactionHandler) createDiffTx(tx proto.Transaction, info *differInfo) (txDiff, error) {
	tv := tx.GetTypeVersion()
	funcs, ok := h.funcs[tv]
	if !ok {
		return txDiff{}, errors.Errorf("No function handler implemented for tx type %d and version %v\n", tv.Type, tv.Version)
	}
	if funcs.createDiff == nil {
		// No createDiff func for this combination of transaction type and version.
		return txDiff{}, nil
	}
	return funcs.createDiff(tx, info)
}

func (h *transactionHandler) minerFeeTx(tx proto.Transaction, distr *feeDistribution) error {
	tv := tx.GetTypeVersion()
	funcs, ok := h.funcs[tv]
	if !ok {
		return errors.Errorf("No function handler implemented for tx type %d and version %v\n", tv.Type, tv.Version)
	}
	if funcs.countFee == nil {
		// No countFee func for this combination of transaction type and version.
		return nil
	}
	return funcs.countFee(tx, distr)
}
