package fride

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

func assetInfoToObject(info proto.AssetInfo) rideObject {
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("Asset")
	obj["id"] = rideBytes(info.ID.Bytes())
	obj["quantity"] = rideInt(info.Quantity)
	obj["decimals"] = rideInt(info.Decimals)
	obj["issuer"] = rideAddress(info.Issuer)
	obj["issuerPublicKey"] = rideBytes(common.Dup(info.IssuerPublicKey.Bytes()))
	obj["reissuable"] = rideBoolean(info.Reissuable)
	obj["scripted"] = rideBoolean(info.Scripted)
	obj["sponsored"] = rideBoolean(info.Sponsored)
	return obj
}

func fullAssetInfoToObject(info proto.FullAssetInfo) rideObject {
	obj := assetInfoToObject(info.AssetInfo)
	obj["name"] = rideString(info.Name)
	obj["description"] = rideString(info.Description)
	return obj
}

func blockHeaderToObject(_ byte, _ *proto.BlockHeader, _ proto.Height) (rideObject, error) {
	return nil, errors.New("not implemented")
}

//func transferToObject(scheme byte, tx *proto.Transaction) (rideObject, error) {
//switch t := tx.(type) {
//case *proto.TransferWithProofs:
//	rs, err := transferTonewVariablesFromTransferWithProofs(s.Scheme(), t)
//	if err != nil {
//		return nil, errors.Wrap(err, funcName)
//	}
//	return NewObject(rs), nil
//case *proto.TransferWithSig:
//	rs, err := newVariablesFromTransferWithSig(s.Scheme(), t)
//	if err != nil {
//		return nil, errors.Wrap(err, funcName)
//	}
//	return NewObject(rs), nil
//default:
//	return rideUnit{}, nil
//}

//return nil, errors.New("not implemented")
//}

func genesisToObject(scheme byte, tx *proto.Genesis) (rideObject, error) {
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "paymentToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = rideString("GenesisTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["recipient"] = rideRecipient(proto.NewRecipientFromAddress(tx.Recipient))
	r["amount"] = rideInt(tx.Amount)
	r["fee"] = rideInt(0)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("PaymentTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["recipient"] = rideRecipient(proto.NewRecipientFromAddress(tx.Recipient))
	r["amount"] = rideInt(tx.Amount)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("IssueTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["name"] = rideString(tx.Name)
	r["description"] = rideString(tx.Description)
	r["quantity"] = rideInt(tx.Quantity)
	r["decimals"] = rideInt(tx.Decimals)
	r["reissuable"] = rideBoolean(tx.Reissuable)
	r["script"] = rideUnit{}
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("IssueTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["name"] = rideString(tx.Name)
	r["description"] = rideString(tx.Description)
	r["quantity"] = rideInt(tx.Quantity)
	r["decimals"] = rideInt(tx.Decimals)
	r["reissuable"] = rideBoolean(tx.Reissuable)
	r["script"] = rideUnit{}
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("TransferTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["recipient"] = rideRecipient(tx.Recipient)
	r["assetId"] = optionalAsset(tx.AmountAsset)
	r["amount"] = rideInt(tx.Amount)
	r["fee"] = rideInt(tx.Fee)
	r["feeAssetId"] = optionalAsset(tx.FeeAsset)
	r["attachment"] = rideBytes(tx.Attachment)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("TransferTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["recipient"] = rideRecipient(tx.Recipient)
	r["assetId"] = optionalAsset(tx.AmountAsset)
	r["amount"] = rideInt(tx.Amount)
	r["fee"] = rideInt(tx.Fee)
	r["feeAssetId"] = optionalAsset(tx.FeeAsset)
	r["attachment"] = rideBytes(tx.Attachment)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("ReissueTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = rideBytes(tx.AssetID.Bytes())
	r["quantity"] = rideInt(tx.Quantity)
	r["reissuable"] = rideBoolean(tx.Reissuable)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("ReissueTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = rideBytes(tx.AssetID.Bytes())
	r["quantity"] = rideInt(tx.Quantity)
	r["reissuable"] = rideBoolean(tx.Reissuable)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("BurnTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = rideBytes(tx.AssetID.Bytes())
	r["quantity"] = rideInt(tx.Amount)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("BurnTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = rideBytes(tx.AssetID.Bytes())
	r["quantity"] = rideInt(tx.Amount)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func assetPairToObject(aa, pa proto.OptionalAsset) rideObject {
	r := make(rideObject)
	r[instanceFieldName] = rideString("AssetPair")
	r["amountAsset"] = optionalAsset(aa)
	r["priceAsset"] = optionalAsset(pa)
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
	r[instanceFieldName] = rideString("Order")
	r["id"] = rideBytes(id)
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(senderPK.Bytes()))
	r["matcherPublicKey"] = rideBytes(common.Dup(matcherPk.Bytes()))
	r["assetPair"] = assetPairToObject(pair.AmountAsset, pair.PriceAsset)
	r["orderType"] = orderType(o.GetOrderType())
	r["price"] = rideInt(o.GetPrice())
	r["amount"] = rideInt(o.GetAmount())
	r["timestamp"] = rideInt(o.GetTimestamp())
	r["expiration"] = rideInt(o.GetExpiration())
	r["matcherFee"] = rideInt(o.GetMatcherFee())
	r["matcherFeeAssetId"] = optionalAsset(o.GetMatcherFeeAsset())
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("ExchangeTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(addr)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["buyOrder"] = buy
	r["sellOrder"] = sell
	r["price"] = rideInt(tx.Price)
	r["amount"] = rideInt(tx.Amount)
	r["buyMatcherFee"] = rideInt(tx.BuyMatcherFee)
	r["sellMatcherFee"] = rideInt(tx.SellMatcherFee)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(bts)
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
	r[instanceFieldName] = rideString("ExchangeTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(addr)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["buyOrder"] = buy
	r["sellOrder"] = sell
	r["price"] = rideInt(tx.Price)
	r["amount"] = rideInt(tx.Amount)
	r["buyMatcherFee"] = rideInt(tx.BuyMatcherFee)
	r["sellMatcherFee"] = rideInt(tx.SellMatcherFee)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(bts)
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
	r[instanceFieldName] = rideString("LeaseTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["recipient"] = rideRecipient(tx.Recipient)
	r["amount"] = rideInt(tx.Amount)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("LeaseTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["recipient"] = rideRecipient(tx.Recipient)
	r["amount"] = rideInt(tx.Amount)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("LeaseCancelTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["leaseId"] = rideBytes(tx.LeaseID.Bytes())
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("LeaseCancelTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["leaseId"] = rideBytes(tx.LeaseID.Bytes())
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("CreateAliasTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["alias"] = rideString(tx.Alias.String())
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("CreateAliasTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["alias"] = rideString(tx.Alias.String())
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	transfers := make(rideList, count)
	for i, transfer := range tx.Transfers {
		m := make(rideObject)
		m["recipient"] = rideRecipient(transfer.Recipient)
		m["amount"] = rideInt(transfer.Amount)
		transfers[i] = m
		total += int(transfer.Amount)
	}
	r := make(rideObject)
	r[instanceFieldName] = rideString("MassTransferTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = optionalAsset(tx.Asset)
	r["transfers"] = transfers
	r["transferCount"] = rideInt(count)
	r["totalAmount"] = rideInt(total)
	r["attachment"] = rideBytes(tx.Attachment)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func dataEntryToObject(entry proto.DataEntry) rideType {
	r := make(rideObject)
	r["key"] = rideString(entry.GetKey())
	switch e := entry.(type) {
	case *proto.IntegerDataEntry:
		r["value"] = rideInt(e.Value)
	case *proto.BooleanDataEntry:
		r["value"] = rideBoolean(e.Value)
	case *proto.BinaryDataEntry:
		r["value"] = rideBytes(e.Value)
	case *proto.StringDataEntry:
		r["value"] = rideString(e.Value)
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
		return nil, errors.Wrap(err, "dataWithProofsToObject")
	}
	body, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, "dataWithProofsToObject")
	}
	r := make(rideObject)
	r[instanceFieldName] = rideString("DataTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["data"] = dataEntriesToList(tx.Entries)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("SetScriptTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["script"] = rideUnit{}
	if len(tx.Script) > 0 {
		r["script"] = rideBytes(common.Dup(tx.Script))
	}
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("SponsorFeeTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = rideBytes(tx.AssetID.Bytes())
	r["minSponsoredAssetFee"] = rideUnit{}
	if tx.MinAssetFee > 0 {
		r["minSponsoredAssetFee"] = rideInt(tx.MinAssetFee)
	}
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("SetAssetScriptTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = rideBytes(tx.AssetID.Bytes())
	r["script"] = rideUnit{}
	if len(tx.Script) > 0 {
		r["script"] = rideBytes(common.Dup(tx.Script))
	}
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func attachedPaymentToObject(p proto.ScriptPayment) rideObject {
	r := make(rideObject)
	r[instanceFieldName] = rideString("AttachedPayment")
	r["assetId"] = optionalAsset(p.Asset)
	r["amount"] = rideInt(p.Amount)
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
	var args rideList
	for _, arg := range tx.FunctionCall.Arguments {
		switch t := arg.(type) {
		case *proto.BooleanArgument:
			args = append(args, rideBoolean(t.Value))
		case *proto.IntegerArgument:
			args = append(args, rideInt(t.Value))
		case *proto.StringArgument:
			args = append(args, rideString(t.Value))
		case *proto.BinaryArgument:
			args = append(args, rideBytes(common.Dup(t.Value)))
		default:
			return nil, errors.Errorf("invokeScriptWithProofsToObject: invalid argument type %T", arg)
		}
	}
	r := make(rideObject)
	r[instanceFieldName] = rideString("InvokeScriptTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["dApp"] = rideRecipient(tx.ScriptRecipient)
	r["payment"] = rideUnit{}
	switch {
	case len(tx.Payments) == 1:
		r["payment"] = attachedPaymentToObject(tx.Payments[0])
	case len(tx.Payments) > 1:
		pl := make(rideList, len(tx.Payments))
		for i, p := range tx.Payments {
			pl[i] = attachedPaymentToObject(p)
		}
		r["payments"] = pl
	default:
		r["payment"] = rideUnit{}
	}
	r["feeAssetId"] = optionalAsset(tx.FeeAsset)
	r["function"] = rideString(tx.FunctionCall.Name)
	r["args"] = args
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
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
	r[instanceFieldName] = rideString("UpdateAssetInfoTransaction")
	r["version"] = rideInt(tx.Version)
	r["id"] = rideBytes(tx.ID.Bytes())
	r["sender"] = rideAddress(sender)
	r["senderPublicKey"] = rideBytes(common.Dup(tx.SenderPK.Bytes()))
	r["assetId"] = rideBytes(tx.AssetID.Bytes())
	r["name"] = rideString(tx.Name)
	r["description"] = rideString(tx.Description)
	r["feeAssetId"] = optionalAsset(tx.FeeAsset)
	r["fee"] = rideInt(tx.Fee)
	r["timestamp"] = rideInt(tx.Timestamp)
	r["bodyBytes"] = rideBytes(body)
	r["proofs"] = proofs(tx.Proofs)
	return r, nil
}

func balanceDetailsToObject(_ *proto.FullWavesBalance) (rideObject, error) {
	return nil, errors.New("not implemented")
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
		r[0] = make(rideBytes, 0)
	}
	for i := 1; i < 8; i++ {
		r[i] = make(rideBytes, 0)
	}
	return r
}

func proofs(proofs *proto.ProofsV1) rideList {
	r := make(rideList, 8)
	pl := len(proofs.Proofs)
	for i := 0; i < 8; i++ {
		if i < pl {
			r[i] = rideBytes(common.Dup(proofs.Proofs[i].Bytes()))
			continue
		}
		r[i] = make(rideBytes, 0)
	}
	return r
}
