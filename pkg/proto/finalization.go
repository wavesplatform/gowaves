package proto

import (
	"encoding/binary"
	"log/slog"

	"github.com/ccoveille/go-safecast/v2"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

type EndorseBlock struct {
	EndorserIndex        int32         `json:"endorserIndex"`
	FinalizedBlockID     BlockID       `json:"finalizedBlockID"`
	FinalizedBlockHeight uint32        `json:"finalizedBlockHeight"`
	EndorsedBlockID      BlockID       `json:"endorsedBlockId"`
	Signature            bls.Signature `json:"signature"`
}

func (e *EndorseBlock) Marshal() ([]byte, error) {
	endBlockProto := e.ToProtobuf()
	return endBlockProto.MarshalVTStrict()
}

func (e *EndorseBlock) EndorsementMessage() ([]byte, error) {
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
		Signature:            e.Signature.Bytes(),
	}
	return &endBlockProto
}

type FinalizationVoting struct {
	EndorserIndexes                []int32        `json:"endorserIndexes"`
	FinalizedBlockHeight           Height         `json:"finalizedBlockHeight"`
	AggregatedEndorsementSignature bls.Signature  `json:"aggregatedEndorsementSignature"`
	ConflictEndorsements           []EndorseBlock `json:"conflictEndorsements"`
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
	conflictEndorsements := make([]*g.EndorseBlock, len(f.ConflictEndorsements))
	for i, ce := range f.ConflictEndorsements {
		conflictEndorsements[i] = ce.ToProtobuf()
	}

	finalizedBlockHeight, err := safecast.Convert[int32](f.FinalizedBlockHeight)
	if err != nil {
		return nil, errors.Errorf("finalized block height conversion error: %v", err)
	}
	finalizationVoting := g.FinalizationVoting{
		EndorserIndexes:                f.EndorserIndexes,
		FinalizedBlockHeight:           finalizedBlockHeight,
		AggregatedEndorsementSignature: f.AggregatedEndorsementSignature.Bytes(),
		ConflictEndorsements:           conflictEndorsements,
	}
	return &finalizationVoting, nil
}

func CalculateLastFinalizedHeight(currentHeight Height) Height {
	var genesisHeight uint64 = 1
	var maxRollbackDeltaHeight uint64 = 100
	if currentHeight <= maxRollbackDeltaHeight {
		slog.Debug("The last finalized height was calculated", "finalizedHeight", genesisHeight)
		return genesisHeight
	}
	slog.Debug("The last finalized height was calculated", "finalizedHeight",
		currentHeight-maxRollbackDeltaHeight)
	return currentHeight - maxRollbackDeltaHeight
}
