package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	FeeUnit = 100000
)

var feeConstants = map[proto.TransactionType]uint64{
	proto.GenesisTransaction:        0,
	proto.PaymentTransaction:        1,
	proto.TransferTransaction:       1,
	proto.IssueTransaction:          1000,
	proto.ReissueTransaction:        1000,
	proto.BurnTransaction:           1,
	proto.ExchangeTransaction:       3,
	proto.MassTransferTransaction:   1,
	proto.LeaseTransaction:          1,
	proto.LeaseCancelTransaction:    1,
	proto.CreateAliasTransaction:    1,
	proto.DataTransaction:           1,
	proto.SetScriptTransaction:      10,
	proto.SponsorshipTransaction:    1000,
	proto.SetAssetScriptTransaction: (1000 - 4),
	proto.InvokeScriptTransaction:   5,
}

func minFeeInUnits(tx proto.Transaction) (uint64, error) {
	txType := tx.GetTypeVersion().Type
	baseFee, ok := feeConstants[txType]
	if !ok {
		return 0, errors.Errorf("bad tx type %v\n", txType)
	}
	fee := baseFee
	switch txType {
	case proto.MassTransferTransaction:
		mtx, ok := tx.(*proto.MassTransferV1)
		if !ok {
			return 0, errors.New("failed to convert interface to MassTransfer transaction")
		}
		fee += uint64((len(mtx.Transfers) + 1) / 2)
	case proto.DataTransaction:
		dtx, ok := tx.(*proto.DataV1)
		if !ok {
			return 0, errors.New("failed to convert interface to DataTransaction")
		}
		dtxBytes, err := dtx.MarshalBinary()
		if err != nil {
			return 0, err
		}
		fee += uint64((len(dtxBytes) - 1) / 1024)
	}
	return fee, nil
}

func minFeeInWaves(tx proto.Transaction) (uint64, error) {
	feeInUnits, err := minFeeInUnits(tx)
	if err != nil {
		return 0, err
	}
	return feeInUnits * FeeUnit, nil
}

func checkMinFeeWaves(tx proto.Transaction) error {
	minWaves, err := minFeeInWaves(tx)
	if err != nil {
		return errors.Errorf("failed to calculate min fee in Waves: %v\n", err)
	}
	fee := tx.GetFee()
	if fee < minWaves {
		return errors.Errorf("fee %d is less than minimum value of %d\n", fee, minWaves)
	}
	return nil
}

func checkMinFeeAsset(sponsoredAssets *sponsoredAssets, tx proto.Transaction, feeAssetID crypto.Digest) error {
	isSponsored, err := sponsoredAssets.newestIsSponsored(feeAssetID, true)
	if err != nil {
		return errors.Errorf("newestIsSponsored: %v\n", err)
	}
	if !isSponsored {
		return errors.Errorf("asset %s is not sponsored", feeAssetID.String())
	}
	minWaves, err := minFeeInWaves(tx)
	if err != nil {
		return errors.Errorf("failed to calculate min fee in Waves: %v\n", err)
	}
	minAsset, err := sponsoredAssets.wavesToSponsoredAsset(feeAssetID, minWaves)
	if err != nil {
		return errors.Errorf("wavesToSponsoredAsset() failed: %v\n", err)
	}
	fee := tx.GetFee()
	if fee < minAsset {
		return errors.Errorf("fee %d is less than minimum value of %d\n", fee, minAsset)
	}
	return nil
}
