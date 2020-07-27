package state

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
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

type assetParams struct {
	quantity   int64
	decimals   int32
	reissuable bool
}

func isNFT(features *features, params assetParams) (bool, error) {
	nftAsset := params.quantity == 1 && params.decimals == 0 && !params.reissuable
	if !nftAsset {
		return false, nil
	}
	nftActivated, err := features.newestIsActivated(int16(settings.ReduceNFTFee))
	if err != nil {
		return false, err
	}
	return nftActivated, nil
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
		var asset assetParams
		switch itx := tx.(type) {
		case *proto.IssueWithSig:
			asset = assetParams{int64(itx.Quantity), int32(itx.Decimals), itx.Reissuable}
		case *proto.IssueWithProofs:
			asset = assetParams{int64(itx.Quantity), int32(itx.Decimals), itx.Reissuable}
		default:
			return 0, errors.New("failed to convert interface to Issue transaction")
		}
		nft, err := isNFT(params.stor.features, asset)
		if err != nil {
			return 0, err
		}
		if nft {
			return fee / 1000, nil
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
		smartAccountsActive, err := params.stor.features.newestIsActivated(int16(settings.SmartAccounts))
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
	case proto.ReissueTransaction, proto.SponsorshipTransaction:
		blockV5Activated, err := params.stor.features.newestIsActivated(int16(settings.BlockV5))
		if err != nil {
			return 0, err
		}
		if blockV5Activated {
			return fee / 1000, nil
		}
	}
	return fee, nil
}

type txCosts struct {
	smartAssets      uint64
	smartAssetsFee   uint64
	smartAccounts    uint64
	smartAccountsFee uint64
	total            uint64
}

func newTxCosts(smartAssets, smartAccounts uint64) *txCosts {
	smartAssetsFee := smartAssets * scriptExtraFee
	smartAccountsFee := smartAccounts * scriptExtraFee
	return &txCosts{
		smartAssets:      smartAssets,
		smartAssetsFee:   smartAssetsFee,
		smartAccounts:    smartAccounts,
		smartAccountsFee: smartAccountsFee,
		total:            smartAssetsFee + smartAccountsFee,
	}
}

// toString is mostly added for integration tests compatibility with Scala.
func (tc *txCosts) toString() string {
	str := ""
	if tc.smartAccounts > 0 {
		str = fmt.Sprintf("State check failed. Reason: Transaction sent from smart account. Requires %d extra fee.", tc.smartAccountsFee)
	} else if tc.smartAssets > 0 {
		str = fmt.Sprintf("State check failed. Reason: Transaction involves %d scripted assets. Requires %d extra fee.", tc.smartAssets, tc.smartAssetsFee)
	}
	return str
}

func scriptsCost(tx proto.Transaction, params *feeValidationParams) (*txCosts, error) {
	smartAssets := uint64(len(params.txAssets.smartAssets))
	senderAddr, err := proto.NewAddressFromPublicKey(params.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return nil, err
	}
	accountScripted, err := params.stor.scriptsStorage.newestAccountHasVerifier(senderAddr, !params.initialisation)
	if err != nil {
		return nil, err
	}
	smartAccounts := uint64(0)
	if accountScripted {
		smartAccounts = 1
	}
	// TODO: the code below is wrong, because scripts for fee assets are never run.
	// Even if sponsorship is disabled, and fee assets can be smart, we don't run scripts for them,
	// because Scala implementation does not.
	// Therefore, the extra fee for smart fee asset below is also wrong, but it must be there,
	// again for compatibility with Scala.
	if params.txAssets.feeAsset.Present {
		hasScript := params.stor.scriptsStorage.newestIsSmartAsset(params.txAssets.feeAsset.ID, !params.initialisation)
		if hasScript {
			smartAssets += 1
		}
	}
	return newTxCosts(smartAssets, smartAccounts), nil
}

func minFeeInWaves(tx proto.Transaction, params *feeValidationParams) (*txCosts, error) {
	feeInUnits, err := minFeeInUnits(params, tx)
	if err != nil {
		return nil, err
	}
	minFee := feeInUnits * FeeUnit
	cost, err := scriptsCost(tx, params)
	if err != nil {
		return nil, err
	}
	cost.total += minFee
	return cost, nil
}

func checkMinFeeWaves(tx proto.Transaction, params *feeValidationParams) error {
	minWaves, err := minFeeInWaves(tx, params)
	if err != nil {
		return errors.Errorf("failed to calculate min fee in Waves: %v\n", err)
	}
	fee := tx.GetFee()
	if fee < minWaves.total {
		feeInfoStr := minWaves.toString()
		return errs.NewFeeValidation(fmt.Sprintf("Fee %d does not exceed minimal value of %d WAVES. %s", fee, minWaves.total, feeInfoStr))
	}
	return nil
}

func checkMinFeeAsset(tx proto.Transaction, feeAssetID crypto.Digest, params *feeValidationParams) error {
	isSponsored, err := params.stor.sponsoredAssets.newestIsSponsored(feeAssetID, !params.initialisation)
	if err != nil {
		return errors.Errorf("newestIsSponsored: %v\n", err)
	}
	if !isSponsored {
		return errs.NewTxValidationError(fmt.Sprintf("Asset %s is not sponsored, cannot be used to pay fees", feeAssetID.String()))
	}
	minWaves, err := minFeeInWaves(tx, params)
	if err != nil {
		return errors.Errorf("failed to calculate min fee in Waves: %v\n", err)
	}
	minAsset, err := params.stor.sponsoredAssets.wavesToSponsoredAsset(feeAssetID, minWaves.total)
	if err != nil {
		return errors.Errorf("wavesToSponsoredAsset() failed: %v\n", err)
	}
	fee := tx.GetFee()
	if fee < minAsset {
		feeInfoStr := minWaves.toString()
		return errs.NewFeeValidation(fmt.Sprintf("fee %d is less than minimum value of %d. %s\n", fee, minAsset, feeInfoStr))
	}
	return nil
}
