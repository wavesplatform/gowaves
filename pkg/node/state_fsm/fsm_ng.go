package state_fsm

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/miner"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

type NGFsm struct {
	baseInfo    BaseInfo
	blocksCache blockStatesCache
}

var (
	ngSkipMessageList proto.PeerMessageIDs
)

func (a *NGFsm) Transaction(p peer.Peer, t proto.Transaction) (FSM, Async, error) {
	return tryBroadcastTransaction(a, a.baseInfo, p, t)
}

func (a *NGFsm) Task(task AsyncTask) (FSM, Async, error) {
	switch task.TaskType {
	case Ping:
		return noop(a)
	case AskPeers:
		zap.S().Debug("[NG] Requesting peers")
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case MineMicro:
		t := task.Data.(MineMicroTaskData)
		return a.mineMicro(t.Block, t.Limits, t.KeyPair, t.Vrf)
	default:
		return a, nil, a.Errorf(errors.Errorf("unexpected internal task '%d' with data '%+v' received by %s FSM", task.TaskType, task.Data, a.String()))
	}
}

func (a *NGFsm) Halt() (FSM, Async, error) {
	return HaltTransition(a.baseInfo)
}

func NewNGFsm12(info BaseInfo) *NGFsm {
	info.skipMessageList.SetList(ngSkipMessageList)
	return &NGFsm{
		baseInfo:    info,
		blocksCache: blockStatesCache{blockStates: map[proto.BlockID]proto.Block{}},
	}
}

func (a *NGFsm) NewPeer(p peer.Peer) (FSM, Async, error) {
	fsm, as, fsmErr := newPeer(a, p, a.baseInfo.peers)
	if a.baseInfo.peers.ConnectedCount() == a.baseInfo.minPeersMining {
		a.baseInfo.Reschedule()
	}
	sendScore(p, a.baseInfo.storage)
	return fsm, as, fsmErr
}

func (a *NGFsm) PeerError(p peer.Peer, e error) (FSM, Async, error) {
	return a.baseInfo.d.PeerError(a, p, a.baseInfo, e)
}

func (a *NGFsm) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
	metrics.FSMScore("ng", score, p.Handshake().NodeName)
	if err := a.baseInfo.peers.UpdateScore(p, score); err != nil {
		return a, nil, a.Errorf(proto.NewInfoMsg(err))
	}
	nodeScore, err := a.baseInfo.storage.CurrentScore()
	if err != nil {
		return a, nil, a.Errorf(err)
	}
	if score.Cmp(nodeScore) == 1 {
		return syncWithNewPeer(a, a.baseInfo, p)
	}
	return noop(a)
}

func (a *NGFsm) rollbackToStateFromCache(blockFromCache *proto.Block) error {
	previousBlockID := blockFromCache.Parent
	err := a.baseInfo.storage.RollbackTo(previousBlockID)
	if err != nil {
		return errors.Wrapf(err, "failed to rollback to parent block '%s' of cached block '%s'",
			previousBlockID.String(), blockFromCache.ID.String())
	}
	_, err = a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{blockFromCache})
	if err != nil {
		return errors.Wrapf(err, "failed to apply cached block %q", blockFromCache.ID.String())
	}
	return nil
}

func (a *NGFsm) Block(peer peer.Peer, block *proto.Block) (FSM, Async, error) {
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
		zap.S().Debugf("[%s] Key-block '%s' has parent '%s' which is not the top block '%s'",
			a, block.ID.String(), block.Parent.String(), top.ID.String())
		if blockFromCache, ok := a.blocksCache.Get(block.Parent); ok {
			zap.S().Debugf("[%s] Re-applying block '%s' from cache", a, blockFromCache.ID.String())
			err := a.rollbackToStateFromCache(blockFromCache)
			if err != nil {
				return a, nil, a.Errorf(err)
			}
		}
	}

	_, err = a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{block})
	if err != nil {
		metrics.FSMKeyBlockDeclined("ng", block, err)
		return a, nil, a.Errorf(errors.Wrapf(err, "peer '%s'", peer.ID()))
	}
	metrics.FSMKeyBlockApplied("ng", block)
	zap.S().Debugf("[%s] Handle received key block message: block '%s' applied to state", a, block.BlockID())

	a.blocksCache.Clear()
	a.blocksCache.AddBlockState(block)

	a.baseInfo.Scheduler.Reschedule()
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	a.baseInfo.CleanUtx()

	return NewNGFsm12(a.baseInfo), nil, nil
}

func (a *NGFsm) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	metrics.FSMKeyBlockGenerated("ng", block)
	err := a.baseInfo.storage.Map(func(state state.NonThreadSafeState) error {
		var err error
		_, err = a.baseInfo.blocksApplier.Apply(state, []*proto.Block{block})
		return err
	})
	if err != nil {
		zap.S().Warnf("[%s] Failed to apply mined block '%s': %v", a, block.ID.String(), err)
		metrics.FSMKeyBlockDeclined("ng", block, err)
		return a, nil, a.Errorf(err)
	}
	metrics.FSMKeyBlockApplied("ng", block)
	zap.S().Infof("[%s] Generating key block: block '%s' applied to state", a, block.ID.String())

	a.blocksCache.Clear()
	a.blocksCache.AddBlockState(block)

	a.baseInfo.Reschedule()
	a.baseInfo.actions.SendBlock(block)
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	a.baseInfo.CleanUtx()

	// Try to mine micro-block just after key-block generation
	return NewNGFsm12(a.baseInfo), Tasks(NewMineMicroTask(0, block, limits, keyPair, vrf)), nil
}

func (a *NGFsm) BlockIDs(_ peer.Peer, _ []proto.BlockID) (FSM, Async, error) {
	return noop(a)
}

// MicroBlock handles new microblock message.
func (a *NGFsm) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error) {
	metrics.FSMMicroBlockReceived("ng", micro, p.Handshake().NodeName)
	block, err := a.checkAndAppendMicroblock(micro) // the TopBlock() is used here
	if err != nil {
		metrics.FSMMicroBlockDeclined("ng", micro, err)
		return a, nil, a.Errorf(err)
	}
	zap.S().Debugf("[%s] Handle received microblock message: block '%s' applied to state, microblock ref '%s'",
		a, block.BlockID(), micro.Reference,
	)
	a.baseInfo.MicroBlockCache.Add(block.BlockID(), micro)
	a.blocksCache.AddBlockState(block)
	a.baseInfo.Reschedule()

	// Notify all connected peers about new microblock, send them microblock inv network message
	if inv, ok := a.baseInfo.MicroBlockInvCache.Get(block.BlockID()); ok {
		//TODO: We have to exclude from recipients peers that already have this microblock
		if err := a.broadcastMicroBlockInv(inv); err != nil {
			return a, nil, a.Errorf(errors.Wrap(err, "failed to handle microblock message"))
		}
	}
	return a, nil, nil
}

// New microblock generated by miner
func (a *NGFsm) mineMicro(minedBlock *proto.Block, rest proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	block, micro, rest, err := a.baseInfo.microMiner.Micro(minedBlock, rest, keyPair)
	switch {
	case errors.Is(err, miner.NoTransactionsErr):
		zap.S().Debugf("[%s] Generating microblock, skip: %v", a, err)
		return a, Tasks(NewMineMicroTask(a.baseInfo.microblockInterval, minedBlock, rest, keyPair, vrf)), nil
	case errors.Is(err, miner.StateChangedErr):
		return a, nil, a.Errorf(proto.NewInfoMsg(err))
	case err != nil:
		return a, nil, a.Errorf(errors.Wrap(err, "NGFsm.mineMicro"))
	}
	metrics.FSMMicroBlockGenerated("ng", micro)
	err = a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
		_, err := a.baseInfo.blocksApplier.ApplyMicro(s, block)
		return err
	})
	if err != nil {
		return a, nil, a.Errorf(err)
	}
	zap.S().Debugf("[%s] Generating microblock: block '%s' applied to state, microblock ref '%s'",
		a, block.BlockID(), micro.Reference,
	)
	a.blocksCache.AddBlockState(block)
	a.baseInfo.Reschedule()
	metrics.FSMMicroBlockApplied("ng", micro)
	inv := proto.NewUnsignedMicroblockInv(
		micro.SenderPK,
		block.BlockID(),
		micro.Reference)
	err = inv.Sign(keyPair.Secret, a.baseInfo.scheme)
	if err != nil {
		return a, nil, a.Errorf(err)
	}

	if err := a.broadcastMicroBlockInv(inv); err != nil {
		return a, nil, a.Errorf(errors.Wrap(err, "failed to broadcast generated microblock"))
	}

	a.baseInfo.MicroBlockCache.Add(block.BlockID(), micro)
	a.baseInfo.MicroBlockInvCache.Add(block.BlockID(), inv)

	return a, Tasks(NewMineMicroTask(a.baseInfo.microblockInterval, block, rest, keyPair, vrf)), nil
}

// broadcastMicroBlockInv broadcasts proto.MicroBlockInv message.
func (a *NGFsm) broadcastMicroBlockInv(inv *proto.MicroBlockInv) error {
	invBts, err := inv.MarshalBinary()
	if err != nil {
		return errors.Wrapf(err, "failed to marshal binary '%T'", inv)
	}
	var (
		cnt int
		msg = &proto.MicroBlockInvMessage{
			Body: invBts,
		}
	)
	a.baseInfo.peers.EachConnected(func(p peer.Peer, score *proto.Score) {
		p.SendMessage(msg)
		cnt++
	})
	a.baseInfo.invRequester.Add2Cache(inv.TotalBlockID.Bytes()) // prevent further unnecessary microblock request
	zap.S().Debugf("Network message '%T' sent to %d peers: blockID='%s', ref='%s'",
		msg, cnt, inv.TotalBlockID, inv.Reference,
	)
	return nil
}

// Check than microblock is appendable and append it
func (a *NGFsm) checkAndAppendMicroblock(micro *proto.MicroBlock) (*proto.Block, error) {
	top := a.baseInfo.storage.TopBlock()  // Get the last block
	if top.BlockID() != micro.Reference { // Microblock doesn't refer to last block
		err := errors.Errorf("microblock TBID '%s' refer to block ID '%s' but last block ID is '%s'", micro.TotalBlockID.String(), micro.Reference.String(), top.BlockID().String())
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
	newBlock, err := proto.CreateBlock(newTrs, top.Timestamp, top.Parent, top.GeneratorPublicKey, top.NxtConsensus, top.Version, top.Features, top.RewardVote, a.baseInfo.scheme)
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
		return nil, errors.Wrap(err, "NGFsm microBlockByID: failed generate block id")
	}
	err = a.baseInfo.storage.Map(func(state state.State) error {
		_, err := a.baseInfo.blocksApplier.ApplyMicro(state, newBlock)
		return err
	})
	if err != nil {
		metrics.FSMMicroBlockDeclined("ng", micro, err)
		return nil, errors.Wrap(err, "failed to apply created from micro block")
	}
	metrics.FSMMicroBlockApplied("ng", micro)
	return newBlock, nil
}

func (a *NGFsm) MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (FSM, Async, error) {
	metrics.MicroBlockInv(inv, p.Handshake().NodeName)
	existed := a.baseInfo.invRequester.Request(p, inv.TotalBlockID.Bytes()) // TODO: add logs about microblock request
	if existed {
		zap.S().Debugf("[%s] Handle received microblock-inv message: block '%s' already in cache", a, inv.TotalBlockID)
	} else {
		zap.S().Debugf("[%s] Handle received microblock-inv message: requested '%s' about block '%s'", a, p.ID(), inv.TotalBlockID)
	}
	a.baseInfo.MicroBlockInvCache.Add(inv.TotalBlockID, inv)
	return a, nil, nil
}

func (a *NGFsm) String() string {
	return "NG"
}

func (a *NGFsm) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func MinedBlockNgTransition(info BaseInfo, block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	return NewNGFsm12(info).MinedBlock(block, limits, keyPair, vrf)
}

type blockStatesCache struct {
	blockStates map[proto.BlockID]proto.Block
}

func (c *blockStatesCache) AddBlockState(block *proto.Block) {
	c.blockStates[block.ID] = *block
	zap.S().Debugf("Block '%s' added to cache, total blocks in cache: %d", block.ID.String(), len(c.blockStates))
}

func (c *blockStatesCache) Clear() {
	c.blockStates = map[proto.BlockID]proto.Block{}
	zap.S().Debug("Block cache is empty")
}

func (c *blockStatesCache) Get(blockID proto.BlockID) (*proto.Block, bool) {
	block, ok := c.blockStates[blockID]
	if !ok {
		return nil, false
	}
	return &block, true
}
