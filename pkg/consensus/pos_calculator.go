package consensus

import (
	"bytes"
	"math"
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	nxtPosHeightDiffForHit  = 1
	fairPosHeightDiffForHit = 100
	hitSize                 = 8
)

var (
	// Fair PoS values.
	maxSignature = bytes.Repeat([]byte{0xff}, hitSize)
	c1           = float64(70000)
	c2           = float64(0x5E17)
	tMin         = float64(5000)
)

func generatorSignature(signature crypto.Digest, pk crypto.PublicKey) (crypto.Digest, error) {
	s := make([]byte, crypto.DigestSize*2)
	copy(s[:crypto.DigestSize], signature[:])
	copy(s[crypto.DigestSize:], pk[:])
	return crypto.FastHash(s)
}

func hit(generatorSig []byte) (*big.Int, error) {
	var hit big.Int
	hit.SetBytes(generatorSig[:hitSize])
	return &hit, nil
}

type posCalculator interface {
	heightForHit(height uint64) uint64
	calculateBaseTarget(
		targetBlockDelaySeconds uint64,
		prevHeight uint64,
		prevTarget uint64,
		parentTimestamp uint64,
		greatGrandParentTimestamp uint64,
		currentTimestamp uint64,
	) (uint64, error)
	calculateDelay(hit *big.Int, parentTarget, balance uint64) (uint64, error)
}

type nxtPosCalculator struct {
}

func (calc *nxtPosCalculator) heightForHit(height uint64) uint64 {
	return height - nxtPosHeightDiffForHit
}

func (calc *nxtPosCalculator) calculateBaseTarget(
	targetBlockDelaySeconds uint64,
	prevHeight uint64,
	prevTarget uint64,
	parentTimestamp uint64,
	greatGrandParentTimestamp uint64,
	currentTimestamp uint64,
) (uint64, error) {
	return 0, errors.New("Not implemented")
}

func (calc *nxtPosCalculator) calculateDelay(hit *big.Int, parentTarget, balance uint64) (uint64, error) {
	var targetFloat big.Float
	targetFloat.SetUint64(parentTarget)
	var balanceFloat big.Float
	balanceFloat.SetUint64(balance)
	targetFloat.Mul(&targetFloat, &balanceFloat)
	var hitFloat big.Float
	hitFloat.SetInt(hit)
	var quo big.Float
	quo.Quo(&hitFloat, &targetFloat)
	ratio, _ := quo.Float64()
	return uint64(math.Ceil(ratio)) * 1000, nil
}

type fairPosCalculator struct {
}

func (calc *fairPosCalculator) heightForHit(height uint64) uint64 {
	return height - fairPosHeightDiffForHit
}

func (calc *fairPosCalculator) calculateBaseTarget(
	targetBlockDelaySeconds uint64,
	prevHeight uint64,
	prevTarget uint64,
	parentTimestamp uint64,
	greatGrandParentTimestamp uint64,
	currentTimestamp uint64,
) (uint64, error) {
	return 0, errors.New("Not implemented")
}

func (calc *fairPosCalculator) calculateDelay(hit *big.Int, parentTarget, balance uint64) (uint64, error) {
	var maxHit big.Int
	maxHit.SetBytes(maxSignature)
	var maxHitFloat big.Float
	maxHitFloat.SetInt(&maxHit)
	var hitFloat big.Float
	hitFloat.SetInt(hit)
	var quo big.Float
	quo.Quo(&hitFloat, &maxHitFloat)
	h, _ := quo.Float64()
	parentTargetF := float64(parentTarget)
	balanceF := float64(balance)
	return uint64(tMin + c1*math.Log(1-c2*math.Log(h)/parentTargetF/balanceF)), nil
}

func posAlgo(height uint64) (posCalculator, error) {
	// TODO: support features concept.
	// Always return Nxt for now, since FairPos appeared later.
	return &nxtPosCalculator{}, nil
}
