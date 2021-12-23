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
		amount, err = tf.stor.sponsoredAssets.sponsoredAssetToWaves(proto.AssetIDFromDigest(asset.ID), fee)
		if err != nil {
			return err
		}
		// Asset is now Waves.
		asset = proto.NewOptionalAssetWaves()
	}
	ngActivated, err := tf.stor.features.newestIsActivatedForNBlocks(int16(settings.NG), 1)
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
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeTransferWithSig(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return errors.New("failed to convert interface to TransferWithSig tx")
	}
	return tf.minerFee(distr, tx.Fee, tx.FeeAsset)
}

func (tf *transactionFeeCounter) minerFeeTransferWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return errors.New("failed to convert interface to TransferWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, tx.FeeAsset)
}

func (tf *transactionFeeCounter) minerFeeEthereumTxWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return errors.New("failed to convert interface to EthereumTransaction transaction")
	}

	return tf.minerFee(distr, tx.GetFee(), proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeIssueWithSig(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.IssueWithSig)
	if !ok {
		return errors.New("failed to convert interface to IssueWithSig tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeIssueWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.IssueWithProofs)
	if !ok {
		return errors.New("failed to convert interface to IssueWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeReissueWithSig(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return errors.New("failed to convert interface to ReissueWithSig tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeReissueWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return errors.New("failed to convert interface to ReissueWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeBurnWithSig(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return errors.New("failed to convert interface to BurnWithSig tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeBurnWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return errors.New("failed to convert interface to BurnWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeExchange(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return errors.New("failed to convert interface to Exchange tx")
	}
	return tf.minerFee(distr, tx.GetFee(), proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeLeaseWithSig(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return errors.New("failed to convert interface to LeaseWithSig tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeLeaseWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return errors.New("failed to convert interface to LeaseWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeLeaseCancelWithSig(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.LeaseCancelWithSig)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelWithSig tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeLeaseCancelWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.LeaseCancelWithProofs)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeCreateAliasWithSig(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasWithSig tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeCreateAliasWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeMassTransferWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.MassTransferWithProofs)
	if !ok {
		return errors.New("failed to convert interface to MassTransferWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeDataWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.DataWithProofs)
	if !ok {
		return errors.New("failed to convert interface to DataWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeSponsorshipWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return errors.New("failed to convert interface to SponsorshipWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeSetScriptWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.SetScriptWithProofs)
	if !ok {
		return errors.New("failed to convert interface to SetScriptWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeSetAssetScriptWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.SetAssetScriptWithProofs)
	if !ok {
		return errors.New("failed to convert interface to SetAssetScriptWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, proto.NewOptionalAssetWaves())
}

func (tf *transactionFeeCounter) minerFeeInvokeScriptWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.InvokeScriptWithProofs)
	if !ok {
		return errors.New("failed to convert interface to InvokeScriptWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, tx.FeeAsset)
}

func (tf *transactionFeeCounter) minerFeeInvokeExpressionWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs)
	if !ok {
		return errors.New("failed to convert interface to InvokeExpressionWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, tx.FeeAsset)
}

func (tf *transactionFeeCounter) minerFeeUpdateAssetInfoWithProofs(transaction proto.Transaction, distr *feeDistribution) error {
	tx, ok := transaction.(*proto.UpdateAssetInfoWithProofs)
	if !ok {
		return errors.New("failed to convert interface to UpdateAssetInfoWithProofs tx")
	}
	return tf.minerFee(distr, tx.Fee, tx.FeeAsset)
}
