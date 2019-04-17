package byte_helpers

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type TransferV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.TransferV1
	MessageBytes     []byte
}

var TransferV1 TransferV1Struct

func init() {
	t := util.NewTransferV1Builder().MustBuild()
	b, _ := t.MarshalBinary()
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	TransferV1 = TransferV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}
