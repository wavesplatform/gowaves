package blockchaininfo_test

import (
	"github.com/golang/mock/gomock"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/blockchaininfo"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const natsTestURL = "nats://127.0.0.1:4756"

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

func RunNatsTestServer() (*server.Server, error) {
	opts := &server.Options{
		MaxPayload: 1024 * 1024,
		Host:       "127.0.0.1",
		Port:       4756,
		NoSigs:     true,
	}
	s, err := server.NewServer(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create NATS server")
	}
	go s.Start()
	if !s.ReadyForConnections(5 * server.AUTH_TIMEOUT) {
		return nil, errors.New("NATS server is not ready for connections")
	}
	return s, nil
}

func TestSendRestartSignal(t *testing.T) {
	ts, err := RunNatsTestServer()
	require.NoError(t, err, "failed to run nats test server")
	defer ts.Shutdown()
	// Connect to NATS (adjust URL to match your environment).
	nc, err := nats.Connect(natsTestURL)
	require.NoError(t, err, "failed to connect to NATS")
	defer nc.Close()

	// Subscribe to the L2RequestsTopic to simulate a service that handles the request.
	_, err = nc.Subscribe(blockchaininfo.L2RequestsTopic, func(msg *nats.Msg) {
		if string(msg.Data) == blockchaininfo.RequestRestartSubTopic {
			_ = msg.Respond([]byte("ok"))
		} else {
			t.Errorf("unexpected message: %s", msg.Data)
		}
	})
	require.NoError(t, err, "Failed to subscribe to topic")

	// Call the function we're testing.
	msg, err := blockchaininfo.SendRestartSignal(nc)
	require.NoError(t, err, "Failed to send a restart signal")

	require.Equal(t, msg.Data, []byte("ok"))
}

//type BlockchainInfoExtensionStateTest struct {
//}
//
//func (be *BlockchainInfoExtensionStateTest) HasStateChanged() (bool, proto.BUpdatesInfo, error) {
//	return false, proto.BUpdatesInfo{}, nil
//}
//func (be *BlockchainInfoExtensionStateTest) StatesEqual(scheme proto.Scheme) (bool, proto.BUpdatesInfo, error) {
//	return false, proto.BUpdatesInfo{}, nil
//}
//func (be *BlockchainInfoExtensionStateTest) PublishContractUpdates(contractUpdates proto.L2ContractDataEntries, nc *nats.Conn) error {
//	return nil
//}
//func (be *BlockchainInfoExtensionStateTest) PublishBlockUpdates(updates proto.BUpdatesInfo, nc *nats.Conn, scheme proto.Scheme) error {
//	return nil
//}
//func (be *BlockchainInfoExtensionStateTest) PublishUpdates(updates proto.BUpdatesInfo, nc *nats.Conn, scheme proto.Scheme) error {
//	return nil
//}
//func (be *BlockchainInfoExtensionStateTest) AddEntriesToHistoryJournalAndCache(updates proto.BUpdatesInfo) {
//}
//func (be *BlockchainInfoExtensionStateTest) RollbackHappened(updates proto.BUpdatesInfo, previousState proto.BUpdatesInfo) bool {
//	return false
//}
//func (be *BlockchainInfoExtensionStateTest) GeneratePatch(latestUpdates proto.BUpdatesInfo) (proto.BUpdatesInfo, error) {
//	return proto.BUpdatesInfo{}, nil
//}
//func (be *BlockchainInfoExtensionStateTest) BuildPatch(keysForPatch []string, targetHeight uint64) (proto.DataEntries, error) {
//	return nil, nil
//}
//func (be *BlockchainInfoExtensionStateTest) CleanRecordsAfterRollback(latestHeightFromHistory uint64, heightAfterRollback uint64) error {
//	return nil
//}
//func (be *BlockchainInfoExtensionStateTest) SetPreviousState(updates proto.BUpdatesInfo) {}

func TestRollback(t *testing.T) {
	//var be *BUpdatesExtensionState
	//
	var updates proto.BUpdatesInfo
	//
	//var blockchainInfoExt BlockchainInfoExtensionStateTest
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBlockchaininfo := mock.NewMockUpdatesExtensionState(ctrl)
	mockBlockchaininfo.EXPECT().
		PublishBlockUpdates(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(updates, nc, scheme interface{}) { // do nothing
	})
	blockchaininfo.HandleRollback(mockBlockchaininfo, updates, nil, proto.TestNetScheme)
}
