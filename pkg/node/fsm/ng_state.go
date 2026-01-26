package fsm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
	"github.com/wavesplatform/gowaves/pkg/crypto"

	"github.com/ccoveille/go-safecast/v2"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/miner"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

var errNoFinalization = errors.New("no finalization available")

// endorsementID hashes the endorsement payload to generate a stable identifier.
func endorsementID(e *proto.EndorseBlock) (crypto.Digest, error) {
	data, err := e.Marshal()
	if err != nil {
		return crypto.Digest{}, err
	}
	return crypto.FastHash(data)
}

type NGState struct {
	baseInfo    BaseInfo
	blocksCache blockStatesCache
}

func newNGState(baseInfo BaseInfo) State {
	baseInfo.syncPeer.Clear()
	return &NGState{
		baseInfo:    baseInfo,
		blocksCache: newBlockStatesCache(baseInfo.logger, NGStateName),
	}
}

func newNGStateWithCache(baseInfo BaseInfo, cache blockStatesCache) State {
	baseInfo.syncPeer.Clear()
	return &NGState{
		baseInfo:    baseInfo,
		blocksCache: cache,
	}
}

func (a *NGState) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func (a *NGState) String() string {
	return NGStateName
}

func (a *NGState) Transaction(p peer.Peer, t proto.Transaction) (State, Async, error) {
	return tryBroadcastTransaction(a, a.baseInfo, p, t)
}

func (a *NGState) StopMining() (State, Async, error) {
	return newIdleState(a.baseInfo), nil, nil
}

func (a *NGState) Task(task tasks.AsyncTask) (State, Async, error) {
	switch task.TaskType {
	case tasks.Ping:
		return a, nil, nil
	case tasks.AskPeers:
		a.baseInfo.logger.Debug("Requesting peers", "state", a.String())
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case tasks.MineMicro:
		t, ok := task.Data.(tasks.MineMicroTaskData)
		if !ok {
			return a, nil, a.Errorf(errors.Errorf(
				"unexpected type %T, expected 'tasks.MineMicroTaskData'", task.Data))
		}
		return a.mineMicro(t.Block, t.Limits, t.KeyPair, t.Vrf)
	case tasks.SnapshotTimeout:
		return a, nil, nil
	default:
		return a, nil, a.Errorf(errors.Errorf(
			"unexpected internal task '%d' with data '%+v' received by %s State",
			task.TaskType, task.Data, a.String()))
	}
}

func (a *NGState) Score(p peer.Peer, score *proto.Score) (State, Async, error) {
	metrics.Score(score, p.Handshake().NodeName)
	if err := a.baseInfo.peers.UpdateScore(p, score); err != nil {
		return a, nil, a.Errorf(proto.NewInfoMsg(err))
	}
	nodeScore, err := a.baseInfo.storage.CurrentScore()
	if err != nil {
		return a, nil, a.Errorf(err)
	}
	if score.Cmp(nodeScore) == 1 {
		// received score is larger than local score
		return syncWithNewPeer(a, a.baseInfo, p)
	}
	return a, nil, nil
}

func (a *NGState) rollbackToStateFromCache(blockFromCache *proto.Block) error {
	previousBlockID := blockFromCache.Parent
	err := a.baseInfo.storage.RollbackTo(previousBlockID)
	if err != nil {
		return errors.Wrapf(err, "failed to rollback to parent block '%s' of cached block '%s'",
			previousBlockID.String(), blockFromCache.ID.String())
	}
	_, err = a.baseInfo.blocksApplier.Apply(
		a.baseInfo.storage,
		[]*proto.Block{blockFromCache},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to apply cached block %q", blockFromCache.ID.String())
	}
	return nil
}

func (a *NGState) rollbackToStateFromCacheInLightNode(parentID proto.BlockID) error {
	blockFromCache, okB := a.blocksCache.Get(parentID)
	snapshotFromCache, okS := a.blocksCache.GetSnapshot(parentID)
	if !okB && !okS {
		// no blocks in cache
		return nil
	}
	if !okS || !okB {
		if !okS {
			return a.Errorf(errors.Errorf("snapshot for block %s doesn't exist in cache", parentID.String()))
		}
		return a.Errorf(errors.Errorf("block %s doesn't exist in cache", parentID.String()))
	}
	a.baseInfo.logger.Debug("Re-applying block from cache", "state", a.String(),
		"blockID", blockFromCache.ID.String())
	previousBlockID := blockFromCache.Parent
	err := a.baseInfo.storage.RollbackTo(previousBlockID)
	if err != nil {
		return errors.Wrapf(err, "failed to rollback to parent block '%s' of cached block '%s'",
			previousBlockID.String(), blockFromCache.ID.String())
	}
	_, err = a.baseInfo.blocksApplier.ApplyWithSnapshots(
		a.baseInfo.storage,
		[]*proto.Block{blockFromCache},
		[]*proto.BlockSnapshot{snapshotFromCache},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to apply cached block %q", blockFromCache.ID.String())
	}
	return nil
}

func (a *NGState) Block(peer peer.Peer, block *proto.Block) (State, Async, error) {
	a.baseInfo.CancelCleanUTX() // cancel UTX cleaning task if it was scheduled
	ok, err := a.baseInfo.blocksApplier.BlockExists(a.baseInfo.storage, block)
	if err != nil {
		return a, nil, a.Errorf(errors.Wrapf(err, "peer '%s'", peer.ID()))
	}
	if ok {
		return a, nil, a.Errorf(proto.NewInfoMsg(errors.Errorf("Block '%s' already exists", block.BlockID().String())))
	}

	height, errHeight := a.baseInfo.storage.Height()
	if errHeight != nil {
		return a, nil, a.Errorf(errHeight)
	}
	metrics.BlockReceived(block, peer.Handshake().NodeName)

	finalityActivated, errFin := a.baseInfo.storage.IsActiveAtHeight(int16(settings.DeterministicFinality), height+1)
	if errFin != nil {
		return a, nil, a.Errorf(errFin)
	}

	top := a.baseInfo.storage.TopBlock()
	if top.BlockID() != block.Parent { // does block refer to last block
		a.baseInfo.logger.Debug("Key-block has parent which is not the top block", "state", a.String(),
			"blockID", block.ID.String(), "parent", block.Parent.String(), "top", top.ID.String())
		if a.baseInfo.enableLightMode {
			if err = a.rollbackToStateFromCacheInLightNode(block.Parent); err != nil {
				return a, nil, a.Errorf(err)
			}
		} else {
			if blockFromCache, okGet := a.blocksCache.Get(block.Parent); okGet {
				a.baseInfo.logger.Debug("Re-applying block from cache", "state", a.String(),
					"blockID", blockFromCache.ID.String())
				if err = a.rollbackToStateFromCache(blockFromCache); err != nil {
					return a, nil, a.Errorf(err)
				}
			}
		}
	}

	if a.baseInfo.enableLightMode {
		defer func() {
			pe := extension.NewPeerExtension(peer, a.baseInfo.scheme, a.baseInfo.netLogger)
			pe.AskBlockSnapshot(block.BlockID())
		}()
		st, timeoutTask := newWaitSnapshotState(a.baseInfo, block, a.blocksCache)
		return st, tasks.Tasks(timeoutTask), nil
	}
	_, err = a.baseInfo.blocksApplier.Apply(
		a.baseInfo.storage,
		[]*proto.Block{block},
	)
	if err != nil {
		return a, nil, a.Errorf(errors.Wrapf(err, "failed to apply block %s", block.BlockID()))
	}
	metrics.BlockApplied(block, height+1)
	a.baseInfo.endorsements.CleanAll()
	a.blocksCache.Clear()
	a.blocksCache.AddBlockState(block)
	a.baseInfo.scheduler.Reschedule()
	a.baseInfo.actions.SendBlock(block)
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	a.baseInfo.CleanUtx()

	if a.baseInfo.embeddedWallet != nil && finalityActivated {
		pks, sks, walErr := a.baseInfo.embeddedWallet.KeyPairsBLS()
		if walErr != nil {
			return a, nil, a.Errorf(errors.Wrapf(walErr, "failed to generate key pairs for %s", block.BlockID()))
		}
		logErr := a.logNewFinalizationVoting(block, height)
		if logErr != nil {
			return a, nil, a.Errorf(errors.Wrapf(logErr, "failed to log new finalization voting for block %s",
				block.BlockID()))
		}
		endorseErr := a.EndorseParentWithEachKey(pks, sks, block, height)
		if endorseErr != nil {
			return a, nil, a.Errorf(errors.Wrapf(endorseErr, "failed to endorse parent block with available keys"))
		}
	}
	return newNGState(a.baseInfo), nil, nil
}

func (a *NGState) logNewFinalizationVoting(currentBlock *proto.Block, height proto.Height) error {
	activationHeight, actErr := a.baseInfo.storage.ActivationHeight(int16(settings.DeterministicFinality))
	if actErr != nil {
		return a.Errorf(errors.Wrapf(actErr, "failed to get activation height for finality %s",
			currentBlock.BlockID()))
	}
	periodStart, genErr := state.CurrentGenerationPeriodStart(activationHeight, height, a.baseInfo.generationPeriod)
	if genErr != nil {
		return a.Errorf(errors.Wrapf(genErr, "failed to get current generation period, block %s",
			currentBlock.BlockID()))
	}
	commitedGenerators, comgenErr := a.baseInfo.storage.CommittedGenerators(periodStart)
	if comgenErr != nil {
		return a.Errorf(errors.Wrapf(comgenErr, "failed to get committed generators for %s", currentBlock.BlockID()))
	}
	if len(commitedGenerators) > 0 {
		slog.Debug("New finalization voting started",
			"blockID", currentBlock.Parent.String(), "CommitedGeneratorsNumber", len(commitedGenerators))
	}
	return nil
}
func (a *NGState) EndorseParentWithEachKey(
	pks []bls.PublicKey,
	sks []bls.SecretKey,
	block *proto.Block,
	height proto.Height,
) error {
	if len(pks) != len(sks) {
		return a.Errorf(errors.Errorf("pks/sks length mismatch: %d != %d", len(pks), len(sks)))
	}

	activationHeight, err := a.baseInfo.storage.ActivationHeight(int16(settings.DeterministicFinality))
	if err != nil {
		return a.Errorf(errors.Wrapf(err, "failed to get activation height for finality %s", block.BlockID()))
	}

	periodStart, err := state.CurrentGenerationPeriodStart(activationHeight, height, a.baseInfo.generationPeriod)
	if err != nil {
		return a.Errorf(errors.Wrapf(err, "failed to get current generation period, block %s", block.BlockID()))
	}

	endorsers, err := a.baseInfo.storage.NewestCommitedEndorsers(periodStart)
	if err != nil {
		return a.Errorf(errors.Wrap(err, "failed to find committed generators"))
	}

	for i := range pks {
		pk := pks[i]
		sk := sks[i]

		slog.Debug("checking commitment record for my BLS public key", "myPublicKeyBLS",
			pk.String(), "periodStart", periodStart)

		committed, storErr := a.baseInfo.storage.NewestCommitmentExistsByEndorserPK(periodStart, pk)
		if storErr != nil {
			a.logCommittedEndorsers(periodStart, endorsers)
			return a.Errorf(errors.Wrapf(
				storErr,
				"failed to find commitments at block %s for endorsers PK %s",
				block.BlockID(),
				pk.String(),
			))
		}

		if !committed {
			a.logCommitmentMiss(periodStart, endorsers)
			continue
		}
		if endorseErr := a.Endorse(block.Parent, height, pk, sk); endorseErr != nil {
			return a.Errorf(errors.Wrapf(err, "failed to endorse parent block"))
		}
	}
	return nil
}

func (a *NGState) logCommittedEndorsers(periodStart uint32, endorsers []bls.PublicKey) {
	slog.Debug("Committed endorsers for period", "periodStart", periodStart)
	for _, e := range endorsers {
		slog.Debug("committed endorser", "endorser", e.String())
	}
}

func (a *NGState) logCommitmentMiss(periodStart uint32, endorsers []bls.PublicKey) {
	slog.Debug("did not find my BLS public key in the commitment records", "periodStart", periodStart)
	if len(endorsers) == 0 {
		slog.Debug("no BLS public keys in the commitment records", "periodStart", periodStart)
		return
	}
	for _, e := range endorsers {
		slog.Debug("commitment record", "endorserBlsPublicKey", e.String())
	}
}

func (a *NGState) BlockEndorsement(blockEndorsement *proto.EndorseBlock) (State, Async, error) {
	slog.Debug("Received a block endorsement:",
		"EndorserIndex", blockEndorsement.EndorserIndex,
		"FinalizedBlockID", blockEndorsement.FinalizedBlockID,
		"FinalizedBlockHeight", blockEndorsement.FinalizedBlockHeight,
		"EndorsedBlockID", blockEndorsement.EndorsedBlockID,
		"Signature", blockEndorsement.Signature.String())
	id, idErr := endorsementID(blockEndorsement)
	if idErr != nil {
		return a, nil, a.Errorf(errors.Wrap(idErr, "failed to compute endorsement id"))
	}
	if a.baseInfo.endorsementIDsCache.SeenEndorsement(id) {
		slog.Debug("Duplicate block endorsement received, skipping",
			"EndorserIndex", blockEndorsement.EndorserIndex,
			"EndorsedBlockID", blockEndorsement.EndorsedBlockID)
		return a, nil, nil
	}

	top := a.baseInfo.storage.TopBlock()
	if top.Parent != blockEndorsement.EndorsedBlockID {
		err := errors.Errorf("endorsed Block ID '%s' must match the current parent's block ID '%s'",
			blockEndorsement.EndorsedBlockID.String(), top.BlockID().String())
		return a, nil, proto.NewInfoMsg(err)
	}

	activationHeight, actErr := a.baseInfo.storage.ActivationHeight(int16(settings.DeterministicFinality))
	if actErr != nil {
		return a, nil,
			proto.NewInfoMsg(errors.Errorf("failed to get DeterministicFinality activation height, %v", actErr))
	}
	height, heightErr := a.baseInfo.storage.Height()
	if heightErr != nil {
		return a, nil, a.Errorf(errors.Wrapf(heightErr, "failed to find height in storage"))
	}
	periodStart, err := state.CurrentGenerationPeriodStart(activationHeight, height, a.baseInfo.generationPeriod)
	if err != nil {
		return a, nil, a.Errorf(errors.Wrapf(err, "failed to get current generation period"))
	}

	endorserPK, err := a.baseInfo.storage.FindEndorserPKByIndex(periodStart, int(blockEndorsement.EndorserIndex))
	if err != nil {
		return a, nil, a.Errorf(errors.Wrapf(err, "failed to find endorser PK by index"))
	}
	generatorWavesPK, findErr := a.baseInfo.storage.FindGeneratorPKByEndorserPK(periodStart, endorserPK)
	if findErr != nil {
		return a, nil, a.Errorf(errors.Wrapf(err, "failed to find waves generator PK by BLS endorser PK"))
	}
	generatorAddress := proto.MustAddressFromPublicKey(a.baseInfo.scheme, generatorWavesPK)
	generatorRec := proto.NewRecipientFromAddress(generatorAddress)
	balance, err := a.baseInfo.storage.GeneratingBalance(generatorRec, height)
	if err != nil {
		return a, nil, a.Errorf(errors.Wrapf(err, "failed to generate balance for generator address %s", generatorAddress))
	}
	localFinalizedHeight, err := a.baseInfo.storage.LastFinalizedHeight()
	if err != nil {
		return a, nil, a.Errorf(errors.Wrapf(err, "failed to get last finalized height for endorser address"))
	}
	localFinalizedBlockHeader, err := a.baseInfo.storage.LastFinalizedBlock()
	if err != nil {
		return a, nil, a.Errorf(errors.Wrapf(err, "failed to get last finalized block header for endorser address"))
	}
	addErr := a.baseInfo.endorsements.Add(blockEndorsement, endorserPK,
		localFinalizedHeight, localFinalizedBlockHeader.BlockID(), balance)
	if addErr != nil {
		return a, nil, errors.Errorf("failed to add an endorsement, %v", addErr)
	}

	a.baseInfo.endorsementIDsCache.RememberEndorsement(id)
	a.baseInfo.actions.SendEndorseBlock(blockEndorsement) // TODO should we send it out if conflicting?
	slog.Debug("Forwarded a block endorsement:",
		"EndorserIndex", blockEndorsement.EndorserIndex,
		"FinalizedBlockID", blockEndorsement.FinalizedBlockID,
		"FinalizedBlockHeight", blockEndorsement.FinalizedBlockHeight,
		"EndorsedBlockID", blockEndorsement.EndorsedBlockID,
		"Signature", blockEndorsement.Signature.String())
	return newNGState(a.baseInfo), nil, nil
}

func (a *NGState) getPartialFinalization(lastFinalizedHeight proto.Height) (*proto.FinalizationVoting, error) {
	if a.baseInfo.endorsements.Len() == 0 {
		return nil, errNoFinalization
	}
	fin, err := a.baseInfo.endorsements.FormFinalization(lastFinalizedHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize endorsements for microblock: %w", err)
	}
	return &fin, nil
}

func (a *NGState) getBlockFinalization(height proto.Height,
	lastFinalizedHeight proto.Height) (*proto.FinalizationVoting, error) {
	blockFinalization, err := a.tryFinalize(height, lastFinalizedHeight)
	if err != nil {
		if !errors.Is(err, errNoFinalization) {
			return nil, a.Errorf(errors.Wrap(err, "failed to try finalize last block"))
		}
		return nil, errNoFinalization
	}
	return blockFinalization, nil
}

func (a *NGState) tryFinalize(height proto.Height,
	lastFinalizedHeight proto.Height) (*proto.FinalizationVoting, error) {
	// No finalization since nobody endorsed the last block.
	if a.baseInfo.endorsements.Len() == 0 {
		return nil, errNoFinalization
	}

	activationHeight, err := a.baseInfo.storage.ActivationHeight(int16(settings.DeterministicFinality))
	if err != nil {
		return nil, fmt.Errorf("failed to get DeterministicFinality activation height: %w", err)
	}

	ok, err := a.baseInfo.endorsements.Verify()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("endorsement verification failed at height %d", height)
	}

	periodStart, err := state.CurrentGenerationPeriodStart(activationHeight, height, a.baseInfo.generationPeriod)
	if err != nil {
		return nil, err
	}

	allEndorsers := a.baseInfo.endorsements.GetEndorsers()
	endorsersAddresses := make([]proto.WavesAddress, 0, len(allEndorsers))
	for _, endorser := range allEndorsers {
		pk, findErr := a.baseInfo.storage.FindGeneratorPKByEndorserPK(periodStart, endorser)
		if findErr != nil {
			return nil, findErr
		}
		addr := proto.MustAddressFromPublicKey(a.baseInfo.scheme, pk)
		endorsersAddresses = append(endorsersAddresses, addr)
	}

	generators, err := a.baseInfo.storage.CommittedGenerators(periodStart)
	if err != nil {
		return nil, err
	}
	if len(generators) == 0 {
		slog.Debug("No committed generators found for finalization calculation")
	}
	canFinalize, err := a.baseInfo.storage.CalculateVotingFinalization(endorsersAddresses, height, generators)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate finalization voting: %w", err)
	}

	if canFinalize {
		finalization, finErr := a.baseInfo.endorsements.FormFinalization(lastFinalizedHeight)
		if finErr != nil {
			return nil, finErr
		}
		return &finalization, nil
	}
	return nil, errNoFinalization
}

func (a *NGState) MinedBlock(
	block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte,
) (State, Async, error) {
	// Defer rescheduling to the end of the function to ensure that
	// the scheduler is rescheduled even if an error occurs.
	//
	// For example, an error may occur when applying a scheduled block after a rollback.
	// Without this deferred call, the scheduler would not be rescheduled,
	// and the next block would not be generated without an external trigger.
	defer a.baseInfo.scheduler.Reschedule()

	height, heightErr := a.baseInfo.storage.Height()
	if heightErr != nil {
		return a, nil, a.Errorf(heightErr)
	}

	finalityActivated, errFin := a.baseInfo.storage.IsActiveAtHeight(int16(settings.DeterministicFinality), height+1)
	if errFin != nil {
		return a, nil, a.Errorf(errFin)
	}
	if finalityActivated {
		logErr := a.logNewFinalizationVoting(block, height)
		if logErr != nil {
			return a, nil, a.Errorf(errors.Wrapf(logErr, "failed to log new finalization voting for block %s",
				block.BlockID()))
		}
	}

	metrics.BlockMined(block)
	err := a.baseInfo.storage.Map(func(state state.NonThreadSafeState) error {
		var err error
		_, err = a.baseInfo.blocksApplier.Apply(
			state,
			[]*proto.Block{block},
		)
		return err
	})
	if err != nil {
		slog.Warn("Failed to apply generated key block", slog.String("state", a.String()),
			slog.String("blockID", block.ID.String()), logging.Error(err))
		metrics.BlockDeclined(block)
		return a, nil, a.Errorf(err)
	}
	metrics.BlockApplied(block, height+1)
	metrics.Utx(a.baseInfo.utx.Len())
	slog.Info("Generated key block successfully applied to state", "state", a.String(),
		"blockID", block.ID.String())

	a.baseInfo.endorsements.CleanAll()
	a.blocksCache.Clear()
	a.blocksCache.AddBlockState(block)
	a.baseInfo.actions.SendBlock(block)
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	a.baseInfo.CleanUtx()

	return a, tasks.Tasks(tasks.NewMineMicroTask(0, block, limits, keyPair, vrf)), nil
}

func (a *NGState) Endorse(parentBlockID proto.BlockID, height proto.Height,
	endorserPK bls.PublicKey, endorserSK bls.SecretKey) error {
	activationHeight, actErr := a.baseInfo.storage.ActivationHeight(int16(settings.DeterministicFinality))
	if actErr != nil {
		return proto.NewInfoMsg(errors.Errorf("failed to get DeterministicFinality activation height, %v", actErr))
	}
	periodStart, err := state.CurrentGenerationPeriodStart(activationHeight, height, a.baseInfo.generationPeriod)
	if err != nil {
		return err
	}
	endorserIndex, err := a.baseInfo.storage.IndexByEndorserPK(periodStart, endorserPK)
	if err != nil {
		return a.Errorf(errors.Wrap(err, "failed to get endorser index by generator pk"))
	}
	lastFinalizedHeight, err := a.baseInfo.storage.LastFinalizedHeight()
	if err != nil {
		return a.Errorf(errors.Wrap(err, "failed to get last finalized block height"))
	}
	lastFinalizedBlock, err := a.baseInfo.storage.BlockByHeight(lastFinalizedHeight)
	if err != nil {
		return a.Errorf(errors.Wrap(err, "failed to get last finalized block"))
	}
	message, err := proto.EndorsementMessage(
		lastFinalizedBlock.BlockID(),
		parentBlockID,
		lastFinalizedHeight,
	)
	if err != nil {
		return a.Errorf(errors.Wrap(err, "failed to create endorsement message"))
	}
	signature, err := bls.Sign(endorserSK, message)
	if err != nil {
		return a.Errorf(errors.Wrap(err, "failed to sign block endorsement"))
	}
	endorserIndex32, cErr := safecast.Convert[int32](endorserIndex)
	if cErr != nil {
		return a.Errorf(errors.Wrapf(cErr, "endorserIndex overflows int32: %v", endorserIndex))
	}

	finalizedHeight32, cErr := safecast.Convert[uint32](lastFinalizedHeight)
	if cErr != nil {
		return a.Errorf(errors.Wrapf(cErr, "lastFinalizedHeight overflows uint32: %v", lastFinalizedHeight))
	}
	endorseParentBlock := &proto.EndorseBlock{
		EndorserIndex:        endorserIndex32,
		FinalizedBlockID:     lastFinalizedBlock.BlockID(),
		FinalizedBlockHeight: finalizedHeight32,
		EndorsedBlockID:      parentBlockID,
		Signature:            signature,
	}
	id, idErr := endorsementID(endorseParentBlock)
	if idErr != nil {
		return a.Errorf(errors.Wrap(idErr, "failed to compute endorsement id"))
	}
	endorserWavesPK, findErr := a.baseInfo.storage.FindGeneratorPKByEndorserPK(periodStart, endorserPK)
	if findErr != nil {
		return findErr
	}
	endorserAddress := proto.MustAddressFromPublicKey(a.baseInfo.scheme, endorserWavesPK)
	endorserRec := proto.NewRecipientFromAddress(endorserAddress)
	balance, err := a.baseInfo.storage.GeneratingBalance(endorserRec, height)
	if err != nil {
		return err
	}
	addErr := a.baseInfo.endorsements.Add(endorseParentBlock, endorserPK,
		lastFinalizedHeight, lastFinalizedBlock.BlockID(), balance)
	if addErr != nil {
		return errors.Errorf("failed to add an endorsement, %v", addErr)
	}

	a.baseInfo.endorsementIDsCache.RememberEndorsement(id)
	a.baseInfo.actions.SendEndorseBlock(endorseParentBlock)
	slog.Debug("Sent a block endorsement:",
		"EndorserIndex", endorseParentBlock.EndorserIndex,
		"FinalizedBlockID", endorseParentBlock.FinalizedBlockID,
		"FinalizedBlockHeight", endorseParentBlock.FinalizedBlockHeight,
		"EndorsedBlockID", endorseParentBlock.EndorsedBlockID,
		"Signature", endorseParentBlock.Signature.String())
	return nil
}

func (a *NGState) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (State, Async, error) {
	metrics.MicroBlockReceived(micro, p.Handshake().NodeName)
	if !a.baseInfo.enableLightMode {
		block, err := a.checkAndAppendMicroBlock(micro) // the TopBlock() is used here
		if err != nil {
			metrics.MicroBlockDeclined(micro)
			return a, nil, a.Errorf(err)
		}

		a.baseInfo.logger.Debug("Received microblock successfully applied to state",
			"state", a.String(), "blockID", block.BlockID(), "ref", micro.Reference)
		a.baseInfo.MicroBlockCache.AddMicroBlock(block.BlockID(), micro)
		a.blocksCache.AddBlockState(block)
		a.baseInfo.scheduler.Reschedule()
		// Notify all connected peers about new microblock, send them microblock inv network message
		if inv, ok := a.baseInfo.MicroBlockInvCache.Get(block.BlockID()); ok {
			//TODO: We have to exclude from recipients peers that already have this microblock
			if err = broadcastMicroBlockInv(a.baseInfo, inv); err != nil {
				return a, nil, a.Errorf(errors.Wrap(err, "failed to handle microblock message"))
			}
		}
		return a, nil, nil
	}
	defer func() {
		pe := extension.NewPeerExtension(p, a.baseInfo.scheme, a.baseInfo.netLogger)
		pe.AskMicroBlockSnapshot(micro.TotalBlockID)
	}()
	st, timeoutTask := newWaitMicroSnapshotState(a.baseInfo, micro, a.blocksCache)
	return st, tasks.Tasks(timeoutTask), nil
}

// mineMicro handles a new microblock generated by miner.
func (a *NGState) mineMicro(
	minedBlock *proto.Block, rest proto.MiningLimits, keyPair proto.KeyPair, vrf []byte,
) (State, Async, error) {
	height, heightErr := a.baseInfo.storage.Height()
	if heightErr != nil {
		return a, nil, a.Errorf(heightErr)
	}
	finalityActivated, err := a.baseInfo.storage.IsActiveAtHeight(int16(settings.DeterministicFinality), height+1)
	if err != nil {
		return a, nil, a.Errorf(err)
	}
	var partialFinalization *proto.FinalizationVoting
	var blockFinalization *proto.FinalizationVoting
	if finalityActivated {
		lastFinalizedHeight, lastHeightErr := a.baseInfo.storage.LastFinalizedHeight()
		if lastHeightErr != nil {
			return a, nil, a.Errorf(lastHeightErr)
		}
		partialFinalization, err = a.getPartialFinalization(lastFinalizedHeight)
		if err != nil && !errors.Is(err, errNoFinalization) {
			return a, nil, a.Errorf(err)
		}
		blockFinalization, err = a.getBlockFinalization(height, lastFinalizedHeight)
		if err != nil && !errors.Is(err, errNoFinalization) {
			return a, nil, a.Errorf(err)
		}
	}
	block, micro, rest, err := a.baseInfo.microMiner.Micro(minedBlock, rest, keyPair, partialFinalization,
		blockFinalization)
	switch {
	case errors.Is(err, miner.ErrNoTransactions) || errors.Is(err, miner.ErrBlockIsFull): // no txs to include in micro
		a.baseInfo.logger.Debug(
			"No transactions to put in microblock",
			slog.String("state", a.String()),
			logging.Error(err),
			slog.Any("miningLimits", rest),
		)
		return a, tasks.Tasks(tasks.NewMineMicroTask(a.baseInfo.microblockInterval, minedBlock, rest, keyPair, vrf)), nil
	case errors.Is(err, miner.ErrStateChanged):
		return a, nil, a.Errorf(proto.NewInfoMsg(err))
	case err != nil:
		return a, nil, a.Errorf(errors.Wrap(err, "failed to generate microblock"))
	}
	metrics.MicroBlockMined(micro, block.TransactionCount)

	err = a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
		_, er := a.baseInfo.blocksApplier.ApplyMicro(s, block)
		return er
	})
	if err != nil {
		return a, nil, a.Errorf(err)
	}
	a.baseInfo.logger.Debug("Generated microblock successfully applied to state", "state", a.String(),
		"blockID", block.BlockID(), "ref", micro.Reference)
	a.blocksCache.AddBlockState(block)
	a.baseInfo.scheduler.Reschedule()
	metrics.MicroBlockApplied(micro)
	a.baseInfo.CleanUtx()
	inv := proto.NewUnsignedMicroblockInv(
		micro.SenderPK,
		block.BlockID(),
		micro.Reference)
	err = inv.Sign(keyPair.Secret, a.baseInfo.scheme)
	if err != nil {
		return a, nil, a.Errorf(err)
	}

	if err = broadcastMicroBlockInv(a.baseInfo, inv); err != nil {
		return a, nil, a.Errorf(errors.Wrap(err, "failed to broadcast generated microblock"))
	}

	blockchainHeight, err := a.baseInfo.storage.Height()
	if err != nil {
		return a, nil, a.Errorf(errors.Wrap(err, "failed to get blockchain height"))
	}
	// here the blockchainHeight is equal to lastBlockHeight because we are appending a microblock to the last block
	lastBlockHeight := blockchainHeight
	ok, err := a.baseInfo.storage.IsActiveLightNodeNewBlocksFields(lastBlockHeight)
	if err != nil {
		return a, nil, a.Errorf(err)
	}
	if ok {
		sh, errSh := a.baseInfo.storage.SnapshotStateHashAtHeight(lastBlockHeight)
		if errSh != nil {
			return a, nil, a.Errorf(errSh)
		}
		micro.StateHash = &sh
	}
	a.baseInfo.MicroBlockCache.AddMicroBlock(block.BlockID(), micro)
	a.baseInfo.MicroBlockInvCache.Add(block.BlockID(), inv)

	return a, tasks.Tasks(tasks.NewMineMicroTask(a.baseInfo.microblockInterval, block, rest, keyPair, vrf)), nil
}

// checkAndAppendMicroBlock checks that microblock is appendable and appends it.
func (a *NGState) checkAndAppendMicroBlock(
	micro *proto.MicroBlock,
) (*proto.Block, error) {
	top := a.baseInfo.storage.TopBlock()  // Get the last block
	if top.BlockID() != micro.Reference { // Microblock doesn't refer to last block
		err := errors.Errorf("microblock TBID '%s' refer to block ID '%s' but last block ID is '%s'",
			micro.TotalBlockID.String(), micro.Reference.String(), top.BlockID().String())
		metrics.MicroBlockDeclined(micro)
		return &proto.Block{}, proto.NewInfoMsg(err)
	}
	ok, err := micro.VerifySignature(a.baseInfo.scheme)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.Errorf("microblock '%s' has invalid signature", micro.TotalBlockID.String())
	}
	newTrs := top.Transactions.Join(micro.Transactions)
	newBlock, err := proto.CreateBlock(newTrs, top.Timestamp, top.Parent, top.GeneratorPublicKey, top.NxtConsensus,
		top.Version, top.Features, top.RewardVote, a.baseInfo.scheme, micro.StateHash, nil)
	if err != nil {
		return nil, err
	}
	newBlock.BlockSignature = micro.TotalResBlockSigField
	ok, err = newBlock.VerifySignature(a.baseInfo.scheme)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("incorrect signature for applied microblock")
	}
	err = newBlock.GenerateBlockID(a.baseInfo.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "NGState microBlockByID: failed generate block id")
	}
	err = a.baseInfo.storage.Map(func(state state.State) error {
		_, er := a.baseInfo.blocksApplier.ApplyMicro(state, newBlock)
		return er
	})

	if err != nil {
		metrics.MicroBlockDeclined(micro)
		return nil, errors.Wrap(err, "failed to apply created from micro block")
	}
	metrics.MicroBlockApplied(micro)
	a.baseInfo.CleanUtx()
	return newBlock, nil
}

func (a *NGState) MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (State, Async, error) {
	metrics.MicroBlockInv(inv, p.Handshake().NodeName)
	existed := a.baseInfo.invRequester.Request(p, inv.TotalBlockID)
	if existed {
		a.baseInfo.logger.Debug("Microblock inv received, but block already in cache", "state", a.String(),
			"blockID", inv.TotalBlockID)
	} else {
		a.baseInfo.logger.Debug("Microblock inv received, requesting block from peer", "state", a.String(),
			"blockID", inv.TotalBlockID, "peer", p.ID())
	}
	a.baseInfo.MicroBlockInvCache.Add(inv.TotalBlockID, inv)
	return a, nil, nil
}

func (a *NGState) Halt() (State, Async, error) {
	return newHaltState(a.baseInfo)
}

type blockStatesCache struct {
	blockStates map[proto.BlockID]proto.Block
	snapshots   map[proto.BlockID]proto.BlockSnapshot
	logger      *slog.Logger
	stateName   string
}

func newBlockStatesCache(logger *slog.Logger, stateName string) blockStatesCache {
	return blockStatesCache{
		blockStates: map[proto.BlockID]proto.Block{},
		snapshots:   map[proto.BlockID]proto.BlockSnapshot{},
		logger:      logger,
		stateName:   stateName,
	}
}

func (c *blockStatesCache) AddBlockState(block *proto.Block) {
	c.blockStates[block.ID] = *block
	c.logger.Debug("Block added to cache", "state", c.stateName, "blockID", block.ID.String(),
		"size", len(c.blockStates))
}

func (c *blockStatesCache) AddSnapshot(blockID proto.BlockID, snapshot proto.BlockSnapshot) {
	c.snapshots[blockID] = snapshot
	c.logger.Debug("Snapshot added to cache", "state", c.stateName, "blockID", blockID.String(),
		"size", len(c.snapshots))
}

func (c *blockStatesCache) Clear() {
	c.blockStates = map[proto.BlockID]proto.Block{}
	c.snapshots = map[proto.BlockID]proto.BlockSnapshot{}
	c.logger.Debug("Block cache is empty", "state", c.stateName)
}

func (c *blockStatesCache) Get(blockID proto.BlockID) (*proto.Block, bool) {
	block, ok := c.blockStates[blockID]
	if !ok {
		return nil, false
	}
	return &block, true
}

func (c *blockStatesCache) GetSnapshot(blockID proto.BlockID) (*proto.BlockSnapshot, bool) {
	snapshot, ok := c.snapshots[blockID]
	if !ok {
		return nil, false
	}
	return &snapshot, true
}

func initNGStateInFSM(state *StateData, fsm *stateless.StateMachine, info BaseInfo) {
	var ngSkipMessageList = proto.PeerMessageIDs{
		proto.ContentIDMicroBlockSnapshot,
		proto.ContentIDBlockSnapshot,
	}
	fsm.Configure(NGStateName).
		OnEntry(func(_ context.Context, _ ...any) error {
			info.skipMessageList.SetList(ngSkipMessageList)
			return nil
		}).
		Ignore(BlockIDsEvent).
		Ignore(StartMiningEvent).
		Ignore(ChangeSyncPeerEvent).
		Ignore(StopSyncEvent).
		Ignore(BlockSnapshotEvent).
		Ignore(MicroBlockSnapshotEvent).
		PermitDynamic(StopMiningEvent,
			createPermitDynamicCallback(StopMiningEvent, state, func(_ ...any) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.StopMining()
			})).
		PermitDynamic(TransactionEvent,
			createPermitDynamicCallback(TransactionEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.Transaction(convertToInterface[peer.Peer](args[0]),
					convertToInterface[proto.Transaction](args[1]))
			})).
		PermitDynamic(TaskEvent,
			createPermitDynamicCallback(TaskEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.Task(args[0].(tasks.AsyncTask))
			})).
		PermitDynamic(ScoreEvent,
			createPermitDynamicCallback(ScoreEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.Score(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Score))
			})).
		PermitDynamic(BlockEvent,
			createPermitDynamicCallback(BlockEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.Block(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Block))
			})).
		PermitDynamic(BlockEndorsementEvent,
			createPermitDynamicCallback(BlockEndorsementEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				endorse, ok := args[0].(*proto.EndorseBlock)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf("unexpected type %T, expected *proto.EndorseBlock", args[0]))
				}
				return a.BlockEndorsement(endorse)
			})).
		PermitDynamic(MinedBlockEvent,
			createPermitDynamicCallback(MinedBlockEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.MinedBlock(args[0].(*proto.Block), args[1].(proto.MiningLimits),
					args[2].(proto.KeyPair), args[3].([]byte))
			})).
		PermitDynamic(MicroBlockEvent,
			createPermitDynamicCallback(MicroBlockEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.MicroBlock(convertToInterface[peer.Peer](args[0]), args[1].(*proto.MicroBlock))
			})).
		PermitDynamic(MicroBlockInvEvent,
			createPermitDynamicCallback(MicroBlockInvEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.MicroBlockInv(convertToInterface[peer.Peer](args[0]), args[1].(*proto.MicroBlockInv))
			})).
		PermitDynamic(HaltEvent,
			createPermitDynamicCallback(HaltEvent, state, func(_ ...any) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.Halt()
			}))
}
