package byte_helpers

import (
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

type ReissueWithSigStruct struct {
	TransactionBytes []byte
	Transaction      *proto.ReissueWithSig
	MessageBytes     []byte
}

var ReissueWithSig ReissueWithSigStruct

type ReissueWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.ReissueWithProofs
	MessageBytes     []byte
}

var ReissueWithProofs ReissueWithProofsStruct

type TransferWithSigStruct struct {
	TransactionBytes []byte
	Transaction      *proto.TransferWithSig
	MessageBytes     []byte
}

var TransferWithSig TransferWithSigStruct

type BurnWithSigStruct struct {
	TransactionBytes []byte
	Transaction      *proto.BurnWithSig
	MessageBytes     []byte
}

var BurnWithSig BurnWithSigStruct

type BurnWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.BurnWithProofs
	MessageBytes     []byte
}

var BurnWithProofs BurnWithProofsStruct

type TransferWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.TransferWithProofs
	MessageBytes     []byte
}

var TransferWithProofs TransferWithProofsStruct

type IssueWithSigStruct struct {
	TransactionBytes []byte
	Transaction      *proto.IssueWithSig
	MessageBytes     []byte
}

var IssueWithSig IssueWithSigStruct

type IssueWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.IssueWithProofs
	MessageBytes     []byte
}

var IssueWithProofs IssueWithProofsStruct

type MassTransferWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.MassTransferWithProofs
	MessageBytes     []byte
}

var MassTransferWithProofs MassTransferWithProofsStruct

type ExchangeWithSigStruct struct {
	TransactionBytes []byte
	Transaction      *proto.ExchangeWithSig
	MessageBytes     []byte
}

var ExchangeWithSig ExchangeWithSigStruct

type ExchangeWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.ExchangeWithProofs
	MessageBytes     []byte
}

var ExchangeWithProofs ExchangeWithProofsStruct

type SetAssetScriptWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.SetAssetScriptWithProofs
	MessageBytes     []byte
}

var SetAssetScriptWithProofs SetAssetScriptWithProofsStruct

type InvokeScriptWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.InvokeScriptWithProofs
	MessageBytes     []byte
}

var InvokeScriptWithProofs InvokeScriptWithProofsStruct

type LeaseWithSigStruct struct {
	TransactionBytes []byte
	Transaction      *proto.LeaseWithSig
	MessageBytes     []byte
}

var LeaseWithSig LeaseWithSigStruct

type LeaseWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.LeaseWithProofs
	MessageBytes     []byte
}

var LeaseWithProofs LeaseWithProofsStruct

type LeaseCancelWithSigStruct struct {
	TransactionBytes []byte
	Transaction      *proto.LeaseCancelWithSig
	MessageBytes     []byte
}

var LeaseCancelWithSig LeaseCancelWithSigStruct

type LeaseCancelWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.LeaseCancelWithProofs
	MessageBytes     []byte
}

var LeaseCancelWithProofs LeaseCancelWithProofsStruct

type DataWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.DataWithProofs
	MessageBytes     []byte
}

var DataWithProofs DataWithProofsStruct

type SponsorshipWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.SponsorshipWithProofs
	MessageBytes     []byte
}

var SponsorshipWithProofs SponsorshipWithProofsStruct

type CreateAliasWithSigStruct struct {
	TransactionBytes []byte
	Transaction      *proto.CreateAliasWithSig
	MessageBytes     []byte
}

var CreateAliasWithSig CreateAliasWithSigStruct

type CreateAliasWithProofsStruct struct {
	TransactionBytes []byte
	Transaction      *proto.CreateAliasWithProofs
	MessageBytes     []byte
}

var CreateAliasWithProofs CreateAliasWithProofsStruct

var sk, pk, _ = crypto.GenerateKeyPair([]byte("test"))

func init() {
	initGenesis()
	initPayment()
	initTransferWithSig()
	initTransferWithProofs()
	initIssueWithSig()
	initIssueWithProofs()
	initReissueWithSig()
	initReissueWithProofs()
	initBurnWithSig()
	initBurnWithProofs()
	initMassTransferWithProofs()
	initExchangeWithSig()
	initExchangeWithProofs()
	initSetAssetScriptWithProofs()
	initInvokeScriptWithProofs()
	initLeaseWithSig()
	initLeaseWithProofs()
	initLeaseCancelWithSig()
	initLeaseCancelWithProofs()
	initDataWithProofs()
	initSponsorshipWithProofs()
	initCreateAliasWithSig()
	initCreateAliasWithProofs()
}

func initTransferWithSig() {
	t := newTransferWithSigBuilder().MustBuild()
	b, _ := t.MarshalBinary(proto.MainNetScheme)
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	TransferWithSig = TransferWithSigStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

var _, publicKey, _ = crypto.GenerateKeyPair([]byte("test"))
var address, _ = proto.NewAddressFromPublicKey(proto.MainNetScheme, publicKey)

func initTransferWithProofs() {
	sk, pk, _ := crypto.GenerateKeyPair([]byte("test"))
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)
	if err != nil {
		panic(err)
	}
	t := proto.NewUnsignedTransferWithProofs(
		2,
		pk,
		proto.OptionalAsset{},
		proto.OptionalAsset{},
		TIMESTAMP,
		100000,
		10000,
		proto.NewRecipientFromAddress(addr),
		[]byte("abc"),
	)

	_ = t.Sign(proto.MainNetScheme, sk)
	b, _ := t.MarshalBinary(proto.MainNetScheme)
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	TransferWithProofs = TransferWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}

}

func initIssueWithSig() {

	sk, pk, _ := crypto.GenerateKeyPair([]byte("test"))

	t := proto.NewUnsignedIssueWithSig(
		pk,
		"name",
		"description",
		1000,
		4,
		false,
		proto.NewTimestampFromTime(time.Now()),
		10000)

	_ = t.Sign(proto.MainNetScheme, sk)
	b, _ := t.MarshalBinary(proto.MainNetScheme)
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	IssueWithSig = IssueWithSigStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initIssueWithProofs() {

	sk, pk, _ := crypto.GenerateKeyPair([]byte("test"))

	t := proto.NewUnsignedIssueWithProofs(2, pk, "name", "description", 1000, 4, false, []byte("script"), proto.NewTimestampFromTime(time.Now()), 10000)

	_ = t.Sign(proto.MainNetScheme, sk)
	b, _ := t.MarshalBinary(proto.MainNetScheme)
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	IssueWithProofs = IssueWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initGenesis() {
	sk, pk, _ := crypto.GenerateKeyPair([]byte("test"))
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)
	if err != nil {
		panic(err)
	}
	t := proto.NewUnsignedGenesis(addr, 100000, TIMESTAMP)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, _ := t.MarshalBinary(proto.MainNetScheme)
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

	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
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

func initReissueWithSig() {

	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}

	t := proto.NewUnsignedReissueWithSig(pk, d, 100000, true, TIMESTAMP, 10000)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	ReissueWithSig = ReissueWithSigStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initReissueWithProofs() {
	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}

	t := proto.NewUnsignedReissueWithProofs(2, pk, d, 100000, true, TIMESTAMP, 10000)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	ReissueWithProofs = ReissueWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initBurnWithSig() {

	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}

	t := proto.NewUnsignedBurnWithSig(pk, d, 100000, TIMESTAMP, 10000)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	BurnWithSig = BurnWithSigStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initBurnWithProofs() {
	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}

	t := proto.NewUnsignedBurnWithProofs(2, pk, d, 100000, TIMESTAMP, 10000)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	BurnWithProofs = BurnWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initMassTransferWithProofs() {
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
	t := proto.NewUnsignedMassTransferWithProofs(1, pk, *proto.NewOptionalAssetFromDigest(d), []proto.MassTransferEntry{entry}, 10000, TIMESTAMP, []byte("attachment"))
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	MassTransferWithProofs = MassTransferWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initExchangeWithSig() {
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

	_ = buyOrder.Sign(proto.MainNetScheme, sk)

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

	_ = sellOrder.Sign(proto.MainNetScheme, sk)

	t := proto.NewUnsignedExchangeWithSig(
		buyOrder,
		sellOrder,
		100000,
		100000,
		10000,
		10000,
		10000,
		TIMESTAMP,
	)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	ExchangeWithSig = ExchangeWithSigStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initExchangeWithProofs() {
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

	_ = buyOrder.Sign(proto.MainNetScheme, sk)

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

	_ = sellOrder.Sign(proto.MainNetScheme, sk)

	t := proto.NewUnsignedExchangeWithProofs(
		2,
		buyOrder,
		sellOrder,
		100000,
		100000,
		10000,
		10000,
		10000,
		TIMESTAMP,
	)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	ExchangeWithProofs = ExchangeWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

// SetAssetScriptWithProofs
func initSetAssetScriptWithProofs() {
	d, err := crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	if err != nil {
		panic(err)
	}

	t := proto.NewUnsignedSetAssetScriptWithProofs(1, pk, d, []byte("hello"), 10000, TIMESTAMP)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	SetAssetScriptWithProofs = SetAssetScriptWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

// InvokeScriptWithProofs
func initInvokeScriptWithProofs() {
	asset := proto.NewOptionalAssetFromDigest(Digest)

	t := proto.NewUnsignedInvokeScriptWithProofs(1, pk, proto.NewRecipientFromAddress(address), proto.FunctionCall{
		Default:   true,
		Name:      "funcname",
		Arguments: proto.Arguments{proto.NewStringArgument("StringArgument")},
	}, proto.ScriptPayments{proto.ScriptPayment{
		Amount: 100000,
		Asset:  *asset,
	}}, *asset, 10000, TIMESTAMP)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	InvokeScriptWithProofs = InvokeScriptWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initLeaseWithSig() {
	addr, _ := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)

	t := proto.NewUnsignedLeaseWithSig(
		pk,
		proto.NewRecipientFromAddress(addr),
		100000, 10000, TIMESTAMP)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	LeaseWithSig = LeaseWithSigStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initLeaseWithProofs() {
	addr, _ := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)
	t := proto.NewUnsignedLeaseWithProofs(
		2,
		pk,
		proto.NewRecipientFromAddress(addr),
		100000, 10000, TIMESTAMP)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	LeaseWithProofs = LeaseWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initLeaseCancelWithSig() {
	t := proto.NewUnsignedLeaseCancelWithSig(
		pk,
		Digest,
		10000, TIMESTAMP)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	LeaseCancelWithSig = LeaseCancelWithSigStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initLeaseCancelWithProofs() {
	t := proto.NewUnsignedLeaseCancelWithProofs(2, pk, Digest, 10000, TIMESTAMP)
	_ = t.Sign(proto.MainNetScheme, sk)
	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	LeaseCancelWithProofs = LeaseCancelWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initDataWithProofs() {
	t := proto.NewUnsignedDataWithProofs(
		1,
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

	_ = t.Sign(proto.MainNetScheme, sk)

	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	DataWithProofs = DataWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initSponsorshipWithProofs() {
	t := proto.NewUnsignedSponsorshipWithProofs(
		1,
		pk,
		Digest,
		1000,
		10000,
		TIMESTAMP)

	_ = t.Sign(proto.MainNetScheme, sk)

	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	SponsorshipWithProofs = SponsorshipWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initCreateAliasWithSig() {
	alias := proto.NewAlias(proto.MainNetScheme, "testalias")
	t := proto.NewUnsignedCreateAliasWithSig(
		pk,
		*alias,
		10000,
		TIMESTAMP)

	_ = t.Sign(proto.MainNetScheme, sk)

	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	CreateAliasWithSig = CreateAliasWithSigStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}

func initCreateAliasWithProofs() {
	alias := proto.NewAlias(proto.MainNetScheme, "testalias")
	t := proto.NewUnsignedCreateAliasWithProofs(
		2,
		pk,
		*alias,
		10000,
		TIMESTAMP)

	_ = t.Sign(proto.MainNetScheme, sk)

	b, err := t.MarshalBinary(proto.MainNetScheme)
	if err != nil {
		panic(err)
	}
	tm := proto.TransactionMessage{
		Transaction: b,
	}
	tmb, _ := tm.MarshalBinary()

	CreateAliasWithProofs = CreateAliasWithProofsStruct{
		TransactionBytes: b,
		Transaction:      t,
		MessageBytes:     tmb,
	}
}
