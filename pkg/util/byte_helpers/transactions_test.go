package byte_helpers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferV1(t *testing.T) {
	require.NotEmpty(t, TransferV1.TransactionBytes)
	require.NotEmpty(t, TransferV1.Transaction)
	require.NotEmpty(t, TransferV1.MessageBytes)
}

func TestIssueV1(t *testing.T) {
	require.NotEmpty(t, IssueV1.TransactionBytes)
	require.NotEmpty(t, IssueV1.Transaction)
	require.NotEmpty(t, IssueV1.MessageBytes)
}
