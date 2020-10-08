package consensus

import (
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
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

	minimalEffectiveBalanceForGenerator1 = 1000000000000
	minimalEffectiveBalanceForGenerator2 = 100000000000
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
	BlockchainSettings() (*settings.BlockchainSettings, error)
	HeaderByHeight(height uint64) (*proto.BlockHeader, error)
	HitSourceAtHeight(height uint64) ([]byte, error)
	SaveHitSources(startHeight uint64, hs [][]byte) error
	NewestEffectiveBalance(addr proto.Recipient, startHeight, endHeight uint64) (uint64, error)
	NewestIsActiveAtHeight(featureID int16, height proto.Height) (bool, error)
}

type ConsensusValidator struct {
	state       stateInfoProvider
	settings    *settings.BlockchainSettings
	startHeight uint64
	// Headers to validate.
	headers    []proto.BlockHeader
	hitSources [][]byte
	ntpTime    types.Time
}

func NewConsensusValidator(state stateInfoProvider, tm types.Time) (*ConsensusValidator, error) {
	s, err := state.BlockchainSettings()
	if err != nil {
		return nil, errors.Errorf("failed to get blockchain settings: %v\n", err)
	}
	return &ConsensusValidator{
		state:    state,
		settings: s,
		ntpTime:  tm,
	}, nil

}

func (cv *ConsensusValidator) smallerMinimalGeneratingBalanceActivated(height uint64) (bool, error) {
	return cv.state.NewestIsActiveAtHeight(int16(settings.SmallerMinimalGeneratingBalance), height)
}

func (cv *ConsensusValidator) fairPosActivated(height uint64) (bool, error) {
	return cv.state.NewestIsActiveAtHeight(int16(settings.FairPoS), height)
}

func (cv *ConsensusValidator) blockV5Activated(height uint64) (bool, error) {
	return cv.state.NewestIsActiveAtHeight(int16(settings.BlockV5), height)
}

func (cv *ConsensusValidator) posAlgo(height uint64) (PosCalculator, error) {
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
			return &FairPosCalculatorV2{}, nil
		}
		return &FairPosCalculatorV1{}, nil
	}
	return &NxtPosCalculator{}, nil
}

func (cv *ConsensusValidator) generationSignatureProvider(height uint64) (GenerationSignatureProvider, error) {
	vrf, err := cv.state.NewestIsActiveAtHeight(int16(settings.BlockV5), height)
	if err != nil {
		return nil, err
	}
	if vrf {
		return &VRFGenerationSignatureProvider{}, nil
	}
	return &NXTGenerationSignatureProvider{}, nil
}

func (cv *ConsensusValidator) headerByHeight(height uint64) (*proto.BlockHeader, error) {
	if height <= cv.startHeight {
		return cv.state.HeaderByHeight(height)
	}
	return &cv.headers[height-cv.startHeight-1], nil
}

func (cv *ConsensusValidator) hitSourceByHeight(height uint64) ([]byte, error) {
	if height <= cv.startHeight {
		return cv.state.HitSourceAtHeight(height)
	}
	return cv.hitSources[height-cv.startHeight-1], nil
}

func (cv *ConsensusValidator) ValidateHeaders(headers []proto.BlockHeader, startHeight uint64) error {
	cv.startHeight = startHeight
	cv.headers = headers
	cv.hitSources = make([][]byte, 0, len(headers))
	for i, header := range headers {
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
		if err := cv.validateGeneratorSignatureAndBlockDelay(height, &header); err != nil {
			return errors.Wrap(err, "block generator signature validation failed")
		}
		if err := cv.validateBlockTimestamp(&header); err != nil {
			return errors.Wrap(err, "block timestamp validation failed")
		}
		if err := cv.validateBaseTarget(height, &header, parent, greatGrandParent); err != nil {
			return errors.Wrapf(err, "base target validation failed at %d", height)
		}
		if err := cv.validateBlockVersion(&header, height); err != nil {
			return errors.Wrap(err, "block version validation failed")
		}
	}
	if err := cv.state.SaveHitSources(cv.startHeight, cv.hitSources); err != nil {
		return errors.Wrap(err, "failed to update hit source")
	}
	return nil
}

func (cv *ConsensusValidator) validateEffectiveBalance(header *proto.BlockHeader, balance, height uint64) error {
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

func (cv *ConsensusValidator) RangeForGeneratingBalanceByHeight(height uint64) (uint64, uint64) {
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

func (cv *ConsensusValidator) generatingBalance(height uint64, addr proto.Address) (uint64, error) {
	start, end := cv.RangeForGeneratingBalanceByHeight(height)
	balance, err := cv.state.NewestEffectiveBalance(proto.NewRecipientFromAddress(addr), start, end)
	if err != nil {
		return 0, err
	}
	return balance, nil
}

func (cv *ConsensusValidator) minerGeneratingBalance(height uint64, header *proto.BlockHeader) (uint64, error) {
	minerAddr, err := proto.NewAddressFromPublicKey(cv.settings.AddressSchemeCharacter, header.GenPublicKey)
	if err != nil {
		return 0, err
	}
	return cv.generatingBalance(height, minerAddr)
}

func (cv *ConsensusValidator) validBlockVersionAtHeight(blockchainHeight uint64) (proto.BlockVersion, error) {
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

func (cv *ConsensusValidator) validateBlockVersion(block *proto.BlockHeader, blockchainHeight uint64) error {
	validVersion, err := cv.validBlockVersionAtHeight(blockchainHeight)
	if err != nil {
		return err
	}
	if block.Version > proto.PlainBlockVersion && blockchainHeight <= cv.settings.BlockVersion3AfterHeight {
		return errors.Errorf("block version 3 or higher can only appear at height greater than %v", cv.settings.BlockVersion3AfterHeight)
	}
	if block.Version < validVersion {
		return errors.Errorf("block version %v is less than valid version %v for height %v", block.Version, validVersion, blockchainHeight)
	}
	return nil
}

func (cv *ConsensusValidator) checkTargetLimit(height, target uint64) error {
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

func (cv *ConsensusValidator) validateBaseTarget(height uint64, header, parent, greatGrandParent *proto.BlockHeader) error {
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

func (cv *ConsensusValidator) validateGeneratorSignatureAndBlockDelay(height uint64, header *proto.BlockHeader) error {
	pos, err := cv.posAlgo(height)
	if err != nil {
		return errors.Wrapf(err, "failed to validate generation signature")
	}

	gsp, err := cv.generationSignatureProvider(height + 1)
	if err != nil {
		return errors.Wrap(err, "failed to get generation signature provider")
	}

	vrf, err := cv.state.NewestIsActiveAtHeight(int16(settings.BlockV5), height+1)
	if err != nil {
		return errors.Wrapf(err, "failed to validate generation signature")
	}

	var hitSource []byte
	if vrf {
		p := pos.HeightForHit(height)
		refGenSig, err := cv.hitSourceByHeight(p)
		if err != nil {
			return errors.Wrap(err, "failed to validate generation signature")
		}
		var ok bool
		ok, hitSource, err = gsp.VerifyGenerationSignature(header.GenPublicKey, refGenSig, header.GenSignature)
		if err != nil {
			return errors.Wrapf(err, "failed to verify generator signature")
		}
		if !ok {
			return errors.Errorf("invalid generation signature '%s' of block '%s' at %d (ref gen-sig '%s'), with vrf",
				header.GenSignature.String(), header.ID.String(), height, base58.Encode(refGenSig))
		}
		cv.hitSources = append(cv.hitSources, hitSource)

	} else {

		refHeader, err := cv.headerByHeight(height)
		if err != nil {
			return errors.Wrap(err, "failed to validate generation signature")
		}
		refGenSig := refHeader.GenSignature
		ok, gs, err := gsp.VerifyGenerationSignature(header.GenPublicKey, refGenSig, header.GenSignature)
		if err != nil {
			return errors.Wrapf(err, "failed to verify generator signature")
		}
		if !ok {
			return errors.Errorf("invalid generation signature '%s' of block '%s' at %d (ref gen-sig '%s'), without vrf",
				header.GenSignature.String(), header.ID.String(), height, base58.Encode(refGenSig))
		}

		cv.hitSources = append(cv.hitSources, gs)

		prevHitSource, err := cv.hitSourceByHeight(pos.HeightForHit(height))
		if err != nil {
			return errors.Wrap(err, "failed to validate generation signature")
		}
		hitSource, err = gsp.HitSource(header.GenPublicKey, prevHitSource)
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

func (cv *ConsensusValidator) validateBlockTimestamp(header *proto.BlockHeader) error {
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
