package proto

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// Benchmarks for the transaction serialization hot paths.
// Every fixture is checked once before its benchmarks run, so the benchmarks also work as smoke tests
// with `-benchtime=1x`. Operations that are not defined for a fixture (for example, binary unmarshalling of
// a protobuf-only version) are skipped.

const (
	benchScheme          = TestNetScheme
	benchDataEntries     = 100
	benchMassTransfers   = 100
	benchTimestamp       = 1_700_000_000_000
	benchSetScriptLength = 1024
)

type benchTxFixture struct {
	name string
	tx   Transaction
	pk   crypto.PublicKey
}

type benchVerifier interface {
	Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error)
}

func benchKeyPair(tb testing.TB, seed string) KeyPair {
	kp, err := NewKeyPair([]byte(seed))
	require.NoError(tb, err)
	return kp
}

func benchDigest(b byte) crypto.Digest {
	var d crypto.Digest
	for i := range d {
		d[i] = b + byte(i)
	}
	return d
}

func benchRecipient(tb testing.TB, seed string) Recipient {
	kp := benchKeyPair(tb, seed)
	addr, err := NewAddressFromPublicKey(benchScheme, kp.Public)
	require.NoError(tb, err)
	return NewRecipientFromAddress(addr)
}

func benchSign(tb testing.TB, tx Transaction, kp KeyPair) {
	require.NoError(tb, tx.Sign(benchScheme, kp.Secret))
}

func benchTransfer(tb testing.TB, v byte, kp KeyPair) Transaction {
	asset := NewOptionalAssetFromDigest(benchDigest(1))
	tx := NewUnsignedTransferWithProofs(v, kp.Public, *asset, NewOptionalAssetWaves(), benchTimestamp, 100_000,
		100_000, benchRecipient(tb, "recipient"), Attachment("benchmark attachment"))
	benchSign(tb, tx, kp)
	return tx
}

func benchData(tb testing.TB, v byte, kp KeyPair) Transaction {
	tx := NewUnsignedDataWithProofs(v, kp.Public, 1_500_000, benchTimestamp)
	for i := range benchDataEntries {
		var e DataEntry
		key := fmt.Sprintf("key-%03d", i)
		switch i % 4 {
		case 0:
			e = &IntegerDataEntry{Key: key, Value: int64(i) * 1_000_003}
		case 1:
			e = &BooleanDataEntry{Key: key, Value: i%3 == 0}
		case 2:
			e = &BinaryDataEntry{Key: key, Value: []byte(fmt.Sprintf("binary-value-%d", i))}
		default:
			e = &StringDataEntry{Key: key, Value: fmt.Sprintf("string-value-%d", i)}
		}
		require.NoError(tb, tx.AppendEntry(e))
	}
	benchSign(tb, tx, kp)
	return tx
}

func benchInvoke(tb testing.TB, v byte, kp KeyPair) Transaction {
	args := Arguments{
		NewIntegerArgument(12345),
		NewStringArgument("string argument"),
		&BinaryArgument{Value: []byte("binary argument")},
		&BooleanArgument{Value: true},
		&ListArgument{Items: Arguments{
			NewIntegerArgument(1), NewIntegerArgument(2), NewIntegerArgument(3),
			NewStringArgument("a"), NewStringArgument("b"),
		}},
	}
	payments := ScriptPayments{
		{Amount: 100_000, Asset: NewOptionalAssetWaves()},
		{Amount: 200_000, Asset: *NewOptionalAssetFromDigest(benchDigest(2))},
	}
	tx := NewUnsignedInvokeScriptWithProofs(v, kp.Public, benchRecipient(tb, "dapp"),
		NewFunctionCall("benchmarkFunction", args), payments, NewOptionalAssetWaves(), 500_000, benchTimestamp)
	benchSign(tb, tx, kp)
	return tx
}

func benchOrder(tb testing.TB, orderVersion byte, orderType OrderType, sender, matcher KeyPair) Order {
	amountAsset := *NewOptionalAssetFromDigest(benchDigest(3))
	priceAsset := NewOptionalAssetWaves()
	const (
		price      = 100_000
		amount     = 1_000
		expiration = benchTimestamp + 1_000_000
		matcherFee = 300_000
	)
	switch orderVersion {
	case 3:
		o := NewUnsignedOrderV3(sender.Public, matcher.Public, amountAsset, priceAsset, orderType, price, amount,
			benchTimestamp, expiration, matcherFee, NewOptionalAssetWaves())
		require.NoError(tb, o.Sign(benchScheme, sender.Secret))
		return o
	case 4:
		o := NewUnsignedOrderV4(sender.Public, matcher.Public, amountAsset, priceAsset, orderType, price, amount,
			benchTimestamp, expiration, matcherFee, NewOptionalAssetWaves(), OrderPriceModeDefault, nil)
		require.NoError(tb, o.Sign(benchScheme, sender.Secret))
		return o
	default:
		require.FailNow(tb, "unsupported order version")
		return nil
	}
}

func benchExchange(tb testing.TB, v byte, kp KeyPair) Transaction {
	orderVersion := byte(3)
	if v >= 3 {
		orderVersion = 4
	}
	buy := benchOrder(tb, orderVersion, Buy, benchKeyPair(tb, "buyer"), kp)
	sell := benchOrder(tb, orderVersion, Sell, benchKeyPair(tb, "seller"), kp)
	tx := NewUnsignedExchangeWithProofs(v, buy, sell, 100_000, 1_000, 300_000, 300_000, 700_000, benchTimestamp)
	benchSign(tb, tx, kp)
	return tx
}

func benchMassTransfer(tb testing.TB, v byte, kp KeyPair) Transaction {
	transfers := make([]MassTransferEntry, benchMassTransfers)
	for i := range transfers {
		transfers[i] = MassTransferEntry{
			Recipient: benchRecipient(tb, fmt.Sprintf("recipient-%d", i)),
			Amount:    uint64(i+1) * 1_000,
		}
	}
	tx := NewUnsignedMassTransferWithProofs(v, kp.Public, NewOptionalAssetWaves(), transfers, 5_100_000,
		benchTimestamp, Attachment("mass transfer"))
	benchSign(tb, tx, kp)
	return tx
}

func benchIssue(tb testing.TB, v byte, kp KeyPair) Transaction {
	tx := NewUnsignedIssueWithProofs(v, kp.Public, "BENCH", "Benchmark asset description", 1_000_000_000, 8,
		true, nil, benchTimestamp, 100_000_000)
	benchSign(tb, tx, kp)
	return tx
}

func benchSetScript(tb testing.TB, v byte, kp KeyPair) Transaction {
	script := make([]byte, benchSetScriptLength)
	for i := range script {
		script[i] = byte(i)
	}
	tx := NewUnsignedSetScriptWithProofs(v, kp.Public, script, 1_000_000, benchTimestamp)
	benchSign(tb, tx, kp)
	return tx
}

func benchTxFixtures(tb testing.TB) []benchTxFixture {
	kp := benchKeyPair(tb, "benchmark sender")
	type ctor struct {
		name string
		v    byte
		f    func(testing.TB, byte, KeyPair) Transaction
	}
	ctors := []ctor{
		{"TransferV2", 2, benchTransfer},
		{"TransferV3", 3, benchTransfer},
		{"Data100V1", 1, benchData},
		{"Data100V2", 2, benchData},
		{"InvokeScriptV1", 1, benchInvoke},
		{"InvokeScriptV2", 2, benchInvoke},
		{"ExchangeV2", 2, benchExchange},
		{"ExchangeV3", 3, benchExchange},
		{"MassTransfer100V1", 1, benchMassTransfer},
		{"MassTransfer100V2", 2, benchMassTransfer},
		{"IssueV2", 2, benchIssue},
		{"IssueV3", 3, benchIssue},
		{"SetScriptV1", 1, benchSetScript},
		{"SetScriptV2", 2, benchSetScript},
	}
	res := make([]benchTxFixture, len(ctors))
	for i, c := range ctors {
		res[i] = benchTxFixture{name: c.name, tx: c.f(tb, c.v, kp), pk: kp.Public}
	}
	return res
}

// benchIDField returns a pointer to the ID field of the transaction, so the benchmark can reset it cheaply.
func benchIDField(tb testing.TB, tx Transaction) **crypto.Digest {
	f := reflect.ValueOf(tx).Elem().FieldByName("ID")
	require.True(tb, f.IsValid(), "transaction %T has no ID field", tx)
	p, ok := reflect.TypeAssert[**crypto.Digest](f.Addr())
	require.True(tb, ok, "unexpected ID field type in %T", tx)
	return p
}

func isBenchProtobufTx(tx Transaction) bool {
	return IsProtobufTx(tx)
}

func BenchmarkTxMarshalBinary(b *testing.B) {
	for _, f := range benchTxFixtures(b) {
		b.Run(f.name, func(b *testing.B) {
			if isBenchProtobufTx(f.tx) {
				b.Skip("binary format is not used for protobuf versions")
			}
			_, err := f.tx.MarshalBinary(benchScheme)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				if _, err = f.tx.MarshalBinary(benchScheme); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTxBodyMarshalBinary(b *testing.B) {
	for _, f := range benchTxFixtures(b) {
		b.Run(f.name, func(b *testing.B) {
			if isBenchProtobufTx(f.tx) {
				b.Skip("binary format is not used for protobuf versions")
			}
			_, err := f.tx.BodyMarshalBinary(benchScheme)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				if _, err = f.tx.BodyMarshalBinary(benchScheme); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTxUnmarshalBinary(b *testing.B) {
	for _, f := range benchTxFixtures(b) {
		b.Run(f.name, func(b *testing.B) {
			if isBenchProtobufTx(f.tx) {
				b.Skip("binary format is not used for protobuf versions")
			}
			data, err := f.tx.MarshalBinary(benchScheme)
			require.NoError(b, err)
			tx, err := BytesToTransaction(data, benchScheme)
			require.NoError(b, err)
			id, err := tx.GetID(benchScheme)
			require.NoError(b, err)
			expectedID, err := f.tx.GetID(benchScheme)
			require.NoError(b, err)
			require.Equal(b, expectedID, id)
			b.ReportAllocs()
			for b.Loop() {
				if _, err = BytesToTransaction(data, benchScheme); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTxGenerateID(b *testing.B) {
	for _, f := range benchTxFixtures(b) {
		b.Run(f.name, func(b *testing.B) {
			idp := benchIDField(b, f.tx)
			expected := **idp
			*idp = nil
			require.NoError(b, f.tx.GenerateID(benchScheme))
			require.Equal(b, expected, **idp)
			b.ReportAllocs()
			for b.Loop() {
				*idp = nil
				if err := f.tx.GenerateID(benchScheme); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTxVerify(b *testing.B) {
	for _, f := range benchTxFixtures(b) {
		b.Run(f.name, func(b *testing.B) {
			v, ok := f.tx.(benchVerifier)
			require.True(b, ok)
			valid, err := v.Verify(benchScheme, f.pk)
			require.NoError(b, err)
			require.True(b, valid)
			b.ReportAllocs()
			for b.Loop() {
				if _, err = v.Verify(benchScheme, f.pk); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTxMarshalToProtobuf(b *testing.B) {
	for _, f := range benchTxFixtures(b) {
		b.Run(f.name, func(b *testing.B) {
			_, err := f.tx.MarshalToProtobuf(benchScheme)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				if _, err = f.tx.MarshalToProtobuf(benchScheme); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTxMarshalSignedToProtobuf(b *testing.B) {
	for _, f := range benchTxFixtures(b) {
		b.Run(f.name, func(b *testing.B) {
			_, err := f.tx.MarshalSignedToProtobuf(benchScheme)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				if _, err = f.tx.MarshalSignedToProtobuf(benchScheme); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTxSignedFromProtobuf(b *testing.B) {
	for _, f := range benchTxFixtures(b) {
		b.Run(f.name, func(b *testing.B) {
			data, err := f.tx.MarshalSignedToProtobuf(benchScheme)
			require.NoError(b, err)
			tx, err := SignedTxFromProtobuf(data)
			require.NoError(b, err)
			require.Equal(b, f.tx.GetType(), tx.GetType())
			require.Equal(b, f.tx.GetVersion(), tx.GetVersion())
			b.ReportAllocs()
			for b.Loop() {
				if _, err = SignedTxFromProtobuf(data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
