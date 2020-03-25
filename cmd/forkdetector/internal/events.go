package internal

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type idsEvent struct {
	conn *Conn
	ids  []proto.BlockID
}

func newIdsEvent(conn *Conn, ids []proto.BlockID) idsEvent {
	s := make([]proto.BlockID, len(ids))
	copy(s, ids)
	return idsEvent{
		conn: conn,
		ids:  s,
	}
}

type blockEvent struct {
	conn  *Conn
	block *proto.Block
}
