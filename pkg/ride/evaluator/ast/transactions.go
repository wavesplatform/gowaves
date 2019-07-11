package ast

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	case *proto.Payment:
		out["id"] = NewBytes(tx.ID.Bytes())
		return out, nil
	case *proto.TransferV1:
		addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		out["sender"] = NewAddressFromProtoAddress(addr)
		out["amount"] = NewLong(int64(tx.Amount))
		out["timestamp"] = NewLong(int64(tx.Timestamp))
		bts, err := tx.MarshalBinary()
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		out["bodyBytes"] = NewBytes(bts)
		out[InstanceFieldName] = NewString("TransferTransaction")
		return out, nil
	case *proto.TransferV2:
		addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		out["sender"] = NewAddressFromProtoAddress(addr)
		out["amount"] = NewLong(int64(tx.Amount))
		out["timestamp"] = NewLong(int64(tx.Timestamp))
		bts, err := tx.BodyMarshalBinary()
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		out["bodyBytes"] = NewBytes(bts)
		out[InstanceFieldName] = NewString("TransferTransaction")

		proofs := Exprs{}
		for _, row := range tx.Proofs.Proofs {
			proofs = append(proofs, NewBytes(row.Bytes()))
		}
		out["proofs"] = proofs
		return out, nil
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
