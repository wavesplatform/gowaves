package consensus

import (
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
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

func isInvalidMainNetBlock(blockID crypto.Signature, height uint64) bool {
	if h, ok := mainNetInvalidBlocks[blockID.String()]; ok {
		return h == height
	}
	return false
}

type stateInfoProvider interface {
	BlockchainSettings() (*settings.BlockchainSettings, error)
	HeaderByHeight(height uint64) (*proto.BlockHeader, error)
	EffectiveBalance(addr proto.Recipient, startHeight, endHeight uint64) (uint64, error)
	IsActivated(featureID int16) (bool, error)
}

type ConsensusValidator struct {
	state       stateInfoProvider
	settings    *settings.BlockchainSettings
	startHeight uint64
	// Headers to validate.
	headers []proto.BlockHeader
}

func NewConsensusValidator(state stateInfoProvider) (*ConsensusValidator, error) {
	settings, err := state.BlockchainSettings()
	if err != nil {
		return nil, errors.Errorf("failed to get blockchain settings: %v\n", err)
	}
	return &ConsensusValidator{state: state, settings: settings}, nil
}

func (cv *ConsensusValidator) smallerMinimalGeneratingBalanceActivated() (bool, error) {
	return cv.state.IsActivated(int16(settings.SmallerMinimalGeneratingBalance))
}

func (cv *ConsensusValidator) fairPosActivated() (bool, error) {
	return cv.state.IsActivated(int16(settings.FairPoS))
}

func (cv *ConsensusValidator) posAlgo() (posCalculator, error) {
	fair, err := cv.fairPosActivated()
	if err != nil {
		return &NxtPosCalculator{}, err
	}
	if fair {
		return &FairPosCalculator{}, nil
	}
	return &NxtPosCalculator{}, nil
}

func (cv *ConsensusValidator) headerByHeight(height uint64) (*proto.BlockHeader, error) {
	if height <= cv.startHeight {
		return cv.state.HeaderByHeight(height)
	}
	return &cv.headers[height-cv.startHeight-1], nil
}

func (cv *ConsensusValidator) ValidateHeaders(headers []proto.BlockHeader, startHeight uint64) error {
	cv.startHeight = startHeight
	cv.headers = headers
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
		if err := cv.validateBlockTimestamp(&header); err != nil {
			return errors.Wrap(err, "block timestamp validation failed")
		}
		if err := cv.validateBlockDelay(height, &header); err != nil {
			return errors.Wrap(err, "block delay validation failed")
		}
		if err := cv.validateGeneratorSignature(height, &header); err != nil {
			return errors.Wrap(err, "block generator signature validation failed")
		}
		if err := cv.validateBaseTarget(height, &header, parent, greatGrandParent); err != nil {
			return errors.Wrap(err, "base target validation failed")
		}
		if err := cv.validateBlockVersion(height, &header); err != nil {
			return errors.Wrap(err, "block version validation failed")
		}
	}
	return nil
}

func (cv *ConsensusValidator) validateEffectiveBalance(header *proto.BlockHeader, balance, height uint64) error {
	if header.Timestamp < cv.settings.MinimalGeneratingBalanceCheckAfterTime {
		return nil
	}
	smallerGeneratingBalance, err := cv.smallerMinimalGeneratingBalanceActivated()
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

func (cv *ConsensusValidator) generatingBalance(height uint64, addr proto.Address) (uint64, error) {
	depth := uint64(firstDepth)
	if height >= cv.settings.GenerationBalanceDepthFrom50To1000AfterHeight {
		depth = secondDepth
	}
	bottomLimit := height - depth + 1
	if height < depth {
		bottomLimit = 1
	}
	balance, err := cv.state.EffectiveBalance(proto.NewRecipientFromAddress(addr), bottomLimit, height)
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

func (cv *ConsensusValidator) validateBlockVersion(height uint64, header *proto.BlockHeader) error {
	if header.Version == proto.GenesisBlockVersion || header.Version == proto.PlainBlockVersion {
		return nil
	}
	if height < cv.settings.BlockVersion3AfterHeight {
		return errors.Errorf("block version 3 can only appear after %d height", cv.settings.BlockVersion3AfterHeight)
	}
	return nil
}

func (cv *ConsensusValidator) checkTargetLimit(height, target uint64) error {
	fair, err := cv.fairPosActivated()
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
	pos, err := cv.posAlgo()
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

func (cv *ConsensusValidator) validateGeneratorSignature(height uint64, header *proto.BlockHeader) error {
	last, err := cv.headerByHeight(height)
	if err != nil {
		return errors.Errorf("failed to get last block header: %v\n", err)
	}
	expectedGenSig, err := GeneratorSignature(last.GenSignature, header.GenPublicKey)
	if err != nil {
		return errors.Errorf("failed to calculate generator signature: %v\n", err)
	}
	if expectedGenSig != header.GenSignature {
		return errors.Errorf("invalid generation signature %s, expected %s", header.GenSignature.String(), expectedGenSig.String())
	}
	return nil
}

func (cv *ConsensusValidator) validBlockDelay(height uint64, pk crypto.PublicKey, parentTarget, effectiveBalance uint64) (uint64, error) {
	pos, err := cv.posAlgo()
	if err != nil {
		return 0, err
	}
	header, err := cv.headerByHeight(pos.heightForHit(height))
	if err != nil {
		return 0, err
	}
	genSig, err := GeneratorSignature(header.GenSignature, pk)
	if err != nil {
		return 0, err
	}
	hit, err := GenHit(genSig[:])
	if err != nil {
		return 0, err
	}
	return pos.CalculateDelay(hit, parentTarget, effectiveBalance)
}

func (cv *ConsensusValidator) validateBlockDelay(height uint64, header *proto.BlockHeader) error {
	if cv.settings.Type == settings.MainNet && isInvalidMainNetBlock(header.BlockSignature, height) {
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
	delay, err := cv.validBlockDelay(height, header.GenPublicKey, parent.BaseTarget, effectiveBalance)
	if err != nil {
		return errors.Errorf("failed to calculate valid block delay: %v\n", err)
	}
	minTimestamp := parent.Timestamp + delay
	if header.Timestamp < minTimestamp {
		return errors.Errorf("invalid block timestamp %d: less than min valid timestamp %d", header.Timestamp, minTimestamp)
	}
	return nil
}

func (cv *ConsensusValidator) validateBlockTimestamp(header *proto.BlockHeader) error {
	// Milliseconds.
	currentTimestamp := proto.NewTimestampFromTime(time.Now())
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
