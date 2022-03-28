package ride

import "github.com/pkg/errors"

const (
	MaxChainInvokeComplexityV3V4 = 4000
	MaxChainInvokeComplexityV5   = 26000
	MaxChainInvokeComplexityV6   = 52000
)

func maxChainInvokeComplexityByVersion(version libraryVersion) (int, error) {
	// libV1 and libV2 don't have callables
	switch version {
	case libV3, libV4:
		return MaxChainInvokeComplexityV3V4, nil
	case libV5:
		return MaxChainInvokeComplexityV5, nil
	case libV6:
		return MaxChainInvokeComplexityV6, nil
	default:
		return 0, errors.Errorf("unsupported library version %d", version)
	}
}
