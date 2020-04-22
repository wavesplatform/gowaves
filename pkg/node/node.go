package node

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"time"

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

type Config struct {
	AppName  string
	NodeName string
	Listen   string
	DeclAddr string
}

type Node struct {
	peers     peer_manager.PeerManager
	state     state.State
	declAddr  proto.TCPAddr
	bindAddr  proto.TCPAddr
	scheduler types.Scheduler
	utx       types.UtxPool
	services  services.Services
	outdate   proto.Timestamp
}

func NewNode(services services.Services, declAddr proto.TCPAddr, bindAddr proto.TCPAddr, outdate proto.Timestamp) *Node {
	if bindAddr.Empty() {
		bindAddr = declAddr
	}
	return &Node{
		state:     services.State,
		peers:     services.Peers,
		declAddr:  declAddr,
		bindAddr:  bindAddr,
		scheduler: services.Scheduler,
		utx:       services.UtxPool,
		services:  services,
		outdate:   outdate,
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

func (a *Node) Serve(ctx context.Context) error {
	// if empty declared address, listen on port doesn't make sense
	if a.declAddr.Empty() {
		return nil
	}

	if a.bindAddr.Empty() {
		return nil
	}

	zap.S().Info("start listening on ", a.bindAddr.String())
	l, err := net.Listen("tcp", a.bindAddr.String())
	if err != nil {
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			zap.S().Error(err)
			continue
		}

		go func() {
			if err := a.peers.SpawnIncomingConnection(ctx, conn); err != nil {
				zap.S().Error(err)
				return
			}
		}()
	}
}

func (a *Node) Run(ctx context.Context, p peer.Parent, InternalMessageCh chan messages.InternalMessage) {
	go func() {
		for {
			a.SpawnOutgoingConnections(ctx)
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Minute):
			}
		}
	}()

	go func() {
		if err := a.Serve(ctx); err != nil {
			return
		}
	}()

	tasksCh := make(chan tasks.AsyncTask, 10)

	fsm, async, err := state_fsm.NewFsm(a.services, a.outdate)
	if err != nil {
		zap.S().Error(err)
		return
	}
	spawnAsync(ctx, tasksCh, a.services.LoggableRunner, async)
	actions := CreateActions()

	for {
		select {
		case internalMess := <-InternalMessageCh:
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
				zap.S().Errorf("unknown internalMess %T", t)
				continue
			}
		case task := <-tasksCh:
			fsm, async, err = fsm.Task(task)
		case m := <-p.InfoCh:
			switch t := m.Value.(type) {
			case *peer.Connected:
				fsm, async, err = fsm.NewPeer(t.Peer)
			case error:
				zap.S().Error("infoCH error ", m.Peer, t)
				fsm, async, err = fsm.PeerError(m.Peer, t)
			}
		case mess := <-p.MessageCh:
			zap.S().Debugf("received proto Message %T", mess.Message)
			action, ok := actions[reflect.TypeOf(mess.Message)]
			if !ok {
				zap.S().Errorf("unknown proto Message %T", mess.Message)
				continue
			}
			fsm, async, err = action(a.services, mess, fsm)
		}
		if err != nil {
			zap.S().Error(err)
		}
		spawnAsync(ctx, tasksCh, a.services.LoggableRunner, async)
		zap.S().Debugf("fsm %T", fsm)
	}
}

type BlockIds struct {
	ids    []proto.BlockID
	unique map[proto.BlockID]struct{}
}

func (a *BlockIds) Ids() []proto.BlockID {
	return a.ids
}

func NewBlockIds(ids ...proto.BlockID) *BlockIds {
	unique := make(map[proto.BlockID]struct{})
	for _, v := range ids {
		unique[v] = struct{}{}
	}

	return &BlockIds{
		ids:    ids,
		unique: unique,
	}
}

func spawnAsync(ctx context.Context, ch chan tasks.AsyncTask, r runner.LogRunner, a state_fsm.Async) {
	for _, t := range a {
		func(t tasks.Task) {
			r.Named(fmt.Sprintf("Async Task %T", t), func() {
				err := t.Run(ctx, ch)
				if err != nil {
					zap.S().Errorf("Async Task %T, error %q", t, err)
				}
			})
		}(t)
	}
}

func (a *BlockIds) Exists(id proto.BlockID) bool {
	_, ok := a.unique[id]
	return ok
}

func (a *BlockIds) Revert() *BlockIds {
	out := make([]proto.BlockID, len(a.ids))
	for k, v := range a.ids {
		out[len(a.ids)-1-k] = v
	}
	return NewBlockIds(out...)
}
