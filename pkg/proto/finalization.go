package proto

import (
	"encoding/binary"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

const heightSize = 4

type EndorseBlock struct {
	EndorserIndex        int32  `json:"endorserIndex"`
	FinalizedBlockID     []byte `json:"finalizedBlockID"`
	FinalizedBlockHeight uint32 `json:"finalizedBlockHeight"`
	EndorsedBlockID      []byte `json:"endorsedBlockId"`
	Signature            []byte `json:"signature"`
}

func (e *EndorseBlock) Marshal() ([]byte, error) {
	endBlockProto := e.ToProtobuf()
	return endBlockProto.MarshalVTStrict()
}

// EndorsementMessage serializes endorsement structure into base58 message.
func (e *EndorseBlock) EndorsementMessage() ([]byte, error) {
	if len(e.FinalizedBlockID) == 0 || len(e.EndorsedBlockID) == 0 {
		return nil, errors.New("invalid endorsement: missing block IDs")
	}
	// finalizedBlockId + 4 bytes height + endorsedBlockId
	size := len(e.FinalizedBlockID) + heightSize + len(e.EndorsedBlockID)
	buf := make([]byte, size)
	// finalizedBlockId
	copy(buf[0:len(e.FinalizedBlockID)], e.FinalizedBlockID)
	// finalizedBlockHeight
	binary.BigEndian.PutUint32(buf[len(e.FinalizedBlockID):len(e.FinalizedBlockID)+4], e.FinalizedBlockHeight)
	// endorsedBlockId
	copy(buf[len(e.FinalizedBlockID)+4:], e.EndorsedBlockID)

	return []byte(base58.Encode(buf)), nil
}

func (e *EndorseBlock) UnmarshalFromProtobuf(data []byte) error {
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

func (e *EndorseBlock) ToProtobuf() *g.EndorseBlock {
	endBlockProto := g.EndorseBlock{
		EndorserIndex:        e.EndorserIndex,
		FinalizedBlockId:     e.FinalizedBlockID,
		FinalizedBlockHeight: e.FinalizedBlockHeight,
		EndorsedBlockId:      e.EndorsedBlockID,
		Signature:            e.Signature,
	}
	return &endBlockProto
}

type FinalizationVoting struct {
	EndorserIndexes                []int32        `json:"endorserIndexes"`
	AggregatedEndorsementSignature []byte         `json:"aggregatedEndorsementSignature"`
	ConflictEndorsements           []EndorseBlock `json:"conflictEndorsements"`
}

func (f *FinalizationVoting) Marshal() ([]byte, error) {
	endBlockProto := f.ToProtobuf()
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

func (f *FinalizationVoting) ToProtobuf() *g.FinalizationVoting {
	conflictEndorsements := make([]*g.EndorseBlock, len(f.ConflictEndorsements))
	for i, ce := range f.ConflictEndorsements {
		conflictEndorsements[i] = ce.ToProtobuf()
	}
	finalizationVoting := g.FinalizationVoting{
		EndorserIndexes:                f.EndorserIndexes,
		AggregatedEndorsementSignature: f.AggregatedEndorsementSignature,
		ConflictEndorsements:           conflictEndorsements,
	}
	return &finalizationVoting
}
