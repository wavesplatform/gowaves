package blockchaininfo

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

//func TestAddressFromPublicKey(t *testing.T) {
//	tests := []struct {
//		publicKey string
//		scheme    byte
//		address   string
//	}{
//		{"5CnGfSjguYfzWzaRmbxzCbF5qRNGTXEvayytSANkqQ6A", MainNetScheme, "3PQ8bp1aoqHQo3icNqFv6VM36V1jzPeaG1v"},
//		{"BstqhtQjQN9X78i6mEpaNnf6cMsZZRDVHNv3CqguXbxq", MainNetScheme, "3PQvBCHPnxXprTNq1rwdcDuxt6VGKRTM9wT"},
//		{"FckK43s6tQ9BBW77hSKuyRnfnrKuf6B7sEuJzcgkSDVf", MainNetScheme, "3PETfqHg9HyL92nfiujN5fBW6Ac1TYiVAAc"},
//		{"5CnGfSjguYfzWzaRmbxzCbF5qRNGTXEvayytSANkqQ6A", TestNetScheme, "3NC7nrggwhk2AbRC7kzv92yDjbVyALeGzE5"},
//		{"BstqhtQjQN9X78i6mEpaNnf6cMsZZRDVHNv3CqguXbxq", TestNetScheme, "3NCuNExVvpzSE15QkngdemY9XCyVVGhHA9h"},
//		{"5CnGfSjguYfzWzaRmbxzCbF5qRNGTXEvayytSANkqQ6A", 'x', "3cgHWJbRKGEhi32DEe6ucVV24FfF7u2mxit"},
//		{"BstqhtQjQN9X78i6mEpaNnf6cMsZZRDVHNv3CqguXbxq", 'x', "3ch55gsEJPV7mSgRsfnd8E3wqs8mSqyTNCj"},
//	}
//
//
//
//	for _, tc := range tests {
//		if b, err := base58.Decode(tc.publicKey); assert.NoError(t, err) {
//			var pk crypto.PublicKey
//			copy(pk[:], b)
//			if address, err := NewAddressFromPublicKey(tc.scheme, pk); assert.NoError(t, err) {
//				assert.Equal(t, tc.address, address.String())
//			}
//		}
//	}
//}

func testBlockUpdates() BlockUpdatesInfo {
	var b BlockUpdatesInfo

	var (
		height      uint64 = 100
		VRF                = proto.B58Bytes{}
		BlockID            = proto.BlockID{}
		BlockHeader        = proto.BlockHeader{}
	)

	b.Height = &height
	b.VRF = &VRF
	b.BlockID = &BlockID
	b.BlockHeader = &BlockHeader

	return b
}

func TestChangesGeneration(t *testing.T) {
	previousDataEntries := []proto.DataEntry{
		&proto.BinaryDataEntry{Key: "block_0x3a85dedc42db076c91cf61d72fa17c80777aeed70ba68dbc14d6829dd6e88614", Value: []byte("base64:AAAAAAAQYVwAAAAAADDSCpJJsd11jrOMW7AS/AHIMIDQXjqmFyhDuGt2RPNvmcCXAVTy/URmfMOj7GNweXnZpzidmxHfPBfcP5A=")}, // height 3199498
		&proto.BinaryDataEntry{Key: "block_0x3a9209ce524553a75fd0e9bde5c99ff254b1fb231916fc89755be957e51e5516", Value: []byte("base64:AAAAAAAQYywAAAAAADDSQBBubtiRmKwtaNFF1TrBhsfBu61fj3qiSrtyu1/kLLAlAVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199552
		&proto.BinaryDataEntry{Key: "block_0x3b0181d3f66d9f0ddd8e1e8567b836a01f652b4cb873aa7b7c46fc8bd1e4eeee", Value: []byte("base64:AAAAAAAQZEUAAAAAADDSZeUUyashoWjUKurzA/wVU5prm68CambvjIo1ESLoLuAaAVRaS4vOsPl9cxvg7aeRj7RFZQzdpmvV/+A=")}, // height 3199589
	}
	var previousHeight uint64 = 3199552

	currentDataEntries := []proto.DataEntry{
		&proto.BinaryDataEntry{Key: "block_0x3b5ad5c176473be02cc3d19207204af87af03f6fd75c76916765745658f7e842", Value: []byte("base64:AAAAAAAQZKkAAAAAADDSb6xEaq4RsFQruGNeGdooPmtLBnlERR15qzc/mcKcQ461AVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199599
		&proto.BinaryDataEntry{Key: "block_0x3b72ee917fea7057fb88a357f619c22f6f8ddae03b701fab7c284953ecebbc8c", Value: []byte("base64:AAAAAAAQZQkAAAAAADDSe+2CGv9zgiR7s65XEBkYzIbv6jbxcR7Zi3ByUqsX0bkwAVTEkyC5glOJH8Upe49iT3+BUV5zRaDT2dM=")}, // height 3199611
		&proto.BinaryDataEntry{Key: "block_0x3b973acae11f248a524b463db7d198c7ddb47fd8aeda2f14699e639a0db19911", Value: []byte("base64:AAAAAAAQZf8AAAAAADDSolzqc5gjHWP/sCzqK7+HkAjybjGxq8SxL9ID8yEIKxrlAVRN71D/MD4dykS8vqW7cXqCh5QOclg6DEU=")}, // height 3199650
	}
	var currentHeight uint64 = 3199611

	previousBlockInfo := BUpdatesInfo{
		BlockUpdatesInfo: testBlockUpdates(),
		ContractUpdatesInfo: L2ContractDataEntries{
			AllDataEntries: &previousDataEntries,
			Height:         &previousHeight,
		},
	}

	currentBlockInfo := BUpdatesInfo{
		BlockUpdatesInfo: testBlockUpdates(),
		ContractUpdatesInfo: L2ContractDataEntries{
			AllDataEntries: &currentDataEntries,
			Height:         &currentHeight,
		},
	}

	equal, changes, err := compareBUpdatesInfo(currentBlockInfo, previousBlockInfo, proto.TestNetScheme, StoreBlocksLimit)
	if err != nil {
		return
	}
	require.False(t, equal)

	fmt.Println(changes.BlockUpdatesInfo)

}

func TestChangesGenerationSecond(t *testing.T) {
	previousDataEntries := []proto.DataEntry{
		&proto.BinaryDataEntry{Key: "block_0x3a85dedc42db076c91cf61d72fa17c80777aeed70ba68dbc14d6829dd6e88614", Value: []byte("base64:AAAAAAAQYVwAAAAAADDSCpJJsd11jrOMW7AS/AHIMIDQXjqmFyhDuGt2RPNvmcCXAVTy/URmfMOj7GNweXnZpzidmxHfPBfcP5A=")}, // height 3199498
		&proto.BinaryDataEntry{Key: "block_0x3a9209ce524553a75fd0e9bde5c99ff254b1fb231916fc89755be957e51e5516", Value: []byte("base64:AAAAAAAQYywAAAAAADDSQBBubtiRmKwtaNFF1TrBhsfBu61fj3qiSrtyu1/kLLAlAVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199552
		&proto.BinaryDataEntry{Key: "block_0x3b0181d3f66d9f0ddd8e1e8567b836a01f652b4cb873aa7b7c46fc8bd1e4eeee", Value: []byte("base64:AAAAAAAQZEUAAAAAADDSZeUUyashoWjUKurzA/wVU5prm68CambvjIo1ESLoLuAaAVRaS4vOsPl9cxvg7aeRj7RFZQzdpmvV/+A=")}, // height 3199589
	}
	var previousHeight uint64 = 3199552

	currentDataEntries := []proto.DataEntry{
		&proto.BinaryDataEntry{Key: "block_0x3a85dedc42db076c91cf61d72fa17c80777aeed70ba68dbc14d6829dd6e88614", Value: []byte("base64:AAAAAAAQYVwAAAAAADDSCpJJsd11jrOMW7AS/AHIMIDQXjqmFyhDuGt2RPNvmcCXAVTy/URmfMOj7GNweXnZpzidmxHfPBfcP5A=")}, // height 3199498
		&proto.BinaryDataEntry{Key: "block_0x3a9209ce524553a75fd0e9bde5c99ff254b1fb231916fc89755be957e51e5516", Value: []byte("base64:AAAAAAAQYywAAAAAADDSQBBubtiRmKwtaNFF1TrBhsfBu61fj3qiSrtyu1/kLLAlAVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199552
		&proto.BinaryDataEntry{Key: "block_0x3b0181d3f66d9f0ddd8e1e8567b836a01f652b4cb873aa7b7c46fc8bd1e4eeee", Value: []byte("base64:AAAAAAAQZEUAAAAAADDSZeUUyashoWjUKurzA/wVU5prm68CambvjIo1ESLoLuAaAVRaS4vOsPl9cxvg7aeRj7RFZQzdpmvV/+A=")}, // height 3199589

		&proto.BinaryDataEntry{Key: "block_0x3b5ad5c176473be02cc3d19207204af87af03f6fd75c76916765745658f7e842", Value: []byte("base64:AAAAAAAQZKkAAAAAADDSb6xEaq4RsFQruGNeGdooPmtLBnlERR15qzc/mcKcQ461AVQp5GtuF7Hxji8CQ9SFOEZLLUv88nvIgg8=")}, // height 3199599
		&proto.BinaryDataEntry{Key: "block_0x3b72ee917fea7057fb88a357f619c22f6f8ddae03b701fab7c284953ecebbc8c", Value: []byte("base64:AAAAAAAQZQkAAAAAADDSe+2CGv9zgiR7s65XEBkYzIbv6jbxcR7Zi3ByUqsX0bkwAVTEkyC5glOJH8Upe49iT3+BUV5zRaDT2dM=")}, // height 3199611
		&proto.BinaryDataEntry{Key: "block_0x3b973acae11f248a524b463db7d198c7ddb47fd8aeda2f14699e639a0db19911", Value: []byte("base64:AAAAAAAQZf8AAAAAADDSolzqc5gjHWP/sCzqK7+HkAjybjGxq8SxL9ID8yEIKxrlAVRN71D/MD4dykS8vqW7cXqCh5QOclg6DEU=")}, // height 3199650
	}
	var currentHeight uint64 = 3199611

	previousBlockInfo := BUpdatesInfo{
		BlockUpdatesInfo: testBlockUpdates(),
		ContractUpdatesInfo: L2ContractDataEntries{
			AllDataEntries: &previousDataEntries,
			Height:         &previousHeight,
		},
	}

	currentBlockInfo := BUpdatesInfo{
		BlockUpdatesInfo: testBlockUpdates(),
		ContractUpdatesInfo: L2ContractDataEntries{
			AllDataEntries: &currentDataEntries,
			Height:         &currentHeight,
		},
	}

	equal, changes, err := compareBUpdatesInfo(currentBlockInfo, previousBlockInfo, proto.TestNetScheme, StoreBlocksLimit)
	if err != nil {
		return
	}
	require.False(t, equal)

	fmt.Println(changes.BlockUpdatesInfo)

}
