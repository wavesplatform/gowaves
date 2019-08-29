package byte_helpers

import (
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const TIMESTAMP = proto.Timestamp(1544715621)

var Digest = crypto.MustDigestFromBase58("WmryL34P6UwwUphNbhjBRwiCWxX15Nf5D8T7AmQY7yx")

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

type IssueV2Struct struct {
	TransactionBytes []byte
	Transaction      *proto.IssueV2
	MessageBytes     []byte
}

var IssueV2 IssueV2Struct

type MassTransferV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.MassTransferV1
	MessageBytes     []byte
}

var MassTransferV1 MassTransferV1Struct

type ExchangeV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.ExchangeV1
	MessageBytes     []byte
}

var ExchangeV1 ExchangeV1Struct

type ExchangeV2Struct struct {
	TransactionBytes []byte
	Transaction      *proto.ExchangeV2
	MessageBytes     []byte
}

var ExchangeV2 ExchangeV2Struct

type SetAssetScriptV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.SetAssetScriptV1
	MessageBytes     []byte
}

var SetAssetScriptV1 SetAssetScriptV1Struct

type InvokeScriptV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.InvokeScriptV1
	MessageBytes     []byte
}

var InvokeScriptV1 InvokeScriptV1Struct

type LeaseV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.LeaseV1
	MessageBytes     []byte
}

var LeaseV1 LeaseV1Struct

type LeaseV2Struct struct {
	TransactionBytes []byte
	Transaction      *proto.LeaseV2
	MessageBytes     []byte
}

var LeaseV2 LeaseV2Struct

type LeaseCancelV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.LeaseCancelV1
	MessageBytes     []byte
}

var LeaseCancelV1 LeaseCancelV1Struct

type LeaseCancelV2Struct struct {
	TransactionBytes []byte
	Transaction      *proto.LeaseCancelV2
	MessageBytes     []byte
}

var LeaseCancelV2 LeaseCancelV2Struct

type DataV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.DataV1
	MessageBytes     []byte
}

var DataV1 DataV1Struct

type SponsorshipV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.SponsorshipV1
	MessageBytes     []byte
}

var SponsorshipV1 SponsorshipV1Struct

type CreateAliasV1Struct struct {
	TransactionBytes []byte
	Transaction      *proto.CreateAliasV1
	MessageBytes     []byte
}

var CreateAliasV1 CreateAliasV1Struct

type CreateAliasV2Struct struct {
	TransactionBytes []byte
	Transaction      *proto.CreateAliasV2
	MessageBytes     []byte
}

var CreateAliasV2 CreateAliasV2Struct

var sk, pk, _ = crypto.GenerateKeyPair([]byte("test"))

func init() {
	initGenesis()
	initPayment()
	initTransferV1()
	initTransferV2()
	initIssueV1()
	initIssueV2()
	initReissueV1()
	initReissueV2()
	initBurnV1()
	initBurnV2()
	initMassTransferV1()
	initExchangeV1()
	initExchangeV2()
	initSetAssetScriptV1()
	initInvokeScriptV1()
	initLeaseV1()
	initLeaseV2()
	initLeaseCancelV1()
	initLeaseCancelV2()
	initDataV1()
	initSponsorshipV1()
	initCreateAliasV1()
	initCreateAliasV2()
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

var _, publicKey, _ = crypto.GenerateKeyPair([]byte("test"))
var address, _ = proto.NewAddressFromPublicKey(proto.MainNetScheme, publicKey)

func initTransferV2() {
	sk, pk, _ := crypto.GenerateKeyPair([]byte("test"))
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

	sk, pk, _ := crypto.GenerateKeyPair([]byte("test"))

	t := proto.NewUnsignedIssueV1(
		pk,
		"name",
		"description",
		1000,
		4,
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

func initIssueV2() {

	sk, pk, _ := crypto.GenerateKeyPair([]byte("test"))

	t := proto.NewUnsignedIssueV2(
		proto.MainNetScheme,
		pk,
		"name",
		"description",
		1000,
		4,
		false,
		[]byte("script"),
		proto.NewTimestampFromTime(time.Now()),
		10000)

	_ = t.Sign(sk)
	b, _ := t.MarshalBinary()
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	IssueV2 = IssueV2Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initGenesis() {
	_, pk, _ := crypto.GenerateKeyPair([]byte("test"))
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
	sk, pk, _ := crypto.GenerateKeyPair([]byte("test"))
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

func initExchangeV1() {
	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}
	_, matcherPk, _ := crypto.GenerateKeyPair([]byte("test1"))

	buyOrder := proto.NewUnsignedOrderV1(
		pk,
		matcherPk,
		*proto.NewOptionalAssetFromDigest(d),
		*proto.NewOptionalAssetFromDigest(d),
		proto.Buy,
		100000,
		10000,
		TIMESTAMP,
		TIMESTAMP,
		10000)

	_ = buyOrder.Sign(sk)

	sellOrder := proto.NewUnsignedOrderV1(
		pk,
		matcherPk,
		*proto.NewOptionalAssetFromDigest(d),
		*proto.NewOptionalAssetFromDigest(d),
		proto.Sell,
		100000,
		10000,
		TIMESTAMP,
		TIMESTAMP,
		10000)

	_ = sellOrder.Sign(sk)

	t := proto.NewUnsignedExchangeV1(
		buyOrder,
		sellOrder,
		100000,
		100000,
		10000,
		10000,
		10000,
		TIMESTAMP,
	)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	ExchangeV1 = ExchangeV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initExchangeV2() {
	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}
	_, matcherPk, _ := crypto.GenerateKeyPair([]byte("test1"))

	buyOrder := proto.NewUnsignedOrderV1(
		pk,
		matcherPk,
		*proto.NewOptionalAssetFromDigest(d),
		*proto.NewOptionalAssetFromDigest(d),
		proto.Buy,
		100000,
		10000,
		TIMESTAMP,
		TIMESTAMP,
		10000)

	_ = buyOrder.Sign(sk)

	sellOrder := proto.NewUnsignedOrderV1(
		pk,
		matcherPk,
		*proto.NewOptionalAssetFromDigest(d),
		*proto.NewOptionalAssetFromDigest(d),
		proto.Sell,
		100000,
		10000,
		TIMESTAMP,
		TIMESTAMP,
		10000)

	_ = sellOrder.Sign(sk)

	t := proto.NewUnsignedExchangeV2(
		buyOrder,
		sellOrder,
		100000,
		100000,
		10000,
		10000,
		10000,
		TIMESTAMP,
	)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	ExchangeV2 = ExchangeV2Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

//SetAssetScriptV1
func initSetAssetScriptV1() {
	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}

	t := proto.NewUnsignedSetAssetScriptV1(proto.MainNetScheme, pk, d, []byte("hello"), 10000, TIMESTAMP)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	SetAssetScriptV1 = SetAssetScriptV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

//InvokeScriptV1
func initInvokeScriptV1() {
	asset := proto.NewOptionalAssetFromDigest(Digest)

	t := proto.NewUnsignedInvokeScriptV1(
		proto.MainNetScheme,
		pk,
		proto.NewRecipientFromAddress(address),
		proto.FunctionCall{
			Default:   true,
			Name:      "funcname",
			Arguments: proto.Arguments{proto.NewStringArgument("StringArgument")},
		},
		proto.ScriptPayments{proto.ScriptPayment{
			Amount: 100000,
			Asset:  *asset,
		}},
		*asset,
		10000,
		TIMESTAMP)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	InvokeScriptV1 = InvokeScriptV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initLeaseV1() {
	addr, _ := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)

	t := proto.NewUnsignedLeaseV1(
		pk,
		proto.NewRecipientFromAddress(addr),
		100000, 10000, TIMESTAMP)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	LeaseV1 = LeaseV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initLeaseV2() {
	addr, _ := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)
	t := proto.NewUnsignedLeaseV2(
		pk,
		proto.NewRecipientFromAddress(addr),
		100000, 10000, TIMESTAMP)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	LeaseV2 = LeaseV2Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initLeaseCancelV1() {
	t := proto.NewUnsignedLeaseCancelV1(
		pk,
		Digest,
		10000, TIMESTAMP)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	LeaseCancelV1 = LeaseCancelV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initLeaseCancelV2() {
	t := proto.NewUnsignedLeaseCancelV2(
		proto.MainNetScheme,
		pk,
		Digest,
		10000, TIMESTAMP)
	_ = t.Sign(sk)
	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	LeaseCancelV2 = LeaseCancelV2Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initDataV1() {
	t := proto.NewUnsignedData(
		pk,
		10000,
		TIMESTAMP)

	err := t.AppendEntry(&proto.BinaryDataEntry{
		Key:   "bin",
		Value: []byte("hello"),
	})
	if err != nil {
		panic(err)
	}

	err = t.AppendEntry(&proto.StringDataEntry{
		Key:   "str",
		Value: "hello",
	})
	if err != nil {
		panic(err)
	}

	err = t.AppendEntry(&proto.BooleanDataEntry{
		Key:   "bool",
		Value: true,
	})
	if err != nil {
		panic(err)
	}

	err = t.AppendEntry(&proto.IntegerDataEntry{
		Key:   "int",
		Value: 5,
	})
	if err != nil {
		panic(err)
	}

	_ = t.Sign(sk)

	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	DataV1 = DataV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initSponsorshipV1() {
	t := proto.NewUnsignedSponsorshipV1(
		pk,
		Digest,
		1000,
		10000,
		TIMESTAMP)

	_ = t.Sign(sk)

	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	SponsorshipV1 = SponsorshipV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initCreateAliasV1() {
	alias := proto.NewAlias(proto.MainNetScheme, "testalias")
	t := proto.NewUnsignedCreateAliasV1(
		pk,
		*alias,
		10000,
		TIMESTAMP)

	_ = t.Sign(sk)

	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	CreateAliasV1 = CreateAliasV1Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initCreateAliasV2() {
	alias := proto.NewAlias(proto.MainNetScheme, "testalias")
	t := proto.NewUnsignedCreateAliasV2(
		pk,
		*alias,
		10000,
		TIMESTAMP)

	_ = t.Sign(sk)

	b, err := t.MarshalBinary()
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	CreateAliasV2 = CreateAliasV2Struct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}
