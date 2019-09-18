package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	scriptExtraFee = 400000
	FeeUnit        = 100000
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

type feeValidationParams struct {
	stor           *blockchainEntitiesStorage
	settings       *settings.BlockchainSettings
	initialisation bool
}

func minFeeInUnits(features *features, tx proto.Transaction) (uint64, error) {
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
		smartAccountsActive, err := features.isActivated(int16(settings.SmartAccounts))
		if err != nil {
			return 0, err
		}
		var dtxBytes []byte
		if smartAccountsActive {
			dtxBytes, err = dtx.BodyMarshalBinary()
			if err != nil {
				return 0, err
			}
		} else {
			dtxBytes, err = dtx.MarshalBinary()
			if err != nil {
				return 0, err
			}
		}
		fee += uint64((len(dtxBytes) - 1) / 1024)
	}
	return fee, nil
}

func minFeeInWaves(tx proto.Transaction, params *feeValidationParams) (uint64, error) {
	feeInUnits, err := minFeeInUnits(params.stor.features, tx)
	if err != nil {
		return 0, err
	}
	minFee := feeInUnits * FeeUnit
	senderAddr, err := proto.NewAddressFromPublicKey(params.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return 0, err
	}
	scripted, err := params.stor.accountsScripts.newestHasVerifier(senderAddr, !params.initialisation)
	if err != nil {
		return 0, err
	}
	if scripted {
		minFee += scriptExtraFee
	}
	return minFee, nil
}

func checkMinFeeWaves(tx proto.Transaction, params *feeValidationParams) error {
	minWaves, err := minFeeInWaves(tx, params)
	if err != nil {
		return errors.Errorf("failed to calculate min fee in Waves: %v\n", err)
	}
	fee := tx.GetFee()
	if fee < minWaves {
		return errors.Errorf("fee %d is less than minimum value of %d\n", fee, minWaves)
	}
	return nil
}

func checkMinFeeAsset(tx proto.Transaction, feeAssetID crypto.Digest, params *feeValidationParams) error {
	isSponsored, err := params.stor.sponsoredAssets.newestIsSponsored(feeAssetID, !params.initialisation)
	if err != nil {
		return errors.Errorf("newestIsSponsored: %v\n", err)
	}
	if !isSponsored {
		return errors.Errorf("asset %s is not sponsored", feeAssetID.String())
	}
	minWaves, err := minFeeInWaves(tx, params)
	if err != nil {
		return errors.Errorf("failed to calculate min fee in Waves: %v\n", err)
	}
	minAsset, err := params.stor.sponsoredAssets.wavesToSponsoredAsset(feeAssetID, minWaves)
	if err != nil {
		return errors.Errorf("wavesToSponsoredAsset() failed: %v\n", err)
	}
	fee := tx.GetFee()
	if fee < minAsset {
		return errors.Errorf("fee %d is less than minimum value of %d\n", fee, minAsset)
	}
	return nil
}
