package ride

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

func transactionToObject(ver ast.LibraryVersion, scheme proto.Scheme, consensusImprovementsActivated bool, tx proto.Transaction) (rideType, error) {
	switch transaction := tx.(type) {
	case *proto.Genesis:
		return genesisToObject(scheme, transaction)
	case *proto.Payment:
		return paymentToObject(scheme, transaction)
	case *proto.IssueWithSig:
		return issueWithSigToObject(scheme, transaction)
	case *proto.IssueWithProofs:
		return issueWithProofsToObject(scheme, transaction)
	case *proto.TransferWithSig:
		return transferWithSigToObject(scheme, transaction)
	case *proto.TransferWithProofs:
		return transferWithProofsToObject(scheme, transaction)
	case *proto.ReissueWithSig:
		return reissueWithSigToObject(scheme, transaction)
	case *proto.ReissueWithProofs:
		return reissueWithProofsToObject(scheme, transaction)
	case *proto.BurnWithSig:
		return burnWithSigToObject(scheme, transaction)
	case *proto.BurnWithProofs:
		return burnWithProofsToObject(scheme, transaction)
	case *proto.ExchangeWithSig:
		return exchangeWithSigToObject(scheme, transaction)
	case *proto.ExchangeWithProofs:
		return exchangeWithProofsToObject(scheme, transaction)
	case *proto.LeaseWithSig:
		return leaseWithSigToObject(scheme, transaction)
	case *proto.LeaseWithProofs:
		return leaseWithProofsToObject(scheme, transaction)
	case *proto.LeaseCancelWithSig:
		return leaseCancelWithSigToObject(scheme, transaction)
	case *proto.LeaseCancelWithProofs:
		return leaseCancelWithProofsToObject(scheme, transaction)
	case *proto.CreateAliasWithSig:
		return createAliasWithSigToObject(scheme, transaction)
	case *proto.CreateAliasWithProofs:
		return createAliasWithProofsToObject(scheme, transaction)
	case *proto.MassTransferWithProofs:
		return massTransferWithProofsToObject(scheme, transaction)
	case *proto.DataWithProofs:
		return dataWithProofsToObject(scheme, transaction)
	case *proto.SetScriptWithProofs:
		return setScriptWithProofsToObject(scheme, consensusImprovementsActivated, transaction)
	case *proto.SponsorshipWithProofs:
		return sponsorshipWithProofsToObject(scheme, transaction)
	case *proto.SetAssetScriptWithProofs:
		return setAssetScriptWithProofsToObject(scheme, transaction)
	case *proto.InvokeScriptWithProofs:
		return invokeScriptWithProofsToObject(ver, scheme, transaction)
	case *proto.UpdateAssetInfoWithProofs:
		return updateAssetInfoWithProofsToObject(scheme, transaction)
	case *proto.EthereumTransaction:
		return ethereumTransactionToObject(ver, scheme, transaction)
	case *proto.InvokeExpressionTransactionWithProofs:
		return invokeExpressionWithProofsToObject(scheme, transaction)
	default:
		return nil, EvaluationFailure.Errorf("conversion to RIDE object is not implemented for %T", transaction)
	}
}

func assetInfoToObject(info *proto.AssetInfo) rideType {
	return newRideAssetV3(
		common.Dup(info.IssuerPublicKey.Bytes()),
		info.ID.Bytes(),
		rideInt(info.Quantity),
		rideInt(info.Decimals),
		rideAddress(info.Issuer),
		rideBoolean(info.Scripted),
		rideBoolean(info.Sponsored),
		rideBoolean(info.Reissuable),
	)
}

func fullAssetInfoToObject(info *proto.FullAssetInfo) rideType {
	return newRideAssetV4(
		rideString(info.Description),
		rideString(info.Name),
		common.Dup(info.IssuerPublicKey.Bytes()),
		info.ID.Bytes(),
		rideInt(info.SponsorshipCost),
		rideInt(info.Decimals),
		rideInt(info.Quantity),
		rideAddress(info.Issuer),
		rideBoolean(info.Reissuable),
		rideBoolean(info.Scripted),
	)
}

func blockInfoToObject(info *proto.BlockInfo) rideType {
	var vrf rideType = rideUnit{}
	if len(info.VRF) > 0 {
		vrf = rideBytes(common.Dup(info.VRF.Bytes()))
	}
	return newRideBlockInfoV4(
		vrf,
		common.Dup(info.GenerationSignature.Bytes()),
		common.Dup(info.GeneratorPublicKey.Bytes()),
		rideInt(info.BaseTarget),
		rideInt(info.Timestamp),
		rideInt(info.Height),
		rideAddress(info.Generator),
	)
}

func blockHeaderToObject(scheme byte, height proto.Height, header *proto.BlockHeader, vrf []byte) (rideType, error) {
	address, err := proto.NewAddressFromPublicKey(scheme, header.GeneratorPublicKey)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "blockHeaderToObject")
	}
	var vf rideType = rideUnit{}
	if len(vrf) > 0 {
		vf = rideBytes(common.Dup(vrf))
	}
	return newRideBlockInfoV4(
		vf,
		common.Dup(header.GenSignature.Bytes()),
		common.Dup(header.GeneratorPublicKey.Bytes()),
		rideInt(header.BaseTarget),
		rideInt(header.Timestamp),
		rideInt(height),
		rideAddress(address),
	), nil
}

func genesisToObject(_ byte, tx *proto.Genesis) (rideGenesisTransaction, error) {
	return newRideGenesisTransaction(
		rideAddress(tx.Recipient),
		tx.ID.Bytes(),
		rideInt(tx.Timestamp),
		rideInt(tx.Amount),
		rideInt(tx.Version),
		rideInt(0),
	), nil
}

func paymentToObject(scheme byte, tx *proto.Payment) (ridePaymentTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return ridePaymentTransaction{}, EvaluationFailure.Wrap(err, "paymentToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return ridePaymentTransaction{}, EvaluationFailure.Wrap(err, "paymentToObject")
	}
	return newRidePaymentTransaction(
		signatureToProofs(tx.Signature),
		rideAddress(tx.Recipient),
		body,
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Amount),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func issueWithSigToObject(scheme byte, tx *proto.IssueWithSig) (rideType, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "issueWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "issueWithSigToObject")
	}
	return newRideIssueTransaction(
		signatureToProofs(tx.Signature),
		rideUnit{},
		rideString(tx.Description),
		rideString(tx.Name),
		body,
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideInt(tx.Quantity),
		rideInt(tx.Decimals),
		rideBoolean(tx.Reissuable),
		rideAddress(sender),
	), nil
}

func issueWithProofsToObject(scheme byte, tx *proto.IssueWithProofs) (rideIssueTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideIssueTransaction{}, EvaluationFailure.Wrap(err, "issueWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideIssueTransaction{}, EvaluationFailure.Wrap(err, "issueWithProofsToObject")
	}
	var sf rideType = rideUnit{}
	if tx.NonEmptyScript() {
		sf = rideBytes(common.Dup(tx.Script))
	}
	return newRideIssueTransaction(
		proofs(tx.Proofs),
		sf,
		rideString(tx.Description),
		rideString(tx.Name),
		body,
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideInt(tx.Quantity),
		rideInt(tx.Decimals),
		rideBoolean(tx.Reissuable),
		rideAddress(sender),
	), nil
}

func transferWithSigToObject(scheme byte, tx *proto.TransferWithSig) (rideType, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "transferWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "transferWithSigToObject")
	}
	return newRideTransferTransaction(
		optionalAsset(tx.AmountAsset),
		rideBytes(body),
		optionalAsset(tx.FeeAsset),
		rideInt(tx.Version),
		rideBytes(tx.Attachment),
		signatureToProofs(tx.Signature),
		rideInt(tx.Fee),
		recipientToObject(tx.Recipient),
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Amount),
		rideAddress(sender),
	), nil
}

func transferWithProofsToObject(scheme byte, tx *proto.TransferWithProofs) (rideTransferTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideTransferTransaction{}, EvaluationFailure.Wrap(err, "transferWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideTransferTransaction{}, EvaluationFailure.Wrap(err, "transferWithProofsToObject")
	}
	return newRideTransferTransaction(
		optionalAsset(tx.AmountAsset),
		rideBytes(body),
		optionalAsset(tx.FeeAsset),
		rideInt(tx.Version),
		rideBytes(tx.Attachment),
		proofs(tx.Proofs),
		rideInt(tx.Fee),
		recipientToObject(tx.Recipient),
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Amount),
		rideAddress(sender),
	), nil
}

func reissueWithSigToObject(scheme byte, tx *proto.ReissueWithSig) (rideType, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "reissueWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "reissueWithSigToObject")
	}
	return newRideReissueTransaction(
		rideBytes(body),
		signatureToProofs(tx.Signature),
		common.Dup(tx.SenderPK.Bytes()),
		tx.AssetID.Bytes(),
		tx.ID.Bytes(),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Quantity),
		rideInt(tx.Fee),
		rideAddress(sender),
		rideBoolean(tx.Reissuable),
	), nil
}

func reissueWithProofsToObject(scheme byte, tx *proto.ReissueWithProofs) (rideReissueTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideReissueTransaction{}, EvaluationFailure.Wrap(err, "reissueWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideReissueTransaction{}, EvaluationFailure.Wrap(err, "reissueWithProofsToObject")
	}
	return newRideReissueTransaction(
		rideBytes(body),
		proofs(tx.Proofs),
		common.Dup(tx.SenderPK.Bytes()),
		tx.AssetID.Bytes(),
		tx.ID.Bytes(),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Quantity),
		rideInt(tx.Fee),
		rideAddress(sender),
		rideBoolean(tx.Reissuable),
	), nil
}

func burnWithSigToObject(scheme byte, tx *proto.BurnWithSig) (rideType, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "burnWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "burnWithSigToObject")
	}
	return newRideBurnTransaction(
		rideBytes(body),
		signatureToProofs(tx.Signature),
		common.Dup(tx.SenderPK.Bytes()),
		tx.AssetID.Bytes(),
		tx.ID.Bytes(),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Amount),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func burnWithProofsToObject(scheme byte, tx *proto.BurnWithProofs) (rideBurnTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideBurnTransaction{}, EvaluationFailure.Wrap(err, "burnWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideBurnTransaction{}, EvaluationFailure.Wrap(err, "burnWithProofsToObject")
	}
	return newRideBurnTransaction(
		rideBytes(body),
		proofs(tx.Proofs),
		common.Dup(tx.SenderPK.Bytes()),
		tx.AssetID.Bytes(),
		tx.ID.Bytes(),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Amount),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func assetPairToObject(aa, pa proto.OptionalAsset) rideType {
	return newRideAssetPair(optionalAsset(aa), optionalAsset(pa))
}

func orderType(orderType proto.OrderType) rideType {
	if orderType == proto.Buy {
		return newBuy(nil)
	}
	if orderType == proto.Sell {
		return newSell(nil)
	}
	panic("invalid orderType")
}

func orderToObject(scheme proto.Scheme, o proto.Order) (rideOrder, error) {
	id, err := o.GetID()
	if err != nil {
		return rideOrder{}, EvaluationFailure.Wrap(err, "orderToObject")
	}
	senderAddr, err := o.GetSender(scheme)
	if err != nil {
		return rideOrder{}, EvaluationFailure.Wrap(err, "failed to execute 'orderToObject' func, failed to get sender of order")
	}
	// note that in ride we use only proto.WavesAddress addresses
	senderWavesAddr, err := senderAddr.ToWavesAddress(scheme)
	if err != nil {
		return rideOrder{}, EvaluationFailure.Wrapf(err, "failed to transform (%T) address type to WavesAddress type", senderAddr)
	}
	var body []byte
	// we should leave bodyBytes empty only for proto.EthereumOrderV4
	if _, ok := o.(*proto.EthereumOrderV4); !ok {
		body, err = proto.MarshalOrderBody(scheme, o)
		if err != nil {
			return rideOrder{}, EvaluationFailure.Wrap(err, "orderToObject")
		}
	}
	p, err := o.GetProofs()
	if err != nil {
		return rideOrder{}, EvaluationFailure.Wrap(err, "orderToObject")
	}
	matcherPk := o.GetMatcherPK()
	pair := o.GetAssetPair()
	return newRideOrder(
		assetPairToObject(pair.AmountAsset, pair.PriceAsset),
		orderType(o.GetOrderType()),
		optionalAsset(o.GetMatcherFeeAsset()),
		proofs(p),
		body,
		id,
		common.Dup(o.GetSenderPKBytes()),
		common.Dup(matcherPk.Bytes()),
		rideInt(o.GetAmount()),
		rideInt(o.GetTimestamp()),
		rideInt(o.GetExpiration()),
		rideInt(o.GetMatcherFee()),
		rideInt(o.GetPrice()),
		rideAddress(senderWavesAddr),
	), nil
}

func exchangeWithSigToObject(scheme byte, tx *proto.ExchangeWithSig) (rideType, error) {
	buy, err := orderToObject(scheme, tx.Order1)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "exchangeWithSigToObject")
	}
	sell, err := orderToObject(scheme, tx.Order2)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "exchangeWithSigToObject")
	}
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "exchangeWithSigToObject")
	}
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "exchangeWithSigToObject")
	}
	return newRideExchangeTransaction(
		signatureToProofs(tx.Signature),
		buy,
		sell,
		tx.ID.Bytes(),
		bts,
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Price),
		rideInt(tx.Amount),
		rideInt(tx.Version),
		rideInt(tx.BuyMatcherFee),
		rideInt(tx.Fee),
		rideInt(tx.SellMatcherFee),
		rideAddress(addr),
	), nil
}

func exchangeWithProofsToObject(scheme byte, tx *proto.ExchangeWithProofs) (rideExchangeTransaction, error) {
	buy, err := orderToObject(scheme, tx.Order1)
	if err != nil {
		return rideExchangeTransaction{}, EvaluationFailure.Wrap(err, "exchangeWithProofsToObject")
	}
	sell, err := orderToObject(scheme, tx.Order2)
	if err != nil {
		return rideExchangeTransaction{}, EvaluationFailure.Wrap(err, "exchangeWithProofsToObject")
	}
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideExchangeTransaction{}, EvaluationFailure.Wrap(err, "exchangeWithProofsToObject")
	}
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideExchangeTransaction{}, EvaluationFailure.Wrap(err, "exchangeWithProofsToObject")
	}
	return newRideExchangeTransaction(
		proofs(tx.Proofs),
		buy,
		sell,
		tx.ID.Bytes(),
		bts,
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Price),
		rideInt(tx.Amount),
		rideInt(tx.Version),
		rideInt(tx.BuyMatcherFee),
		rideInt(tx.Fee),
		rideInt(tx.SellMatcherFee),
		rideAddress(addr),
	), nil
}

func leaseWithSigToObject(scheme byte, tx *proto.LeaseWithSig) (rideType, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseWithSigToObject")
	}
	return newRideLeaseTransaction(
		signatureToProofs(tx.Signature),
		recipientToObject(tx.Recipient),
		body,
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Amount),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func leaseWithProofsToObject(scheme byte, tx *proto.LeaseWithProofs) (rideLeaseTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideLeaseTransaction{}, EvaluationFailure.Wrap(err, "leaseWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideLeaseTransaction{}, EvaluationFailure.Wrap(err, "leaseWithProofsToObject")
	}
	return newRideLeaseTransaction(
		proofs(tx.Proofs),
		recipientToObject(tx.Recipient),
		body,
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Amount),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func leaseCancelWithSigToObject(scheme byte, tx *proto.LeaseCancelWithSig) (rideType, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseCancelWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseCancelWithSigToObject")
	}
	return newRideLeaseCancelTransaction(
		signatureToProofs(tx.Signature),
		body,
		common.Dup(tx.SenderPK.Bytes()),
		tx.ID.Bytes(),
		tx.LeaseID.Bytes(),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func leaseCancelWithProofsToObject(scheme byte, tx *proto.LeaseCancelWithProofs) (rideLeaseCancelTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideLeaseCancelTransaction{}, EvaluationFailure.Wrap(err, "leaseCancelWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideLeaseCancelTransaction{}, EvaluationFailure.Wrap(err, "leaseCancelWithProofsToObject")
	}
	return newRideLeaseCancelTransaction(
		proofs(tx.Proofs),
		body,
		common.Dup(tx.SenderPK.Bytes()),
		tx.ID.Bytes(),
		tx.LeaseID.Bytes(),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func createAliasWithSigToObject(scheme byte, tx *proto.CreateAliasWithSig) (rideType, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "createAliasWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "createAliasWithSigToObject")
	}
	return newRideCreateAliasTransaction(
		signatureToProofs(tx.Signature),
		rideString(tx.Alias.Alias),
		body,
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Fee),
		rideInt(tx.Version),
		rideAddress(sender),
	), nil
}

func createAliasWithProofsToObject(scheme byte, tx *proto.CreateAliasWithProofs) (rideCreateAliasTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideCreateAliasTransaction{}, EvaluationFailure.Wrap(err, "createAliasWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideCreateAliasTransaction{}, EvaluationFailure.Wrap(err, "createAliasWithProofsToObject")
	}
	return newRideCreateAliasTransaction(
		proofs(tx.Proofs),
		rideString(tx.Alias.Alias),
		body,
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Fee),
		rideInt(tx.Version),
		rideAddress(sender),
	), nil
}

func transferEntryToObject(transferEntry proto.MassTransferEntry) rideType {
	return newRideTransfer(
		recipientToObject(transferEntry.Recipient),
		rideInt(transferEntry.Amount),
	)
}

func massTransferWithProofsToObject(scheme byte, tx *proto.MassTransferWithProofs) (rideMassTransferTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideMassTransferTransaction{}, EvaluationFailure.Wrap(err, "massTransferWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideMassTransferTransaction{}, EvaluationFailure.Wrap(err, "massTransferWithProofsToObject")
	}
	var total int64 = 0
	count := len(tx.Transfers)
	transfers := make(rideList, count)
	for i, transfer := range tx.Transfers {
		transfers[i] = transferEntryToObject(transfer)
		total += int64(transfer.Amount)
	}
	return newRideMassTransferTransaction(
		proofs(tx.Proofs),
		optionalAsset(tx.Asset),
		body,
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideBytes(tx.Attachment),
		transfers,
		rideInt(count),
		rideInt(tx.Timestamp),
		rideInt(tx.Fee),
		rideInt(total),
		rideInt(tx.Version),
		rideAddress(sender),
	), nil
}

func dataEntryToObject(entry proto.DataEntry) rideType {
	switch e := entry.(type) {
	case *proto.IntegerDataEntry:
		return newRideIntegerEntry(rideString(entry.GetKey()), rideInt(e.Value))
	case *proto.BooleanDataEntry:
		return newRideBooleanEntry(rideString(entry.GetKey()), rideBoolean(e.Value))
	case *proto.BinaryDataEntry:
		return newRideBinaryEntry(rideString(entry.GetKey()), e.Value)
	case *proto.StringDataEntry:
		return newRideStringEntry(rideString(entry.GetKey()), rideString(e.Value))
	case *proto.DeleteDataEntry:
		return newRideDeleteEntry(rideString(entry.GetKey()))
	default:
		return rideUnit{}
	}
}

func dataEntriesToList(entries []proto.DataEntry) rideList {
	r := make(rideList, len(entries))
	for i, entry := range entries {
		r[i] = dataEntryToObject(entry)
	}
	return r
}

func dataWithProofsToObject(scheme byte, tx *proto.DataWithProofs) (rideDataTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideDataTransaction{}, EvaluationFailure.Wrap(err, "dataWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideDataTransaction{}, EvaluationFailure.Wrap(err, "dataWithProofsToObject")
	}
	return newRideDataTransaction(
		proofs(tx.Proofs),
		body,
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		dataEntriesToList(tx.Entries),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func setScriptWithProofsToObject(scheme byte, consensusImprovementsActivated bool, tx *proto.SetScriptWithProofs) (rideSetScriptTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideSetScriptTransaction{}, EvaluationFailure.Wrap(err, "setScriptWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideSetScriptTransaction{}, EvaluationFailure.Wrap(err, "setScriptWithProofsToObject")
	}
	var sf rideType = rideUnit{}
	if l := len(tx.Script); l > 0 && (l <= proto.MaxContractScriptSizeV1V5 || consensusImprovementsActivated) {
		sf = rideBytes(common.Dup(tx.Script))
	}
	return newRideSetScriptTransaction(
		proofs(tx.Proofs),
		sf,
		body,
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func sponsorshipWithProofsToObject(scheme byte, tx *proto.SponsorshipWithProofs) (rideSponsorFeeTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideSponsorFeeTransaction{}, EvaluationFailure.Wrap(err, "sponsorshipWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideSponsorFeeTransaction{}, EvaluationFailure.Wrap(err, "sponsorshipWithProofsToObject")
	}
	var f rideType = rideUnit{}
	if tx.MinAssetFee > 0 {
		f = rideInt(tx.MinAssetFee)
	}
	return newRideSponsorFeeTransaction(
		proofs(tx.Proofs),
		f,
		body,
		tx.AssetID.Bytes(),
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func setAssetScriptWithProofsToObject(scheme byte, tx *proto.SetAssetScriptWithProofs) (rideSetAssetScriptTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideSetAssetScriptTransaction{}, EvaluationFailure.Wrap(err, "setAssetScriptWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideSetAssetScriptTransaction{}, EvaluationFailure.Wrap(err, "setAssetScriptWithProofsToObject")
	}
	var sf rideType = rideUnit{}
	if len(tx.Script) > 0 {
		sf = rideBytes(common.Dup(tx.Script))
	}
	return newRideSetAssetScriptTransaction(
		proofs(tx.Proofs),
		sf,
		body,
		tx.AssetID.Bytes(),
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func attachedPaymentToObject(p proto.ScriptPayment) rideType {
	return newRideAttachedPayment(optionalAsset(p.Asset), rideInt(p.Amount))
}

func invokeScriptWithProofsToObject(ver ast.LibraryVersion, scheme byte, tx *proto.InvokeScriptWithProofs) (rideType, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideUnit{}, EvaluationFailure.Wrap(err, "invokeScriptWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideUnit{}, EvaluationFailure.Wrap(err, "invokeScriptWithProofsToObject")
	}
	args := make(rideList, len(tx.FunctionCall.Arguments))
	for i, arg := range tx.FunctionCall.Arguments {
		a, err := convertArgument(arg)
		if err != nil {
			return rideUnit{}, EvaluationFailure.Wrap(err, "invokeScriptWithProofsToObject")
		}
		args[i] = a
	}
	switch ver {
	case ast.LibV1, ast.LibV2, ast.LibV3:
		var p rideType = rideUnit{}
		if len(tx.Payments) > 0 {
			p = attachedPaymentToObject(tx.Payments[0])
		}
		return newRideInvokeScriptTransactionV3(
			proofs(tx.Proofs),
			optionalAsset(tx.FeeAsset),
			recipientToObject(tx.ScriptRecipient),
			rideString(tx.FunctionCall.Name),
			body,
			tx.ID.Bytes(),
			common.Dup(tx.SenderPK.Bytes()),
			p,
			args,
			rideInt(tx.Timestamp),
			rideInt(tx.Fee),
			rideInt(tx.Version),
			rideAddress(sender),
		), nil
	default:
		pl := make(rideList, len(tx.Payments))
		for i, p := range tx.Payments {
			pl[i] = attachedPaymentToObject(p)
		}
		return newRideInvokeScriptTransactionV4(
			proofs(tx.Proofs),
			optionalAsset(tx.FeeAsset),
			recipientToObject(tx.ScriptRecipient),
			rideString(tx.FunctionCall.Name),
			body,
			tx.ID.Bytes(),
			common.Dup(tx.SenderPK.Bytes()),
			pl,
			args,
			rideInt(tx.Timestamp),
			rideInt(tx.Fee),
			rideInt(tx.Version),
			rideAddress(sender),
		), nil
	}
}

func invokeExpressionWithProofsToObject(scheme byte, tx *proto.InvokeExpressionTransactionWithProofs) (rideInvokeExpressionTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideInvokeExpressionTransaction{}, EvaluationFailure.Wrap(err, "invokeScriptWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideInvokeExpressionTransaction{}, EvaluationFailure.Wrap(err, "invokeScriptWithProofsToObject")
	}
	return newRideInvokeExpressionTransaction(
		proofs(tx.Proofs),
		optionalAsset(tx.FeeAsset),
		body,
		tx.ID.Bytes(),
		common.Dup(tx.Expression.Bytes()),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func ConvertEthereumRideArgumentsToSpecificArgument(decodedArg rideType) (proto.Argument, error) {
	var arg proto.Argument
	switch m := decodedArg.(type) {
	case rideInt:
		arg = &proto.IntegerArgument{Value: int64(m)}
	case rideBoolean:
		arg = &proto.BooleanArgument{Value: bool(m)}
	case rideBytes:
		arg = &proto.BinaryArgument{Value: m}
	case rideString:
		arg = &proto.StringArgument{Value: string(m)}
	case rideList:
		var miniArgs proto.Arguments
		for _, v := range m {
			a, err := ConvertEthereumRideArgumentsToSpecificArgument(v)
			if err != nil {
				return nil, err
			}
			miniArgs = append(miniArgs, a)
		}
		arg = &proto.ListArgument{Items: miniArgs}
	default:
		return nil, EvaluationFailure.New("unknown argument type")
	}

	return arg, nil
}

func ConvertDecodedEthereumArgumentsToProtoArguments(decodedArgs []ethabi.DecodedArg) ([]proto.Argument, error) {
	var arguments []proto.Argument
	for _, decodedArg := range decodedArgs {
		value, err := ethABIDataTypeToRideType(decodedArg.Value)
		if err != nil {
			return nil, EvaluationFailure.Errorf("failed to convert data type to ride type %v", err)
		}
		arg, err := ConvertEthereumRideArgumentsToSpecificArgument(value)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, arg)

	}
	return arguments, nil
}

func ethereumTransactionToObject(ver ast.LibraryVersion, scheme proto.Scheme, tx *proto.EthereumTransaction) (rideType, error) {
	sender, err := tx.WavesAddressFrom(scheme)
	if err != nil {
		return nil, err
	}
	callerEthereumPK, err := tx.FromPK()
	if err != nil {
		return nil, EvaluationFailure.Errorf("failed to get public key from ethereum transaction %v", err)
	}
	callerPK := callerEthereumPK.SerializeXYCoordinates() // 64 bytes
	to, err := tx.WavesAddressTo(scheme)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "ethereumTransactionToObject")
	}

	// TODO: check whether we should resolve eth tx kind first
	// TODO: we have to fill it according to the spec
	switch kind := tx.TxKind.(type) {
	case *proto.EthereumTransferWavesTxKind:
		res := new(big.Int).Div(tx.Value(), big.NewInt(int64(proto.DiffEthWaves)))
		if ok := res.IsInt64(); !ok {
			return nil, EvaluationFailure.Errorf(
				"transferWithProofsToObject: failed to convert amount from ethereum transaction (big int) to int64. value is %s",
				tx.Value().String())
		}
		amount := res.Int64()
		return newRideTransferTransaction(
			optionalAsset(proto.NewOptionalAssetWaves()),
			rideBytes(nil),
			optionalAsset(proto.NewOptionalAssetWaves()),
			rideInt(tx.GetVersion()),
			rideBytes(nil),
			proofs(proto.NewProofs()),
			rideInt(tx.GetFee()),
			rideAddress(*to),
			tx.ID.Bytes(),
			callerPK,
			rideInt(tx.GetTimestamp()),
			rideInt(amount),
			rideAddress(sender),
		), nil

	case *proto.EthereumTransferAssetsErc20TxKind:
		recipientAddr, err := proto.EthereumAddress(kind.Arguments.Recipient).ToWavesAddress(scheme)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert ethereum ERC20 transfer recipient to WavesAddress")
		}
		return newRideTransferTransaction(
			optionalAsset(kind.Asset),
			rideBytes(nil),
			optionalAsset(proto.NewOptionalAssetWaves()),
			rideInt(tx.GetVersion()),
			rideBytes(nil),
			proofs(proto.NewProofs()),
			rideInt(tx.GetFee()),
			rideAddress(recipientAddr),
			tx.ID.Bytes(),
			callerPK,
			rideInt(tx.GetTimestamp()),
			rideInt(kind.Arguments.Amount),
			rideAddress(sender),
		), nil

	case *proto.EthereumInvokeScriptTxKind:
		abiPayments := tx.TxKind.DecodedData().Payments
		scriptPayments := make([]proto.ScriptPayment, 0, len(abiPayments))
		for _, p := range abiPayments {
			optAsset := proto.NewOptionalAsset(p.PresentAssetID, p.AssetID)
			payment := proto.ScriptPayment{Amount: uint64(p.Amount), Asset: optAsset}
			scriptPayments = append(scriptPayments, payment)
		}
		arguments, err := ConvertDecodedEthereumArgumentsToProtoArguments(tx.TxKind.DecodedData().Inputs)
		if err != nil {
			return nil, errors.Errorf("failed to convert ethereum arguments, %v", err)
		}
		args := make(rideList, len(arguments))
		for i, arg := range arguments {
			a, err := convertArgument(arg)
			if err != nil {
				return nil, errors.Wrap(err, "invokeScriptWithProofsToObject")
			}
			args[i] = a
		}
		switch ver {
		case ast.LibV1, ast.LibV2, ast.LibV3:
			var payment rideType = rideUnit{}
			if len(scriptPayments) > 0 {
				payment = attachedPaymentToObject(scriptPayments[0])
			}
			return newRideInvokeScriptTransactionV3(
				proofs(proto.NewProofs()),
				optionalAsset(proto.NewOptionalAssetWaves()),
				rideAddress(*to),
				rideString(tx.TxKind.DecodedData().Name),
				rideBytes(nil),
				tx.ID.Bytes(),
				callerPK,
				payment,
				args,
				rideInt(tx.GetTimestamp()),
				rideInt(tx.GetFee()),
				rideInt(tx.GetVersion()),
				rideAddress(sender),
			), nil
		default:
			var payments = make(rideList, len(scriptPayments))
			for i, p := range scriptPayments {
				payments[i] = attachedPaymentToObject(p)
			}
			return newRideInvokeScriptTransactionV4(
				proofs(proto.NewProofs()),
				optionalAsset(proto.NewOptionalAssetWaves()),
				rideAddress(*to),
				rideString(tx.TxKind.DecodedData().Name),
				rideBytes(nil),
				tx.ID.Bytes(),
				callerPK,
				payments,
				args,
				rideInt(tx.GetTimestamp()),
				rideInt(tx.GetFee()),
				rideInt(tx.GetVersion()),
				rideAddress(sender),
			), nil
		}
	default:
		return nil, errors.New("unknown ethereum transaction kind")
	}
}

func updateAssetInfoWithProofsToObject(scheme byte, tx *proto.UpdateAssetInfoWithProofs) (rideUpdateAssetInfoTransaction, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return rideUpdateAssetInfoTransaction{}, EvaluationFailure.Wrap(err, "updateAssetInfoWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return rideUpdateAssetInfoTransaction{}, EvaluationFailure.Wrap(err, "updateAssetInfoWithProofsToObject")
	}
	return newRideUpdateAssetInfoTransaction(
		proofs(tx.Proofs),
		rideBytes(tx.AssetID.Bytes()),
		rideString(tx.Name),
		rideString(tx.Description),
		body,
		tx.ID.Bytes(),
		common.Dup(tx.SenderPK.Bytes()),
		rideInt(tx.Timestamp),
		rideInt(tx.Version),
		rideInt(tx.Fee),
		rideAddress(sender),
	), nil
}

func convertListArguments(args rideList, check bool) ([]rideType, error) {
	r := make([]rideType, len(args))
	for i, a := range args {
		if check {
			if err := checkArgument(a); err != nil {
				return nil, err
			}
		}
		r[i] = a
	}
	return r, nil
}

func checkArgument(arg rideType) error {
	switch a := arg.(type) {
	case rideInt, rideBoolean, rideString, rideBytes:
		return nil
	case rideList:
		for _, item := range a {
			if err := checkArgument(item); err != nil {
				return err
			}
		}
		return nil
	default:
		return EvaluationFailure.Errorf("invalid argument type '%T'", arg)
	}
}

func convertProtoArguments(args proto.Arguments) ([]rideType, error) {
	r := make([]rideType, len(args))
	var err error
	for i, arg := range args {
		r[i], err = convertArgument(arg)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

func convertArgument(arg proto.Argument) (rideType, error) {
	switch a := arg.(type) {
	case *proto.IntegerArgument:
		return rideInt(a.Value), nil
	case *proto.BooleanArgument:
		return rideBoolean(a.Value), nil
	case *proto.StringArgument:
		return rideString(a.Value), nil
	case *proto.BinaryArgument:
		return rideBytes(a.Value), nil
	case *proto.ListArgument:
		r := make(rideList, len(a.Items))
		var err error
		for i, item := range a.Items {
			r[i], err = convertArgument(item)
			if err != nil {
				return nil, EvaluationFailure.Wrap(err, "failed to convert argument")
			}
		}
		return r, nil
	default:
		return nil, EvaluationFailure.Errorf("unknown argument type %T", arg)
	}
}

func invocationToObject(rideVersion ast.LibraryVersion, scheme byte, tx proto.Transaction) (rideType, error) {
	var (
		senderPK crypto.PublicKey
		id       crypto.Digest
		feeAsset proto.OptionalAsset
		fee      uint64
		payment  rideType = rideUnit{}
		payments          = rideList{}
	)
	switch transaction := tx.(type) {
	case *proto.InvokeScriptWithProofs:
		senderPK = transaction.SenderPK
		id = *transaction.ID
		feeAsset = transaction.FeeAsset
		fee = transaction.Fee
		switch rideVersion {
		case ast.LibV1, ast.LibV2, ast.LibV3:
			if len(transaction.Payments) > 0 {
				payment = attachedPaymentToObject(transaction.Payments[0])
			}
		default:
			ps := make(rideList, len(transaction.Payments))
			for i, p := range transaction.Payments {
				ps[i] = attachedPaymentToObject(p)
			}
			payments = ps
		}
	case *proto.InvokeExpressionTransactionWithProofs:
		senderPK = transaction.SenderPK
		id = *transaction.ID
		feeAsset = transaction.FeeAsset
		fee = transaction.Fee
	default:
		return rideInvocationV5{}, errors.Errorf("failed to fill invocation object: wrong transaction type (%T)", tx)
	}
	sender, err := proto.NewAddressFromPublicKey(scheme, senderPK)
	if err != nil {
		return rideInvocationV5{}, err
	}
	callerPK := rideBytes(common.Dup(senderPK.Bytes()))
	var oca rideType = rideUnit{}
	var ock rideType = rideUnit{}
	if rideVersion >= ast.LibV5 {
		oca = rideAddress(sender)
		ock = callerPK
	}

	switch rideVersion {
	case ast.LibV1, ast.LibV2, ast.LibV3:
		return newRideInvocationV3(
			payment,
			callerPK,
			optionalAsset(feeAsset),
			id.Bytes(),
			rideAddress(sender),
			rideInt(int64(tx.GetFee())),
		), nil
	case ast.LibV4:
		return newRideInvocationV4(
			payments,
			callerPK,
			optionalAsset(feeAsset),
			id.Bytes(),
			rideAddress(sender),
			rideInt(fee),
		), nil
	default:
		return newRideInvocationV5(
			oca,
			payments,
			callerPK,
			optionalAsset(feeAsset),
			ock,
			id.Bytes(),
			rideAddress(sender),
			rideInt(fee),
		), nil
	}
}

func ethereumInvocationToObject(rideVersion ast.LibraryVersion, scheme proto.Scheme, tx *proto.EthereumTransaction, scriptPayments []proto.ScriptPayment) (rideType, error) {
	sender, err := tx.WavesAddressFrom(scheme)
	if err != nil {
		return rideInvocationV5{}, err
	}
	callerEthereumPK, err := tx.FromPK()
	if err != nil {
		return rideInvocationV5{}, errors.Errorf("failed to get public key from ethereum transaction %v", err)
	}
	callerPK := rideBytes(callerEthereumPK.SerializeXYCoordinates()) // 64 bytes
	var ocf1 rideType = rideUnit{}
	var ocf2 rideType = rideUnit{}
	if rideVersion >= ast.LibV5 {
		ocf1 = rideAddress(sender)
		ocf2 = callerPK
	}
	wavesAsset := proto.NewOptionalAssetWaves()
	switch rideVersion {
	case ast.LibV1, ast.LibV2, ast.LibV3:
		var pf rideType = rideUnit{}
		if len(scriptPayments) > 0 {
			pf = attachedPaymentToObject(scriptPayments[0])
		}
		return newRideInvocationV3(
			pf,
			callerPK,
			optionalAsset(wavesAsset),
			tx.ID.Bytes(),
			rideAddress(sender),
			rideInt(int64(tx.GetFee())),
		), nil
	case ast.LibV4:
		payments := make(rideList, len(scriptPayments))
		for i, p := range scriptPayments {
			payments[i] = attachedPaymentToObject(p)
		}
		return newRideInvocationV4(
			payments,
			callerPK,
			optionalAsset(wavesAsset),
			tx.ID.Bytes(),
			rideAddress(sender),
			rideInt(int64(tx.GetFee())),
		), nil
	default:
		payments := make(rideList, len(scriptPayments))
		for i, p := range scriptPayments {
			payments[i] = attachedPaymentToObject(p)
		}
		return newRideInvocationV5(
			ocf1,
			payments,
			callerPK,
			optionalAsset(wavesAsset),
			ocf2,
			tx.ID.Bytes(),
			rideAddress(sender),
			rideInt(int64(tx.GetFee())),
		), nil
	}
}

func recipientToObject(recipient proto.Recipient) rideType {
	if addr := recipient.Address(); addr != nil {
		return rideAddress(*addr)
	}
	if alias := recipient.Alias(); alias != nil {
		return rideAlias(*alias)
	}
	return rideUnit{}
}

func scriptTransferToObject(tr *proto.FullScriptTransfer) rideType {
	return newRideScriptTransfer(
		optionalAsset(tr.Asset),
		recipientToObject(tr.Recipient),
		rideInt(tr.Amount),
	)
}

func scriptTransferToTransferTransactionObject(st *proto.FullScriptTransfer) rideType {
	return newRideTransferTransaction(
		optionalAsset(st.Asset),
		rideUnit{},
		rideUnit{},
		rideUnit{},
		rideUnit{},
		rideList{},
		rideUnit{},
		recipientToObject(st.Recipient),
		st.ID.Bytes(),
		common.Dup(st.SenderPK.Bytes()),
		rideInt(st.Amount),
		rideInt(st.Timestamp),
		rideAddress(st.Sender),
	)
}

func balanceDetailsToObject(fwb *proto.FullWavesBalance) rideType {
	return newRideBalanceDetails(
		rideInt(fwb.Available),
		rideInt(fwb.Regular),
		rideInt(fwb.Generating),
		rideInt(fwb.Effective),
	)
}

func objectToActions(env environment, obj rideType) ([]proto.ScriptAction, error) {
	switch obj.instanceOf() { //TODO: remake with type switch
	case writeSetTypeName:
		data, err := obj.get(dataField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert WriteSet to actions")
		}
		list, ok := data.(rideList)
		if !ok {
			return nil, EvaluationFailure.Errorf("data is not a list")
		}
		res := make([]proto.ScriptAction, len(list))
		for i, entry := range list {
			action, err := convertToAction(env, entry)
			if err != nil {
				return nil, EvaluationFailure.Wrapf(err, "failed to convert item %d of type '%s'", i+1, entry.instanceOf())
			}
			res[i] = action
		}
		return res, nil

	case transferSetTypeName:
		transfers, err := obj.get(transfersField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert TransferSet to actions")
		}
		list, ok := transfers.(rideList)
		if !ok {
			return nil, EvaluationFailure.Errorf("transfers is not a list")
		}
		res := make([]proto.ScriptAction, len(list))
		for i, transfer := range list {
			action, err := convertToAction(env, transfer)
			if err != nil {
				return nil, EvaluationFailure.Wrapf(err, "failed to convert transfer %d of type '%s'", i+1, transfer.instanceOf())
			}
			res[i] = action
		}
		return res, nil

	case scriptResultTypeName:
		actions := make([]proto.ScriptAction, 0)
		writes, err := obj.get(writeSetField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "ScriptResult has no writes")
		}
		transfers, err := obj.get(transferSetField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "ScriptResult has no transfers")
		}
		wa, err := objectToActions(env, writes)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert writes to ScriptActions")
		}
		actions = append(actions, wa...)
		ta, err := objectToActions(env, transfers)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert transfers to ScriptActions")
		}
		actions = append(actions, ta...)
		return actions, nil
	default:
		return nil, EvaluationFailure.Errorf("unexpected type '%s'", obj.instanceOf())
	}
}

func getKeyProperty(v rideType) (string, error) {
	k, err := v.get(keyField)
	if err != nil {
		return "", err
	}
	key, ok := k.(rideString)
	if !ok {
		return "", EvaluationFailure.Errorf("property is not a String")
	}
	return string(key), nil
}

func convertToAction(env environment, obj rideType) (proto.ScriptAction, error) {
	switch obj.instanceOf() {
	case burnTypeName:
		id, err := digestProperty(obj, assetIDField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Burn to ScriptAction")
		}
		quantity, err := intProperty(obj, quantityField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Burn to ScriptAction")
		}
		return &proto.BurnScriptAction{AssetID: id, Quantity: int64(quantity)}, nil
	case binaryEntryTypeName:
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert BinaryEntry to ScriptAction")
		}
		b, err := bytesProperty(obj, valueField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert BinaryEntry to ScriptAction")
		}
		return &proto.DataEntryScriptAction{Entry: &proto.BinaryDataEntry{Key: key, Value: b}}, nil
	case booleanEntryTypeName:
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert BooleanEntry to ScriptAction")
		}
		b, err := booleanProperty(obj, valueField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert BooleanEntry to ScriptAction")
		}
		return &proto.DataEntryScriptAction{Entry: &proto.BooleanDataEntry{Key: key, Value: bool(b)}}, nil
	case deleteEntryTypeName:
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert DeleteEntry to ScriptAction")
		}
		return &proto.DataEntryScriptAction{Entry: &proto.DeleteDataEntry{Key: key}}, nil
	case integerEntryTypeName:
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert IntegerEntry to ScriptAction")
		}
		i, err := intProperty(obj, valueField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert IntegerEntry to ScriptAction")
		}
		return &proto.DataEntryScriptAction{Entry: &proto.IntegerDataEntry{Key: key, Value: int64(i)}}, nil
	case stringEntryTypeName:
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert StringEntry to ScriptAction")
		}
		s, err := stringProperty(obj, valueField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert StringEntry to ScriptAction")
		}
		return &proto.DataEntryScriptAction{Entry: &proto.StringDataEntry{Key: key, Value: string(s)}}, nil
	case dataEntryTypeName:
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert DataEntry to ScriptAction")
		}
		v, err := obj.get(valueField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert DataEntry to ScriptAction")
		}
		switch tv := v.(type) {
		case rideInt:
			return &proto.DataEntryScriptAction{Entry: &proto.IntegerDataEntry{Key: key, Value: int64(tv)}}, nil
		case rideBoolean:
			return &proto.DataEntryScriptAction{Entry: &proto.BooleanDataEntry{Key: key, Value: bool(tv)}}, nil
		case rideString:
			return &proto.DataEntryScriptAction{Entry: &proto.StringDataEntry{Key: key, Value: string(tv)}}, nil
		case rideBytes:
			return &proto.DataEntryScriptAction{Entry: &proto.BinaryDataEntry{Key: key, Value: tv}}, nil
		default:
			return nil, EvaluationFailure.Errorf("unexpected type of DataEntry '%s'", v.instanceOf())
		}
	case issueTypeName:
		parent := env.txID()
		if parent.instanceOf() == unitTypeName {
			return nil, EvaluationFailure.New("empty parent for IssueExpr")
		}
		name, err := stringProperty(obj, nameField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		description, err := stringProperty(obj, descriptionField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		decimals, err := intProperty(obj, decimalsField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		quantity, err := intProperty(obj, quantityField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		reissuable, err := booleanProperty(obj, isReissuableField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		nonce, err := intProperty(obj, nonceField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		id, err := calcAssetID(env, name, description, decimals, quantity, reissuable, nonce)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		d, err := crypto.NewDigestFromBytes(id)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		return &proto.IssueScriptAction{
			ID:          d,
			Name:        string(name),
			Description: string(description),
			Quantity:    int64(quantity),
			Decimals:    int32(decimals),
			Reissuable:  bool(reissuable),
			Script:      nil,
			Nonce:       int64(nonce),
		}, nil
	case reissueTypeName:
		id, err := digestProperty(obj, assetIDField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Reissue to ScriptAction")
		}
		quantity, err := intProperty(obj, quantityField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Reissue to ScriptAction")
		}
		reissuable, err := booleanProperty(obj, isReissuableField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Reissue to ScriptAction")
		}
		return &proto.ReissueScriptAction{
			AssetID:    id,
			Quantity:   int64(quantity),
			Reissuable: bool(reissuable),
		}, nil
	case scriptTransferTypeName:
		recipient, err := recipientProperty(obj, recipientField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert ScriptTransfer to ScriptAction")
		}
		recipient, err = ensureRecipientAddress(env, recipient)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert ScriptTransfer to ScriptAction")
		}
		amount, err := intProperty(obj, amountField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert ScriptTransfer to ScriptAction")
		}
		asset, err := optionalAssetProperty(obj, assetField)
		// On asset ID conversion error we return empty action as in Scala
		// See example on MainNet: transaction (https://wavesexplorer.com/tx/AUpiEr49Jo43Q9zXKkNN23rstiq87hguvhfQqV8ov9uQ)
		// and script (https://wavesexplorer.com/tx/Bp1oieWHWpLz8vBFZui9tY1oDTAKUPTrBAGcwfRe9q5K)
		if err != nil {
			return &proto.TransferScriptAction{
				Recipient: recipient,
				Amount:    0,
				Asset:     proto.NewOptionalAssetWaves(),
			}, nil
		}
		return &proto.TransferScriptAction{
			Recipient: recipient,
			Amount:    int64(amount),
			Asset:     asset,
		}, nil
	case sponsorFeeTypeName:
		id, err := digestProperty(obj, assetIDField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert SponsorFee to ScriptAction")
		}
		fee, err := intProperty(obj, minSponsoredAssetFeeField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert SponsorFee to ScriptAction")
		}
		return &proto.SponsorshipScriptAction{
			AssetID: id,
			MinFee:  int64(fee),
		}, nil

	case leaseTypeName:
		recipient, err := recipientProperty(obj, recipientField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		recipient, err = ensureRecipientAddress(env, recipient)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		amount, err := intProperty(obj, amountField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		nonce, err := intProperty(obj, nonceField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		id, err := calcLeaseID(env, recipient, amount, nonce)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		d, err := crypto.NewDigestFromBytes(id)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		return &proto.LeaseScriptAction{
			ID:        d,
			Recipient: recipient,
			Amount:    int64(amount),
			Nonce:     int64(nonce),
		}, nil

	case leaseCancelTypeName:
		id, err := digestProperty(obj, leaseIDField)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert LeaseCancel to LeaseCancelScriptAction")
		}
		return &proto.LeaseCancelScriptAction{
			LeaseID: id,
		}, nil

	default:
		return nil, EvaluationFailure.Errorf("unexpected type '%s'", obj.instanceOf())
	}
}

func scriptActionToObject(scheme byte, action proto.ScriptAction, pk crypto.PublicKey, id crypto.Digest, timestamp uint64) (rideType, error) {
	address, err := proto.NewAddressFromPublicKey(scheme, pk)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "failed to convert action to object")
	}
	switch a := action.(type) {
	case *proto.ReissueScriptAction:
		return newRideReissueTransaction(
			rideUnit{},
			rideList{},
			common.Dup(pk.Bytes()),
			a.AssetID.Bytes(),
			id.Bytes(),
			rideInt(timestamp),
			rideInt(0),
			rideInt(a.Quantity),
			rideInt(0),
			rideAddress(address),
			rideBoolean(a.Reissuable),
		), nil
	case *proto.BurnScriptAction:
		return newRideBurnTransaction(
			rideUnit{},
			rideList{},
			common.Dup(pk.Bytes()),
			a.AssetID.Bytes(),
			id.Bytes(),
			rideInt(timestamp),
			rideInt(0),
			rideInt(a.Quantity),
			rideInt(0),
			rideAddress(address),
		), nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported script action '%T'", action)
	}
}

func optionalAsset(o proto.OptionalAsset) rideType {
	if o.Present {
		return rideBytes(o.ID.Bytes())
	}
	return rideUnit{}
}

func signatureToProofs(sig *crypto.Signature) rideList {
	r := make(rideList, 8)
	if sig != nil {
		r[0] = rideBytes(sig.Bytes())
	} else {
		r[0] = rideBytes(nil)
	}
	for i := 1; i < 8; i++ {
		r[i] = rideBytes(nil)
	}
	return r
}

func proofs(proofs *proto.ProofsV1) rideList {
	r := make(rideList, 8)
	proofsLen := len(proofs.Proofs)
	for i := range r {
		if i < proofsLen {
			r[i] = rideBytes(common.Dup(proofs.Proofs[i].Bytes()))
		} else {
			r[i] = rideBytes(nil)
		}
	}
	return r
}
