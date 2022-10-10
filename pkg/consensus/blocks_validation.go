package consensus

import (
	"fmt"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	// Depth for generating balance calculation (in number of blocks).
	firstDepth  = 50
	secondDepth = 1000
	// Maximum forward offset (to the future) for block timestamps.
	// In milliseconds.
	maxTimeDrift = 100

	minimalEffectiveBalanceForGenerator1 = uint64(1000000000000)
	minimalEffectiveBalanceForGenerator2 = uint64(100000000000)
)

// Invalid blocks that are already in blockchain.
var mainNetInvalidBlocks = map[string]uint64{
	"2GNCYVy7k3kEPXzz12saMtRDeXFKr8cymVsG8Yxx3sZZ75eHj9csfXnGHuuJe7XawbcwjKdifUrV1uMq4ZNCWPf1": 812608,
	"5uZoDnRKeWZV9Thu2nvJVZ5dBvPB7k2gvpzFD618FMXCbBVBMN2rRyvKBZBhAGnGdgeh2LXEeSr9bJqruJxngsE7": 813207,
}

func isInvalidMainNetBlock(blockID proto.BlockID, height uint64) bool {
	if h, ok := mainNetInvalidBlocks[blockID.String()]; ok {
		return h == height
	}
	return false
}

type stateInfoProvider interface {
	HeaderByHeight(height uint64) (*proto.BlockHeader, error)
	NewestHitSourceAtHeight(height uint64) ([]byte, error)
	NewestEffectiveBalance(addr proto.Recipient, startHeight, endHeight uint64) (uint64, error)
	NewestIsActiveAtHeight(featureID int16, height proto.Height) (bool, error)
	NewestAccountHasScript(addr proto.WavesAddress) (bool, error)
}

type Validator struct {
	state       stateInfoProvider
	settings    *settings.BlockchainSettings
	startHeight uint64
	// Headers to validate.
	headers []proto.BlockHeader
	ntpTime types.Time
}

func NewValidator(state stateInfoProvider, settings *settings.BlockchainSettings, tm types.Time) *Validator {
	return &Validator{
		state:    state,
		settings: settings,
		ntpTime:  tm,
	}
}

func (cv *Validator) smallerMinimalGeneratingBalanceActivated(height uint64) (bool, error) {
	return cv.state.NewestIsActiveAtHeight(int16(settings.SmallerMinimalGeneratingBalance), height)
}

func (cv *Validator) fairPosActivated(height uint64) (bool, error) {
	return cv.state.NewestIsActiveAtHeight(int16(settings.FairPoS), height)
}

func (cv *Validator) blockV5Activated(height uint64) (bool, error) {
	return cv.state.NewestIsActiveAtHeight(int16(settings.BlockV5), height)
}

func (cv *Validator) rideV6Activated(height uint64) (bool, error) {
	return cv.state.NewestIsActiveAtHeight(int16(settings.RideV6), height)
}

func (cv *Validator) posAlgo(height uint64) (PosCalculator, error) {
	fair, err := cv.fairPosActivated(height)
	if err != nil {
		return nil, err
	}
	if fair {
		blockV5, err := cv.blockV5Activated(height)
		if err != nil {
			return nil, err
		}
		if blockV5 {
			return NewFairPosCalculator(cv.settings.DelayDelta, cv.settings.MinBlockTime), nil
		}
		return FairPosCalculatorV1, nil
	}
	return &nxtPosCalculator{}, nil
}

func (cv *Validator) generationSignatureProvider(height uint64) (GenerationSignatureProvider, error) {
	vrf, err := cv.state.NewestIsActiveAtHeight(int16(settings.BlockV5), height)
	if err != nil {
		return nil, err
	}
	if vrf {
		return VRFGenerationSignatureProvider, nil
	}
	return NXTGenerationSignatureProvider, nil
}

func (cv *Validator) headerByHeight(height uint64) (*proto.BlockHeader, error) {
	if height <= cv.startHeight {
		return cv.state.HeaderByHeight(height)
	}
	return &cv.headers[height-cv.startHeight-1], nil
}

func (cv *Validator) RangeForGeneratingBalanceByHeight(height uint64) (uint64, uint64) {
	depth := uint64(firstDepth)
	if height >= cv.settings.GenerationBalanceDepthFrom50To1000AfterHeight {
		depth = secondDepth
	}
	bottomLimit := height - depth + 1
	if height < depth {
		bottomLimit = 1
	}
	return bottomLimit, height
}

func (cv *Validator) GenerateHitSource(height uint64, header proto.BlockHeader) ([]byte, error) {
	hs, _, _, _, err := cv.generateAndCheckNextHitSource(height, &header)
	if err != nil {
		return nil, err
	}
	return hs, nil
}

func (cv *Validator) ValidateHeaderBeforeBlockApplying(newestHeader *proto.BlockHeader, height proto.Height) error {
	if err := cv.validateMinerAccount(newestHeader, height); err != nil {
		return errors.Wrap(err, "miner account validation failed")
	}
	return nil
}

func (cv *Validator) ValidateHeadersBatch(headers []proto.BlockHeader, startHeight proto.Height) error {
	cv.startHeight = startHeight
	cv.headers = headers
	for i := range headers {
		header := &headers[i] // prevent implicit memory aliasing in for loop

		height := startHeight + uint64(i)
		parent, err := cv.headerByHeight(height)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve block's parent")
		}
		var greatGrandParent *proto.BlockHeader
		if height > 2 {
			greatGrandParent, err = cv.headerByHeight(height - 2)
			if err != nil {
				return errors.Wrap(err, "failed to retrieve block's great grandparent")
			}
		}
		if err := cv.validateGeneratorSignatureAndBlockDelay(height, header); err != nil {
			return errors.Wrapf(err, "generator signature validation failed for block '%s'", header.ID.String())
		}
		if err := cv.validateBlockTimestamp(header); err != nil {
			return errors.Wrapf(err, "timestamp validation failed for block '%s'", header.ID.String())
		}
		if err := cv.validateBaseTarget(height, header, parent, greatGrandParent); err != nil {
			return errors.Wrapf(err, "base target validation failed at height %d for block '%s'", height, header.ID.String())
		}
		if err := cv.validateBlockVersion(header, height); err != nil {
			return errors.Wrapf(err, "version validation failed for block '%s'", header.ID.String())
		}
	}
	return nil
}

func (cv *Validator) validateEffectiveBalance(header *proto.BlockHeader, balance, height uint64) error {
	if header.Timestamp < cv.settings.MinimalGeneratingBalanceCheckAfterTime {
		return nil
	}
	smallerGeneratingBalance, err := cv.smallerMinimalGeneratingBalanceActivated(height)
	if err != nil {
		return err
	}
	if smallerGeneratingBalance {
		if balance < minimalEffectiveBalanceForGenerator2 {
			return errors.Errorf("generator's effective balance is less than required for generation: expected %d, found %d", minimalEffectiveBalanceForGenerator2, balance)
		}
		return nil
	}
	if balance < minimalEffectiveBalanceForGenerator1 {
		return errors.Errorf("generator's effective balance is less than required for generation: expected %d, %d", minimalEffectiveBalanceForGenerator1, balance)
	}
	return nil
}

func (cv *Validator) generatingBalance(height uint64, addr proto.WavesAddress) (uint64, error) {
	start, end := cv.RangeForGeneratingBalanceByHeight(height)
	balance, err := cv.state.NewestEffectiveBalance(proto.NewRecipientFromAddress(addr), start, end)
	if err != nil {
		return 0, err
	}
	return balance, nil
}

func (cv *Validator) minerGeneratingBalance(height uint64, header *proto.BlockHeader) (uint64, error) {
	minerAddr, err := proto.NewAddressFromPublicKey(cv.settings.AddressSchemeCharacter, header.GeneratorPublicKey)
	if err != nil {
		return 0, err
	}
	return cv.generatingBalance(height, minerAddr)
}

func (cv *Validator) validBlockVersionAtHeight(blockchainHeight uint64) (proto.BlockVersion, error) {
	blockRewardActivated, err := cv.state.NewestIsActiveAtHeight(int16(settings.BlockReward), blockchainHeight)
	if err != nil {
		return proto.GenesisBlockVersion, errors.Wrap(err, "IsActiveAtHeight failed")
	}
	blockHeight := blockchainHeight + 1
	blockV5Activated, err := cv.state.NewestIsActiveAtHeight(int16(settings.BlockV5), blockHeight)
	if err != nil {
		return proto.GenesisBlockVersion, errors.Wrap(err, "IsActiveAtHeight failed")
	}
	if blockV5Activated {
		return proto.ProtobufBlockVersion, nil
	} else if blockRewardActivated {
		return proto.RewardBlockVersion, nil
	} else if blockchainHeight > cv.settings.BlockVersion3AfterHeight {
		return proto.NgBlockVersion, nil
	} else if blockchainHeight > 0 {
		return proto.PlainBlockVersion, nil
	}
	return proto.GenesisBlockVersion, nil
}

func (cv *Validator) validateBlockVersion(block *proto.BlockHeader, blockchainHeight uint64) error {
	validVersion, err := cv.validBlockVersionAtHeight(blockchainHeight)
	if err != nil {
		return err
	}
	if block.Version > proto.PlainBlockVersion && blockchainHeight <= cv.settings.BlockVersion3AfterHeight {
		return errs.NewBlockValidationError(fmt.Sprintf("block version 3 or higher can only appear at height greater than %v", cv.settings.BlockVersion3AfterHeight))
	}
	if block.Version < validVersion {
		return errs.NewBlockValidationError(fmt.Sprintf("block version %v is less than valid version %v for height %v", block.Version, validVersion, blockchainHeight))
	}
	return nil
}

func (cv *Validator) validateMinerAccount(block *proto.BlockHeader, blockchainHeight uint64) error {
	rideV6Activated, err := cv.rideV6Activated(blockchainHeight)
	if err != nil {
		return errors.Wrap(err, "failed to validate miner address")
	}
	minerAddr, err := proto.NewAddressFromPublicKey(cv.settings.AddressSchemeCharacter, block.GeneratorPublicKey)
	if err != nil {
		return errors.Wrapf(err, "faield to get miner address from pub key %q", block.GeneratorPublicKey.String())
	}
	blockMinerHasScript, err := cv.state.NewestAccountHasScript(minerAddr)
	if err != nil {
		return errors.Wrapf(err, "failed to determine miner account has script for addr %q", minerAddr)
	}
	if !rideV6Activated && blockMinerHasScript {
		return errors.New("mining with scripted account isn't allowed before feature 17 (RideV6) activation")
	}
	return nil
}

func (cv *Validator) checkTargetLimit(height, target uint64) error {
	fair, err := cv.fairPosActivated(height)
	if err != nil {
		return err
	}
	if !fair {
		return nil
	}
	if target >= cv.settings.MaxBaseTarget {
		return errors.New("base target is greater than maximum value from blockchain settings")
	}
	return nil
}

func (cv *Validator) validateBaseTarget(height uint64, header, parent, greatGrandParent *proto.BlockHeader) error {
	if err := cv.checkTargetLimit(height, header.BaseTarget); err != nil {
		return err
	}
	pos, err := cv.posAlgo(height)
	if err != nil {
		return err
	}
	greatGrandParentTimestamp := uint64(0)
	if greatGrandParent != nil {
		greatGrandParentTimestamp = greatGrandParent.Timestamp
	}
	expectedTarget, err := pos.CalculateBaseTarget(
		cv.settings.AverageBlockDelaySeconds,
		height,
		parent.BaseTarget,
		parent.Timestamp,
		greatGrandParentTimestamp,
		header.Timestamp,
	)
	if err != nil {
		return err
	}
	if expectedTarget != header.BaseTarget {
		return errors.Errorf("declared base target %d does not match calculated base target %d", header.BaseTarget, expectedTarget)
	}
	return nil
}

func (cv *Validator) generateAndCheckNextHitSource(height uint64, header *proto.BlockHeader) ([]byte, PosCalculator, GenerationSignatureProvider, bool, error) {
	pos, err := cv.posAlgo(height)
	if err != nil {
		return nil, nil, nil, false, errors.Wrapf(err, "failed to generate hit source")
	}
	gsp, err := cv.generationSignatureProvider(height + 1)
	if err != nil {
		return nil, nil, nil, false, errors.Wrap(err, "failed to generate hit source")
	}
	vrf, err := cv.state.NewestIsActiveAtHeight(int16(settings.BlockV5), height+1)
	if err != nil {
		return nil, nil, nil, false, errors.Wrap(err, "failed to generate hit source")
	}
	if vrf {
		refGenSig, err := cv.state.NewestHitSourceAtHeight(pos.HeightForHit(height))
		if err != nil {
			return nil, nil, nil, false, errors.Wrap(err, "failed to generate hit source")
		}
		ok, hs, err := gsp.VerifyGenerationSignature(header.GeneratorPublicKey, refGenSig, header.GenSignature)
		if err != nil {
			return nil, nil, nil, false, errors.Wrap(err, "failed to validate hit source")
		}
		if !ok {
			return nil, nil, nil, false, errors.Errorf("invalid hit source '%s' of block '%s' at height %d (ref gen-sig '%s'), with vrf",
				header.GenSignature.String(), header.ID.String(), height, base58.Encode(refGenSig))
		}
		return hs, pos, gsp, vrf, nil
	} else {
		refGenSig, err := cv.state.NewestHitSourceAtHeight(height)
		if err != nil {
			return nil, nil, nil, false, errors.Wrap(err, "failed to generate hit source")
		}
		ok, hs, err := gsp.VerifyGenerationSignature(header.GeneratorPublicKey, refGenSig, header.GenSignature)
		if err != nil {
			return nil, nil, nil, false, errors.Wrap(err, "failed to validate hit source")
		}
		if !ok {
			return nil, nil, nil, false, errors.Errorf("invalid hit source '%s' of block '%s' at height %d (ref gen-sig '%s'), without vrf",
				header.GenSignature.String(), header.ID.String(), height, base58.Encode(refGenSig))
		}
		return hs, pos, gsp, vrf, nil
	}
}

func (cv *Validator) validateGeneratorSignatureAndBlockDelay(height uint64, header *proto.BlockHeader) error {
	hitSource, pos, gsp, isVRF, err := cv.generateAndCheckNextHitSource(height, header)
	if err != nil {
		return errors.Wrap(err, "failed to validate generation signature")
	}
	if !isVRF {
		prevHitSource, err := cv.state.NewestHitSourceAtHeight(pos.HeightForHit(height))
		if err != nil {
			return errors.Wrap(err, "failed to validate generation signature")
		}
		hitSource, err = gsp.HitSource(header.GeneratorPublicKey, prevHitSource)
		if err != nil {
			return errors.Wrap(err, "failed to validate generation signature")
		}
	}
	if cv.settings.Type == settings.MainNet && isInvalidMainNetBlock(header.BlockID(), height) {
		return nil
	}
	parent, err := cv.headerByHeight(height)
	if err != nil {
		return errors.Errorf("failed to get parent by height: %v\n", err)
	}
	effectiveBalance, err := cv.minerGeneratingBalance(height, header)
	if err != nil {
		return errors.Errorf("failed to get effective balance: %v\n", err)
	}
	if err := cv.validateEffectiveBalance(header, effectiveBalance, height); err != nil {
		return errors.Errorf("invalid generating balance at height %d: %v\n", height, err)
	}
	hit, err := GenHit(hitSource)
	if err != nil {
		return err
	}
	delay, err := pos.CalculateDelay(hit, parent.BaseTarget, effectiveBalance)
	if err != nil {
		return errors.Wrap(err, "failed to calculate valid block delay")
	}
	minTimestamp := parent.Timestamp + delay
	if header.Timestamp < minTimestamp {
		return errors.Errorf("block '%s' at %d: invalid block timestamp %d: less than min valid timestamp %d (hit source %s)",
			header.ID, height, header.Timestamp, minTimestamp, base58.Encode(hitSource))
	}
	return nil
}

func (cv *Validator) validateBlockTimestamp(header *proto.BlockHeader) error {
	currentTimestamp := proto.NewTimestampFromTime(cv.ntpTime.Now())
	if int64(header.Timestamp)-int64(currentTimestamp) > maxTimeDrift {
		return errors.Errorf(
			"block from future error: block's timestamp is too far in the future, current timestamp %d, received %d, maxTimeDrift %d, delta %d",
			currentTimestamp,
			header.Timestamp,
			maxTimeDrift,
			header.Timestamp-currentTimestamp)
	}
	return nil
}
