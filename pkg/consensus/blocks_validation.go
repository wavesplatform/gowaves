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

type stateInfoProvider interface {
	BlockchainSettings() (*settings.BlockchainSettings, error)
	HeaderByHeight(height uint64) (*proto.BlockHeader, error)
	EffectiveBalance(addr proto.Address, startHeight, endHeight uint64) (uint64, error)
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
		return &nxtPosCalculator{}, err
	}
	if fair {
		return &fairPosCalculator{}, nil
	}
	return &nxtPosCalculator{}, nil
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
			return errors.Errorf("generator's effective balance is less than required for generation: %d", balance)
		}
	}
	if balance < minimalEffectiveBalanceForGenerator1 {
		return errors.Errorf("generator's effective balance is less than required for generation: %d", balance)
	}
	return nil
}

func (cv *ConsensusValidator) generatingBalance(height uint64, addr proto.Address) (uint64, error) {
	depth := uint64(firstDepth)
	if height >= cv.settings.GenerationBalanceDepthFrom50To1000AfterHeight {
		depth = secondDepth
	}
	bottomLimit := height - depth
	if height < 1+depth {
		bottomLimit = 1
	}
	balance, err := cv.state.EffectiveBalance(addr, bottomLimit, height)
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
	if target > cv.settings.MaxBaseTarget {
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
	expectedTarget, err := pos.calculateBaseTarget(
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
		return errors.New("declared base target does not match calculated base target")
	}
	return nil
}

func (cv *ConsensusValidator) validateGeneratorSignature(height uint64, header *proto.BlockHeader) error {
	last, err := cv.headerByHeight(height)
	if err != nil {
		return errors.Errorf("failed to get last block header: %v\n", err)
	}
	expectedGenSig, err := generatorSignature(last.GenSignature, header.GenPublicKey)
	if err != nil {
		return errors.Errorf("failed to calculate generator signature: %v\n", err)
	}
	if expectedGenSig != header.GenSignature {
		return errors.Errorf("invalid generation signature")
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
	genSig, err := generatorSignature(header.GenSignature, pk)
	if err != nil {
		return 0, err
	}
	hit, err := hit(genSig[:])
	if err != nil {
		return 0, err
	}
	return pos.calculateDelay(hit, parentTarget, effectiveBalance)
}

func (cv *ConsensusValidator) validateBlockDelay(height uint64, header *proto.BlockHeader) error {
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
	currentTime := time.Now().UnixNano() / 1000
	if int64(header.Timestamp)-currentTime > maxTimeDrift {
		return errors.New("block from future error: block's timestamp is too far in the future")
	}
	return nil
}
