package byte_helpers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferWithSig(t *testing.T) {
	require.NotEmpty(t, TransferWithSig.TransactionBytes)
	require.NotEmpty(t, TransferWithSig.Transaction)
	require.NotEmpty(t, TransferWithSig.MessageBytes)
}

func TestTransferWithProofs(t *testing.T) {
	require.NotEmpty(t, TransferWithProofs.TransactionBytes)
	require.NotEmpty(t, TransferWithProofs.Transaction)
	require.NotEmpty(t, TransferWithProofs.MessageBytes)
}

func TestIssueWithSig(t *testing.T) {
	require.NotEmpty(t, IssueWithSig.TransactionBytes)
	require.NotEmpty(t, IssueWithSig.Transaction)
	require.NotEmpty(t, IssueWithSig.MessageBytes)
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

func TestReissueWithSig(t *testing.T) {
	require.NotEmpty(t, ReissueWithSig.TransactionBytes)
	require.NotEmpty(t, ReissueWithSig.Transaction)
	require.NotEmpty(t, ReissueWithSig.MessageBytes)
}

func TestReissueWithProofs(t *testing.T) {
	require.NotEmpty(t, ReissueWithProofs.TransactionBytes)
	require.NotEmpty(t, ReissueWithProofs.Transaction)
	require.NotEmpty(t, ReissueWithProofs.MessageBytes)
}

func TestBurnWithSig(t *testing.T) {
	require.NotEmpty(t, BurnWithSig.TransactionBytes)
	require.NotEmpty(t, BurnWithSig.Transaction)
	require.NotEmpty(t, BurnWithSig.MessageBytes)
}

func TestBurnWithProofs(t *testing.T) {
	require.NotEmpty(t, BurnWithProofs.TransactionBytes)
	require.NotEmpty(t, BurnWithProofs.Transaction)
	require.NotEmpty(t, BurnWithProofs.MessageBytes)
}

func TestMassTransferWithProofs(t *testing.T) {
	require.NotEmpty(t, MassTransferWithProofs.TransactionBytes)
	require.NotEmpty(t, MassTransferWithProofs.Transaction)
	require.NotEmpty(t, MassTransferWithProofs.MessageBytes)
}

func TestExchangeWithSig(t *testing.T) {
	require.NotEmpty(t, ExchangeWithSig.TransactionBytes)
	require.NotEmpty(t, ExchangeWithSig.Transaction)
	require.NotEmpty(t, ExchangeWithSig.MessageBytes)
}

func TestExchangeWithProofs(t *testing.T) {
	require.NotEmpty(t, ExchangeWithProofs.TransactionBytes)
	require.NotEmpty(t, ExchangeWithProofs.Transaction)
	require.NotEmpty(t, ExchangeWithProofs.MessageBytes)
}

func TestSetAssetScriptWithProofs(t *testing.T) {
	require.NotEmpty(t, SetAssetScriptWithProofs.TransactionBytes)
	require.NotEmpty(t, SetAssetScriptWithProofs.Transaction)
	require.NotEmpty(t, SetAssetScriptWithProofs.MessageBytes)
}

func TestInvokeScriptWithProofs(t *testing.T) {
	require.NotEmpty(t, InvokeScriptWithProofs.TransactionBytes)
	require.NotEmpty(t, InvokeScriptWithProofs.Transaction)
	require.NotEmpty(t, InvokeScriptWithProofs.MessageBytes)
}

func TestIssueWithProofs(t *testing.T) {
	require.NotEmpty(t, IssueWithProofs.TransactionBytes)
	require.NotEmpty(t, IssueWithProofs.Transaction)
	require.NotEmpty(t, IssueWithProofs.MessageBytes)
}

func TestLeaseWithSig(t *testing.T) {
	require.NotEmpty(t, LeaseWithSig.TransactionBytes)
	require.NotEmpty(t, LeaseWithSig.Transaction)
	require.NotEmpty(t, LeaseWithSig.MessageBytes)
}

func TestLeaseWithProofs(t *testing.T) {
	require.NotEmpty(t, LeaseWithProofs.TransactionBytes)
	require.NotEmpty(t, LeaseWithProofs.Transaction)
	require.NotEmpty(t, LeaseWithProofs.MessageBytes)
}

func TestLeaseCancelWithSig(t *testing.T) {
	require.NotEmpty(t, LeaseCancelWithSig.TransactionBytes)
	require.NotEmpty(t, LeaseCancelWithSig.Transaction)
	require.NotEmpty(t, LeaseCancelWithSig.MessageBytes)
}

func TestLeaseCancelWithProofs(t *testing.T) {
	require.NotEmpty(t, LeaseCancelWithProofs.TransactionBytes)
	require.NotEmpty(t, LeaseCancelWithProofs.Transaction)
	require.NotEmpty(t, LeaseCancelWithProofs.MessageBytes)
}

func TestDataWithProofs(t *testing.T) {
	require.NotEmpty(t, DataWithProofs.TransactionBytes)
	require.NotEmpty(t, DataWithProofs.Transaction)
	require.NotEmpty(t, DataWithProofs.MessageBytes)
}

func TestSponsorshipWithProofs(t *testing.T) {
	require.NotEmpty(t, SponsorshipWithProofs.TransactionBytes)
	require.NotEmpty(t, SponsorshipWithProofs.Transaction)
	require.NotEmpty(t, SponsorshipWithProofs.MessageBytes)
}

func TestCreateAliasWithSig(t *testing.T) {
	require.NotEmpty(t, CreateAliasWithSig.TransactionBytes)
	require.NotEmpty(t, CreateAliasWithSig.Transaction)
	require.NotEmpty(t, CreateAliasWithSig.MessageBytes)
}

func TestCreateAliasWithProofs(t *testing.T) {
	require.NotEmpty(t, CreateAliasWithProofs.TransactionBytes)
	require.NotEmpty(t, CreateAliasWithProofs.Transaction)
	require.NotEmpty(t, CreateAliasWithProofs.MessageBytes)
}
