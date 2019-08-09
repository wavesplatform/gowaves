package byte_helpers

import (
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const TIMESTAMP = proto.Timestamp(1544715621)

type GenesisStruct struct {
	TransactionBytes []byte
	Transaction      *proto.Genesis
	MessageBytes     []byte
}

var Genesis GenesisStruct

type TransferV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.TransferV1
	MessageBytes     []byte
}

var TransferV1 TransferV1Struct

type TransferV2Struct struct {
	TransactionBytes []byte
	Transaction      *proto.TransferV2
	MessageBytes     []byte
}

var TransferV2 TransferV2Struct

type IssueV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.IssueV1
	MessageBytes     []byte
}

var IssueV1 IssueV1Struct

func init() {
	initGenesis()
	initTransferV1()
	initTransferV2()
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

func initTransferV2() {
	sk, pk := crypto.GenerateKeyPair([]byte("test"))
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)
	if err != nil {
		panic(err)
	}
	t := proto.NewUnsignedTransferV2(
		pk,
		proto.OptionalAsset{},
		proto.OptionalAsset{},
		TIMESTAMP,
		100000,
		10000,
		proto.NewRecipientFromAddress(addr),
		"abc",
	)

	_ = t.Sign(sk)
	b, _ := t.MarshalBinary()
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	TransferV2 = TransferV2Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}

}

func initIssueV1() {

	sk, pk := crypto.GenerateKeyPair([]byte("test"))

	t := proto.NewUnsignedIssueV1(
		pk,
		"name",
		"description",
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

func initGenesis() {
	_, pk := crypto.GenerateKeyPair([]byte("test"))
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)
	if err != nil {
		panic(err)
	}
	t := proto.NewUnsignedGenesis(addr, 100000, TIMESTAMP)
	b, _ := t.MarshalBinary()
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	Genesis = GenesisStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}
