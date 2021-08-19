package state_fsm

import (
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/miner"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/sync_internal"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

type NGFsm struct {
	baseInfo    BaseInfo
	blocksCache blockStatesCache
}

func (a *NGFsm) Transaction(p peer.Peer, t proto.Transaction) (FSM, Async, error) {
	err := a.baseInfo.utx.Add(t)
	if err != nil {
		return a, nil, proto.NewInfoMsg(err)
	}
	a.baseInfo.BroadcastTransaction(t, p)
	return a, nil, nil
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
		zap.S().Debug("[NG] Generating micro-block")
		t := task.Data.(MineMicroTaskData)
		return a.mineMicro(t.Block, t.Limits, t.KeyPair, t.Vrf)
	default:
		return a, nil, errors.Errorf("unexpected internal task '%d' with data '%+v' received by %s FSM", task.TaskType, task.Data, a.String())
	}
}

func (a *NGFsm) Halt() (FSM, Async, error) {
	return HaltTransition(a.baseInfo)
}

func NewNGFsm12(info BaseInfo) *NGFsm {
	return &NGFsm{
		baseInfo:    info,
		blocksCache: blockStatesCache{blockStates: map[proto.BlockID]proto.Block{}},
	}
}

func (a *NGFsm) NewPeer(p peer.Peer) (FSM, Async, error) {
	fsm, as, err := newPeer(a, p, a.baseInfo.peers)
	if a.baseInfo.peers.ConnectedCount() == a.baseInfo.minPeersMining {
		a.baseInfo.Reschedule()
	}
	sendScore(p, a.baseInfo.storage)
	return fsm, as, err
}

func (a *NGFsm) PeerError(p peer.Peer, e error) (FSM, Async, error) {
	return peerError(a, p, a.baseInfo.peers, e)
}

func (a *NGFsm) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
	metrics.FSMScore("ng", score, p.Handshake().NodeName)
	if err := a.baseInfo.peers.UpdateScore(p, score); err != nil {
		return a, nil, proto.NewInfoMsg(err)
	}
	nodeScore, err := a.baseInfo.storage.CurrentScore()
	if err != nil {
		return a, nil, err
	}
	if score.Cmp(nodeScore) == 1 {
		lastSignatures, err := signatures.LastSignaturesImpl{}.LastBlockIDs(a.baseInfo.storage)
		if err != nil {
			return a, nil, err
		}
		internal := sync_internal.InternalFromLastSignatures(extension.NewPeerExtension(p, a.baseInfo.scheme), lastSignatures)
		c := conf{
			peerSyncWith: p,
			timeout:      30 * time.Second,
		}
		zap.S().Debugf("[NG] Higher score received, starting synchronisation with peer '%s'", p.ID())
		return NewSyncFsm(a.baseInfo, c.Now(a.baseInfo.tm), internal)
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
		return err
	}
	return nil
}

func (a *NGFsm) Block(peer peer.Peer, block *proto.Block) (FSM, Async, error) {
	ok, err := a.baseInfo.blocksApplier.BlockExists(a.baseInfo.storage, block)
	if err != nil {
		return a, nil, err
	}
	if ok {
		return a, nil, proto.NewInfoMsg(errors.Errorf("Block '%s' already exists", block.BlockID().String()))
	}

	metrics.FSMKeyBlockReceived("ng", block, peer.Handshake().NodeName)

	top := a.baseInfo.storage.TopBlock()
	if top.BlockID() != block.Parent { // does block refer to last block
		zap.S().Debugf("Key-block '%s' has parent '%s' which is not the top block '%s'",
			block.ID.String(), block.Parent.String(), top.ID.String())
		if blockFromCache, ok := a.blocksCache.Get(block.Parent); ok {
			zap.S().Debugf("Re-applying block '%s' from cache", blockFromCache.ID.String())
			err := a.rollbackToStateFromCache(blockFromCache)
			if err != nil {
				return a, nil, err
			}
		}
	}

	_, err = a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{block})
	if err != nil {
		metrics.FSMKeyBlockDeclined("ng", block, err)
		return a, nil, err
	}
	metrics.FSMKeyBlockApplied("ng", block)

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
		zap.S().Warnf("Failed to apply mined block '%s': %v", block.ID.String(), err)
		metrics.FSMKeyBlockDeclined("ng", block, err)
		return a, nil, err
	}
	zap.S().Infof("Key block '%s' generated and applied", block.ID.String())
	metrics.FSMKeyBlockApplied("ng", block)

	a.blocksCache.Clear()
	a.blocksCache.AddBlockState(block)

	a.baseInfo.Reschedule()
	a.baseInfo.actions.SendBlock(block)
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	a.baseInfo.CleanUtx()

	return NewNGFsm12(a.baseInfo), Tasks(NewMineMicroTask(1*time.Second, block, limits, keyPair, vrf)), nil
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
		return a, nil, err
	}
	a.baseInfo.MicroBlockCache.Add(block.BlockID(), micro)
	a.blocksCache.AddBlockState(block)
	a.baseInfo.Reschedule()

	// Notify all connected peers about new microblock, send them microblock inv network message
	inv, ok := a.baseInfo.MicroBlockInvCache.Get(block.BlockID())
	if ok {
		invBts, err := inv.MarshalBinary()
		if err == nil {
			//TODO: We have to exclude from recipients peers that already have this microblock
			a.baseInfo.peers.EachConnected(func(p peer.Peer, score *proto.Score) {
				p.SendMessage(
					&proto.MicroBlockInvMessage{
						Body: invBts,
					},
				)
			})
		} else {
			zap.S().Errorf("NGFsm.MicroBlock inv.MarshalBinary %q", err)
		}
	}
	return a, nil, nil
}

// New microblock generated by miner
func (a *NGFsm) mineMicro(minedBlock *proto.Block, rest proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	block, micro, rest, err := a.baseInfo.microMiner.Micro(minedBlock, rest, keyPair)
	if err == miner.NoTransactionsErr {
		return a, Tasks(NewMineMicroTask(5*time.Second, minedBlock, rest, keyPair, vrf)), nil
	}
	if err == miner.StateChangedErr {
		return a, nil, proto.NewInfoMsg(err)
	}
	if err != nil {
		return a, nil, errors.Wrap(err, "NGFsm.mineMicro")
	}
	metrics.FSMMicroBlockGenerated("ng", micro)
	err = a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
		_, err := a.baseInfo.blocksApplier.ApplyMicro(s, block)
		return err
	})
	if err != nil {
		return a, nil, err
	}
	a.blocksCache.AddBlockState(block)
	a.baseInfo.Reschedule()
	metrics.FSMMicroBlockApplied("ng", micro)
	inv := proto.NewUnsignedMicroblockInv(
		micro.SenderPK,
		block.BlockID(),
		micro.Reference)
	err = inv.Sign(keyPair.Secret, a.baseInfo.scheme)
	if err != nil {
		return a, nil, err
	}
	invBts, err := inv.MarshalBinary()
	if err != nil {
		return a, nil, err
	}

	a.baseInfo.MicroBlockCache.Add(block.BlockID(), micro)
	a.baseInfo.MicroBlockInvCache.Add(block.BlockID(), inv)
	// TODO wrap
	a.baseInfo.peers.EachConnected(func(p peer.Peer, score *proto.Score) {
		p.SendMessage(
			&proto.MicroBlockInvMessage{
				Body: invBts,
			},
		)
	})

	return a, Tasks(NewMineMicroTask(5*time.Second, block, rest, keyPair, vrf)), nil
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
	newBlock, err := proto.CreateBlock(newTrs, top.Timestamp, top.Parent, top.GenPublicKey, top.NxtConsensus, top.Version, top.Features, top.RewardVote, a.baseInfo.scheme)
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
	a.baseInfo.invRequester.Request(p, inv.TotalBlockID.Bytes())
	a.baseInfo.MicroBlockInvCache.Add(inv.TotalBlockID, inv)
	return a, nil, nil
}

func (a *NGFsm) String() string {
	return "NG"
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
