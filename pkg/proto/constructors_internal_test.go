package proto

import (
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Old reflection-based tables kept as oracles for the switch constructors.

func bytesToTransactionsV2Oracle() map[TransactionType]reflect.Type {
	return map[TransactionType]reflect.Type{ //nolint:exhaustive // Only types with binary proofs format.
		IssueTransaction:          reflect.TypeFor[IssueWithProofs](),
		TransferTransaction:       reflect.TypeFor[TransferWithProofs](),
		ReissueTransaction:        reflect.TypeFor[ReissueWithProofs](),
		BurnTransaction:           reflect.TypeFor[BurnWithProofs](),
		ExchangeTransaction:       reflect.TypeFor[ExchangeWithProofs](),
		LeaseTransaction:          reflect.TypeFor[LeaseWithProofs](),
		LeaseCancelTransaction:    reflect.TypeFor[LeaseCancelWithProofs](),
		CreateAliasTransaction:    reflect.TypeFor[CreateAliasWithProofs](),
		DataTransaction:           reflect.TypeFor[DataWithProofs](),
		SetScriptTransaction:      reflect.TypeFor[SetScriptWithProofs](),
		SponsorshipTransaction:    reflect.TypeFor[SponsorshipWithProofs](),
		SetAssetScriptTransaction: reflect.TypeFor[SetAssetScriptWithProofs](),
		InvokeScriptTransaction:   reflect.TypeFor[InvokeScriptWithProofs](),
	}
}

func bytesToTransactionsV1Oracle() map[TransactionType]reflect.Type {
	return map[TransactionType]reflect.Type{ //nolint:exhaustive // Only types with V1 binary format.
		GenesisTransaction:      reflect.TypeFor[Genesis](),
		PaymentTransaction:      reflect.TypeFor[Payment](),
		IssueTransaction:        reflect.TypeFor[IssueWithSig](),
		TransferTransaction:     reflect.TypeFor[TransferWithSig](),
		ReissueTransaction:      reflect.TypeFor[ReissueWithSig](),
		BurnTransaction:         reflect.TypeFor[BurnWithSig](),
		ExchangeTransaction:     reflect.TypeFor[ExchangeWithSig](),
		LeaseTransaction:        reflect.TypeFor[LeaseWithSig](),
		LeaseCancelTransaction:  reflect.TypeFor[LeaseCancelWithSig](),
		CreateAliasTransaction:  reflect.TypeFor[CreateAliasWithSig](),
		MassTransferTransaction: reflect.TypeFor[MassTransferWithProofs](),
	}
}

func bytesToDataEntryOracle() map[DataValueType]reflect.Type {
	return map[DataValueType]reflect.Type{
		DataInteger: reflect.TypeFor[IntegerDataEntry](),
		DataBoolean: reflect.TypeFor[BooleanDataEntry](),
		DataString:  reflect.TypeFor[StringDataEntry](),
		DataBinary:  reflect.TypeFor[BinaryDataEntry](),
		DataDelete:  reflect.TypeFor[DeleteDataEntry](),
	}
}

func checkCtor[T ~byte](t *testing.T, oracle map[T]reflect.Type, key T, got any) {
	t.Helper()
	typ, ok := oracle[key]
	if !ok {
		assert.Nil(t, got, "key %v", key)
		return
	}
	require.NotNil(t, got, "key %v", key)
	gt := reflect.TypeOf(got)
	require.Equal(t, reflect.Pointer, gt.Kind(), "key %v", key)
	assert.Equal(t, typ, gt.Elem(), "key %v", key)
	assert.True(t, reflect.ValueOf(got).Elem().IsZero(), "key %v: constructed value must be zero", key)
}

func TestTxConstructorsMatchOracle(t *testing.T) {
	for i := range math.MaxUint8 + 1 {
		tt := TransactionType(i)
		pt := newBinaryTxWithProofs(tt)
		v1 := newBinaryTxV1(tt)
		var ptAny, v1Any any
		if pt != nil {
			ptAny = pt
		}
		if v1 != nil {
			v1Any = v1
		}
		checkCtor(t, bytesToTransactionsV2Oracle(), tt, ptAny)
		checkCtor(t, bytesToTransactionsV1Oracle(), tt, v1Any)
	}
}

func TestDataEntryConstructorMatchesOracle(t *testing.T) {
	for i := range math.MaxUint8 + 1 {
		vt := DataValueType(i)
		var got any
		if e := newDataEntryByType(vt); e != nil {
			got = e
		}
		checkCtor(t, bytesToDataEntryOracle(), vt, got)
	}
}

func TestNewDataEntryFromValueBytesRoundTrip(t *testing.T) {
	entries := []DataEntry{
		&IntegerDataEntry{Value: -12345},
		&BooleanDataEntry{Value: true},
		&StringDataEntry{Value: "some string"},
		&BinaryDataEntry{Value: []byte{1, 2, 3}},
		&DeleteDataEntry{},
	}
	for _, e := range entries {
		vb, err := e.MarshalValue()
		require.NoError(t, err)
		got, err := NewDataEntryFromValueBytes(vb)
		require.NoError(t, err)
		assert.Equal(t, e, got)
	}
	_, err := NewDataEntryFromValueBytes(nil)
	assert.Error(t, err)
	_, err = NewDataEntryFromValueBytes([]byte{0x7f})
	assert.Error(t, err)
}

func TestBytesToTransactionInvalidType(t *testing.T) {
	_, err := BytesToTransaction([]byte{0, 0xff, 0}, TestNetScheme)
	assert.ErrorContains(t, err, "invalid transaction type")
	_, err = BytesToTransaction([]byte{0xfe, 0}, TestNetScheme)
	assert.ErrorContains(t, err, "invalid transaction type")
	_, err = BytesToTransaction([]byte{0}, TestNetScheme)
	assert.Error(t, err)
}

func isProtobufTxOracle(tx Transaction) bool {
	protobufVersion, ok := ProtobufTransactionsVersions[tx.GetTypeInfo().Type]
	if !ok {
		return false
	}
	return tx.GetVersion() >= protobufVersion
}

func TestIsProtobufTxMatchesOracle(t *testing.T) {
	for tt, minV := range ProtobufTransactionsVersions {
		require.NotZero(t, minV, "type %v", tt)
		assert.Equal(t, minV, protobufTxVersions[tt], "type %v", tt)
	}
	for i := range math.MaxUint8 + 1 {
		if _, ok := ProtobufTransactionsVersions[TransactionType(i)]; !ok {
			assert.Zero(t, protobufTxVersions[i], "type %d", i)
		}
	}
	txs := []Transaction{
		&Genesis{Type: GenesisTransaction, Version: 1},
		&Genesis{Type: GenesisTransaction, Version: 2},
		&Payment{Type: PaymentTransaction, Version: 1},
		&Payment{Type: PaymentTransaction, Version: 2},
	}
	for v := byte(1); v <= 4; v++ {
		txs = append(txs,
			&IssueWithProofs{Type: IssueTransaction, Version: v},
			&TransferWithProofs{Type: TransferTransaction, Version: v},
			&ReissueWithProofs{Type: ReissueTransaction, Version: v},
			&BurnWithProofs{Type: BurnTransaction, Version: v},
			&ExchangeWithProofs{Type: ExchangeTransaction, Version: v},
			&LeaseWithProofs{Type: LeaseTransaction, Version: v},
			&LeaseCancelWithProofs{Type: LeaseCancelTransaction, Version: v},
			&CreateAliasWithProofs{Type: CreateAliasTransaction, Version: v},
			&MassTransferWithProofs{Type: MassTransferTransaction, Version: v},
			&DataWithProofs{Type: DataTransaction, Version: v},
			&SetScriptWithProofs{Type: SetScriptTransaction, Version: v},
			&SponsorshipWithProofs{Type: SponsorshipTransaction, Version: v},
			&SetAssetScriptWithProofs{Type: SetAssetScriptTransaction, Version: v},
			&InvokeScriptWithProofs{Type: InvokeScriptTransaction, Version: v},
			&UpdateAssetInfoWithProofs{Type: UpdateAssetInfoTransaction, Version: v},
			&InvokeExpressionTransactionWithProofs{Type: InvokeExpressionTransaction, Version: v},
			&CommitToGenerationWithProofs{Type: CommitToGenerationTransaction, Version: v},
			&IssueWithSig{Type: IssueTransaction, Version: v},
			&TransferWithSig{Type: TransferTransaction, Version: v},
			&ExchangeWithSig{Type: ExchangeTransaction, Version: v},
		)
	}
	for _, tx := range txs {
		assert.Equal(t, isProtobufTxOracle(tx), IsProtobufTx(tx), "tx %T v%d", tx, tx.GetVersion())
	}
}

func BenchmarkIsProtobufTx(b *testing.B) {
	tx := Transaction(&TransferWithProofs{Type: TransferTransaction, Version: 3})
	b.ReportAllocs()
	for b.Loop() {
		_ = IsProtobufTx(tx)
	}
}
