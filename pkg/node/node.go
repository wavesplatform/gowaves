package node

import (
	"context"
	"log/slog"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/node/fsm"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/network"
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
	enableLightMode    bool
	netLogger          *slog.Logger
	fsmLogger          *slog.Logger
}

func NewNode(
	services services.Services, declAddr proto.TCPAddr, bindAddr proto.TCPAddr, microblockInterval time.Duration,
	enableLightMode bool, netLogger, fsmLogger *slog.Logger,
) *Node {
	if bindAddr.EmptyNoPort() {
		slog.Warn("Bind IP address and port are empty, using declared address", "address", declAddr.String())
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
		enableLightMode:    enableLightMode,
		netLogger:          netLogger,
		fsmLogger:          fsmLogger,
	}
}

func (a *Node) Close() error {
	ch := make(chan struct{})
	a.services.InternalChannel <- messages.NewHaltMessage(ch)
	<-ch
	return nil
}

func (a *Node) SpawnOutgoingConnections(ctx context.Context) {
	a.peers.SpawnOutgoingConnections(ctx)
}

func (a *Node) serveIncomingPeers(ctx context.Context) error {
	var wg sync.WaitGroup
	defer wg.Wait()
	// it's important defer wg.Wait before deferring the context cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// if empty declared address, listen on port doesn't make sense
	if a.declAddr.Empty() {
		slog.Warn("Declared IP address is empty")
		return nil
	}

	if a.bindAddr.EmptyNoPort() {
		slog.Warn("Bind IP address and port are empty")
		return nil
	}

	slog.Info("Start listening", "address", a.bindAddr.String())
	var lc net.ListenConfig
	l, err := lc.Listen(ctx, "tcp", a.bindAddr.String())
	if err != nil {
		return err
	}

	// Close the listener when the context is done
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		if clErr := l.Close(); clErr != nil {
			slog.Error("Failed to close listener", slog.String("address", l.Addr().String()),
				logging.Error(clErr))
		}
	}()

	for {
		conn, acErr := l.Accept()
		if acErr != nil {
			if ctx.Err() != nil { // context has been canceled
				return nil
			}
			slog.Error("Failed to accept new peer", logging.Error(acErr))
			continue
		}

		go func() {
			if sErr := a.peers.SpawnIncomingConnection(ctx, conn); sErr != nil {
				a.netLogger.Debug("Failed to establish incoming connection",
					slog.String("address", conn.RemoteAddr().String()), logging.Error(sErr))
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
		a.netLogger.Debug("Node failure", logging.Error(infoMsg))
	default:
		slog.Error("Node failure", logging.Error(err))
	}
}

func (a *Node) Run(
	ctx context.Context, p peer.Parent, internalMessageCh <-chan messages.InternalMessage,
	networkMsgCh <-chan network.InfoMessage, syncPeer *network.SyncPeer,
) {
	messageCh, protoMessagesLenProvider, wg := deduplicateProtoTxMessages(ctx, p.MessageCh)
	defer wg.Wait()

	go a.runOutgoingConnections(ctx)
	go a.runInternalMetrics(ctx, protoMessagesLenProvider)
	go a.runIncomingConnections(ctx)

	tasksCh := make(chan tasks.AsyncTask, 10)

	// TODO: Consider using context `ctx` in FSM, for now FSM works in the background context.
	m, async, err := fsm.NewFSM(a.services, a.microblockInterval, a.obsolescence, syncPeer, a.enableLightMode,
		a.fsmLogger, a.netLogger)
	if err != nil {
		slog.Error("Failed to create FSM", logging.Error(err))
		return
	}
	spawnAsync(ctx, tasksCh, async)
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
				slog.Error("Unknown internal message", slog.Any("state", m.State.State), logging.Type(t))
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
				slog.Warn("Unknown network info message", slog.Any("state", m.State.State), logging.Type(msg))
			}
		case mess := <-messageCh:
			a.fsmLogger.Debug("Network message received", slog.Any("state", m.State.State),
				logging.Type(mess.Message), slog.Any("peer", mess.ID.ID()))
			action, ok := actions[reflect.TypeOf(mess.Message)]
			if !ok {
				slog.Error("Unknown network message", slog.Any("state", m.State.State),
					logging.Type(mess.Message), slog.Any("peer", mess.ID.ID()))
				continue
			}
			async, err = action(a.services, mess, m, a.netLogger)
		}
		if err != nil {
			a.logErrors(err)
		}
		spawnAsync(ctx, tasksCh, async)
	}
}

func (a *Node) runIncomingConnections(ctx context.Context) {
	if err := a.serveIncomingPeers(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("Failed to continue serving incoming peers", logging.Error(err))
	}
}

type lenProvider interface {
	Len() int
}

func (a *Node) runInternalMetrics(ctx context.Context, protoMessagesChan lenProvider) {
	ticker := time.NewTicker(metricInternalChannelSizeUpdateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l := protoMessagesChan.Len()
			metrics.FSMChannelLength(l)
			metricInternalChannelSize.Set(float64(l))
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
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
		}
	}
}

func spawnAsync(ctx context.Context, ch chan tasks.AsyncTask, a fsm.Async) {
	for _, t := range a {
		go func(t tasks.Task) {
			err := t.Run(ctx, ch)
			if err != nil && !errors.Is(err, context.Canceled) {
				slog.Warn("Async task finished with error", logging.Type(t), logging.Error(err))
			}
		}(t)
	}
}
