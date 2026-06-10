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
	deposit            int64 = 100_0000_0000 // 100 WAVES
	fee                int64 = 1000_0000     // 0.1 WAVES
	goWalletAddress          = "3Jy49E8GFuUQ7urTuVR2md3TmHDZQMLEmAY"
	scalaWalletAddress       = "3JbGqxNqwBfwnCbzLbo4HwjA9NR1wDjrRTr"
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
	miner, err := s.Cfg.GetAccount(goWalletAddress)
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

func (s *IsolatedFinalitySuite) TestDepositRollback() {
	// Get the miner account (the same as in wallet) to create commitment transactions for generation.
	miner, err := s.Cfg.GetAccount(goWalletAddress)
	require.NoError(s.T(), err)
	// Get the richest account to check balance changes, because it's easier to check on balance unaffected by mining.
	acc := s.Cfg.GetRichestAccount()

	// Ensure that the height is at least 2 (to avoid activation block).
	s.Client.WaitForHeight(s.T(), 2, config.WaitWithContext(s.MainCtx))

	// Get initial available balance of the account.
	b0 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	ab0 := b0.GetAvailable()

	s0 := s.nextPeriodStart()

	// Create first commitment transaction and broadcast it.
	tx1 := s.commitmentTransaction(acc, safecast.MustConvert[uint32](s0))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx1)
	require.NoError(s.T(), err)
	s.Client.WaitForTransaction(s.T(), *tx1.ID, config.WaitWithContext(s.MainCtx))
	b1 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	eb1 := ab0 - (deposit + fee) // Initial balance reduced by deposit and transaction fee amounts.
	assert.Equal(s.T(), eb1, b1.GetAvailable())

	// Commit miner for next generation period.
	tx := s.commitmentTransaction(miner, safecast.MustConvert[uint32](s0))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx)
	require.NoError(s.T(), err)

	// Wait for first generation period to start.
	s.Client.WaitForHeight(s.T(), s0, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))

	// Create second commitment transaction and broadcast it.
	s1 := s.nextPeriodStart()
	tx2 := s.commitmentTransaction(acc, safecast.MustConvert[uint32](s1))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx2)
	require.NoError(s.T(), err)
	s.Client.WaitForTransaction(s.T(), *tx2.ID, config.WaitWithContext(s.MainCtx))
	b2 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	eb2 := ab0 - 2*(deposit+fee) // Balance reduced twice by deposit and fee amounts.
	assert.Equal(s.T(), eb2, b2.GetAvailable())

	// Commit miner for the second period also.
	tx = s.commitmentTransaction(miner, safecast.MustConvert[uint32](s1))
	_, err = s.Client.HTTPClient.TransactionBroadcast(tx)
	require.NoError(s.T(), err)

	s.Client.WaitForHeight(s.T(), s1, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))

	// Rollback to the height before the second commitment.
	s.Client.HTTPClient.RollbackToHeight(s.T(), s0-1, false)

	// Check that the second deposit has been rolled-back.
	s.Client.WaitForHeight(s.T(), s0+1, config.WaitWithContext(s.MainCtx))
	b3 := s.Client.GRPCClient.GetWavesBalance(s.T(), acc.Address)
	// Balance reduced only by the first deposit and fee amounts, second deposit and fee should be rolled back.
	eb3 := ab0 - (deposit + fee)
	assert.Equal(s.T(), eb3, b3.GetAvailable())

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
	f, err := safecast.Convert[uint64](fee)
	require.NoError(s.T(), err)
	tx := proto.NewUnsignedCommitToGenerationWithProofs(ver, acc.PublicKey, start, acc.BLSPublicKey, cs, f, ts)
	err = tx.Sign(s.Cfg.BlockchainSettings.AddressSchemeCharacter, acc.SecretKey)
	require.NoError(s.T(), err)
	return tx
}

func TestIsolatedFinalitySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IsolatedFinalitySuite))
}
