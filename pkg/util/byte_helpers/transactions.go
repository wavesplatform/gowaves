package byte_helpers

import (
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type TransferV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.TransferV1
	MessageBytes     []byte
}

var TransferV1 TransferV1Struct

type IssueV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.IssueV1
	MessageBytes     []byte
}

var IssueV1 IssueV1Struct

func init() {
	initTransferV1()
	initIssueV1()
}

func initTransferV1() {
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
func initIssueV1() {

	sk, pk, _ := crypto.GenerateKeyPair([]byte("test"))

	t := proto.NewUnsignedIssueV1(
		pk,
		"name",
		"",
		1000,
		0,
		false,
		proto.NewTimestampFromTime(time.Now()),
		10000)

	_ = t.Sign(sk)
	b, _ := t.MarshalBinary()
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	IssueV1 = IssueV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}
