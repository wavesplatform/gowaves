package blockchaininfo

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const epochKeyPrefix = "epoch_"
const blockMeta0xKeyPrefix = "block_0x"

// Helper function to read uint64 from bytes.
func readInt64(data *bytes.Reader) (int64, error) {
	var num int64
	err := binary.Read(data, binary.BigEndian, &num)
	if err != nil {
		return 0, err
	}
	return num, nil
}

// Decode base64 and extract blockHeight and height.
func extractEpochFromBlockMeta(blockMetaValue []byte) (int64, error) {
	var blockMeta BlockMeta
	err := blockMeta.UnmarshalBinary(blockMetaValue)
	if err != nil {
		return 0, errors.Errorf("failed to unmarshal blockMeta, %v", err)
	}

	return blockMeta.BlockEpoch, nil
}

func filterEpochEntry(entry proto.DataEntry, beforeHeight uint64) ([]proto.DataEntry, error) {
	key := entry.GetKey()

	if !strings.HasPrefix(key, epochKeyPrefix) {
		return nil, errors.Errorf("failed to filter epoch entry, the key %s doesn't have prefix %s",
			key, epochKeyPrefix)
	}
	// Extract the part after "epoch_"
	epochStr := key[len(epochKeyPrefix):]

	epochNumber, err := strconv.ParseUint(epochStr, 10, 64)
	if err != nil {
		return nil, err
	}

	// Return this entry only if epochNumber is greater than beforeHeight
	if epochNumber > beforeHeight {
		return []proto.DataEntry{entry}, nil
	}
	return nil, nil
}

func filterBlock0xEntry(entry proto.DataEntry, beforeHeight uint64) ([]proto.DataEntry, error) {
	// Extract blockHeight and height from base64.
	binaryEntry, ok := entry.(*proto.BinaryDataEntry)
	if !ok {
		return nil, errors.New("failed to convert block meta key to binary data entry")
	}
	epoch, err := extractEpochFromBlockMeta(binaryEntry.Value)
	if err != nil {
		return nil, errors.Errorf("failed to filter data entries, %v", err)
	}

	if epoch < 0 {
		return nil, errors.New("epoch is less than 0")
	}
	// Return this entry only if epochNumber is greater than beforeHeight
	if uint64(epoch) > beforeHeight {
		return []proto.DataEntry{entry}, nil
	}
	return nil, nil
}

func filterDataEntries(beforeHeight uint64, dataEntries []proto.DataEntry) ([]proto.DataEntry, error) {
	var filteredDataEntries []proto.DataEntry

	for _, entry := range dataEntries {
		key := entry.GetKey()

		switch {
		// Filter "epoch_" prefixed keys.
		case strings.HasPrefix(key, epochKeyPrefix):
			entryOrNil, err := filterEpochEntry(entry, beforeHeight)
			if err != nil {
				return nil, err
			}
			filteredDataEntries = append(filteredDataEntries, entryOrNil...)

		// Filter block_0x binary entries.
		case strings.HasPrefix(key, blockMeta0xKeyPrefix):
			entryOrNil, err := filterBlock0xEntry(entry, beforeHeight)
			if err != nil {
				return nil, err
			}
			filteredDataEntries = append(filteredDataEntries, entryOrNil...)

		// Default case to handle non-epoch and non-base64 entries.
		default:
			filteredDataEntries = append(filteredDataEntries, entry)
		}
	}

	return filteredDataEntries, nil
}
