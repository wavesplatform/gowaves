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

func TestTransferV2(t *testing.T) {
	require.NotEmpty(t, TransferV2.TransactionBytes)
	require.NotEmpty(t, TransferV2.Transaction)
	require.NotEmpty(t, TransferV2.MessageBytes)
}

func TestIssueV1(t *testing.T) {
	require.NotEmpty(t, IssueV1.TransactionBytes)
	require.NotEmpty(t, IssueV1.Transaction)
	require.NotEmpty(t, IssueV1.MessageBytes)
}

func TestGenesis(t *testing.T) {
	require.NotEmpty(t, Genesis.TransactionBytes)
	require.NotEmpty(t, Genesis.Transaction)
	require.NotEmpty(t, Genesis.MessageBytes)
}

func TestPayment(t *testing.T) {
	require.NotEmpty(t, Payment.TransactionBytes)
	require.NotEmpty(t, Payment.Transaction)
	require.NotEmpty(t, Payment.MessageBytes)
}

func TestReissueV1(t *testing.T) {
	require.NotEmpty(t, ReissueV1.TransactionBytes)
	require.NotEmpty(t, ReissueV1.Transaction)
	require.NotEmpty(t, ReissueV1.MessageBytes)
}

func TestReissueV2(t *testing.T) {
	require.NotEmpty(t, ReissueV2.TransactionBytes)
	require.NotEmpty(t, ReissueV2.Transaction)
	require.NotEmpty(t, ReissueV2.MessageBytes)
}

func TestBurnV1(t *testing.T) {
	require.NotEmpty(t, BurnV1.TransactionBytes)
	require.NotEmpty(t, BurnV1.Transaction)
	require.NotEmpty(t, BurnV1.MessageBytes)
}

func TestBurnV2(t *testing.T) {
	require.NotEmpty(t, BurnV2.TransactionBytes)
	require.NotEmpty(t, BurnV2.Transaction)
	require.NotEmpty(t, BurnV2.MessageBytes)
}

func TestMassTransferV1(t *testing.T) {
	require.NotEmpty(t, MassTransferV1.TransactionBytes)
	require.NotEmpty(t, MassTransferV1.Transaction)
	require.NotEmpty(t, MassTransferV1.MessageBytes)
}

func TestExchangeV1(t *testing.T) {
	require.NotEmpty(t, ExchangeV1.TransactionBytes)
	require.NotEmpty(t, ExchangeV1.Transaction)
	require.NotEmpty(t, ExchangeV1.MessageBytes)
}

func TestExchangeV2(t *testing.T) {
	require.NotEmpty(t, ExchangeV2.TransactionBytes)
	require.NotEmpty(t, ExchangeV2.Transaction)
	require.NotEmpty(t, ExchangeV2.MessageBytes)
}
