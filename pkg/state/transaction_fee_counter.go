package state

import (
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

func (tf *transactionFeeCounter) minerFeeByTransaction(transaction proto.Transaction, distr *feeDistribution) error {
	var (
		fee      = transaction.GetFee()
		feeAsset = transaction.GetFeeAsset()
	)
	return tf.minerFee(distr, fee, feeAsset)
}

func (tf *transactionFeeCounter) minerFee(distr *feeDistribution, fee uint64, feeAsset proto.OptionalAsset) error {
	amount := fee
	sponsorshipActivated, err := tf.stor.sponsoredAssets.isSponsorshipActivated()
	if err != nil {
		return err
	}
	if sponsorshipActivated && feeAsset.Present {
		// If sponsorship is activated and there is fee asset, we must convert it to Waves.
		amount, err = tf.stor.sponsoredAssets.sponsoredAssetToWaves(proto.AssetIDFromDigest(feeAsset.ID), fee)
		if err != nil {
			return err
		}
		// Asset is now Waves.
		feeAsset = proto.NewOptionalAssetWaves()
	}
	ngActivated, err := tf.stor.features.newestIsActivatedForNBlocks(int16(settings.NG), 1)
	if err != nil {
		return err
	}
	if !feeAsset.Present {
		// Waves.
		distr.totalWavesFees += amount
		distr.currentWavesBlockFees += calculateCurrentBlockTxFee(amount, ngActivated)
	} else {
		// Other asset.
		distr.totalFees[feeAsset.ID] += amount
		distr.currentBlockFees[feeAsset.ID] += calculateCurrentBlockTxFee(amount, ngActivated)
	}
	return nil
}
