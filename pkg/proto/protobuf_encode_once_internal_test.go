package proto

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

type pbTxFixture struct {
	name string
	tx   Transaction
	pk   crypto.PublicKey
}

// pbTxFixtures returns a signed transaction of every Waves type that has a protobuf version, at that version.
func pbTxFixtures(tb testing.TB) []pbTxFixture {
	kp := benchKeyPair(tb, "protobuf sender")
	asset := benchDigest(7)
	script := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	sign := func(tx Transaction) Transaction {
		benchSign(tb, tx, kp)
		return tx
	}
	txs := []Transaction{
		benchTransfer(tb, TransferTransactionProtobufVersion, kp),
		benchIssue(tb, IssueTransactionProtobufVersion, kp),
		sign(NewUnsignedIssueWithProofs(IssueTransactionProtobufVersion, kp.Public, "SCRIPTED", "with script",
			1000, 2, false, script, benchTimestamp, 100_000_000)),
		sign(NewUnsignedReissueWithProofs(ReissueTransactionProtobufVersion, kp.Public, asset, 1000, true,
			benchTimestamp, 100_000)),
		sign(NewUnsignedBurnWithProofs(BurnTransactionProtobufVersion, kp.Public, asset, 1000, benchTimestamp,
			100_000)),
		benchExchange(tb, ExchangeTransactionProtobufVersion, kp),
		sign(NewUnsignedLeaseWithProofs(LeaseTransactionProtobufVersion, kp.Public, benchRecipient(tb, "lessee"),
			1000, 100_000, benchTimestamp)),
		sign(NewUnsignedLeaseCancelWithProofs(LeaseCancelTransactionProtobufVersion, kp.Public, asset, 100_000,
			benchTimestamp)),
		sign(NewUnsignedCreateAliasWithProofs(CreateAliasTransactionProtobufVersion, kp.Public,
			*NewAlias(benchScheme, "protobuf-alias"), 100_000, benchTimestamp)),
		benchMassTransfer(tb, MassTransferTransactionProtobufVersion, kp),
		benchData(tb, DataTransactionProtobufVersion, kp),
		benchSetScript(tb, SetScriptTransactionProtobufVersion, kp),
		sign(NewUnsignedSponsorshipWithProofs(SponsorshipTransactionProtobufVersion, kp.Public, asset, 1000,
			100_000, benchTimestamp)),
		sign(NewUnsignedSetAssetScriptWithProofs(SetAssetScriptTransactionProtobufVersion, kp.Public, asset,
			script, 100_000, benchTimestamp)),
		benchInvoke(tb, InvokeScriptTransactionProtobufVersion, kp),
		sign(NewUnsignedUpdateAssetInfoWithProofs(UpdateAssetInfoTransactionProtobufVersion, asset, kp.Public,
			"name", "description", benchTimestamp, NewOptionalAssetWaves(), 100_000)),
		sign(NewUnsignedInvokeExpressionWithProofs(InvokeExpressionTransactionProtobufVersion, kp.Public,
			B64Bytes(script), NewOptionalAssetWaves(), 100_000, benchTimestamp)),
		sign(NewUnsignedCommitToGenerationWithProofs(CommitToGenerationTransactionProtobufVersion, kp.Public, 1000,
			bls.PublicKey{}, bls.Signature{}, 100_000, benchTimestamp)),
	}
	res := make([]pbTxFixture, len(txs))
	for i, tx := range txs {
		require.True(tb, IsProtobufTx(tx), "fixture %T v%d is not a protobuf transaction", tx, tx.GetVersion())
		res[i] = pbTxFixture{
			name: reflect.TypeOf(tx).Elem().Name(),
			tx:   tx,
			pk:   kp.Public,
		}
	}
	return res
}

func txProofsField(tb testing.TB, tx Transaction) **ProofsV1 {
	f := reflect.ValueOf(tx).Elem().FieldByName("Proofs")
	require.True(tb, f.IsValid(), "transaction %T has no Proofs field", tx)
	p, ok := reflect.TypeAssert[**ProofsV1](f.Addr())
	require.True(tb, ok, "unexpected Proofs field type in %T", tx)
	return p
}

// oldProtobufEncodings is the oracle: the copying encoding path used before the encode-once helpers.
func oldProtobufEncodings(t *testing.T, tx Transaction) ([]byte, []byte) {
	unsigned, err := tx.ToProtobuf(benchScheme)
	require.NoError(t, err)
	body, err := unsigned.MarshalVTStrict()
	require.NoError(t, err)
	signedPB, err := tx.ToProtobufSigned(benchScheme)
	require.NoError(t, err)
	signed, err := signedPB.MarshalVTStrict()
	require.NoError(t, err)
	return signed, body
}

func TestProtobufEncodeOnce(t *testing.T) {
	fixtures := pbTxFixtures(t)
	types := make(map[TransactionType]struct{})
	for _, fx := range fixtures {
		types[fx.tx.GetType()] = struct{}{}
	}
	require.Len(t, types, len(ProtobufTransactionsVersions)-3) // Genesis, Payment and Ethereum excluded.
	for _, fx := range fixtures {
		t.Run(fx.name, func(t *testing.T) {
			oldSigned, oldBody := oldProtobufEncodings(t, fx.tx)
			pbBody, err := fx.tx.MarshalToProtobuf(benchScheme)
			require.NoError(t, err)
			assert.Equal(t, oldBody, pbBody)
			txBody, err := MarshalTxBody(benchScheme, fx.tx)
			require.NoError(t, err)
			assert.Equal(t, oldBody, txBody)

			pbSigned, err := fx.tx.MarshalSignedToProtobuf(benchScheme)
			require.NoError(t, err)
			assert.Equal(t, oldSigned, pbSigned)
			merkle, err := fx.tx.MerkleBytes(benchScheme)
			require.NoError(t, err)
			assert.Equal(t, oldSigned, merkle)
			stored, err := MarshalTx(benchScheme, fx.tx)
			require.NoError(t, err)
			assert.Equal(t, oldSigned, stored)

			// The round trip through the signed protobuf bytes gives the same transaction bytes back.
			back, err := SignedTxFromProtobuf(pbSigned)
			require.NoError(t, err)
			backSigned, err := back.MarshalSignedToProtobuf(benchScheme)
			require.NoError(t, err)
			assert.Equal(t, oldSigned, backSigned)
		})
	}
}

func TestProtobufVerifyGeneratesID(t *testing.T) {
	other := benchKeyPair(t, "someone else")
	for _, fx := range pbTxFixtures(t) {
		t.Run(fx.name, func(t *testing.T) {
			_, oldBody := oldProtobufEncodings(t, fx.tx)
			wantID := crypto.MustFastHash(oldBody)
			v, ok := fx.tx.(benchVerifier)
			require.True(t, ok)
			idField := benchIDField(t, fx.tx)

			// The ID generated by GenerateID is unchanged.
			*idField = nil
			require.NoError(t, fx.tx.GenerateID(benchScheme))
			genID := **idField
			if _, isAlias := fx.tx.(*CreateAliasWithProofs); !isAlias {
				assert.Equal(t, wantID, genID)
			}

			// Verify sets the same ID when it is missing; CreateAlias keeps its own ID scheme.
			*idField = nil
			valid, err := v.Verify(benchScheme, fx.pk)
			require.NoError(t, err)
			assert.True(t, valid)
			if _, isAlias := fx.tx.(*CreateAliasWithProofs); isAlias {
				assert.Nil(t, *idField)
			} else {
				require.NotNil(t, *idField)
				assert.Equal(t, genID, **idField)
			}

			// Verify does not replace an existing ID.
			preset := benchDigest(99)
			*idField = &preset
			valid, err = v.Verify(benchScheme, fx.pk)
			require.NoError(t, err)
			assert.True(t, valid)
			assert.Equal(t, preset, **idField)

			// The result of Verify with a wrong key is unchanged.
			valid, err = v.Verify(benchScheme, other.Public)
			require.NoError(t, err)
			assert.False(t, valid)

			*idField = &genID
		})
	}
}

func TestProtobufSignedProofsOwnership(t *testing.T) {
	for _, fx := range pbTxFixtures(t) {
		t.Run(fx.name, func(t *testing.T) {
			proofs := *txProofsField(t, fx.tx)
			require.NotEmpty(t, proofs.Proofs)
			orig := append([]byte(nil), proofs.Proofs[0]...)

			// The public ToProtobufSigned keeps returning a copy of the proofs.
			pb, err := fx.tx.ToProtobufSigned(benchScheme)
			require.NoError(t, err)
			pb.Proofs[0][0] ^= 0xff
			assert.Equal(t, orig, []byte(proofs.Proofs[0]))

			// The internal accessor aliases them.
			noCopy, err := signedWavesTxToProtobufNoCopy(fx.tx, benchScheme, proofs)
			require.NoError(t, err)
			require.Len(t, noCopy.Proofs, len(proofs.Proofs))
			for i := range noCopy.Proofs {
				assert.Same(t, &proofs.Proofs[i][0], &noCopy.Proofs[i][0])
			}
		})
	}
}

func TestProtobufSignedNoProofs(t *testing.T) {
	for _, fx := range pbTxFixtures(t) {
		t.Run(fx.name, func(t *testing.T) {
			pp := txProofsField(t, fx.tx)
			saved := *pp
			*pp = nil
			defer func() { *pp = saved }()
			_, err := fx.tx.MarshalSignedToProtobuf(benchScheme)
			require.EqualError(t, err, "no proofs provided")
			_, err = fx.tx.ToProtobufSigned(benchScheme)
			require.EqualError(t, err, "no proofs provided")
		})
	}
}

func TestProofsV1BytesNoCopy(t *testing.T) {
	p := &ProofsV1{Version: proofsVersion, Proofs: []B58Bytes{{1, 2}, {3}, {}}}
	b := p.bytesNoCopy()
	assert.Equal(t, p.Bytes(), b)
	assert.Empty(t, (&ProofsV1{}).bytesNoCopy())
}

func TestMarshalToProtobufDeterministic(t *testing.T) {
	for _, fx := range pbTxFixtures(t) {
		pb, err := fx.tx.ToProtobufSigned(benchScheme)
		require.NoError(t, err)
		want, err := pb.MarshalVTStrict()
		require.NoError(t, err)
		got, err := MarshalToProtobufDeterministic(pb)
		require.NoError(t, err)
		assert.Equal(t, want, got, fx.name)
	}
	got, err := MarshalToProtobufDeterministic(&g.SignedTransaction{})
	require.NoError(t, err)
	assert.Empty(t, got)
	big := &g.SignedTransaction{Proofs: [][]byte{make([]byte, 10*1024*1024+1)}}
	_, err = MarshalToProtobufDeterministic(big)
	assert.ErrorContains(t, err, "exceeds 10MB limit")
}
