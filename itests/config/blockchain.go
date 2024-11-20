package config

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/genesis_generator"
)

// RewardAddresses contains information about DAO and XTNBuyback addresses.
// Methods of RewardAddresses allow to represent this information in a form suitable for Go node configuration.
type RewardAddresses struct {
	DAORewardAddress  *proto.WavesAddress
	XTNBuybackAddress *proto.WavesAddress
}

// NewRewardAddresses creates RewardAddresses from two string representations of DAO and XTNBuyback addresses.
func NewRewardAddresses(daoAddress, xtnAddress string) (RewardAddresses, error) {
	r := RewardAddresses{}
	if len(daoAddress) != 0 {
		a, err := proto.NewAddressFromString(daoAddress)
		if err != nil {
			return RewardAddresses{}, errors.Wrap(err, "failed to create reward addresses")
		}
		r.DAORewardAddress = &a
	}
	if len(xtnAddress) != 0 {
		a, err := proto.NewAddressFromString(xtnAddress)
		if err != nil {
			return RewardAddresses{}, errors.Wrap(err, "failed to create reward addresses")
		}
		r.XTNBuybackAddress = &a
	}
	return r, nil
}

// Addresses returns DAO and XTNBuyback addresses as a slice of Waves addresses.
func (ra *RewardAddresses) Addresses() []proto.WavesAddress {
	r := make([]proto.WavesAddress, 0, 2)
	if ra.DAORewardAddress != nil {
		r = append(r, *ra.DAORewardAddress)
	}
	if ra.XTNBuybackAddress != nil {
		r = append(r, *ra.XTNBuybackAddress)
	}
	return r
}

// AddressesAfter21 returns DAO address as a slice of Waves addresses that doesn't contain XTNBuyback address to
// represent the set of reward addresses after the activation of feature 21.
func (ra *RewardAddresses) AddressesAfter21() []proto.WavesAddress {
	if ra.DAORewardAddress != nil {
		return []proto.WavesAddress{*ra.DAORewardAddress}
	}
	return []proto.WavesAddress{}
}

// BlockchainOption is a function type that allows to set additional parameters to BlockchainConfig.
type BlockchainOption func(*BlockchainConfig) error

// BlockchainConfig is a struct that contains settings for blockchain.
// This configuration is used both for building Scala and Go configuration files.
// Also, it's used to produce a Docker container run configurations for both nodes.
type BlockchainConfig struct {
	accounts      []AccountInfo
	supported     []int16
	desiredReward uint64

	Settings          *settings.BlockchainSettings
	Features          []FeatureInfo
	RewardAddresses   RewardAddresses
	EnableScalaMining bool
}

func NewBlockchainConfig(options ...BlockchainOption) (*BlockchainConfig, error) {
	gs, err := parseGenesisSettings()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create blockchain configuration")
	}

	// Generate new genesis block.
	ts := safeNow()
	txs, acs, err := makeTransactionAndKeyPairs(gs, ts)
	if err != nil {
		return nil, err
	}
	bt, err := calcInitialBaseTarget(gs)
	if err != nil {
		return nil, err
	}
	b, err := genesis_generator.GenerateGenesisBlock(gs.Scheme, txs, bt, ts)
	if err != nil {
		return nil, err
	}
	bs := settings.MustDefaultCustomSettings()
	bs.Genesis = *b
	bs.AddressSchemeCharacter = gs.Scheme
	bs.AverageBlockDelaySeconds = gs.AverageBlockDelay
	bs.MinBlockTime = gs.MinBlockTime
	bs.DelayDelta = gs.DelayDelta
	bs.DoubleFeaturesPeriodsAfterHeight = 0
	bs.SponsorshipSingleActivationPeriod = true
	bs.MinUpdateAssetInfoInterval = 2
	bs.FeaturesVotingPeriod = 1
	bs.VotesForFeatureActivation = 1
	bs.InitialBlockReward = defaultInitialBlockReward
	bs.BlockRewardIncrement = defaultBlockRewardIncrement
	bs.BlockRewardVotingPeriod = defaultBlockRewardVotingPeriod
	bs.BlockRewardTermAfter20 = defaultBlockRewardTermAfter20
	bs.BlockRewardTerm = defaultBlockRewardTerm
	bs.MinXTNBuyBackPeriod = defaultMinXTNBuyBackPeriod

	cfg := &BlockchainConfig{
		Settings:      bs,
		accounts:      acs,
		desiredReward: defaultDesiredBlockReward,
	}

	if ftErr := cfg.UpdatePreactivatedFeatures(gs.PreactivatedFeatures); ftErr != nil {
		return nil, errors.Wrap(ftErr, "failed to create blockchain configuration")
	}

	// Apply additional options.
	for _, opt := range options {
		if optErr := opt(cfg); optErr != nil {
			return nil, errors.Wrap(optErr, "failed to create blockchain configuration")
		}
	}
	return cfg, nil
}

// UpdatePreactivatedFeatures checks and inserts new preactivated features in BlockchainConfig.
func (c *BlockchainConfig) UpdatePreactivatedFeatures(features []FeatureInfo) error {
	for _, f := range features {
		if f.Feature <= 0 {
			return errors.Errorf("invalid feature ID '%d'", f.Feature)
		}
		if !slices.ContainsFunc(c.Features, func(fi FeatureInfo) bool {
			return fi.Feature == f.Feature
		}) {
			c.Features = append(c.Features, f)
		}
	}
	// Replace preactivated features of blockchain settings with a new set of features.
	c.Settings.PreactivatedFeatures = make([]int16, len(c.Features))
	for i, f := range c.Features {
		c.Settings.PreactivatedFeatures[i] = f.Feature
	}
	return nil
}

func (c *BlockchainConfig) SupportedFeaturesString() string {
	ss := make([]string, len(c.supported))
	for i, s := range c.supported {
		ss[i] = strconv.FormatInt(int64(s), 10)
	}
	return strings.Join(ss, ",")
}

func (c *BlockchainConfig) DesiredBlockRewardString() string {
	return strconv.FormatUint(c.desiredReward, 10)
}

func (c *BlockchainConfig) TestConfig() TestConfig {
	return TestConfig{
		Accounts:           c.accounts,
		BlockchainSettings: c.Settings,
	}
}

// WithFeatureSettingFromFile is a BlockchainOption that allows to set feature settings from configuration file.
// Feature settings configuration file is a JSON file with the structure of `featureSettings`.
func WithFeatureSettingFromFile(path ...string) BlockchainOption {
	return func(cfg *BlockchainConfig) error {
		fs, err := NewFeatureSettingsFromFile(path...)
		if err != nil {
			return errors.Wrap(err, "failed to modify features settings")
		}
		cfg.supported = fs.SupportedFeatures
		if ftErr := cfg.UpdatePreactivatedFeatures(fs.PreactivatedFeatures); ftErr != nil {
			return errors.Wrap(ftErr, "failed to modify preactivated features")
		}
		return nil
	}
}

// WithPaymentsSettingFromFile is a BlockchainOption that allows to set payment settings from configuration file.
// Payment settings configuration file is a JSON file with the structure of `paymentSettings`.
func WithPaymentsSettingFromFile(path ...string) BlockchainOption {
	return func(cfg *BlockchainConfig) error {
		fs, err := NewPaymentSettingsFromFile(path...)
		if err != nil {
			return errors.Wrap(err, "failed to modify payments settings")
		}
		cfg.Settings.PaymentsFixAfterHeight = fs.PaymentsFixAfterHeight
		cfg.Settings.InternalInvokePaymentsValidationAfterHeight = fs.InternalInvokePaymentsValidationAfterHeight
		cfg.Settings.InternalInvokeCorrectFailRejectBehaviourAfterHeight =
			fs.InternalInvokeCorrectFailRejectBehaviourAfterHeight
		cfg.Settings.InvokeNoZeroPaymentsAfterHeight = fs.InvokeNoZeroPaymentsAfterHeight
		return nil
	}
}

// WithRewardSettingFromFile is a BlockchainOption that allows to set reward settings from configuration file.
// Reward settings configuration file is a JSON file with the structure of `rewardSettings`.
func WithRewardSettingFromFile(path ...string) BlockchainOption {
	return func(cfg *BlockchainConfig) error {
		rs, err := NewRewardSettingsFromFile(path...)
		if err != nil {
			return errors.Wrap(err, "failed to modify reward settings")
		}
		cfg.Settings.InitialBlockReward = rs.InitialBlockReward
		cfg.Settings.BlockRewardIncrement = rs.BlockRewardIncrement
		cfg.Settings.BlockRewardVotingPeriod = rs.BlockRewardVotingPeriod
		cfg.Settings.BlockRewardTermAfter20 = rs.BlockRewardTermAfter20
		cfg.Settings.BlockRewardTerm = rs.BlockRewardTerm
		cfg.Settings.MinXTNBuyBackPeriod = rs.MinXTNBuyBackPeriod

		ras, err := NewRewardAddresses(rs.DaoAddress, rs.XtnBuybackAddress)
		if err != nil {
			return errors.Wrap(err, "failed to modify reward settings")
		}
		cfg.RewardAddresses = ras
		cfg.Settings.RewardAddresses = ras.Addresses()
		cfg.Settings.RewardAddressesAfter21 = ras.AddressesAfter21()
		cfg.desiredReward = rs.DesiredBlockReward
		return nil
	}
}

func WithScalaMining() BlockchainOption {
	return func(cfg *BlockchainConfig) error {
		cfg.EnableScalaMining = true
		return nil
	}
}

func safeNow() uint64 {
	now := time.Now().UnixMilli()
	if now < 0 {
		return 0
	}
	return uint64(now)
}
