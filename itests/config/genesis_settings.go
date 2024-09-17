package config

import (
	"encoding/binary"
	"encoding/json"
	slerr "errors"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/genesis_generator"
)

const (
	genesisSettingsFileName = "genesis.json"
	configFolder            = "config"
	testdataFolder          = "testdata"
	rewardConfigFolder      = "reward_settings_testdata"

	maxBaseTarget = 1000000

	defaultBlockRewardVotingPeriod = 3
	defaultBlockRewardTerm         = 10
	defaultBlockRewardTermAfter20  = 5
	defaultInitialBlockReward      = 600000000
	defaultBlockRewardIncrement    = 100000000
	defaultDesiredBlockReward      = 600000000
	defaultMinXTNBuyBackPeriod     = 3
)

var (
	averageHit = big.NewInt(math.MaxUint64 / 2)
)

type GenesisConfig struct {
	GenesisTimestamp  int64
	GenesisSignature  crypto.Signature
	GenesisBaseTarget types.BaseTarget
	AverageBlockDelay uint64
	Transaction       []genesis_generator.GenesisTransactionInfo
}

type DistributionItem struct {
	SeedText string `json:"seed_text"`
	Amount   uint64 `json:"amount"`
	IsMiner  bool   `json:"is_miner"`
}

type FeatureInfo struct {
	Feature int16  `json:"feature"`
	Height  uint64 `json:"height"`
}

type GenesisSettings struct {
	Scheme               proto.Scheme
	SchemeRaw            string             `json:"scheme"`
	AverageBlockDelay    uint64             `json:"average_block_delay"`
	MinBlockTime         float64            `json:"min_block_time"`
	DelayDelta           uint64             `json:"delay_delta"`
	Distributions        []DistributionItem `json:"distributions"`
	PreactivatedFeatures []FeatureInfo      `json:"preactivated_features"`
}

type scalaCustomOptions struct {
	Features          []FeatureInfo
	EnableMining      bool
	DaoAddress        string `json:"dao_address"`
	XtnBuybackAddress string `json:"xtn_buyback_address"`
}

type goEnvOptions struct {
	DesiredBlockReward string `json:"desired_reward"`
	SupportedFeatures  string `json:"supported_features"`
}

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

type config struct {
	BlockchainSettings *settings.BlockchainSettings
	ScalaOpts          *scalaCustomOptions
	GoOpts             *goEnvOptions
}

func parseGenesisSettings() (*GenesisSettings, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Clean(filepath.Join(pwd, configFolder, genesisSettingsFileName))
	f, err := os.Open(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = slerr.Join(err, closeErr)
		}
	}()
	jsonParser := json.NewDecoder(f)
	s := &GenesisSettings{}
	if err = jsonParser.Decode(s); err != nil {
		return nil, errors.Wrap(err, "failed to decode genesis settings")
	}
	s.Scheme = s.SchemeRaw[0]
	return s, nil
}

func parseRewardSettings(rewardArgsPath string) (*RewardSettings, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	rewardSettingsPath := filepath.Clean(filepath.Join(pwd, testdataFolder, rewardConfigFolder, rewardArgsPath))
	f, err := os.Open(rewardSettingsPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = slerr.Join(err, closeErr)
		}
	}()
	jsonParser := json.NewDecoder(f)
	s := &RewardSettings{}
	if err = jsonParser.Decode(s); err != nil {
		return nil, errors.Wrap(err, "failed to decode reward settings")
	}
	return s, nil
}

func getRewardAddresses(rewardSettings *RewardSettings) ([]proto.WavesAddress, []proto.WavesAddress) {
	var rewardAddresses []proto.WavesAddress
	var rewardAddressesAfter21 []proto.WavesAddress

	if rewardSettings.DaoAddress != "" {
		rewardAddresses = append(rewardAddresses, proto.MustAddressFromString(rewardSettings.DaoAddress))
		rewardAddressesAfter21 = append(rewardAddressesAfter21, proto.MustAddressFromString(rewardSettings.DaoAddress))
	}

	if rewardSettings.XtnBuybackAddress != "" {
		rewardAddresses = append(rewardAddresses, proto.MustAddressFromString(rewardSettings.XtnBuybackAddress))
	}
	return rewardAddresses, rewardAddressesAfter21
}

func getPreactivatedFeatures(genSettings *GenesisSettings, rewardSettings *RewardSettings) ([]FeatureInfo, error) {
	var initPF, result []FeatureInfo
	initPF = append(initPF, genSettings.PreactivatedFeatures...)
	initPF = append(initPF, rewardSettings.PreactivatedFeatures...)
	for _, pf := range initPF {
		if pf.Feature <= 0 {
			err := errors.Errorf("Feature with id %d not exist", pf.Feature)
			return nil, err
		}
		result = append(result, pf)
	}
	return result, nil
}

func getSupportedFeaturesAsString(rewardSettings *RewardSettings) string {
	values := rewardSettings.SupportedFeatures
	valuesStr := make([]string, 0, len(values))
	for i := range values {
		valuesStr = append(valuesStr, strconv.FormatInt(int64(values[i]), 10))
	}
	result := strings.Join(valuesStr, ",")
	return result
}

func newBlockchainConfig(additionalArgsPath ...string) (*config, []AccountInfo, error) {
	var rewardSettings *RewardSettings
	genSettings, err := parseGenesisSettings()
	if err != nil {
		return nil, nil, err
	}

	switch l := len(additionalArgsPath); l {
	case 0:
		// default values for some reward parameters
		rewardSettings = &RewardSettings{
			BlockRewardVotingPeriod: defaultBlockRewardVotingPeriod,
			BlockRewardTerm:         defaultBlockRewardTerm,
			BlockRewardTermAfter20:  defaultBlockRewardTermAfter20,
			InitialBlockReward:      defaultInitialBlockReward,
			BlockRewardIncrement:    defaultBlockRewardIncrement,
			DesiredBlockReward:      defaultDesiredBlockReward,
			MinXTNBuyBackPeriod:     defaultMinXTNBuyBackPeriod,
		}
	case 1:
		rewardSettings, err = parseRewardSettings(additionalArgsPath[0])
		if err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, errors.Errorf("unexpected additional arguments count: want 0 or 1, got %d, args=%+v",
			l, additionalArgsPath)
	}

	ts := time.Now().UnixMilli()
	txs, acc, err := makeTransactionAndKeyPairs(genSettings, uint64(ts))
	if err != nil {
		return nil, nil, err
	}
	bt, err := calcInitialBaseTarget(genSettings)
	if err != nil {
		return nil, nil, err
	}
	b, err := genesis_generator.GenerateGenesisBlock(genSettings.Scheme, txs, bt, uint64(ts))
	if err != nil {
		return nil, nil, err
	}

	cfg := settings.MustDefaultCustomSettings()
	cfg.Genesis = *b
	cfg.AddressSchemeCharacter = genSettings.Scheme
	cfg.AverageBlockDelaySeconds = genSettings.AverageBlockDelay
	cfg.MinBlockTime = genSettings.MinBlockTime
	cfg.DelayDelta = genSettings.DelayDelta
	cfg.DoubleFeaturesPeriodsAfterHeight = 0
	cfg.SponsorshipSingleActivationPeriod = true
	cfg.MinUpdateAssetInfoInterval = 2

	cfg.FeaturesVotingPeriod = 1
	cfg.VotesForFeatureActivation = 1

	// reward settings
	cfg.InitialBlockReward = rewardSettings.InitialBlockReward
	cfg.BlockRewardIncrement = rewardSettings.BlockRewardIncrement
	cfg.BlockRewardVotingPeriod = rewardSettings.BlockRewardVotingPeriod
	cfg.BlockRewardTermAfter20 = rewardSettings.BlockRewardTermAfter20
	cfg.BlockRewardTerm = rewardSettings.BlockRewardTerm
	cfg.MinXTNBuyBackPeriod = rewardSettings.MinXTNBuyBackPeriod

	rewardsAddresses, rewardsAddressesAfter21 := getRewardAddresses(rewardSettings)
	cfg.RewardAddresses = rewardsAddresses
	cfg.RewardAddressesAfter21 = rewardsAddressesAfter21

	// preactivated features
	preactivatedFeatures, err := getPreactivatedFeatures(genSettings, rewardSettings)
	if err != nil {
		return nil, nil, err
	}
	cfg.PreactivatedFeatures = make([]int16, len(preactivatedFeatures))
	for i, f := range preactivatedFeatures {
		cfg.PreactivatedFeatures[i] = f.Feature
	}

	return &config{
		BlockchainSettings: cfg,
		ScalaOpts: &scalaCustomOptions{Features: preactivatedFeatures, EnableMining: false,
			DaoAddress: rewardSettings.DaoAddress, XtnBuybackAddress: rewardSettings.XtnBuybackAddress},
		GoOpts: &goEnvOptions{DesiredBlockReward: strconv.FormatUint(rewardSettings.DesiredBlockReward, 10),
			SupportedFeatures: getSupportedFeaturesAsString(rewardSettings)},
	}, acc, nil
}

type AccountInfo struct {
	PublicKey crypto.PublicKey
	SecretKey crypto.SecretKey
	Amount    uint64
	Address   proto.WavesAddress
}

func makeTransactionAndKeyPairs(settings *GenesisSettings, timestamp uint64) ([]genesis_generator.GenesisTransactionInfo, []AccountInfo, error) {
	r := make([]genesis_generator.GenesisTransactionInfo, 0, len(settings.Distributions))
	accounts := make([]AccountInfo, 0, len(settings.Distributions))
	for _, dist := range settings.Distributions {
		seed := []byte(dist.SeedText)
		iv := [4]byte{}
		binary.BigEndian.PutUint32(iv[:], uint32(0))
		s := append(iv[:], seed...)
		h, err := crypto.SecureHash(s)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to generate hash from seed '%s'", string(seed))
		}
		sk, pk, err := crypto.GenerateKeyPair(h[:])
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to generate keyPair from seed '%s'", string(seed))
		}
		addr, err := proto.NewAddressFromPublicKey(settings.Scheme, pk)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to generate address from seed '%s'", string(seed))
		}
		r = append(r, genesis_generator.GenesisTransactionInfo{Address: addr, Amount: dist.Amount, Timestamp: timestamp})
		accounts = append(accounts, AccountInfo{PublicKey: pk, SecretKey: sk, Amount: dist.Amount, Address: addr})
	}
	return r, accounts, nil
}

func calculateBaseTarget(pos consensus.PosCalculator, minBT types.BaseTarget, maxBT types.BaseTarget, balance uint64, averageDelay uint64) (types.BaseTarget, error) {
	if maxBT-minBT <= 1 {
		return maxBT, nil
	}
	newBT := (maxBT + minBT) / 2
	delay, err := pos.CalculateDelay(averageHit, newBT, balance)
	if err != nil {
		return 0, err
	}
	diff := int64(delay) - int64(averageDelay)*1000
	if (diff >= 0 && diff < 100) || (diff < 0 && diff > -100) {
		return newBT, nil
	}

	var min, max uint64
	if delay > averageDelay*1000 {
		min, max = newBT, maxBT
	} else {
		min, max = minBT, newBT
	}
	return calculateBaseTarget(pos, min, max, balance, averageDelay)
}

func isFeaturePreactivated(features []FeatureInfo, feature int16) bool {
	for _, f := range features {
		if f.Feature == feature {
			return true
		}
	}
	return false
}

func getPosCalculator(genSettings *GenesisSettings) consensus.PosCalculator {
	fairActivated := isFeaturePreactivated(genSettings.PreactivatedFeatures, int16(settings.FairPoS))
	if fairActivated {
		blockV5Activated := isFeaturePreactivated(genSettings.PreactivatedFeatures, int16(settings.BlockV5))
		if blockV5Activated {
			return consensus.NewFairPosCalculator(genSettings.DelayDelta, genSettings.MinBlockTime)
		}
		return consensus.FairPosCalculatorV1
	}
	return consensus.NXTPosCalculator
}

func calcInitialBaseTarget(genSettings *GenesisSettings) (types.BaseTarget, error) {
	maxBT := uint64(0)
	pos := getPosCalculator(genSettings)
	for _, acc := range genSettings.Distributions {
		if !acc.IsMiner {
			continue
		}
		bt, err := calculateBaseTarget(pos, consensus.MinBaseTarget, maxBaseTarget, acc.Amount, genSettings.AverageBlockDelay)
		if err != nil {
			return 0, err
		}
		if bt > maxBT {
			maxBT = bt
		}
	}
	return maxBT, nil
}
