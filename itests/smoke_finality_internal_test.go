//go:build smoke

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
	"github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

const period = 4

type SmokeFinalitySuite struct {
	fixtures.BaseSuite
	goMiner    config.AccountInfo
	scalaMiner config.AccountInfo
}

func (s *SmokeFinalitySuite) SetupSuite() {
	s.BaseSetupWithImages("ghcr.io/wavesplatform/waves", "block-time-logging",
		config.WithFeatureSettingFromFile("feature_settings", "finality_supported_setting.json"),
		config.WithGenerationPeriod(period),
	)
	var err error
	s.goMiner, err = s.Cfg.GetAccount(goWalletAddress)
	require.NoError(s.T(), err)
	s.scalaMiner, err = s.Cfg.GetAccount(scalaWalletAddress)
	require.NoError(s.T(), err)
}

func (s *SmokeFinalitySuite) TestFinalization() {
	// Wait for feature activation.
	h := s.Clients.GetMinNodesHeight(s.T())
	activationHeight := utilities.WaitForFeatureActivation(&s.BaseSuite, settings.DeterministicFinality, h)

	// Commit for the second generation period.
	s0 := s.commitToGeneration(activationHeight)

	// Wait for second generation period to start.
	s.Clients.WaitForHeight(s.T(), s0, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(period))

	// Commit for second generation period.
	s1 := s.commitToGeneration(activationHeight)

	// Wait for third generation period to start.
	s.Clients.WaitForHeight(s.T(), s1, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(period))
}

func (s *SmokeFinalitySuite) commitToGeneration(activationHeight uint64) uint64 {
	const txTimeout = time.Second * 30
	gps := s.nextPeriodStart(activationHeight)
	s.T().Logf("Committing for generation period starting at %d", gps)

	goTx := s.commitmentTransaction(s.goMiner, safecast.MustConvert[uint32](gps))
	_, err := s.Clients.BroadcastToGoNode(s.T(), goTx)
	require.NoError(s.T(), err)
	scalaTx := s.commitmentTransaction(s.scalaMiner, safecast.MustConvert[uint32](gps))
	_, err = s.Clients.BroadcastToScalaNode(s.T(), scalaTx)
	require.NoError(s.T(), err)

	goErr, scalaErr := s.Clients.WaitForTransaction(*goTx.ID, txTimeout)
	require.NoError(s.T(), goErr)
	require.NoError(s.T(), scalaErr)
	s.T().Logf("Go miner: Address: %s, BLS PK: %s", s.goMiner.Address, s.goMiner.BLSPublicKey.String())
	s.T().Logf("Go commitment transaction ID: %s", *goTx.ID)

	goErr, scalaErr = s.Clients.WaitForTransaction(*scalaTx.ID, txTimeout)
	require.NoError(s.T(), goErr)
	require.NoError(s.T(), scalaErr)
	s.T().Logf("Scala miner: Address: %s, BLS PK: %s", s.scalaMiner.Address, s.scalaMiner.BLSPublicKey.String())
	s.T().Logf("Scala commitment transaction ID: %s", *scalaTx.ID)
	return gps
}

func (s *SmokeFinalitySuite) nextPeriodStart(activationHeight proto.Height) proto.Height {
	h := s.Clients.WaitForNewHeight(s.T())
	if h < activationHeight {
		s.T().Fatalf("height %d is too low", h)
	}
	res, err := state.NextGenerationPeriodStart(activationHeight, h, period)
	require.NoError(s.T(), err)
	return uint64(res)
}

// commitmentTransaction creates and signs a CommitToGenerationWithProofs transaction.
func (s *SmokeFinalitySuite) commitmentTransaction(
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

func TestSmokeFinalitySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SmokeFinalitySuite))
}
