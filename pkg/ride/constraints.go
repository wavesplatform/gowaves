package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

const (
	MaxChainInvokeComplexityV3V4 = 4000
	MaxChainInvokeComplexityV5   = 26000
	MaxChainInvokeComplexityV6   = 52000
)

func maxChainInvokeComplexityByVersion(version ast.LibraryVersion) (int, error) {
	// libV1 and libV2 don't have callables
	switch version {
	case ast.LibV3, ast.LibV4:
		return MaxChainInvokeComplexityV3V4, nil
	case ast.LibV5:
		return MaxChainInvokeComplexityV5, nil
	case ast.LibV6:
		return MaxChainInvokeComplexityV6, nil
	default:
		return 0, errors.Errorf("unsupported library version %d", version)
	}
}
