package config

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
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

	maxBaseTarget = 1000000
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
}

type FeatureInfo struct {
	Feature int16  `json:"feature"`
	Height  uint64 `json:"height"`
}

type GenesisSettings struct {
	Scheme            proto.Scheme `json:"address_scheme_character"`
	AverageBlockDelay uint64       `json:"average_block_delay"` // in sec

	// In Milliseconds.
	MinBlockTime float64 `json:"min_block_time"`
	// FairPosCalculator
	DelayDelta                 uint64 `json:"delay_delta"`
	MinUpdateAssetInfoInterval uint64 `json:"min_update_asset_info_interval"` // in blocks
	// Lease cancellation.
	ResetEffectiveBalanceAtHeight uint64 `json:"reset_effective_balance_at_height"`
	// Heights when some rules change.
	GenerationBalanceDepthFrom50To1000AfterHeight uint64 `json:"generation_balance_depth_from_50_to_1000_after_height"`
	BlockVersion3AfterHeight                      uint64 `json:"block_version_3_after_height"`
	// Diff in milliseconds.
	MaxTxTimeBackOffset    uint64 `json:"max_tx_time_back_offset"`
	MaxTxTimeForwardOffset uint64 `json:"max_tx_time_forward_offset"`
	// Block Reward
	BlockRewardTerm         uint64 `json:"block_reward_term"`
	InitialBlockReward      uint64 `json:"initial_block_reward"`
	BlockRewardIncrement    uint64 `json:"block_reward_increment"`
	BlockRewardVotingPeriod uint64 `json:"block_reward_voting_period"`

	// Features.
	FeaturesVotingPeriod              uint64        `json:"features_voting_period"`
	VotesForFeatureActivation         uint64        `json:"votes_for_feature_activation"`
	PreactivatedFeatures              []FeatureInfo `json:"preactivated_features"`
	DoubleFeaturesPeriodsAfterHeight  uint64        `json:"double_features_periods_after_height"`
	SponsorshipSingleActivationPeriod bool          `json:"sponsorship_single_activation_period"`

	Distributions []DistributionItem `json:"distributions"`
}

type ScalaCustomOptions struct {
	Features     []FeatureInfo
	EnableMining bool
}

type Config struct {
	BlockchainSettings *settings.BlockchainSettings
	ScalaOpts          *ScalaCustomOptions
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
	jsonParser := json.NewDecoder(f)
	s := &GenesisSettings{}
	if err = jsonParser.Decode(s); err != nil {
		return nil, errors.Wrap(err, "failed to decode genesis settings")
	}
	return s, nil
}

func NewBlockchainConfig() (*Config, []AccountInfo, error) {
	genSettings, err := parseGenesisSettings()
	if err != nil {
		return nil, nil, err
	}
	ts := time.Now().UnixMilli()
	txs, acc, err := makeTransactionAndKeyPairs(genSettings, uint64(ts))
	if err != nil {
		return nil, nil, err
	}
	bt, err := calcInitialBaseTarget(acc, genSettings)
	if err != nil {
		return nil, nil, err
	}
	b, err := genesis_generator.GenerateGenesisBlock(genSettings.Scheme, txs, bt, uint64(ts))
	if err != nil {
		return nil, nil, err
	}

	cfg := settings.DefaultCustomSettings
	cfg.Genesis = *b
	cfg.AddressSchemeCharacter = genSettings.Scheme
	cfg.AverageBlockDelaySeconds = genSettings.AverageBlockDelay
	cfg.MinBlockTime = genSettings.MinBlockTime
	cfg.DelayDelta = genSettings.DelayDelta

	cfg.MinUpdateAssetInfoInterval = genSettings.MinUpdateAssetInfoInterval

	cfg.ResetEffectiveBalanceAtHeight = genSettings.ResetEffectiveBalanceAtHeight

	cfg.GenerationBalanceDepthFrom50To1000AfterHeight = genSettings.GenerationBalanceDepthFrom50To1000AfterHeight
	cfg.BlockVersion3AfterHeight = genSettings.BlockVersion3AfterHeight

	cfg.MaxTxTimeBackOffset = genSettings.MaxTxTimeBackOffset
	cfg.MaxTxTimeForwardOffset = genSettings.MaxTxTimeForwardOffset

	cfg.BlockRewardTerm = genSettings.BlockRewardTerm
	cfg.InitialBlockReward = genSettings.InitialBlockReward
	cfg.BlockRewardIncrement = genSettings.BlockRewardIncrement
	cfg.BlockRewardVotingPeriod = genSettings.BlockRewardVotingPeriod

	cfg.FeaturesVotingPeriod = genSettings.FeaturesVotingPeriod
	cfg.VotesForFeatureActivation = genSettings.VotesForFeatureActivation
	cfg.DoubleFeaturesPeriodsAfterHeight = genSettings.DoubleFeaturesPeriodsAfterHeight
	cfg.SponsorshipSingleActivationPeriod = genSettings.SponsorshipSingleActivationPeriod

	for _, feature := range genSettings.PreactivatedFeatures {
		cfg.PreactivatedFeatures = append(cfg.PreactivatedFeatures, feature.Feature)
	}
	return &Config{
		BlockchainSettings: cfg,
		ScalaOpts:          &ScalaCustomOptions{Features: genSettings.PreactivatedFeatures, EnableMining: false},
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

func calculateBaseTarget(hit *consensus.Hit, pos consensus.PosCalculator, minBT types.BaseTarget, maxBT types.BaseTarget, balance uint64, averageDelay uint64) (types.BaseTarget, error) {
	if maxBT-minBT <= 1 {
		return maxBT, nil
	}
	newBT := (maxBT + minBT) / 2
	delay, err := pos.CalculateDelay(hit, newBT, balance)
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
	return calculateBaseTarget(hit, pos, min, max, balance, averageDelay)
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

func calcInitialBaseTarget(accounts []AccountInfo, genSettings *GenesisSettings) (types.BaseTarget, error) {
	maxBT := uint64(0)
	pos := getPosCalculator(genSettings)
	for _, info := range accounts {
		hit, err := getHit(info, genSettings)
		if err != nil {
			return 0, err
		}
		bt, err := calculateBaseTarget(hit, pos, consensus.MinBaseTarget, maxBaseTarget, info.Amount, genSettings.AverageBlockDelay)
		if err != nil {
			return 0, err
		}
		if bt > maxBT {
			maxBT = bt
		}
	}
	return maxBT, nil
}

func getHit(acc AccountInfo, genSettings *GenesisSettings) (*consensus.Hit, error) {
	hitSource := make([]byte, crypto.DigestSize)
	var gs []byte
	var err error
	if isFeaturePreactivated(genSettings.PreactivatedFeatures, int16(settings.BlockV5)) {
		proof, err := crypto.SignVRF(acc.SecretKey, hitSource)
		if err != nil {
			return nil, err
		}
		ok, hs, err := crypto.VerifyVRF(acc.PublicKey, hitSource, proof)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, err
		}
		gs = hs
	} else {
		genSigProvider := consensus.NXTGenerationSignatureProvider
		gs, err = genSigProvider.GenerationSignature(acc.PublicKey, hitSource)
		if err != nil {
			return nil, err
		}
	}
	hit, err := consensus.GenHit(gs)
	if err != nil {
		return nil, err
	}
	return hit, nil
}
