package node

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/rhansen/go-kairos/kairos"

	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/network"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	defaultChannelSize     = 100
	blockIDsSequenceLength = 101
	defaultSyncTimeout     = 30 * time.Second
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
	microblockInterval time.Duration
	obsolescence       time.Duration
	reward             int64

	// TODO: scheduler types.Scheduler
	tm        types.Time
	utx       types.UtxPool
	skipList  *messages.SkipMessageList
	st        state.State
	applier   *Applier
	syncPeer  peer.Peer
	syncTimer *kairos.Timer

	blocksCache        blockCache
	microBlockCache    microBlockCache
	microBlockInvCache microBlockInvCache

	sequence *blockSequence
	lastIDs  blockIDs // Put IDs in reverse order here.
	lastPeer peer.Peer
}

func NewNode(
	networkCh <-chan peer.ProtoMessage,
	notificationsCh <-chan network.Notification,
	broadcastCh <-chan *messages.BroadcastTransaction,
	scheme proto.Scheme, microblockInterval, obsolescence time.Duration,
	utx types.UtxPool, skipList *messages.SkipMessageList, tm types.Time, st state.State, applier *Applier,
	reward int64,
) (*Node, <-chan network.Command) {
	commandsCh := make(chan network.Command, defaultChannelSize)
	n := &Node{
		sm:                 stateless.NewStateMachine(stageIdle),
		networkCh:          networkCh,
		notificationsCh:    notificationsCh,
		commandsCh:         commandsCh,
		broadcastCh:        broadcastCh,
		scheme:             scheme,
		microblockInterval: microblockInterval,
		obsolescence:       obsolescence,
		reward:             reward,
		utx:                utx,
		skipList:           skipList,
		tm:                 tm,
		st:                 st,
		applier:            applier,
		blocksCache:        blockCache{blocks: map[proto.BlockID]proto.Block{}},
		microBlockCache:    newDefaultMicroblockCache(),
		microBlockInvCache: newDefaultMicroblockInvCache(),
		sequence:           newBlockSequence(blockIDsSequenceLength),
	}

	n.configureTriggers()
	n.configureIdleState()
	n.configureTillingState()
	n.configureSowingState()
	n.configureHarvestingState()
	n.configureGleaningState()
	n.configureOperationState()
	n.configureOperationNGState()
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
			zap.S().Named(logging.FSMNamespace).Debugf("[%s] Synchronization timeout", n.sm.MustState())
			if err := n.sm.Fire(eventAbortSync); err != nil {
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
		tx, err := proto.BytesToTransaction(msg.Transaction, n.scheme)
		if err != nil {
			zap.S().Named(logging.FSMNamespace).
				Errorf("[%s] Failed to deserialize transaction from '%s': %v",
					n.sm.MustState(), m.ID.ID().String(), err)
			// TODO: Consider black listing of the peer on transaction deserialization error
			return nil // Don't fail on deserialization error.
		}
		if fireErr := n.sm.Fire(eventTransaction, m.ID, tx); fireErr != nil {
			zap.S().Named(logging.FSMNamespace).
				Warnf("[%s] Failed to handle Transaction message: %v", n.sm.MustState(), fireErr)
		}
	case *proto.PBTransactionMessage:
		tx, err := proto.SignedTxFromProtobuf(msg.Transaction)
		if err != nil {
			zap.S().Named(logging.FSMNamespace).
				Errorf("[%s] Failed to deserialize transaction from '%s': %v",
					n.sm.MustState(), m.ID.ID().String(), err)
			// TODO: Consider black listing of the peer on transaction deserialization error
			return nil // Don't fail on deserialization error.
		}
		if fireErr := n.sm.Fire(eventTransaction, m.ID, tx); fireErr != nil {
			zap.S().Named(logging.FSMNamespace).
				Warnf("[%s] Failed to handle Transaction message: %v", n.sm.MustState(), fireErr)
		}
	case *proto.BlockMessage:
		n.handleBlockMessage(m.ID, msg)
	case *proto.PBBlockMessage:
		n.handlePBBlockMessage(m.ID, msg)
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
		if err := n.sm.Fire(eventResume, ntf.Peer); err != nil {
			zap.S().Named(logging.FSMNamespace).
				Warnf("[%s] Failed to handle QuorumMet notification: %v", n.sm.MustState(), err)
		}
	case network.QuorumLostNotification:
		if err := n.sm.Fire(eventSuspend); err != nil {
			zap.S().Named(logging.FSMNamespace).
				Warnf("[%s] Failed to handle QuorumLost notification: %v", n.sm.MustState(), err)
		}
	case network.SyncPeerChangedNotification:
		if err := n.sm.Fire(eventChangeSyncPeer, ntf.Peer, ntf.Score); err != nil {
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
		reflect.TypeOf((*proto.Transaction)(nil)).Elem())
	n.sm.SetTriggerParameters(eventBlock, reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf((*proto.Block)(nil)))
	n.sm.SetTriggerParameters(eventBlockIDs, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf([]proto.BlockID{}))
	n.sm.SetTriggerParameters(eventMicroBlockInv, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf((*proto.MicroBlockInv)(nil)))
	n.sm.SetTriggerParameters(eventGetMicroBlock, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf(proto.BlockID{}))
	n.sm.SetTriggerParameters(eventMicroBlock, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf((*proto.MicroBlock)(nil)))
	n.sm.SetTriggerParameters(eventChangeSyncPeer, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf((*big.Int)(nil)))
	n.sm.SetTriggerParameters(eventResume, reflect.TypeOf((*peer.Peer)(nil)).Elem())
	n.sm.SetTriggerParameters(eventBlockGenerated, reflect.TypeOf((*proto.Block)(nil)),
		reflect.TypeOf(proto.MiningLimits{}), reflect.TypeOf(proto.KeyPair{}), reflect.TypeOf([]byte{}))
	n.sm.SetTriggerParameters(eventBroadcastTransaction,
		reflect.TypeOf((*proto.Transaction)(nil)).Elem(), reflect.TypeOf((*error)(nil)).Elem())
}

func (n *Node) configureIdleState() {
	n.sm.Configure(stageIdle).
		OnEntry(n.onEnterIdle).
		Ignore(eventTransaction).
		Ignore(eventBlock).
		Ignore(eventBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		Ignore(eventChangeSyncPeer).
		Permit(eventResume, stageTilling).
		Ignore(eventSuspend).
		Ignore(eventBlockGenerated).
		Permit(eventPersistenceRequired, stagePersistence).
		Ignore(eventPersistenceComplete).
		Ignore(eventAbortSync).
		Ignore(eventBlockSequenceComplete).
		Ignore(eventBroadcastTransaction).
		Permit(eventHalt, stageHalt)
}

func (n *Node) configureTillingState() {
	n.sm.Configure(stageTilling).
		OnEntryFrom(eventResume, n.onEnterTillingFromIdle).
		OnEntryFrom(eventBlockSequenceComplete, n.onEnterTillingFromHarvesting).
		OnEntryFrom(eventChangeSyncPeer, n.onEnterTillingFromIdle).
		Ignore(eventTransaction).
		Ignore(eventBlock).
		Permit(eventBlockIDs, stageSowing, n.completeIDsBatch).
		Permit(eventBlockIDs, stageGleaning, n.incompleteIDsBatch).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		InternalTransition(eventChangeSyncPeer, n.onChangeSyncPeer).
		Ignore(eventResume).
		Permit(eventSuspend, stageIdle).
		Ignore(eventBlockGenerated).
		Permit(eventPersistenceRequired, stagePersistence).
		Ignore(eventPersistenceComplete).
		Permit(eventAbortSync, stageIdle).
		Ignore(eventBlockSequenceComplete).
		Ignore(eventBroadcastTransaction).
		Permit(eventHalt, stageHalt)
}

func (n *Node) configureSowingState() {
	n.sm.Configure(stageSowing).
		OnEntry(n.onEnterSowing).
		Ignore(eventTransaction).
		Permit(eventBlock, stageHarvesting, n.anticipatedBlock).
		Ignore(eventBlock, n.unanticipatedBlock).
		Ignore(eventBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		InternalTransition(eventChangeSyncPeer, n.onChangeSyncPeer).
		Ignore(eventResume).
		Permit(eventSuspend, stageIdle).
		Ignore(eventBlockGenerated).
		Permit(eventPersistenceRequired, stagePersistence).
		Ignore(eventPersistenceComplete).
		Permit(eventAbortSync, stageIdle).
		Ignore(eventBlockSequenceComplete).
		Ignore(eventBroadcastTransaction).
		Permit(eventHalt, stageHalt)
}

func (n *Node) configureHarvestingState() {
	n.sm.Configure(stageHarvesting).
		OnEntry(n.onEnterHarvesting).
		OnExit(n.applyBlockSequence).
		Ignore(eventTransaction).
		InternalTransition(eventBlock, n.onBlockSync).
		Ignore(eventBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		InternalTransition(eventChangeSyncPeer, n.onChangeSyncPeer).
		Ignore(eventResume).
		Permit(eventSuspend, stageIdle).
		Ignore(eventBlockGenerated).
		Permit(eventPersistenceRequired, stagePersistence).
		Ignore(eventPersistenceComplete).
		Permit(eventAbortSync, stageIdle).
		Permit(eventBlockSequenceComplete, stageTilling).
		Ignore(eventBroadcastTransaction).
		Permit(eventHalt, stageHalt)
}

func (n *Node) configureGleaningState() {
	n.sm.Configure(stageGleaning).
		OnEntry(n.onEnterGleaning).
		OnExit(n.applyBlockSequence).
		Ignore(eventTransaction).
		InternalTransition(eventBlock, n.onBlockSync).
		Ignore(eventBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		InternalTransition(eventChangeSyncPeer, n.onChangeSyncPeer).
		Ignore(eventResume).
		Permit(eventSuspend, stageIdle).
		Ignore(eventBlockGenerated).
		Permit(eventPersistenceRequired, stagePersistence).
		Ignore(eventPersistenceComplete).
		Permit(eventAbortSync, stageIdle).
		Permit(eventBlockSequenceComplete, stageOperation, n.inactiveNG).
		Permit(eventBlockSequenceComplete, stageOperationNG, n.activeNG).
		Ignore(eventBroadcastTransaction).
		Permit(eventHalt, stageHalt)
}

func (n *Node) configureOperationState() {
	n.sm.Configure(stageOperation).
		OnEntry(n.onEnterOperation).
		InternalTransition(eventTransaction, n.onTransaction).
		InternalTransition(eventBlock, n.onKeyBlock).
		Ignore(eventBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		Permit(eventChangeSyncPeer, stageTilling, n.higherScore).
		Ignore(eventChangeSyncPeer, n.lowerOrEqualScore).
		Ignore(eventResume).
		Permit(eventSuspend, stageIdle).
		InternalTransition(eventBlockGenerated, n.onBlockGenerated).
		Permit(eventPersistenceRequired, stagePersistence).
		Ignore(eventPersistenceComplete).
		Ignore(eventAbortSync).
		Ignore(eventBlockSequenceComplete).
		InternalTransition(eventBroadcastTransaction, n.onBroadcastTransaction).
		Permit(eventHalt, stageHalt)
}

func (n *Node) configureOperationNGState() {
	n.sm.Configure(stageOperationNG).
		OnEntry(n.onEnterOperationNG).
		// InternalTransition(eventTransaction, n.onTransaction).
		Ignore(eventTransaction). // TODO: Remove this line and uncomment the line above.
		InternalTransition(eventBlock, n.onKeyBlock).
		Ignore(eventBlockIDs).
		InternalTransition(eventMicroBlockInv, n.onMicroblockInv).
		InternalTransition(eventGetMicroBlock, n.onGetMicroblock).
		InternalTransition(eventMicroBlock, n.onMicroblock).
		Permit(eventChangeSyncPeer, stageTilling, n.higherScore).
		Ignore(eventChangeSyncPeer, n.lowerOrEqualScore).
		Ignore(eventResume).
		Permit(eventSuspend, stageIdle).
		InternalTransition(eventBlockGenerated, n.onBlockGenerated).
		Permit(eventPersistenceRequired, stagePersistence).
		Ignore(eventPersistenceComplete).
		Ignore(eventAbortSync).
		Ignore(eventBlockSequenceComplete).
		InternalTransition(eventBroadcastTransaction, n.onBroadcastTransaction).
		Permit(eventHalt, stageHalt)
}

func (n *Node) configurePersistState() {
	n.sm.Configure(stagePersistence).
		OnEntry(n.onEnterPersistence).
		Ignore(eventTransaction).
		Ignore(eventBlock).
		Ignore(eventBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		Ignore(eventChangeSyncPeer).
		Ignore(eventResume).
		Ignore(eventSuspend).
		Ignore(eventBlockGenerated).
		Ignore(eventPersistenceRequired).
		Permit(eventPersistenceComplete, stageIdle).
		Ignore(eventAbortSync).
		Ignore(eventBlockSequenceComplete).
		Ignore(eventBroadcastTransaction).
		Permit(eventHalt, stageHalt)
}

func (n *Node) configureHaltState() {
	n.sm.Configure(stageHalt).
		OnEntry(n.onEnterHalt).
		Ignore(eventTransaction).
		Ignore(eventBlock).
		Ignore(eventBlockIDs).
		Ignore(eventMicroBlockInv).
		Ignore(eventGetMicroBlock).
		Ignore(eventMicroBlock).
		Ignore(eventChangeSyncPeer).
		Ignore(eventResume).
		Ignore(eventSuspend).
		Ignore(eventBlockGenerated).
		Ignore(eventPersistenceRequired).
		Ignore(eventPersistenceComplete).
		Ignore(eventAbortSync).
		Ignore(eventBlockSequenceComplete).
		Ignore(eventBroadcastTransaction).
		Ignore(eventHalt)
}

func (n *Node) onEnterIdle(_ context.Context, _ ...any) error {
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Entered Idle state", n.sm.MustState())
	n.skipList.DisableForIdle()
	// Check if we need to persist transactions.
	required, err := n.st.ShouldPersistAddressTransactions()
	if err != nil {
		return errors.Wrap(err, "failed to check necessity for persistence in Idle state")
	}
	if required {
		return n.sm.Fire(eventPersistenceRequired)
	}
	// Requesting current state of the quorum.
	n.commandsCh <- network.RequestQuorumUpdate{}
	return nil
}

func (n *Node) onEnterOperation(_ context.Context, _ ...any) error {
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Entered Operation state", n.sm.MustState())
	n.skipList.DisableForOperation()
	n.commandsCh <- network.FollowLeaderCommand{}
	// TODO: Start mining
	// TODO: n.scheduler.Reschedule()
	return nil
}

func (n *Node) onEnterOperationNG(_ context.Context, _ ...any) error {
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Entered OperationNG state", n.sm.MustState())
	n.skipList.DisableForOperationNG()
	n.commandsCh <- network.FollowLeaderCommand{}
	// TODO: Start mining
	// TODO: n.scheduler.Reschedule()
	return nil
}

func (n *Node) onEnterTillingFromIdle(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	if p == nil {
		return errors.New("failed to start synchronization with an empty peer")
	}
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Entered Tilling state", n.sm.MustState())
	n.skipList.DisableForSync()
	n.sequence.reset()
	n.syncPeer = p
	// Ask first batch of block IDs by providing the last blocks from state.
	b := n.st.TopBlock()
	n.chooseFollowingMode(b)
	ids, err := n.lastBlockIDsFromState()
	if err != nil {
		return errors.Wrap(err, "on enter Tilling state")
	}
	n.lastPeer = n.syncPeer
	n.lastIDs = ids // From state IDs comes in reverse order.
	n.askIDs()
	n.startSyncTimer()
	return nil
}

func (n *Node) onEnterTillingFromHarvesting(_ context.Context, _ ...any) error {
	n.sequence.reset()
	n.lastPeer = n.syncPeer
	n.askIDs()
	n.startSyncTimer()
	return nil
}

func (n *Node) onEnterSowing(_ context.Context, args ...any) error {
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Entering Sowing state form Tilling state", n.sm.MustState())
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	if !n.lastPeer.Equal(p) {
		zap.S().Named(logging.FSMNamespace).
			Debugf("[%s] Block IDs received from unexpected peer '%s', expecting from '%s'",
				n.sm.MustState(), p.ID().String(), n.lastPeer.ID().String())
		return n.sm.Fire(eventAbortSync)
	}
	ids, ok := args[1].([]proto.BlockID) // From other nodes IDs comes in natural order.
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected '[]proto.BlockID'", args[1])
	}
	// Extract unknown sequential IDs from received sequence.
	unknownIDs, intersects := relativeCompliment(n.lastIDs.reverse(), ids)
	if !intersects {
		return n.sm.Fire(eventAbortSync) // We received IDs that has no intersection with our IDs sent in request.
	}
	// Store unknown IDs in `last` field in reverse order, to ask for next IDs using them.
	n.lastIDs = blockIDs(unknownIDs).reverse()
	// Ask for Blocks of unknown IDs.
	zap.S().Named(logging.FSMNamespace).
		Debugf("[%s] Requesting blocks [%s..%s](%d) from peer '%s'", n.sm.MustState(),
			unknownIDs[0].ShortString(), unknownIDs[len(unknownIDs)-1].ShortString(), len(unknownIDs), p.ID().String())
	for _, id := range unknownIDs {
		if pushed := n.sequence.pushID(id); !pushed {
			zap.S().Named(logging.FSMNamespace).
				Debugf("[%s] Malformed sequence of block IDs received from peer '%s'", n.sm.MustState(), p.ID().String())
			return n.sm.Fire(eventAbortSync)
		}
		n.askBlock(id)
	}
	n.startSyncTimer()
	return nil
}

func (n *Node) onEnterHarvesting(ctx context.Context, args ...any) error {
	n.startSyncTimer()
	return n.onBlockSync(ctx, args...)
}

func (n *Node) onEnterGleaning(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	if !p.Equal(n.syncPeer) {
		zap.S().Named(logging.FSMNamespace).
			Debugf("[%s] Block IDs received from unexpected peer '%s', expecting from '%s'",
				n.sm.MustState(), p.ID().String(), n.syncPeer.ID().String())
		return nil
	}
	ids, ok := args[1].([]proto.BlockID)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected '[]proto.BlockID'", args[1])
	}
	// Extract unknown sequential IDs from received sequence.
	unknownIDs, intersects := relativeCompliment(n.lastIDs.reverse(), ids)
	if !intersects {
		return n.sm.Fire(eventAbortSync) // We received IDs that has no intersection with our IDs sent in request.
	}
	// This is the last batch of block IDs, no need to store them, reset the field.
	n.lastIDs = nil
	// Ask for Blocks of unknown IDs.
	zap.S().Named(logging.FSMNamespace).
		Debugf("[%s] Requesting blocks [%s..%s](%d) from peer '%s'", n.sm.MustState(),
			unknownIDs[0].ShortString(), unknownIDs[len(unknownIDs)-1].ShortString(), len(unknownIDs), p.ID().String())
	for _, id := range unknownIDs {
		if pushed := n.sequence.pushID(id); !pushed {
			zap.S().Named(logging.FSMNamespace).
				Debugf("[%s] Malformed sequence of block IDs received from peer '%s'", n.sm.MustState(), p.ID().String())
			return n.sm.Fire(eventAbortSync)
		}
		n.askBlock(id)
	}
	n.startSyncTimer()
	return nil
}

func (n *Node) onEnterPersistence(_ context.Context, _ ...any) error {
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Entered Persistence state", n.sm.MustState())
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
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Entered Halt state", n.sm.MustState())
	n.skipList.DisableEverything()
	n.syncPeer = nil
	close(n.commandsCh)
	if err := n.st.Close(); err != nil {
		return err
	}
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] state closed", n.sm.MustState())
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

func (n *Node) onBlockSync(_ context.Context, args ...any) error {
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

	if !n.lastPeer.Equal(p) { // Block received from unexpected peer, ignore it.
		zap.S().Named(logging.FSMNamespace).
			Debugf("[%s] Block '%s' received from unexpected peer '%s', expecting from '%s'",
				st, b.BlockID().String(), p.ID().String(), n.lastPeer.ID().String())
		return nil
	}

	metrics.FSMKeyBlockReceived(st.String(), b, p.Handshake().NodeName)
	zap.S().Named(logging.FSMNamespace).
		Debugf("[%s] Block '%s' received from '%s'", st, b.BlockID().String(), p.ID().String())
	if !n.sequence.putBlock(b) {
		zap.S().Named(logging.FSMNamespace).Debugf("[%s] Unexpected block '%s'", st, b.BlockID().String())
		return nil
	}
	if n.sequence.full() {
		return n.sm.Fire(eventBlockSequenceComplete)
	}
	return nil
}

func (n *Node) applyBlockSequence(_ context.Context, _ ...any) error {
	n.stopSyncTimer()
	st, ok := n.sm.MustState().(stage)
	if !ok {
		return errors.Errorf("invlid type of FSM state '%T'", n.sm.MustState())
	}
	blocks := n.sequence.blocks()
	if len(blocks) == 0 {
		zap.S().Named(logging.FSMNamespace).Debugf("[%s] No blocks to apply", n.sm.MustState())
		return nil
	}
	zap.S().Named(logging.FSMNamespace).
		Debugf("[%s] Applying blocks [%s...%s](%d)",
			n.sm.MustState(), blocks[0].BlockID().ShortString(), blocks[len(blocks)-1].BlockID().ShortString(),
			len(blocks))
	topBlock, err := n.applier.applyBlocks(blocks)
	if err != nil {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Blocks [%s...%s](%d) application error: %v", n.sm.MustState(),
				blocks[0].BlockID().ShortString(), blocks[len(blocks)-1].BlockID().ShortString(), len(blocks), err)
		// TODO: Consider suspending the peer regardless the type of error.
		if errs.IsValidationError(err) || errs.IsValidationError(errors.Cause(err)) {
			zap.S().Named(logging.FSMNamespace).
				Debugf("[%s] Suspending peer '%s' because of blocks application error: %v",
					st, n.syncPeer.ID().String(), err)
			msg := fmt.Sprintf("[%s] Failed to apply blocks: %s", st, err.Error())
			n.commandsCh <- network.SuspendPeerCommand{Peer: n.syncPeer, Message: msg}
		}
		// TODO: Not all blocks can be rejected, consider returning number of applied blocks from applyBlocks.
		for _, rjb := range blocks {
			metrics.FSMKeyBlockDeclined(st.String(), rjb, err)
		}
		return n.sm.Fire(eventAbortSync)
	}
	for _, apb := range blocks {
		metrics.FSMKeyBlockApplied(st.String(), apb)
	}
	// TODO: Reschedule, eg: a.baseInfo.scheduler.Reschedule()

	// Announce new score to all connected peers.
	n.commandsCh <- network.AnnounceScoreCommand{}

	// Update following mode regarding top block time.
	n.chooseFollowingMode(topBlock)

	should, err := n.st.ShouldPersistAddressTransactions()
	if err != nil {
		return errors.Wrapf(err, "failed to check necessity to persist transactions in state '%s'", st.String())
	}
	if should {
		return n.sm.Fire(eventPersistenceRequired)
	}
	return nil
}

func (n *Node) onKeyBlock(_ context.Context, args ...any) error {
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
			Debugf("[%s] Inapplicable block '%s' has parent '%s' which is not the top block '%s'",
				n.sm.MustState(), b.ID.String(), b.Parent.String(), top.ID.String(),
			)
		var blockFromCache *proto.Block
		if blockFromCache, ok = n.blocksCache.get(b.Parent); ok {
			// Restore parent block from cache if any.
			zap.S().Named(logging.FSMNamespace).
				Debugf("[%s] Re-applying block '%s' from cache", n.sm.MustState(), blockFromCache.ID.String())
			if err = n.rollbackToStateFromCache(blockFromCache); err != nil {
				zap.S().Named(logging.FSMNamespace).
					Errorf("[%s] Failed to rollback state to block '%s': %v", n.sm.MustState(),
						blockFromCache.BlockID().String(), err)
				// TODO: Failed to apply block from cache, what should we do?
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
	n.commandsCh <- network.BroadcastBlockCommand{Block: b, Origin: p}
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

func (n *Node) higherScore(_ context.Context, args ...any) bool {
	s, ok := args[1].(*big.Int)
	if !ok {
		zap.S().Named(logging.FSMNamespace).
			Errorf("[%s] Invalid type '%T' of second argument, expected '*big.Int'", n.sm.MustState(), args[1])
		return false
	}
	ls, err := n.st.CurrentScore()
	if err != nil {
		zap.S().Named(logging.FSMNamespace).Errorf("[%s] Failed to get current score: %v", n.sm.MustState(), err)
		return false
	}
	return s.Cmp(ls) == 1 // Remote score is higher than local score.
}

func (n *Node) lowerOrEqualScore(ctx context.Context, args ...any) bool {
	return !n.higherScore(ctx, args...)
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
		Debugf("[%s] Micro-block Inv with block ID '%s' received from peer '%s'",
			n.sm.MustState(), inv.TotalBlockID.String(), p.ID().String())
	n.microBlockInvCache.put(inv.TotalBlockID, inv)

	p.SendMessage(&proto.MicroBlockRequestMessage{TotalBlockSig: inv.TotalBlockID.Bytes()})

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
		bts, err := mb.MarshalToProtobuf(n.scheme)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal micro-block in state '%s'", n.sm.MustState())
		}
		p.SendMessage(&proto.PBMicroBlockMessage{MicroBlockBytes: bts})
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

	zap.S().Named(logging.FSMNamespace).
		Debugf("[%s] Micro-block '%s' received from peer '%s'", n.sm.MustState(), mb.TotalBlockID.String(),
			p.ID().String())
	b, err := n.checkAndAppendMicroBlock(mb)
	if err != nil {
		metrics.FSMMicroBlockDeclined(st.String(), mb, err)
		zap.S().Named(logging.FSMNamespace).
			Errorf("[%s] Failed to apply micro-block '%s': %v", n.sm.MustState(), mb.TotalBlockID.String(), err)
		return n.sm.Fire(eventSuspend)
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

func (n *Node) activeNG(_ context.Context, _ ...any) bool {
	activated, err := n.st.IsActivated(int16(settings.NG))
	if err != nil {
		zap.S().Named(logging.FSMNamespace).
			Errorf("[%s] Unable to check NG feature activation: %v", n.sm.MustState(), err)
		return false
	}
	return activated
}

func (n *Node) inactiveNG(ctx context.Context, args ...any) bool {
	return !n.activeNG(ctx, args...)
}

func (n *Node) completeIDsBatch(_ context.Context, args ...any) bool {
	ids, ok := args[1].([]proto.BlockID)
	if !ok {
		zap.S().Named(logging.FSMNamespace).
			Fatalf("[%s] Invalid type '%T' of second argument, expected '[]proto.BlockID'",
				n.sm.MustState(), args[1])
		return false
	}
	return len(ids) == blockIDsSequenceLength
}

func (n *Node) incompleteIDsBatch(ctx context.Context, args ...any) bool {
	return !n.completeIDsBatch(ctx, args...)
}

func (n *Node) anticipatedBlock(_ context.Context, args ...any) bool {
	p, ok := args[0].(peer.Peer)
	if !ok {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Invalid type '%T' of first argument, expected 'peer.Peer'", n.sm.MustState(), args[0])
		return false
	}
	b, ok := args[1].(*proto.Block)
	if !ok {
		zap.S().Named(logging.FSMNamespace).
			Warnf("[%s] Invalid type '%T' of second argument, expected '*proto.Block'", n.sm.MustState(), args[1])
	}
	return n.lastPeer.Equal(p) && n.sequence.requested(b.BlockID())
}

func (n *Node) unanticipatedBlock(ctx context.Context, args ...any) bool {
	return !n.anticipatedBlock(ctx, args...)
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
	if _, aplErr := n.applier.applyMicroBlock(newBlock); aplErr != nil {
		return nil, errors.Wrapf(err, "failed to apply block created from micro-block '%s'",
			mb.TotalBlockID.String())
	}
	return newBlock, nil
}

// lastBlockIDsFromState returns blockIDs sequence built in reverse order (from newer to older blocks).
func (n *Node) lastBlockIDsFromState() (blockIDs, error) {
	ids := make([]proto.BlockID, 0, blockIDsSequenceLength)

	h, err := n.st.Height()
	if err != nil {
		zap.S().Named(logging.FSMNamespace).Errorf("Failed to get height from state: %v", err)
		return nil, err
	}

	for i := 0; i < blockIDsSequenceLength && h > 0; i++ {
		id, heightErr := n.st.HeightToBlockID(h)
		if heightErr != nil {
			zap.S().Named(logging.FSMNamespace).Errorf("Failed to get blockID for height %d: %v", h, heightErr)
			return nil, heightErr
		}
		ids = append(ids, id)
		h--
	}
	return ids, nil
}

// askIDs sends the lastIDs wrapped into the GetSignaturesMessage to the lastPeer.
// IDs in the message must be presented in the reverse order.
func (n *Node) askIDs() {
	if n.lastPeer == nil || len(n.lastIDs) == 0 {
		return
	}
	zap.S().Named(logging.FSMNamespace).
		Debugf("[%s] Requesting blocks IDs for IDs range [%s...%s](%d) from '%s'",
			n.sm.MustState(), n.lastIDs[0].ShortString(), n.lastIDs[len(n.lastIDs)-1].ShortString(), len(n.lastIDs),
			n.lastPeer.ID().String())
	n.lastPeer.SendMessage(&proto.GetBlockIdsMessage{Blocks: n.lastIDs})
}

func (n *Node) askBlock(id proto.BlockID) {
	if n.lastPeer == nil {
		return
	}
	zap.S().Named(logging.FSMNamespace).
		Debugf("[%s] Requesting block '%s' from '%s'", n.sm.MustState(), id.ShortString(),
			n.lastPeer.ID().String())
	n.lastPeer.SendMessage(&proto.GetBlockMessage{BlockID: id})
}

func (n *Node) startSyncTimer() {
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Restarting synchronization timeout timer", n.sm.MustState())
	n.syncTimer.Reset(defaultSyncTimeout)
}

func (n *Node) stopSyncTimer() {
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Stopping synchronization timeout timer", n.sm.MustState())
	n.syncTimer.Stop()
}

func (n *Node) obsolete(block *proto.Block) bool {
	now := n.tm.Now()
	obsolescenceTime := now.Add(-n.obsolescence)
	blockTime := time.UnixMilli(int64(block.Timestamp))
	return blockTime.Before(obsolescenceTime)
}

func (n *Node) chooseFollowingMode(block *proto.Block) {
	if block == nil {
		return
	}
	if n.obsolete(block) {
		n.commandsCh <- network.FollowGroupCommand{}
		return
	}
	n.commandsCh <- network.FollowLeaderCommand{}
}
