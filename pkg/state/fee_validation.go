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
	proto.GenesisTransaction:         0,
	proto.PaymentTransaction:         1,
	proto.TransferTransaction:        1,
	proto.IssueTransaction:           1000,
	proto.ReissueTransaction:         1000,
	proto.BurnTransaction:            1,
	proto.ExchangeTransaction:        3,
	proto.MassTransferTransaction:    1,
	proto.LeaseTransaction:           1,
	proto.LeaseCancelTransaction:     1,
	proto.CreateAliasTransaction:     1,
	proto.DataTransaction:            1,
	proto.SetScriptTransaction:       10,
	proto.SponsorshipTransaction:     1000,
	proto.SetAssetScriptTransaction:  1000 - 4,
	proto.InvokeScriptTransaction:    5,
	proto.UpdateAssetInfoTransaction: 1,
}

type feeValidationParams struct {
	stor           *blockchainEntitiesStorage
	settings       *settings.BlockchainSettings
	initialisation bool
	txAssets       *txAssets
}

func minFeeInUnits(params *feeValidationParams, tx proto.Transaction) (uint64, error) {
	txType := tx.GetTypeInfo().Type
	baseFee, ok := feeConstants[txType]
	if !ok {
		return 0, errors.Errorf("bad tx type %v\n", txType)
	}
	fee := baseFee
	switch txType {
	case proto.IssueTransaction:
		nft := false
		switch itx := tx.(type) {
		case *proto.IssueWithSig:
			nft = itx.Quantity == 1 && itx.Decimals == 0 && !itx.Reissuable
		case *proto.IssueWithProofs:
			nft = itx.Quantity == 1 && itx.Decimals == 0 && !itx.Reissuable
		default:
			return 0, errors.New("failed to convert interface to Issue transaction")
		}
		if nft {
			nftActive, err := params.stor.features.isActivated(int16(settings.ReduceNFTFee))
			if err != nil {
				return 0, err
			}
			if nftActive {
				return fee / 1000, nil
			}
		}
		return fee, nil
	case proto.MassTransferTransaction:
		mtx, ok := tx.(*proto.MassTransferWithProofs)
		if !ok {
			return 0, errors.New("failed to convert interface to MassTransfer transaction")
		}
		fee += uint64((len(mtx.Transfers) + 1) / 2)
	case proto.DataTransaction:
		dtx, ok := tx.(*proto.DataWithProofs)
		if !ok {
			return 0, errors.New("failed to convert interface to DataTransaction")
		}
		smartAccountsActive, err := params.stor.features.isActivated(int16(settings.SmartAccounts))
		if err != nil {
			return 0, err
		}
		scheme := params.settings.AddressSchemeCharacter
		var dtxBytes []byte
		if smartAccountsActive {
			dtxBytes, err = proto.MarshalTxBody(scheme, dtx)
			if err != nil {
				return 0, err
			}
		} else {
			dtxBytes, err = proto.MarshalTx(scheme, dtx)
			if err != nil {
				return 0, err
			}
		}
		fee += uint64((len(dtxBytes) - 1) / 1024)
	case proto.ReissueTransaction:
		multiPayerActive, err := params.stor.features.isActivated(int16(settings.BlockV5))
		if err != nil {
			return 0, err
		}
		if multiPayerActive {
			return fee / 1000, nil
		}
	}
	return fee, nil
}

func scriptsCost(tx proto.Transaction, params *feeValidationParams) (uint64, error) {
	scriptsCost := uint64(0)
	senderAddr, err := proto.NewAddressFromPublicKey(params.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return 0, err
	}
	// TODO: figure out if scripts without verifier count here.
	accountScripted, err := params.stor.scriptsStorage.newestAccountHasVerifier(senderAddr, !params.initialisation)
	if err != nil {
		return 0, err
	}
	if accountScripted {
		scriptsCost += scriptExtraFee
	}
	if params.txAssets.smartAssets != nil {
		// Add extra fee for each of smart assets found.
		scriptsCost += scriptExtraFee * uint64(len(params.txAssets.smartAssets))
	}
	// TODO: the code below is wrong, because scripts for fee assets are never run.
	// Even if sponsorship is disabled, and fee assets can be smart, we don't run scripts for them,
	// because Scala implementation does not.
	// Therefore, the extra fee for smart fee asset below is also wrong, but it must be there,
	// again for compatibility with Scala.
	if params.txAssets.feeAsset.Present {
		hasScript, err := params.stor.scriptsStorage.newestIsSmartAsset(params.txAssets.feeAsset.ID, !params.initialisation)
		if err != nil {
			return 0, err
		}
		if hasScript {
			scriptsCost += scriptExtraFee
		}
	}
	return scriptsCost, nil
}

func minFeeInWaves(tx proto.Transaction, params *feeValidationParams) (uint64, error) {
	feeInUnits, err := minFeeInUnits(params, tx)
	if err != nil {
		return 0, err
	}
	minFee := feeInUnits * FeeUnit
	scriptsCost, err := scriptsCost(tx, params)
	if err != nil {
		return 0, err
	}
	minFee += scriptsCost
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
