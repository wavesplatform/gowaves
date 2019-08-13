package ast

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

func NewVariablesFromTransaction(scheme byte, t proto.Transaction) (map[string]Expr, error) {

	funcName := "NewVariablesFromTransaction"

	out := make(map[string]Expr)
	tID, err := t.GetID()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out["id"] = NewBytes(tID)

	switch tx := t.(type) {
	case *proto.Genesis:
		return newVariableFromGenesis(scheme, tx)
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
	case *proto.DataV1:
		addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		out["sender"] = NewAddressFromProtoAddress(addr)
		out["timestamp"] = NewLong(int64(tx.Timestamp))
		bts, err := tx.BodyMarshalBinary()
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		out["bodyBytes"] = NewBytes(bts)
		proofs := Exprs{}
		for _, row := range tx.Proofs.Proofs {
			proofs = append(proofs, NewBytes(row.Bytes()))
		}
		out["proofs"] = proofs
		out["data"] = NewDataEntryList(tx.Entries)
		out[InstanceFieldName] = NewString("DataTransaction")
		return out, nil
	default:
		return nil, errors.Errorf("NewVariablesFromTransaction not implemented for %T", tx)
	}

}

func makeOptionalAsset(o proto.OptionalAsset) Expr {
	if o.Present {
		return NewBytes(util.Dup(o.ID.Bytes()))
	}
	return NewUnit()
}

func newVariableFromGenesis(scheme proto.Scheme, tx *proto.Genesis) (map[string]Expr, error) {
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
	out["proofs"] = Exprs{NewBytes(util.Dup(tx.Signature.Bytes()))}
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
	out["proofs"] = Exprs{NewBytes(util.Dup(tx.Signature.Bytes()))}
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

	exprs := Exprs{}
	for _, proof := range tx.Proofs.Proofs {
		exprs = append(exprs, NewBytes(util.Dup(proof)))
	}
	out["proofs"] = exprs
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
	out["proofs"] = Exprs{NewBytes(util.Dup(tx.Signature.Bytes()))}
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
	exprs := Exprs{}
	for _, proof := range tx.Proofs.Proofs {
		exprs = append(exprs, NewBytes(util.Dup(proof)))
	}
	out["proofs"] = exprs
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
	out["proofs"] = Exprs{NewBytes(util.Dup(tx.Signature.Bytes()))}
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
	exprs := Exprs{}
	for _, proof := range tx.Proofs.Proofs {
		exprs = append(exprs, NewBytes(util.Dup(proof)))
	}
	out["proofs"] = exprs
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
	exprs := Exprs{}
	for _, proof := range tx.Proofs.Proofs {
		exprs = append(exprs, NewBytes(util.Dup(proof)))
	}
	out["proofs"] = exprs
	out[InstanceFieldName] = NewString("MassTransferTransaction")

	return out, nil
}
