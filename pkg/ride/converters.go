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
	return rideObject{
		instanceField:       rideString(assetTypeName),
		idField:             rideBytes(info.ID.Bytes()),
		quantityField:       rideInt(info.Quantity),
		decimalsField:       rideInt(info.Decimals),
		issuerField:         rideAddress(info.Issuer),
		issuePublicKeyField: rideBytes(common.Dup(info.IssuerPublicKey.Bytes())),
		reissuableField:     rideBoolean(info.Reissuable),
		scriptedField:       rideBoolean(info.Scripted),
		sponsoredField:      rideBoolean(info.Sponsored),
	}
}

func fullAssetInfoToObject(info *proto.FullAssetInfo) rideObject {
	return rideObject{
		instanceField:        rideString(assetTypeName),
		idField:              rideBytes(info.ID.Bytes()),
		quantityField:        rideInt(info.Quantity),
		decimalsField:        rideInt(info.Decimals),
		issuerField:          rideAddress(info.Issuer),
		issuePublicKeyField:  rideBytes(common.Dup(info.IssuerPublicKey.Bytes())),
		reissuableField:      rideBoolean(info.Reissuable),
		scriptedField:        rideBoolean(info.Scripted),
		sponsoredField:       rideBoolean(info.Sponsored),
		nameField:            rideString(info.Name),
		descriptionField:     rideString(info.Description),
		minSponsoredFeeField: rideInt(info.SponsorshipCost),
	}
}

func blockInfoToObject(info *proto.BlockInfo) rideObject {
	var vrf rideType = rideUnit{}
	if len(info.VRF) > 0 {
		vrf = rideBytes(common.Dup(info.VRF.Bytes()))
	}
	return rideObject{
		instanceField:            rideString(blockInfoTypeName),
		timestampField:           rideInt(info.Timestamp),
		heightField:              rideInt(info.Height),
		baseTargetField:          rideInt(info.BaseTarget),
		generationSignatureField: rideBytes(common.Dup(info.GenerationSignature.Bytes())),
		generatorField:           rideAddress(info.Generator),
		generatorPublicKeyField:  rideBytes(common.Dup(info.GeneratorPublicKey.Bytes())),
		vrfField:                 vrf,
	}
}

func blockHeaderToObject(scheme byte, height proto.Height, header *proto.BlockHeader, vrf []byte) (rideObject, error) {
	address, err := proto.NewAddressFromPublicKey(scheme, header.GenPublicKey)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "blockHeaderToObject")
	}
	var vf rideType = rideUnit{}
	if len(vrf) > 0 {
		vf = rideBytes(common.Dup(vrf))
	}
	return rideObject{
		instanceField:            rideString(blockInfoTypeName),
		timestampField:           rideInt(header.Timestamp),
		heightField:              rideInt(height),
		baseTargetField:          rideInt(header.BaseTarget),
		generationSignatureField: rideBytes(common.Dup(header.GenSignature.Bytes())),
		generatorField:           rideAddress(address),
		generatorPublicKeyField:  rideBytes(common.Dup(header.GenPublicKey.Bytes())),
		vrfField:                 vf,
	}, nil
}

func genesisToObject(scheme byte, tx *proto.Genesis) (rideObject, error) {
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "genesisToObject")
	}
	return rideObject{
		instanceField:  rideString(genesisTransactionTypeName),
		versionField:   rideInt(tx.Version),
		idField:        rideBytes(tx.ID.Bytes()),
		recipientField: rideRecipient(proto.NewRecipientFromAddress(tx.Recipient)),
		amountField:    rideInt(tx.Amount),
		feeField:       rideInt(0),
		timestampField: rideInt(tx.Timestamp),
		bodyBytesField: rideBytes(body),
	}, nil
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
	return rideObject{
		instanceField:        rideString(paymentTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		recipientField:       rideRecipient(proto.NewRecipientFromAddress(tx.Recipient)),
		amountField:          rideInt(tx.Amount),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          signatureToProofs(tx.Signature),
	}, nil
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
	return rideObject{
		instanceField:        rideString(issueTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		nameField:            rideString(tx.Name),
		descriptionField:     rideString(tx.Description),
		quantityField:        rideInt(tx.Quantity),
		decimalsField:        rideInt(tx.Decimals),
		reissuableField:      rideBoolean(tx.Reissuable),
		scriptField:          rideUnit{},
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          signatureToProofs(tx.Signature),
	}, nil
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
	var sf rideType = rideUnit{}
	if tx.NonEmptyScript() {
		sf = rideBytes(common.Dup(tx.Script))
	}
	return rideObject{
		instanceField:        rideString(issueTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		nameField:            rideString(tx.Name),
		descriptionField:     rideString(tx.Description),
		quantityField:        rideInt(tx.Quantity),
		decimalsField:        rideInt(tx.Decimals),
		reissuableField:      rideBoolean(tx.Reissuable),
		scriptField:          sf,
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
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
	return rideObject{
		instanceField:        rideString(transferTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		recipientField:       rideRecipient(tx.Recipient),
		assetIDField:         optionalAsset(tx.AmountAsset),
		amountField:          rideInt(tx.Amount),
		feeField:             rideInt(tx.Fee),
		feeAssetIDField:      optionalAsset(tx.FeeAsset),
		attachmentField:      rideBytes(tx.Attachment),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          signatureToProofs(tx.Signature),
	}, nil
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
	return rideObject{
		instanceField:        rideString(transferTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		recipientField:       rideRecipient(tx.Recipient),
		assetIDField:         optionalAsset(tx.AmountAsset),
		amountField:          rideInt(tx.Amount),
		feeField:             rideInt(tx.Fee),
		feeAssetIDField:      optionalAsset(tx.FeeAsset),
		attachmentField:      rideBytes(tx.Attachment),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
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
	return rideObject{
		instanceField:        rideString(reissueTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		assetIDField:         rideBytes(tx.AssetID.Bytes()),
		quantityField:        rideInt(tx.Quantity),
		reissuableField:      rideBoolean(tx.Reissuable),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          signatureToProofs(tx.Signature),
	}, nil
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
	return rideObject{
		instanceField:        rideString(reissueTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		assetIDField:         rideBytes(tx.AssetID.Bytes()),
		quantityField:        rideInt(tx.Quantity),
		reissuableField:      rideBoolean(tx.Reissuable),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
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
	return rideObject{
		instanceField:        rideString(burnTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		assetIDField:         rideBytes(tx.AssetID.Bytes()),
		quantityField:        rideInt(tx.Amount),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          signatureToProofs(tx.Signature),
	}, nil
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
	return rideObject{
		instanceField:        rideString(burnTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		assetIDField:         rideBytes(tx.AssetID.Bytes()),
		quantityField:        rideInt(tx.Amount),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
}

func assetPairToObject(aa, pa proto.OptionalAsset) rideObject {
	return rideObject{
		instanceField:    rideString(assetPairTypeName),
		amountAssetField: optionalAsset(aa),
		priceAssetField:  optionalAsset(pa),
	}
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
	return rideObject{
		instanceField:          rideString(orderTypeName),
		idField:                rideBytes(id),
		senderField:            rideAddress(senderWavesAddr),
		senderPublicKeyField:   rideBytes(common.Dup(o.GetSenderPKBytes())),
		matcherPublicKeyField:  rideBytes(common.Dup(matcherPk.Bytes())),
		assetPairField:         assetPairToObject(pair.AmountAsset, pair.PriceAsset),
		orderTypeField:         orderType(o.GetOrderType()),
		priceField:             rideInt(o.GetPrice()),
		amountField:            rideInt(o.GetAmount()),
		timestampField:         rideInt(o.GetTimestamp()),
		expirationField:        rideInt(o.GetExpiration()),
		matcherFeeField:        rideInt(o.GetMatcherFee()),
		matcherFeeAssetIDField: optionalAsset(o.GetMatcherFeeAsset()),
		bodyBytesField:         rideBytes(body),
		proofsField:            proofs(p),
	}, nil
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
	return rideObject{
		instanceField:        rideString(exchangeTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(addr),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		buyOrderField:        buy,
		sellOrderField:       sell,
		priceField:           rideInt(tx.Price),
		amountField:          rideInt(tx.Amount),
		buyMatcherFeeField:   rideInt(tx.BuyMatcherFee),
		sellMatcherFeeField:  rideInt(tx.SellMatcherFee),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(bts),
		proofsField:          signatureToProofs(tx.Signature),
	}, nil
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
	return rideObject{
		instanceField:        rideString(exchangeTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(addr),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		buyOrderField:        buy,
		sellOrderField:       sell,
		priceField:           rideInt(tx.Price),
		amountField:          rideInt(tx.Amount),
		buyMatcherFeeField:   rideInt(tx.BuyMatcherFee),
		sellMatcherFeeField:  rideInt(tx.SellMatcherFee),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(bts),
		proofsField:          proofs(tx.Proofs),
	}, nil
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
	return rideObject{
		instanceField:        rideString(leaseTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		recipientField:       rideRecipient(tx.Recipient),
		amountField:          rideInt(tx.Amount),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          signatureToProofs(tx.Signature),
	}, nil
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
	return rideObject{
		instanceField:        rideString(leaseTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		recipientField:       rideRecipient(tx.Recipient),
		amountField:          rideInt(tx.Amount),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
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
	return rideObject{
		instanceField:        rideString(leaseCancelTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		leaseIDField:         rideBytes(tx.LeaseID.Bytes()),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          signatureToProofs(tx.Signature),
	}, nil
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
	return rideObject{
		instanceField:        rideString(leaseCancelTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		leaseIDField:         rideBytes(tx.LeaseID.Bytes()),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
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
	return rideObject{
		instanceField:        rideString(createAliasTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		aliasField:           rideString(tx.Alias.Alias),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          signatureToProofs(tx.Signature),
	}, nil
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
	return rideObject{
		instanceField:        rideString(createAliasTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		aliasField:           rideString(tx.Alias.Alias),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
}

func transferEntryToObject(transferEntry proto.MassTransferEntry) rideObject {
	return rideObject{
		instanceField:  rideString(transferEntryTypeName),
		recipientField: rideRecipient(transferEntry.Recipient),
		amountField:    rideInt(transferEntry.Amount),
	}
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
	var total int64 = 0
	count := len(tx.Transfers)
	transfers := make(rideList, count)
	for i, transfer := range tx.Transfers {
		transfers[i] = transferEntryToObject(transfer)
		total += int64(transfer.Amount)
	}
	return rideObject{
		instanceField:        rideString(massTransferTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		assetIDField:         optionalAsset(tx.Asset),
		transfersField:       transfers,
		transfersCountField:  rideInt(count),
		totalAmountField:     rideInt(total),
		attachmentField:      rideBytes(tx.Attachment),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
}

func dataEntryToObject(entry proto.DataEntry) rideType {
	switch e := entry.(type) {
	case *proto.IntegerDataEntry:
		return rideObject{
			instanceField: rideString(integerEntryTypeName),
			keyField:      rideString(entry.GetKey()),
			valueField:    rideInt(e.Value),
		}
	case *proto.BooleanDataEntry:
		return rideObject{
			instanceField: rideString(booleanEntryTypeName),
			keyField:      rideString(entry.GetKey()),
			valueField:    rideBoolean(e.Value),
		}
	case *proto.BinaryDataEntry:
		return rideObject{
			instanceField: rideString(binaryEntryTypeName),
			keyField:      rideString(entry.GetKey()),
			valueField:    rideBytes(e.Value),
		}
	case *proto.StringDataEntry:
		return rideObject{
			instanceField: rideString(stringEntryTypeName),
			keyField:      rideString(entry.GetKey()),
			valueField:    rideString(e.Value),
		}
	case *proto.DeleteDataEntry:
		return rideObject{
			instanceField: rideString(deleteEntryTypeName),
			keyField:      rideString(entry.GetKey()),
		}
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

func dataWithProofsToObject(scheme byte, tx *proto.DataWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "dataWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "dataWithProofsToObject")
	}
	return rideObject{
		instanceField:        rideString(dataTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		dataField:            dataEntriesToList(tx.Entries),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
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
	var sf rideType = rideUnit{}
	if len(tx.Script) > 0 {
		sf = rideBytes(common.Dup(tx.Script))
	}
	return rideObject{
		instanceField:        rideString(setScriptTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		scriptField:          sf,
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
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
	var f rideType = rideUnit{}
	if tx.MinAssetFee > 0 {
		f = rideInt(tx.MinAssetFee)
	}
	return rideObject{
		instanceField:             rideString(sponsorFeeTransactionTypeName),
		versionField:              rideInt(tx.Version),
		idField:                   rideBytes(tx.ID.Bytes()),
		senderField:               rideAddress(sender),
		senderPublicKeyField:      rideBytes(common.Dup(tx.SenderPK.Bytes())),
		assetIDField:              rideBytes(tx.AssetID.Bytes()),
		minSponsoredAssetFeeField: f,
		feeField:                  rideInt(tx.Fee),
		timestampField:            rideInt(tx.Timestamp),
		bodyBytesField:            rideBytes(body),
		proofsField:               proofs(tx.Proofs),
	}, nil
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
	var sf rideType = rideUnit{}
	if len(tx.Script) > 0 {
		sf = rideBytes(common.Dup(tx.Script))
	}
	return rideObject{
		instanceField:        rideString(setAssetScriptTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		assetIDField:         rideBytes(tx.AssetID.Bytes()),
		scriptField:          sf,
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
}

func attachedPaymentToObject(p proto.ScriptPayment) rideObject {
	return rideObject{
		instanceField: rideString(attachedPaymentTypeName),
		assetIDField:  optionalAsset(p.Asset),
		amountField:   rideInt(p.Amount),
	}
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
	var pf rideType = rideUnit{}
	var psf rideType = rideList{}
	switch {
	case len(tx.Payments) == 1:
		p := attachedPaymentToObject(tx.Payments[0])
		pf = p
		psf = rideList{p}
	case len(tx.Payments) > 1:
		pl := make(rideList, len(tx.Payments))
		for i, p := range tx.Payments {
			pl[i] = attachedPaymentToObject(p)
		}
		psf = pl
	}
	return rideObject{
		instanceField:        rideString(invokeScriptTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		dAppField:            rideRecipient(tx.ScriptRecipient),
		feeAssetIDField:      optionalAsset(tx.FeeAsset),
		functionField:        rideString(tx.FunctionCall.Name),
		argsField:            args,
		paymentField:         pf,
		paymentsField:        psf,
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
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
	return rideObject{
		instanceField:        rideString(invokeExpressionTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		expressionField:      rideBytes(common.Dup(tx.Expression.Bytes())),
		feeAssetIDField:      optionalAsset(tx.FeeAsset),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
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
		return rideObject{
			instanceField:        rideString(transferTransactionTypeName),
			bodyBytesField:       rideBytes(nil),
			proofsField:          proofs(proto.NewProofs()),
			versionField:         rideInt(tx.GetVersion()),
			idField:              rideBytes(tx.ID.Bytes()),
			senderField:          rideAddress(sender),
			senderPublicKeyField: rideBytes(callerPK),
			recipientField:       rideRecipient(proto.NewRecipientFromAddress(*to)),
			assetIDField:         optionalAsset(proto.NewOptionalAssetWaves()),
			amountField:          rideInt(amount),
			feeField:             rideInt(tx.GetFee()),
			feeAssetIDField:      optionalAsset(proto.NewOptionalAssetWaves()),
			attachmentField:      rideBytes(nil),
			timestampField:       rideInt(tx.GetTimestamp()),
		}, nil

	case *proto.EthereumTransferAssetsErc20TxKind:
		recipientAddr, err := proto.EthereumAddress(kind.Arguments.Recipient).ToWavesAddress(scheme)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert ethereum ERC20 transfer recipient to WavesAddress")
		}
		return rideObject{
			instanceField:        rideString(transferTransactionTypeName),
			bodyBytesField:       rideBytes(nil),
			proofsField:          proofs(proto.NewProofs()),
			versionField:         rideInt(tx.GetVersion()),
			idField:              rideBytes(tx.ID.Bytes()),
			senderField:          rideAddress(sender),
			senderPublicKeyField: rideBytes(callerPK),
			recipientField:       rideRecipient(proto.NewRecipientFromAddress(recipientAddr)),
			assetIDField:         optionalAsset(kind.Asset),
			amountField:          rideInt(kind.Arguments.Amount),
			feeField:             rideInt(tx.GetFee()),
			feeAssetIDField:      optionalAsset(proto.NewOptionalAssetWaves()),
			attachmentField:      rideBytes(nil),
			timestampField:       rideInt(tx.GetTimestamp()),
		}, nil

	case *proto.EthereumInvokeScriptTxKind:
		abiPayments := tx.TxKind.DecodedData().Payments
		scriptPayments := make([]proto.ScriptPayment, 0, len(abiPayments))
		for _, p := range abiPayments {
			optAsset := proto.NewOptionalAsset(p.PresentAssetID, p.AssetID)
			payment := proto.ScriptPayment{Amount: uint64(p.Amount), Asset: optAsset}
			scriptPayments = append(scriptPayments, payment)
		}
		var payment rideType = rideUnit{}
		var payments rideType = rideList{}
		switch {
		case len(scriptPayments) == 1:
			payment = attachedPaymentToObject(scriptPayments[0])
			payments = rideList{payment}
		case len(scriptPayments) > 1:
			pl := make(rideList, len(scriptPayments))
			for i, p := range scriptPayments {
				pl[i] = attachedPaymentToObject(p)
			}
			payments = pl
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
		return rideObject{
			instanceField:        rideString(invokeScriptTransactionTypeName),
			bodyBytesField:       rideBytes(nil),
			proofsField:          proofs(proto.NewProofs()),
			versionField:         rideInt(tx.GetVersion()),
			idField:              rideBytes(tx.ID.Bytes()),
			senderField:          rideAddress(sender),
			senderPublicKeyField: rideBytes(callerPK),
			dAppField:            rideRecipient(proto.NewRecipientFromAddress(*to)),
			paymentField:         payment,
			paymentsField:        payments,
			argsField:            args,
			feeAssetIDField:      optionalAsset(proto.NewOptionalAssetWaves()),
			functionField:        rideString(tx.TxKind.DecodedData().Name),
			feeField:             rideInt(tx.GetFee()),
			timestampField:       rideInt(tx.GetTimestamp()),
		}, nil
	default:
		return nil, errors.New("unknown ethereum transaction kind")
	}
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
	return rideObject{
		instanceField:        rideString(updateAssetInfoTransactionTypeName),
		versionField:         rideInt(tx.Version),
		idField:              rideBytes(tx.ID.Bytes()),
		senderField:          rideAddress(sender),
		senderPublicKeyField: rideBytes(common.Dup(tx.SenderPK.Bytes())),
		assetIDField:         rideBytes(tx.AssetID.Bytes()),
		nameField:            rideString(tx.Name),
		descriptionField:     rideString(tx.Description),
		feeAssetIDField:      optionalAsset(tx.FeeAsset),
		feeField:             rideInt(tx.Fee),
		timestampField:       rideInt(tx.Timestamp),
		bodyBytesField:       rideBytes(body),
		proofsField:          proofs(tx.Proofs),
	}, nil
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
		id       crypto.Digest
		feeAsset proto.OptionalAsset
		fee      uint64
		payment  rideType = rideUnit{}
		payments rideType = rideUnit{}
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
		return nil, errors.Errorf("failed to fill invocation object: wrong transaction type (%T)", tx)
	}
	sender, err := proto.NewAddressFromPublicKey(scheme, senderPK)
	if err != nil {
		return nil, err
	}
	callerPK := rideBytes(common.Dup(senderPK.Bytes()))
	var oca rideType = rideUnit{}
	var ock rideType = rideUnit{}
	if rideVersion >= ast.LibV5 {
		oca = rideAddress(sender)
		ock = callerPK
	}
	return rideObject{
		instanceField:              rideString(invocationTypeName),
		transactionIDField:         rideBytes(id.Bytes()),
		callerField:                rideAddress(sender),
		callerPublicKeyField:       callerPK,
		originCallerField:          oca,
		originCallerPublicKeyField: ock,
		paymentField:               payment,
		paymentsField:              payments,
		feeAssetIDField:            optionalAsset(feeAsset),
		feeField:                   rideInt(fee),
	}, nil
}

func ethereumInvocationToObject(rideVersion ast.LibraryVersion, scheme proto.Scheme, tx *proto.EthereumTransaction, scriptPayments []proto.ScriptPayment) (rideObject, error) {
	sender, err := tx.WavesAddressFrom(scheme)
	if err != nil {
		return nil, err
	}
	callerEthereumPK, err := tx.FromPK()
	if err != nil {
		return nil, errors.Errorf("failed to get public key from ethereum transaction %v", err)
	}
	callerPK := rideBytes(callerEthereumPK.SerializeXYCoordinates()) // 64 bytes
	var ocf1 rideType = rideUnit{}
	var ocf2 rideType = rideUnit{}
	if rideVersion >= ast.LibV5 {
		ocf1 = rideAddress(sender)
		ocf2 = callerPK
	}
	var pf rideType = rideUnit{}
	var psf rideType = rideUnit{}
	switch rideVersion {
	case ast.LibV1, ast.LibV2, ast.LibV3:
		if len(scriptPayments) > 0 {
			pf = attachedPaymentToObject(scriptPayments[0])
		}
	default:
		payments := make(rideList, len(scriptPayments))
		for i, p := range scriptPayments {
			payments[i] = attachedPaymentToObject(p)
		}
		psf = payments
	}

	wavesAsset := proto.NewOptionalAssetWaves()
	return rideObject{
		instanceField:              rideString(invocationTypeName),
		transactionIDField:         rideBytes(tx.ID.Bytes()),
		callerField:                rideAddress(sender),
		callerPublicKeyField:       callerPK,
		originCallerField:          ocf1,
		originCallerPublicKeyField: ocf2,
		paymentField:               pf,
		paymentsField:              psf,
		feeAssetIDField:            optionalAsset(wavesAsset),
		feeField:                   rideInt(int64(tx.GetFee())),
	}, nil
}

func scriptTransferToObject(tr *proto.FullScriptTransfer) rideObject {
	return rideObject{
		instanceField:        rideString(scriptTransferTypeName),
		versionField:         rideUnit{},
		idField:              rideBytes(tr.ID.Bytes()),
		senderField:          rideAddress(tr.Sender),
		senderPublicKeyField: rideBytes(common.Dup(tr.SenderPK.Bytes())),
		recipientField:       rideRecipient(tr.Recipient),
		assetField:           optionalAsset(tr.Asset),
		assetIDField:         optionalAsset(tr.Asset),
		amountField:          rideInt(tr.Amount),
		feeAssetIDField:      rideUnit{},
		feeField:             rideUnit{},
		timestampField:       rideInt(tr.Timestamp),
		attachmentField:      rideUnit{},
		bodyBytesField:       rideUnit{},
		proofsField:          rideUnit{},
	}
}

func scriptTransferToTransferTransactionObject(st *proto.FullScriptTransfer) rideObject {
	obj := scriptTransferToObject(st)
	obj[instanceField] = rideString(transferTransactionTypeName)
	return obj
}

func balanceDetailsToObject(fwb *proto.FullWavesBalance) rideObject {
	return rideObject{
		instanceField:   rideString(balanceDetailsTypeName),
		availableField:  rideInt(fwb.Available),
		regularField:    rideInt(fwb.Regular),
		generatingField: rideInt(fwb.Generating),
		effectiveField:  rideInt(fwb.Effective),
	}
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

func scriptActionToObject(scheme byte, action proto.ScriptAction, pk crypto.PublicKey, id crypto.Digest, timestamp uint64) (rideObject, error) {
	address, err := proto.NewAddressFromPublicKey(scheme, pk)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "failed to convert action to object")
	}
	switch a := action.(type) {
	case *proto.ReissueScriptAction:
		return rideObject{
			instanceField:        rideString(reissueTransactionTypeName),
			versionField:         rideInt(0),
			idField:              rideBytes(id.Bytes()),
			senderField:          rideAddress(address),
			senderPublicKeyField: rideBytes(common.Dup(pk.Bytes())),
			assetIDField:         rideBytes(a.AssetID.Bytes()),
			quantityField:        rideInt(a.Quantity),
			reissuableField:      rideBoolean(a.Reissuable),
			feeField:             rideInt(0),
			timestampField:       rideInt(timestamp),
			bodyBytesField:       rideUnit{},
			proofsField:          rideUnit{},
		}, nil
	case *proto.BurnScriptAction:
		return rideObject{
			instanceField:        rideString(burnTransactionTypeName),
			idField:              rideBytes(id.Bytes()),
			versionField:         rideInt(0),
			senderField:          rideAddress(address),
			senderPublicKeyField: rideBytes(common.Dup(pk.Bytes())),
			assetIDField:         rideBytes(a.AssetID.Bytes()),
			quantityField:        rideInt(a.Quantity),
			feeField:             rideInt(0),
			timestampField:       rideInt(timestamp),
			bodyBytesField:       rideUnit{},
			proofsField:          rideUnit{},
		}, nil
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
