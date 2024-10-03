package blockchaininfo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"strconv"
	"strings"
)

var EpochKeyPrefix = "epoch_"
var blockMeta0xKeyPrefix = "block_0x"
var blockMetaKeyPrefix = "blockMeta"

func removeOldEpochs(height uint64) {

}

// Helper function to read uint64 from bytes
func readInt64(data *bytes.Reader) int64 {
	var num int64
	err := binary.Read(data, binary.BigEndian, &num)
	if err != nil {
		panic(fmt.Sprintf("Failed to read uint64: %v", err))
	}
	return num
}

// Decode base64 and extract blockHeight and height
func extractEpochFromBlockMeta(metaBlockValueBytes []byte) (int64, int64, error) {
	//const base64Prefix = "base64:"

	//metaBlockValue := string(metaBlockValueBytes)
	// Strip the "base64:" prefix
	//if !strings.HasPrefix(metaBlockValue, base64Prefix) {
	//	return 0, 0, fmt.Errorf("invalid base64 string")
	//}

	// Decode Base64 string
	//data, err := base64.StdEncoding.DecodeString(metaBlockValue[len(base64Prefix):])
	//if err != nil {
	//	return 0, 0, fmt.Errorf("failed to decode base64 string: %w", err)
	//}

	// Create a bytes reader for easier parsing
	reader := bytes.NewReader(metaBlockValueBytes)

	// Extract blockHeight and height
	blockHeight := readInt64(reader)
	height := readInt64(reader)

	return blockHeight, height, nil
}

func filterDataEntries(beforeHeight uint64, dataEntries []proto.DataEntry) ([]proto.DataEntry, error) {
	var filteredDataEntries []proto.DataEntry

	for _, entry := range dataEntries {
		key := entry.GetKey()

		switch {
		// Filter "epoch_" prefixed keys
		case strings.HasPrefix(key, EpochKeyPrefix):
			// Extract the numeric part after "epoch_"
			epochStr := key[len(EpochKeyPrefix):]

			// Convert the epoch number to uint64
			epochNumber, err := strconv.ParseUint(epochStr, 10, 64)
			if err != nil {
				return nil, err
			}

			// Compare epoch number with beforeHeight
			if epochNumber > beforeHeight {
				// Add to filtered list if epochNumber is greater
				filteredDataEntries = append(filteredDataEntries, entry)
			}

		// Filter block_0x binary entries
		case strings.HasPrefix(key, blockMeta0xKeyPrefix):
			// Extract blockHeight and height from base64
			binaryEntry, ok := entry.(*proto.BinaryDataEntry)
			if !ok {
				return nil, errors.New("failed to convert block meta key to binary data entry")
			}
			_, epoch, err := extractEpochFromBlockMeta(binaryEntry.Value)
			if err != nil {
				return nil, err
			}

			// Compare height with beforeHeight
			if epoch > int64(beforeHeight) {
				// Add to filtered list if height is less than beforeHeight
				filteredDataEntries = append(filteredDataEntries, entry)
			}

			// Filter blockMeta binary entries
		case strings.HasPrefix(key, blockMetaKeyPrefix):
			// Extract blockHeight and height from base64
			binaryEntry, ok := entry.(*proto.BinaryDataEntry)
			if !ok {
				return nil, errors.New("failed to convert block meta key to binary data entry")
			}
			_, epoch, err := extractEpochFromBlockMeta(binaryEntry.Value)
			if err != nil {
				return nil, err
			}

			// Compare height with beforeHeight
			if epoch > int64(beforeHeight) {
				// Add to filtered list if height is less than beforeHeight
				filteredDataEntries = append(filteredDataEntries, entry)
			}

		// Default case to handle non-epoch and non-base64 entries
		default:
			filteredDataEntries = append(filteredDataEntries, entry)
		}
	}

	return filteredDataEntries, nil
}

//block_0xhash, i.e. block_0xee7e9ae625c8be417f239337d82ed5e577458dec8d305a35746777fd17297a03

//func mkBlockMetaEntry(
//	blockHashHex: String, blockHeight: Int, blockParentHex: String, blockGenerator: Address, chainId: Int,
//	elToClTransfersRootHashHex: String, lastClToElTransferIndex: Int
//) = {
//let blockMetaBytes = blockHeight.toBytes() + height.toBytes() + blockParentHex.fromBase16String() + blockGenerator.bytes +
//chainId.toBytes() + elToClTransfersRootHashHex.fromBase16String() + lastClToElTransferIndex.toBytes()
//
//BinaryEntry(blockMetaK + blockHashHex, blockMetaBytes)
//}
