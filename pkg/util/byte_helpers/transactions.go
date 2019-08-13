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

type PaymentStruct struct {
	TransactionBytes []byte
	Transaction      *proto.Payment
	MessageBytes     []byte
}

var Payment PaymentStruct

type ReissueV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.ReissueV1
	MessageBytes     []byte
}

var ReissueV1 ReissueV1Struct

type ReissueV2Struct struct {
	TransactionBytes []byte
	Transaction      *proto.ReissueV2
	MessageBytes     []byte
}

var ReissueV2 ReissueV2Struct

type TransferV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.TransferV1
	MessageBytes     []byte
}

var TransferV1 TransferV1Struct

type BurnV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.BurnV1
	MessageBytes     []byte
}

var BurnV1 BurnV1Struct

type BurnV2Struct struct {
	TransactionBytes []byte
	Transaction      *proto.BurnV2
	MessageBytes     []byte
}

var BurnV2 BurnV2Struct

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

type MassTransferV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.MassTransferV1
	MessageBytes     []byte
}

var MassTransferV1 MassTransferV1Struct

func init() {
	initGenesis()
	initPayment()
	initTransferV1()
	initTransferV2()
	initIssueV1()
	initReissueV1()
	initReissueV2()
	initBurnV1()
	initBurnV2()
	initMassTransferV1()
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

func initPayment() {
	sk, pk := crypto.GenerateKeyPair([]byte("test"))
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)
	if err != nil {
		panic(err)
	}
	t := proto.NewUnsignedPayment(pk, addr, 100000, 10000, TIMESTAMP)

	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	Payment = PaymentStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initReissueV1() {
	sk, pk := crypto.GenerateKeyPair([]byte("test"))
	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}

	t := proto.NewUnsignedReissueV1(pk, d, 100000, true, TIMESTAMP, 10000)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	ReissueV1 = ReissueV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initReissueV2() {
	sk, pk := crypto.GenerateKeyPair([]byte("test"))
	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}

	t := proto.NewUnsignedReissueV2(proto.MainNetScheme, pk, d, 100000, true, TIMESTAMP, 10000)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	ReissueV2 = ReissueV2Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initBurnV1() {
	sk, pk := crypto.GenerateKeyPair([]byte("test"))
	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}

	t := proto.NewUnsignedBurnV1(pk, d, 100000, TIMESTAMP, 10000)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	BurnV1 = BurnV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initBurnV2() {
	sk, pk := crypto.GenerateKeyPair([]byte("test"))
	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}

	t := proto.NewUnsignedBurnV2(proto.MainNetScheme, pk, d, 100000, TIMESTAMP, 10000)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	BurnV2 = BurnV2Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initMassTransferV1() {
	sk, pk := crypto.GenerateKeyPair([]byte("test"))
	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}

	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)
	if err != nil {
		panic(err)
	}

	entry := proto.MassTransferEntry{
		Recipient: proto.NewRecipientFromAddress(addr),
		Amount:    100000,
	}
	t := proto.NewUnsignedMassTransferV1(pk, *proto.NewOptionalAssetFromDigest(d), []proto.MassTransferEntry{entry}, 10000, TIMESTAMP, "attachment")
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	MassTransferV1 = MassTransferV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}
