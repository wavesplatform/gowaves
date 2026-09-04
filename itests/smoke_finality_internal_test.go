//go:build smoke

package itests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ccoveille/go-safecast/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/client"
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

	// Commit for third generation period.
	s1 := s.commitToGeneration(activationHeight)

	goFH, scalaFH, equal := s.Clients.FinalityCmp(s.T())
	assert.True(s.T(), equal, fmt.Sprintf("finalized height mismatch: Go finalized at %d, Scala finalized at %d",
		goFH, scalaFH))

	// Wait for third generation period to start.
	s.Clients.WaitForHeight(s.T(), s1, config.WaitWithContext(s.MainCtx), config.WaitWithTimeoutInBlocks(period))
}

// TestConflictingEndorsements verifies that a generator that broadcasts the conflicting endorsement
// is banned from generation by both nodes.
func (s *SmokeFinalitySuite) TestConflictingEndorsements() {
	const txTimeout = time.Second * 30

	// Pick a third account as the violator.
	violator, ok := s.selectViolator()
	require.True(s.T(), ok, "no spare account available to act as the violator")
	s.T().Logf("Violator: Address: %s, BLS PK: %s", violator.Address, violator.BLSPublicKey.String())

	// Wait for the feature activation.
	h := s.Clients.GetMinNodesHeight(s.T())
	activationHeight := utilities.WaitForFeatureActivation(&s.BaseSuite, settings.DeterministicFinality, h)

	// Commit for next generation period from all generators.
	gps := s.commitToGenerationFrom(activationHeight, txTimeout, s.goMiner, s.scalaMiner, violator)

	// Wait for the generation period to start.
	s.Clients.WaitForHeight(s.T(), gps, config.WaitWithContext(s.MainCtx),
		config.WaitWithTimeoutInBlocks(period))

	// Keep micro-blocks filled with transfer transactions.
	stopTransfers := s.startTransferLoop(s.goMiner, s.scalaMiner)

	// Locate the violator's index in the committed generators set.
	violatorIndex := s.findViolatorIndex(violator, gps)
	s.T().Logf("Violator's index in the generators set at height %d: %d", gps, violatorIndex)

	// Build a conflicting block endorsement signed by the violator. To make the endorsement conflict
	// with the local state, the message claims a wrong finalized block ID for the genesis height.
	endorsement := s.buildConflictingEndorsement(violator, violatorIndex)
	s.T().Logf("Sending conflicting endorsement: %s", endorsement.String())

	payload, err := endorsement.Marshal()
	require.NoError(s.T(), err)
	endorseMessage := &proto.EndorseBlockMessage{Bytes: payload}
	s.Clients.SendToGoNode(s.T(), endorseMessage)
	s.Clients.SendToScalaNode(s.T(), endorseMessage)

	// Wait until the end of the current generation period for the conflicting endorsement to be
	// included into a mined block and applied on both nodes.
	checkHeight := gps + period - 1
	s.Clients.WaitForHeight(s.T(), checkHeight, config.WaitWithContext(s.MainCtx),
		config.WaitWithTimeoutInBlocks(period))

	stopTransfers()

	// Verify that the violator is banned on both nodes.
	s.assertViolatorBanned(violator, checkHeight)
}

func (s *SmokeFinalitySuite) startTransferLoop(from, to config.AccountInfo) func() {
	ctx, cancel := context.WithCancel(s.MainCtx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		s.broadcastTransfer(from, to) // Seed the mempool immediately, then continue on the tick.
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.broadcastTransfer(from, to)
			}
		}
	}()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()
			<-done
		})
	}
	s.T().Cleanup(stop)
	return stop
}

func (s *SmokeFinalitySuite) broadcastTransfer(from, to config.AccountInfo) {
	ts, err := safecast.Convert[uint64](time.Now().UnixMilli())
	if err != nil {
		return
	}
	f, err := safecast.Convert[uint64](fee)
	if err != nil {
		return
	}
	tx := proto.NewUnsignedTransferWithProofs(3, from.PublicKey,
		proto.NewOptionalAssetWaves(), proto.NewOptionalAssetWaves(),
		ts, f, f, proto.NewRecipientFromAddress(to.Address), nil)
	if err = tx.Sign(s.Cfg.BlockchainSettings.AddressSchemeCharacter, from.SecretKey); err != nil {
		return
	}
	_, _ = s.Clients.BroadcastToGoNode(s.T(), tx)
}

func (s *SmokeFinalitySuite) selectViolator() (config.AccountInfo, bool) {
	for _, a := range s.Cfg.Accounts {
		addr := a.Address.String()
		if addr == goWalletAddress || addr == scalaWalletAddress {
			continue
		}
		return a, true
	}
	return config.AccountInfo{}, false
}

func (s *SmokeFinalitySuite) commitToGenerationFrom(
	activationHeight uint64, txTimeout time.Duration, accounts ...config.AccountInfo,
) uint64 {
	gps := s.nextPeriodStart(activationHeight)
	s.T().Logf("Committing for generation period starting at %d", gps)

	wg := new(sync.WaitGroup)
	for _, acc := range accounts {
		wg.Go(func() {
			s.T().Logf("Committing account: Address: %s, BLS PK: %s", acc.Address, acc.BLSPublicKey.String())
			tx := s.commitmentTransaction(acc, safecast.MustConvert[uint32](gps))
			_, err := s.Clients.BroadcastToGoNode(s.T(), tx)
			require.NoError(s.T(), err)
			goErr, scalaErr := s.Clients.WaitForTransaction(*tx.ID, txTimeout)
			require.NoError(s.T(), goErr)
			require.NoError(s.T(), scalaErr)
			s.T().Logf("Commitment transaction ID for %s: %s", acc.Address, *tx.ID)
		})
	}
	wg.Wait()
	return gps
}

func (s *SmokeFinalitySuite) findViolatorIndex(violator config.AccountInfo, height uint64) uint32 {
	gens := s.Clients.GoClient.HTTPClient.CommitmentGeneratorsAt(s.T(), height)
	addr := violator.Address.String()
	for i, g := range gens {
		if g.Address == addr {
			return safecast.MustConvert[uint32](i)
		}
	}
	s.T().Fatalf("violator address %s not found in committed generators set at height %d", addr, height)
	return 0
}

func (s *SmokeFinalitySuite) buildConflictingEndorsement(
	violator config.AccountInfo, violatorIndex uint32,
) *proto.BlockEndorsement {
	const forgedFinalizedHeight uint32 = 1 // Genesis height: always ≤ local finalized height.

	topHeader := s.Clients.GoClient.HTTPClient.BlockHeader(
		s.T(), s.Clients.GoClient.HTTPClient.GetHeight(s.T()).Height,
	)
	parentID := topHeader.ID
	// Forge the finalized block ID by using the current top block's ID instead of the genesis ID.
	forgedFinalizedID := parentID

	cryptoMsg := proto.NewEndorsementCryptoMessage(forgedFinalizedID, parentID, forgedFinalizedHeight)
	msgBytes, err := cryptoMsg.Bytes()
	require.NoError(s.T(), err)
	sig, err := bls.Sign(violator.BLSSecretKey, msgBytes)
	require.NoError(s.T(), err)
	return &proto.BlockEndorsement{
		EndorserIndex:        violatorIndex,
		FinalizedBlockID:     forgedFinalizedID,
		FinalizedBlockHeight: forgedFinalizedHeight,
		EndorsedBlockID:      parentID,
		Signature:            sig,
	}
}

func (s *SmokeFinalitySuite) assertViolatorBanned(violator config.AccountInfo, height uint64) {
	addr := violator.Address.String()
	check := func(impl string, gens []client.GeneratorInfoResponse) {
		for _, g := range gens {
			if g.Address != addr {
				continue
			}
			assert.Positive(s.T(), g.ConflictHeight,
				"%s node: violator must have non-zero ConflictHeight (be banned) at height %d", impl, height)
			assert.Zero(s.T(), g.Balance,
				"%s node: violator's generating balance must be reset to 0 after ban at height %d", impl, height)
			return
		}
		s.T().Errorf("%s node: violator address %s not found in generators set at height %d", impl, addr, height)
	}
	check("Go", s.Clients.GoClient.HTTPClient.CommitmentGeneratorsAt(s.T(), height))
	check("Scala", s.Clients.ScalaClient.HTTPClient.CommitmentGeneratorsAt(s.T(), height))
}

func (s *SmokeFinalitySuite) commitToGeneration(activationHeight uint64) uint64 {
	const txTimeout = time.Second * 30
	gps := s.nextPeriodStart(activationHeight)
	s.T().Logf("Committing for generation period starting at %d", gps)

	wg := new(sync.WaitGroup)
	wg.Go(func() {
		s.T().Logf("Go miner: Address: %s, BLS PK: %s", s.goMiner.Address, s.goMiner.BLSPublicKey.String())
		goTx := s.commitmentTransaction(s.goMiner, safecast.MustConvert[uint32](gps))
		_, err := s.Clients.BroadcastToGoNode(s.T(), goTx)
		require.NoError(s.T(), err)
		goErr, scalaErr := s.Clients.WaitForTransaction(*goTx.ID, txTimeout)
		require.NoError(s.T(), goErr)
		require.NoError(s.T(), scalaErr)
		s.T().Logf("Go commitment transaction ID: %s", *goTx.ID)
	})
	wg.Go(func() {
		s.T().Logf("Scala miner: Address: %s, BLS PK: %s", s.scalaMiner.Address, s.scalaMiner.BLSPublicKey.String())
		scalaTx := s.commitmentTransaction(s.scalaMiner, safecast.MustConvert[uint32](gps))
		_, err := s.Clients.BroadcastToScalaNode(s.T(), scalaTx)
		require.NoError(s.T(), err)
		goErr, scalaErr := s.Clients.WaitForTransaction(*scalaTx.ID, txTimeout)
		require.NoError(s.T(), goErr)
		require.NoError(s.T(), scalaErr)
		s.T().Logf("Scala commitment transaction ID: %s", *scalaTx.ID)
	})
	wg.Wait()
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
