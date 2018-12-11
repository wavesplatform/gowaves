package ast

import (
	"github.com/go-errors/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewVariablesFromTransaction(t proto.Transaction) (map[string]Expr, error) {

	out := make(map[string]Expr)

	switch tx := t.(type) {
	case *proto.Payment:
		out["id"] = NewBytes(tx.ID.Bytes())
		return out, nil
	default:
		return nil, errors.Errorf("NewVariablesFromTransaction not implemented for %T", tx)
	}

}
