package state

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestCommitmentsRecordRoundTrip(t *testing.T) {
	for i, test := range []int{
		1, 8, 16, 32, 64, 128, 256, 512, 1024,
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			rec := commitmentsRecord{Commitments: generateCommitments(t, test)}

			data, err := rec.marshalBinary()
			require.NoError(t, err)
			assert.NotNil(t, data)

			var decoded commitmentsRecord
			err = decoded.unmarshalBinary(data)
			require.NoError(t, err)
			assert.Equal(t, rec, decoded)
			for i, cm := range rec.Commitments {
				assert.Equal(t, cm.GeneratorPK, decoded.Commitments[i].GeneratorPK)
				assert.Equal(t, cm.EndorserPK, decoded.Commitments[i].EndorserPK)
			}
		})
	}
}

func BenchmarkCommitmentsRecordMarshalling(b *testing.B) {
	for _, n := range []int{
		1, 8, 16, 32, 64, 128, 256, 512, 1024,
	} {
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			rec := commitmentsRecord{Commitments: generateCommitments(b, n)}
			b.ResetTimer()
			for b.Loop() {
				_, err := rec.marshalBinary()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCommitmentsRecordUnmarshalling(b *testing.B) {
	for _, n := range []int{
		1, 8, 16, 32, 64, 128, 256, 512, 1024,
	} {
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			rec := commitmentsRecord{Commitments: generateCommitments(b, n)}
			data, err := rec.marshalBinary()
			if err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for b.Loop() {
				var decoded commitmentsRecord
				err = decoded.unmarshalBinary(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestCommitments_Exists(t *testing.T) {
	for i, test := range []struct {
		periodStart uint32
		n           int
	}{
		{periodStart: 1_000_000, n: 1},
		{periodStart: 2_000_000, n: 32},
		{periodStart: 3_000_000, n: 64},
		{periodStart: 4_000_000, n: 128},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			to := createStorageObjects(t, true)
			cms := generateCommitments(t, test.n+1)
			for j := range test.n {
				blockID := generateRandomBlockID(t)
				to.addBlock(t, blockID)
				err := to.entities.commitments.store(test.periodStart, cms[j].GeneratorPK, cms[j].EndorserPK, blockID)
				require.NoError(t, err)

				// Check that all added commitments exist.
				for k := range j {
					ok, eErr := to.entities.commitments.newestExists(test.periodStart, cms[k].GeneratorPK, cms[k].EndorserPK)
					require.NoError(t, eErr)
					assert.True(t, ok)
				}

				// Check that non-existing commitment does not exist.
				ok, err := to.entities.commitments.newestExists(test.periodStart, cms[test.n].GeneratorPK, cms[test.n].EndorserPK)
				require.NoError(t, err)
				assert.False(t, ok)

				to.flush(t)

				// Check that all added commitments exist after flush.
				for k := range j {
					ex, eErr := to.entities.commitments.exists(test.periodStart, cms[k].GeneratorPK, cms[k].EndorserPK)
					require.NoError(t, eErr)
					assert.True(t, ex)
				}

				// Check that non-existing commitment does not exist after flush.
				ok, err = to.entities.commitments.exists(test.periodStart, cms[test.n].GeneratorPK, cms[test.n].EndorserPK)
				require.NoError(t, err)
				assert.False(t, ok)
			}
		})
	}
}

func TestCommitments_Size(t *testing.T) {
	for i, test := range []struct {
		periodStart uint32
		n           int
	}{
		{periodStart: 1_000_000, n: 1},
		{periodStart: 2_000_000, n: 32},
		{periodStart: 3_000_000, n: 64},
		{periodStart: 4_000_000, n: 128},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			to := createStorageObjects(t, true)
			cms := generateCommitments(t, test.n+1)
			for j := range test.n {
				blockID := generateRandomBlockID(t)
				to.addBlock(t, blockID)
				err := to.entities.commitments.store(test.periodStart, cms[j].GeneratorPK, cms[j].EndorserPK, blockID)
				require.NoError(t, err)
				// Unflushed size check.
				gs, err := to.entities.commitments.newestGenerators(test.periodStart)
				require.NoError(t, err)
				assert.Equal(t, j+1, len(gs))
				newestSize, err := to.entities.commitments.newestSize(test.periodStart)
				require.NoError(t, err)
				assert.Equal(t, newestSize, len(gs))
				// Check after flush.
				to.flush(t)
				gs, err = to.entities.commitments.generators(test.periodStart)
				require.NoError(t, err)
				assert.Equal(t, j+1, len(gs))
				regularSize, err := to.entities.commitments.size(test.periodStart)
				require.NoError(t, err)
				assert.Equal(t, regularSize, len(gs))
			}
		})
	}
}

func TestRepeatedUsageOfBLSKey(t *testing.T) {
	to := createStorageObjects(t, true)
	periodStart := uint32(1_000_000)
	cms := generateCommitments(t, 2)
	bID1 := generateRandomBlockID(t)
	to.addBlock(t, bID1)
	err := to.entities.commitments.store(periodStart, cms[0].GeneratorPK, cms[0].EndorserPK, bID1)
	require.NoError(t, err)

	// Check that the commitment exist.
	ok, err := to.entities.commitments.newestExists(periodStart, cms[0].GeneratorPK, cms[0].EndorserPK)
	require.NoError(t, err)
	assert.True(t, ok)

	// Check that a commitment with different generator and same endorser keys leads to the error.
	ok, err = to.entities.commitments.newestExists(periodStart, cms[1].GeneratorPK, cms[0].EndorserPK)
	assert.False(t, ok)
	assert.EqualError(t, err, "endorser public key is already used by another generator")

	// Flush and check again.
	to.flush(t)

	ok, err = to.entities.commitments.exists(periodStart, cms[0].GeneratorPK, cms[0].EndorserPK)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = to.entities.commitments.exists(periodStart, cms[1].GeneratorPK, cms[0].EndorserPK)
	assert.False(t, ok)
	assert.EqualError(t, err, "endorser public key is already used by another generator")
}

func generateCommitments(t testing.TB, n int) []commitmentItem {
	r := make([]commitmentItem, n)
	for i := range n {
		_, wpk, err := crypto.GenerateKeyPair(fmt.Appendf(nil, "WAVES_%d", i))
		require.NoError(t, err)
		bsk, err := bls.GenerateSecretKey(fmt.Appendf(nil, "BLS_%d", i))
		require.NoError(t, err)
		bpk, err := bsk.PublicKey()
		require.NoError(t, err)
		r[i] = commitmentItem{
			GeneratorPK: wpk,
			EndorserPK:  bpk,
		}
	}
	return r
}

func generateRandomBlockID(t testing.TB) proto.BlockID {
	var sig crypto.Signature
	_, err := rand.Read(sig[:])
	require.NoError(t, err)
	return proto.NewBlockIDFromSignature(sig)
}
