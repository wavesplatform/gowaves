package itests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ccoveille/go-safecast/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	deposit = 100_0000_0000 // 100 WAVES
	fee     = 1000_0000     // 0.1 WAVES
)

type IsolatedFinalitySuite struct {
	fixtures.SingleGoNodeSuite
}

func (s *IsolatedFinalitySuite) SetupSuite() {
	s.BaseSetup(
		config.WithPreactivatedFeatures([]config.FeatureInfo{{Feature: int16(settings.DeterministicFinality), Height: 1}}),
		config.WithGenerationPeriod(3),
	)
}

func (s *IsolatedFinalitySuite) TestDepositsReset() {
	s.Client.SendStartMessage(s.T())

	acc := s.Cfg.GetRichestAccount()

	// Wait for nodes to start mining
	s.Client.WaitForHeight(s.T(), 2, config.WaitWithContext(s.MainCtx))

	// Get initial available balance of the account.
	b0 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	ab0 := b0.GetAvailable()
	s.T().Logf("Initial available balance: %d", ab0)

	// Create first commitment transaction and broadcast it.
	tx1 := commitmentTransaction(s.T(), s.Cfg.BlockchainSettings.AddressSchemeCharacter, acc, 5)
	js1, err := json.Marshal(tx1)
	require.NoError(s.T(), err)
	s.T().Logf("tx1: %s", string(js1))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx1)
	require.NoError(s.T(), err)
	s.Client.WaitForTransaction(s.T(), *tx1.ID, config.WaitWithContext(s.MainCtx))
	b1 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	ab1 := b1.GetAvailable()
	s.T().Logf("Available balance after first commitment: %d", ab1)
	assert.Equal(s.T(), ab0-deposit-fee, ab1)

	// Wait for second generation period to start.
	s.Client.WaitForHeight(s.T(), 6, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))
	s.Client.HTTPClient.GetHeight(s.T())

	// Create second commitment transaction and broadcast it.
	tx2 := commitmentTransaction(s.T(), s.Cfg.BlockchainSettings.AddressSchemeCharacter, acc, 8)
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx2)
	require.NoError(s.T(), err)
	s.Client.WaitForTransaction(s.T(), *tx2.ID, config.WaitWithContext(s.MainCtx))
	b2 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	ab2 := b2.GetAvailable()
	s.T().Logf("Available balance after second commitment: %d", ab2)
	assert.Equal(s.T(), ab0-deposit-fee-deposit-fee, ab2)

	s.Client.WaitForHeight(s.T(), 9, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))
	s.Client.HTTPClient.GetHeight(s.T())
	b3 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	ab3 := b3.GetAvailable()
	s.T().Logf("Available balance after cancel of first commitment: %d", ab3)
	assert.Equal(s.T(), ab0-deposit-fee-fee, ab3)

	s.Client.WaitForHeight(s.T(), 12, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))
	s.Client.HTTPClient.GetHeight(s.T())
	b4 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	ab4 := b4.GetAvailable()
	s.T().Logf("Available balance after cancel of second commitment: %d", ab4)
	assert.Equal(s.T(), ab0-fee-fee, ab4)
}

func TestIsolatedFinalitySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IsolatedFinalitySuite))
}

// commitmentTransaction creates and signs a CommitToGenerationWithProofs transaction.
func commitmentTransaction(
	t testing.TB, scheme proto.Scheme, acc config.AccountInfo, start uint32,
) *proto.CommitToGenerationWithProofs {
	const ver = 1
	ts, err := safecast.Convert[uint64](time.Now().UnixMilli())
	require.NoError(t, err)

	_, cs, err := bls.ProvePoP(acc.BLSSecretKey, acc.BLSPublicKey, start)
	require.NoError(t, err)
	ok, err := bls.VerifyPoP(acc.BLSPublicKey, start, cs)
	require.NoError(t, err)
	assert.True(t, ok)
	tx := proto.NewUnsignedCommitToGenerationWithProofs(ver, acc.PublicKey, start, acc.BLSPublicKey, cs, fee, ts)
	err = tx.Sign(scheme, acc.SecretKey)
	require.NoError(t, err)
	return tx
}
