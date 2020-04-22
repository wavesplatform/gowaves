package state

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/settings"
)

func validateSettings(s *settings.BlockchainSettings) error {
	if s.BlockRewardTerm == 0 {
		return errors.New("invalid value `0` for settings.BlockRewardTerm, suggest value `100000`")
	}
	return nil
}
