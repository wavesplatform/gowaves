package state_fsm

import (
	"time"

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
	BaseInfo
}

func (a *NGFsm) Transaction(p peer.Peer, t proto.Transaction) (FSM, Async, error) {
	err := a.utx.Add(t)
	if err != nil {
		a.BroadcastTransaction(t, p)
	}
	return a, nil, err
}

func (a *NGFsm) Task(task AsyncTask) (FSM, Async, error) {
	switch task.TaskType {
	case PING:
		return noop(a)
	case ASK_PEERS:
		a.peers.AskPeers()
		return a, nil, nil
	case MINE_MICRO:
		t := task.Data.(MineMicroTaskData)
		return a.mineMicro(t.Block, t.Limits, t.KeyPair, t.Vrf)
	default:
		return a, nil, errors.Errorf("NGFsm Task: unknown task type %d, data %+v", task.TaskType, task.Data)
	}
}

func (a *NGFsm) Halt() (FSM, Async, error) {
	return HaltTransition(a.BaseInfo)
}

func NewNGFsm12(info BaseInfo) *NGFsm {
	return &NGFsm{
		BaseInfo: info,
	}
}

func (a *NGFsm) NewPeer(p peer.Peer) (FSM, Async, error) {
	fsm, as, err := newPeer(a, p, a.peers)
	if a.peers.ConnectedCount() == a.minPeersMining {
		a.Reschedule()
	}
	sendScore(p, a.storage)
	return fsm, as, err
}

func (a *NGFsm) PeerError(p peer.Peer, e error) (FSM, Async, error) {
	return peerError(a, p, a.peers, e)
}

func (a *NGFsm) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
	return handleScore(a, a.BaseInfo, p, score)
}

func (a *NGFsm) Block(peer peer.Peer, block *proto.Block) (FSM, Async, error) {
	metrics.BlockReceived(block, peer.Handshake().NodeName)
	h, err := a.blocksApplier.Apply(a.storage, []*proto.Block{block})
	if err != nil {
		metrics.BlockDeclined(block)
		return a, nil, err
	}
	metrics.BlockApplied(block, h)
	a.Scheduler.Reschedule()
	a.actions.SendScore(a.storage)
	a.CleanUtx()
	return NewNGFsm12(a.BaseInfo), nil, nil
}

func (a *NGFsm) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	var h proto.Height
	err := a.storage.Map(func(state state.NonThreadSafeState) error {
		var err error
		h, err = a.blocksApplier.Apply(state, []*proto.Block{block})
		return err
	})
	if err != nil {
		zap.S().Info("NGFsm MinedBlock  err ", err)
		return a, nil, err
	}
	metrics.BlockMined(block, h)
	a.Reschedule()
	a.actions.SendBlock(block)
	a.actions.SendScore(a.storage)
	a.CleanUtx()
	return NewNGFsm12(a.BaseInfo), Tasks(NewMineMicroTask(1*time.Second, block, limits, keyPair, vrf)), nil
}

func (a *NGFsm) BlockIDs(peer peer.Peer, sigs []proto.BlockID) (FSM, Async, error) {
	return noop(a)
}

// received microblock
func (a *NGFsm) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error) {
	defer func() {
		zap.S().Debug("Reschedule form NGFsm.MicroBlock defer")
		a.BaseInfo.Reschedule()
	}()
	metrics.MicroBlockReceived(micro, p.Handshake().NodeName)
	_, _, err := a.microBlockByID(micro)
	if err != nil {
		return a, nil, err
	}
	a.MicroBlockCache.Add(a.storage.TopBlock().BlockID(), micro)
	inv, ok := a.MicroBlockInvCache.Get(a.storage.TopBlock().BlockID())
	if ok {
		invBts, err := inv.MarshalBinary()
		if err == nil {
			a.peers.EachConnected(func(p peer.Peer, score *proto.Score) {
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

func (a *NGFsm) mineMicro(minedBlock *proto.Block, rest proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	defer func() {
		zap.S().Debug("Reschedule form NGFsm.mineMicro defer")
		a.Reschedule()
	}()
	block, micro, rest, err := a.microMiner.Micro(minedBlock, rest, keyPair, vrf)
	if err == miner.NoTransactionsErr {
		return a, Tasks(NewMineMicroTask(5*time.Second, minedBlock, rest, keyPair, vrf)), nil
	}
	if err != nil {
		return a, nil, errors.Wrap(err, "NGFsm.mineMicro")
	}
	err = a.storage.Map(func(s state.NonThreadSafeState) error {
		_, err := a.blocksApplier.Apply(s, []*proto.Block{block})
		return err
	})
	if err != nil {
		return a, nil, err
	}
	inv := proto.NewUnsignedMicroblockInv(
		micro.SenderPK,
		block.ID,
		micro.Reference)
	err = inv.Sign(keyPair.Secret, a.scheme)
	if err != nil {
		return a, nil, err
	}
	invBts, err := inv.MarshalBinary()
	if err != nil {
		return a, nil, err
	}
	a.MicroBlockCache.Add(block.ID, micro)
	a.MicroBlockInvCache.Add(block.ID, inv)
	// TODO wrap
	a.peers.EachConnected(func(p peer.Peer, score *proto.Score) {
		p.SendMessage(
			&proto.MicroBlockInvMessage{
				Body: invBts,
			},
		)
	})
	return a, Tasks(NewMineMicroTask(5*time.Second, block, rest, keyPair, vrf)), nil
}

func (a *NGFsm) microBlockByID(micro *proto.MicroBlock) (FSM, Async, error) {
	top := a.storage.TopBlock()
	if top.BlockID() != micro.Reference {
		return a, nil, errors.New("micro reference not found")
	}
	b, err := a.storage.Block(micro.Reference)
	if err != nil {
		return a, nil, err
	}
	newTrs := b.Transactions.Join(micro.Transactions)
	newBlock, err := proto.CreateBlock(newTrs, b.Timestamp, b.Parent, b.GenPublicKey, b.NxtConsensus, b.Version, b.Features, b.RewardVote, a.scheme)
	if err != nil {
		return a, nil, err
	}
	newBlock.BlockSignature = micro.TotalResBlockSigField
	ok, err := newBlock.VerifySignature(a.scheme)
	if err != nil {
		return a, nil, err
	}
	if !ok {
		return a, nil, errors.New("incorrect signature for applied microblock")
	}
	err = newBlock.GenerateBlockID(a.scheme)
	if err != nil {
		return a, nil, errors.Wrap(err, "NGFsm microBlockByID: failed generate block id")
	}
	err = a.storage.Map(func(state state.State) error {
		_, err := a.blocksApplier.Apply(state, []*proto.Block{newBlock})
		return err
	})
	if err != nil {
		return a, nil, errors.Wrap(err, "failed to apply created from micro block")
	}
	metrics.MicroBlockApplied(micro)
	return a, nil, nil
}

func (a *NGFsm) MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (FSM, Async, error) {
	metrics.MicroBlockInv(inv, p.Handshake().NodeName)
	a.invRequester.Request(p, inv.TotalBlockID.Bytes())
	a.MicroBlockInvCache.Add(inv.TotalBlockID, inv)
	return a, nil, nil
}

func MinedBlockNgTransition(info BaseInfo, block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	return NewNGFsm12(info).MinedBlock(block, limits, keyPair, vrf)
}
