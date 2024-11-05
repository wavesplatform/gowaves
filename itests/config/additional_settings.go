package config

import (
	"encoding/json"
	stderrs "errors"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	testdataFolder = "testdata"
)

func readSettingsFromFile(path ...string) (*os.File, error) {
	var pwd string
	var err error
	var f *os.File

	pwd, err = os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current working directory")
	}
	settingPath := filepath.Join(pwd, filepath.Join(path...))

	f, err = os.Open(filepath.Clean(settingPath))
	if err != nil {
		return nil, errors.Wrap(err, "could not open settings file")
	}
	return f, nil
}

// PaymentSettings stores parts of genesis configuration related to payments.
// It's used to modify the blockchain settings.
type PaymentSettings struct {
	PaymentsFixAfterHeight                              uint64 `json:"payments_fix_after_height"`
	InternalInvokePaymentsValidationAfterHeight         uint64 `json:"internal_invoke_payments_validation_after_height"`
	InternalInvokeCorrectFailRejectBehaviourAfterHeight uint64 `json:"internal_invoke_correct_fail_reject_behaviour_after_height"` //nolint:lll
	InvokeNoZeroPaymentsAfterHeight                     uint64 `json:"invoke_no_zero_payments_after_height"`
}

// NewPaymentSettingsFromFile reads payment settings from file.
// The `path` is a relative path to the configuration JSON file.
func NewPaymentSettingsFromFile(path ...string) (_ *PaymentSettings, err error) {
	f, err := readSettingsFromFile(testdataFolder, filepath.Join(path...))
	defer func() {
		if cErr := f.Close(); cErr != nil {
			err = stderrs.Join(err, errors.Wrapf(cErr, "could not close settings file %q", f.Name()))
		}
	}()
	js := json.NewDecoder(f)
	s := &PaymentSettings{}
	if jsErr := js.Decode(s); jsErr != nil {
		return nil, errors.Wrap(jsErr, "failed to read payment settings")
	}
	return s, nil
}

// FeatureSettings stores parts of genesis configuration related to features.
// It's used to modify the blockchain settings.
type FeatureSettings struct {
	PreactivatedFeatures []FeatureInfo `json:"preactivated_features"`
	SupportedFeatures    []int16       `json:"supported_features"`
}

// NewFeatureSettingsFromFile reads features settings from file.
// The `path` is a relative path to the configuration JSON file.
func NewFeatureSettingsFromFile(path ...string) (_ *FeatureSettings, err error) {
	f, err := readSettingsFromFile(testdataFolder, filepath.Join(path...))
	defer func() {
		if cErr := f.Close(); cErr != nil {
			err = stderrs.Join(err, errors.Wrapf(cErr, "could not close settings file %q", f.Name()))
		}
	}()
	js := json.NewDecoder(f)
	s := &FeatureSettings{}
	if jsErr := js.Decode(s); jsErr != nil {
		return nil, errors.Wrap(jsErr, "failed to read features settings")
	}
	return s, nil
}

// RewardSettings stores parts of genesis configuration related to rewards.
// It's used to modify the blockchain settings in test on rewards.
type RewardSettings struct {
	BlockRewardVotingPeriod uint64 `json:"voting_interval"`
	BlockRewardTerm         uint64 `json:"term"`
	BlockRewardTermAfter20  uint64 `json:"term_after_capped_reward_feature"`
	InitialBlockReward      uint64 `json:"initial_block_reward"`
	BlockRewardIncrement    uint64 `json:"block_reward_increment"`
	DesiredBlockReward      uint64 `json:"desired_reward"`
	DaoAddress              string `json:"dao_address"`
	XtnBuybackAddress       string `json:"xtn_buyback_address"`
	MinXTNBuyBackPeriod     uint64 `json:"min_xtn_buy_back_period"`
}

// NewRewardSettingsFromFile reads reward settings from file.
// The `path` is a relative path to the configuration JSON file inside the project's "rewards_settings_testdata" folder.
func NewRewardSettingsFromFile(path ...string) (_ *RewardSettings, err error) {
	f, err := readSettingsFromFile(testdataFolder, filepath.Join(path...))
	defer func() {
		if cErr := f.Close(); cErr != nil {
			err = stderrs.Join(err, errors.Wrapf(cErr, "could not close settings file %q", f.Name()))
		}
	}()
	js := json.NewDecoder(f)
	s := &RewardSettings{}
	if jsErr := js.Decode(s); jsErr != nil {
		return nil, errors.Wrap(jsErr, "failed to read reward settings")
	}
	return s, nil
}
