package ast

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewVariablesFromTransaction(scheme byte, t proto.Transaction) (map[string]Expr, error) {

	out := make(map[string]Expr)
	out["id"] = NewBytes(t.GetID())

	switch tx := t.(type) {
	case *proto.Payment:
		out["id"] = NewBytes(tx.ID.Bytes())
		return out, nil
	case *proto.TransferV1:
		addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "NewVariablesFromTransaction")
		}
		out["sender"] = NewAddressFromProtoAddress(addr)
		return out, nil
	default:
		return nil, errors.Errorf("NewVariablesFromTransaction not implemented for %T", tx)
	}

}
