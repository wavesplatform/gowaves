package config

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	Nonce    int    `json:"nonce"`
	Amount   uint64 `json:"amount"`
	Miner    bool   `json:"miner"`
}

type GenesisSettings struct {
	Scheme            proto.Scheme
	SchemeRaw         string             `json:"scheme"`
	AverageBlockDelay uint64             `json:"average_block_delay"`
	Distributions     []DistributionItem `json:"distributions"`
}

func parseGenesisSettings() (GenesisSettings, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return GenesisSettings{}, err
	}
	configPath := filepath.Clean(filepath.Join(pwd, configFolder, genesisSettingsFileName))
	f, err := os.Open(configPath)
	if err != nil {
		return GenesisSettings{}, fmt.Errorf("failed to open file: %s", err)
	}
	jsonParser := json.NewDecoder(f)
	s := GenesisSettings{}
	if err := jsonParser.Decode(&s); err != nil {
		return GenesisSettings{}, fmt.Errorf("failed to decode genesis settings: %s", err)
	}
	s.Scheme = s.SchemeRaw[0]
	return s, nil
}

func NewBlockchainConfig() (*settings.BlockchainSettings, []AccountInfo, error) {
	genSettings, err := parseGenesisSettings()
	if err != nil {
		return nil, nil, err
	}
	ts := time.Now().UnixMilli()
	txs, acc, err := makeTransactionAndKeyPairs(genSettings, uint64(ts))
	if err != nil {
		return nil, nil, err
	}
	bt, err := calcInitialBaseTarget(acc, genSettings.AverageBlockDelay)
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
	cfg.PreactivatedFeatures = []int16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	return cfg, acc, nil
}

type AccountInfo struct {
	PublicKey crypto.PublicKey
	SecretKey crypto.SecretKey
	Amount    uint64
	Address   proto.WavesAddress
}

func makeTransactionAndKeyPairs(settings GenesisSettings, timestamp uint64) ([]genesis_generator.GenesisTransactionInfo, []AccountInfo, error) {
	r := make([]genesis_generator.GenesisTransactionInfo, 0, len(settings.Distributions))
	accounts := make([]AccountInfo, 0, len(settings.Distributions))
	for i, dist := range settings.Distributions {
		seed := []byte(dist.SeedText)
		iv := [4]byte{}
		binary.BigEndian.PutUint32(iv[:], uint32(i))
		s := append(iv[:], seed...)
		h, err := crypto.SecureHash(s)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate hash from seed '%s': %s", string(seed), err)
		}
		sk, pk, err := crypto.GenerateKeyPair(h[:])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate keyPair from seed '%s': %s", string(seed), err)
		}
		addr, err := proto.NewAddressFromPublicKey(settings.Scheme, pk)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate address from seed '%s': %s", string(seed), err)
		}
		r = append(r, genesis_generator.GenesisTransactionInfo{Address: addr, Amount: dist.Amount, Timestamp: timestamp})
		accounts = append(accounts, AccountInfo{PublicKey: pk, SecretKey: sk, Amount: dist.Amount, Address: addr})
	}
	return r, accounts, nil
}

func calculateBaseTarget(hit *consensus.Hit, minBT types.BaseTarget, maxBT types.BaseTarget, balance uint64, averageDelay uint64) (types.BaseTarget, error) {
	if maxBT-minBT <= 1 {
		return maxBT, nil
	}
	var newBT = (maxBT + minBT) / 2
	posCalculator := consensus.NxtPosCalculator{}
	delay, err := posCalculator.CalculateDelay(hit, newBT, balance)
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
	return calculateBaseTarget(hit, min, max, balance, averageDelay)
}

func calcInitialBaseTarget(accounts []AccountInfo, averageDelay uint64) (types.BaseTarget, error) {
	maxBT := uint64(0)
	for _, info := range accounts {
		hit, err := getHit(info.PublicKey)
		if err != nil {
			return 0, err
		}
		bt, err := calculateBaseTarget(hit, consensus.MinBaseTarget, 1000000, info.Amount, averageDelay)
		if err != nil {
			return 0, err
		}
		if bt > maxBT {
			maxBT = bt
		}
	}
	return maxBT, nil
}

func getHit(pk crypto.PublicKey) (*consensus.Hit, error) {
	hitSource := make([]byte, crypto.DigestSize)
	genSigProvider := consensus.NXTGenerationSignatureProvider{}
	gs, err := genSigProvider.GenerationSignature(pk, hitSource)
	if err != nil {
		return nil, err
	}
	hit, err := consensus.GenHit(gs)
	if err != nil {
		return nil, err
	}
	return hit, nil
}
