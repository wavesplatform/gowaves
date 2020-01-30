package consensus

import (
	"bytes"
	"math"
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	nxtPosHeightDiffForHit  = 0
	fairPosHeightDiffForHit = 100
	hitSize                 = 8
	minBaseTarget           = 9

	// Nxt values.
	minBlockDelaySeconds = 53
	maxBlockDelaySeconds = 67
	baseTargetGamma      = 64
	meanCalculationDepth = 3
	// Fair PoS values.
	c1   = float64(70000)
	c2   = float64(500000000000000000)
	tMin = float64(5000)
)

type Hit = big.Int

var (
	maxSignature = bytes.Repeat([]byte{0xff}, hitSize)
)

func normalize(value, targetBlockDelaySeconds uint64) float64 {
	return float64(value*targetBlockDelaySeconds) / 60
}

func normalizeBaseTarget(baseTarget, targetBlockDelaySeconds uint64) uint64 {
	maxBaseTarget := math.MaxUint64 / targetBlockDelaySeconds
	if baseTarget <= minBaseTarget {
		return minBaseTarget
	}
	if baseTarget >= maxBaseTarget {
		return maxBaseTarget
	}
	return baseTarget
}

type GenerationSignatureProvider interface {
	// GenerationSignatureAndHitSource calculates generation signature and hit source bytes from given message using secret or public key of block's generator.
	GenerationSignatureAndHitSource(key [crypto.KeySize]byte, msg []byte) ([]byte, []byte, error)

	// VerifyGenerationSignature checks that generation signature is valid for given message and generator's public key.
	// It returns verification result and error if any.
	VerifyGenerationSignature(pk crypto.PublicKey, msg, sig []byte) (bool, error)
}

// NXTGenerationSignatureProvider implements the original NXT way to create generation signature using generator's
// public key and generation signature from the previous block.
type NXTGenerationSignatureProvider struct {
}

// Only generator's public key is used then building NXT generation signature.
func (p *NXTGenerationSignatureProvider) GenerationSignatureAndHitSource(key [crypto.KeySize]byte, msg []byte) ([]byte, []byte, error) {
	if len(msg) < crypto.DigestSize {
		return nil, nil, errors.New("invalid message length")
	}
	s := make([]byte, crypto.DigestSize*2)
	copy(s[:crypto.DigestSize], msg[:crypto.DigestSize])
	copy(s[crypto.DigestSize:], key[:])
	d, err := crypto.FastHash(s)
	if err != nil {
		return nil, nil, errors.Wrap(err, "NXT generation signature provider")
	}
	return d[:], d[:], nil
}

func (p *NXTGenerationSignatureProvider) VerifyGenerationSignature(pk crypto.PublicKey, msg, sig []byte) (bool, error) {
	calculated, _, err := p.GenerationSignatureAndHitSource(pk, msg)
	if err != nil {
		return false, errors.Wrap(err, "NXT generation signature provider")
	}
	if !bytes.Equal(sig, calculated) {
		return false, nil
	}
	return true, nil
}

// VRFGenerationSignatureProvider implements generation of VRF pseudo-random value calculated from generation signature
// of previous block and generator's secret key.
type VRFGenerationSignatureProvider struct {
}

func (p *VRFGenerationSignatureProvider) GenerationSignatureAndHitSource(key [crypto.KeySize]byte, msg []byte) ([]byte, []byte, error) {
	proof, err := crypto.SignVRF(key, msg)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "VRF generation signature provider")
	}
	vrf := crypto.ComputeVRF(key, msg)
	return proof, vrf[:], nil
}

// Verify checks that provided signature is valid against given generator's public key and message.
func (p *VRFGenerationSignatureProvider) VerifyGenerationSignature(pk crypto.PublicKey, msg, sig []byte) (bool, error) {
	ok, _, err := crypto.VerifyVRF(pk, msg[:], sig[:])
	if err != nil {
		return false, errors.Wrap(err, "VRF generation signature provider")
	}
	return ok, nil
}

func GenHit(source []byte) (*Hit, error) {
	s := make([]byte, hitSize)
	copy(s, source[:hitSize])
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	var hit big.Int
	hit.SetBytes(s)
	return &hit, nil
}

type PosCalculator interface {
	HeightForHit(height uint64) uint64
	CalculateBaseTarget(
		targetBlockDelaySeconds uint64,
		prevHeight uint64,
		prevTarget uint64,
		parentTimestamp uint64,
		greatGrandParentTimestamp uint64,
		currentTimestamp uint64,
	) (uint64, error)
	CalculateDelay(hit *big.Int, parentTarget, balance uint64) (uint64, error)
}

type NxtPosCalculator struct {
}

func (calc *NxtPosCalculator) HeightForHit(height uint64) uint64 {
	if nxtPosHeightDiffForHit >= height {
		return height
	}
	return height - nxtPosHeightDiffForHit
}

func (calc *NxtPosCalculator) CalculateBaseTarget(
	targetBlockDelaySeconds uint64,
	prevHeight uint64,
	prevTarget uint64,
	parentTimestamp uint64,
	greatGrandParentTimestamp uint64,
	currentTimestamp uint64,
) (types.BaseTarget, error) {
	if prevHeight%2 == 0 {
		meanBlockDelay := (currentTimestamp - parentTimestamp) / 1000
		if greatGrandParentTimestamp > 0 {
			meanBlockDelay = ((currentTimestamp - greatGrandParentTimestamp) / uint64(meanCalculationDepth)) / 1000
		}
		minBlockDelay := normalize(uint64(minBlockDelaySeconds), targetBlockDelaySeconds)
		maxBlockDelay := normalize(uint64(maxBlockDelaySeconds), targetBlockDelaySeconds)
		baseTargetGammaV := normalize(uint64(baseTargetGamma), targetBlockDelaySeconds)
		var baseTargetF float64
		if meanBlockDelay > targetBlockDelaySeconds {
			baseTargetF = float64(prevTarget) * math.Min(float64(meanBlockDelay), maxBlockDelay) / float64(targetBlockDelaySeconds)
		} else {
			baseTargetF = float64(prevTarget) - float64(prevTarget)*baseTargetGammaV*(float64(targetBlockDelaySeconds)-math.Max(float64(meanBlockDelay), minBlockDelay))/float64(targetBlockDelaySeconds*100)
		}
		baseTarget := uint64(baseTargetF)
		target := normalizeBaseTarget(baseTarget, targetBlockDelaySeconds)
		return target, nil
	} else {
		return prevTarget, nil
	}
}

func (calc *NxtPosCalculator) CalculateDelay(hit *Hit, parentTarget types.BaseTarget, balance uint64) (uint64, error) {
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
	delay := uint64(math.Ceil(ratio)) * 1000
	return delay, nil
}

type FairPosCalculator struct {
}

func (calc *FairPosCalculator) HeightForHit(height uint64) uint64 {
	if fairPosHeightDiffForHit >= height {
		return height
	}
	return height - fairPosHeightDiffForHit
}

func (calc *FairPosCalculator) CalculateBaseTarget(
	targetBlockDelaySeconds uint64,
	confirmedHeight uint64,
	confirmedTarget uint64,
	confirmedTimestamp uint64,
	greatGrandParentTimestamp uint64,
	applyingBlockTimestamp uint64,
) (types.BaseTarget, error) {
	maxDelay := normalize(90, targetBlockDelaySeconds)
	minDelay := normalize(30, targetBlockDelaySeconds)
	if greatGrandParentTimestamp == 0 {
		return confirmedTarget, nil
	}
	average := (applyingBlockTimestamp - greatGrandParentTimestamp) / 3 / 1000
	if float64(average) > maxDelay {
		return (confirmedTarget + uint64(math.Max(1, float64(confirmedTarget/100)))), nil
	} else if float64(average) < minDelay {
		return (confirmedTarget - uint64(math.Max(1, float64(confirmedTarget/100)))), nil
	} else {
		return confirmedTarget, nil
	}
}

func (calc *FairPosCalculator) CalculateDelay(hit *Hit, confirmedTarget types.BaseTarget, balance uint64) (uint64, error) {
	var maxHit big.Int
	maxHit.SetBytes(maxSignature)
	var maxHitFloat big.Float
	maxHitFloat.SetInt(&maxHit)
	var hitFloat big.Float
	hitFloat.SetInt(hit)
	var quo big.Float
	quo.Quo(&hitFloat, &maxHitFloat)
	h, _ := quo.Float64()
	log := math.Log(1 - c2*math.Log(h)/float64(confirmedTarget)/float64(balance))
	res := uint64(tMin + c1*log)
	return res, nil
}
