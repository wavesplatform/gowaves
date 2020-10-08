package common

import "github.com/wavesplatform/gowaves/pkg/proto"

type UtxPool interface {
	AddBytes([]byte) error
}

func PreHandler(message proto.Message, u UtxPool) (handled bool) {
	switch m := message.(type) {
	case *proto.TransactionMessage:
		_ = u.AddBytes(m.Transaction)
		return true
	default:
		return false
	}
}
