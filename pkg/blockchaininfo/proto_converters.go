package blockchaininfo

import (
	"errors"
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

func L2ContractDataEntriesFromProto(protoDataEntries *g.L2ContractDataEntries) (L2ContractDataEntries, error) {
	var dataEntries []proto.DataEntry
	for _, protoEntry := range protoDataEntries.DataEntries {
		dataEntryType := proto.DataEntryType{Type: protoEntry.Key}
		entry, err := proto.GuessDataEntryType(dataEntryType)
		if err != nil {
			return L2ContractDataEntries{}, err
		}
		entry.SetKey(protoEntry.Key)
		switch e := entry.(type) {
		case *proto.IntegerDataEntry:
			e.Value = protoEntry.GetIntValue()
			entry = e
		case *proto.BooleanDataEntry:
			e.Value = protoEntry.GetBoolValue()
			entry = e
		case *proto.BinaryDataEntry:
			e.Value = protoEntry.GetBinaryValue()
			entry = e
		case *proto.StringDataEntry:
			e.Value = protoEntry.GetStringValue()
			entry = e
		case *proto.DeleteDataEntry:
		default:
			return L2ContractDataEntries{}, errors.New("failed to convert proto data entries into data entries")
		}
		dataEntries = append(dataEntries, entry)
	}

	return L2ContractDataEntries{AllDataEntries: dataEntries}, nil
}
