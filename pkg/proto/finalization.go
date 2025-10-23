package proto

import (
	"encoding/binary"

	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

const heightSize = 4

type EndorseBlock struct {
	EndorserIndex        int32   `json:"endorserIndex"`
	FinalizedBlockID     BlockID `json:"finalizedBlockID"`
	FinalizedBlockHeight uint32  `json:"finalizedBlockHeight"`
	EndorsedBlockID      BlockID `json:"endorsedBlockId"`
	Signature            []byte  `json:"signature"`
}

func (e *EndorseBlock) Marshal() ([]byte, error) {
	endBlockProto := e.ToProtobuf()
	return endBlockProto.MarshalVTStrict()
}

func (e *EndorseBlock) EndorsementMessage() ([]byte, error) {
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
		FinalizedBlockId:     e.FinalizedBlockID.Bytes(),
		FinalizedBlockHeight: e.FinalizedBlockHeight,
		EndorsedBlockId:      e.EndorsedBlockID.Bytes(),
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
