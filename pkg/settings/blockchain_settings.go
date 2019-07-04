package settings

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

type BlockchainType byte

const (
	MainNet BlockchainType = iota
	TestNet
	Custom
)

type FunctionalitySettings struct {
	// Features.
	FeaturesVotingPeriod             uint64
	VotesForFeatureActivation        uint64
	PreactivatedFeatures             []int16
	DoubleFeaturesPeriodsAfterHeight uint64

	// Heights when some of rules change.
	GenerationBalanceDepthFrom50To1000AfterHeight uint64
	BlockVersion3AfterHeight                      uint64

	// Lease cancellation.
	ResetEffectiveBalanceAtHeight uint64
	// Window when stolen aliases are valid.
	StolenAliasesWindowTimeStart uint64
	StolenAliasesWindowTimeEnd   uint64
	// Window when unreissueable assets can be reissued.
	ReissueBugWindowTimeStart           uint64
	ReissueBugWindowTimeEnd             uint64
	AllowMultipleLeaseCancelUntilTime   uint64
	AllowLeasedBalanceTransferUntilTime uint64
	// Timestamps when different kinds of checks become relevant.
	CheckTempNegativeAfterTime             uint64
	TxChangesSortedCheckAfterTime          uint64
	TxFromFutureCheckAfterTime             uint64
	UnissuedAssetUntilTime                 uint64
	InvalidReissueInSameBlockUntilTime     uint64
	MinimalGeneratingBalanceCheckAfterTime uint64

	// Diff in milliseconds.
	MaxTxTimeBackOffset    uint64
	MaxTxTimeForwardOffset uint64

	AddressSchemeCharacter proto.Schema

	AverageBlockDelaySeconds uint64
	// Configurable.
	MaxBaseTarget uint64
}

func (f *FunctionalitySettings) VotesForFeatureElection(height uint64) uint64 {
	if height > f.DoubleFeaturesPeriodsAfterHeight {
		return f.VotesForFeatureActivation * 2
	} else {
		return f.VotesForFeatureActivation
	}
}

func (f *FunctionalitySettings) ActivationWindowSize(height uint64) uint64 {
	if height > f.DoubleFeaturesPeriodsAfterHeight {
		return f.FeaturesVotingPeriod * 2
	} else {
		return f.FeaturesVotingPeriod
	}
}

type BlockchainSettings struct {
	FunctionalitySettings
	Type BlockchainType
	// GenesisGetter is way how you get genesis file.
	GenesisGetter GenesisGetter
}

var (
	MainNetSettings = &BlockchainSettings{
		Type: MainNet,
		FunctionalitySettings: FunctionalitySettings{
			FeaturesVotingPeriod:             5000,
			VotesForFeatureActivation:        4000,
			DoubleFeaturesPeriodsAfterHeight: 810000,

			GenerationBalanceDepthFrom50To1000AfterHeight: 232000,
			BlockVersion3AfterHeight:                      795000,

			ResetEffectiveBalanceAtHeight:          462000,
			StolenAliasesWindowTimeStart:           1522463241035,
			StolenAliasesWindowTimeEnd:             1530161445559,
			ReissueBugWindowTimeStart:              1522463241035,
			ReissueBugWindowTimeEnd:                1530161445559,
			AllowMultipleLeaseCancelUntilTime:      1492768800000,
			AllowLeasedBalanceTransferUntilTime:    1513357014002,
			CheckTempNegativeAfterTime:             1479168000000,
			TxChangesSortedCheckAfterTime:          1479416400000,
			TxFromFutureCheckAfterTime:             1479168000000,
			UnissuedAssetUntilTime:                 1479416400000,
			InvalidReissueInSameBlockUntilTime:     1492768800000,
			MinimalGeneratingBalanceCheckAfterTime: 1479168000000,

			MaxTxTimeBackOffset:    120 * 60000,
			MaxTxTimeForwardOffset: 90 * 60000,

			AddressSchemeCharacter: proto.MainNetScheme,

			AverageBlockDelaySeconds: 60,
			MaxBaseTarget:            200,
		},
		GenesisGetter: MainnetGenesis,
	}

	TestNetSettings = &BlockchainSettings{
		Type: TestNet,
		FunctionalitySettings: FunctionalitySettings{
			FeaturesVotingPeriod:             3000,
			VotesForFeatureActivation:        2700,
			DoubleFeaturesPeriodsAfterHeight: math.MaxUint64,

			GenerationBalanceDepthFrom50To1000AfterHeight: 0,
			BlockVersion3AfterHeight:                      161700,

			ResetEffectiveBalanceAtHeight:          51500,
			ReissueBugWindowTimeStart:              1520411086003,
			ReissueBugWindowTimeEnd:                1523096218005,
			AllowMultipleLeaseCancelUntilTime:      1492560000000,
			AllowLeasedBalanceTransferUntilTime:    1508230496004,
			CheckTempNegativeAfterTime:             1477958400000,
			TxChangesSortedCheckAfterTime:          1479416400000,
			TxFromFutureCheckAfterTime:             1478100000000,
			UnissuedAssetUntilTime:                 1479416400000,
			InvalidReissueInSameBlockUntilTime:     1492560000000,
			MinimalGeneratingBalanceCheckAfterTime: 0,

			MaxTxTimeBackOffset:    120 * 60000,
			MaxTxTimeForwardOffset: 90 * 60000,

			AddressSchemeCharacter: proto.TestNetScheme,

			AverageBlockDelaySeconds: 60,
			MaxBaseTarget:            200,
		},
		GenesisGetter: TestnetGenesis,
	}
)

type GenesisGetter interface {
	Get() (*proto.Block, error)
}

type localGenesisGetter struct {
	paths []string
}

func (a localGenesisGetter) Get() (*proto.Block, error) {
	genesisCfgPath := filepath.Join(a.paths...)
	return fromPath(genesisCfgPath)
}

type absoluteGenesisGetter struct {
	path []string
}

func (a absoluteGenesisGetter) Get() (*proto.Block, error) {
	return fromPath(filepath.Join(a.path...))
}

func fromPath(genesisCfgPath string) (*proto.Block, error) {
	genesisFile, err := os.Open(genesisCfgPath)
	if err != nil {
		return nil, errors.Errorf("failed to open genesis file: %v\n", err)
	}
	jsonParser := json.NewDecoder(genesisFile)
	genesis := proto.Block{}
	if err := jsonParser.Decode(&genesis); err != nil {
		return nil, errors.Errorf("failed to parse JSON of genesis block: %v\n", err)
	}
	if err := genesisFile.Close(); err != nil {
		return nil, errors.Errorf("failed to close genesis file: %v\n", err)
	}
	return &genesis, nil
}

func getLocalDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("Unable to find current package file")
	}
	return filepath.Dir(filename), nil
}

func getCurrentDir() (string, error) {
	_, filename, _, ok := runtime.Caller(2)
	if !ok {
		return "", errors.Errorf("Unable to find current package file")
	}
	return filepath.Dir(filename), nil
}

func FromCurrentDir(path ...string) GenesisGetter {
	c, _ := getCurrentDir()
	return localGenesisGetter{
		paths: append([]string{c}, path...),
	}
}

func FromPath(path ...string) GenesisGetter {
	return absoluteGenesisGetter{
		path: path,
	}
}

var MainnetGenesis = FromCurrentDir("../state/genesis", "mainnet.json")
var TestnetGenesis = FromCurrentDir("../state/genesis", "testnet.json")
