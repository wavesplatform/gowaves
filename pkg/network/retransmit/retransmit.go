package retransmit

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/network/retransmit/utils"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"time"
)

var invalidTransaction = errors.New("invalid transaction")

// This function knows how to create outgoing client and encapsulates logic of creating client inside
type PeerOutgoingSpawner func(peer.OutgoingPeerParams)

// This function knows how to create incoming client and encapsulates logic of creating client inside
type PeerIncomingSpawner func(peer.IncomingPeerParams)

// Base struct that makes transaction transmit
type Retransmitter struct {
	spawner                   PeerOutgoingSpawner
	incomingSpawner           PeerIncomingSpawner
	connectedPeers            *utils.Addr2Peers
	income                    chan peer.ProtoMessage
	outgoing                  chan peer.ProtoMessage
	tl                        *TransactionList
	infoCh                    chan peer.InfoMessage
	knownPeers                *utils.KnownPeers
	declAddr                  proto.PeerInfo
	receiveFromRemoteCallback peer.ReceiveFromRemoteCallback
	pool                      conn.Pool
	spawnedPeers              *utils.SpawnedPeers
	counter                   *utils.Counter
}

// creates new Retransmitter
func NewRetransmitter(declAddr proto.PeerInfo, knownPeers *utils.KnownPeers, counter *utils.Counter, outgoingSpawner PeerOutgoingSpawner, incomingSpawner PeerIncomingSpawner, ReceiveFromRemoteCallback peer.ReceiveFromRemoteCallback, pool conn.Pool) *Retransmitter {
	return &Retransmitter{
		declAddr:                  declAddr,
		knownPeers:                knownPeers,
		spawner:                   outgoingSpawner,
		incomingSpawner:           incomingSpawner,
		receiveFromRemoteCallback: ReceiveFromRemoteCallback,
		pool:                      pool,

		income:         make(chan peer.ProtoMessage, 100),
		connectedPeers: utils.NewAddr2Peers(),
		outgoing:       make(chan peer.ProtoMessage, 10),
		infoCh:         make(chan peer.InfoMessage, 100),
		tl:             NewTransactionList(500),
		spawnedPeers:   utils.NewSpawnedPeers(),
		counter:        counter,
	}
}

// this function starts main process of transaction transmitting
func (a *Retransmitter) Run(ctx context.Context) {

	go a.serveSendAllMyKnownPeers(ctx, 5*time.Minute)
	go a.askPeersAboutKnownPeers(ctx, 1*time.Minute)
	go a.periodicallySpawnPeers(ctx)

	for {
		select {
		case <-ctx.Done():
			a.knownPeers.Stop()
			a.connectedPeers.Each(func(id string, p *utils.PeerInfo) {
				p.Peer.Close()
			})
			return
		case incomeMessage := <-a.income:
			switch t := incomeMessage.Message.(type) {
			case *proto.TransactionMessage:
				transaction, err := getTransaction(t)
				if err != nil {
					zap.S().Error(err, incomeMessage.ID, t)
					continue
				}

				if !a.tl.Exists(transaction) {
					a.tl.Add(transaction)
					select {
					case a.outgoing <- incomeMessage:
					default:
					}
				}

			case *proto.GetPeersMessage:
				a.handleGetPeersMess(incomeMessage.ID)
			case *proto.PeersMessage:
				zap.S().Debugf("got *proto.PeersMessage, from %s len=%d", incomeMessage.ID, len(t.Peers))
				for _, p := range t.Peers {
					a.knownPeers.Add(p, proto.Version{})
				}
			default:
				zap.S().Warnf("got unknown incomeMessage.Message of type %T\n", incomeMessage.Message)
			}
		case out := <-a.outgoing:
			a.counter.IncUniqueTransaction()
			a.connectedPeers.Each(func(id string, c *utils.PeerInfo) {
				if id != out.ID {
					c.Peer.SendMessage(out.Message)
					a.counter.IncEachTransaction()
				}
			})
		case info := <-a.infoCh:
			switch t := info.Value.(type) {
			case error:
				zap.S().Infof("got error message %s from %s", t, info.ID)
				a.errorHandler(info.ID, t)
			case *peer.Connected:
				a.connectedPeers.Add(info.ID, &utils.PeerInfo{
					Version:    t.Version,
					Peer:       t.Peer,
					DeclAddr:   t.DeclAddr,
					RemoteAddr: t.RemoteAddr,
					LocalAddr:  t.LocalAddr,
				})
				if !t.DeclAddr.Empty() {
					a.knownPeers.Add(t.DeclAddr, t.Version)
				}
			default:
				zap.S().Warnf("got unknown info message of type %T\n", info.Value)
			}
		}
	}
}

func (a *Retransmitter) errorHandler(id string, e error) {
	p := a.connectedPeers.Get(id)
	if p != nil {
		p.Peer.Close()
		a.connectedPeers.Delete(id)
	}
}

func (a *Retransmitter) AddAddress(ctx context.Context, addr string) {
	p, err := proto.NewPeerInfoFromString(addr)
	if err != nil {
		zap.S().Error(err)
		return
	}

	a.knownPeers.Add(p, proto.Version{})
	if !a.connectedPeers.Exists(addr) && !a.spawnedPeers.Exists(addr) {
		go a.spawnOutgoingPeer(ctx, addr)
		a.spawnedPeers.Add(addr)
	}
}

func (a *Retransmitter) handleGetPeersMess(id string) {
	zap.S().Debug("call retransmitter handleGetPeersMess")
	p := a.knownPeers.Addresses()
	pm := proto.PeersMessage{
		Peers: p,
	}
	c := a.connectedPeers.Get(id)
	if c != nil {
		c.Peer.SendMessage(&pm)
	}
}

// send known peers list
func (a *Retransmitter) serveSendAllMyKnownPeers(ctx context.Context, interval time.Duration) {
	for {
		select {
		case <-time.After(interval):
			addrs := a.connectedPeers.Addresses()
			pm := proto.PeersMessage{}
			for _, addr := range addrs {
				parsed, err := proto.NewPeerInfoFromString(addr)
				if err != nil {
					zap.S().Warn(err)
					continue
				}
				pm.Peers = append(pm.Peers, parsed)
			}
			a.connectedPeers.Each(func(id string, p *utils.PeerInfo) {
				p.Peer.SendMessage(&pm)
			})
		case <-ctx.Done():
			return
		}
	}
}

// ask peers about knows addresses
func (a *Retransmitter) askPeersAboutKnownPeers(ctx context.Context, interval time.Duration) {
	for {
		select {
		case <-time.After(interval):
			zap.S().Debug("ask about peers")
			a.connectedPeers.Each(func(id string, p *utils.PeerInfo) {
				p.Peer.SendMessage(&proto.GetPeersMessage{})
			})
		case <-ctx.Done():
			return
		}
	}
}

// listen incoming connections on provided address
func (a *Retransmitter) ServeInconingConnections(ctx context.Context, listenAddr string) error {
	_, err := proto.NewPeerInfoFromString(listenAddr)
	if err != nil {
		return err
	}
	go a.serve(ctx, listenAddr)
	return nil
}

func (a *Retransmitter) spawnOutgoingPeer(ctx context.Context, addr string) {
	parent := peer.Parent{
		MessageCh: a.income,
		InfoCh:    a.infoCh,
	}

	params := peer.OutgoingPeerParams{
		Ctx:                       ctx,
		Address:                   addr,
		Parent:                    parent,
		ReceiveFromRemoteCallback: a.receiveFromRemoteCallback,
		Pool:                      a.pool,
		DeclAddr:                  a.declAddr,
		SpawnedPeers:              a.spawnedPeers,
	}

	a.spawner(params)
}

func (a *Retransmitter) serve(ctx context.Context, listenAddr string) {
	lst, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	zap.S().Infof("started listen on %s", listenAddr)

	for {
		c, err := lst.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		parent := peer.Parent{
			MessageCh: a.income,
			InfoCh:    a.infoCh,
		}

		params := peer.IncomingPeerParams{
			Ctx:                       ctx,
			Conn:                      c,
			ReceiveFromRemoteCallback: a.receiveFromRemoteCallback,
			Parent:                    parent,
			DeclAddr:                  a.declAddr,
			Pool:                      a.pool,
		}

		go a.incomingSpawner(params)
	}
}

// returns active connections
func (a *Retransmitter) ActiveConnections() *utils.Addr2Peers {
	return a.connectedPeers
}

// returns knows peers
func (a *Retransmitter) KnownPeers() *utils.KnownPeers {
	return a.knownPeers
}

// returns currently spawned peers
func (a *Retransmitter) SpawnedPeers() *utils.SpawnedPeers {
	return a.spawnedPeers
}

func (a *Retransmitter) Counter() *utils.Counter {
	return a.counter
}

func (a *Retransmitter) periodicallySpawnPeers(ctx context.Context) {
	for {
		select {
		case <-time.After(1 * time.Minute):
			peers := a.knownPeers.GetAll()
			for _, addr := range peers {
				if a.connectedPeers.Exists(addr) {
					continue
				}

				if !a.spawnedPeers.Exists(addr) {
					a.spawnedPeers.Add(addr)
					go a.spawnOutgoingPeer(ctx, addr)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func getTransaction(message proto.Message) (proto.Transaction, error) {
	switch t := message.(type) {
	case *proto.TransactionMessage:
		txb := t.Transaction
		switch txb[0] {
		case 0:
			switch txb[1] {
			case byte(proto.IssueTransaction):
				var tx proto.IssueV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return tx, nil
			case byte(proto.TransferTransaction):
				var tx proto.TransferV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return tx, nil
			case byte(proto.ReissueTransaction):
				var tx proto.ReissueV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return tx, nil
			case byte(proto.BurnTransaction):
				var tx proto.BurnV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return tx, nil
			case byte(proto.ExchangeTransaction):
				var tx proto.ExchangeV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return tx, nil
			case byte(proto.LeaseTransaction):
				var tx proto.LeaseV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return tx, nil
			case byte(proto.LeaseCancelTransaction):
				var tx proto.LeaseCancelV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return tx, nil
			case byte(proto.CreateAliasTransaction):
				var tx proto.CreateAliasV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return tx, nil
			case byte(proto.DataTransaction):
				var tx proto.DataV1
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return tx, nil
			case byte(proto.SetScriptTransaction):
				var tx proto.SetScriptV1
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return tx, nil
			case byte(proto.SponsorshipTransaction):
				var tx proto.SponsorshipV1
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return tx, nil
			default:
				return nil, errors.New("unknown transaction")
			}

		case byte(proto.IssueTransaction):
			var tx proto.IssueV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return tx, nil

		case byte(proto.TransferTransaction):
			var tx proto.TransferV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return tx, nil
		case byte(proto.ReissueTransaction):
			var tx proto.ReissueV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return tx, nil
		case byte(proto.BurnTransaction):
			var tx proto.BurnV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return tx, nil
		case byte(proto.ExchangeTransaction):
			var tx proto.ExchangeV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return tx, nil
		case byte(proto.LeaseTransaction):
			var tx proto.LeaseV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return tx, nil
		case byte(proto.LeaseCancelTransaction):
			var tx proto.LeaseCancelV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return tx, nil
		case byte(proto.CreateAliasTransaction):
			var tx proto.CreateAliasV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return tx, nil
		case byte(proto.MassTransferTransaction):
			var tx proto.MassTransferV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return tx, nil
		}

	}
	return nil, errors.New("unknown transaction")
}
