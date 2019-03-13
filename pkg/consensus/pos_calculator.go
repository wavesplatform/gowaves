package consensus

import (
	"math"
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	nxtPosHeightDiffForHit  = 1
	fairPosHeightDiffForHit = 100
)

func generatorSignature(signature crypto.Digest, pk crypto.PublicKey) (crypto.Digest, error) {
	s := make([]byte, crypto.DigestSize*2)
	copy(s[:crypto.DigestSize], signature[:])
	copy(s[crypto.DigestSize:], pk[:])
	return crypto.FastHash(s)
}

func hit(generatorSig []byte) (*big.Int, error) {
	var hit big.Int
	hit.SetBytes(generatorSig)
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
	return 0, errors.New("Not implemented")
}

func posAlgo(height uint64) (posCalculator, error) {
	// TODO: support features concept.
	// Always return Nxt for now, since FairPos appeared later.
	return &nxtPosCalculator{}, nil
}
