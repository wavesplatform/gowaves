package config

import "github.com/pkg/errors"

// BlockchainOption is a function type that allows to set additional parameters to BlockchainConfig.
type BlockchainOption func(*BlockchainConfig) error

// WithRewardSettingFromFile is a BlockchainOption that allows to set reward settings from configuration file.
// Reward settings configuration file is a JSON file with the structure of `rewardSettings`.
func WithRewardSettingFromFile(dir, file string) BlockchainOption {
	return func(cfg *BlockchainConfig) error {
		rs, err := NewRewardSettingsFromFile(dir, file)
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

		cfg.supported = rs.SupportedFeatures
		cfg.desiredReward = rs.DesiredBlockReward

		if ftErr := cfg.UpdatePreactivatedFeatures(rs.PreactivatedFeatures); ftErr != nil {
			return errors.Wrap(ftErr, "failed to modify preactivated features")
		}
		return nil
	}
}

// WithNoScalaMining disables mining on the Scala node.
func WithNoScalaMining() BlockchainOption {
	return func(cfg *BlockchainConfig) error {
		cfg.disableScalaMining = true
		return nil
	}
}

// WithNoGoMining disables mining on the Go node.
func WithNoGoMining() BlockchainOption {
	return func(cfg *BlockchainConfig) error {
		cfg.disableGoMining = true
		return nil
	}
}

func WithPreactivatedFeatures(features []FeatureInfo) BlockchainOption {
	return func(cfg *BlockchainConfig) error {
		if ftErr := cfg.UpdatePreactivatedFeatures(features); ftErr != nil {
			return errors.Wrap(ftErr, "failed to modify preactivated features")
		}
		return nil
	}
}

func WithAbsencePeriod(period uint64) BlockchainOption {
	return func(cfg *BlockchainConfig) error {
		cfg.Settings.LightNodeBlockFieldsAbsenceInterval = period
		return nil
	}
}
