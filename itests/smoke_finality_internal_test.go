//go:build smoke

package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/itests/fixtures"
)

type SmokeFinalitySuite struct {
	fixtures.BaseSuite
}

func (s *SmokeFinalitySuite) SetupSuite() {
	s.BaseSetup(
		config.WithFeatureSettingFromFile("feature_settings", "finality_supported_setting.json"),
		config.WithGenerationPeriod(3),
	)
}

func (s *SmokeFinalitySuite) TestFinalization() {
	miner, err := s.Cfg.GetAccount(walletAddress)
	require.NoError(s.T(), err)

	acc := s.Cfg.GetRichestAccount()

	// Ensure that the height is at least 2 (to avoid activation block).
	s.Client.WaitForHeight(s.T(), 2, config.WaitWithContext(s.MainCtx))

	// Get initial available balance of the account.
	b0 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	ab0 := b0.GetAvailable()

	// Create first commitment transaction and broadcast it.
	s0 := s.nextPeriodStart()
	tx1 := s.commitmentTransaction(acc, safecast.MustConvert[uint32](s0))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx1)
	require.NoError(s.T(), err)
	s.Client.WaitForTransaction(s.T(), *tx1.ID, config.WaitWithContext(s.MainCtx))
	b1 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	assert.Equal(s.T(), ab0-deposit-fee, b1.GetAvailable())

	// Commit miner for generation.
	tx := s.commitmentTransaction(miner, safecast.MustConvert[uint32](s0))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx)
	require.NoError(s.T(), err)

	// Wait for second generation period to start.
	s.Client.WaitForHeight(s.T(), s0, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))

	// Create second commitment transaction and broadcast it.
	s1 := s.nextPeriodStart()
	tx2 := s.commitmentTransaction(acc, safecast.MustConvert[uint32](s1))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx2)
	require.NoError(s.T(), err)
	s.Client.WaitForTransaction(s.T(), *tx2.ID, config.WaitWithContext(s.MainCtx))
	b2 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	assert.Equal(s.T(), ab0-deposit-fee-deposit-fee, b2.GetAvailable())

	// Commit miner for generation.
	tx = s.commitmentTransaction(miner, safecast.MustConvert[uint32](s1))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx)
	require.NoError(s.T(), err)

	// Wait for third generation period to start, first deposit should be returned.
	s.Client.WaitForHeight(s.T(), s1, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))
	s2 := s.nextPeriodStart()
	b3 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	assert.Equal(s.T(), ab0-deposit-fee-fee, b3.GetAvailable())

	// Commit miner for generation.
	tx = s.commitmentTransaction(miner, safecast.MustConvert[uint32](s2))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx)
	require.NoError(s.T(), err)

	// Wait for fourth generation period to start, second deposit should be returned.
	s.Client.WaitForHeight(s.T(), s2, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))
	b4 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	assert.Equal(s.T(), ab0-fee-fee, b4.GetAvailable())
}

func TestSmokeFinalitySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SmokeFinalitySuite))
}
