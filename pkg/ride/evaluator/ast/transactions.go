package ast

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

func NewVariablesFromTransaction(scheme byte, t proto.Transaction) (map[string]Expr, error) {
	switch tx := t.(type) {
	case *proto.Genesis:
		return newVariableFromGenesis(tx)
	case *proto.Payment:
		return newVariablesFromPayment(scheme, tx)
	case *proto.TransferV1:
		return newVariablesFromTransferV1(scheme, tx)
	case *proto.TransferV2:
		return newVariablesFromTransferV2(scheme, tx)
	case *proto.ReissueV1:
		return newVariablesFromReissueV1(scheme, tx)
	case *proto.ReissueV2:
		return newVariablesFromReissueV2(scheme, tx)
	case *proto.BurnV1:
		return newVariablesFromBurnV1(scheme, tx)
	case *proto.BurnV2:
		return newVariablesFromBurnV2(scheme, tx)
	case *proto.MassTransferV1:
		return newVariablesFromMassTransferV1(scheme, tx)
	case *proto.ExchangeV1:
		return newVariablesFromExchangeV1(scheme, tx)
	case *proto.ExchangeV2:
		return newVariablesFromExchangeV2(scheme, tx)
	case *proto.SetAssetScriptV1:
		return newVariablesFromSetAssetScriptV1(scheme, tx)
	case *proto.InvokeScriptV1:
		return newVariablesFromInvokeScriptV1(scheme, tx)
	case *proto.IssueV1:
		return newVariablesFromIssueV1(scheme, tx)
	case *proto.IssueV2:
		return newVariablesFromIssueV2(scheme, tx)
	case *proto.LeaseV1:
		return newVariablesFromLeaseV1(scheme, tx)
	case *proto.LeaseV2:
		return newVariablesFromLeaseV2(scheme, tx)
	case *proto.LeaseCancelV1:
		return newVariablesFromLeaseCancelV1(scheme, tx)
	case *proto.LeaseCancelV2:
		return newVariablesFromLeaseCancelV2(scheme, tx)
	case *proto.DataV1:
		return newVariablesFromDataV1(scheme, tx)
	case *proto.SponsorshipV1:
		return newVariablesFromSponsorshipV1(scheme, tx)
	case *proto.CreateAliasV1:
		return newVariablesFromCreateAliasV1(scheme, tx)
	case *proto.CreateAliasV2:
		return newVariablesFromCreateAliasV2(scheme, tx)
	case *proto.SetScriptV1:
		return newVariablesFromSetScriptV1(scheme, tx)
	default:
		return nil, errors.Errorf("NewVariablesFromTransaction not implemented for %T", tx)
	}
}

func NewVariablesFromOrder(scheme proto.Scheme, tx proto.Order) (map[string]Expr, error) {
	funcName := "newVariablesFromOrder"
	out := make(map[string]Expr)

	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	matcherPk := tx.GetMatcherPK()
	out["matcherPublicKey"] = NewBytes(util.Dup(matcherPk.Bytes()))
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
	out["senderPublicKey"] = NewBytes(util.Dup(pk.Bytes()))
	bts, err := tx.BodyMarshalBinary()
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
	m["generator"] = NewBytes(util.Dup(info.Generator.Bytes()))
	m["generatorPublicKey"] = NewBytes(util.Dup(info.GeneratorPublicKey.Bytes()))
	return NewObject(m)
}

func newMapAssetInfo(info proto.AssetInfo) object {
	obj := newObject()
	obj["id"] = NewBytes(info.ID.Bytes())
	obj["quantity"] = NewLong(int64(info.Quantity))
	obj["decimals"] = NewLong(int64(info.Decimals))
	obj["issuer"] = NewAddressFromProtoAddress(info.Issuer)
	obj["issuerPublicKey"] = NewBytes(util.Dup(info.IssuerPublicKey.Bytes()))
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
			out[i] = NewBytes(util.Dup(proofs.Proofs[i].Bytes()))
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

func newVariableFromGenesis(tx *proto.Genesis) (map[string]Expr, error) {
	funcName := "newVariableFromGenesis"

	out := make(map[string]Expr)
	out["amount"] = NewLong(int64(tx.Amount))
	out["recipient"] = NewRecipientFromProtoRecipient(proto.NewRecipientFromAddress(tx.Recipient))
	id, err := tx.GetID()
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
	out["id"] = NewBytes(util.Dup(tx.ID.Bytes()))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("PaymentTransaction")
	return out, nil
}

func newVariablesFromTransferV1(scheme byte, tx *proto.TransferV1) (map[string]Expr, error) {
	funcName := "newVariablesFromTransferV1"

	out := make(map[string]Expr)
	out["feeAssetId"] = makeOptionalAsset(tx.FeeAsset)
	out["amount"] = NewLong(int64(tx.Amount))
	out["assetId"] = makeOptionalAsset(tx.AmountAsset)
	out["recipient"] = NewRecipientFromProtoRecipient(tx.Recipient)
	out["attachment"] = NewBytes(tx.Attachment.Bytes())
	out["id"] = NewBytes(util.Dup(tx.ID.Bytes()))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))

	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("TransferTransaction")
	return out, nil
}

func newVariablesFromTransferV2(scheme byte, tx *proto.TransferV2) (map[string]Expr, error) {
	funcName := "newVariablesFromTransferV2"

	out := make(map[string]Expr)

	out["feeAssetId"] = makeOptionalAsset(tx.FeeAsset)
	out["amount"] = NewLong(int64(tx.Amount))
	out["assetId"] = makeOptionalAsset(tx.AmountAsset)
	out["recipient"] = NewRecipientFromProtoRecipient(tx.Recipient)
	out["attachment"] = NewBytes(tx.Attachment.Bytes())
	out["id"] = NewBytes(util.Dup(tx.ID.Bytes()))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))

	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("TransferTransaction")
	return out, nil
}

func newVariablesFromReissueV1(scheme proto.Scheme, tx *proto.ReissueV1) (map[string]Expr, error) {
	funcName := "newVariablesFromReissueV1"

	out := make(map[string]Expr)

	out["quantity"] = NewLong(int64(tx.Quantity))
	out["assetId"] = NewBytes(util.Dup(tx.AssetID.Bytes()))
	out["reissuable"] = NewBoolean(tx.Reissuable)
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("ReissueTransaction")
	return out, nil
}

func newVariablesFromReissueV2(scheme proto.Scheme, tx *proto.ReissueV2) (map[string]Expr, error) {
	funcName := "newVariablesFromReissueV1"
	out := make(map[string]Expr)
	out["quantity"] = NewLong(int64(tx.Quantity))
	out["assetId"] = NewBytes(util.Dup(tx.AssetID.Bytes()))
	out["reissuable"] = NewBoolean(tx.Reissuable)
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("ReissueTransaction")
	return out, nil
}

func newVariablesFromBurnV1(scheme proto.Scheme, tx *proto.BurnV1) (map[string]Expr, error) {
	funcName := "newVariablesFromBurnV1"

	out := make(map[string]Expr)

	out["quantity"] = NewLong(int64(tx.Amount))
	out["assetId"] = NewBytes(util.Dup(tx.AssetID.Bytes()))
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("BurnTransaction")
	return out, nil
}

func newVariablesFromBurnV2(scheme proto.Scheme, tx *proto.BurnV2) (map[string]Expr, error) {
	funcName := "newVariablesFromBurnV2"

	out := make(map[string]Expr)

	out["quantity"] = NewLong(int64(tx.Amount))
	out["assetId"] = NewBytes(util.Dup(tx.AssetID.Bytes()))
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("BurnTransaction")
	return out, nil
}

func newVariablesFromMassTransferV1(scheme proto.Scheme, tx *proto.MassTransferV1) (map[string]Expr, error) {
	funcName := "newVariablesFromMassTransferV1"
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
	out["attachment"] = NewBytes(tx.Attachment.Bytes())
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))

	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("MassTransferTransaction")

	return out, nil
}

func newVariablesFromExchangeV1(scheme proto.Scheme, tx *proto.ExchangeV1) (map[string]Expr, error) {
	funcName := "newVariablesFromExchangeV1"
	out := make(map[string]Expr)
	buy, err := NewVariablesFromOrder(scheme, tx.BuyOrder)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["buyOrder"] = NewObject(buy)

	sell, err := NewVariablesFromOrder(scheme, tx.SellOrder)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sellOrder"] = NewObject(sell)
	out["price"] = NewLong(int64(tx.Price))
	out["amount"] = NewLong(int64(tx.Amount))
	out["buyMatcherFee"] = NewLong(int64(tx.BuyMatcherFee))
	out["sellMatcherFee"] = NewLong(int64(tx.SellMatcherFee))

	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("ExchangeTransaction")
	return out, nil
}

func newVariablesFromExchangeV2(scheme proto.Scheme, tx *proto.ExchangeV2) (map[string]Expr, error) {
	funcName := "newVariablesFromExchangeV2"
	out := make(map[string]Expr)

	buy, err := NewVariablesFromOrder(scheme, tx.BuyOrder)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["buyOrder"] = NewObject(buy)

	sell, err := NewVariablesFromOrder(scheme, tx.SellOrder)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sellOrder"] = NewObject(sell)

	out["price"] = NewLong(int64(tx.Price))
	out["amount"] = NewLong(int64(tx.Amount))

	out["buyMatcherFee"] = NewLong(int64(tx.BuyMatcherFee))
	out["sellMatcherFee"] = NewLong(int64(tx.SellMatcherFee))

	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
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

func newVariablesFromSetAssetScriptV1(scheme proto.Scheme, tx *proto.SetAssetScriptV1) (map[string]Expr, error) {
	funcName := "newVariablesFromSetAssetScriptV1"

	out := make(map[string]Expr)

	out["script"] = NewBytes(util.Dup(tx.Script))
	out["assetId"] = NewBytes(util.Dup(tx.AssetID.Bytes()))
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("SetAssetScriptTransaction")
	return out, nil
}

func newVariablesFromInvokeScriptV1(scheme proto.Scheme, tx *proto.InvokeScriptV1) (map[string]Expr, error) {
	funcName := "newVariablesFromInvokeScriptV1"

	out := make(map[string]Expr)

	out["dappAddress"] = NewRecipientFromProtoRecipient(tx.ScriptRecipient)
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
			args = append(args, NewBytes(util.Dup(t.Value)))
		default:
			return nil, errors.Errorf("%s: invalid argument type %T", funcName, arg)
		}
	}
	out["args"] = args
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("InvokeScriptTransaction")
	return out, nil
}

func newVariablesFromIssueV1(scheme proto.Scheme, tx *proto.IssueV1) (map[string]Expr, error) {
	funcName := "newVariablesFromReissueV1"

	out := make(map[string]Expr)

	out["quantity"] = NewLong(int64(tx.Quantity))
	out["name"] = NewString(tx.Name)
	out["description"] = NewString(tx.Description)
	out["reissuable"] = NewBoolean(tx.Reissuable)
	out["decimals"] = NewLong(int64(tx.Decimals))
	out["script"] = NewUnit()
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("IssueTransaction")
	return out, nil
}

func newVariablesFromIssueV2(scheme proto.Scheme, tx *proto.IssueV2) (map[string]Expr, error) {
	funcName := "newVariablesFromReissueV1"

	out := make(map[string]Expr)

	out["quantity"] = NewLong(int64(tx.Quantity))
	out["name"] = NewString(tx.Name)
	out["description"] = NewString(tx.Description)
	out["reissuable"] = NewBoolean(tx.Reissuable)
	out["decimals"] = NewLong(int64(tx.Decimals))
	out["script"] = NewUnit()
	if tx.NonEmptyScript() {
		out["script"] = NewBytes(util.Dup(tx.Script))
	}
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("IssueTransaction")
	return out, nil
}

func newVariablesFromLeaseV1(scheme proto.Scheme, tx *proto.LeaseV1) (map[string]Expr, error) {
	funcName := "newVariablesFromLeaseV1"

	out := make(map[string]Expr)

	out["amount"] = NewLong(int64(tx.Amount))
	out["recipient"] = NewRecipientFromProtoRecipient(tx.Recipient)
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("LeaseTransaction")
	return out, nil
}

func newVariablesFromLeaseV2(scheme proto.Scheme, tx *proto.LeaseV2) (map[string]Expr, error) {
	funcName := "newVariablesFromLeaseV2"

	out := make(map[string]Expr)

	out["amount"] = NewLong(int64(tx.Amount))
	out["recipient"] = NewRecipientFromProtoRecipient(tx.Recipient)
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("LeaseTransaction")
	return out, nil
}

func newVariablesFromLeaseCancelV1(scheme proto.Scheme, tx *proto.LeaseCancelV1) (map[string]Expr, error) {
	funcName := "newVariablesFromLeaseCancelV1"

	out := make(map[string]Expr)
	out["leaseId"] = NewBytes(util.Dup(tx.LeaseID.Bytes()))
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("LeaseCancelTransaction")
	return out, nil
}

func newVariablesFromLeaseCancelV2(scheme proto.Scheme, tx *proto.LeaseCancelV2) (map[string]Expr, error) {
	funcName := "newVariablesFromLeaseCancelV2"

	out := make(map[string]Expr)
	out["leaseId"] = NewBytes(util.Dup(tx.LeaseID.Bytes()))
	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("LeaseCancelTransaction")
	return out, nil
}

func newVariablesFromDataV1(scheme proto.Scheme, tx *proto.DataV1) (map[string]Expr, error) {
	funcName := "newVariablesFromDataV1"

	out := make(map[string]Expr)

	out["data"] = NewDataEntryList(tx.Entries)

	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("DataTransaction")
	return out, nil
}

func newVariablesFromSponsorshipV1(scheme proto.Scheme, tx *proto.SponsorshipV1) (map[string]Expr, error) {
	funcName := "newVariablesFromSponsorshipV1"

	out := make(map[string]Expr)

	out["assetId"] = NewBytes(util.Dup(tx.AssetID.Bytes()))
	out["minSponsoredAssetFee"] = NewUnit()
	if tx.MinAssetFee > 0 {
		out["minSponsoredAssetFee"] = NewLong(int64(tx.MinAssetFee))
	}

	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("SponsorFeeTransaction")
	return out, nil
}

func newVariablesFromCreateAliasV1(scheme proto.Scheme, tx *proto.CreateAliasV1) (map[string]Expr, error) {
	funcName := "newVariablesFromCreateAliasV1"

	out := make(map[string]Expr)

	out["alias"] = NewString(tx.Alias.String())

	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofsFromSignature(tx.Signature)
	out[InstanceFieldName] = NewString("CreateAliasTransaction")
	return out, nil
}

func newVariablesFromCreateAliasV2(scheme proto.Scheme, tx *proto.CreateAliasV2) (map[string]Expr, error) {
	funcName := "newVariablesFromCreateAliasV1"

	out := make(map[string]Expr)

	out["alias"] = NewString(tx.Alias.String())

	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("CreateAliasTransaction")
	return out, nil
}

func newVariablesFromSetScriptV1(scheme proto.Scheme, tx *proto.SetScriptV1) (map[string]Expr, error) {
	funcName := "newVariablesFromSetScriptV1"

	out := make(map[string]Expr)

	if len(tx.Script) == 0 {
		out["script"] = NewUnit()
	} else {
		out["script"] = NewBytes(tx.Script)
	}

	id, err := tx.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(util.Dup(id))
	out["fee"] = NewLong(int64(tx.Fee))
	out["timestamp"] = NewLong(int64(tx.Timestamp))
	out["version"] = NewLong(int64(tx.Version))

	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["sender"] = NewAddressFromProtoAddress(addr)
	out["senderPublicKey"] = NewBytes(util.Dup(tx.SenderPK.Bytes()))
	bts, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["bodyBytes"] = NewBytes(bts)
	out["proofs"] = makeProofs(tx.Proofs)
	out[InstanceFieldName] = NewString("SetScriptTransaction")
	return out, nil
}
