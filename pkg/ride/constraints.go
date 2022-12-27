package ride

import (
	"math"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

const (
	maxChainInvokeComplexityV3V4   = 4000
	maxChainInvokeComplexityV5     = 26000
	maxChainInvokeComplexityV6     = 52000
	maxAssetVerifierComplexityV1V2 = 2000
	maxAssetVerifierComplexityV3V6 = 4000
	maxVerifierComplexity          = 2000
	unlimitedVerifierComplexity    = math.MaxInt16
)

func MaxChainInvokeComplexityByVersion(version ast.LibraryVersion) (uint32, error) {
	// libV1 and libV2 don't have callables
	switch version {
	case ast.LibV3, ast.LibV4:
		return maxChainInvokeComplexityV3V4, nil
	case ast.LibV5:
		return maxChainInvokeComplexityV5, nil
	case ast.LibV6:
		return maxChainInvokeComplexityV6, nil
	default:
		return 0, errors.Errorf("unsupported library version %d", version)
	}
}

func MaxAssetVerifierComplexity(v ast.LibraryVersion) uint32 {
	if v > ast.LibV2 {
		return maxAssetVerifierComplexityV3V6
	}
	return maxAssetVerifierComplexityV1V2
}

func MaxVerifierComplexity(rideV5Activated bool) uint32 {
	if rideV5Activated {
		return maxVerifierComplexity
	}
	return unlimitedVerifierComplexity
}
