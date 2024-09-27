package config

import (
	"encoding/json"
	stderrs "errors"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	testdataFolder       = "testdata"
	rewardSettingsFolder = "reward_settings_testdata"
)

// RewardSettings stores parts of genesis configuration related to rewards and features.
// It's used to modify the blockchain settings in test on rewards.
// TODO: Separate into 2 or more structs: one for rewards and one for features.
type RewardSettings struct {
	BlockRewardVotingPeriod uint64        `json:"voting_interval"`
	BlockRewardTerm         uint64        `json:"term"`
	BlockRewardTermAfter20  uint64        `json:"term_after_capped_reward_feature"`
	InitialBlockReward      uint64        `json:"initial_block_reward"`
	BlockRewardIncrement    uint64        `json:"block_reward_increment"`
	DesiredBlockReward      uint64        `json:"desired_reward"`
	DaoAddress              string        `json:"dao_address"`
	XtnBuybackAddress       string        `json:"xtn_buyback_address"`
	MinXTNBuyBackPeriod     uint64        `json:"min_xtn_buy_back_period"`
	PreactivatedFeatures    []FeatureInfo `json:"preactivated_features"`
	SupportedFeatures       []int16       `json:"supported_features"`
}

// NewRewardSettingsFromFile reads reward settings from file.
// The `path` is a relative path to the configuration JSON file inside the project's "rewards_settings_testdata" folder.
func NewRewardSettingsFromFile(dir, file string) (_ *RewardSettings, err error) {
	var pwd string
	pwd, err = os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read reward settings")
	}
	rewardSettingsPath := filepath.Join(pwd, testdataFolder, rewardSettingsFolder, dir, file)
	var f *os.File
	f, err = os.Open(rewardSettingsPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read reward settings")
	}
	defer func() {
		if clErr := f.Close(); clErr != nil {
			err = stderrs.Join(err, errors.Wrapf(clErr, "failed to close reward settings file %q", f.Name()))
		}
	}()

	js := json.NewDecoder(f)
	s := &RewardSettings{}
	if jsErr := js.Decode(s); jsErr != nil {
		return nil, errors.Wrap(jsErr, "failed to read reward settings")
	}
	return s, nil
}
