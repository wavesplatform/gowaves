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

func NewVariablesFromScriptAction(scheme proto.Scheme, action proto.ScriptAction, invokerPK crypto.PublicKey, txID crypto.Digest, txTimestamp uint64) (map[string]Expr, error) {
	out := make(map[string]Expr)
	switch a := action.(type) {
	case *proto.ReissueScriptAction:
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
	case *proto.BurnScriptAction:
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
		return nil, errors.Errorf("unsupported script action '%T'", action)
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

func newMapAssetInfoV3(info proto.AssetInfo) object {
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

func newMapAssetInfoV4(info proto.FullAssetInfo) object {
	obj := newObject()
	obj["id"] = NewBytes(info.ID.Bytes())
	obj["quantity"] = NewLong(int64(info.Quantity))
	obj["decimals"] = NewLong(int64(info.Decimals))
	obj["issuer"] = NewAddressFromProtoAddress(info.Issuer)
	obj["issuerPublicKey"] = NewBytes(common.Dup(info.IssuerPublicKey.Bytes()))
	obj["reissuable"] = NewBoolean(info.Reissuable)
	obj["scripted"] = NewBoolean(info.Scripted)
	obj["sponsored"] = NewBoolean(info.Sponsored)
	obj["name"] = NewString(info.Name)
	obj["description"] = NewString(info.Description)
	return obj
}

func NewObjectFromAssetInfoV3(info proto.AssetInfo) Expr {
	return NewObject(newMapAssetInfoV3(info))
}

func NewObjectFromAssetInfoV4(info proto.FullAssetInfo) Expr {
	return NewObject(newMapAssetInfoV4(info))
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

func makeOrderType(orderType proto.OrderType) Expr {
	if orderType == proto.Buy {
		return &BuyExpr{}
	}
	if orderType == proto.Sell {
		return &SellExpr{}
	}
	panic("invalid orderType")
}
