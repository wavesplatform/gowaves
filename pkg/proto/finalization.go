package proto

import (
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/ccoveille/go-safecast/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

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

func (e *BlockEndorsement) EndorsementMessage() ([]byte, error) {
	const heightSize = uint32Size

	finalizedID := e.FinalizedBlockID.Bytes()
	endorsedID := e.EndorsedBlockID.Bytes()

	size := len(finalizedID) + heightSize + len(endorsedID)
	buf := make([]byte, size)

	// finalizedBlockId
	copy(buf[0:len(finalizedID)], finalizedID)

	// finalizedBlockHeight (4 bytes big-endian, same as Scala Ints.toByteArray)
	binary.BigEndian.PutUint32(buf[len(finalizedID):len(finalizedID)+heightSize], e.FinalizedBlockHeight)

	// endorsedBlockId
	copy(buf[len(finalizedID)+heightSize:], endorsedID)

	return buf, nil
}

func EndorsementMessage(finalizedBlockID BlockID, endorsedBlockID BlockID,
	finalizedBlockHeight Height) ([]byte, error) {
	const heightSize = uint32Size

	finalizedID := finalizedBlockID.Bytes()
	endorsedID := endorsedBlockID.Bytes()

	size := len(finalizedID) + heightSize + len(endorsedID)
	buf := make([]byte, size)

	// finalizedBlockId
	copy(buf[0:len(finalizedID)], finalizedID)

	finalizedBlockHeightUint, err := safecast.Convert[uint32](finalizedBlockHeight)
	if err != nil {
		return nil, errors.Errorf("finalized block height conversion error: %v", err)
	}
	// finalizedBlockHeight (4 bytes big-endian, same as Scala Ints.toByteArray)
	binary.BigEndian.PutUint32(buf[len(finalizedID):len(finalizedID)+heightSize], finalizedBlockHeightUint)

	// endorsedBlockId
	copy(buf[len(finalizedID)+heightSize:], endorsedID)

	return buf, nil
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

func CalculateLastFinalizedHeight(currentHeight Height) Height {
	var genesisHeight uint64 = 1
	var maxRollbackDeltaHeight uint64 = 100
	if currentHeight <= maxRollbackDeltaHeight {
		return genesisHeight
	}
	return currentHeight - maxRollbackDeltaHeight
}
