package blockchaininfo

import (
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/l2/blockchain_info"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func BUpdatesInfoToProto(blockInfo BUpdatesInfo, scheme proto.Scheme) (*g.BlockInfo, error) {
	var (
		blockHeader *waves.Block_Header
		err         error
	)

	blockHeader, err = blockInfo.BlockUpdatesInfo.BlockHeader.HeaderToProtobufHeader(scheme)
	if err != nil {
		return nil, err
	}
	return &g.BlockInfo{
		Height:      blockInfo.BlockUpdatesInfo.Height,
		VRF:         blockInfo.BlockUpdatesInfo.VRF,
		BlockID:     blockInfo.BlockUpdatesInfo.BlockID.Bytes(),
		BlockHeader: blockHeader,
	}, nil
}

func BUpdatesInfoFromProto(blockInfoProto *g.BlockInfo) (BlockUpdatesInfo, error) {
	if blockInfoProto == nil {
		return BlockUpdatesInfo{}, errors.New("empty block info")
	}
	blockID, err := proto.NewBlockIDFromBytes(blockInfoProto.BlockID)
	if err != nil {
		return BlockUpdatesInfo{}, errors.Wrap(err, "failed to convert block ID")
	}
	var c proto.ProtobufConverter
	blockHeader, err := c.PartialBlockHeader(blockInfoProto.BlockHeader)
	if err != nil {
		return BlockUpdatesInfo{}, errors.Wrap(err, "failed to convert block header")
	}
	blockHeader.ID = blockID // Set block ID to the one from the protobuf.
	vrf := proto.B58Bytes(blockInfoProto.VRF)
	return BlockUpdatesInfo{
		Height:      blockInfoProto.Height,
		VRF:         vrf,
		BlockID:     blockID,
		BlockHeader: blockHeader,
	}, nil
}

func L2ContractDataEntriesToProto(contractData L2ContractDataEntries) *g.L2ContractDataEntries {
	var protobufDataEntries []*waves.DataEntry
	for _, entry := range contractData.AllDataEntries {
		entryProto := entry.ToProtobuf()
		protobufDataEntries = append(protobufDataEntries, entryProto)
	}
	return &g.L2ContractDataEntries{
		DataEntries: protobufDataEntries,
		Height:      contractData.Height,
	}
}

func L2ContractDataEntriesFromProto(protoDataEntries *g.L2ContractDataEntries,
	scheme proto.Scheme) (L2ContractDataEntries, error) {
	converter := proto.ProtobufConverter{FallbackChainID: scheme}
	var dataEntries []proto.DataEntry
	for _, protoEntry := range protoDataEntries.DataEntries {
		entry, err := converter.Entry(protoEntry)
		if err != nil {
			return L2ContractDataEntries{}, err
		}
		dataEntries = append(dataEntries, entry)
	}

	return L2ContractDataEntries{AllDataEntries: dataEntries, Height: protoDataEntries.Height}, nil
}
