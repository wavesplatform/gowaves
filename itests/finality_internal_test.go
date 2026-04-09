package itests

import (
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
	acc := s.Cfg.GetRichestAccount()

	// Ensure that the height is at least 2 (to avoid activation block).
	s.Client.WaitForHeight(s.T(), 2, config.WaitWithContext(s.MainCtx))

	// Get initial available balance of the account.
	b0 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	ab0 := b0.GetAvailable()

	// Create first commitment transaction and broadcast it.
	s0 := s.nextPeriodStart()
	s.T().Logf("s0=%d", s0)
	tx1 := s.commitmentTransaction(acc, safecast.MustConvert[uint32](s0))
	_, err := s.Client.HTTPClient.TransactionBroadcast(tx1)
	require.NoError(s.T(), err)
	s.Client.WaitForTransaction(s.T(), *tx1.ID, config.WaitWithContext(s.MainCtx))
	b1 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	assert.Equal(s.T(), ab0-deposit-fee, b1.GetAvailable())

	// Wait for second generation period to start.
	s.Client.WaitForHeight(s.T(), s0, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))

	// Create second commitment transaction and broadcast it.
	s1 := s.nextPeriodStart()
	s.T().Logf("s1=%d", s1)
	tx2 := s.commitmentTransaction(acc, safecast.MustConvert[uint32](s1))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx2)
	require.NoError(s.T(), err)
	s.Client.WaitForTransaction(s.T(), *tx2.ID, config.WaitWithContext(s.MainCtx))
	b2 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	assert.Equal(s.T(), ab0-deposit-fee-deposit-fee, b2.GetAvailable())

	// Wait for third generation period to start, first deposit should be returned.
	s.Client.WaitForHeight(s.T(), s1, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))
	s2 := s.nextPeriodStart()
	s.T().Logf("s2=%d", s2)
	b3 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	assert.Equal(s.T(), ab0-deposit-fee-fee, b3.GetAvailable())

	// Wait for fourth generation period to start, second deposit should be returned.
	s.Client.WaitForHeight(s.T(), s2, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))
	b4 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	assert.Equal(s.T(), ab0-fee-fee, b4.GetAvailable())
}

func (s *IsolatedFinalitySuite) TestDepositRollback() {
	acc := s.Cfg.GetRichestAccount()

	// Ensure that the height is at least 2 (to avoid activation block).
	s.Client.WaitForHeight(s.T(), 2, config.WaitWithContext(s.MainCtx))

	// Get initial available balance of the account.
	b0 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	ab0 := b0.GetAvailable()

	s0 := s.nextPeriodStart()
	s.T().Logf("s0=%d", s0)

	// Create first commitment transaction and broadcast it.
	tx1 := s.commitmentTransaction(acc, safecast.MustConvert[uint32](s0))
	_, err := s.Client.HTTPClient.TransactionBroadcast(tx1)
	require.NoError(s.T(), err)
	s.Client.WaitForTransaction(s.T(), *tx1.ID, config.WaitWithContext(s.MainCtx))
	b1 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	assert.Equal(s.T(), ab0-deposit-fee, b1.GetAvailable())

	// Wait for first generation period to start.
	s.Client.WaitForHeight(s.T(), s0, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))

	// Create second commitment transaction and broadcast it.
	s1 := s.nextPeriodStart()
	tx2 := s.commitmentTransaction(acc, safecast.MustConvert[uint32](s1))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx2)
	require.NoError(s.T(), err)
	s.Client.WaitForTransaction(s.T(), *tx2.ID, config.WaitWithContext(s.MainCtx))
	b2 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	assert.Equal(s.T(), ab0-deposit-fee-deposit-fee, b2.GetAvailable())

	s.Client.WaitForHeight(s.T(), s1, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))

	// Rollback to the height before the second commitment.
	s.Client.HTTPClient.RollbackToHeight(s.T(), s0-1, false)

	// Check that the second deposit has been rolled-back.
	s.Client.WaitForHeight(s.T(), s0+1, config.WaitWithContext(s.MainCtx))
	b3 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	assert.Equal(s.T(), ab0-deposit-fee, b3.GetAvailable())

	// Wait for the first deposit to be returned, we need no active deposits before proceeding with the other tests.
	s.Client.WaitForHeight(s.T(), s1, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))
}

func (s *IsolatedFinalitySuite) nextPeriodStart() proto.Height {
	const (
		base   = 2
		period = 3
	)
	h := s.Client.HTTPClient.GetHeight(s.T()).Height
	if h < base {
		s.T().Fatalf("height %d is too low", h)
	}
	k := (h - base) / period
	return base + (k+1)*period
}

// commitmentTransaction creates and signs a CommitToGenerationWithProofs transaction.
func (s *IsolatedFinalitySuite) commitmentTransaction(
	acc config.AccountInfo, start uint32,
) *proto.CommitToGenerationWithProofs {
	const ver = 1
	ts, err := safecast.Convert[uint64](time.Now().UnixMilli())
	require.NoError(s.T(), err)

	_, cs, err := bls.ProvePoP(acc.BLSSecretKey, acc.BLSPublicKey, start)
	require.NoError(s.T(), err)
	ok, err := bls.VerifyPoP(acc.BLSPublicKey, start, cs)
	require.NoError(s.T(), err)
	assert.True(s.T(), ok)
	tx := proto.NewUnsignedCommitToGenerationWithProofs(ver, acc.PublicKey, start, acc.BLSPublicKey, cs, fee, ts)
	err = tx.Sign(s.Cfg.BlockchainSettings.AddressSchemeCharacter, acc.SecretKey)
	require.NoError(s.T(), err)
	return tx
}

func TestIsolatedFinalitySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IsolatedFinalitySuite))
}
