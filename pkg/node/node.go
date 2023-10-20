package node

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/rhansen/go-kairos/kairos"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/network"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	defaultChannelSize     = 100
	blockIDsSequenceLength = 101
)

type Node struct {
	sm *stateless.StateMachine

	ctx  context.Context
	wait func() error

	networkCh       <-chan peer.ProtoMessage
	notificationsCh <-chan network.Notification
	broadcastCh     <-chan *messages.BroadcastTransaction
	commandsCh      chan<- network.Command

	scheme             proto.Scheme
	bindAddr           proto.TCPAddr
	declAddr           proto.TCPAddr
	microblockInterval time.Duration
	obsolescence       time.Duration
	reward             int64

	// TODO: scheduler types.Scheduler
	tm        types.Time
	utx       types.UtxPool
	skipList  *messages.SkipMessageList
	st        state.State
	applier   *applier
	syncPeer  peer.Peer
	syncTimer *kairos.Timer

	blocksCache        blocksCache
	microBlockCache    microBlockCache
	microBlockInvCache microBlockInvCache
}

func NewNode(
	networkCh <-chan peer.ProtoMessage,
	notificationsCh <-chan network.Notification,
	broadcastCh <-chan *messages.BroadcastTransaction,
	scheme proto.Scheme, bindAddr, declAddr proto.TCPAddr, microblockInterval, obsolescence time.Duration,
	utx types.UtxPool, skipList *messages.SkipMessageList, tm types.Time, st state.State, reward int64,
) (*Node, <-chan network.Command) {
	commandsCh := make(chan network.Command, defaultChannelSize)
	n := &Node{
		sm:                 stateless.NewStateMachine(stageIdle),
		networkCh:          networkCh,
		notificationsCh:    notificationsCh,
		commandsCh:         commandsCh,
		broadcastCh:        broadcastCh,
		scheme:             scheme,
		bindAddr:           bindAddr,
		declAddr:           declAddr,
		microblockInterval: microblockInterval,
		obsolescence:       obsolescence,
		reward:             reward,
		utx:                utx,
		skipList:           skipList,
		tm:                 tm,
		st:                 st,
		applier:            newApplier(st),
		blocksCache:        blocksCache{blocks: map[proto.BlockID]proto.Block{}},
		microBlockCache:    newDefaultMicroblockCache(),
		microBlockInvCache: newDefaultMicroblockInvCache(),
	}

	n.configureTriggers()
	n.configureIdleState()
	n.configureOperationState()
	n.configureOperationNGState()
	n.configureSyncState()
	n.configurePersistState()
	n.configureHaltState()

	return n, commandsCh
}

func (n *Node) Run(ctx context.Context) {
	g, gc := errgroup.WithContext(ctx)
	n.ctx = gc
	n.wait = g.Wait
	n.syncTimer = kairos.NewStoppedTimer()
	g.Go(n.handleEvents)
}

func (n *Node) Shutdown() {
	if err := n.wait(); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to properly shutdown node: %v", n.sm.MustState(), err)
	}
}

func (n *Node) handleEvents() error {
	for {
		select {
		case <-n.ctx.Done():
			if err := n.sm.Fire(eventHalt); err != nil {
				zap.S().Named(logging.FSMNamespace).
					Warnf("[%s] Failed to handle halt event: %v", n.sm.MustState(), err)
			}
			zap.S().Named(logging.FSMNamespace).Infof("[%s] Node termination started", n.sm.MustState())
			return nil
		case m, ok := <-n.networkCh:
			if err := n.handleNetworkMessages(m, ok); err != nil {
				return err
			}
		case m, ok := <-n.notificationsCh:
			if err := n.handleNotifications(m, ok); err != nil {
				return err
			}
		case m, ok := <-n.broadcastCh:
			if err := n.handleBroadcast(m, ok); err != nil {
				return err
			}
		case <-n.syncTimer.C:
			if err := n.sm.Fire(eventSyncTimeout); err != nil {
				zap.S().Named(logging.FSMNamespace).
					Warnf("[%s] Failed to handle sync timeout: %v", n.sm.MustState(), err)
			}
		}
	}
}

func (n *Node) handleBroadcast(m *messages.BroadcastTransaction, ok bool) error {
	if !ok {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Broadcast channel was closed by producer", n.sm.MustState())
		return errors.New("broadcast channel was closed")
	}
	if err := n.sm.Fire(eventBroadcastTransaction, m.Transaction, m.Response); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle transaction broadcast: %v", n.sm.MustState(), err)
	}
	return nil
}

func (n *Node) handleNetworkMessages(m peer.ProtoMessage, ok bool) error {
	if !ok {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Network messages channel was closed by producer", n.sm.MustState())
		return errors.New("network messages channel was closed")
	}
	switch msg := m.Message.(type) {
	case *proto.TransactionMessage:
		if err := n.sm.Fire(eventTransaction, msg.Transaction); err != nil {
			zap.S().Named(logging.FSMNamespace).
				Warnf("[%s] Failed to handle Transaction message: %v", n.sm.MustState(), err)
		}
	case *proto.PBTransactionMessage:
		if err := n.sm.Fire(eventTransaction, msg.Transaction); err != nil {
			zap.S().Named(logging.FSMNamespace).
				Warnf("[%s] Failed to handle Transaction message: %v", n.sm.MustState(), err)
		}
	case *proto.GetBlockMessage:
		n.handleGetBlockMessage(m.ID, msg)
	case *proto.BlockMessage:
		n.handleBlockMessage(m.ID, msg)
	case *proto.PBBlockMessage:
		n.handlePBBlockMessage(m.ID, msg)
	case *proto.GetSignaturesMessage:
		n.handleGetSignaturesMessage(m.ID, msg)
	case *proto.GetBlockIdsMessage:
		if err := n.sm.Fire(eventGetBlockIDs, msg.Blocks, false); err != nil {
			zap.S().Named(logging.FSMNamespace).
				Warnf("[%s] Failed to handle GetBlockIDs message: %v", n.sm.MustState(), err)
		}
	case *proto.SignaturesMessage:
		n.handleSignaturesMessage(m.ID, msg)
	case *proto.BlockIdsMessage:
		if err := n.sm.Fire(eventBlockIDs, m.ID, msg.Blocks); err != nil {
			zap.S().Named(logging.FSMNamespace).
				Warnf("[%s] Failed to handle BlockIDs message: %v", n.sm.MustState(), err)
		}
	case *proto.MicroBlockInvMessage:
		n.handleMicroBlockInvMessage(m.ID, msg)
	case *proto.MicroBlockRequestMessage:
		n.handleMicroBlockRequestMessage(m.ID, msg)
	case *proto.MicroBlockMessage:
		n.handleMicroBlockMessage(m.ID, msg)
	case *proto.PBMicroBlockMessage:
		n.handlePBMicroBlockMessage(m.ID, msg)
	default:
		zap.S().Named(logging.FSMNamespace).
			Errorf("[%s] Unexpected network message '%T'", n.sm.MustState(), m)
		return errors.Errorf("unexpected network message type '%T'", m)
	}
	return nil
}

func (n *Node) handleGetBlockMessage(p peer.Peer, msg *proto.GetBlockMessage) {
	metricGetBlockMessage.Inc()
	if err := n.sm.Fire(eventGetBlock, p, msg.BlockID); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle GetBlock message: %v", n.sm.MustState(), err)
	}
}

func (n *Node) handleBlockMessage(p peer.Peer, msg *proto.BlockMessage) {
	metricBlockMessage.Inc()
	b := &proto.Block{}
	if err := b.UnmarshalBinary(msg.BlockBytes, n.scheme); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle Block message: %v", n.sm.MustState(), err)
	}
	if err := n.sm.Fire(eventBlock, p, b); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle Block message: %v", n.sm.MustState(), err)
	}
}

func (n *Node) handlePBBlockMessage(p peer.Peer, msg *proto.PBBlockMessage) {
	metricBlockMessage.Inc()
	b := &proto.Block{}
	if err := b.UnmarshalFromProtobuf(msg.PBBlockBytes); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle PBBlock message: %v", n.sm.MustState(), err)
	}
	if err := n.sm.Fire(eventBlock, p, b); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle PBBlock message: %v", n.sm.MustState(), err)
	}
}

func (n *Node) handleGetSignaturesMessage(p peer.Peer, msg *proto.GetSignaturesMessage) {
	blockIDs := make([]proto.BlockID, len(msg.Signatures))
	for i, sig := range msg.Signatures {
		blockIDs[i] = proto.NewBlockIDFromSignature(sig)
	}
	if err := n.sm.Fire(eventGetBlockIDs, p, blockIDs, true); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle GetSignatures message: %v", n.sm.MustState(), err)
	}
}

func (n *Node) handleSignaturesMessage(p peer.Peer, msg *proto.SignaturesMessage) {
	blockIDs := make([]proto.BlockID, len(msg.Signatures))
	for i, sig := range msg.Signatures {
		blockIDs[i] = proto.NewBlockIDFromSignature(sig)
	}
	if err := n.sm.Fire(eventBlockIDs, p, blockIDs); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle Signatures message: %v", n.sm.MustState(), err)
	}
}

func (n *Node) handleMicroBlockInvMessage(p peer.Peer, msg *proto.MicroBlockInvMessage) {
	inv := &proto.MicroBlockInv{}
	if err := inv.UnmarshalBinary(msg.Body); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle MicroBlockInv message: %v", n.sm.MustState(), err)
	}
	if err := n.sm.Fire(eventMicroBlockInv, p, inv); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle MicroBlockInv message: %v", n.sm.MustState(), err)
	}
}

func (n *Node) handleMicroBlockRequestMessage(p peer.Peer, msg *proto.MicroBlockRequestMessage) {
	blockID, err := proto.NewBlockIDFromBytes(msg.TotalBlockSig)
	if err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle MicroBlockRequest message: %v", n.sm.MustState(), err)
	}
	if err = n.sm.Fire(eventGetMicroBlock, p, blockID); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle MicroBlockRequest message: %v", n.sm.MustState(), err)
	}
}

func (n *Node) handleMicroBlockMessage(p peer.Peer, msg *proto.MicroBlockMessage) {
	mb := &proto.MicroBlock{}
	if err := mb.UnmarshalBinary(msg.Body, n.scheme); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle MicroBlock message: %v", n.sm.MustState(), err)
	}
	if err := n.sm.Fire(eventMicroBlock, p, mb); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle MicroBlock message: %v", n.sm.MustState(), err)
	}
}

func (n *Node) handlePBMicroBlockMessage(p peer.Peer, msg *proto.PBMicroBlockMessage) {
	mb := &proto.MicroBlock{}
	if err := mb.UnmarshalFromProtobuf(msg.MicroBlockBytes); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle PBMicroBlock message: %v", n.sm.MustState(), err)
	}
	if err := n.sm.Fire(eventMicroBlock, p, mb); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Failed to handle PBMicroBlock message: %v", n.sm.MustState(), err)
	}
}

func (n *Node) handleNotifications(m network.Notification, ok bool) error {
	if !ok {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Notifications channel was closed by producer", n.sm.MustState())
		return errors.New("notifications channel was closed")
	}
	switch ntf := m.(type) {
	case network.QuorumMetNotification:
		if err := n.sm.Fire(eventResume); err != nil {
			zap.S().Named(logging.FSMNamespace).
				Warnf("[%s] Failed to handle QuorumMet notification: %v", n.sm.MustState(), err)
		}
	case network.QuorumLostNotification:
		if err := n.sm.Fire(eventSuspend); err != nil {
			zap.S().Named(logging.FSMNamespace).
				Warnf("[%s] Failed to handle QuorumLost notification: %v", n.sm.MustState(), err)
		}
	case network.SyncPeerSelectedNotification:
		if err := n.sm.Fire(eventChangeSyncPeer, ntf.Peer); err != nil {
			zap.S().Named(logging.FSMNamespace).
				Warnf("[%s] Failed to handle ChangeSyncPeer notification: %v", n.sm.MustState(), err)
		}
	default:
		zap.S().Named(logging.FSMNamespace).
			Errorf("[%s] Unexpected notification '%T'", n.sm.MustState(), m)
		return errors.Errorf("unexpected notification type '%T'", m)
	}
	return nil
}

func (n *Node) configureTriggers() {
	n.sm.SetTriggerParameters(eventTransaction, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf((proto.Transaction)(nil)))
	n.sm.SetTriggerParameters(eventGetBlock, reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf(proto.BlockID{}))
	n.sm.SetTriggerParameters(eventBlock, reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf((*proto.Block)(nil)))
	n.sm.SetTriggerParameters(eventBlockIDs, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf([]crypto.Signature{}))
	n.sm.SetTriggerParameters(eventGetBlockIDs, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf([]proto.BlockID{}),
		reflect.TypeOf(false))
	n.sm.SetTriggerParameters(eventMicroBlockInv, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf((*proto.MicroBlockInv)(nil)))
	n.sm.SetTriggerParameters(eventGetMicroBlock, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf((*proto.BlockID)(nil)))
	n.sm.SetTriggerParameters(eventMicroBlock, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf((*proto.MicroBlock)(nil)))
	n.sm.SetTriggerParameters(eventChangeSyncPeer, reflect.TypeOf((*peer.Peer)(nil)).Elem())
	n.sm.SetTriggerParameters(eventBlockGenerated, reflect.TypeOf((*proto.Block)(nil)),
		reflect.TypeOf(proto.MiningLimits{}), reflect.TypeOf(proto.KeyPair{}), reflect.TypeOf([]byte{}))
	n.sm.SetTriggerParameters(eventBroadcastTransaction,
		reflect.TypeOf((proto.Transaction)(nil)), reflect.TypeOf((error)(nil)))
}

func (n *Node) configureIdleState() {
	n.sm.Configure(stageIdle).
		OnEntry(n.onEnterIdle).
		Ignore(eventTransaction).
		Ignore(eventGetBlock).
		Ignore(eventBlock).
		Ignore(eventGetBlockIDs).
		Ignore(eventBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		Ignore(eventSuspend).
		Ignore(eventChangeSyncPeer).
		Permit(eventResume, stageSync).
		Permit(eventHalt, stageHalt).
		Permit(eventPersistenceRequired, stagePersistence).
		Ignore(eventPersistenceComplete).
		Ignore(eventBroadcastTransaction)
}

func (n *Node) configureOperationState() {
	n.sm.Configure(stageOperation).
		OnEntry(n.onEnterOperation).
		InternalTransition(eventTransaction, n.onTransaction).
		InternalTransition(eventGetBlock, n.onGetBlock).
		InternalTransition(eventBlock, n.onBlock).
		InternalTransition(eventGetBlockIDs, n.onGetBlockIDs).
		Ignore(eventBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		Ignore(eventChangeSyncPeer).
		Ignore(eventResume).
		Permit(eventSuspend, stageIdle).
		InternalTransition(eventBlockGenerated, n.onBlockGenerated).
		Permit(eventHalt, stageHalt).
		Permit(eventPersistenceRequired, stagePersistence).
		Ignore(eventPersistenceComplete).
		InternalTransition(eventBroadcastTransaction, n.onBroadcastTransaction)
}

func (n *Node) configureOperationNGState() {
	n.sm.Configure(stageOperation).
		OnEntry(n.onEnterOperation).
		InternalTransition(eventTransaction, n.onTransaction).
		InternalTransition(eventGetBlock, n.onGetBlock).
		InternalTransition(eventBlock, n.onBlock).
		InternalTransition(eventGetBlockIDs, n.onGetBlockIDs).
		Ignore(eventBlockIDs).
		InternalTransition(eventMicroBlockInv, n.onMicroblockInv).
		InternalTransition(eventGetMicroBlock, n.onGetMicroblock).
		InternalTransition(eventMicroBlock, n.onMicroblock).
		Ignore(eventChangeSyncPeer).
		Ignore(eventResume).
		Permit(eventSuspend, stageIdle).
		InternalTransition(eventBlockGenerated, n.onBlockGenerated).
		Permit(eventHalt, stageHalt).
		Permit(eventPersistenceRequired, stagePersistence).
		Ignore(eventPersistenceComplete).
		InternalTransition(eventBroadcastTransaction, n.onBroadcastTransaction)
}

func (n *Node) configureSyncState() {
	n.sm.Configure(stageSync).
		OnEntry(n.onEnterSync).
		InternalTransition(eventTransaction, n.onTransaction).
		InternalTransition(eventGetBlock, n.onGetBlock).
		InternalTransition(eventBlock, n.onBlock).
		InternalTransition(eventGetBlockIDs, n.onGetBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		InternalTransition(eventChangeSyncPeer, n.onChangeSyncPeer).
		Ignore(eventResume).
		Permit(eventSuspend, stageIdle).
		InternalTransition(eventBlockGenerated, n.onBlockGenerated).
		Permit(eventHalt, stageHalt).
		Permit(eventPersistenceRequired, stagePersistence).
		Ignore(eventPersistenceComplete).
		Ignore(eventBroadcastTransaction)
}

func (n *Node) configurePersistState() {
	n.sm.Configure(stagePersistence).
		OnEntry(n.onEnterPersistence).
		Ignore(eventTransaction).
		Ignore(eventGetBlock).
		Ignore(eventBlock).
		Ignore(eventGetBlockIDs).
		Ignore(eventBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		Ignore(eventChangeSyncPeer).
		Ignore(eventResume).
		Ignore(eventSuspend).
		Ignore(eventBlockGenerated).
		Permit(eventHalt, stageHalt).
		Ignore(eventPersistenceRequired).
		Permit(eventPersistenceComplete, stageIdle).
		Ignore(eventBroadcastTransaction)
}

func (n *Node) configureHaltState() {
	n.sm.Configure(stageHalt).
		OnEntry(n.onEnterHalt).
		Ignore(eventTransaction).
		Ignore(eventGetBlock).
		Ignore(eventBlock).
		Ignore(eventGetBlockIDs).
		Ignore(eventBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		Ignore(eventChangeSyncPeer).
		Ignore(eventResume).
		Ignore(eventSuspend).
		Ignore(eventBlockGenerated).
		Ignore(eventHalt).
		Ignore(eventPersistenceRequired).
		Ignore(eventPersistenceComplete).
		Ignore(eventBroadcastTransaction)
}

func (n *Node) onEnterIdle(_ context.Context, _ ...any) error {
	n.skipList.DisableForIdle()
	return nil
}

func (n *Node) onEnterOperation(_ context.Context, _ ...any) error {
	n.skipList.DisableForOperation()
	// TODO: Start mining
	// TODO: n.scheduler.Reschedule()
	return nil
}

func (n *Node) onEnterSync(_ context.Context, _ ...any) error {
	n.skipList.DisableForSync()
	// TODO: n.scheduler.Reschedule()
	return nil
}

func (n *Node) onEnterPersistence(_ context.Context, _ ...any) error {
	n.skipList.DisableEverything()
	n.syncPeer = nil
	if err := n.st.PersistAddressTransactions(); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Errorf("[%s] Failed to persist transactions: %v", n.sm.MustState(), err)
		return nil
	}
	return n.sm.Fire(eventPersistenceComplete)
}

func (n *Node) onEnterHalt(_ context.Context, _ ...any) error {
	n.skipList.DisableEverything()
	n.syncPeer = nil
	close(n.commandsCh)
	if err := n.st.Close(); err != nil {
		return err
	}
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] state closed", n.sm.MustState())
	return nil
}

func (n *Node) onGetBlockIDs(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	ids, ok := args[1].([]proto.BlockID)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected '[]proto.BlockID'", args[1])
	}
	asSignatures, ok := args[2].(bool)
	if !ok {
		return errors.Errorf("invalid type '%T' of third argument, expected 'bool'", args[2])
	}
	for _, id := range ids {
		if h, err := n.st.BlockIDToHeight(id); err == nil {
			n.sendNextBlockIDs(p, h, id, asSignatures)
			break
		}
	}
	return nil
}

func (n *Node) onTransaction(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	tx, ok := args[1].(proto.Transaction)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected 'proto.Transaction'", args[1])
	}

	if _, err := tx.Validate(n.scheme); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Debugf("[%s] Failed to validate transaction '%s' from peer '%s': %v",
				n.sm.MustState(), n.transactionID(tx), p.ID().String(), err)
		err = errors.Wrap(err, "failed to validate transaction")
		if p != nil {
			msg := fmt.Sprintf("[%s] Invalid transaction %s: %s",
				n.sm.MustState(), n.transactionID(tx), err.Error())
			n.commandsCh <- network.BlacklistPeerCommand{Peer: p, Message: msg}
		}
		return nil
	}

	if err := n.utx.Add(tx); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Debugf("[%s] Failed to add transaction '%s' into UTX: %v",
				n.sm.MustState(), n.transactionID(tx), err)
		return nil
	}
	n.commandsCh <- network.BroadcastTransactionCommand{Transaction: tx, Origin: p}
	return nil
}

func (n *Node) onGetBlock(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	id, ok := args[1].(proto.BlockID)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'proto.BlockID'", args[1])
	}
	block, err := n.st.Block(id)
	if err != nil {
		zap.S().Named(logging.FSMNamespace).
			Errorf("[%s] Failed to retriev a block by ID '%s': %v", n.sm.MustState(), id.String(), err)
		return nil
	}
	bm, err := proto.MessageByBlock(block, n.scheme)
	if err != nil {
		zap.S().Named(logging.FSMNamespace).
			Errorf("[%s] Failed to build Block message: %v", n.sm.MustState(), err)
		return nil
	}
	p.SendMessage(bm)
	return nil
}

func (n *Node) onBlock(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	b, ok := args[1].(*proto.Block)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected '*proto.Block'", args[1])
	}
	st, ok := n.sm.MustState().(stage)
	if !ok {
		return errors.Errorf("invlid type of FSM state '%T'", n.sm.MustState())
	}

	ok, err := n.applier.exists(b)
	if err != nil { // Not a retrieval error, real state problem, actually no such error at this time can occur.
		return err
	}
	if ok {
		zap.S().Named(logging.FSMNamespace).
			Debugf("[%s] Block '%s' already exists", n.sm.MustState(), b.BlockID().String())
		return nil
	}

	metrics.FSMKeyBlockReceived(st.String(), b, p.Handshake().NodeName)
	zap.S().Named(logging.FSMNamespace).
		Debugf("[%s] Block '%s' received from %s", n.sm.MustState(), b.BlockID().String(), p.ID())

	top := n.st.TopBlock()
	if top.BlockID() != b.Parent { // Received block doesn't refer to the last block.
		zap.S().Named(logging.FSMNamespace).
			Debugf("[%s] Key-block '%s' has parent '%s' which is not the top block '%s'",
				n.sm.MustState(), b.ID.String(), b.Parent.String(), top.ID.String(),
			)
		var blockFromCache *proto.Block
		if blockFromCache, ok = n.blocksCache.get(b.Parent); ok {
			zap.S().Named(logging.FSMNamespace).
				Debugf("[%s] Re-applying block '%s' from cache", n.sm.MustState(), blockFromCache.ID.String())
			if err = n.rollbackToStateFromCache(blockFromCache); err != nil {
				zap.S().Named(logging.FSMNamespace).
					Errorf("[%s] Failed to rollback state to block '%s': %v", n.sm.MustState(),
						blockFromCache.BlockID().String(), err)
				return nil
			}
		}
	}

	_, err = n.applier.applyBlocks([]*proto.Block{b})
	if err != nil {
		metrics.FSMKeyBlockDeclined(st.String(), b, err)
		zap.S().Named(logging.FSMNamespace).Errorf("[%s] Failed to apply block '%s' from '%s': %v",
			n.sm.MustState(), b.BlockID().String(), p.ID().String(), err)
		return nil
	}
	metrics.FSMKeyBlockApplied(st.String(), b)
	zap.S().Named(logging.FSMNamespace).
		Debugf("[%s] Block '%s' successfully applied to state", n.sm.MustState(), b.BlockID().String())

	n.blocksCache.clear()
	n.blocksCache.put(b)

	// TODO: n.scheduler.Reschedule()
	n.commandsCh <- network.AnnounceScoreCommand{}
	n.cleanUTX()
	return nil
}

func (n *Node) onBlockGenerated(_ context.Context, _ ...any) error {
	// TODO: Implement
	return errors.New("Not implemented")
}

func (n *Node) onChangeSyncPeer(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	if p == nil {
		zap.S().Named(logging.FSMNamespace).Debugf("[%s] Empty sync peer received", n.sm.MustState())
		return nil
	}
	n.syncPeer = p
	return nil
}

func (n *Node) onMicroblockInv(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	inv, ok := args[1].(*proto.MicroBlockInv)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected '*proto.MicroBlockInv'", args[1])
	}

	metrics.MicroBlockInv(inv, p.Handshake().NodeName)

	if n.microBlockInvCache.exist(inv.TotalBlockID) {
		zap.S().Named(logging.FSMNamespace).
			Debugf("[%s] MicroBlockInv received: block '%s' already in cache",
				n.sm.MustState(), inv.TotalBlockID)
		return nil
	}

	zap.S().Named(logging.FSMNamespace).
		Debugf("[%s] MicroBlockInv received: peer '%s' proposed a new block '%s'",
			n.sm.MustState(), p.ID().String(), inv.TotalBlockID.String())
	n.microBlockInvCache.put(inv.TotalBlockID, inv)

	msg := &proto.MicroBlockRequestMessage{TotalBlockSig: inv.TotalBlockID.Bytes()}
	p.SendMessage(msg)

	return nil
}

func (n *Node) onGetMicroblock(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	id, ok := args[1].(*proto.BlockID)
	if !ok {
		return errors.Errorf("invalid type '%T' of secod argument, expected '*proto.BlockID'", args[1])
	}
	var mb *proto.MicroBlock
	if mb, ok = n.microBlockCache.get(*id); ok {
		_ = extension.NewPeerExtension(p, n.scheme).SendMicroBlock(mb)
	}
	return nil
}

func (n *Node) onMicroblock(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	mb, ok := args[1].(*proto.MicroBlock)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected '*proto.MicroBlock'", args[1])
	}
	st, ok := n.sm.MustState().(stage)
	if !ok {
		return errors.Errorf("invlid type of FSM state '%T'", n.sm.MustState())
	}
	metrics.FSMMicroBlockReceived(st.String(), mb, p.Handshake().NodeName)

	b, err := n.checkAndAppendMicroBlock(mb)
	if err != nil {
		metrics.FSMMicroBlockDeclined(st.String(), mb, err)
		zap.S().Named(logging.FSMNamespace).
			Errorf("[%s]", n.sm.MustState())
		return nil
	}
	metrics.FSMMicroBlockApplied(st.String(), mb)
	zap.S().Named(logging.FSMNamespace).
		Debugf("[%s] Received microblock '%s' (referencing '%s') successfully applied to state",
			n.sm.MustState(), b.BlockID(), mb.Reference)
	n.microBlockCache.put(b.BlockID(), mb)
	n.blocksCache.put(b)

	// Notify all connected peers about new microblock, send them microblock inv network message
	var inv *proto.MicroBlockInv
	if inv, ok = n.microBlockInvCache.get(b.BlockID()); ok {
		n.commandsCh <- network.BroadcastMicroBlockInvCommand{MicroBlockInv: inv, Origin: p}
	}
	return nil
}

func (n *Node) onBroadcastTransaction(_ context.Context, args ...any) error {
	tx, ok := args[0].(proto.Transaction)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'proto.Transaction'", args[0])
	}
	replyTo, ok := args[1].(chan error)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected 'chan error'", args[1])
	}

	if _, err := tx.Validate(n.scheme); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Debugf("[%s] Failed to validate transaction '%s' from API: %v",
				n.sm.MustState(), n.transactionID(tx), err)
		replyTo <- errors.Wrap(err, "failed to validate transaction")
		return nil
	}

	if err := n.utx.Add(tx); err != nil {
		zap.S().Named(logging.FSMNamespace).
			Debugf("[%s] Failed to add transaction '%s' into UTX: %v",
				n.sm.MustState(), n.transactionID(tx), err)
		replyTo <- errors.Wrap(err, "failed to add to UTX")
		return nil
	}

	n.commandsCh <- network.BroadcastTransactionCommand{Transaction: tx, Origin: nil}
	return nil
}

func (n *Node) sendNextBlockIDs(p peer.Peer, height proto.Height, id proto.BlockID, asSignatures bool) {
	ids := make([]proto.BlockID, 1, blockIDsSequenceLength)
	ids[0] = id                                   // Put the common block ID as first in result
	for i := 1; i < blockIDsSequenceLength; i++ { // Add up to 100 more IDs
		b, err := n.st.HeaderByHeight(height + uint64(i))
		if err != nil {
			break
		}
		ids = append(ids, b.BlockID())
	}

	// There are block signatures to send in addition to requested one
	if len(ids) > 1 {
		if asSignatures {
			sigs := convertToSignatures(ids) // It could happen that only part of IDs can be converted to signatures
			if len(sigs) > 1 {
				p.SendMessage(&proto.SignaturesMessage{Signatures: sigs})
			}
			return
		}
		p.SendMessage(&proto.BlockIdsMessage{Blocks: ids})
	}
}

func (n *Node) transactionID(tx proto.Transaction) string {
	id, err := tx.GetID(n.scheme)
	if err != nil {
		return "n/a"
	}
	return base58.Encode(id)
}

func (n *Node) cleanUTX() {
	utxpool.NewCleaner(n.st, n.utx, n.tm).Clean()
}

func (n *Node) rollbackToStateFromCache(blockFromCache *proto.Block) error {
	previousBlockID := blockFromCache.Parent
	err := n.st.RollbackTo(previousBlockID)
	if err != nil {
		return errors.Wrapf(err, "failed to rollback to parent block '%s' of cached block '%s'",
			previousBlockID.String(), blockFromCache.ID.String())
	}
	_, err = n.applier.applyBlocks([]*proto.Block{blockFromCache})
	if err != nil {
		return errors.Wrapf(err, "failed to apply cached block %q", blockFromCache.ID.String())
	}
	return nil
}

func (n *Node) checkAndAppendMicroBlock(mb *proto.MicroBlock) (*proto.Block, error) {
	top := n.st.TopBlock()             // Get the last block
	if top.BlockID() != mb.Reference { // Microblock doesn't refer to last block
		return &proto.Block{}, errors.Errorf("microblock '%s' refers to block '%s' but last block is '%s'",
			mb.TotalBlockID.String(), mb.Reference.String(), top.BlockID().String())
	}
	ok, err := mb.VerifySignature(n.scheme)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.Errorf("microblock '%s' has invalid signature", mb.TotalBlockID.String())
	}
	newTrs := top.Transactions.Join(mb.Transactions)
	newBlock, err := proto.CreateBlock(newTrs, top.Timestamp, top.Parent, top.GeneratorPublicKey, top.NxtConsensus,
		top.Version, top.Features, top.RewardVote, n.scheme)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create new block from micro-block '%s'",
			mb.TotalBlockID.String())
	}
	newBlock.BlockSignature = mb.TotalResBlockSigField
	ok, err = newBlock.VerifySignature(n.scheme)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to verify new block created from micro-block '%s'",
			mb.TotalBlockID.String())
	}
	if !ok {
		return nil, errors.Errorf("incorrect signature for block created from micro-block '%s'",
			mb.TotalBlockID.String())
	}
	err = newBlock.GenerateBlockID(n.scheme)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate block ID for micro-block '%s'",
			mb.TotalBlockID.String())
	}
	err = n.st.Map(func(state state.State) error {
		_, aplErr := n.applier.applyMicroBlock(newBlock)
		return aplErr
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to apply block created from micro-block '%s'",
			mb.TotalBlockID.String())
	}
	return newBlock, nil
}

func convertToSignatures(ids []proto.BlockID) []crypto.Signature {
	sigs := make([]crypto.Signature, len(ids))
	for i, id := range ids {
		if !id.IsSignature() {
			break
		}
		sigs[i] = id.Signature()
	}
	return sigs
}
