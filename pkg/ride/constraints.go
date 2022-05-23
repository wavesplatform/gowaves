package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/scripting"
)

const (
	MaxChainInvokeComplexityV3V4 = 4000
	MaxChainInvokeComplexityV5   = 26000
	MaxChainInvokeComplexityV6   = 52000
)

func maxChainInvokeComplexityByVersion(version scripting.LibraryVersion) (int, error) {
	// libV1 and libV2 don't have callables
	switch version {
	case scripting.LibV3, scripting.LibV4:
		return MaxChainInvokeComplexityV3V4, nil
	case scripting.LibV5:
		return MaxChainInvokeComplexityV5, nil
	case scripting.LibV6:
		return MaxChainInvokeComplexityV6, nil
	default:
		return 0, errors.Errorf("unsupported library version %d", version)
	}
}
