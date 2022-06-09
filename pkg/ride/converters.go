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

func transactionToObject(scheme byte, tx proto.Transaction) (rideObject, error) {
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
		return setScriptWithProofsToObject(scheme, transaction)
	case *proto.SponsorshipWithProofs:
		return sponsorshipWithProofsToObject(scheme, transaction)
	case *proto.SetAssetScriptWithProofs:
		return setAssetScriptWithProofsToObject(scheme, transaction)
	case *proto.InvokeScriptWithProofs:
		return invokeScriptWithProofsToObject(scheme, transaction)
	case *proto.UpdateAssetInfoWithProofs:
		return updateAssetInfoWithProofsToObject(scheme, transaction)
	case *proto.EthereumTransaction:
		return ethereumTransactionToObject(scheme, transaction)
	case *proto.InvokeExpressionTransactionWithProofs:
		return invokeExpressionWithProofsToObject(scheme, transaction)
	default:
		return nil, EvaluationFailure.Errorf("conversion to RIDE object is not implemented for %T", transaction)
	}
}

func assetInfoToObject(info *proto.AssetInfo) rideObject {
	obj := make(rideObject)
	obj[instanceField] = rideString(assetTypeName)
	obj[idField] = rideBytes(info.ID.Bytes())
	obj[quantityField] = rideInt(info.Quantity)
	obj[decimalsField] = rideInt(info.Decimals)
	obj[issuerField] = rideAddress(info.Issuer)
	obj[issuePublicKeyField] = rideBytes(common.Dup(info.IssuerPublicKey.Bytes()))
	obj[reissuableField] = rideBoolean(info.Reissuable)
	obj[scriptedField] = rideBoolean(info.Scripted)
	obj[sponsoredField] = rideBoolean(info.Sponsored)
	return obj
}

func fullAssetInfoToObject(info *proto.FullAssetInfo) rideObject {
	obj := assetInfoToObject(&info.AssetInfo)
	obj[nameField] = rideString(info.Name)
	obj[descriptionField] = rideString(info.Description)
	obj[minSponsoredFeeField] = rideInt(info.SponsorshipCost)
	return obj
}

func blockInfoToObject(info *proto.BlockInfo) rideObject {
	r := make(rideObject)
	r[instanceField] = rideString(blockInfoTypeName)
	r[timestampField] = rideInt(info.Timestamp)
	r[heightField] = rideInt(info.Height)
	r[baseTargetField] = rideInt(info.BaseTarget)
	r[generationSignatureField] = rideBytes(common.Dup(info.GenerationSignature.Bytes()))
	r[generatorField] = rideAddress(info.Generator)
	r[generatorPublicKeyField] = rideBytes(common.Dup(info.GeneratorPublicKey.Bytes()))
	r[vrfField] = rideUnit{}
	if len(info.VRF) > 0 {
		r[vrfField] = rideBytes(common.Dup(info.VRF.Bytes()))
	}
	return r
}

func blockHeaderToObject(scheme byte, height proto.Height, header *proto.BlockHeader, vrf []byte) (rideObject, error) {
	address, err := proto.NewAddressFromPublicKey(scheme, header.GenPublicKey)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "blockHeaderToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(blockInfoTypeName)
	r[timestampField] = rideInt(header.Timestamp)
	r[heightField] = rideInt(height)
	r[baseTargetField] = rideInt(header.BaseTarget)
	r[generationSignatureField] = rideBytes(common.Dup(header.GenSignature.Bytes()))
	r[generatorField] = rideAddress(address)
	r[generatorPublicKeyField] = rideBytes(common.Dup(header.GenPublicKey.Bytes()))
	r[vrfField] = rideUnit{}
	if len(vrf) > 0 {
		r[vrfField] = rideBytes(common.Dup(vrf))
	}
	return r, nil
}

func genesisToObject(scheme byte, tx *proto.Genesis) (rideObject, error) {
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "genesisToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(genesisTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[recipientField] = rideRecipient(proto.NewRecipientFromAddress(tx.Recipient))
	r[amountField] = rideInt(tx.Amount)
	r[feeField] = rideInt(0)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	return r, nil
}

func paymentToObject(scheme byte, tx *proto.Payment) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "paymentToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "paymentToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(paymentTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[recipientField] = rideRecipient(proto.NewRecipientFromAddress(tx.Recipient))
	r[amountField] = rideInt(tx.Amount)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = signatureToProofs(tx.Signature)
	return r, nil
}

func issueWithSigToObject(scheme byte, tx *proto.IssueWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "issueWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "issueWithSigToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(issueTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[nameField] = rideString(tx.Name)
	r[descriptionField] = rideString(tx.Description)
	r[quantityField] = rideInt(tx.Quantity)
	r[decimalsField] = rideInt(tx.Decimals)
	r[reissuableField] = rideBoolean(tx.Reissuable)
	r[scriptField] = rideUnit{}
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = signatureToProofs(tx.Signature)
	return r, nil
}

func issueWithProofsToObject(scheme byte, tx *proto.IssueWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "issueWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "issueWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(issueTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[nameField] = rideString(tx.Name)
	r[descriptionField] = rideString(tx.Description)
	r[quantityField] = rideInt(tx.Quantity)
	r[decimalsField] = rideInt(tx.Decimals)
	r[reissuableField] = rideBoolean(tx.Reissuable)
	r[scriptField] = rideUnit{}
	if tx.NonEmptyScript() {
		r[scriptField] = rideBytes(common.Dup(tx.Script))
	}
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func transferWithSigToObject(scheme byte, tx *proto.TransferWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "transferWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "transferWithSigToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(transferTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[recipientField] = rideRecipient(tx.Recipient)
	r[assetIDField] = optionalAsset(tx.AmountAsset)
	r[amountField] = rideInt(tx.Amount)
	r[feeField] = rideInt(tx.Fee)
	r[feeAssetIDField] = optionalAsset(tx.FeeAsset)
	r[attachmentField] = rideBytes(tx.Attachment)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = signatureToProofs(tx.Signature)
	return r, nil
}

func transferWithProofsToObject(scheme byte, tx *proto.TransferWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "transferWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "transferWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(transferTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[recipientField] = rideRecipient(tx.Recipient)
	r[assetIDField] = optionalAsset(tx.AmountAsset)
	r[amountField] = rideInt(tx.Amount)
	r[feeField] = rideInt(tx.Fee)
	r[feeAssetIDField] = optionalAsset(tx.FeeAsset)
	r[attachmentField] = rideBytes(tx.Attachment)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func reissueWithSigToObject(scheme byte, tx *proto.ReissueWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "reissueWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "reissueWithSigToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(reissueTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[assetIDField] = rideBytes(tx.AssetID.Bytes())
	r[quantityField] = rideInt(tx.Quantity)
	r[reissuableField] = rideBoolean(tx.Reissuable)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = signatureToProofs(tx.Signature)
	return r, nil
}

func reissueWithProofsToObject(scheme byte, tx *proto.ReissueWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "reissueWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "reissueWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(reissueTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[assetIDField] = rideBytes(tx.AssetID.Bytes())
	r[quantityField] = rideInt(tx.Quantity)
	r[reissuableField] = rideBoolean(tx.Reissuable)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func burnWithSigToObject(scheme byte, tx *proto.BurnWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "burnWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "burnWithSigToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(burnTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[assetIDField] = rideBytes(tx.AssetID.Bytes())
	r[quantityField] = rideInt(tx.Amount)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = signatureToProofs(tx.Signature)
	return r, nil
}

func burnWithProofsToObject(scheme byte, tx *proto.BurnWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "burnWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "burnWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(burnTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[assetIDField] = rideBytes(tx.AssetID.Bytes())
	r[quantityField] = rideInt(tx.Amount)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func assetPairToObject(aa, pa proto.OptionalAsset) rideObject {
	r := make(rideObject)
	r[instanceField] = rideString(assetPairTypeName)
	r[amountAssetField] = optionalAsset(aa)
	r[priceAssetField] = optionalAsset(pa)
	return r
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

func orderToObject(scheme proto.Scheme, o proto.Order) (rideObject, error) {
	id, err := o.GetID()
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "orderToObject")
	}
	senderAddr, err := o.GetSender(scheme)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "failed to execute 'orderToObject' func, failed to get sender of order")
	}
	// note that in ride we use only proto.WavesAddress addresses
	senderWavesAddr, err := senderAddr.ToWavesAddress(scheme)
	if err != nil {
		return nil, EvaluationFailure.Wrapf(err, "failed to transform (%T) address type to WavesAddress type", senderAddr)
	}
	var body []byte
	// we should leave bodyBytes empty only for proto.EthereumOrderV4
	if _, ok := o.(*proto.EthereumOrderV4); !ok {
		body, err = proto.MarshalOrderBody(scheme, o)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "orderToObject")
		}
	}
	p, err := o.GetProofs()
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "orderToObject")
	}
	matcherPk := o.GetMatcherPK()
	pair := o.GetAssetPair()
	r := make(rideObject)
	r[instanceField] = rideString(orderTypeName)
	r[idField] = rideBytes(id)
	r[senderField] = rideAddress(senderWavesAddr)
	r[senderPublicKeyField] = rideBytes(common.Dup(o.GetSenderPKBytes()))
	r[matcherPublicKeyField] = rideBytes(common.Dup(matcherPk.Bytes()))
	r[assetPairField] = assetPairToObject(pair.AmountAsset, pair.PriceAsset)
	r[orderTypeField] = orderType(o.GetOrderType())
	r[priceField] = rideInt(o.GetPrice())
	r[amountField] = rideInt(o.GetAmount())
	r[timestampField] = rideInt(o.GetTimestamp())
	r[expirationField] = rideInt(o.GetExpiration())
	r[matcherFeeField] = rideInt(o.GetMatcherFee())
	r[matcherFeeAssetIDField] = optionalAsset(o.GetMatcherFeeAsset())
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(p)
	return r, nil
}

func exchangeWithSigToObject(scheme byte, tx *proto.ExchangeWithSig) (rideObject, error) {
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
	r := make(rideObject)
	r[instanceField] = rideString(exchangeTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(addr)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[buyOrderField] = buy
	r[sellOrderField] = sell
	r[priceField] = rideInt(tx.Price)
	r[amountField] = rideInt(tx.Amount)
	r[buyMatcherFeeField] = rideInt(tx.BuyMatcherFee)
	r[sellMatcherFeeField] = rideInt(tx.SellMatcherFee)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(bts)
	r[proofsField] = signatureToProofs(tx.Signature)
	return r, nil
}

func exchangeWithProofsToObject(scheme byte, tx *proto.ExchangeWithProofs) (rideObject, error) {
	buy, err := orderToObject(scheme, tx.Order1)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "exchangeWithProofsToObject")
	}
	sell, err := orderToObject(scheme, tx.Order2)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "exchangeWithProofsToObject")
	}
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "exchangeWithProofsToObject")
	}
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "exchangeWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(exchangeTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(addr)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[buyOrderField] = buy
	r[sellOrderField] = sell
	r[priceField] = rideInt(tx.Price)
	r[amountField] = rideInt(tx.Amount)
	r[buyMatcherFeeField] = rideInt(tx.BuyMatcherFee)
	r[sellMatcherFeeField] = rideInt(tx.SellMatcherFee)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(bts)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func leaseWithSigToObject(scheme byte, tx *proto.LeaseWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseWithSigToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(leaseTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[recipientField] = rideRecipient(tx.Recipient)
	r[amountField] = rideInt(tx.Amount)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = signatureToProofs(tx.Signature)
	return r, nil
}

func leaseWithProofsToObject(scheme byte, tx *proto.LeaseWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(leaseTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[recipientField] = rideRecipient(tx.Recipient)
	r[amountField] = rideInt(tx.Amount)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func leaseCancelWithSigToObject(scheme byte, tx *proto.LeaseCancelWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseCancelWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseCancelWithSigToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(leaseCancelTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[leaseIDField] = rideBytes(tx.LeaseID.Bytes())
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = signatureToProofs(tx.Signature)
	return r, nil
}

func leaseCancelWithProofsToObject(scheme byte, tx *proto.LeaseCancelWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseCancelWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "leaseCancelWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(leaseCancelTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[leaseIDField] = rideBytes(tx.LeaseID.Bytes())
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func createAliasWithSigToObject(scheme byte, tx *proto.CreateAliasWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "createAliasWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "createAliasWithSigToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(createAliasTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[aliasField] = rideString(tx.Alias.Alias)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = signatureToProofs(tx.Signature)
	return r, nil
}

func createAliasWithProofsToObject(scheme byte, tx *proto.CreateAliasWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "createAliasWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "createAliasWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(createAliasTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[aliasField] = rideString(tx.Alias.Alias)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func transferEntryToObject(transferEntry proto.MassTransferEntry) rideObject {
	m := make(rideObject)
	m[instanceField] = rideString(transferEntryTypeName)
	m[recipientField] = rideRecipient(transferEntry.Recipient)
	m[amountField] = rideInt(transferEntry.Amount)
	return m
}

func massTransferWithProofsToObject(scheme byte, tx *proto.MassTransferWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "massTransferWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "massTransferWithProofsToObject")
	}
	total := 0
	count := len(tx.Transfers)
	transfers := make(rideList, count)
	for i, transfer := range tx.Transfers {
		transfers[i] = transferEntryToObject(transfer)
		total += int(transfer.Amount)
	}
	r := make(rideObject)
	r[instanceField] = rideString(massTransferTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[assetIDField] = optionalAsset(tx.Asset)
	r[transfersField] = transfers
	r[transfersCountField] = rideInt(count)
	r[totalAmountField] = rideInt(total)
	r[attachmentField] = rideBytes(tx.Attachment)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func dataEntryToObject(entry proto.DataEntry) rideType {
	r := make(rideObject)
	r[instanceField] = rideString(dataEntryTypeName)
	r[keyField] = rideString(entry.GetKey())
	switch e := entry.(type) {
	case *proto.IntegerDataEntry:
		r[instanceField] = rideString(integerEntryTypeName)
		r[valueField] = rideInt(e.Value)
	case *proto.BooleanDataEntry:
		r[instanceField] = rideString(booleanEntryTypeName)
		r[valueField] = rideBoolean(e.Value)
	case *proto.BinaryDataEntry:
		r[instanceField] = rideString(binaryEntryTypeName)
		r[valueField] = rideBytes(e.Value)
	case *proto.StringDataEntry:
		r[instanceField] = rideString(stringEntryTypeName)
		r[valueField] = rideString(e.Value)
	case *proto.DeleteDataEntry:
		r[instanceField] = rideString(deleteEntryTypeName)
	default:
		return rideUnit{}
	}
	return r
}

func dataEntriesToList(entries []proto.DataEntry) rideList {
	r := make(rideList, len(entries))
	for i, entry := range entries {
		r[i] = dataEntryToObject(entry)
	}
	return r
}

func dataWithProofsToObject(scheme byte, tx *proto.DataWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "dataWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "dataWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(dataTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[dataField] = dataEntriesToList(tx.Entries)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func setScriptWithProofsToObject(scheme byte, tx *proto.SetScriptWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "setScriptWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "setScriptWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(setScriptTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[scriptField] = rideUnit{}
	if len(tx.Script) > 0 {
		r[scriptField] = rideBytes(common.Dup(tx.Script))
	}
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func sponsorshipWithProofsToObject(scheme byte, tx *proto.SponsorshipWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "sponsorshipWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "sponsorshipWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(sponsorFeeTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[assetIDField] = rideBytes(tx.AssetID.Bytes())
	r[minSponsoredAssetFeeField] = rideUnit{}
	if tx.MinAssetFee > 0 {
		r[minSponsoredAssetFeeField] = rideInt(tx.MinAssetFee)
	}
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func setAssetScriptWithProofsToObject(scheme byte, tx *proto.SetAssetScriptWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "setAssetScriptWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "setAssetScriptWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(setAssetScriptTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[assetIDField] = rideBytes(tx.AssetID.Bytes())
	r[scriptField] = rideUnit{}
	if len(tx.Script) > 0 {
		r[scriptField] = rideBytes(common.Dup(tx.Script))
	}
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func attachedPaymentToObject(p proto.ScriptPayment) rideObject {
	r := make(rideObject)
	r[instanceField] = rideString(attachedPaymentTypeName)
	r[assetIDField] = optionalAsset(p.Asset)
	r[amountField] = rideInt(p.Amount)
	return r
}

func invokeScriptWithProofsToObject(scheme byte, tx *proto.InvokeScriptWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "invokeScriptWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "invokeScriptWithProofsToObject")
	}
	args := make(rideList, len(tx.FunctionCall.Arguments))
	for i, arg := range tx.FunctionCall.Arguments {
		a, err := convertArgument(arg)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "invokeScriptWithProofsToObject")
		}
		args[i] = a
	}
	r := make(rideObject)
	r[instanceField] = rideString(invokeScriptTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[dAppField] = rideRecipient(tx.ScriptRecipient)
	switch {
	case len(tx.Payments) == 1:
		p := attachedPaymentToObject(tx.Payments[0])
		r[paymentField] = p
		r[paymentsField] = rideList{p}
	case len(tx.Payments) > 1:
		pl := make(rideList, len(tx.Payments))
		for i, p := range tx.Payments {
			pl[i] = attachedPaymentToObject(p)
		}
		r[paymentsField] = pl
	default:
		r[paymentField] = rideUnit{}
		r[paymentsField] = make(rideList, 0)
	}
	r[feeAssetIDField] = optionalAsset(tx.FeeAsset)
	r[functionField] = rideString(tx.FunctionCall.Name)
	r[argsField] = args
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
}

func invokeExpressionWithProofsToObject(scheme byte, tx *proto.InvokeExpressionTransactionWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "invokeScriptWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "invokeScriptWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(invokeExpressionTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[expressionField] = rideBytes(common.Dup(tx.Expression.Bytes()))
	r[feeAssetIDField] = optionalAsset(tx.FeeAsset)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
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

func ethereumTransactionToObject(scheme proto.Scheme, tx *proto.EthereumTransaction) (rideObject, error) {
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
	r := make(rideObject)

	// TODO check whether we should resolve eth tx kind first
	// we have to fill it according to the spec
	r[bodyBytesField] = rideBytes(nil)
	r[proofsField] = proofs(proto.NewProofs())

	switch kind := tx.TxKind.(type) {
	case *proto.EthereumTransferWavesTxKind:
		r[instanceField] = rideString(transferTransactionTypeName)
		r[versionField] = rideInt(tx.GetVersion())
		r[idField] = rideBytes(tx.ID.Bytes())
		r[senderField] = rideAddress(sender)
		r[senderPublicKeyField] = rideBytes(callerPK)
		r[recipientField] = rideRecipient(proto.NewRecipientFromAddress(*to))
		r[assetIDField] = optionalAsset(proto.NewOptionalAssetWaves())
		res := new(big.Int).Div(tx.Value(), big.NewInt(int64(proto.DiffEthWaves)))
		if ok := res.IsInt64(); !ok {
			return nil, EvaluationFailure.Errorf(
				"transferWithProofsToObject: failed to convert amount from ethereum transaction (big int) to int64. value is %s",
				tx.Value().String())
		}
		amount := res.Int64()
		r[amountField] = rideInt(amount)
		r[feeField] = rideInt(tx.GetFee())
		r[feeAssetIDField] = optionalAsset(proto.NewOptionalAssetWaves())
		r[attachmentField] = rideBytes(nil)
		r[timestampField] = rideInt(tx.GetTimestamp())

	case *proto.EthereumTransferAssetsErc20TxKind:
		r[instanceField] = rideString(transferTransactionTypeName)
		r[versionField] = rideInt(tx.GetVersion())
		r[idField] = rideBytes(tx.ID.Bytes())
		r[senderField] = rideAddress(sender)
		r[senderPublicKeyField] = rideBytes(callerPK)

		recipientAddr, err := proto.EthereumAddress(kind.Arguments.Recipient).ToWavesAddress(scheme)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert ethereum ERC20 transfer recipient to WavesAddress")
		}
		r[recipientField] = rideRecipient(proto.NewRecipientFromAddress(recipientAddr))
		r[assetIDField] = optionalAsset(kind.Asset)
		r[amountField] = rideInt(kind.Arguments.Amount)
		r[feeField] = rideInt(tx.GetFee())
		r[feeAssetIDField] = optionalAsset(proto.NewOptionalAssetWaves())
		r[attachmentField] = rideBytes(nil)
		r[timestampField] = rideInt(tx.GetTimestamp())

	case *proto.EthereumInvokeScriptTxKind:
		r[instanceField] = rideString(invokeScriptTransactionTypeName)
		r[versionField] = rideInt(tx.GetVersion())
		r[idField] = rideBytes(tx.ID.Bytes())
		r[senderField] = rideAddress(sender)
		r[senderPublicKeyField] = rideBytes(callerPK)
		r[dAppField] = rideRecipient(proto.NewRecipientFromAddress(*to))

		abiPayments := tx.TxKind.DecodedData().Payments
		scriptPayments := make([]proto.ScriptPayment, 0, len(abiPayments))
		for _, p := range abiPayments {
			optAsset := proto.NewOptionalAsset(p.PresentAssetID, p.AssetID)
			payment := proto.ScriptPayment{Amount: uint64(p.Amount), Asset: optAsset}
			scriptPayments = append(scriptPayments, payment)
		}

		switch {
		case len(scriptPayments) == 1:

			p := attachedPaymentToObject(scriptPayments[0])
			r[paymentField] = p
			r[paymentsField] = rideList{p}
		case len(scriptPayments) > 1:
			pl := make(rideList, len(scriptPayments))
			for i, p := range scriptPayments {
				pl[i] = attachedPaymentToObject(p)
			}
			r[paymentsField] = pl
		default:
			r[paymentField] = rideUnit{}
			r[paymentsField] = make(rideList, 0)
		}
		r[feeAssetIDField] = optionalAsset(proto.NewOptionalAssetWaves())
		r[functionField] = rideString(tx.TxKind.DecodedData().Name)
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
		r[argsField] = args
		r[feeField] = rideInt(tx.GetFee())
		r[timestampField] = rideInt(tx.GetTimestamp())

	default:
		return nil, errors.New("unknown ethereum transaction kind")
	}
	return r, nil
}

func updateAssetInfoWithProofsToObject(scheme byte, tx *proto.UpdateAssetInfoWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "updateAssetInfoWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "updateAssetInfoWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceField] = rideString(updateAssetInfoTransactionTypeName)
	r[versionField] = rideInt(tx.Version)
	r[idField] = rideBytes(tx.ID.Bytes())
	r[senderField] = rideAddress(sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r[assetIDField] = rideBytes(tx.AssetID.Bytes())
	r[nameField] = rideString(tx.Name)
	r[descriptionField] = rideString(tx.Description)
	r[feeAssetIDField] = optionalAsset(tx.FeeAsset)
	r[feeField] = rideInt(tx.Fee)
	r[timestampField] = rideInt(tx.Timestamp)
	r[bodyBytesField] = rideBytes(body)
	r[proofsField] = proofs(tx.Proofs)
	return r, nil
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

func invocationToObject(rideVersion ast.LibraryVersion, scheme byte, tx proto.Transaction) (rideObject, error) {
	var (
		senderPK crypto.PublicKey
		ID       crypto.Digest
		FeeAsset proto.OptionalAsset
		Fee      uint64
	)
	r := make(rideObject)
	r[instanceField] = rideString(invocationTypeName)

	switch transaction := tx.(type) {
	case *proto.InvokeScriptWithProofs:
		senderPK = transaction.SenderPK
		ID = *transaction.ID
		FeeAsset = transaction.FeeAsset
		Fee = transaction.Fee
		switch rideVersion {
		case 1, 2, 3:
			r[paymentField] = rideUnit{}
			if len(transaction.Payments) > 0 {
				r[paymentField] = attachedPaymentToObject(transaction.Payments[0])
			}
		default:
			payments := make(rideList, len(transaction.Payments))
			for i, p := range transaction.Payments {
				payments[i] = attachedPaymentToObject(p)
			}
			r[paymentsField] = payments
		}
	case *proto.InvokeExpressionTransactionWithProofs:
		senderPK = transaction.SenderPK
		ID = *transaction.ID
		FeeAsset = transaction.FeeAsset
		Fee = transaction.Fee
		r[paymentsField] = nil
	default:
		return nil, errors.Errorf("failed to fill invocation object: wrong transaction type (%T)", tx)
	}
	sender, err := proto.NewAddressFromPublicKey(scheme, senderPK)
	if err != nil {
		return nil, err
	}
	r[transactionIDField] = rideBytes(ID.Bytes())
	r[callerField] = rideAddress(sender)
	callerPK := rideBytes(common.Dup(senderPK.Bytes()))
	r[callerPublicKeyField] = callerPK
	if rideVersion >= ast.LibV5 {
		r[originCallerField] = rideAddress(sender)
		r[originCallerPublicKeyField] = callerPK
	}

	r[feeAssetIDField] = optionalAsset(FeeAsset)
	r[feeField] = rideInt(Fee)
	return r, nil
}

func ethereumInvocationToObject(rideVersion ast.LibraryVersion, scheme proto.Scheme, tx *proto.EthereumTransaction, scriptPayments []proto.ScriptPayment) (rideObject, error) {
	sender, err := tx.WavesAddressFrom(scheme)
	if err != nil {
		return nil, err
	}
	r := make(rideObject)
	r[instanceField] = rideString(invocationTypeName)
	r[transactionIDField] = rideBytes(tx.ID.Bytes())
	r[callerField] = rideAddress(sender)
	callerEthereumPK, err := tx.FromPK()
	if err != nil {
		return nil, errors.Errorf("failed to get public key from ethereum transaction %v", err)
	}
	callerPK := rideBytes(callerEthereumPK.SerializeXYCoordinates()) // 64 bytes
	r[callerPublicKeyField] = callerPK
	if rideVersion >= ast.LibV5 {
		r[originCallerField] = rideAddress(sender)
		r[originCallerPublicKeyField] = callerPK
	}

	switch rideVersion {
	case ast.LibV1, ast.LibV2, ast.LibV3:
		r[paymentField] = rideUnit{}
		if len(scriptPayments) > 0 {
			r[paymentField] = attachedPaymentToObject(scriptPayments[0])
		}
	default:
		payments := make(rideList, len(scriptPayments))
		for i, p := range scriptPayments {
			payments[i] = attachedPaymentToObject(p)
		}
		r[paymentsField] = payments
	}

	wavesAsset := proto.NewOptionalAssetWaves()
	r[feeAssetIDField] = optionalAsset(wavesAsset)
	r[feeField] = rideInt(int64(tx.GetFee()))
	return r, nil
}

func scriptTransferToObject(tr *proto.FullScriptTransfer) rideObject {
	r := make(rideObject)
	r[instanceField] = rideString(scriptTransferTypeName)
	r[versionField] = rideUnit{}
	r[idField] = rideBytes(tr.ID.Bytes())
	r[senderField] = rideAddress(tr.Sender)
	r[senderPublicKeyField] = rideBytes(common.Dup(tr.SenderPK.Bytes()))
	r[recipientField] = rideRecipient(tr.Recipient)
	r[assetField] = optionalAsset(tr.Asset)
	r[assetIDField] = optionalAsset(tr.Asset)
	r[amountField] = rideInt(tr.Amount)
	r[feeAssetIDField] = rideUnit{}
	r[feeField] = rideUnit{}
	r[timestampField] = rideInt(tr.Timestamp)
	r[attachmentField] = rideUnit{}
	r[bodyBytesField] = rideUnit{}
	r[proofsField] = rideUnit{}
	return r
}

func scriptTransferToTransferTransactionObject(st *proto.FullScriptTransfer) rideObject {
	obj := scriptTransferToObject(st)
	obj[instanceField] = rideString(transferTransactionTypeName)
	return obj
}

func balanceDetailsToObject(fwb *proto.FullWavesBalance) rideObject {
	r := make(rideObject)
	r[instanceField] = rideString(balanceDetailsTypeName)
	r[availableField] = rideInt(fwb.Available)
	r[regularField] = rideInt(fwb.Regular)
	r[generatingField] = rideInt(fwb.Generating)
	r[effectiveField] = rideInt(fwb.Effective)
	return r
}

func objectToActions(env environment, obj rideType) ([]proto.ScriptAction, error) {
	switch obj.instanceOf() {
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
	case "ScriptTransfer":
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
	case "SponsorFee":
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

func scriptActionToObject(scheme byte, action proto.ScriptAction, pk crypto.PublicKey, id crypto.Digest, timestamp uint64) (rideObject, error) {
	address, err := proto.NewAddressFromPublicKey(scheme, pk)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "failed to convert action to object")
	}
	r := make(rideObject)
	switch a := action.(type) {
	case *proto.ReissueScriptAction:
		r[instanceField] = rideString(reissueTransactionTypeName)
		r[versionField] = rideInt(0)
		r[idField] = rideBytes(id.Bytes())
		r[senderField] = rideAddress(address)
		r[senderPublicKeyField] = rideBytes(common.Dup(pk.Bytes()))
		r[assetIDField] = rideBytes(a.AssetID.Bytes())
		r[quantityField] = rideInt(a.Quantity)
		r[reissuableField] = rideBoolean(a.Reissuable)
		r[feeField] = rideInt(0)
		r[timestampField] = rideInt(timestamp)
		r[bodyBytesField] = rideUnit{}
		r[proofsField] = rideUnit{}
	case *proto.BurnScriptAction:
		r[instanceField] = rideString(burnTransactionTypeName)
		r[idField] = rideBytes(id.Bytes())
		r[versionField] = rideInt(0)
		r[senderField] = rideAddress(address)
		r[senderPublicKeyField] = rideBytes(common.Dup(pk.Bytes()))
		r[assetIDField] = rideBytes(a.AssetID.Bytes())
		r[quantityField] = rideInt(a.Quantity)
		r[feeField] = rideInt(0)
		r[timestampField] = rideInt(timestamp)
		r[bodyBytesField] = rideUnit{}
		r[proofsField] = rideUnit{}
	default:
		return nil, EvaluationFailure.Errorf("unsupported script action '%T'", action)
	}
	return r, nil
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
