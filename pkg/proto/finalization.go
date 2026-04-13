package proto

import (
	"bytes"
	"fmt"
	"io"
	"slices"

	"github.com/ccoveille/go-safecast/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

// EndorsementCryptoMessage is used to calculate and validate signatures of block endorsements.
// Only one-way serialization is implemented. The structure is never intended for deserialization from bytes.
type EndorsementCryptoMessage struct {
	FinalizedBlockID     BlockID
	FinalizedBlockHeight uint32
	EndorsedBlockID      BlockID
}

func NewEndorsementCryptoMessage(
	finalizedBlockID, endorsedBlockID BlockID, finalizedBlockHeight uint32,
) *EndorsementCryptoMessage {
	return &EndorsementCryptoMessage{
		FinalizedBlockID:     finalizedBlockID,
		FinalizedBlockHeight: finalizedBlockHeight,
		EndorsedBlockID:      endorsedBlockID,
	}
}

func (msg *EndorsementCryptoMessage) WriteTo(w io.Writer) (int64, error) {
	n, err := msg.FinalizedBlockID.WriteTo(w)
	if err != nil {
		return n, err
	}
	n1, err := U32(msg.FinalizedBlockHeight).WriteTo(w)
	n += n1
	if err != nil {
		return n, err
	}
	n2, err := msg.EndorsedBlockID.WriteTo(w)
	n += n2
	if err != nil {
		return n, err
	}
	return n, nil
}

func (msg *EndorsementCryptoMessage) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := msg.WriteTo(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BlockEndorsement represents an endorsement of a block by a validator.
type BlockEndorsement struct {
	EndorserIndex        uint32        `json:"endorserIndex"`
	FinalizedBlockID     BlockID       `json:"finalizedBlockID"`
	FinalizedBlockHeight uint32        `json:"finalizedBlockHeight"`
	EndorsedBlockID      BlockID       `json:"endorsedBlockId"`
	Signature            bls.Signature `json:"signature"`
}

func (e *BlockEndorsement) Marshal() ([]byte, error) {
	endBlockProto, err := e.ToProtobuf()
	if err != nil {
		return nil, err
	}
	return endBlockProto.MarshalVTStrict()
}

func (e *BlockEndorsement) CryptoMessage() *EndorsementCryptoMessage {
	return &EndorsementCryptoMessage{
		FinalizedBlockID:     e.FinalizedBlockID,
		FinalizedBlockHeight: e.FinalizedBlockHeight,
		EndorsedBlockID:      e.EndorsedBlockID,
	}
}
func (e *BlockEndorsement) UnmarshalFromProtobuf(data []byte) error {
	var pbEndorsement = &g.EndorseBlock{}
	err := pbEndorsement.UnmarshalVT(data)
	if err != nil {
		return err
	}
	var c ProtobufConverter
	res, err := c.EndorseBlock(pbEndorsement)
	if err != nil {
		return err
	}
	*e = res
	return nil
}

func (e *BlockEndorsement) ToProtobuf() (*g.EndorseBlock, error) {
	idx, err := safecast.Convert[int32](e.EndorserIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to convert block endorsement: %w", err)
	}
	eb := &g.EndorseBlock{
		EndorserIndex:        idx,
		FinalizedBlockId:     e.FinalizedBlockID.Bytes(),
		FinalizedBlockHeight: e.FinalizedBlockHeight,
		EndorsedBlockId:      e.EndorsedBlockID.Bytes(),
		Signature:            e.Signature.Bytes(),
	}
	return eb, nil
}

type FinalizationVoting struct {
	EndorserIndexes                []uint32           `json:"endorserIndexes"`
	FinalizedBlockHeight           Height             `json:"finalizedBlockHeight"`
	AggregatedEndorsementSignature bls.Signature      `json:"aggregatedEndorsementSignature"`
	ConflictEndorsements           []BlockEndorsement `json:"conflictEndorsements"`
}

// Validate checks that FinalizationVoting doesn't have any duplicate endorsers indexes.
func (f *FinalizationVoting) Validate() error {
	indexes := make(map[uint32]struct{})
	for _, ce := range f.ConflictEndorsements {
		if _, seen := indexes[ce.EndorserIndex]; seen {
			return fmt.Errorf(
				"invalid finalization voting: duplicate conflicting endorsement with endorser index %d",
				ce.EndorserIndex,
			)
		}
		indexes[ce.EndorserIndex] = struct{}{}
	}
	for _, idx := range f.EndorserIndexes {
		if _, seen := indexes[idx]; seen {
			return fmt.Errorf("invalid finalization voting: duplicate endorser index %d", idx)
		}
		indexes[idx] = struct{}{}
	}
	return nil
}

func (f *FinalizationVoting) Marshal() ([]byte, error) {
	endBlockProto, err := f.ToProtobuf()
	if err != nil {
		return nil, err
	}
	return endBlockProto.MarshalVTStrict()
}

func (f *FinalizationVoting) UnmarshalFromProtobuf(data []byte) error {
	var pbFinalization = &g.FinalizationVoting{}
	err := pbFinalization.UnmarshalVT(data)
	if err != nil {
		return err
	}
	var c ProtobufConverter
	res, err := c.FinalizationVoting(pbFinalization)
	if err != nil {
		return err
	}
	*f = res
	return nil
}

func (f *FinalizationVoting) ToProtobuf() (*g.FinalizationVoting, error) {
	indexes := make([]int32, len(f.EndorserIndexes))
	for i, v := range f.EndorserIndexes {
		idx, err := safecast.Convert[int32](v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert finalization voting to protobuf: %w", err)
		}
		indexes[i] = idx
	}
	conflictEndorsements := make([]*g.EndorseBlock, len(f.ConflictEndorsements))
	for i, ce := range f.ConflictEndorsements {
		var err error
		conflictEndorsements[i], err = ce.ToProtobuf()
		if err != nil {
			return nil, fmt.Errorf("failed to convert finalization voting to protobuf: %w", err)
		}
	}
	finalizedBlockHeight, err := safecast.Convert[int32](f.FinalizedBlockHeight)
	if err != nil {
		return nil, errors.Errorf("finalized block height conversion error: %v", err)
	}
	finalizationVoting := g.FinalizationVoting{
		EndorserIndexes:                indexes,
		FinalizedBlockHeight:           finalizedBlockHeight,
		AggregatedEndorsementSignature: f.AggregatedEndorsementSignature.Bytes(),
		ConflictEndorsements:           conflictEndorsements,
	}
	return &finalizationVoting, nil
}

func CombineFinalizationVoting(voting1, voting2 *FinalizationVoting) *FinalizationVoting {
	switch {
	case voting1 == nil && voting2 == nil:
		return nil
	case voting1 == nil:
		return voting2
	case voting2 == nil:
		return voting1
	default:
		res := *voting2
		res.ConflictEndorsements = slices.Concat(voting1.ConflictEndorsements, voting2.ConflictEndorsements)
		return &res
	}
}
