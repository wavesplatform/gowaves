package blockchaininfo

import (
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/l2/blockchain_info"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func BUpdatesInfoToProto(blockInfo BUpdatesInfo, scheme proto.Scheme) (*g.BlockInfo, error) {
	blockHeader, err := blockInfo.BlockUpdatesInfo.BlockHeader.HeaderToProtobufHeader(scheme)
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
	blockID, err := proto.NewBlockIDFromBytes(blockInfoProto.BlockID)
	if err != nil {
		return BlockUpdatesInfo{}, err
	}
	blockHeader, err := proto.ProtobufHeaderToBlockHeader(blockInfoProto.BlockHeader)
	if err != nil {
		return BlockUpdatesInfo{}, err
	}
	return BlockUpdatesInfo{
		Height:      blockInfoProto.Height,
		VRF:         blockInfoProto.VRF,
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

	return L2ContractDataEntries{AllDataEntries: dataEntries}, nil
}
