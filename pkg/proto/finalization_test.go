package proto_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestFinalizationVotingCombination(t *testing.T) {
	fv1 := &proto.FinalizationVoting{
		EndorserIndexes:                []uint32{0, 1, 2},
		FinalizedBlockHeight:           123,
		AggregatedEndorsementSignature: nil,
		ConflictEndorsements:           nil,
	}
	fv2 := &proto.FinalizationVoting{
		EndorserIndexes:                []uint32{3, 4, 5},
		FinalizedBlockHeight:           123,
		AggregatedEndorsementSignature: nil,
		ConflictEndorsements:           nil,
	}
	eb1 := proto.BlockEndorsement{
		EndorserIndex:        0,
		FinalizedBlockID:     proto.MustBlockIDFromBase58("4L1nScCRDdRkvVHwrhubtQtn5n7EWh68WFn6oZMt8KHW"),
		FinalizedBlockHeight: 123,
		EndorsedBlockID:      proto.MustBlockIDFromBase58("7rm2AyHHb2iud2hqid2jVD8z4cJ8iAuWQoAQ441VvfVc"),
		Signature:            bls.Signature{},
	}
	eb2 := proto.BlockEndorsement{
		EndorserIndex:        1,
		FinalizedBlockID:     proto.MustBlockIDFromBase58("4L1nScCRDdRkvVHwrhubtQtn5n7EWh68WFn6oZMt8KHW"),
		FinalizedBlockHeight: 123,
		EndorsedBlockID:      proto.MustBlockIDFromBase58("7rm2AyHHb2iud2hqid2jVD8z4cJ8iAuWQoAQ441VvfVc"),
		Signature:            bls.Signature{},
	}
	fv1c := &proto.FinalizationVoting{
		EndorserIndexes:                []uint32{0, 1, 2},
		FinalizedBlockHeight:           123,
		AggregatedEndorsementSignature: nil,
		ConflictEndorsements:           []proto.BlockEndorsement{eb1},
	}
	fv2c := &proto.FinalizationVoting{
		EndorserIndexes:                []uint32{3, 4, 5},
		FinalizedBlockHeight:           123,
		AggregatedEndorsementSignature: nil,
		ConflictEndorsements:           []proto.BlockEndorsement{eb2},
	}
	for i, test := range []struct {
		fv1 *proto.FinalizationVoting
		fv2 *proto.FinalizationVoting
		res *proto.FinalizationVoting
	}{
		{fv1, fv2, fv2},
		{nil, fv2, fv2},
		{fv1, nil, fv1},
		{nil, nil, nil},
		{nil, fv2c, fv2c},
		{fv1c, nil, fv1c},
		{fv1c, fv2c,
			&proto.FinalizationVoting{
				EndorserIndexes:                []uint32{3, 4, 5},
				FinalizedBlockHeight:           123,
				AggregatedEndorsementSignature: nil,
				ConflictEndorsements:           []proto.BlockEndorsement{eb1, eb2},
			},
		},
		{fv2c, fv1c,
			&proto.FinalizationVoting{
				EndorserIndexes:                []uint32{0, 1, 2},
				FinalizedBlockHeight:           123,
				AggregatedEndorsementSignature: nil,
				ConflictEndorsements:           []proto.BlockEndorsement{eb2, eb1},
			},
		},
		{fv1, fv2c, fv2c},
		{fv1c, fv2,
			&proto.FinalizationVoting{
				EndorserIndexes:                []uint32{3, 4, 5},
				FinalizedBlockHeight:           123,
				AggregatedEndorsementSignature: nil,
				ConflictEndorsements:           []proto.BlockEndorsement{eb1},
			},
		},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			r := proto.CombineFinalizationVoting(test.fv1, test.fv2)
			assert.Equal(t, test.res, r)
		})
	}
}

func TestFinalizationVotingValidation(t *testing.T) {
	buildConflicts := func(indexes []uint32) []proto.BlockEndorsement {
		if len(indexes) == 0 {
			return nil
		}
		res := make([]proto.BlockEndorsement, len(indexes))
		for i, idx := range indexes {
			be := proto.BlockEndorsement{
				EndorserIndex:        idx,
				FinalizedBlockID:     proto.MustBlockIDFromBase58("4L1nScCRDdRkvVHwrhubtQtn5n7EWh68WFn6oZMt8KHW"),
				FinalizedBlockHeight: 123,
				EndorsedBlockID:      proto.MustBlockIDFromBase58("7rm2AyHHb2iud2hqid2jVD8z4cJ8iAuWQoAQ441VvfVc"),
				Signature:            bls.Signature{},
			}
			res[i] = be
		}
		return res
	}
	for i, test := range []struct {
		indexes   []uint32
		conflicts []uint32
		fail      bool
		err       string
	}{
		{indexes: nil, conflicts: nil, fail: true,
			err: "invalid finalization voting: both endorsers and conflict endorsements are empty"},
		{indexes: nil, conflicts: []uint32{2, 0, 1}, fail: false},
		{indexes: nil, conflicts: []uint32{2, 1, 2}, fail: true,
			err: "invalid finalization voting: duplicate conflicting endorsement with endorser index 2"},
		{indexes: []uint32{1, 2, 1}, conflicts: nil, fail: true,
			err: "invalid finalization voting: duplicate endorser index 1"},
		{indexes: []uint32{0, 2, 1}, conflicts: []uint32{2}, fail: true,
			err: "invalid finalization voting: duplicate endorser index 2"},
		{indexes: []uint32{0, 1, 2}, conflicts: []uint32{3, 4, 5}, fail: false},
		{indexes: []uint32{0, 1, 2}, conflicts: []uint32{4, 3, 2}, fail: true,
			err: "invalid finalization voting: duplicate endorser index 2",
		},
		{indexes: []uint32{0, 1, 2}, conflicts: []uint32{0, 3, 4}, fail: true,
			err: "invalid finalization voting: duplicate endorser index 0",
		},
		{indexes: []uint32{0, 1, 2}, conflicts: []uint32{1, 3, 4}, fail: true,
			err: "invalid finalization voting: duplicate endorser index 1",
		},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			fv := proto.FinalizationVoting{
				EndorserIndexes:                test.indexes,
				FinalizedBlockHeight:           123,
				AggregatedEndorsementSignature: nil,
				ConflictEndorsements:           buildConflicts(test.conflicts),
			}
			err := fv.Validate()
			if test.fail {
				assert.EqualError(t, err, test.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFinalizationVotingFinalizedHeightValidation(t *testing.T) {
	conflict := proto.BlockEndorsement{
		EndorserIndex:        0,
		FinalizedBlockID:     proto.MustBlockIDFromBase58("4L1nScCRDdRkvVHwrhubtQtn5n7EWh68WFn6oZMt8KHW"),
		FinalizedBlockHeight: 123,
		EndorsedBlockID:      proto.MustBlockIDFromBase58("7rm2AyHHb2iud2hqid2jVD8z4cJ8iAuWQoAQ441VvfVc"),
		Signature:            bls.Signature{},
	}
	for i, test := range []struct {
		indexes   []uint32
		conflicts []proto.BlockEndorsement
		height    proto.Height
		err       string
	}{
		{indexes: nil, conflicts: []proto.BlockEndorsement{conflict}, height: 0}, // Conflicts only, no height.
		{indexes: []uint32{1}, conflicts: nil, height: 123},
		{indexes: []uint32{1}, conflicts: nil, height: 0,
			err: "invalid finalization voting: finalized block height 0 is less than genesis block height 1"},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			fv := proto.FinalizationVoting{
				EndorserIndexes:                test.indexes,
				FinalizedBlockHeight:           test.height,
				AggregatedEndorsementSignature: nil,
				ConflictEndorsements:           test.conflicts,
			}
			err := fv.Validate()
			if test.err != "" {
				assert.EqualError(t, err, test.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFinalizationVotingProtobufRoundTrip(t *testing.T) {
	conflict := proto.BlockEndorsement{
		EndorserIndex:        0,
		FinalizedBlockID:     proto.MustBlockIDFromBase58("4L1nScCRDdRkvVHwrhubtQtn5n7EWh68WFn6oZMt8KHW"),
		FinalizedBlockHeight: 123,
		EndorsedBlockID:      proto.MustBlockIDFromBase58("7rm2AyHHb2iud2hqid2jVD8z4cJ8iAuWQoAQ441VvfVc"),
		Signature:            bls.Signature{},
	}
	sig := bls.Signature{}
	for i, fv := range []proto.FinalizationVoting{
		{
			EndorserIndexes:                []uint32{0, 1, 2},
			FinalizedBlockHeight:           123,
			AggregatedEndorsementSignature: &sig,
			ConflictEndorsements:           nil,
		},
		{ // Conflicting endorsements only, no aggregated signature.
			EndorserIndexes:                nil,
			FinalizedBlockHeight:           0,
			AggregatedEndorsementSignature: nil,
			ConflictEndorsements:           []proto.BlockEndorsement{conflict},
		},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			b, err := fv.Marshal()
			require.NoError(t, err)
			var r proto.FinalizationVoting
			require.NoError(t, r.UnmarshalFromProtobuf(b))
			assert.Equal(t, fv.AggregatedEndorsementSignature, r.AggregatedEndorsementSignature)
			assert.Equal(t, fv.FinalizedBlockHeight, r.FinalizedBlockHeight)
			rb, err := r.Marshal()
			require.NoError(t, err)
			assert.Equal(t, b, rb) // Serialization must be stable, block ID and signature depend on it.
		})
	}
}
