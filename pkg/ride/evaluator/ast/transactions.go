package ast

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

func NewVariablesFromScriptTransfer(tx *proto.FullScriptTransfer) (map[string]Expr, error) {
	out := make(map[string]Expr)
	out["amount"] = NewLong(int64(tx.Amount))
	out["assetId"] = makeOptionalAsset(tx.Asset)
	out["recipient"] = NewRecipientFromProtoRecipient(tx.Recipient)
	out["id"] = NewBytes(common.Dup(tx.ID.Bytes()))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["sender"] = NewAddressFromProtoAddress(tx.Sender)
	out[InstanceFieldName] = NewString("TransferTransaction")

	out["senderPublicKey"] = NewUnit()
	out["feeAssetId"] = NewUnit()
	out["attachment"] = NewUnit()
	out["fee"] = NewUnit()
	out["version"] = NewUnit()
	out["bodyBytes"] = NewUnit()
	out["proofs"] = NewUnit()

	return out, nil
}

func NewVariablesFromTransaction(scheme byte, t proto.Transaction) (map[string]Expr, error) {
	switch tx := t.(type) {
	case *proto.Genesis:
		return newVariableFromGenesis(scheme, tx)
	case *proto.Payment:
		return newVariablesFromPayment(scheme, tx)
	case *proto.TransferWithSig:
		return newVariablesFromTransferWithSig(scheme, tx)
	case *proto.TransferWithProofs:
		return newVariablesFromTransferWithProofs(scheme, tx)
	case *proto.ReissueWithSig:
		return newVariablesFromReissueWithSig(scheme, tx)
	case *proto.ReissueWithProofs:
		return newVariablesFromReissueWithProofs(scheme, tx)
	case *proto.BurnWithSig:
		return newVariablesFromBurnWithSig(scheme, tx)
	case *proto.BurnWithProofs:
		return newVariablesFromBurnWithProofs(scheme, tx)
	case *proto.MassTransferWithProofs:
		return newVariablesFromMassTransferWithProofs(scheme, tx)
	case *proto.ExchangeWithSig:
		return newVariablesFromExchangeWithSig(scheme, tx)
	case *proto.ExchangeWithProofs:
		return newVariablesFromExchangeWithProofs(scheme, tx)
	case *proto.SetAssetScriptWithProofs:
		return newVariablesFromSetAssetScriptWithProofs(scheme, tx)
	case *proto.InvokeScriptWithProofs:
		return newVariablesFromInvokeScriptWithProofs(scheme, tx)
	case *proto.IssueWithSig:
		return newVariablesFromIssueWithSig(scheme, tx)
	case *proto.IssueWithProofs:
		return newVariablesFromIssueWithProofs(scheme, tx)
	case *proto.LeaseWithSig:
		return newVariablesFromLeaseWithSig(scheme, tx)
	case *proto.LeaseWithProofs:
		return newVariablesFromLeaseWithProofs(scheme, tx)
	case *proto.LeaseCancelWithSig:
		return newVariablesFromLeaseCancelWithSig(scheme, tx)
	case *proto.LeaseCancelWithProofs:
		return newVariablesFromLeaseCancelWithProofs(scheme, tx)
	case *proto.DataWithProofs:
		return newVariablesFromDataWithProofs(scheme, tx)
	case *proto.SponsorshipWithProofs:
		return newVariablesFromSponsorshipWithProofs(scheme, tx)
	case *proto.CreateAliasWithSig:
		return newVariablesFromCreateAliasWithSig(scheme, tx)
	case *proto.CreateAliasWithProofs:
		return newVariablesFromCreateAliasWithProofs(scheme, tx)
	case *proto.SetScriptWithProofs:
		return newVariablesFromSetScriptWithProofs(scheme, tx)
	default:
		return nil, errors.Errorf("NewVariablesFromTransaction not implemented for %T", tx)
	}
}

func NewVariablesFromScriptAction(scheme proto.Scheme, action proto.ScriptAction, invokerPK crypto.PublicKey, txID crypto.Digest, txTimestamp uint64) (map[string]Expr, error) {
	out := make(map[string]Expr)
	switch a := action.(type) {
	case proto.ReissueScriptAction:
		out["quantity"] = NewLong(a.Quantity)
		out["assetId"] = NewBytes(a.AssetID.Bytes())
		out["reissuable"] = NewBoolean(a.Reissuable)
		out["id"] = NewBytes(txID.Bytes())
		out["fee"] = NewLong(0)
		out["timestamp"] = NewLong(int64(txTimestamp))
		out["version"] = NewLong(0)
		addr, err := proto.NewAddressFromPublicKey(scheme, invokerPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert action to object")
		}
		out["sender"] = NewAddressFromProtoAddress(addr)
		out["senderPublicKey"] = NewBytes(common.Dup(invokerPK.Bytes()))
		out["bodyBytes"] = NewUnit()
		out["proofs"] = NewUnit()
		out[InstanceFieldName] = NewString("ReissueTransaction")
	case proto.BurnScriptAction:
		out["quantity"] = NewLong(a.Quantity)
		out["assetId"] = NewBytes(a.AssetID.Bytes())
		out["id"] = NewBytes(txID.Bytes())
		out["fee"] = NewLong(0)
		out["timestamp"] = NewLong(int64(txTimestamp))
		out["version"] = NewLong(0)
		addr, err := proto.NewAddressFromPublicKey(scheme, invokerPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert action to object")
		}
		out["sender"] = NewAddressFromProtoAddress(addr)
		out["senderPublicKey"] = NewBytes(common.Dup(invokerPK.Bytes()))
		out["bodyBytes"] = NewUnit()
		out["proofs"] = NewUnit()
		out[InstanceFieldName] = NewString("BurnTransaction")
		return out, nil
	default:
		return nil, errors.New("unsupported script action")
	}
	return out, nil
}

func NewVariablesFromOrder(scheme proto.Scheme, tx proto.Order) (map[string]Expr, error) {
	funcName := "newVariablesFromOrder"
	out := make(map[string]Expr)

	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	matcherPk := tx.GetMatcherPK()
	out["matcherPublicKey"] = NewBytes(common.Dup(matcherPk.Bytes()))
	pair := tx.GetAssetPair()
	out["assetPair"] = NewAssetPair(makeOptionalAsset(pair.AmountAsset), makeOptionalAsset(pair.PriceAsset))
	out["orderType"] = makeOrderType(tx.GetOrderType())
	out["price"] = NewLong(int64(tx.GetPrice()))
	out["amount"] = NewLong(int64(tx.GetAmount()))
	out["timestamp"] = NewLong(int64(tx.GetTimestamp()))
	out["expiration"] = NewLong(int64(tx.GetExpiration()))
	out["matcherFee"] = NewLong(int64(tx.GetMatcherFee()))
	out["matcherFeeAssetId"] = makeOptionalAsset(tx.GetMatcherFeeAsset())
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.GetSenderPK())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	pk := tx.GetSenderPK()
	out["senderPublicKey"] = NewBytes(common.Dup(pk.Bytes()))
	bts, err := proto.MarshalOrderBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	proofs, err := tx.GetProofs()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["proofs"] = makeProofs(proofs)
	out[InstanceFieldName] = NewString("Order")

	return out, nil
}

func NewObjectFromBlockInfo(info proto.BlockInfo) Expr {
	m := make(map[string]Expr)
	m["timestamp"] = NewLong(int64(info.Timestamp))
	m["height"] = NewLong(int64(info.Height))
	m["baseTarget"] = NewLong(int64(info.BaseTarget))
	m["generationSignature"] = NewBytes(info.GenerationSignature.Bytes())
	m["generator"] = NewBytes(common.Dup(info.Generator.Bytes()))
	m["generatorPublicKey"] = NewBytes(common.Dup(info.GeneratorPublicKey.Bytes()))
	m["vfr"] = NewUnit()
	if len(info.VRF) > 0 {
		m["vrf"] = NewBytes(common.Dup(info.VRF.Bytes()))
	}
	return NewObject(m)
}

func newMapAssetInfo(info proto.AssetInfo) object {
	obj := newObject()
	obj["id"] = NewBytes(info.ID.Bytes())
	obj["quantity"] = NewLong(int64(info.Quantity))
	obj["decimals"] = NewLong(int64(info.Decimals))
	obj["issuer"] = NewAddressFromProtoAddress(info.Issuer)
	obj["issuerPublicKey"] = NewBytes(common.Dup(info.IssuerPublicKey.Bytes()))
	obj["reissuable"] = NewBoolean(info.Reissuable)
	obj["scripted"] = NewBoolean(info.Scripted)
	obj["sponsored"] = NewBoolean(info.Sponsored)
	return obj
}

func NewObjectFromAssetInfo(info proto.AssetInfo) Expr {
	return NewObject(newMapAssetInfo(info))
}

func makeProofsFromSignature(sig *crypto.Signature) Exprs {
	out := make([]Expr, 8)
	for i := 0; i < 8; i++ {
		if i == 0 && sig != nil {
			out[i] = NewBytes(sig.Bytes()) //already a copy of bytes of signature
			continue
		}
		out[i] = NewBytes(nil)
	}
	return out
}

func makeProofs(proofs *proto.ProofsV1) Exprs {
	out := make([]Expr, 8)
	pl := len(proofs.Proofs)
	for i := 0; i < 8; i++ {
		if i < pl {
			out[i] = NewBytes(common.Dup(proofs.Proofs[i].Bytes()))
			continue
		}
		out[i] = NewBytes(nil)
	}
	return out
}

func makeOptionalAsset(o proto.OptionalAsset) Expr {
	if o.Present {
		return NewBytes(o.ID.Bytes())
	}
	return NewUnit()
}

func newVariableFromGenesis(scheme proto.Scheme, tx *proto.Genesis) (map[string]Expr, error) {
	funcName := "newVariableFromGenesis"

	out := make(map[string]Expr)
	out["amount"] = NewLong(int64(tx.Amount))
	out["recipient"] = NewRecipientFromProtoRecipient(proto.NewRecipientFromAddress(tx.Recipient))
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(id)
	out["fee"] = NewLong(0)
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))
	return out, nil
}

func newVariablesFromPayment(scheme proto.Scheme, tx *proto.Payment) (map[string]Expr, error) {
	funcName := "newVariablesFromPayment"

	out := make(map[string]Expr)
	out["amount"] = NewLong(int64(tx.Amount))
	out["recipient"] = NewRecipientFromProtoRecipient(proto.NewRecipientFromAddress(tx.Recipient))
	out["id"] = NewBytes(common.Dup(tx.ID.Bytes()))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("PaymentTransaction")
	return out, nil
}

func newVariablesFromTransferWithSig(scheme byte, tx *proto.TransferWithSig) (map[string]Expr, error) {
	funcName := "newVariablesFromTransferWithSig"

	out := make(map[string]Expr)
	out["feeAssetId"] = makeOptionalAsset(tx.FeeAsset)
	out["amount"] = NewLong(int64(tx.Amount))
	out["assetId"] = makeOptionalAsset(tx.AmountAsset)
	out["recipient"] = NewRecipientFromProtoRecipient(tx.Recipient)
	attachmentBytes, err := tx.Attachment.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["attachment"] = NewBytes(attachmentBytes)
	out["id"] = NewBytes(common.Dup(tx.ID.Bytes()))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))

	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("TransferTransaction")
	return out, nil
}

func newVariablesFromTransferWithProofs(scheme byte, tx *proto.TransferWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromTransferWithProofs"

	out := make(map[string]Expr)

	out["feeAssetId"] = makeOptionalAsset(tx.FeeAsset)
	out["amount"] = NewLong(int64(tx.Amount))
	out["assetId"] = makeOptionalAsset(tx.AmountAsset)
	out["recipient"] = NewRecipientFromProtoRecipient(tx.Recipient)
	attachmentBytes, err := tx.Attachment.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["attachment"] = NewBytes(attachmentBytes)
	out["id"] = NewBytes(common.Dup(tx.ID.Bytes()))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))

	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("TransferTransaction")
	return out, nil
}

func newVariablesFromReissueWithSig(scheme proto.Scheme, tx *proto.ReissueWithSig) (map[string]Expr, error) {
	funcName := "newVariablesFromReissueWithSig"

	out := make(map[string]Expr)

	out["quantity"] = NewLong(int64(tx.Quantity))
	out["assetId"] = NewBytes(tx.AssetID.Bytes())
	out["reissuable"] = NewBoolean(tx.Reissuable)
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("ReissueTransaction")
	return out, nil
}

func newVariablesFromReissueWithProofs(scheme proto.Scheme, tx *proto.ReissueWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromReissueWithSig"
	out := make(map[string]Expr)
	out["quantity"] = NewLong(int64(tx.Quantity))
	out["assetId"] = NewBytes(tx.AssetID.Bytes())
	out["reissuable"] = NewBoolean(tx.Reissuable)
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("ReissueTransaction")
	return out, nil
}

func newVariablesFromBurnWithSig(scheme proto.Scheme, tx *proto.BurnWithSig) (map[string]Expr, error) {
	funcName := "newVariablesFromBurnWithSig"

	out := make(map[string]Expr)

	out["quantity"] = NewLong(int64(tx.Amount))
	out["assetId"] = NewBytes(tx.AssetID.Bytes())
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("BurnTransaction")
	return out, nil
}

func newVariablesFromBurnWithProofs(scheme proto.Scheme, tx *proto.BurnWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromBurnWithProofs"

	out := make(map[string]Expr)

	out["quantity"] = NewLong(int64(tx.Amount))
	out["assetId"] = NewBytes(tx.AssetID.Bytes())
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("BurnTransaction")
	return out, nil
}

func newVariablesFromMassTransferWithProofs(scheme proto.Scheme, tx *proto.MassTransferWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromMassTransferWithProofs"
	out := make(map[string]Expr)
	out["assetId"] = makeOptionalAsset(tx.Asset)
	var total uint64
	for _, t := range tx.Transfers {
		total += t.Amount
	}
	out["totalAmount"] = NewLong(int64(total))

	transfers := Exprs{}
	for _, transfer := range tx.Transfers {
		m := make(map[string]Expr)
		m["recipient"] = NewRecipientFromProtoRecipient(transfer.Recipient)
		m["amount"] = NewLong(int64(transfer.Amount))
		transfers = append(transfers, NewObject(m))
	}
	out["transfers"] = transfers
	out["transferCount"] = NewLong(int64(len(tx.Transfers)))
	attachmentBytes, err := tx.Attachment.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["attachment"] = NewBytes(attachmentBytes)
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))

	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("MassTransferTransaction")

	return out, nil
}

func newVariablesFromExchangeWithSig(scheme proto.Scheme, tx *proto.ExchangeWithSig) (map[string]Expr, error) {
	funcName := "newVariablesFromExchangeWithSig"
	out := make(map[string]Expr)
	buy, err := NewVariablesFromOrder(scheme, tx.Order1)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["buyOrder"] = NewObject(buy)

	sell, err := NewVariablesFromOrder(scheme, tx.Order2)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sellOrder"] = NewObject(sell)
	out["price"] = NewLong(int64(tx.Price))
	out["amount"] = NewLong(int64(tx.Amount))
	out["buyMatcherFee"] = NewLong(int64(tx.BuyMatcherFee))
	out["sellMatcherFee"] = NewLong(int64(tx.SellMatcherFee))

	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("ExchangeTransaction")
	return out, nil
}

func newVariablesFromExchangeWithProofs(scheme proto.Scheme, tx *proto.ExchangeWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromExchangeWithProofs"
	out := make(map[string]Expr)

	buy, err := NewVariablesFromOrder(scheme, tx.Order1)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["buyOrder"] = NewObject(buy)

	sell, err := NewVariablesFromOrder(scheme, tx.Order2)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sellOrder"] = NewObject(sell)

	out["price"] = NewLong(int64(tx.Price))
	out["amount"] = NewLong(int64(tx.Amount))

	out["buyMatcherFee"] = NewLong(int64(tx.BuyMatcherFee))
	out["sellMatcherFee"] = NewLong(int64(tx.SellMatcherFee))

	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("ExchangeTransaction")
	return out, nil
}

func makeOrderType(orderType proto.OrderType) Expr {
	if orderType == proto.Buy {
		return &BuyExpr{}
	}
	if orderType == proto.Sell {
		return &SellExpr{}
	}
	panic("invalid orderType")
}

func newVariablesFromSetAssetScriptWithProofs(scheme proto.Scheme, tx *proto.SetAssetScriptWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromSetAssetScriptWithProofs"

	out := make(map[string]Expr)

	out["script"] = NewBytes(common.Dup(tx.Script))
	out["assetId"] = NewBytes(tx.AssetID.Bytes())
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("SetAssetScriptTransaction")
	return out, nil
}

func newVariablesFromInvokeScriptWithProofs(scheme proto.Scheme, tx *proto.InvokeScriptWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromInvokeScriptWithProofs"

	out := make(map[string]Expr)

	out["dApp"] = NewRecipientFromProtoRecipient(tx.ScriptRecipient)
	out["payment"] = NewUnit()
	if len(tx.Payments) > 0 {
		out["payment"] = NewAttachedPaymentExpr(
			makeOptionalAsset(tx.Payments[0].Asset),
			NewLong(int64(tx.Payments[0].Amount)),
		)
	}
	out["feeAssetId"] = makeOptionalAsset(tx.FeeAsset)
	out["function"] = NewString(tx.FunctionCall.Name)

	var args Exprs
	for _, arg := range tx.FunctionCall.Arguments {
		switch t := arg.(type) {
		case *proto.BooleanArgument:
			args = append(args, NewBoolean(t.Value))
		case *proto.IntegerArgument:
			args = append(args, NewLong(t.Value))
		case *proto.StringArgument:
			args = append(args, NewString(t.Value))
		case *proto.BinaryArgument:
			args = append(args, NewBytes(common.Dup(t.Value)))
		default:
			return nil, errors.Errorf("%s: invalid argument type %T", funcName, arg)
		}
	}
	out["args"] = args
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("InvokeScriptTransaction")
	return out, nil
}

func newVariablesFromIssueWithSig(scheme proto.Scheme, tx *proto.IssueWithSig) (map[string]Expr, error) {
	funcName := "newVariablesFromReissueWithSig"

	out := make(map[string]Expr)

	out["quantity"] = NewLong(int64(tx.Quantity))
	out["name"] = NewString(tx.Name)
	out["description"] = NewString(tx.Description)
	out["reissuable"] = NewBoolean(tx.Reissuable)
	out["decimals"] = NewLong(int64(tx.Decimals))
	out["script"] = NewUnit()
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("IssueTransaction")
	return out, nil
}

func newVariablesFromIssueWithProofs(scheme proto.Scheme, tx *proto.IssueWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromReissueWithSig"

	out := make(map[string]Expr)

	out["quantity"] = NewLong(int64(tx.Quantity))
	out["name"] = NewString(tx.Name)
	out["description"] = NewString(tx.Description)
	out["reissuable"] = NewBoolean(tx.Reissuable)
	out["decimals"] = NewLong(int64(tx.Decimals))
	out["script"] = NewUnit()
	if tx.NonEmptyScript() {
		out["script"] = NewBytes(common.Dup(tx.Script))
	}
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("IssueTransaction")
	return out, nil
}

func newVariablesFromLeaseWithSig(scheme proto.Scheme, tx *proto.LeaseWithSig) (map[string]Expr, error) {
	funcName := "newVariablesFromLeaseWithSig"

	out := make(map[string]Expr)

	out["amount"] = NewLong(int64(tx.Amount))
	out["recipient"] = NewRecipientFromProtoRecipient(tx.Recipient)
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("LeaseTransaction")
	return out, nil
}

func newVariablesFromLeaseWithProofs(scheme proto.Scheme, tx *proto.LeaseWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromLeaseWithProofs"

	out := make(map[string]Expr)

	out["amount"] = NewLong(int64(tx.Amount))
	out["recipient"] = NewRecipientFromProtoRecipient(tx.Recipient)
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("LeaseTransaction")
	return out, nil
}

func newVariablesFromLeaseCancelWithSig(scheme proto.Scheme, tx *proto.LeaseCancelWithSig) (map[string]Expr, error) {
	funcName := "newVariablesFromLeaseCancelWithSig"

	out := make(map[string]Expr)
	out["leaseId"] = NewBytes(common.Dup(tx.LeaseID.Bytes()))
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("LeaseCancelTransaction")
	return out, nil
}

func newVariablesFromLeaseCancelWithProofs(scheme proto.Scheme, tx *proto.LeaseCancelWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromLeaseCancelWithProofs"

	out := make(map[string]Expr)
	out["leaseId"] = NewBytes(common.Dup(tx.LeaseID.Bytes()))
	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("LeaseCancelTransaction")
	return out, nil
}

func newVariablesFromDataWithProofs(scheme proto.Scheme, tx *proto.DataWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromDataWithProofs"

	out := make(map[string]Expr)

	out["data"] = NewDataEntryList(tx.Entries)

	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("DataTransaction")
	return out, nil
}

func newVariablesFromSponsorshipWithProofs(scheme proto.Scheme, tx *proto.SponsorshipWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromSponsorshipWithProofs"

	out := make(map[string]Expr)

	out["assetId"] = NewBytes(tx.AssetID.Bytes())
	out["minSponsoredAssetFee"] = NewUnit()
	if tx.MinAssetFee > 0 {
		out["minSponsoredAssetFee"] = NewLong(int64(tx.MinAssetFee))
	}

	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("SponsorFeeTransaction")
	return out, nil
}

func newVariablesFromCreateAliasWithSig(scheme proto.Scheme, tx *proto.CreateAliasWithSig) (map[string]Expr, error) {
	funcName := "newVariablesFromCreateAliasWithSig"

	out := make(map[string]Expr)

	out["alias"] = NewString(tx.Alias.String())

	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("CreateAliasTransaction")
	return out, nil
}

func newVariablesFromCreateAliasWithProofs(scheme proto.Scheme, tx *proto.CreateAliasWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromCreateAliasWithSig"

	out := make(map[string]Expr)

	out["alias"] = NewString(tx.Alias.String())

	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("CreateAliasTransaction")
	return out, nil
}

func newVariablesFromSetScriptWithProofs(scheme proto.Scheme, tx *proto.SetScriptWithProofs) (map[string]Expr, error) {
	funcName := "newVariablesFromSetScriptWithProofs"

	out := make(map[string]Expr)

	if len(tx.Script) == 0 {
		out["script"] = NewUnit()
	} else {
		out["script"] = NewBytes(tx.Script)
	}

	id, err := tx.GetID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(common.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(common.Dup(tx.SenderPK.Bytes()))
	bts, err := proto.MarshalTxBody(scheme, tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("SetScriptTransaction")
	return out, nil
}
