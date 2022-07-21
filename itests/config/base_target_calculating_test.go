package config

import (
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"testing"
)

func createKeyPair(seedText string) (crypto.SecretKey, crypto.PublicKey, error) {
	seed := []byte(seedText)
	iv := [4]byte{}
	binary.BigEndian.PutUint32(iv[:], uint32(0))
	s := append(iv[:], seed...)
	h, err := crypto.SecureHash(s)
	if err != nil {
		return crypto.SecretKey{}, crypto.PublicKey{}, err
	}
	return crypto.GenerateKeyPair(h[:])
}

func TestCalculateBaseTarget(t *testing.T) {
	settings := GenesisSettings{
		AverageBlockDelay:    10,
		MinBlockTime:         5000,
		DelayDelta:           0,
		Distributions:        nil,
		PreactivatedFeatures: []int16{8, 15},
	}
	pos := getPosCalculator(&settings)

	tests := []struct {
		seedText string
		balance  uint64

		baseTarget types.BaseTarget
	}{
		{seedText: "node01", balance: 10000000000000, baseTarget: 140631},
		{seedText: "node02", balance: 15000000000000, baseTarget: 203131},
		{seedText: "node03", balance: 25000000000000, baseTarget: 23445},
		{seedText: "node04", balance: 25000000000000, baseTarget: 453129},
		{seedText: "node05", balance: 40000000000000, baseTarget: 41023},
		{seedText: "node06", balance: 45000000000000, baseTarget: 148443},
		{seedText: "node07", balance: 60000000000000, baseTarget: 40046},
		{seedText: "node08", balance: 80000000000000, baseTarget: 85944},
		{seedText: "node09", balance: 100000000000000, baseTarget: 78132},
		{seedText: "node10", balance: 6000000000000000, baseTarget: 436},
	}
	for _, tc := range tests {
		sk, pk, err := createKeyPair(tc.seedText)
		assert.NoError(t, err)
		
		hit, err := getHit(AccountInfo{pk, sk, tc.balance, proto.WavesAddress{}}, &settings)
		assert.NoError(t, err)

		bt, err := calculateBaseTarget(hit, pos, consensus.MinBaseTarget, maxBaseTarget, tc.balance, settings.AverageBlockDelay)
		assert.NoError(t, err)
		assert.Equal(t, bt, tc.baseTarget)
	}
}
