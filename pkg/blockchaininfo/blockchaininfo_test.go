package blockchaininfo_test

import (
	"context"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/blockchaininfo"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// some random test data.
func testBlockUpdates() proto.BlockUpdatesInfo {
	var b proto.BlockUpdatesInfo

	var (
		height      uint64 = 100
		vrf                = proto.B58Bytes{}
		blockID            = proto.BlockID{}
		blockHeader        = proto.BlockHeader{}
	)

	b.Height = height
	b.VRF = vrf
	b.BlockID = blockID
	b.BlockHeader = blockHeader

	return b
}

func containsDataEntry(changes []proto.DataEntry, key string, dataType string) bool {
	for _, entry := range changes {
		// Check if the key matches
		if entry.GetKey() == key {
			// Use a type switch to check the type
			switch entry.(type) {
			case *proto.BinaryDataEntry:
				if dataType == "binary" {
					return true
				}
			case *proto.DeleteDataEntry:
				if dataType == "delete" {
					return true
				}
			default:
			}
		}
	}
	return false
}

// This tests check whether the changes generation will show the new records and will remove the old ones.
// Previous state contains 3 records, but the current state doesn't contain them and has 3 new records.
// The change result must be - 3 new records, and 3 old records for deletion.

func TestChangesGenerationNewEntries(t *testing.T) {
	previousFirstKey := "block_0x3a85dedc42db076c91cf61d72fa17c80777aeed70ba68dbc14d6829dd6e88614"
	previousSecondKey := "block_0x3a9209ce524553a75fd0e9bde5c99ff254b1fb231916fc89755be957e51e5516"
	previousThirdKey := "block_0x3b0181d3f66d9f0ddd8e1e8567b836a01f652b4cb873aa7b7c46fc8bd1e4eeee"

	previousDataEntries := []proto.DataEntry{
		&proto.BinaryDataEntry{Key: previousFirstKey,
			Value: []byte("base64:AAAAAAAQYVwAAAAAADDSCpJJsd11jrOMW7AS/AHIMIDQ" +
				"XjqmFyhDuGt2RPNvmcCXAVTy/URmfMOj7GNweXnZpzidmxHfPBfcP5A=")}, // height 3199498.
		&proto.BinaryDataEntry{Key: previousSecondKey,
			Value: []byte("base64:AAAAAAAQYywAAAAAADDSQBBubtiRmKwtaNFF1TrBhsfBu" +
				"61fj3qiSrtyu1/kLLAlAVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199552.
		&proto.BinaryDataEntry{Key: previousThirdKey,
			Value: []byte("base64:AAAAAAAQZEUAAAAAADDSZeUUyashoWjUKurzA/wVU5prm" +
				"68CambvjIo1ESLoLuAaAVRaS4vOsPl9cxvg7aeRj7RFZQzdpmvV/+A=")}, // height 3199589.
	}
	var previousHeight uint64 = 3199552

	currentFirstKey := "block_0x3b5ad5c176473be02cc3d19207204af87af03f6fd75c76916765745658f7e842"
	currentSecondKey := "block_0x3b72ee917fea7057fb88a357f619c22f6f8ddae03b701fab7c284953ecebbc8c"
	currentThirdKey := "block_0x3b973acae11f248a524b463db7d198c7ddb47fd8aeda2f14699e639a0db19911"

	currentDataEntries := []proto.DataEntry{
		&proto.BinaryDataEntry{Key: currentFirstKey,
			Value: []byte("base64:AAAAAAAQZKkAAAAAADDSb6xEaq4RsFQruG" +
				"NeGdooPmtLBnlERR15qzc/mcKcQ461AVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199599.
		&proto.BinaryDataEntry{Key: currentSecondKey,
			Value: []byte("base64:AAAAAAAQZQkAAAAAADDSe+2CGv9zgiR7s" +
				"65XEBkYzIbv6jbxcR7Zi3ByUqsX0bkwAVTEkyC5glOJH8Upe49iT3+BUV5zRaDT2dM=")}, // height 3199611.
		&proto.BinaryDataEntry{Key: currentThirdKey,
			Value: []byte("base64:AAAAAAAQZf8AAAAAADDSolzqc5gjHWP/s" +
				"CzqK7+HkAjybjGxq8SxL9ID8yEIKxrlAVRN71D/MD4dykS8vqW7cXqCh5QOclg6DEU=")}, // height 3199650.
	}
	var currentHeight uint64 = 3199611

	previousBlockInfo := proto.BUpdatesInfo{
		BlockUpdatesInfo: testBlockUpdates(),
		ContractUpdatesInfo: proto.L2ContractDataEntries{
			AllDataEntries: previousDataEntries,
			Height:         previousHeight,
		},
	}

	currentBlockInfo := proto.BUpdatesInfo{
		BlockUpdatesInfo: testBlockUpdates(),
		ContractUpdatesInfo: proto.L2ContractDataEntries{
			AllDataEntries: currentDataEntries,
			Height:         currentHeight,
		},
	}

	equal, changes, err := blockchaininfo.CompareBUpdatesInfo(currentBlockInfo, previousBlockInfo,
		proto.TestNetScheme)
	if err != nil {
		return
	}
	require.False(t, equal)
	require.True(t, containsDataEntry(changes.ContractUpdatesInfo.AllDataEntries, currentFirstKey, "binary"))
	require.True(t, containsDataEntry(changes.ContractUpdatesInfo.AllDataEntries, currentSecondKey, "binary"))
	require.True(t, containsDataEntry(changes.ContractUpdatesInfo.AllDataEntries, currentThirdKey, "binary"))

	require.True(t, containsDataEntry(changes.ContractUpdatesInfo.AllDataEntries, previousFirstKey, "delete"))
	require.True(t, containsDataEntry(changes.ContractUpdatesInfo.AllDataEntries, previousSecondKey, "delete"))
	require.True(t, containsDataEntry(changes.ContractUpdatesInfo.AllDataEntries, previousThirdKey, "delete"))
}

// This tests check whether the changes generation will only show the new records and will not remove the old ones.
// Previous state contains 3 records, the current state contains both the previous new records and 3 new ones.
// The change result must be - 3 new records.
func TestChangesGenerationContainsPrevious(t *testing.T) {
	previousFirstKey := "block_0x3a85dedc42db076c91cf61d72fa17c80777aeed70ba68dbc14d6829dd6e88614"
	previousSecondKey := "block_0x3a9209ce524553a75fd0e9bde5c99ff254b1fb231916fc89755be957e51e5516"
	previousThirdKey := "block_0x3b0181d3f66d9f0ddd8e1e8567b836a01f652b4cb873aa7b7c46fc8bd1e4eeee"

	previousDataEntries := []proto.DataEntry{
		&proto.BinaryDataEntry{Key: previousFirstKey,
			Value: []byte("base64:AAAAAAAQYVwAAAAAADDSCpJJsd11jrOMW7AS/AHIMIDQXj" +
				"qmFyhDuGt2RPNvmcCXAVTy/URmfMOj7GNweXnZpzidmxHfPBfcP5A=")}, // height 3199498.
		&proto.BinaryDataEntry{Key: previousSecondKey,
			Value: []byte("base64:AAAAAAAQYywAAAAAADDSQBBubtiRmKwtaNFF1TrBhsfBu61" +
				"fj3qiSrtyu1/kLLAlAVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199552.
		&proto.BinaryDataEntry{Key: previousThirdKey,
			Value: []byte("base64:AAAAAAAQZEUAAAAAADDSZeUUyashoWjUKurzA/wVU5prm68Ca" +
				"mbvjIo1ESLoLuAaAVRaS4vOsPl9cxvg7aeRj7RFZQzdpmvV/+A=")}, // height 3199589.
	}
	var previousHeight uint64 = 3199552

	currentFirstKey := "block_0x3b5ad5c176473be02cc3d19207204af87af03f6fd75c76916765745658f7e842"
	currentSecondKey := "block_0x3b72ee917fea7057fb88a357f619c22f6f8ddae03b701fab7c284953ecebbc8c"
	currentThirdKey := "block_0x3b973acae11f248a524b463db7d198c7ddb47fd8aeda2f14699e639a0db19911"

	currentDataEntries := []proto.DataEntry{
		&proto.BinaryDataEntry{Key: previousFirstKey,
			Value: []byte("base64:AAAAAAAQYVwAAAAAADDSCpJJsd11jrOMW7AS/A" +
				"HIMIDQXjqmFyhDuGt2RPNvmcCXAVTy/URmfMOj7GNweXnZpzidmxHfPBfcP5A=")}, // height 3199498.
		&proto.BinaryDataEntry{Key: previousSecondKey,
			Value: []byte("base64:AAAAAAAQYywAAAAAADDSQBBubtiRmKwtaNFF1T" +
				"rBhsfBu61fj3qiSrtyu1/kLLAlAVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199552.
		&proto.BinaryDataEntry{Key: previousThirdKey,
			Value: []byte("base64:AAAAAAAQZEUAAAAAADDSZeUUyashoWjUKurzA/wV" +
				"U5prm68CambvjIo1ESLoLuAaAVRaS4vOsPl9cxvg7aeRj7RFZQzdpmvV/+A=")}, // height 3199589.

		&proto.BinaryDataEntry{Key: currentFirstKey,
			Value: []byte("base64:AAAAAAAQZKkAAAAAADDSb6xEaq4RsFQruGNeGdoo" +
				"PmtLBnlERR15qzc/mcKcQ461AVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199599.
		&proto.BinaryDataEntry{Key: currentSecondKey,
			Value: []byte("base64:AAAAAAAQZQkAAAAAADDSe+2CGv9zgiR7s65XEBkYz" +
				"Ibv6jbxcR7Zi3ByUqsX0bkwAVTEkyC5glOJH8Upe49iT3+BUV5zRaDT2dM=")}, // height 3199611.
		&proto.BinaryDataEntry{Key: currentThirdKey,
			Value: []byte("base64:AAAAAAAQZf8AAAAAADDSolzqc5gjHWP/sCzqK7+Hk" +
				"AjybjGxq8SxL9ID8yEIKxrlAVRN71D/MD4dykS8vqW7cXqCh5QOclg6DEU=")}, // height 3199650.
	}
	var currentHeight uint64 = 3199611

	previousBlockInfo := proto.BUpdatesInfo{
		BlockUpdatesInfo: testBlockUpdates(),
		ContractUpdatesInfo: proto.L2ContractDataEntries{
			AllDataEntries: previousDataEntries,
			Height:         previousHeight,
		},
	}

	currentBlockInfo := proto.BUpdatesInfo{
		BlockUpdatesInfo: testBlockUpdates(),
		ContractUpdatesInfo: proto.L2ContractDataEntries{
			AllDataEntries: currentDataEntries,
			Height:         currentHeight,
		},
	}

	equal, changes, err := blockchaininfo.CompareBUpdatesInfo(currentBlockInfo, previousBlockInfo,
		proto.TestNetScheme)
	if err != nil {
		return
	}
	require.False(t, equal)

	require.True(t, containsDataEntry(changes.ContractUpdatesInfo.AllDataEntries, currentFirstKey, "binary"))
	require.True(t, containsDataEntry(changes.ContractUpdatesInfo.AllDataEntries, currentSecondKey, "binary"))
	require.True(t, containsDataEntry(changes.ContractUpdatesInfo.AllDataEntries, currentThirdKey, "binary"))
}

// This tests check whether the changes generation will not show anything, because there are no changes.
// Previous state contains 3 records, the current state contains the same records.
// The change result must be - 0 records.
func TestNoChangesGeneration(t *testing.T) {
	previousFirstKey := "block_0x3a85dedc42db076c91cf61d72fa17c80777aeed70ba68dbc14d6829dd6e88614"
	previousSecondKey := "block_0x3a9209ce524553a75fd0e9bde5c99ff254b1fb231916fc89755be957e51e5516"
	previousThirdKey := "block_0x3b0181d3f66d9f0ddd8e1e8567b836a01f652b4cb873aa7b7c46fc8bd1e4eeee"

	previousDataEntries := []proto.DataEntry{
		&proto.BinaryDataEntry{Key: previousFirstKey,
			Value: []byte("base64:AAAAAAAQYVwAAAAAADDSCpJJsd11jrO" +
				"MW7AS/AHIMIDQXjqmFyhDuGt2RPNvmcCXAVTy/URmfMOj7GNweXnZpzidmxHfPBfcP5A=")}, // height 3199498.
		&proto.BinaryDataEntry{Key: previousSecondKey,
			Value: []byte("base64:AAAAAAAQYywAAAAAADDSQBBubtiRmKwt" +
				"aNFF1TrBhsfBu61fj3qiSrtyu1/kLLAlAVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199552.
		&proto.BinaryDataEntry{Key: previousThirdKey,
			Value: []byte("base64:AAAAAAAQZEUAAAAAADDSZeUUyashoWjU" +
				"KurzA/wVU5prm68CambvjIo1ESLoLuAaAVRaS4vOsPl9cxvg7aeRj7RFZQzdpmvV/+A=")}, // height 3199589.
	}
	var previousHeight uint64 = 3199552

	currentDataEntries := []proto.DataEntry{
		&proto.BinaryDataEntry{Key: previousFirstKey,
			Value: []byte("base64:AAAAAAAQYVwAAAAAADDSCpJJsd11jrO" +
				"MW7AS/AHIMIDQXjqmFyhDuGt2RPNvmcCXAVTy/URmfMOj7GNweXnZpzidmxHfPBfcP5A=")}, // height 3199498.
		&proto.BinaryDataEntry{Key: previousSecondKey,
			Value: []byte("base64:AAAAAAAQYywAAAAAADDSQBBubtiRmKwta" +
				"NFF1TrBhsfBu61fj3qiSrtyu1/kLLAlAVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199552.
		&proto.BinaryDataEntry{Key: previousThirdKey,
			Value: []byte("base64:AAAAAAAQZEUAAAAAADDSZeUUyashoWjU" +
				"KurzA/wVU5prm68CambvjIo1ESLoLuAaAVRaS4vOsPl9cxvg7aeRj7RFZQzdpmvV/+A=")}, // height 3199589.
	}
	var currentHeight uint64 = 3199611

	previousBlockInfo := proto.BUpdatesInfo{
		BlockUpdatesInfo: testBlockUpdates(),
		ContractUpdatesInfo: proto.L2ContractDataEntries{
			AllDataEntries: previousDataEntries,
			Height:         previousHeight,
		},
	}

	currentBlockInfo := proto.BUpdatesInfo{
		BlockUpdatesInfo: testBlockUpdates(),
		ContractUpdatesInfo: proto.L2ContractDataEntries{
			AllDataEntries: currentDataEntries,
			Height:         currentHeight,
		},
	}

	equal, changes, err := blockchaininfo.CompareBUpdatesInfo(currentBlockInfo, previousBlockInfo,
		proto.TestNetScheme)
	if err != nil {
		return
	}
	require.True(t, equal)

	require.True(t, len(changes.ContractUpdatesInfo.AllDataEntries) == 0)
}

func TestDecodeBlockMeta(t *testing.T) {
	binaryDataEntryJSON := []byte(`{
		"key": "block_0x000cf2d957da5e30dcfae8b5eba2b585f0102680a5c343a1a107aa529f61c2db",
		"type": "binary",
		"value": "base64:AAAAAAACn6IAAAAAADKE1/L3xxY2i+uZZ0Rzd3XOD2O+12+a8D2j0d4Ymk/7v7YdAAAAAAAAAAAAAAAAAAAAFg=="
	}`)
	var binaryEntry proto.BinaryDataEntry
	err := binaryEntry.UnmarshalJSON(binaryDataEntryJSON)
	require.NoError(t, err)
	var blockMeta blockchaininfo.BlockMeta
	err = blockMeta.UnmarshalBinary(binaryEntry.Value)
	require.NoError(t, err)
	require.Equal(t, blockMeta.BlockHeight, int64(171938))
	require.Equal(t, blockMeta.BlockEpoch, int64(3310807))
}

const (
	blockID1 = "7wKAcTGbvDtruMSSYyndzN9YK3cQ47ZdTPeT8ej22qRg"
	BlockID2 = "gzz8aN4b5rr1rkeAdmuwytuGv1jbm9LLRbXNKNb7ETX"
	BlockID3 = "GrgPhEZ5rruNPSac5QxirgoYA2VwEKBJju3ppPgNyBWi"
	BlockID4 = "5g9Ws6Z3SJ9dXN3JqPQxVWeCEYssmYzFdVNXX1rcyHib"
	BlockID5 = "AEB4sYgpA2wMVSdzSCkVuN3R2moPnQiStDs9gPSRStny"
	BlockID6 = "5bEZ4Y9BiVvM53RtBWmpT5jADeLmSt2vmC1iBB2gKuE8"

	l2ContractAddress = "3Mw2AVgk5xNmkWQkzKKhinhBH1YyBTeVku2"

	checkedBlockNumber = 3
)

func fillThirdCheckedBlock(t *testing.T) ([]proto.DataEntry, proto.BlockUpdatesInfo) {
	var integerEntries []proto.DataEntry
	blockID, err := proto.NewBlockIDFromBase58(BlockID3)

	for j := 1; j <= 3; j++ {
		integerDataEntry := &proto.IntegerDataEntry{
			Key:   strconv.Itoa(j),
			Value: int64(-j),
		}
		assert.NoError(t, err)
		integerEntries = append(integerEntries, integerDataEntry)
	}
	blockInfo := proto.BlockUpdatesInfo{
		Height:  uint64(3),
		BlockID: blockID,
	}
	return integerEntries, blockInfo
}

func fillHistoryJournal(t *testing.T, stateCache *blockchaininfo.StateCache) *blockchaininfo.HistoryJournal {
	var historyJorunal blockchaininfo.HistoryJournal
	blockIDs := []string{blockID1, BlockID2, BlockID3, BlockID4, BlockID5}
	for i := 1; i <= 5; i++ {
		if i == checkedBlockNumber {
			integerEntries, blockInfo := fillThirdCheckedBlock(t)
			historyEntry := blockchaininfo.HistoryEntry{
				Height:  blockInfo.Height,
				BlockID: blockInfo.BlockID,
				Entries: integerEntries,
			}
			historyJorunal.Push(historyEntry)
		}

		var integerEntries []proto.DataEntry
		blockID, err := proto.NewBlockIDFromBase58(blockIDs[i-1])

		for j := 1; j <= i; j++ {
			integerDataEntry := &proto.IntegerDataEntry{
				Key:   strconv.Itoa(j),
				Value: int64(j),
			}
			assert.NoError(t, err)
			integerEntries = append(integerEntries, integerDataEntry)
		}
		historyEntry := blockchaininfo.HistoryEntry{
			Height:  uint64(i),
			BlockID: blockID,
			Entries: integerEntries,
		}
		historyJorunal.Push(historyEntry)
		continue
	}
	historyJorunal.SetStateCache(stateCache)
	return &historyJorunal
}

func fillCache(t *testing.T) *blockchaininfo.StateCache {
	stateCache := blockchaininfo.NewStateCache()
	blockIDs := []string{blockID1, BlockID2, BlockID3, BlockID4, BlockID5}
	for i := 1; i <= 5; i++ {
		if i == checkedBlockNumber {
			integerEntries, blockInfo := fillThirdCheckedBlock(t)
			stateCache.AddCacheRecord(blockInfo.Height, integerEntries, blockInfo)
			continue
		}

		var integerEntries []proto.DataEntry
		blockID, err := proto.NewBlockIDFromBase58(blockIDs[i-1])
		require.NoError(t, err)
		for j := 1; j <= i; j++ {
			integerDataEntry := &proto.IntegerDataEntry{
				Key:   strconv.Itoa(j),
				Value: int64(j),
			}
			integerEntries = append(integerEntries, integerDataEntry)
		}
		blockInfo := proto.BlockUpdatesInfo{
			Height:  uint64(i),
			BlockID: blockID,
		}
		stateCache.AddCacheRecord(uint64(i), integerEntries, blockInfo)
	}
	return stateCache
}

// Rollback from block 5 to block 3.
// On block 3, keys "1", "2", "3" had negative values, so the patch should generate the negative
// values only found on that block.
func TestRollback(t *testing.T) {
	mockPublisherInterface := blockchaininfo.NewMockUpdatesPublisherInterface(t)
	mockPublisherInterface.EXPECT().PublishUpdates(mock.Anything, mock.Anything, mock.Anything, proto.TestNetScheme,
		l2ContractAddress).Return(nil)
	mockPublisherInterface.EXPECT().L2ContractAddress().Return(l2ContractAddress)

	blockID6, err := proto.NewBlockIDFromBase58(BlockID6)
	assert.NoError(t, err)
	currentState := proto.BUpdatesInfo{
		BlockUpdatesInfo: proto.BlockUpdatesInfo{
			Height:  6,
			BlockID: blockID6,
		},
		ContractUpdatesInfo: proto.L2ContractDataEntries{
			Height: 6,
			AllDataEntries: []proto.DataEntry{&proto.IntegerDataEntry{
				Key:   "5",
				Value: 6,
			}},
		},
	}
	blockID5, err := proto.NewBlockIDFromBase58(BlockID5)
	assert.NoError(t, err)
	previousState := proto.BUpdatesInfo{
		BlockUpdatesInfo: proto.BlockUpdatesInfo{
			Height:  5,
			BlockID: blockID5,
		},
		ContractUpdatesInfo: proto.L2ContractDataEntries{
			Height: 5,
			AllDataEntries: []proto.DataEntry{&proto.IntegerDataEntry{
				Key:   "5",
				Value: 5,
			}},
		},
	}
	blockID4, err := proto.NewBlockIDFromBase58(BlockID4)
	assert.NoError(t, err)
	updates := proto.BUpdatesInfo{
		BlockUpdatesInfo: proto.BlockUpdatesInfo{
			Height:  4,
			BlockID: blockID4,
		},
		ContractUpdatesInfo: proto.L2ContractDataEntries{
			Height: 4,
			AllDataEntries: []proto.DataEntry{&proto.IntegerDataEntry{
				Key:   "4",
				Value: 4,
			}},
		},
	}
	updatesExtensionState := &blockchaininfo.BUpdatesExtensionState{
		CurrentState:      &currentState,
		PreviousState:     &previousState,
		Limit:             100,
		Scheme:            proto.TestNetScheme,
		L2ContractAddress: l2ContractAddress,
		HistoryJournal:    fillHistoryJournal(t, fillCache(t)),
		St:                nil,
	}
	ctx := context.Background()
	// Rollback from block 5 to 3
	patch := blockchaininfo.HandleRollback(ctx, updatesExtensionState, updates, mockPublisherInterface,
		nil, proto.TestNetScheme)

	expectedPatchEntries := []proto.DataEntry{
		&proto.IntegerDataEntry{
			Key:   "1",
			Value: -1,
		},
		&proto.IntegerDataEntry{
			Key:   "2",
			Value: -2,
		},
		&proto.IntegerDataEntry{
			Key:   "3",
			Value: -3,
		},
		&proto.DeleteDataEntry{Key: "4"},
		&proto.DeleteDataEntry{Key: "5"},
	}
	expectedL2Patch := proto.L2ContractDataEntries{
		AllDataEntries: expectedPatchEntries,
		Height:         3,
	}

	sort.Sort(patch.ContractUpdatesInfo.AllDataEntries)
	sort.Sort(expectedL2Patch.AllDataEntries)

	assert.Equal(t, patch.ContractUpdatesInfo.AllDataEntries, expectedL2Patch.AllDataEntries)
	assert.Equal(t, patch.ContractUpdatesInfo.Height, expectedL2Patch.Height)
}
