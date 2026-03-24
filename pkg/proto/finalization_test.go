package proto_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestFinalizationVotingCombination(t *testing.T) {
	fv1 := &proto.FinalizationVoting{
		EndorserIndexes:                []uint32{0, 1, 2},
		FinalizedBlockHeight:           123,
		AggregatedEndorsementSignature: bls.Signature{},
		ConflictEndorsements:           nil,
	}
	fv2 := &proto.FinalizationVoting{
		EndorserIndexes:                []uint32{3, 4, 5},
		FinalizedBlockHeight:           123,
		AggregatedEndorsementSignature: bls.Signature{},
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
		AggregatedEndorsementSignature: bls.Signature{},
		ConflictEndorsements:           []proto.BlockEndorsement{eb1},
	}
	fv2c := &proto.FinalizationVoting{
		EndorserIndexes:                []uint32{3, 4, 5},
		FinalizedBlockHeight:           123,
		AggregatedEndorsementSignature: bls.Signature{},
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
				AggregatedEndorsementSignature: bls.Signature{},
				ConflictEndorsements:           []proto.BlockEndorsement{eb1, eb2},
			},
		},
		{fv2c, fv1c,
			&proto.FinalizationVoting{
				EndorserIndexes:                []uint32{0, 1, 2},
				FinalizedBlockHeight:           123,
				AggregatedEndorsementSignature: bls.Signature{},
				ConflictEndorsements:           []proto.BlockEndorsement{eb2, eb1},
			},
		},
		{fv1, fv2c, fv2c},
		{fv1c, fv2,
			&proto.FinalizationVoting{
				EndorserIndexes:                []uint32{3, 4, 5},
				FinalizedBlockHeight:           123,
				AggregatedEndorsementSignature: bls.Signature{},
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
