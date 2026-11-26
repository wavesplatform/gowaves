package state

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

// Benchmarks for the encoding and decoding of hot state records.
// Each benchmark checks the round trip once, so they also work as tests with `-benchtime=1x`.

type benchRecord interface {
	marshalBinary() ([]byte, error)
	unmarshalBinary([]byte) error
}

type benchRecordCase struct {
	name  string
	value benchRecord
	fresh func() benchRecord
}

func benchRecordDigest(b byte) crypto.Digest {
	var d crypto.Digest
	for i := range d {
		d[i] = b + byte(i)
	}
	return d
}

func benchRecordPK(b byte) crypto.PublicKey {
	return crypto.PublicKey(benchRecordDigest(b))
}

func benchRecordAddress(tb testing.TB) proto.WavesAddress {
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, benchRecordPK(3))
	require.NoError(tb, err)
	return addr
}

func benchRecordCases(tb testing.TB) []benchRecordCase {
	originTx := benchRecordDigest(4)
	cancelTx := benchRecordDigest(5)
	var tail [proto.AssetIDTailSize]byte
	copy(tail[:], []byte("asset id tail bytes"))
	return []benchRecordCase{
		{
			name: "wavesBalanceRecord",
			value: &wavesBalanceRecord{balanceProfile{
				Balance: 123_456_789_012, LeaseIn: 1_000_000, LeaseOut: 2_000_000, Deposit: 0,
			}},
			fresh: func() benchRecord { return &wavesBalanceRecord{} },
		},
		{
			name:  "assetBalanceRecord",
			value: &assetBalanceRecord{Balance: math.MaxInt64},
			fresh: func() benchRecord { return &assetBalanceRecord{} },
		},
		{
			name: "assetConstInfo",
			value: &assetConstInfo{
				Tail: tail, Issuer: benchRecordPK(1), Decimals: 8, IssueHeight: 4_000_000, IsNFT: false,
				IssueSequenceInBlock: 12,
			},
			fresh: func() benchRecord { return &assetConstInfo{} },
		},
		{
			name: "leasingActive",
			value: &leasing{
				SenderPK: benchRecordPK(2), RecipientAddr: benchRecordAddress(tb), Amount: 1_000_000_000,
				OriginHeight: 3_000_000, Status: LeaseActive, OriginTransactionID: &originTx,
			},
			fresh: func() benchRecord { return &leasing{} },
		},
		{
			name: "leasingCancelled",
			value: &leasing{
				SenderPK: benchRecordPK(2), RecipientAddr: benchRecordAddress(tb), Amount: 1_000_000_000,
				OriginHeight: 3_000_000, Status: LeaseCancelled, OriginTransactionID: &originTx,
				CancelHeight: 3_100_000, CancelTransactionID: &cancelTx,
			},
			fresh: func() benchRecord { return &leasing{} },
		},
		{
			name: "scriptBasicInfoRecord",
			value: &scriptBasicInfoRecord{
				PK: benchRecordPK(6), ScriptLen: 4096, LibraryVersion: ast.LibV6, HasVerifier: true, IsDApp: true,
			},
			fresh: func() benchRecord { return &scriptBasicInfoRecord{} },
		},
		{
			name: "estimationRecord",
			value: &estimationRecord{
				EstimatorVersion: 4,
				Estimation: ride.TreeEstimation{
					Estimation: 2_500, Verifier: 200,
					Functions: map[string]int{
						"deposit": 1_200, "withdraw": 2_500, "swap": 1_900, "stake": 800, "unstake": 950,
					},
				},
			},
			fresh: func() benchRecord { return &estimationRecord{} },
		},
	}
}

func BenchmarkStateRecordMarshal(b *testing.B) {
	for _, c := range benchRecordCases(b) {
		b.Run(c.name, func(b *testing.B) {
			data, err := c.value.marshalBinary()
			require.NoError(b, err)
			res := c.fresh()
			require.NoError(b, res.unmarshalBinary(data))
			require.Equal(b, c.value, res)
			b.ReportAllocs()
			for b.Loop() {
				if _, err = c.value.marshalBinary(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkStateRecordUnmarshal(b *testing.B) {
	for _, c := range benchRecordCases(b) {
		b.Run(c.name, func(b *testing.B) {
			data, err := c.value.marshalBinary()
			require.NoError(b, err)
			res := c.fresh()
			b.ReportAllocs()
			for b.Loop() {
				if err = res.unmarshalBinary(data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func benchDataEntries() []proto.DataEntry {
	return []proto.DataEntry{
		&proto.IntegerDataEntry{Key: "integer", Value: math.MaxInt64},
		&proto.BooleanDataEntry{Key: "boolean", Value: true},
		&proto.BinaryDataEntry{Key: "binary", Value: make([]byte, benchDataEntryDataSize)},
		&proto.StringDataEntry{Key: "string", Value: fmt.Sprintf("%0*d", benchDataEntryDataSize, 42)},
		&proto.DeleteDataEntry{Key: "delete"},
	}
}

func benchDataEntryName(e proto.DataEntry) string {
	return e.GetKey()
}

// BenchmarkDataEntryRecordWrite measures the data entry write path: value encoding and record encoding.
func BenchmarkDataEntryRecordWrite(b *testing.B) {
	for _, e := range benchDataEntries() {
		b.Run(benchDataEntryName(e), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				valueBytes, err := e.MarshalValue()
				if err != nil {
					b.Fatal(err)
				}
				record := dataEntryRecord{value: valueBytes}
				if _, err = record.marshalBinary(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDataEntryRecordRead measures the data entry read path: record decoding and value decoding.
func BenchmarkDataEntryRecordRead(b *testing.B) {
	for _, e := range benchDataEntries() {
		b.Run(benchDataEntryName(e), func(b *testing.B) {
			valueBytes, err := e.MarshalValue()
			require.NoError(b, err)
			record := dataEntryRecord{value: valueBytes}
			recordBytes, err := record.marshalBinary()
			require.NoError(b, err)
			var check dataEntryRecord
			require.NoError(b, check.unmarshalBinary(recordBytes))
			decoded, err := proto.NewDataEntryFromValueBytes(check.value)
			require.NoError(b, err)
			decoded.SetKey(e.GetKey())
			require.Equal(b, e, decoded)
			b.ReportAllocs()
			for b.Loop() {
				var r dataEntryRecord
				if err = r.unmarshalBinary(recordBytes); err != nil {
					b.Fatal(err)
				}
				if _, err = proto.NewDataEntryFromValueBytes(r.value); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
