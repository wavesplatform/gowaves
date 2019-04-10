package settings

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type BlockchainType byte

const (
	MainNet BlockchainType = iota
	TestNet
	Custom
)

type FunctionalitySettings struct {
	// Heights when some of rules change.
	GenerationBalanceDepthFrom50To1000AfterHeight uint64
	BlockVersion3AfterHeight                      uint64

	// Timestamps when different kinds of checks become relevant.
	NegativeBalanceCheckAfterTime          uint64
	TxChangesSortedCheckAfterTime          uint64
	TxFromFutureCheckAfterTime             uint64
	UnissedAssetUntilTime                  uint64
	InvalidReissueInSameBlockUntilTime     uint64
	MinimalGeneratingBalanceCheckAfterTime uint64

	// Diff in milliseconds.
	MaxTxTimeBackOffset    uint64
	MaxTxTimeForwardOffset uint64

	AddressSchemeCharacter byte

	AverageBlockDelaySeconds uint64
	// Configurable.
	MaxBaseTarget uint64
}

type BlockchainSettings struct {
	FunctionalitySettings
	Type BlockchainType
	// GenesisCfgPath is path to JSON file with complete representation of genesis block.
	// Leave this field blank for MainNet/TestNet, it is only used for custom blockchains.
	GenesisCfgPath string
}

var (
	MainNetSettings = &BlockchainSettings{
		Type: MainNet,
		FunctionalitySettings: FunctionalitySettings{
			GenerationBalanceDepthFrom50To1000AfterHeight: 232000,
			BlockVersion3AfterHeight:                      795000,

			NegativeBalanceCheckAfterTime:          1479168000000,
			TxChangesSortedCheckAfterTime:          1479416400000,
			TxFromFutureCheckAfterTime:             1479168000000,
			UnissedAssetUntilTime:                  1479416400000,
			InvalidReissueInSameBlockUntilTime:     1492768800000,
			MinimalGeneratingBalanceCheckAfterTime: 1479168000000,

			MaxTxTimeBackOffset:    120 * 60000,
			MaxTxTimeForwardOffset: 90 * 60000,

			AddressSchemeCharacter: proto.MainNetScheme,

			AverageBlockDelaySeconds: 60,
			MaxBaseTarget:            200,
		},
	}

	TestNetSettings = &BlockchainSettings{
		Type: TestNet,
		FunctionalitySettings: FunctionalitySettings{
			GenerationBalanceDepthFrom50To1000AfterHeight: 0,
			BlockVersion3AfterHeight:                      161700,

			NegativeBalanceCheckAfterTime:          1477958400000,
			TxChangesSortedCheckAfterTime:          1479416400000,
			TxFromFutureCheckAfterTime:             1478100000000,
			UnissedAssetUntilTime:                  1479416400000,
			InvalidReissueInSameBlockUntilTime:     1492560000000,
			MinimalGeneratingBalanceCheckAfterTime: 0,

			MaxTxTimeBackOffset:    120 * 60000,
			MaxTxTimeForwardOffset: 90 * 60000,

			AddressSchemeCharacter: proto.TestNetScheme,

			AverageBlockDelaySeconds: 60,
			MaxBaseTarget:            200,
		},
	}
)
