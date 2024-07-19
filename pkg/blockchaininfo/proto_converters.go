package blockchaininfo

import (
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/l2/blockchain_info"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func BUpdatesInfoToProto(blockInfo BUpdatesInfo, scheme proto.Scheme) (*g.BlockInfo, error) {
	blockHeader, err := blockInfo.BlockHeader.HeaderToProtobufHeader(scheme)
	if err != nil {
		return nil, err
	}
	return &g.BlockInfo{
		Height:      blockInfo.Height,
		VRF:         blockInfo.VRF,
		BlockID:     blockInfo.BlockID.Bytes(),
		BlockHeader: blockHeader,
	}, nil
}

func BUpdatesInfoFromProto(blockInfoProto *g.BlockInfo) (BUpdatesInfo, error) {
	blockID, err := proto.NewBlockIDFromBytes(blockInfoProto.BlockID)
	if err != nil {
		return BUpdatesInfo{}, err
	}
	blockHeader, err := proto.ProtobufHeaderToBlockHeader(blockInfoProto.BlockHeader)
	if err != nil {
		return BUpdatesInfo{}, err
	}
	return BUpdatesInfo{
		Height:         blockInfoProto.Height,
		VRF:            blockInfoProto.VRF,
		BlockID:        blockID,
		BlockHeader:    blockHeader,
		AllDataEntries: nil,
	}, nil
}

func L2ContractDataEntriesToProto(dataEntries []proto.DataEntry) *g.L2ContractDataEntries {
	var protobufDataEntries []*waves.DataEntry
	for _, entry := range dataEntries {
		entryProto := entry.ToProtobuf()
		protobufDataEntries = append(protobufDataEntries, entryProto)
	}
	return &g.L2ContractDataEntries{
		DataEntries: protobufDataEntries,
	}
}

func L2ContractDataEntriesFromProto(protoDataEntries *g.L2ContractDataEntries, scheme proto.Scheme) (L2ContractDataEntries, error) {
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
