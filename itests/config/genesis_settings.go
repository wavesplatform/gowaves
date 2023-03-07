package config

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"math/big"
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
	s.Scheme = s.SchemeRaw[0]
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
	bt, err := calcInitialBaseTarget(genSettings)
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
	cfg.BlockRewardIncrement = 100000
	cfg.BlockRewardVotingPeriod = 1000
	cfg.InitialBlockReward = 600000000
	cfg.DoubleFeaturesPeriodsAfterHeight = 1000000
	cfg.SponsorshipSingleActivationPeriod = true
	cfg.FeaturesVotingPeriod = 1
	cfg.VotesForFeatureActivation = 1
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
