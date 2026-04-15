//go:build smoke

package itests

import (
	"testing"
	"time"

	"github.com/ccoveille/go-safecast/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/clients"
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
}

func (s *SmokeFinalitySuite) SetupSuite() {
	s.BaseSetupWithImages("ghcr.io/wavesplatform/waves", "block-time-logging",
		config.WithFeatureSettingFromFile("feature_settings", "finality_supported_setting.json"),
		config.WithGenerationPeriod(period),
	)
}

func (s *SmokeFinalitySuite) TestFinalization() {
	miner, err := s.Cfg.GetAccount(goWalletAddress)
	require.NoError(s.T(), err)

	// Wait for feature activation.
	h := s.Clients.GetMinNodesHeight(s.T())
	activationHeight := utilities.WaitForFeatureActivation(&s.BaseSuite, settings.DeterministicFinality, h)

	// Create first commitment transaction and broadcast it.
	s0 := s.nextPeriodStart(activationHeight)
	tx1 := s.commitmentTransaction(miner, safecast.MustConvert[uint32](s0))
	_, errGo, _, errScala := s.Clients.BroadcastToNodes(s.T(), tx1, []clients.Implementation{clients.NodeGo, clients.NodeScala})
	require.NoError(s.T(), errGo)
	require.NoError(s.T(), errScala)

	goErr, scalaErr := s.Clients.WaitForTransaction(*tx1.ID, 30*time.Second)
	require.NoError(s.T(), goErr)
	require.NoError(s.T(), scalaErr)

	// Wait for second generation period to start.
	s.Clients.WaitForHeight(s.T(), s0, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))

	// Create second commitment transaction and broadcast it.
	s1 := s.nextPeriodStart(activationHeight)
	tx2 := s.commitmentTransaction(miner, safecast.MustConvert[uint32](s1))
	_, err = s.Clients.BroadcastToScalaNode(s.T(), tx2)
	require.NoError(s.T(), err)

	// Wait for third generation period to start.
	s.Clients.WaitForHeight(s.T(), s1, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(4))
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
