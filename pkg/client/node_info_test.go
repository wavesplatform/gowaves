package client

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNodeInfo_Version(t *testing.T) {
	tests := []struct {
		version    string
		statusCode int
	}{
		{"Gowaves v0.10.0-14-g56259d4b", 200},
		{"Waves v1.4.8-6-ga6adcae", 200},
		{"", 404},
		{"", 500},
	}
	for _, tc := range tests {
		cl, err := NewClient(Options{
			Client: NewMockHttpRequestFromString(fmt.Sprintf(`{"version":"%s"}`, tc.version), tc.statusCode),
		})
		require.NoError(t, err)
		actualVersion, resp, err := cl.NodeInfo.Version(context.Background())
		if tc.statusCode == 200 {
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, tc.version, actualVersion)
		} else {
			require.Error(t, err)
			require.NotNil(t, resp)
		}
	}
}
