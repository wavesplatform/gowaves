package settings

import (
	"embed"
	"encoding/json"
	"io"
	"math"
	"strings"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type BlockchainType byte

const (
	MainNet BlockchainType = iota
	TestNet
	StageNet
	Custom
)

// Default it is params for FairPosCalculatorV2
const (
	minBlockTimeDefault                        = float64(15000)
	delayDeltaDefault                          = 8
	lightNodeBlockFieldsAbsenceIntervalDefault = 1000
)

const (
	lenWithDAOAddress                = 1
	lenWithDAOAndXTNBuybackAddresses = 2
)

var (
	//go:embed embedded
	res embed.FS
)

type FunctionalitySettings struct {
	// Features.
	FeaturesVotingPeriod              uint64  `json:"features_voting_period"`
	VotesForFeatureActivation         uint64  `json:"votes_for_feature_activation"`
	PreactivatedFeatures              []int16 `json:"preactivated_features"`
	DoubleFeaturesPeriodsAfterHeight  uint64  `json:"double_features_periods_after_height"`
	SponsorshipSingleActivationPeriod bool    `json:"sponsorship_single_activation_period"`

	// Heights when some rules change.
	GenerationBalanceDepthFrom50To1000AfterHeight uint64 `json:"generation_balance_depth_from_50_to_1000_after_height"`
	BlockVersion3AfterHeight                      uint64 `json:"block_version_3_after_height"`

	// Lease cancellation.
	ResetEffectiveBalanceAtHeight uint64 `json:"reset_effective_balance_at_height"`
	// Window when stolen aliases are valid.
	StolenAliasesWindowTimeStart uint64 `json:"stolen_aliases_window_time_start"`
	StolenAliasesWindowTimeEnd   uint64 `json:"stolen_aliases_window_time_end"`
	// Window when non-reissuable assets can be reissued.
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
	// PaymentsFixAfterHeight == 'paymentsCheckHeight' in scala node - reject any invoke tx
	// after this height if account balance become negative
	PaymentsFixAfterHeight                              uint64 `json:"payments_fix_after_height"`
	InternalInvokePaymentsValidationAfterHeight         uint64 `json:"internal_invoke_payments_validation_after_height"`
	InternalInvokeCorrectFailRejectBehaviourAfterHeight uint64 `json:"internal_invoke_correct_fail_reject_behaviour_after_height"`
	InvokeNoZeroPaymentsAfterHeight                     uint64 `json:"invoke_no_zero_payments_after_height"`

	// Diff in milliseconds.
	MaxTxTimeBackOffset    uint64 `json:"max_tx_time_back_offset"`
	MaxTxTimeForwardOffset uint64 `json:"max_tx_time_forward_offset"`

	AddressSchemeCharacter proto.Scheme `json:"address_scheme_character"`

	AverageBlockDelaySeconds uint64 `json:"average_block_delay_seconds"`

	// FairPosCalculator
	DelayDelta uint64 `json:"delay_delta"`
	// In Milliseconds.
	MinBlockTime float64 `json:"min_block_time"`

	// Configurable.
	MaxBaseTarget uint64 `json:"max_base_target"`

	// Block Reward
	BlockRewardTerm         uint64               `json:"block_reward_term"`
	BlockRewardTermAfter20  uint64               `json:"block_reward_term_after_20"`
	InitialBlockReward      uint64               `json:"initial_block_reward"`
	BlockRewardIncrement    uint64               `json:"block_reward_increment"`
	BlockRewardVotingPeriod uint64               `json:"block_reward_voting_period"`
	RewardAddresses         []proto.WavesAddress `json:"reward_addresses"`
	RewardAddressesAfter21  []proto.WavesAddress `json:"reward_addresses_after_21"`
	MinXTNBuyBackPeriod     uint64               `json:"min_xtn_buy_back_period"`
	BlockRewardBoostPeriod  uint64               `json:"block_reward_boost_period"`

	MinUpdateAssetInfoInterval uint64 `json:"min_update_asset_info_interval"`

	LightNodeBlockFieldsAbsenceInterval uint64 `json:"light_node_block_fields_absence_interval"`
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

func (f *FunctionalitySettings) CanReissueNonReissueablePeriod(currentTimestamp uint64) bool {
	// Due to bugs in existing blockchain it is valid to reissue non-reissuable asset in this time period.
	return currentTimestamp <= f.InvalidReissueInSameBlockUntilTime ||
		(currentTimestamp >= f.ReissueBugWindowTimeStart) && (currentTimestamp <= f.ReissueBugWindowTimeEnd)
}

func (f *FunctionalitySettings) CurrentBlockRewardTerm(isCappedRewardActivated bool) uint64 {
	if isCappedRewardActivated {
		return f.BlockRewardTermAfter20
	}
	return f.BlockRewardTerm
}

func (f *FunctionalitySettings) BlockRewardVotingThreshold() uint64 {
	return f.BlockRewardVotingPeriod/2 + 1
}

func (f *FunctionalitySettings) CurrentRewardAddresses(isXTNBuyBackCessationActivated bool) []proto.WavesAddress {
	if isXTNBuyBackCessationActivated {
		return f.RewardAddressesAfter21
	}
	return f.RewardAddresses
}

func (f *FunctionalitySettings) DAOAddress(isXTNBuyBackCessationActivated bool) (proto.WavesAddress, bool) {
	addresses := f.CurrentRewardAddresses(isXTNBuyBackCessationActivated)
	if len(addresses) >= lenWithDAOAddress {
		return addresses[0], true
	}
	return proto.WavesAddress{}, false
}

func (f *FunctionalitySettings) XTNBuybackAddress(isXTNBuyBackCessationActivated bool) (proto.WavesAddress, bool) {
	if !isXTNBuyBackCessationActivated && len(f.RewardAddresses) >= lenWithDAOAndXTNBuybackAddresses {
		return f.RewardAddresses[1], true
	}
	return proto.WavesAddress{}, false
}

func (f *FunctionalitySettings) RangeForGeneratingBalanceByHeight(height proto.Height) (uint64, uint64) {
	// Depth for generating balance calculation (in number of blocks).
	const (
		firstDepth  = 50
		secondDepth = 1000
	)
	depth := uint64(firstDepth)
	if height >= f.GenerationBalanceDepthFrom50To1000AfterHeight {
		depth = secondDepth
	}
	bottomLimit := height - depth + 1
	if height < depth {
		bottomLimit = 1
	}
	return bottomLimit, height
}

type BlockchainSettings struct {
	FunctionalitySettings
	Type    BlockchainType `json:"type"`
	Genesis proto.Block    `json:"genesis"`
}

func (s *BlockchainSettings) UnmarshalJSON(bytes []byte) error {
	type shadowed *BlockchainSettings
	if err := json.Unmarshal(bytes, shadowed(s)); err != nil {
		return err
	}
	return s.validate()
}

// validate BlockchainSettings according to the scala node rules
func (s *BlockchainSettings) validate() error {
	if s.BlockRewardTerm < s.BlockRewardVotingPeriod {
		return errors.New("'block_reward_term' cannot be greater than 'block_reward_voting_period'")
	}
	if s.BlockRewardTermAfter20 < s.BlockRewardVotingPeriod {
		return errors.New("'block_reward_term_after_20' cannot be greater than 'block_reward_voting_period'")
	}
	return nil
}

var (
	MainNetSettings  = mustLoadEmbeddedSettings(MainNet)
	TestNetSettings  = mustLoadEmbeddedSettings(TestNet)
	StageNetSettings = mustLoadEmbeddedSettings(StageNet)
	defaultSettings  = BlockchainSettings{
		FunctionalitySettings: FunctionalitySettings{
			MinBlockTime:                        minBlockTimeDefault,
			DelayDelta:                          delayDeltaDefault,
			LightNodeBlockFieldsAbsenceInterval: lightNodeBlockFieldsAbsenceIntervalDefault,
		},
	}
	DefaultCustomSettings = BlockchainSettings{
		Type: Custom,
		FunctionalitySettings: FunctionalitySettings{
			FeaturesVotingPeriod:                5000,
			VotesForFeatureActivation:           4000,
			MaxTxTimeBackOffset:                 120 * 60000,
			MaxTxTimeForwardOffset:              90 * 60000,
			AddressSchemeCharacter:              proto.CustomNetScheme,
			AverageBlockDelaySeconds:            60,
			MaxBaseTarget:                       math.MaxUint64,
			MinUpdateAssetInfoInterval:          100000,
			BlockRewardTerm:                     100000,
			BlockRewardTermAfter20:              50000,
			LightNodeBlockFieldsAbsenceInterval: lightNodeBlockFieldsAbsenceIntervalDefault,
		},
	}
)

func mustLoadEmbeddedSettings(blockchain BlockchainType) *BlockchainSettings {
	switch blockchain {
	case MainNet:
		s, err := loadEmbeddedSettings("embedded/mainnet.json")
		if err != nil {
			panic(err)
		}
		return s

	case TestNet:
		s, err := loadEmbeddedSettings("embedded/testnet.json")
		if err != nil {
			panic(err)
		}
		return s

	case StageNet:
		s, err := loadEmbeddedSettings("embedded/stagenet.json")
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
	s := defaultSettings
	if err := jsonParser.Decode(&s); err != nil {
		return nil, errors.Wrap(err, "failed to read blockchain settings")
	}
	return &s, nil
}

func loadEmbeddedSettings(name string) (*BlockchainSettings, error) {
	file, err := res.Open(name)
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
