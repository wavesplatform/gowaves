package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	// ngCurrentBlockFeePercentage is percentage of fees miner gets from the current block after activating NG (40%).
	// It is represented as (2 / 5), to make it compatible with Scala implementation.
	ngCurrentBlockFeePercentageDivider  = 5
	ngCurrentBlockFeePercentageDividend = 2
)

func calculateCurrentBlockTxFee(txFee uint64, ngActivated bool) uint64 {
	if ngActivated {
		return txFee / ngCurrentBlockFeePercentageDivider * ngCurrentBlockFeePercentageDividend
	}
	return txFee
}

func minerFee(distr *feeDistribution, fee, curFee uint64, asset proto.OptionalAsset) error {
	if !asset.Present {
		distr.totalWavesFees += fee
		distr.currentWavesBlockFees += curFee
	} else {
		distr.totalFees[asset.ID] += fee
		distr.currentBlockFees[asset.ID] += curFee
	}
	return nil
}

func minerFeePayment(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.Payment)
	if !ok {
		return errors.New("failed to convert interface to Payment tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeTransferV1(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.TransferV1)
	if !ok {
		return errors.New("failed to convert interface to TransferV1 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), tx.FeeAsset)
}

func minerFeeTransferV2(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.TransferV2)
	if !ok {
		return errors.New("failed to convert interface to TransferV2 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), tx.FeeAsset)
}

func minerFeeIssueV1(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.IssueV1)
	if !ok {
		return errors.New("failed to convert interface to IssueV1 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeIssueV2(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.IssueV2)
	if !ok {
		return errors.New("failed to convert interface to IssueV2 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeReissueV1(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.ReissueV1)
	if !ok {
		return errors.New("failed to convert interface to ReissueV1 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeReissueV2(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.ReissueV2)
	if !ok {
		return errors.New("failed to convert interface to ReissueV2 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeBurnV1(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.BurnV1)
	if !ok {
		return errors.New("failed to convert interface to BurnV1 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeBurnV2(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.BurnV2)
	if !ok {
		return errors.New("failed to convert interface to BurnV2 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeExchange(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return errors.New("failed to convert interface to Exchange tx")
	}
	return minerFee(distr, tx.GetFee(), calculateCurrentBlockTxFee(tx.GetFee(), ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeLeaseV1(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.LeaseV1)
	if !ok {
		return errors.New("failed to convert interface to LeaseV1 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeLeaseV2(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.LeaseV2)
	if !ok {
		return errors.New("failed to convert interface to LeaseV2 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeLeaseCancelV1(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.LeaseCancelV1)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelV1 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeLeaseCancelV2(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.LeaseCancelV2)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelV2 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeCreateAliasV1(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.CreateAliasV1)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasV1 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeCreateAliasV2(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.CreateAliasV2)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasV2 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeMassTransferV1(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.MassTransferV1)
	if !ok {
		return errors.New("failed to convert interface to MassTrnasferV1 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}

func minerFeeDataV1(transaction proto.Transaction, distr *feeDistribution, ngActivated bool) error {
	tx, ok := transaction.(*proto.DataV1)
	if !ok {
		return errors.New("failed to convert interface to DataV1 tx")
	}
	return minerFee(distr, tx.Fee, calculateCurrentBlockTxFee(tx.Fee, ngActivated), proto.OptionalAsset{Present: false})
}
