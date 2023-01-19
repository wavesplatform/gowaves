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

	SetScriptTransactionV6Fee = 1
)

var feeConstants = map[proto.TransactionType]uint64{
	proto.GenesisTransaction:          0,
	proto.PaymentTransaction:          1,
	proto.TransferTransaction:         1,
	proto.IssueTransaction:            1000,
	proto.ReissueTransaction:          1000,
	proto.BurnTransaction:             1,
	proto.ExchangeTransaction:         3,
	proto.MassTransferTransaction:     1,
	proto.LeaseTransaction:            1,
	proto.LeaseCancelTransaction:      1,
	proto.CreateAliasTransaction:      1,
	proto.DataTransaction:             1,
	proto.SetScriptTransaction:        10,
	proto.SponsorshipTransaction:      1000,
	proto.SetAssetScriptTransaction:   1000 - 4,
	proto.InvokeScriptTransaction:     5,
	proto.UpdateAssetInfoTransaction:  1,
	proto.EthereumMetamaskTransaction: 0, // special case, should be handled with corresponding EthTxKind
	proto.InvokeExpressionTransaction: 5,
}

type feeValidationParams struct {
	stor             *blockchainEntitiesStorage
	settings         *settings.BlockchainSettings
	txAssets         *txAssets
	rideV5Activated  bool
	estimatorVersion int
}

type assetParams struct {
	quantity   int64
	decimals   int32
	reissuable bool
}

func isNFT(features featuresState, params assetParams) (bool, error) {
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

func isSmartAssetsFree(tx proto.Transaction, rideV5Activated bool) (bool, error) {
	if !rideV5Activated {
		return false, nil
	}
	switch tx.GetTypeInfo().Type {
	// TODO: add case with proto.InvokeExpressionTransaction after this tx type support
	case proto.InvokeScriptTransaction:
		return true, nil
	case proto.EthereumMetamaskTransaction:
		ethTx, ok := tx.(*proto.EthereumTransaction)
		if !ok {
			return false, errors.New("failed to convert interface to EthereumTransaction")
		}
		if _, ok := ethTx.TxKind.(*proto.EthereumInvokeScriptTxKind); ok {
			return true, nil
		}
	}
	return false, nil
}

// minFeeInUnits returns minimal fee in units and error
func minFeeInUnits(params *feeValidationParams, tx proto.Transaction) (uint64, error) {
	txType := tx.GetTypeInfo().Type
	baseFee, ok := feeConstants[txType]
	if !ok {
		return 0, errors.Errorf("bad tx type (%v)", txType)
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
		isRideV6Activated, err := params.stor.features.newestIsActivated(int16(settings.RideV6))
		if err != nil {
			return 0, err
		}
		var (
			scheme         = params.settings.AddressSchemeCharacter
			dtxBytesForFee int
		)
		switch {
		case isRideV6Activated:
			dtxBytesForFee = dtx.Entries.PayloadSize()
		case proto.IsProtobufTx(tx):
			dtxBytesForFee = dtx.ProtoPayloadSize()
		case smartAccountsActive:
			dtxBytes, err := proto.MarshalTxBody(scheme, dtx)
			if err != nil {
				return 0, err
			}
			dtxBytesForFee = len(dtxBytes)
		default:
			dtxBytes, err := dtx.MarshalBinary(scheme)
			if err != nil {
				return 0, err
			}
			dtxBytesForFee = len(dtxBytes)
		}
		if dtxBytesForFee < 0 {
			panic(fmt.Sprintf("BUG, CREATE REPORT: dataTx bytes size (%d) must not be lower than zero", dtxBytesForFee))
		}
		fee += uint64((dtxBytesForFee - 1) / 1024)
	case proto.ReissueTransaction, proto.SponsorshipTransaction:
		blockV5Activated, err := params.stor.features.newestIsActivated(int16(settings.BlockV5))
		if err != nil {
			return 0, err
		}
		if blockV5Activated {
			return fee / 1000, nil
		}
	case proto.SetScriptTransaction:
		isRideV6Activated, err := params.stor.features.newestIsActivated(int16(settings.RideV6))
		if err != nil {
			return 0, err
		}
		if !isRideV6Activated {
			break
		}
		fee = SetScriptTransactionV6Fee
		stx, ok := tx.(*proto.SetScriptWithProofs)
		if !ok {
			return 0, errors.New("failed to convert interface to SetScriptTransaction")
		}

		stxBytesForFee := len(stx.Script)

		fee += uint64((stxBytesForFee - 1) / proto.KiB)
	case proto.EthereumMetamaskTransaction:
		ethTx, ok := tx.(*proto.EthereumTransaction)
		if !ok {
			return 0, errors.New("failed to convert interface to EthereumTransaction")
		}
		switch kind := ethTx.TxKind.(type) {
		case *proto.EthereumTransferWavesTxKind, *proto.EthereumTransferAssetsErc20TxKind:
			fee = feeConstants[proto.TransferTransaction]
		case *proto.EthereumInvokeScriptTxKind:
			fee = feeConstants[proto.InvokeScriptTransaction]
		default:
			return 0, errors.Errorf("unknown ethereum tx kind (%T)", kind)
		}
	}
	if fee == 0 && txType != proto.GenesisTransaction {
		return 0, errors.Errorf("zero fee allowed only for genesis transaction, but not for tx with type (%d)", txType)
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

func newTxCosts(smartAssets, smartAccounts uint64, isSmartAssetsFree, isSmartAccountFree bool) *txCosts {
	smartAssetsFee := smartAssets * scriptExtraFee
	smartAccountsFee := smartAccounts * scriptExtraFee
	if isSmartAssetsFree {
		smartAssetsFee = 0
	}
	if isSmartAccountFree {
		smartAccountsFee = 0
	}
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
	if tc.smartAccounts == 0 && tc.smartAssets == 0 {
		return ""
	}
	str := "State check failed. Reason: "
	if tc.smartAccounts > 0 {
		str += fmt.Sprintf("Transaction sent from smart account. Requires %d extra fee. ", tc.smartAccountsFee)
	}
	if tc.smartAssets > 0 {
		str += fmt.Sprintf("Transaction involves %d scripted assets. Requires %d extra fee.", tc.smartAssets, tc.smartAssetsFee)
	}
	return str
}

func scriptsCost(tx proto.Transaction, params *feeValidationParams) (*txCosts, error) {
	smartAssets := uint64(len(params.txAssets.smartAssets))
	senderAddr, err := tx.GetSender(params.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, err
	}

	// senderWavesAddr needs only for newestAccountHasVerifier check
	senderWavesAddr, err := senderAddr.ToWavesAddress(params.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to transform (%T) address type to WavesAddress type", senderAddr)
	}
	accountScripted, err := params.stor.scriptsStorage.newestAccountHasVerifier(senderWavesAddr)
	if err != nil {
		return nil, err
	}

	// check complexity of script for free verifier if complexity <= 200
	complexity := 0
	if accountScripted && params.rideV5Activated {
		treeEstimation, err := params.stor.scriptsComplexity.newestScriptComplexityByAddr(senderAddr, params.estimatorVersion)
		if err != nil {
			return nil, errors.Errorf("failed to get complexity by addr from store, %v", err)
		}
		complexity = treeEstimation.Verifier
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
		hasScript, err := params.stor.scriptsStorage.newestIsSmartAsset(proto.AssetIDFromDigest(params.txAssets.feeAsset.ID))
		if err != nil {
			return nil, err
		}
		if hasScript {
			smartAssets += 1
		}
	}
	smartAssetsFree, err := isSmartAssetsFree(tx, params.rideV5Activated)
	if err != nil {
		return nil, err
	}
	smartAccountFree := accountScripted && params.rideV5Activated && complexity <= FreeVerifierComplexity
	return newTxCosts(smartAssets, smartAccounts, smartAssetsFree, smartAccountFree), nil
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
		return errors.Errorf("failed to calculate min fee in Waves: %v", err)
	}
	fee := tx.GetFee()
	if fee < minWaves.total {
		feeInfoStr := minWaves.toString()
		return errs.NewFeeValidation(fmt.Sprintf("Fee %d does not exceed minimal value of %d WAVES. %s",
			fee, minWaves.total, feeInfoStr,
		))
	}
	return nil
}

func checkMinFeeAsset(tx proto.Transaction, feeAssetID crypto.Digest, params *feeValidationParams) error {
	shortFeeAssetID := proto.AssetIDFromDigest(feeAssetID)
	isSponsored, err := params.stor.sponsoredAssets.newestIsSponsored(shortFeeAssetID)
	if err != nil {
		return errors.Errorf("newestIsSponsored: %v", err)
	}
	if !isSponsored {
		return errs.NewTxValidationError(fmt.Sprintf("Asset %s is not sponsored, cannot be used to pay fees",
			feeAssetID.String(),
		))
	}
	minWaves, err := minFeeInWaves(tx, params)
	if err != nil {
		return errors.Errorf("failed to calculate min fee in Waves: %v", err)
	}
	minAsset, err := params.stor.sponsoredAssets.wavesToSponsoredAsset(shortFeeAssetID, minWaves.total)
	if err != nil {
		return errors.Errorf("wavesToSponsoredAsset() failed: %v", err)
	}
	fee := tx.GetFee()
	if fee < minAsset {
		feeInfoStr := minWaves.toString()
		return errs.NewFeeValidation(fmt.Sprintf("does not exceed minimal value of 100000 WAVES or %d %s. %s",
			minAsset, feeAssetID, feeInfoStr,
		))
	}
	return nil
}
