package node

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
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
	peers              peer_manager.PeerManager
	state              state.State
	declAddr           proto.TCPAddr
	bindAddr           proto.TCPAddr
	scheduler          types.Scheduler
	utx                types.UtxPool
	services           services.Services
	microblockInterval time.Duration
}

func NewNode(services services.Services, declAddr proto.TCPAddr, bindAddr proto.TCPAddr, microblockInterval time.Duration) *Node {
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
				zap.S().Debugf("Incoming connection failed with addr %q: %v", conn.RemoteAddr().String(), err)
				return
			}
		}()
	}
}

func (a *Node) logErrors(err error) {
	switch e := err.(type) {
	case *proto.InfoMsg:
		zap.S().Debugf("%s", e.Error())
	default:
		zap.S().Errorf("%s", e.Error())
	}
}

func (a *Node) Run(ctx context.Context, p peer.Parent, internalMessageCh <-chan messages.InternalMessage) {
	go func() {
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
	}()

	go func() {
		for {
			timer := time.NewTimer(metricInternalChannelSizeUpdateInterval)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return
			case <-timer.C:
				metricInternalChannelSize.Set(float64(len(p.MessageCh)))
			}
		}
	}()

	go func() {
		if err := a.serveIncomingPeers(ctx); err != nil {
			return
		}
	}()

	tasksCh := make(chan tasks.AsyncTask, 10)

	fsm, async, err := state_fsm.NewFsm(a.services, a.microblockInterval)
	if err != nil {
		zap.S().Errorf("Failed to : %v", err)
		return
	}
	spawnAsync(ctx, tasksCh, a.services.LoggableRunner, async)
	actions := createActions()

	// TODO: implement graceful shutdown
	_ = fmt.Stringer(fsm) // check that fsm implements fmt.Stringer interface
	for {
		select {
		case internalMess := <-internalMessageCh:
			switch t := internalMess.(type) {
			case *messages.MinedBlockInternalMessage:
				fsm, async, err = fsm.MinedBlock(t.Block, t.Limits, t.KeyPair, t.Vrf)
			case *messages.HaltMessage:
				fsm, async, err = fsm.Halt()
				t.Complete()
			case *messages.BroadcastTransaction:
				fsm, async, err = fsm.Transaction(nil, t.Transaction)
				select {
				case t.Response <- err:
				default:
				}
			default:
				zap.S().Errorf("[%s] Unknown internal message '%T'", fsm, t)
				continue
			}
		case task := <-tasksCh:
			fsm, async, err = fsm.Task(task)
		case m := <-p.InfoCh:
			switch t := m.Value.(type) {
			case *peer.Connected:
				fsm, async, err = fsm.NewPeer(t.Peer)
				if err == nil {
					zap.S().Debugf("[%s] Established connection with %s peer '%s'", fsm, t.Peer.Direction(), t.Peer.ID())
				}
			case *peer.InternalErr:
				fsm, async, err = fsm.PeerError(m.Peer, t.Err)
			default:
				zap.S().Warnf("[%s] Unknown info message '%T'", fsm, m)
			}
		case mess := <-p.MessageCh:
			zap.S().Debugf("[%s] Network message '%T' received from '%s'", fsm, mess.Message, mess.ID.ID())
			action, ok := actions[reflect.TypeOf(mess.Message)]
			if !ok {
				zap.S().Errorf("[%s] Unknown network message '%T' from '%s'", fsm, mess.Message, mess.ID.ID())
				continue
			}
			fsm, async, err = action(a.services, mess, fsm)
		}
		if err != nil {
			a.logErrors(err)
		}
		spawnAsync(ctx, tasksCh, a.services.LoggableRunner, async)
	}
}

func spawnAsync(ctx context.Context, ch chan tasks.AsyncTask, r runner.LogRunner, a state_fsm.Async) {
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
