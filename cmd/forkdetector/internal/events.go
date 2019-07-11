package internal

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type signaturesEvent struct {
	conn       *Conn
	signatures []crypto.Signature
}

func newSignaturesEvent(conn *Conn, signatures []crypto.Signature) signaturesEvent {
	s := make([]crypto.Signature, len(signatures))
	copy(s, signatures)
	return signaturesEvent{
		conn:       conn,
		signatures: s,
	}
}

type blockEvent struct {
	conn  *Conn
	block *proto.Block
}
