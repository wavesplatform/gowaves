package metamask

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
)

func TestEthCallSelectors(t *testing.T) {
	tests := []struct {
		selector ethabi.Selector
		expected string
	}{
		{erc20SymbolSelector, "0x95d89b41"},
		{erc20DecimalsSelector, "0x313ce567"},
		{erc20BalanceSelector, "0x70a08231"},
		{erc20SupportsInterfaceSelector, "0x01ffc9a7"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, tc.selector.String())
	}
}
