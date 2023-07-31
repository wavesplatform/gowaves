package node

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/node/network"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/node/fsm"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peers"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	spawnOutgoingConnectionsInterval        = 1 * time.Minute
	metricInternalChannelSizeUpdateInterval = 1 * time.Second
)

type Config struct {
	AppName  string
	NodeName string
	Listen   string
	DeclAddr string
}

type Node struct {
	peers              peers.PeerManager
	state              state.State
	declAddr           proto.TCPAddr
	bindAddr           proto.TCPAddr
	scheduler          types.Scheduler
	utx                types.UtxPool
	services           services.Services
	microblockInterval time.Duration
	obsolescence       time.Duration
}

func NewNode(
	services services.Services, declAddr proto.TCPAddr, bindAddr proto.TCPAddr, microblockInterval time.Duration,
) *Node {
	if bindAddr.Empty() {
		bindAddr = declAddr
	}
	return &Node{
		state:              services.State,
		peers:              services.Peers,
		declAddr:           declAddr,
		bindAddr:           bindAddr,
		scheduler:          services.Scheduler,
		utx:                services.UtxPool,
		services:           services,
		microblockInterval: microblockInterval,
	}
}

func (a *Node) Close() {
	ch := make(chan struct{})
	a.services.InternalChannel <- messages.NewHaltMessage(ch)
	<-ch
}

func (a *Node) SpawnOutgoingConnections(ctx context.Context) {
	a.peers.SpawnOutgoingConnections(ctx)
}

func (a *Node) SpawnOutgoingConnection(ctx context.Context, addr proto.TCPAddr) error {
	return a.peers.Connect(ctx, addr)
}

func (a *Node) serveIncomingPeers(ctx context.Context) error {
	// if empty declared address, listen on port doesn't make sense
	if a.declAddr.Empty() {
		zap.S().Warn("Declared address is empty")
		return nil
	}

	if a.bindAddr.Empty() {
		zap.S().Warn("Bind address is empty")
		return nil
	}

	zap.S().Infof("Start listening on %s", a.bindAddr.String())
	var lc net.ListenConfig
	l, err := lc.Listen(ctx, "tcp", a.bindAddr.String())
	if err != nil {
		return err
	}
	defer func() {
		if err := l.Close(); err != nil {
			zap.S().Errorf("Failed to close %T on addr %q: %v", l, l.Addr().String(), err)
		}
	}()

	// TODO: implement good graceful shutdown
	for {
		conn, err := l.Accept()
		if err != nil {
			if ctx.Err() != nil { // context has been canceled
				return nil
			}
			zap.S().Errorf("Failed to accept new peer: %v", err)
			continue
		}

		go func() {
			if err := a.peers.SpawnIncomingConnection(ctx, conn); err != nil {
				zap.S().Named(logging.NetworkNamespace).Debugf("Incoming connection failed with addr %q: %v",
					conn.RemoteAddr().String(), err)
				return
			}
		}()
	}
}

func (a *Node) logErrors(err error) {
	var infoMsg *proto.InfoMsg
	_ = error(infoMsg) // compile time check
	switch {
	case errors.As(err, &infoMsg):
		zap.S().Named(logging.FSMNamespace).Debugf("Error: %v", infoMsg)
	default:
		zap.S().Errorf("%v", err)
	}
}

func (a *Node) Run(
	ctx context.Context, p peer.Parent, internalMessageCh <-chan messages.InternalMessage,
	networkMsgCh <-chan network.InfoMessage, syncPeer *network.SyncPeer,
) {
	go a.runOutgoingConnections(ctx)
	go a.runInternalMetrics(ctx, p.MessageCh)
	go a.runIncomingConnections(ctx)

	tasksCh := make(chan tasks.AsyncTask, 10)

	// TODO: Consider using context `ctx` in FSM, for now FSM works in the background context.
	m, async, err := fsm.NewFSM(a.services, a.microblockInterval, a.obsolescence, syncPeer)
	if err != nil {
		zap.S().Errorf("Failed to create FSM: %v", err)
		return
	}
	spawnAsync(ctx, tasksCh, a.services.LoggableRunner, async)
	actions := createActions()

	for {
		select {
		case internalMess := <-internalMessageCh:
			switch t := internalMess.(type) {
			case *messages.MinedBlockInternalMessage:
				async, err = m.MinedBlock(t.Block, t.Limits, t.KeyPair, t.Vrf)
			case *messages.HaltMessage:
				async, err = m.Halt()
				t.Complete()
			case *messages.BroadcastTransaction:
				async, err = m.Transaction(nil, t.Transaction)
				select {
				case t.Response <- err:
				default:
				}
			default:
				zap.S().Errorf("[%s] Unknown internal message '%T'", m.State.State, t)
				continue
			}
		case task := <-tasksCh:
			async, err = m.Task(task)
		case msg := <-networkMsgCh:
			switch t := msg.(type) {
			case network.StartMining:
				async, err = m.StartMining()
			case network.StopSync:
				async, err = m.StopSync()
			case network.ChangeSyncPeer:
				async, err = m.ChangeSyncPeer(t.Peer)
			case network.StopMining:
				async, err = m.StopMining()
			default:
				zap.S().Warnf("[%s] Unknown network info message '%T'", m.State.State, msg)
			}
		case mess := <-p.MessageCh:
			zap.S().Named(logging.FSMNamespace).Debugf("[%s] Network message '%T' received from '%s'",
				m.State.State, mess.Message, mess.ID.ID())
			action, ok := actions[reflect.TypeOf(mess.Message)]
			if !ok {
				zap.S().Errorf("[%s] Unknown network message '%T' from '%s'",
					m.State.State, mess.Message, mess.ID.ID())
				continue
			}
			async, err = action(a.services, mess, m)
		}
		if err != nil {
			a.logErrors(err)
		}
		spawnAsync(ctx, tasksCh, a.services.LoggableRunner, async)
	}
}

func (a *Node) runIncomingConnections(ctx context.Context) {
	if err := a.serveIncomingPeers(ctx); err != nil && !errors.Is(err, context.Canceled) {
		zap.S().Errorf("Failed to continue serving incoming peers: %v", err)
	}
}

func (a *Node) runInternalMetrics(ctx context.Context, ch chan peer.ProtoMessage) {
	for {
		timer := time.NewTimer(metricInternalChannelSizeUpdateInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
			metricInternalChannelSize.Set(float64(len(ch)))
		}
	}
}

func (a *Node) runOutgoingConnections(ctx context.Context) {
	for {
		a.SpawnOutgoingConnections(ctx)
		timer := time.NewTimer(spawnOutgoingConnectionsInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func spawnAsync(ctx context.Context, ch chan tasks.AsyncTask, r runner.LogRunner, a fsm.Async) {
	for _, t := range a {
		func(t tasks.Task) {
			r.Named(fmt.Sprintf("Async Task %T", t), func() {
				err := t.Run(ctx, ch)
				if err != nil && !errors.Is(err, context.Canceled) {
					zap.S().Warnf("Async task '%T' finished with error: %q", t, err)
				}
			})
		}(t)
	}
}
