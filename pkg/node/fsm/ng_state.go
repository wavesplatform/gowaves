package fsm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/miner"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type NGState struct {
	baseInfo    BaseInfo
	blocksCache blockStatesCache
}

func newNGState(baseInfo BaseInfo) State {
	baseInfo.syncPeer.Clear()
	return &NGState{
		baseInfo:    baseInfo,
		blocksCache: newBlockStatesCache(),
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
		zap.S().Named(logging.FSMNamespace).Debug("[NG] Requesting peers")
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
	metrics.FSMScore("ng", score, p.Handshake().NodeName)
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
		return a.Errorf(errors.Errorf("block %s doesn't existin cache", parentID.String()))
	}
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Re-applying block '%s' from cache",
		a, blockFromCache.ID.String())
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
	ok, err := a.baseInfo.blocksApplier.BlockExists(a.baseInfo.storage, block)
	if err != nil {
		return a, nil, a.Errorf(errors.Wrapf(err, "peer '%s'", peer.ID()))
	}
	if ok {
		return a, nil, a.Errorf(proto.NewInfoMsg(errors.Errorf("Block '%s' already exists", block.BlockID().String())))
	}

	metrics.FSMKeyBlockReceived("ng", block, peer.Handshake().NodeName)

	top := a.baseInfo.storage.TopBlock()
	if top.BlockID() != block.Parent { // does block refer to last block
		zap.S().Named(logging.FSMNamespace).Debugf(
			"[%s] Key-block '%s' has parent '%s' which is not the top block '%s'",
			a, block.ID.String(), block.Parent.String(), top.ID.String(),
		)
		if a.baseInfo.enableLightMode {
			if err = a.rollbackToStateFromCacheInLightNode(block.Parent); err != nil {
				return a, nil, a.Errorf(err)
			}
		} else {
			if blockFromCache, okGet := a.blocksCache.Get(block.Parent); okGet {
				zap.S().Named(logging.FSMNamespace).Debugf("[%s] Re-applying block '%s' from cache",
					a, blockFromCache.ID.String())
				if err = a.rollbackToStateFromCache(blockFromCache); err != nil {
					return a, nil, a.Errorf(err)
				}
			}
		}
	}

	if a.baseInfo.enableLightMode {
		defer func() {
			pe := extension.NewPeerExtension(peer, a.baseInfo.scheme)
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
	a.blocksCache.Clear()
	a.blocksCache.AddBlockState(block)
	a.baseInfo.scheduler.Reschedule()
	a.baseInfo.actions.SendBlock(block)
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	a.baseInfo.CleanUtx()

	return newNGState(a.baseInfo), nil, nil
}

func (a *NGState) MinedBlock(
	block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte,
) (State, Async, error) {
	metrics.FSMKeyBlockGenerated("ng", block)
	err := a.baseInfo.storage.Map(func(state state.NonThreadSafeState) error {
		var err error
		_, err = a.baseInfo.blocksApplier.Apply(
			state,
			[]*proto.Block{block},
		)
		return err
	})
	if err != nil {
		zap.S().Warnf("[%s] Failed to apply generated key block '%s': %v", a, block.ID.String(), err)
		metrics.FSMKeyBlockDeclined("ng", block, err)
		return a, nil, a.Errorf(err)
	}
	metrics.FSMKeyBlockApplied("ng", block)
	zap.S().Infof("[%s] Generated key block '%s' successfully applied to state", a, block.ID.String())

	a.blocksCache.Clear()
	a.blocksCache.AddBlockState(block)
	a.baseInfo.scheduler.Reschedule()
	a.baseInfo.actions.SendBlock(block)
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	a.baseInfo.CleanUtx()

	return a, tasks.Tasks(tasks.NewMineMicroTask(0, block, limits, keyPair, vrf)), nil
}

func (a *NGState) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (State, Async, error) {
	metrics.FSMMicroBlockReceived("ng", micro, p.Handshake().NodeName)
	if !a.baseInfo.enableLightMode {
		block, err := a.checkAndAppendMicroBlock(micro) // the TopBlock() is used here
		if err != nil {
			metrics.FSMMicroBlockDeclined("ng", micro, err)
			return a, nil, a.Errorf(err)
		}
		zap.S().Named(logging.FSMNamespace).Debugf(
			"[%s] Received microblock '%s' (referencing '%s') successfully applied to state",
			a, block.BlockID(), micro.Reference,
		)
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
		pe := extension.NewPeerExtension(p, a.baseInfo.scheme)
		pe.AskMicroBlockSnapshot(micro.TotalBlockID)
	}()
	st, timeoutTask := newWaitMicroSnapshotState(a.baseInfo, micro, a.blocksCache)
	return st, tasks.Tasks(timeoutTask), nil
}

// mineMicro handles a new microblock generated by miner.
func (a *NGState) mineMicro(
	minedBlock *proto.Block, rest proto.MiningLimits, keyPair proto.KeyPair, vrf []byte,
) (State, Async, error) {
	block, micro, rest, err := a.baseInfo.microMiner.Micro(minedBlock, rest, keyPair)
	switch {
	case errors.Is(err, miner.NoTransactionsErr):
		zap.S().Named(logging.FSMNamespace).Debugf("[%s] No transactions to put in microblock: %v", a, err)
		return a, tasks.Tasks(tasks.NewMineMicroTask(a.baseInfo.microblockInterval, minedBlock, rest, keyPair, vrf)), nil
	case errors.Is(err, miner.StateChangedErr):
		return a, nil, a.Errorf(proto.NewInfoMsg(err))
	case err != nil:
		return a, nil, a.Errorf(errors.Wrap(err, "failed to generate microblock"))
	}
	metrics.FSMMicroBlockGenerated("ng", micro)
	err = a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
		_, er := a.baseInfo.blocksApplier.ApplyMicro(s, block)
		return er
	})
	if err != nil {
		return a, nil, a.Errorf(err)
	}
	zap.S().Named(logging.FSMNamespace).Debugf(
		"[%s] Generated microblock '%s' (referencing '%s') successfully applied to state",
		a, block.BlockID(), micro.Reference,
	)
	a.blocksCache.AddBlockState(block)
	a.baseInfo.scheduler.Reschedule()
	metrics.FSMMicroBlockApplied("ng", micro)
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
		metrics.FSMMicroBlockDeclined("ng", micro, err)
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
		top.Version, top.Features, top.RewardVote, a.baseInfo.scheme, micro.StateHash)
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
		metrics.FSMMicroBlockDeclined("ng", micro, err)
		return nil, errors.Wrap(err, "failed to apply created from micro block")
	}
	metrics.FSMMicroBlockApplied("ng", micro)
	return newBlock, nil
}

func (a *NGState) MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (State, Async, error) {
	metrics.MicroBlockInv(inv, p.Handshake().NodeName)
	existed := a.baseInfo.invRequester.Request(p, inv.TotalBlockID)
	if existed {
		zap.S().Named(logging.FSMNamespace).Debugf("[%s] Microblock inv received: block '%s' already in cache",
			a, inv.TotalBlockID)
	} else {
		zap.S().Named(logging.FSMNamespace).Debugf("[%s] Microblock inv received: block '%s' requested from peer '%s'",
			a, inv.TotalBlockID, p.ID())
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
}

func newBlockStatesCache() blockStatesCache {
	return blockStatesCache{
		blockStates: map[proto.BlockID]proto.Block{},
		snapshots:   map[proto.BlockID]proto.BlockSnapshot{},
	}
}

func (c *blockStatesCache) AddBlockState(block *proto.Block) {
	c.blockStates[block.ID] = *block
	zap.S().Named(logging.FSMNamespace).Debugf("[NG] Block '%s' added to cache, total blocks in cache: %d",
		block.ID.String(), len(c.blockStates))
}

func (c *blockStatesCache) AddSnapshot(blockID proto.BlockID, snapshot proto.BlockSnapshot) {
	c.snapshots[blockID] = snapshot
	zap.S().Named(logging.FSMNamespace).Debugf("[NG] Snapshot '%s' added to cache, total snapshots in cache: %d",
		blockID.String(), len(c.snapshots))
}

func (c *blockStatesCache) Clear() {
	c.blockStates = map[proto.BlockID]proto.Block{}
	c.snapshots = map[proto.BlockID]proto.BlockSnapshot{}
	zap.S().Named(logging.FSMNamespace).Debug("[NG] Block cache is empty")
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
		OnEntry(func(ctx context.Context, args ...interface{}) error {
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
			createPermitDynamicCallback(StopMiningEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.StopMining()
			})).
		PermitDynamic(TransactionEvent,
			createPermitDynamicCallback(TransactionEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.Transaction(convertToInterface[peer.Peer](args[0]),
					convertToInterface[proto.Transaction](args[1]))
			})).
		PermitDynamic(TaskEvent,
			createPermitDynamicCallback(TaskEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.Task(args[0].(tasks.AsyncTask))
			})).
		PermitDynamic(ScoreEvent,
			createPermitDynamicCallback(ScoreEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.Score(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Score))
			})).
		PermitDynamic(BlockEvent,
			createPermitDynamicCallback(BlockEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.Block(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Block))
			})).
		PermitDynamic(MinedBlockEvent,
			createPermitDynamicCallback(MinedBlockEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.MinedBlock(args[0].(*proto.Block), args[1].(proto.MiningLimits),
					args[2].(proto.KeyPair), args[3].([]byte))
			})).
		PermitDynamic(MicroBlockEvent,
			createPermitDynamicCallback(MicroBlockEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.MicroBlock(convertToInterface[peer.Peer](args[0]), args[1].(*proto.MicroBlock))
			})).
		PermitDynamic(MicroBlockInvEvent,
			createPermitDynamicCallback(MicroBlockInvEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.MicroBlockInv(convertToInterface[peer.Peer](args[0]), args[1].(*proto.MicroBlockInv))
			})).
		PermitDynamic(HaltEvent,
			createPermitDynamicCallback(HaltEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*NGState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*NGState'", state.State))
				}
				return a.Halt()
			}))
}
