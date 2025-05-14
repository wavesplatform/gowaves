package blockchaininfo

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/l2/blockchain_info"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func BlockUpdatesInfoToProto(blockInfo proto.BlockUpdatesInfo, scheme proto.Scheme) (*g.BlockInfo, error) {
	var (
		blockHeader *waves.Block_Header
		err         error
	)

	blockHeader, err = blockInfo.BlockHeader.HeaderToProtobufHeader(scheme)
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

func BlockUpdatesInfoFromProto(blockInfoProto *g.BlockInfo) (proto.BlockUpdatesInfo, error) {
	if blockInfoProto == nil {
		return proto.BlockUpdatesInfo{}, errors.New("empty block info")
	}
	blockID, err := proto.NewBlockIDFromBytes(blockInfoProto.BlockID)
	if err != nil {
		return proto.BlockUpdatesInfo{}, errors.Wrap(err, "failed to convert block ID")
	}
	var c proto.ProtobufConverter
	blockHeader, err := c.PartialBlockHeader(blockInfoProto.BlockHeader)
	if err != nil {
		return proto.BlockUpdatesInfo{}, errors.Wrap(err, "failed to convert block header")
	}
	blockHeader.ID = blockID // Set block ID to the one from the protobuf.
	vrf := proto.B58Bytes(blockInfoProto.VRF)
	return proto.BlockUpdatesInfo{
		Height:      blockInfoProto.Height,
		VRF:         vrf,
		BlockID:     blockID,
		BlockHeader: blockHeader,
	}, nil
}

func L2ContractDataEntriesToProto(contractData proto.L2ContractDataEntries) *g.L2ContractDataEntries {
	var protobufDataEntries []*waves.DataEntry
	for _, entry := range contractData.AllDataEntries {
		entryProto := entry.ToProtobuf()
		protobufDataEntries = append(protobufDataEntries, entryProto)
	}
	return &g.L2ContractDataEntries{
		DataEntries: protobufDataEntries,
		Height:      contractData.Height,
		BlockID:     contractData.BlockID.Bytes(),
	}
}

func L2ContractDataEntriesFromProto(
	protoDataEntries *g.L2ContractDataEntries,
	scheme proto.Scheme,
) (proto.L2ContractDataEntries, error) {
	if protoDataEntries == nil {
		return proto.L2ContractDataEntries{}, errors.New("empty contract data")
	}
	converter := proto.ProtobufConverter{FallbackChainID: scheme}
	dataEntries := make([]proto.DataEntry, 0, len(protoDataEntries.DataEntries))
	for _, protoEntry := range protoDataEntries.DataEntries {
		entry, err := converter.Entry(protoEntry)
		if err != nil {
			return proto.L2ContractDataEntries{}, errors.Wrap(err, "failed to convert data entry")
		}
		dataEntries = append(dataEntries, entry)
	}

	blockID, err := proto.NewBlockIDFromBytes(protoDataEntries.BlockID)
	if err != nil {
		return proto.L2ContractDataEntries{}, errors.Wrap(err, "failed to convert block ID")
	}

	return proto.L2ContractDataEntries{AllDataEntries: dataEntries, Height: protoDataEntries.Height, BlockID: blockID}, nil
}

func SerializeConstantKeys(constantKeys []string) ([]byte, error) {
	return json.Marshal(constantKeys)
}

func DeserializeConstantKeys(data []byte) ([]string, error) {
	var keys []string
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}
