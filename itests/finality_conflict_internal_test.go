package itests

import (
	"context"
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

const period = 4 // generation period in blocks

// FinalityConflictSuite runs a two-node (Go + Scala) setup where only the Go node mines.
// The Scala node participates as a validator peer but does not generate blocks.
type FinalityConflictSuite struct {
	fixtures.BaseSuite
	goMiner    config.AccountInfo
	scalaMiner config.AccountInfo
}

func (s *FinalityConflictSuite) SetupSuite() {
	s.BaseSetup(
		config.WithFeatureSettingFromFile("feature_settings", "finality_supported_setting.json"),
		config.WithGenerationPeriod(period),
		config.WithNoScalaMining(),
	)
	var err error
	s.goMiner, err = s.Cfg.GetAccount(goWalletAddress)
	require.NoError(s.T(), err)
	s.scalaMiner, err = s.Cfg.GetAccount(scalaWalletAddress)
	require.NoError(s.T(), err)
}

// TestConflictingEndorsements verifies that a generator that broadcasts a conflicting endorsement
// is banned from generation by both nodes when only the Go node is the miner.
func (s *FinalityConflictSuite) TestConflictingEndorsements() {
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

	time.Sleep(5 * time.Second)
	// Build a conflicting endorsement signed by the violator. The message claims a wrong
	// finalized block ID for the genesis height so it conflicts with the local state.
	endorsement := s.buildConflictingEndorsement(violator, violatorIndex)
	s.T().Logf("Sending conflicting endorsement: %s", endorsement.String())

	payload, err := endorsement.Marshal()
	require.NoError(s.T(), err)
	endorseMessage := &proto.EndorseBlockMessage{Bytes: payload}
	s.Clients.SendToGoNode(s.T(), endorseMessage)

	// Wait until the end of the current generation period for the conflicting endorsement
	// to be included in a mined block and applied on both nodes.
	checkHeight := gps + period - 1
	s.Clients.WaitForHeight(s.T(), checkHeight, config.WaitWithContext(s.MainCtx),
		config.WaitWithTimeoutInBlocks(period))

	stopTransfers()

	// Verify that the violator is banned on both nodes.
	s.assertViolatorBanned(violator, checkHeight)
}

func (s *FinalityConflictSuite) selectViolator() (config.AccountInfo, bool) {
	for _, a := range s.Cfg.Accounts {
		addr := a.Address.String()
		if addr == goWalletAddress || addr == scalaWalletAddress {
			continue
		}
		return a, true
	}
	return config.AccountInfo{}, false
}

func (s *FinalityConflictSuite) commitToGenerationFrom(
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

func (s *FinalityConflictSuite) findViolatorIndex(violator config.AccountInfo, height uint64) uint32 {
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

func (s *FinalityConflictSuite) buildConflictingEndorsement(
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

func (s *FinalityConflictSuite) assertViolatorBanned(violator config.AccountInfo, height uint64) {
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

func (s *FinalityConflictSuite) startTransferLoop(from, to config.AccountInfo) func() {
	ctx, cancel := context.WithCancel(s.MainCtx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		s.broadcastTransfer(from, to) // seed the mempool immediately, then continue on the tick
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

func (s *FinalityConflictSuite) broadcastTransfer(from, to config.AccountInfo) {
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

func (s *FinalityConflictSuite) nextPeriodStart(activationHeight proto.Height) proto.Height {
	h := s.Clients.WaitForNewHeight(s.T())
	if h < activationHeight {
		s.T().Fatalf("height %d is too low", h)
	}
	res, err := state.NextGenerationPeriodStart(activationHeight, h, period)
	require.NoError(s.T(), err)
	return uint64(res)
}

func (s *FinalityConflictSuite) commitmentTransaction(
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

func TestFinalityConflictSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(FinalityConflictSuite))
}
