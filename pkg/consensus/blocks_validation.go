package consensus

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

const (
	firstDepth  = 50
	secondDepth = 1000
)

type ConsensusValidator struct {
	state       state.State
	startHeight uint64
	// Headers to validate.
	headers []*proto.BlockHeader
}

func NewConsensusValidator(state state.State) (*ConsensusValidator, error) {
	return &ConsensusValidator{state: state}, nil
}

func (cv *ConsensusValidator) headerByHeight(height uint64) (*proto.BlockHeader, error) {
	if height < cv.startHeight {
		block, err := cv.state.BlockByHeight(height)
		if err != nil {
			return nil, err
		}
		return &block.BlockHeader, nil
	}
	return cv.headers[height-cv.startHeight], nil
}

func (cv *ConsensusValidator) validateHeaders(headers []*proto.BlockHeader, startHeight uint64) error {
	return nil
}

func (cv *ConsensusValidator) generatingBalance(height uint64, addr proto.Address) (uint64, error) {
	settings, err := cv.state.BlockchainSettings()
	if err != nil {
		return 0, errors.Errorf("failed to get blockchain settings: %v\n", err)
	}
	depth := uint64(firstDepth)
	if height >= settings.GenerationBalanceDepthFrom50To1000AfterHeight {
		depth = secondDepth
	}
	balance, err := cv.state.EffectiveBalance(addr, height, height-depth)
	if err != nil {
		return 0, errors.Errorf("failed to get effective balance: %v\n", err)
	}
	return balance, nil
}

func (cv *ConsensusValidator) minerGeneratingBalance(height uint64, header *proto.BlockHeader) (uint64, error) {
	settings, err := cv.state.BlockchainSettings()
	if err != nil {
		return 0, errors.Errorf("failed to get blockchain settings: %v\n", err)
	}
	minerAddr, err := proto.NewAddressFromPublicKey(settings.AddressSchemeCharacter, header.GenPublicKey)
	if err != nil {
		return 0, err
	}
	return cv.generatingBalance(height, minerAddr)
}

func (cv *ConsensusValidator) validateBlockVersion(height uint64, block *proto.BlockHeader) error {
	return errors.New("not implemented")
}

func (cv *ConsensusValidator) validateBaseTarget(height uint64, block, parent, grandParent *proto.BlockHeader) error {
	return errors.New("not implemented")
}

func (cv *ConsensusValidator) validateGeneratorSignature(height uint64, block *proto.BlockHeader) error {
	return errors.New("not implemented")
}

func (cv *ConsensusValidator) validBlockDelay(height uint64, pk crypto.PublicKey, parentTarget, effectiveBalance uint64) (uint64, error) {
	pos, err := posAlgo(height)
	if err != nil {
		return 0, err
	}
	header, err := cv.headerByHeight(pos.heightForHit(height))
	if err != nil {
		return 0, err
	}
	genSig, err := generatorSignature(header.GenSignature, header.GenPublicKey)
	if err != nil {
		return 0, err
	}
	hit, err := hit(genSig[:])
	if err != nil {
		return 0, err
	}
	return pos.calculateDelay(hit, parentTarget, effectiveBalance)
}

func (cv *ConsensusValidator) validateBlockDelay(height uint64, headerNum int) error {
	header, err := cv.headerByHeight(height)
	if err != nil {
		return errors.Errorf("failed to get header by height: %v\n", err)
	}
	parent, err := cv.headerByHeight(height - 1)
	if err != nil {
		return errors.Errorf("failed to get parent by height: %v\n", err)
	}
	effectiveBalance, err := cv.minerGeneratingBalance(height, header)
	if err != nil {
		return errors.Errorf("failed to get effective balance: %v\n", err)
	}
	delay, err := cv.validBlockDelay(height, header.GenPublicKey, parent.BaseTarget, effectiveBalance)
	if err != nil {
		return errors.Errorf("failed to calculate valid block delay: %v\n", err)
	}
	minTimestamp := parent.Timestamp + delay
	if header.Timestamp <= minTimestamp {
		return errors.New("invalid header timestamp: less than min valid timestamp")
	}
	return nil
}

func (cv *ConsensusValidator) validateBlockTimestamp(block *proto.BlockHeader) error {
	return errors.New("not implemented")
}
