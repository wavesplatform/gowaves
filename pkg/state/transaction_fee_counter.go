package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
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

type transactionFeeCounter struct {
	stor *blockchainEntitiesStorage
}

func newTransactionFeeCounter(stor *blockchainEntitiesStorage) (*transactionFeeCounter, error) {
	return &transactionFeeCounter{stor}, nil
}

func (tf *transactionFeeCounter) minerFee(distr *feeDistribution, fee uint64, asset proto.OptionalAsset) error {
	amount := fee
	sponsorshipActivated, err := tf.stor.sponsoredAssets.isSponsorshipActivated()
	if err != nil {
		return err
	}
	if sponsorshipActivated && asset.Present {
		// If sponsorship is activated and there is fee asset, we must convert it to Waves.
		amount, err = tf.stor.sponsoredAssets.sponsoredAssetToWaves(asset.ID, fee)
		if err != nil {
			return err
		}
		// Asset is now Waves.
		asset.Present = false
	}
	ngActivated, err := tf.stor.features.isActivated(int16(settings.NG))
	if err != nil {
		return err
	}
	if !asset.Present {
		// Waves.
		distr.totalWavesFees += amount
		distr.currentWavesBlockFees += calculateCurrentBlockTxFee(amount, ngActivated)
	} else {
		// Other asset.
		distr.totalFees[asset.ID] += amount
		distr.currentBlockFees[asset.ID] += calculateCurrentBlockTxFee(amount, ngActivated)
	}
	return nil
}

func (tf *transactionFeeCounter) minerFeePayment(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.Payment)
	if !ok {
		return errors.New("failed to convert interface to Payment tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeTransferV1(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.TransferV1)
	if !ok {
		return errors.New("failed to convert interface to TransferV1 tx")
	}
	return tf.minerFee(distr, tx.Fee, tx.FeeAsset)
}

func (tf *transactionFeeCounter) minerFeeTransferV2(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.TransferV2)
	if !ok {
		return errors.New("failed to convert interface to TransferV2 tx")
	}
	return tf.minerFee(distr, tx.Fee, tx.FeeAsset)
}

func (tf *transactionFeeCounter) minerFeeIssueV1(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.IssueV1)
	if !ok {
		return errors.New("failed to convert interface to IssueV1 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeIssueV2(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.IssueV2)
	if !ok {
		return errors.New("failed to convert interface to IssueV2 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeReissueV1(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.ReissueV1)
	if !ok {
		return errors.New("failed to convert interface to ReissueV1 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeReissueV2(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.ReissueV2)
	if !ok {
		return errors.New("failed to convert interface to ReissueV2 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeBurnV1(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.BurnV1)
	if !ok {
		return errors.New("failed to convert interface to BurnV1 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeBurnV2(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.BurnV2)
	if !ok {
		return errors.New("failed to convert interface to BurnV2 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeExchange(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return errors.New("failed to convert interface to Exchange tx")
	}
	return tf.minerFee(distr, tx.GetFee(), proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeLeaseV1(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.LeaseV1)
	if !ok {
		return errors.New("failed to convert interface to LeaseV1 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeLeaseV2(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.LeaseV2)
	if !ok {
		return errors.New("failed to convert interface to LeaseV2 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeLeaseCancelV1(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.LeaseCancelV1)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelV1 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeLeaseCancelV2(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.LeaseCancelV2)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelV2 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeCreateAliasV1(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.CreateAliasV1)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasV1 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeCreateAliasV2(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.CreateAliasV2)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasV2 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeMassTransferV1(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.MassTransferV1)
	if !ok {
		return errors.New("failed to convert interface to MassTrnasferV1 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeDataV1(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.DataV1)
	if !ok {
		return errors.New("failed to convert interface to DataV1 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}

func (tf *transactionFeeCounter) minerFeeSponsorshipV1(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.SponsorshipV1)
	if !ok {
		return errors.New("failed to convert interface to SponsorshipV1 tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.OptionalAsset{Present: false})
}
