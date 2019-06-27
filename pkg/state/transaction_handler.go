package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type txCheckFunc func(proto.Transaction, *checkerInfo) error
type txPerformFunc func(proto.Transaction, *performerInfo) error
type txCreateDiffFunc func(proto.Transaction, *differInfo) (txDiff, error)
type txCountFeeFunc func(proto.Transaction, *feeDistribution, bool) error

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

	funcs handles
}

func buildHanndles(tc *transactionChecker, tp *transactionPerformer, td *transactionDiffer) handles {
	return handles{
		proto.TransactionTypeVersion{Type: proto.GenesisTransaction, Version: 1}: txHandleFuncs{
			tc.checkGenesis, nil, td.createDiffGenesis, nil,
		},
		proto.TransactionTypeVersion{Type: proto.PaymentTransaction, Version: 1}: txHandleFuncs{
			tc.checkPayment, nil, td.createDiffPayment, minerFeePayment,
		},
		proto.TransactionTypeVersion{Type: proto.TransferTransaction, Version: 1}: txHandleFuncs{
			tc.checkTransferV1, nil, td.createDiffTransferV1, minerFeeTransferV1,
		},
		proto.TransactionTypeVersion{Type: proto.TransferTransaction, Version: 2}: txHandleFuncs{
			tc.checkTransferV2, nil, td.createDiffTransferV2, minerFeeTransferV2,
		},
		proto.TransactionTypeVersion{Type: proto.IssueTransaction, Version: 1}: txHandleFuncs{
			tc.checkIssueV1, tp.performIssueV1, td.createDiffIssueV1, minerFeeIssueV1,
		},
		proto.TransactionTypeVersion{Type: proto.IssueTransaction, Version: 2}: txHandleFuncs{
			tc.checkIssueV2, tp.performIssueV2, td.createDiffIssueV2, minerFeeIssueV2,
		},
		proto.TransactionTypeVersion{Type: proto.ReissueTransaction, Version: 1}: txHandleFuncs{
			tc.checkReissueV1, tp.performReissueV1, td.createDiffReissueV1, minerFeeReissueV1,
		},
		proto.TransactionTypeVersion{Type: proto.ReissueTransaction, Version: 2}: txHandleFuncs{
			tc.checkReissueV2, tp.performReissueV2, td.createDiffReissueV2, minerFeeReissueV2,
		},
		proto.TransactionTypeVersion{Type: proto.BurnTransaction, Version: 1}: txHandleFuncs{
			tc.checkBurnV1, tp.performBurnV1, td.createDiffBurnV1, minerFeeBurnV1,
		},
		proto.TransactionTypeVersion{Type: proto.BurnTransaction, Version: 2}: txHandleFuncs{
			tc.checkBurnV2, tp.performBurnV2, td.createDiffBurnV2, minerFeeBurnV2,
		},
		proto.TransactionTypeVersion{Type: proto.ExchangeTransaction, Version: 1}: txHandleFuncs{
			tc.checkExchange, nil, td.createDiffExchange, minerFeeExchange,
		},
		proto.TransactionTypeVersion{Type: proto.ExchangeTransaction, Version: 2}: txHandleFuncs{
			tc.checkExchange, nil, td.createDiffExchange, minerFeeExchange,
		},
		proto.TransactionTypeVersion{Type: proto.LeaseTransaction, Version: 1}: txHandleFuncs{
			tc.checkLeaseV1, tp.performLeaseV1, td.createDiffLeaseV1, minerFeeLeaseV1,
		},
		proto.TransactionTypeVersion{Type: proto.LeaseTransaction, Version: 2}: txHandleFuncs{
			tc.checkLeaseV2, tp.performLeaseV2, td.createDiffLeaseV2, minerFeeLeaseV2,
		},
		proto.TransactionTypeVersion{Type: proto.LeaseCancelTransaction, Version: 1}: txHandleFuncs{
			tc.checkLeaseCancelV1, tp.performLeaseCancelV1, td.createDiffLeaseCancelV1, minerFeeLeaseCancelV1,
		},
		proto.TransactionTypeVersion{Type: proto.LeaseCancelTransaction, Version: 2}: txHandleFuncs{
			tc.checkLeaseCancelV2, tp.performLeaseCancelV2, td.createDiffLeaseCancelV2, minerFeeLeaseCancelV2,
		},
		proto.TransactionTypeVersion{Type: proto.CreateAliasTransaction, Version: 1}: txHandleFuncs{
			tc.checkCreateAliasV1, tp.performCreateAliasV1, td.createDiffCreateAliasV1, minerFeeCreateAliasV1,
		},
		proto.TransactionTypeVersion{Type: proto.CreateAliasTransaction, Version: 2}: txHandleFuncs{
			tc.checkCreateAliasV2, tp.performCreateAliasV2, td.createDiffCreateAliasV2, minerFeeCreateAliasV2,
		},
		proto.TransactionTypeVersion{Type: proto.MassTransferTransaction, Version: 1}: txHandleFuncs{
			tc.checkMassTransferV1, nil, td.createDiffMassTransferV1, minerFeeMassTransferV1,
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
	return &transactionHandler{tc: tc, tp: tp, td: td, funcs: buildHanndles(tc, tp, td)}, nil
}

func (h *transactionHandler) checkTx(tx proto.Transaction, info *checkerInfo) error {
	tv := tx.GetTypeVersion()
	funcs, ok := h.funcs[tv]
	if !ok {
		return errors.Errorf("No function handler implemented for tx type %d and version %v\n", tv.Type, tv.Version)
	}
	if funcs.check == nil {
		// No check func for this combination of transaction type and version.
		return nil
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

func (h *transactionHandler) minerFeeTx(tx proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tv := tx.GetTypeVersion()
	funcs, ok := h.funcs[tv]
	if !ok {
		return errors.Errorf("No function handler implemented for tx type %d and version %v\n", tv.Type, tv.Version)
	}
	if funcs.countFee == nil {
		// No countFee func for this combination of transaction type and version.
		return nil
	}
	return funcs.countFee(tx, distr, ngActivated)
}
