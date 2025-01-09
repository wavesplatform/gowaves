package blockchaininfo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const EpochKeyPrefix = "epoch_"
const blockMeta0xKeyPrefix = "block_0x"

// Helper function to read uint64 from bytes.
func readInt64(data *bytes.Reader) int64 {
	var num int64
	err := binary.Read(data, binary.BigEndian, &num)
	if err != nil {
		panic(fmt.Sprintf("Failed to read uint64: %v", err))
	}
	return num
}

// Decode base64 and extract blockHeight and height.
func extractEpochFromBlockMeta(metaBlockValueBytes []byte) int64 {
	// Create a bytes reader for easier parsing.
	reader := bytes.NewReader(metaBlockValueBytes)

	// Extract blockHeight and epoch.
	readInt64(reader)
	epoch := readInt64(reader)

	return epoch
}

func filterDataEntries(beforeHeight uint64, dataEntries []proto.DataEntry) ([]proto.DataEntry, error) {
	var filteredDataEntries []proto.DataEntry

	for _, entry := range dataEntries {
		key := entry.GetKey()

		switch {
		// Filter "epoch_" prefixed keys.
		case strings.HasPrefix(key, EpochKeyPrefix):
			// Extract the numeric part after "epoch_"
			epochStr := key[len(EpochKeyPrefix):]

			// Convert the epoch number to uint64.
			epochNumber, err := strconv.ParseUint(epochStr, 10, 64)
			if err != nil {
				return nil, err
			}

			// Compare epoch number with beforeHeight.
			if epochNumber > beforeHeight {
				// Add to filtered list if epochNumber is greater.
				filteredDataEntries = append(filteredDataEntries, entry)
			}

		// Filter block_0x binary entries.
		case strings.HasPrefix(key, blockMeta0xKeyPrefix):
			// Extract blockHeight and height from base64.
			binaryEntry, ok := entry.(*proto.BinaryDataEntry)
			if !ok {
				return nil, errors.New("failed to convert block meta key to binary data entry")
			}
			epoch := extractEpochFromBlockMeta(binaryEntry.Value)

			if epoch < 0 {
				return nil, errors.New("epoch is less than 0")
			}
			// Compare height with beforeHeight.
			if uint64(epoch) > beforeHeight {
				// Add to filtered list if height is less than beforeHeight.
				filteredDataEntries = append(filteredDataEntries, entry)
			}

		// Default case to handle non-epoch and non-base64 entries.
		default:
			filteredDataEntries = append(filteredDataEntries, entry)
		}
	}

	return filteredDataEntries, nil
}
