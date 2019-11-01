package settings

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const mainnetGenesis = `
{
  "version": 1,
  "timestamp": 1460678400000,
  "reference": "67rpwLCuS5DGA8KGZXKsVQ7dnPb9goRLoKfgGbLfQg9WoLUgNY77E2jT11fem3coV9nAkguBACzrU1iyZM4B8roQ",
  "nxt-consensus": {
    "base-target": 153722867,
    "generation-signature": "11111111111111111111111111111111"
  },
  "signature": "FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2",
  "transactionBlockLength": 283,
  "transactionCount": 6,
  "transactions": [
    {
      "type": 1,
      "id": "2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8",
      "fee": 0,
      "timestamp": 1465742577614,
      "signature": "2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8",
      "recipient": "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ",
      "amount": 9999999500000000
    },
    {
      "type": 1,
      "id": "2TsxPS216SsZJAiep7HrjZ3stHERVkeZWjMPFcvMotrdGpFa6UCCmoFiBGNizx83Ks8DnP3qdwtJ8WFcN9J4exa3",
      "fee": 0,
      "timestamp": 1465742577614,
      "signature": "2TsxPS216SsZJAiep7HrjZ3stHERVkeZWjMPFcvMotrdGpFa6UCCmoFiBGNizx83Ks8DnP3qdwtJ8WFcN9J4exa3",
      "recipient": "3P8JdJGYc7vaLu4UXUZc1iRLdzrkGtdCyJM",
      "amount": 100000000
    },
    {
      "type": 1,
      "id": "3gF8LFjhnZdgEVjP7P6o1rvwapqdgxn7GCykCo8boEQRwxCufhrgqXwdYKEg29jyPWthLF5cFyYcKbAeFvhtRNTc",
      "fee": 0,
      "timestamp": 1465742577614,
      "signature": "3gF8LFjhnZdgEVjP7P6o1rvwapqdgxn7GCykCo8boEQRwxCufhrgqXwdYKEg29jyPWthLF5cFyYcKbAeFvhtRNTc",
      "recipient": "3PAGPDPqnGkyhcihyjMHe9v36Y4hkAh9yDy",
      "amount": 100000000
    },
    {
      "type": 1,
      "id": "5hjSPLDyqic7otvtTJgVv73H3o6GxgTBqFMTY2PqAFzw2GHAnoQddC4EgWWFrAiYrtPadMBUkoepnwFHV1yR6u6g",
      "fee": 0,
      "timestamp": 1465742577614,
      "signature": "5hjSPLDyqic7otvtTJgVv73H3o6GxgTBqFMTY2PqAFzw2GHAnoQddC4EgWWFrAiYrtPadMBUkoepnwFHV1yR6u6g",
      "recipient": "3P9o3ZYwtHkaU1KxsKkFjJqJKS3dLHLC9oF",
      "amount": 100000000
    },
    {
      "type": 1,
      "id": "ivP1MzTd28yuhJPkJsiurn2rH2hovXqxr7ybHZWoRGUYKazkfaL9MYoTUym4sFgwW7WB5V252QfeFTsM6Uiz3DM",
      "fee": 0,
      "timestamp": 1465742577614,
      "signature": "ivP1MzTd28yuhJPkJsiurn2rH2hovXqxr7ybHZWoRGUYKazkfaL9MYoTUym4sFgwW7WB5V252QfeFTsM6Uiz3DM",
      "recipient": "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3",
      "amount": 100000000
    },
    {
      "type": 1,
      "id": "29gnRjk8urzqc9kvqaxAfr6niQTuTZnq7LXDAbd77nydHkvrTA4oepoMLsiPkJ8wj2SeFB5KXASSPmbScvBbfLiV",
      "fee": 0,
      "timestamp": 1465742577614,
      "signature": "29gnRjk8urzqc9kvqaxAfr6niQTuTZnq7LXDAbd77nydHkvrTA4oepoMLsiPkJ8wj2SeFB5KXASSPmbScvBbfLiV",
      "recipient": "3PBWXDFUc86N2EQxKJmW8eFco65xTyMZx6J",
      "amount": 100000000
    }
  ]
}
`

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

	AddressSchemeCharacter proto.Scheme

	AverageBlockDelaySeconds uint64
	// Configurable.
	MaxBaseTarget uint64

	// Block Reward
	BlockRewardTerm         uint64
	InitialBlockReward      uint64
	BlockRewardIncrement    uint64
	BlockRewardVotingPeriod uint64
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

func DefaultSettingsForCustomBlockchain(genesisGetter GenesisGetter) *BlockchainSettings {
	return &BlockchainSettings{
		Type: Custom,
		FunctionalitySettings: FunctionalitySettings{
			FeaturesVotingPeriod:      5000,
			VotesForFeatureActivation: 4000,

			MaxTxTimeBackOffset:    120 * 60000,
			MaxTxTimeForwardOffset: 90 * 60000,

			AddressSchemeCharacter: 'C',

			AverageBlockDelaySeconds: 60,
			MaxBaseTarget:            math.MaxUint64,
		},
		GenesisGetter: genesisGetter,
	}
}

var (
	MainNetSettings = &BlockchainSettings{
		Type: MainNet,
		FunctionalitySettings: FunctionalitySettings{
			FeaturesVotingPeriod:                          5000,
			VotesForFeatureActivation:                     4000,
			PreactivatedFeatures:                          nil,
			DoubleFeaturesPeriodsAfterHeight:              810000,
			GenerationBalanceDepthFrom50To1000AfterHeight: 232000,
			BlockVersion3AfterHeight:                      795000,
			ResetEffectiveBalanceAtHeight:                 462000,
			StolenAliasesWindowTimeStart:                  1522463241035,
			StolenAliasesWindowTimeEnd:                    1530161445559,
			ReissueBugWindowTimeStart:                     1522463241035,
			ReissueBugWindowTimeEnd:                       1530161445559,
			AllowMultipleLeaseCancelUntilTime:             1492768800000,
			AllowLeasedBalanceTransferUntilTime:           1513357014002,
			CheckTempNegativeAfterTime:                    1479168000000,
			TxChangesSortedCheckAfterTime:                 1479416400000,
			TxFromFutureCheckAfterTime:                    1479168000000,
			UnissuedAssetUntilTime:                        1479416400000,
			InvalidReissueInSameBlockUntilTime:            1492768800000,
			MinimalGeneratingBalanceCheckAfterTime:        1479168000000,
			MaxTxTimeBackOffset:                           120 * 60000,
			MaxTxTimeForwardOffset:                        90 * 60000,
			AddressSchemeCharacter:                        proto.MainNetScheme,
			AverageBlockDelaySeconds:                      60,
			MaxBaseTarget:                                 math.MaxUint64,
			BlockRewardTerm:                               100000,
			InitialBlockReward:                            600000000,
			BlockRewardIncrement:                          50000000,
			BlockRewardVotingPeriod:                       10000,
		},
		GenesisGetter: MainnetGenesis,
	}

	TestNetSettings = &BlockchainSettings{
		Type: TestNet,
		FunctionalitySettings: FunctionalitySettings{
			FeaturesVotingPeriod:                          3000,
			VotesForFeatureActivation:                     2700,
			PreactivatedFeatures:                          nil,
			DoubleFeaturesPeriodsAfterHeight:              math.MaxUint64,
			GenerationBalanceDepthFrom50To1000AfterHeight: 0,
			BlockVersion3AfterHeight:                      161700,
			ResetEffectiveBalanceAtHeight:                 51500,
			StolenAliasesWindowTimeStart:                  0,
			StolenAliasesWindowTimeEnd:                    0,
			ReissueBugWindowTimeStart:                     1520411086003,
			ReissueBugWindowTimeEnd:                       1523096218005,
			AllowMultipleLeaseCancelUntilTime:             1492560000000,
			AllowLeasedBalanceTransferUntilTime:           1508230496004,
			CheckTempNegativeAfterTime:                    1477958400000,
			TxChangesSortedCheckAfterTime:                 1479416400000,
			TxFromFutureCheckAfterTime:                    1478100000000,
			UnissuedAssetUntilTime:                        1479416400000,
			InvalidReissueInSameBlockUntilTime:            1492560000000,
			MinimalGeneratingBalanceCheckAfterTime:        0,
			MaxTxTimeBackOffset:                           120 * 60000,
			MaxTxTimeForwardOffset:                        90 * 60000,
			AddressSchemeCharacter:                        proto.TestNetScheme,
			AverageBlockDelaySeconds:                      60,
			MaxBaseTarget:                                 math.MaxUint64,
			BlockRewardTerm:                               100000,
			InitialBlockReward:                            600000000,
			BlockRewardIncrement:                          50000000,
			BlockRewardVotingPeriod:                       10000,
		},
		GenesisGetter: TestnetGenesis,
	}
)

type GenesisGetter interface {
	Get() (*proto.Block, error)
}

type EmbeddedGenesisGetter struct {
}

func (a EmbeddedGenesisGetter) Get() (*proto.Block, error) {
	genesis := &proto.Block{}
	err := json.Unmarshal([]byte(mainnetGenesis), genesis)
	if err != nil {
		return nil, err
	}
	return genesis, nil
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

var MainnetGenesis = EmbeddedGenesisGetter{}
var TestnetGenesis = FromCurrentDir("../state/genesis", "testnet.json")
