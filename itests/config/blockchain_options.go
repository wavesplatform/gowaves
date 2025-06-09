package config

import "github.com/pkg/errors"

// BlockchainOption is a function type that allows to set additional parameters to BlockchainConfig.
type BlockchainOption func(*BlockchainConfig) error

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

// WithQuorum sets the quorum (number of connected peers) required to start block generation.
func WithQuorum(quorum int) BlockchainOption {
	return func(cfg *BlockchainConfig) error {
		if quorum < 0 {
			return errors.Errorf("invalid quorum size %d", quorum)
		}
		cfg.quorum = quorum
		return nil
	}
}
