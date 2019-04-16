package retransmit

import (
	"context"
	"net"
	"sync"

	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/utils"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type BehaviourImpl struct {
	mu                sync.RWMutex
	tl                *TransactionList
	knownPeers        *utils.KnownPeers
	counter           *utils.Counter
	activeConnections *utils.Addr2Peers
	spawnedPeers      *utils.SpawnedPeers
	peerSpawner       PeerSpawner
}

func NewBehaviour(knownPeers *utils.KnownPeers, peerSpawner PeerSpawner) *BehaviourImpl {
	return &BehaviourImpl{
		tl:                NewTransactionList(6000),
		knownPeers:        knownPeers,
		counter:           utils.NewCounter(),
		activeConnections: utils.NewAddr2Peers(),
		spawnedPeers:      utils.NewSpawnedPeers(),
		peerSpawner:       peerSpawner,
	}
}

func (a *BehaviourImpl) ProtoMessage(incomeMessage peer.ProtoMessage) {
	switch t := incomeMessage.Message.(type) {
	case *proto.TransactionMessage:
		transaction, err := getTransaction(t)
		if err != nil {
			zap.S().Error(err, incomeMessage.ID, t)
			return
		}

		if !a.tl.Exists(transaction) {
			a.tl.Add(transaction)
			a.counter.IncUniqueTransaction()
			a.activeConnections.Each(func(id string, c Peer) {
				if id != incomeMessage.ID {
					c.SendMessage(incomeMessage.Message)
					a.counter.IncEachTransaction()
				}
			})
		}

	case *proto.GetPeersMessage:
		a.sendToPeerMyKnownHosts(incomeMessage.ID)
	case *proto.PeersMessage:
		zap.S().Debugf("got *proto.PeersMessage, from %s len=%d", incomeMessage.ID, len(t.Peers))
		for _, p := range t.Peers {
			a.knownPeers.Add(proto.NewTCPAddr(p.Addr, int(p.Port)), proto.Version{})
		}
	default:
		zap.S().Warnf("got unknown incomeMessage.Message of type %T\n", incomeMessage.Message)
	}
}

func (a *BehaviourImpl) Stop() {
	a.knownPeers.Stop()
	a.activeConnections.Each(func(id string, p Peer) {
		p.Close()
	})
	a.counter.Stop()
}

func (a *BehaviourImpl) InfoMessage(info peer.InfoMessage) {
	switch t := info.Value.(type) {
	case error:
		zap.S().Infof("got error message %s from %s", t, info.ID)
		a.errorHandler(info.ID, t)
	case *peer.Connected:
		a.activeConnections.Add(t.Peer.ID(), t.Peer)
		if !t.Peer.Handshake().DeclaredAddr.Empty() {
			a.knownPeers.Add(proto.TCPAddr(t.Peer.Handshake().DeclaredAddr), t.Peer.Handshake().Version)
		}
	default:
		zap.S().Warnf("got unknown info message of type %T\n", info.Value)
	}
}

func (a *BehaviourImpl) AskAboutKnownPeers() {
	zap.S().Debug("ask about peers")
	a.activeConnections.Each(func(id string, p Peer) {
		p.SendMessage(&proto.GetPeersMessage{})
	})
}

func (a *BehaviourImpl) sendToPeerMyKnownHosts(id string) {
	p := a.knownPeers.Addresses()
	pm := proto.PeersMessage{
		Peers: p,
	}
	c := a.activeConnections.Get(id)
	if c != nil {
		c.SendMessage(&pm)
	}
}

func (a *BehaviourImpl) SendAllMyKnownPeers() {
	pm := proto.PeersMessage{
		Peers: a.knownPeers.Addresses(),
	}
	a.activeConnections.Each(func(id string, p Peer) {
		p.SendMessage(&pm)
	})
}

func (a *BehaviourImpl) SpawnKnownPeers(ctx context.Context) {
	peers := a.knownPeers.GetAll()
	for _, addr := range peers {
		if a.activeConnections.Exists(addr) {
			continue
		}

		if !a.spawnedPeers.Exists(addr) {
			a.spawnedPeers.Add(addr)
			go a.spawnOutgoingPeer(ctx, addr)
		}
	}
}

func (a *BehaviourImpl) errorHandler(id string, e error) {
	p := a.activeConnections.Get(id)
	if p != nil {
		p.Close()
		a.activeConnections.Delete(id)
	}
}

func (a *BehaviourImpl) Address(ctx context.Context, addr string) {
	a.knownPeers.Add(proto.NewTCPAddrFromString(addr), proto.Version{})
	if !a.activeConnections.Exists(addr) && !a.spawnedPeers.Exists(addr) {
		go a.spawnOutgoingPeer(ctx, addr)
		a.spawnedPeers.Add(addr)
	}
}

func (a *BehaviourImpl) spawnOutgoingPeer(ctx context.Context, addr string) {
	// unsubscribe from spawned peer on exit
	defer a.spawnedPeers.Delete(addr)
	a.peerSpawner.SpawnOutgoing(ctx, addr)
}

func (a *BehaviourImpl) IncomeConnection(ctx context.Context, c net.Conn) {
	a.peerSpawner.SpawnIncoming(ctx, c)
}

func (a *BehaviourImpl) ActiveConnections() *utils.Addr2Peers {
	return a.activeConnections
}

func (a *BehaviourImpl) Counter() *utils.Counter {
	return a.counter
}

func (a *BehaviourImpl) KnownPeers() *utils.KnownPeers {
	return a.knownPeers
}
func (a *BehaviourImpl) SpawnedPeers() *utils.SpawnedPeers {
	return a.spawnedPeers
}
