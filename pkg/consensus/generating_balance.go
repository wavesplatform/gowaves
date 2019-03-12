package consensus

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

const (
	firstDepth  = 50
	secondDepth = 1000
)

func GeneratingBalance(st state.State, height uint64, addr proto.Address) (uint64, error) {
	settings, err := st.BlockchainSettings()
	if err != nil {
		return 0, errors.Errorf("failed to get blockchain settings: %v\n", err)
	}
	depth := uint64(firstDepth)
	if height >= settings.GenerationBalanceDepthFrom50To1000AfterHeight {
		depth = secondDepth
	}
	balance, err := st.EffectiveBalance(addr, height, height-depth)
	if err != nil {
		return 0, errors.Errorf("failed to get effective balance: %v\n", err)
	}
	return balance, nil
}
