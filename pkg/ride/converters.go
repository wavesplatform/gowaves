package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	default:
		return nil, errors.Errorf("conversion to RIDE object is not implemented for %T", transaction)
	}
}

func assetInfoToObject(info *proto.AssetInfo) rideObject {
	obj := make(rideObject)
	obj[instanceFieldName] = RideString("Asset")
	obj["id"] = RideBytes(info.ID.Bytes())
	obj["quantity"] = RideInt(info.Quantity)
	obj["decimals"] = RideInt(info.Decimals)
	obj["issuer"] = rideAddress(info.Issuer)
	obj["issuerPublicKey"] = RideBytes(common.Dup(info.IssuerPublicKey.Bytes()))
	obj["reissuable"] = RideBoolean(info.Reissuable)
	obj["scripted"] = RideBoolean(info.Scripted)
	obj["sponsored"] = RideBoolean(info.Sponsored)
	return obj
}

func fullAssetInfoToObject(info *proto.FullAssetInfo) rideObject {
	obj := assetInfoToObject(&info.AssetInfo)
	obj["name"] = RideString(info.Name)
	obj["description"] = RideString(info.Description)
	obj["minSponsoredFee"] = RideInt(info.SponsorshipCost)
	return obj
}

func blockInfoToObject(info *proto.BlockInfo) rideObject {
	r := make(rideObject)
	r[instanceFieldName] = RideString("BlockInfo")
	r["timestamp"] = RideInt(info.Timestamp)
	r["height"] = RideInt(info.Height)
	r["baseTarget"] = RideInt(info.BaseTarget)
	r["generationSignature"] = RideBytes(common.Dup(info.GenerationSignature.Bytes()))
	r["generator"] = RideBytes(common.Dup(info.Generator.Bytes()))
	r["generatorPublicKey"] = RideBytes(common.Dup(info.GeneratorPublicKey.Bytes()))
	r["vrf"] = rideUnit{}
	if len(info.VRF) > 0 {
		r["vrf"] = RideBytes(common.Dup(info.VRF.Bytes()))
	}
	return r
}

func blockHeaderToObject(scheme byte, header *proto.BlockHeader, vrf []byte) (rideObject, error) {
	address, err := proto.NewAddressFromPublicKey(scheme, header.GenPublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "blockHeaderToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("BlockInfo")
	r["timestamp"] = RideInt(header.Timestamp)
	r["height"] = RideInt(header.Height)
	r["baseTarget"] = RideInt(header.BaseTarget)
	r["generationSignature"] = RideBytes(common.Dup(header.GenSignature.Bytes()))
	r["generator"] = rideAddress(address)
	r["generatorPublicKey"] = RideBytes(common.Dup(header.GenPublicKey.Bytes()))
	r["vrf"] = rideUnit{}
	if len(vrf) > 0 {
		r["vrf"] = RideBytes(common.Dup(vrf))
	}
	return r, nil
}

func genesisToObject(scheme byte, tx *proto.Genesis) (rideObject, error) {
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "genesisToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("GenesisTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["recipient"] = rideRecipient(proto.NewRecipientFromAddress(tx.Recipient))
	r["amount"] = RideInt(tx.Amount)
	r["fee"] = RideInt(0)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	return r, nil
}

func paymentToObject(scheme byte, tx *proto.Payment) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "paymentToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "paymentToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("PaymentTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["recipient"] = rideRecipient(proto.NewRecipientFromAddress(tx.Recipient))
	r["amount"] = RideInt(tx.Amount)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = signatureToProofs(tx.Signature)
	return r, nil
}

func issueWithSigToObject(scheme byte, tx *proto.IssueWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "issueWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "issueWithSigToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("IssueTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["name"] = RideString(tx.Name)
	r["description"] = RideString(tx.Description)
	r["quantity"] = RideInt(tx.Quantity)
	r["decimals"] = RideInt(tx.Decimals)
	r["reissuable"] = RideBoolean(tx.Reissuable)
	r["script"] = rideUnit{}
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = signatureToProofs(tx.Signature)
	return r, nil
}

func issueWithProofsToObject(scheme byte, tx *proto.IssueWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "issueWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "issueWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("IssueTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["name"] = RideString(tx.Name)
	r["description"] = RideString(tx.Description)
	r["quantity"] = RideInt(tx.Quantity)
	r["decimals"] = RideInt(tx.Decimals)
	r["reissuable"] = RideBoolean(tx.Reissuable)
	r["script"] = rideUnit{}
	if tx.NonEmptyScript() {
		r["script"] = RideBytes(common.Dup(tx.Script))
	}
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func transferWithSigToObject(scheme byte, tx *proto.TransferWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "transferWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "transferWithSigToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("TransferTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["recipient"] = rideRecipient(tx.Recipient)
	r["assetId"] = optionalAsset(tx.AmountAsset)
	r["amount"] = RideInt(tx.Amount)
	r["fee"] = RideInt(tx.Fee)
	r["feeAssetId"] = optionalAsset(tx.FeeAsset)
	r["attachment"] = RideBytes(tx.Attachment)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = signatureToProofs(tx.Signature)
	return r, nil
}

func transferWithProofsToObject(scheme byte, tx *proto.TransferWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "transferWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "transferWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("TransferTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["recipient"] = rideRecipient(tx.Recipient)
	r["assetId"] = optionalAsset(tx.AmountAsset)
	r["amount"] = RideInt(tx.Amount)
	r["fee"] = RideInt(tx.Fee)
	r["feeAssetId"] = optionalAsset(tx.FeeAsset)
	r["attachment"] = RideBytes(tx.Attachment)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func reissueWithSigToObject(scheme byte, tx *proto.ReissueWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "reissueWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "reissueWithSigToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("ReissueTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = RideBytes(tx.AssetID.Bytes())
	r["quantity"] = RideInt(tx.Quantity)
	r["reissuable"] = RideBoolean(tx.Reissuable)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = signatureToProofs(tx.Signature)
	return r, nil
}

func reissueWithProofsToObject(scheme byte, tx *proto.ReissueWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "reissueWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "reissueWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("ReissueTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = RideBytes(tx.AssetID.Bytes())
	r["quantity"] = RideInt(tx.Quantity)
	r["reissuable"] = RideBoolean(tx.Reissuable)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func burnWithSigToObject(scheme byte, tx *proto.BurnWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "burnWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "burnWithSigToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("BurnTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = RideBytes(tx.AssetID.Bytes())
	r["quantity"] = RideInt(tx.Amount)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = signatureToProofs(tx.Signature)
	return r, nil
}

func burnWithProofsToObject(scheme byte, tx *proto.BurnWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "burnWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "burnWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("BurnTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = RideBytes(tx.AssetID.Bytes())
	r["quantity"] = RideInt(tx.Amount)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func assetPairToObject(aa, pa proto.OptionalAsset) rideObject {
	r := make(rideObject)
	r[instanceFieldName] = RideString("AssetPair")
	r["amountAsset"] = optionalAsset(aa)
	r["priceAsset"] = optionalAsset(pa)
	return r
}

func orderType(orderType proto.OrderType) RideType {
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
		return nil, errors.Wrap(err, "orderToObject")
	}
	senderPK := o.GetSenderPK()
	sender, err := proto.NewAddressFromPublicKey(scheme, senderPK)
	if err != nil {
		return nil, errors.Wrap(err, "orderToObject")
	}
	body, err := proto.MarshalOrderBody(scheme, o)
	if err != nil {
		return nil, errors.Wrap(err, "orderToObject")
	}
	p, err := o.GetProofs()
	if err != nil {
		return nil, errors.Wrap(err, "orderToObject")
	}
	matcherPk := o.GetMatcherPK()
	pair := o.GetAssetPair()
	r := make(rideObject)
	r[instanceFieldName] = RideString("Order")
	r["id"] = RideBytes(id)
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(senderPK.Bytes()))
	r["matcherPublicKey"] = RideBytes(common.Dup(matcherPk.Bytes()))
	r["assetPair"] = assetPairToObject(pair.AmountAsset, pair.PriceAsset)
	r["orderType"] = orderType(o.GetOrderType())
	r["price"] = RideInt(o.GetPrice())
	r["amount"] = RideInt(o.GetAmount())
	r["timestamp"] = RideInt(o.GetTimestamp())
	r["expiration"] = RideInt(o.GetExpiration())
	r["matcherFee"] = RideInt(o.GetMatcherFee())
	r["matcherFeeAssetId"] = optionalAsset(o.GetMatcherFeeAsset())
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(p)
	return r, nil
}

func exchangeWithSigToObject(scheme byte, tx *proto.ExchangeWithSig) (rideObject, error) {
	buy, err := orderToObject(scheme, tx.Order1)
	if err != nil {
		return nil, errors.Wrap(err, "exchangeWithSigToObject")
	}
	sell, err := orderToObject(scheme, tx.Order2)
	if err != nil {
		return nil, errors.Wrap(err, "exchangeWithSigToObject")
	}
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "exchangeWithSigToObject")
	}
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "exchangeWithSigToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("ExchangeTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(addr)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["buyOrder"] = buy
	r["sellOrder"] = sell
	r["price"] = RideInt(tx.Price)
	r["amount"] = RideInt(tx.Amount)
	r["buyMatcherFee"] = RideInt(tx.BuyMatcherFee)
	r["sellMatcherFee"] = RideInt(tx.SellMatcherFee)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(bts)
	r["proofs"] = signatureToProofs(tx.Signature)
	return r, nil
}

func exchangeWithProofsToObject(scheme byte, tx *proto.ExchangeWithProofs) (rideObject, error) {
	buy, err := orderToObject(scheme, tx.Order1)
	if err != nil {
		return nil, errors.Wrap(err, "exchangeWithProofsToObject")
	}
	sell, err := orderToObject(scheme, tx.Order2)
	if err != nil {
		return nil, errors.Wrap(err, "exchangeWithProofsToObject")
	}
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "exchangeWithProofsToObject")
	}
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "exchangeWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("ExchangeTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(addr)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["buyOrder"] = buy
	r["sellOrder"] = sell
	r["price"] = RideInt(tx.Price)
	r["amount"] = RideInt(tx.Amount)
	r["buyMatcherFee"] = RideInt(tx.BuyMatcherFee)
	r["sellMatcherFee"] = RideInt(tx.SellMatcherFee)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(bts)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func leaseWithSigToObject(scheme byte, tx *proto.LeaseWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "leaseWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "leaseWithSigToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("LeaseTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["recipient"] = rideRecipient(tx.Recipient)
	r["amount"] = RideInt(tx.Amount)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = signatureToProofs(tx.Signature)
	return r, nil
}

func leaseWithProofsToObject(scheme byte, tx *proto.LeaseWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "leaseWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "leaseWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("LeaseTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["recipient"] = rideRecipient(tx.Recipient)
	r["amount"] = RideInt(tx.Amount)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func leaseCancelWithSigToObject(scheme byte, tx *proto.LeaseCancelWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "leaseCancelWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "leaseCancelWithSigToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("LeaseCancelTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["leaseId"] = RideBytes(tx.LeaseID.Bytes())
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = signatureToProofs(tx.Signature)
	return r, nil
}

func leaseCancelWithProofsToObject(scheme byte, tx *proto.LeaseCancelWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "leaseCancelWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "leaseCancelWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("LeaseCancelTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["leaseId"] = RideBytes(tx.LeaseID.Bytes())
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func createAliasWithSigToObject(scheme byte, tx *proto.CreateAliasWithSig) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "createAliasWithSigToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "createAliasWithSigToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("CreateAliasTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["alias"] = RideString(tx.Alias.String())
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = signatureToProofs(tx.Signature)
	return r, nil
}

func createAliasWithProofsToObject(scheme byte, tx *proto.CreateAliasWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "createAliasWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "createAliasWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("CreateAliasTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["alias"] = RideString(tx.Alias.String())
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func massTransferWithProofsToObject(scheme byte, tx *proto.MassTransferWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "massTransferWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "massTransferWithProofsToObject")
	}
	total := 0
	count := len(tx.Transfers)
	transfers := make(RideList, count)
	for i, transfer := range tx.Transfers {
		m := make(rideObject)
		m["recipient"] = rideRecipient(transfer.Recipient)
		m["amount"] = RideInt(transfer.Amount)
		transfers[i] = m
		total += int(transfer.Amount)
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("MassTransferTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = optionalAsset(tx.Asset)
	r["transfers"] = transfers
	r["transferCount"] = RideInt(count)
	r["totalAmount"] = RideInt(total)
	r["attachment"] = RideBytes(tx.Attachment)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func dataEntryToObject(entry proto.DataEntry) RideType {
	r := make(rideObject)
	r[instanceFieldName] = RideString("DataEntry")
	r["key"] = RideString(entry.GetKey())
	switch e := entry.(type) {
	case *proto.IntegerDataEntry:
		r[instanceFieldName] = RideString("IntegerEntry")
		r["value"] = RideInt(e.Value)
	case *proto.BooleanDataEntry:
		r[instanceFieldName] = RideString("BooleanEntry")
		r["value"] = RideBoolean(e.Value)
	case *proto.BinaryDataEntry:
		r[instanceFieldName] = RideString("BinaryEntry")
		r["value"] = RideBytes(e.Value)
	case *proto.StringDataEntry:
		r[instanceFieldName] = RideString("StringEntry")
		r["value"] = RideString(e.Value)
	default:
		return rideUnit{}
	}
	return r
}

func dataEntriesToList(entries []proto.DataEntry) RideList {
	r := make(RideList, len(entries))
	for i, entry := range entries {
		r[i] = dataEntryToObject(entry)
	}
	return r
}

func dataWithProofsToObject(scheme byte, tx *proto.DataWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "dataWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "dataWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("DataTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["data"] = dataEntriesToList(tx.Entries)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func setScriptWithProofsToObject(scheme byte, tx *proto.SetScriptWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "setScriptWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "setScriptWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("SetScriptTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["script"] = rideUnit{}
	if len(tx.Script) > 0 {
		r["script"] = RideBytes(common.Dup(tx.Script))
	}
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func sponsorshipWithProofsToObject(scheme byte, tx *proto.SponsorshipWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "sponsorshipWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "sponsorshipWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("SponsorFeeTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = RideBytes(tx.AssetID.Bytes())
	r["minSponsoredAssetFee"] = rideUnit{}
	if tx.MinAssetFee > 0 {
		r["minSponsoredAssetFee"] = RideInt(tx.MinAssetFee)
	}
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func setAssetScriptWithProofsToObject(scheme byte, tx *proto.SetAssetScriptWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "setAssetScriptWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "setAssetScriptWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("SetAssetScriptTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = RideBytes(tx.AssetID.Bytes())
	r["script"] = rideUnit{}
	if len(tx.Script) > 0 {
		r["script"] = RideBytes(common.Dup(tx.Script))
	}
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func attachedPaymentToObject(p proto.ScriptPayment) rideObject {
	r := make(rideObject)
	r[instanceFieldName] = RideString("AttachedPayment")
	r["assetId"] = optionalAsset(p.Asset)
	r["amount"] = RideInt(p.Amount)
	return r
}

func invokeScriptWithProofsToObject(scheme byte, tx *proto.InvokeScriptWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "invokeScriptWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "invokeScriptWithProofsToObject")
	}
	args := make(RideList, len(tx.FunctionCall.Arguments))
	for i, arg := range tx.FunctionCall.Arguments {
		a, err := convertArgument(arg)
		if err != nil {
			return nil, errors.Wrap(err, "invokeScriptWithProofsToObject")
		}
		args[i] = a
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("InvokeScriptTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["dApp"] = rideRecipient(tx.ScriptRecipient)
	switch {
	case len(tx.Payments) == 1:
		p := attachedPaymentToObject(tx.Payments[0])
		r["payment"] = p
		r["payments"] = RideList{p}
	case len(tx.Payments) > 1:
		pl := make(RideList, len(tx.Payments))
		for i, p := range tx.Payments {
			pl[i] = attachedPaymentToObject(p)
		}
		r["payments"] = pl
	default:
		r["payment"] = rideUnit{}
		r["payments"] = make(RideList, 0)
	}
	r["feeAssetId"] = optionalAsset(tx.FeeAsset)
	r["function"] = RideString(tx.FunctionCall.Name)
	r["args"] = args
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func updateAssetInfoWithProofsToObject(scheme byte, tx *proto.UpdateAssetInfoWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, "updateAssetInfoWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "updateAssetInfoWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("UpdateAssetInfoTransaction")
	r["version"] = RideInt(tx.Version)
	r["id"] = RideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = RideBytes(tx.AssetID.Bytes())
	r["name"] = RideString(tx.Name)
	r["description"] = RideString(tx.Description)
	r["feeAssetId"] = optionalAsset(tx.FeeAsset)
	r["fee"] = RideInt(tx.Fee)
	r["timestamp"] = RideInt(tx.Timestamp)
	r["bodyBytes"] = RideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func convertArgument(arg proto.Argument) (RideType, error) {
	switch a := arg.(type) {
	case *proto.IntegerArgument:
		return RideInt(a.Value), nil
	case *proto.BooleanArgument:
		return RideBoolean(a.Value), nil
	case *proto.StringArgument:
		return RideString(a.Value), nil
	case *proto.BinaryArgument:
		return RideBytes(a.Value), nil
	case *proto.ListArgument:
		r := make(RideList, len(a.Items))
		for i, item := range a.Items {
			var err error
			r[i], err = convertArgument(item)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert argument")
			}
		}
		return r, nil
	default:
		return nil, errors.Errorf("unknown argument type %T", arg)
	}
}

func invocationToObject(v int, scheme byte, tx *proto.InvokeScriptWithProofs) (rideObject, error) {
	sender, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, err
	}
	r := make(rideObject)
	r[instanceFieldName] = RideString("Invocation")
	r["transactionId"] = RideBytes(tx.ID.Bytes())
	r["caller"] = rideAddress(sender)
	callerPK := RideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["callerPublicKey"] = callerPK
	if v >= 5 {
		r["originCaller"] = rideAddress(sender)
		r["originCallerPublicKey"] = callerPK
	}
	switch v {
	case 4, 5:
		payments := make(RideList, len(tx.Payments))
		for i, p := range tx.Payments {
			payments[i] = attachedPaymentToObject(p)
		}
		r["payments"] = payments
	default:
		r["payment"] = rideUnit{}
		if len(tx.Payments) > 0 {
			r["payment"] = attachedPaymentToObject(tx.Payments[0])
		}
	}
	r["feeAssetId"] = optionalAsset(tx.FeeAsset)
	r["fee"] = RideInt(tx.Fee)
	return r, nil
}

func scriptTransferToObject(tr *proto.FullScriptTransfer) rideObject {
	r := make(rideObject)
	r[instanceFieldName] = RideString("TransferTransaction")
	r["version"] = rideUnit{}
	r["id"] = RideBytes(tr.ID.Bytes())
	r["sender"] = rideAddress(tr.Sender)
	r["senderPublicKey"] = RideBytes(common.Dup(tr.SenderPK.Bytes()))
	r["recipient"] = rideRecipient(tr.Recipient)
	r["assetId"] = optionalAsset(tr.Asset)
	r["amount"] = RideInt(tr.Amount)
	r["feeAssetId"] = rideUnit{}
	r["fee"] = rideUnit{}
	r["timestamp"] = RideInt(tr.Timestamp)
	r["attachment"] = rideUnit{}
	r["bodyBytes"] = rideUnit{}
	r["proofs"] = rideUnit{}
	return r
}

func balanceDetailsToObject(fwb *proto.FullWavesBalance) rideObject {
	r := make(rideObject)
	r[instanceFieldName] = RideString("BalanceDetails")
	r["available"] = RideInt(fwb.Available)
	r["regular"] = RideInt(fwb.Regular)
	r["generating"] = RideInt(fwb.Generating)
	r["effective"] = RideInt(fwb.Effective)
	return r
}

func objectToActions(env Environment, obj RideType) ([]proto.ScriptAction, error) {
	switch obj.instanceOf() {
	case "WriteSet":
		data, err := obj.get("data")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert WriteSet to actions")
		}
		list, ok := data.(RideList)
		if !ok {
			return nil, errors.Errorf("data is not a list")
		}
		res := make([]proto.ScriptAction, len(list))
		for i, entry := range list {
			action, err := convertToAction(env, entry)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to convert item %d of type '%s'", i+1, entry.instanceOf())
			}
			res[i] = action
		}
		return res, nil

	case "TransferSet":
		transfers, err := obj.get("transfers")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferSet to actions")
		}
		list, ok := transfers.(RideList)
		if !ok {
			return nil, errors.Errorf("transfers is not a list")
		}
		res := make([]proto.ScriptAction, len(list))
		for i, transfer := range list {
			action, err := convertToAction(env, transfer)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to convert transfer %d of type '%s'", i+1, transfer.instanceOf())
			}
			res[i] = action
		}
		return res, nil

	case "ScriptResult":
		actions := make([]proto.ScriptAction, 0)
		writes, err := obj.get("writeSet")
		if err != nil {
			return nil, errors.Wrap(err, "ScriptResult has no writes")
		}
		transfers, err := obj.get("transferSet")
		if err != nil {
			return nil, errors.Wrap(err, "ScriptResult has no transfers")
		}
		wa, err := objectToActions(env, writes)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert writes to ScriptActions")
		}
		actions = append(actions, wa...)
		ta, err := objectToActions(env, transfers)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert transfers to ScriptActions")
		}
		actions = append(actions, ta...)
		return actions, nil
	default:
		return nil, errors.Errorf("unexpected type '%s'", obj.instanceOf())
	}
}

func getKeyProperty(v RideType) (string, error) {
	k, err := v.get("key")
	if err != nil {
		return "", err
	}
	key, ok := k.(RideString)
	if !ok {
		return "", errors.Errorf("property is not a String")
	}
	return string(key), nil
}

func convertToAction(env Environment, obj RideType) (proto.ScriptAction, error) {
	switch obj.instanceOf() {
	case "Burn":
		id, err := digestProperty(obj, "assetId")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Burn to ScriptAction")
		}
		quantity, err := intProperty(obj, "quantity")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Burn to ScriptAction")
		}
		return &proto.BurnScriptAction{AssetID: id, Quantity: int64(quantity)}, nil
	case "BinaryEntry":
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert BinaryEntry to ScriptAction")
		}
		b, err := bytesProperty(obj, "value")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert BinaryEntry to ScriptAction")
		}
		return &proto.DataEntryScriptAction{Entry: &proto.BinaryDataEntry{Key: key, Value: b}}, nil
	case "BooleanEntry":
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert BooleanEntry to ScriptAction")
		}
		b, err := booleanProperty(obj, "value")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert BooleanEntry to ScriptAction")
		}
		return &proto.DataEntryScriptAction{Entry: &proto.BooleanDataEntry{Key: key, Value: bool(b)}}, nil
	case "DeleteEntry":
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert DeleteEntry to ScriptAction")
		}
		return &proto.DataEntryScriptAction{Entry: &proto.DeleteDataEntry{Key: key}}, nil
	case "IntegerEntry":
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert IntegerEntry to ScriptAction")
		}
		i, err := intProperty(obj, "value")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert IntegerEntry to ScriptAction")
		}
		return &proto.DataEntryScriptAction{Entry: &proto.IntegerDataEntry{Key: key, Value: int64(i)}}, nil
	case "StringEntry":
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert StringEntry to ScriptAction")
		}
		s, err := stringProperty(obj, "value")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert StringEntry to ScriptAction")
		}
		return &proto.DataEntryScriptAction{Entry: &proto.StringDataEntry{Key: key, Value: string(s)}}, nil
	case "DataEntry":
		key, err := getKeyProperty(obj)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert DataEntry to ScriptAction")
		}
		v, err := obj.get("value")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert DataEntry to ScriptAction")
		}
		switch tv := v.(type) {
		case RideInt:
			return &proto.DataEntryScriptAction{Entry: &proto.IntegerDataEntry{Key: key, Value: int64(tv)}}, nil
		case RideBoolean:
			return &proto.DataEntryScriptAction{Entry: &proto.BooleanDataEntry{Key: key, Value: bool(tv)}}, nil
		case RideString:
			return &proto.DataEntryScriptAction{Entry: &proto.StringDataEntry{Key: key, Value: string(tv)}}, nil
		case RideBytes:
			return &proto.DataEntryScriptAction{Entry: &proto.BinaryDataEntry{Key: key, Value: tv}}, nil
		default:
			return nil, errors.Errorf("unexpected type of DataEntry '%s'", v.instanceOf())
		}
	case "Issue":
		parent := env.txID()
		if parent.instanceOf() == "Unit" {
			return nil, errors.New("empty parent for IssueExpr")
		}
		name, err := stringProperty(obj, "name")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		description, err := stringProperty(obj, "description")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		decimals, err := intProperty(obj, "decimals")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		quantity, err := intProperty(obj, "quantity")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		reissuable, err := booleanProperty(obj, "isReissuable")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		nonce, err := intProperty(obj, "nonce")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		id, err := calcAssetID(env, name, description, decimals, quantity, reissuable, nonce)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Issue to ScriptAction")
		}
		d, err := crypto.NewDigestFromBytes(id)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Issue to ScriptAction")
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
	case "Reissue":
		id, err := digestProperty(obj, "assetId")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Reissue to ScriptAction")
		}
		quantity, err := intProperty(obj, "quantity")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Reissue to ScriptAction")
		}
		reissuable, err := booleanProperty(obj, "isReissuable")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Reissue to ScriptAction")
		}
		return &proto.ReissueScriptAction{
			AssetID:    id,
			Quantity:   int64(quantity),
			Reissuable: bool(reissuable),
		}, nil
	case "ScriptTransfer":
		recipient, err := recipientProperty(obj, "recipient")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert ScriptTransfer to ScriptAction")
		}
		recipient, err = ensureRecipientAddress(env, recipient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert ScriptTransfer to ScriptAction")
		}
		amount, err := intProperty(obj, "amount")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert ScriptTransfer to ScriptAction")
		}
		asset, err := optionalAssetProperty(obj, "asset")
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
		id, err := digestProperty(obj, "assetId")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert SponsorFee to ScriptAction")
		}
		fee, err := intProperty(obj, "minSponsoredAssetFee")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert SponsorFee to ScriptAction")
		}
		return &proto.SponsorshipScriptAction{
			AssetID: id,
			MinFee:  int64(fee),
		}, nil

	case "Lease":
		recipient, err := recipientProperty(obj, "recipient")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		recipient, err = ensureRecipientAddress(env, recipient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		amount, err := intProperty(obj, "amount")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		nonce, err := intProperty(obj, "nonce")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		id, err := calcLeaseID(env, recipient, amount, nonce)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		d, err := crypto.NewDigestFromBytes(id)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert Lease to LeaseScriptAction")
		}
		return &proto.LeaseScriptAction{
			ID:        d,
			Recipient: recipient,
			Amount:    int64(amount),
			Nonce:     int64(nonce),
		}, nil

	case "LeaseCancel":
		id, err := digestProperty(obj, "leaseId")
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert LeaseCancel to LeaseCancelScriptAction")
		}
		return &proto.LeaseCancelScriptAction{
			LeaseID: id,
		}, nil

	default:
		return nil, errors.Errorf("unexpected type '%s'", obj.instanceOf())
	}
}

func scriptActionToObject(scheme byte, action proto.ScriptAction, pk crypto.PublicKey, id crypto.Digest, timestamp uint64) (rideObject, error) {
	address, err := proto.NewAddressFromPublicKey(scheme, pk)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert action to object")
	}
	r := make(rideObject)
	switch a := action.(type) {
	case *proto.ReissueScriptAction:
		r[instanceFieldName] = RideString("ReissueTransaction")
		r["version"] = RideInt(0)
		r["id"] = RideBytes(id.Bytes())
		r["sender"] = rideAddress(address)
		r["senderPublicKey"] = RideBytes(common.Dup(pk.Bytes()))
		r["assetId"] = RideBytes(a.AssetID.Bytes())
		r["quantity"] = RideInt(a.Quantity)
		r["reissuable"] = RideBoolean(a.Reissuable)
		r["fee"] = RideInt(0)
		r["timestamp"] = RideInt(timestamp)
		r["bodyBytes"] = rideUnit{}
		r["proofs"] = rideUnit{}
	case *proto.BurnScriptAction:
		r[instanceFieldName] = RideString("BurnTransaction")
		r["id"] = RideBytes(id.Bytes())
		r["version"] = RideInt(0)
		r["sender"] = rideAddress(address)
		r["senderPublicKey"] = RideBytes(common.Dup(pk.Bytes()))
		r["assetId"] = RideBytes(a.AssetID.Bytes())
		r["quantity"] = RideInt(a.Quantity)
		r["fee"] = RideInt(0)
		r["timestamp"] = RideInt(timestamp)
		r["bodyBytes"] = rideUnit{}
		r["proofs"] = rideUnit{}
	default:
		return nil, errors.Errorf("unsupported script action '%T'", action)
	}
	return r, nil
}

func optionalAsset(o proto.OptionalAsset) RideType {
	if o.Present {
		return RideBytes(o.ID.Bytes())
	}
	return rideUnit{}
}

func signatureToProofs(sig *crypto.Signature) RideList {
	r := make(RideList, 8)
	if sig != nil {
		r[0] = RideBytes(sig.Bytes())
	} else {
		r[0] = RideBytes(nil)
	}
	for i := 1; i < 8; i++ {
		r[i] = RideBytes(nil)
	}
	return r
}

func proofs(proofs *proto.ProofsV1) RideList {
	r := make(RideList, 8)
	pl := len(proofs.Proofs)
	for i := 0; i < 8; i++ {
		if i < pl {
			r[i] = RideBytes(common.Dup(proofs.Proofs[i].Bytes()))
			continue
		}
		r[i] = RideBytes(nil)
	}
	return r
}
