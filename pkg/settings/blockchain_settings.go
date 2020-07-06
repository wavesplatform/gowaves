package settings

import (
	"encoding/json"
	"io"
	"math"
	"strings"

	"github.com/pkg/errors"
	"github.com/rakyll/statik/fs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	_ "github.com/wavesplatform/gowaves/pkg/settings/embedded"
)

type BlockchainType byte

const (
	MainNet BlockchainType = iota
	TestNet
	StageNet
	Custom
	Integration
)

type FunctionalitySettings struct {
	// Features.
	FeaturesVotingPeriod             uint64  `json:"features_voting_period"`
	VotesForFeatureActivation        uint64  `json:"votes_for_feature_activation"`
	PreactivatedFeatures             []int16 `json:"preactivated_features"`
	DoubleFeaturesPeriodsAfterHeight uint64  `json:"double_features_periods_after_height"`

	// Heights when some of rules change.
	GenerationBalanceDepthFrom50To1000AfterHeight uint64 `json:"generation_balance_depth_from_50_to_1000_after_height"`
	BlockVersion3AfterHeight                      uint64 `json:"block_version_3_after_height"`

	// Lease cancellation.
	ResetEffectiveBalanceAtHeight uint64 `json:"reset_effective_balance_at_height"`
	// Window when stolen aliases are valid.
	StolenAliasesWindowTimeStart uint64 `json:"stolen_aliases_window_time_start"`
	StolenAliasesWindowTimeEnd   uint64 `json:"stolen_aliases_window_time_end"`
	// Window when unreissueable assets can be reissued.
	ReissueBugWindowTimeStart           uint64 `json:"reissue_bug_window_time_start"`
	ReissueBugWindowTimeEnd             uint64 `json:"reissue_bug_window_time_end"`
	AllowMultipleLeaseCancelUntilTime   uint64 `json:"allow_multiple_lease_cancel_until_time"`
	AllowLeasedBalanceTransferUntilTime uint64 `json:"allow_leased_balance_transfer_until_time"`
	// Timestamps when different kinds of checks become relevant.
	CheckTempNegativeAfterTime             uint64 `json:"check_temp_negative_after_time"`
	TxChangesSortedCheckAfterTime          uint64 `json:"tx_changes_sorted_check_after_time"`
	TxFromFutureCheckAfterTime             uint64 `json:"tx_from_future_check_after_time"`
	UnissuedAssetUntilTime                 uint64 `json:"unissued_asset_until_time"`
	InvalidReissueInSameBlockUntilTime     uint64 `json:"invalid_reissue_in_same_block_until_time"`
	MinimalGeneratingBalanceCheckAfterTime uint64 `json:"minimal_generating_balance_check_after_time"`

	// Diff in milliseconds.
	MaxTxTimeBackOffset    uint64 `json:"max_tx_time_back_offset"`
	MaxTxTimeForwardOffset uint64 `json:"max_tx_time_forward_offset"`

	AddressSchemeCharacter proto.Scheme `json:"address_scheme_character"`

	AverageBlockDelaySeconds uint64 `json:"average_block_delay_seconds"`
	// Configurable.
	MaxBaseTarget uint64 `json:"max_base_target"`

	// Block Reward
	BlockRewardTerm         uint64 `json:"block_reward_term"`
	InitialBlockReward      uint64 `json:"initial_block_reward"`
	BlockRewardIncrement    uint64 `json:"block_reward_increment"`
	BlockRewardVotingPeriod uint64 `json:"block_reward_voting_period"`

	MinUpdateAssetInfoInterval uint64 `json:"min_update_asset_info_interval"`
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
	Type    BlockchainType `json:"type"`
	Genesis proto.Block    `json:"genesis"`
}

var (
	MainNetSettings       = mustLoadEmbeddedSettings(MainNet)
	TestNetSettings       = mustLoadEmbeddedSettings(TestNet)
	StageNetSettings      = mustLoadEmbeddedSettings(StageNet)
	DefaultCustomSettings = &BlockchainSettings{
		Type: Custom,
		FunctionalitySettings: FunctionalitySettings{
			FeaturesVotingPeriod:       5000,
			VotesForFeatureActivation:  4000,
			MaxTxTimeBackOffset:        120 * 60000,
			MaxTxTimeForwardOffset:     90 * 60000,
			AddressSchemeCharacter:     proto.CustomNetScheme,
			AverageBlockDelaySeconds:   60,
			MaxBaseTarget:              math.MaxUint64,
			MinUpdateAssetInfoInterval: 100000,
		},
	}
)

func GetIntegrationSetting() *BlockchainSettings {
	rs, err := loadEmbeddedSettings("/integration.json")
	if err != nil {
		panic(err)
	}
	return rs
}

func mustLoadEmbeddedSettings(blockchain BlockchainType) *BlockchainSettings {
	switch blockchain {
	case MainNet:
		s, err := loadEmbeddedSettings("/mainnet.json")
		if err != nil {
			panic(err)
		}
		return s

	case TestNet:
		s, err := loadEmbeddedSettings("/testnet.json")
		if err != nil {
			panic(err)
		}
		return s

	case StageNet:
		s, err := loadEmbeddedSettings("/stagenet.json")
		if err != nil {
			panic(err)
		}
		return s

	default:
		panic("no embedded settings")
	}
}

func ReadBlockchainSettings(r io.Reader) (*BlockchainSettings, error) {
	jsonParser := json.NewDecoder(r)
	s := &BlockchainSettings{}
	if err := jsonParser.Decode(s); err != nil {
		return nil, errors.Wrap(err, "failed to read blockchain settings")
	}
	return s, nil
}

func loadEmbeddedSettings(name string) (*BlockchainSettings, error) {
	root, err := fs.New()
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize built-in storage")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open genesis file")
	}
	defer func() {
		_ = file.Close()
	}()
	return ReadBlockchainSettings(file)
}

func BlockchainSettingsByTypeName(networkType string) (*BlockchainSettings, error) {
	switch strings.ToLower(networkType) {
	case "mainnet":
		return MainNetSettings, nil
	case "testnet":
		return TestNetSettings, nil
	case "stagenet":
		return StageNetSettings, nil
	case "custom":
		return nil, errors.New("no embedded settings for custom blockchain")
	default:
		return nil, errors.New("invalid blockchain type string")
	}
}
