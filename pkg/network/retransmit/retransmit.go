package retransmit

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"time"
)

var invalidTransaction = errors.New("invalid transaction")

type PeerInfo struct {
	Peer      peer.Peer
	CreatedAt time.Time
	Status    int
	Version   proto.Version
	LastError struct {
		At    time.Time
		Error error
	}
}

// This function knows how to create new client and encapsulates logic of creating new client inside
type PeerOutgoingSpawner func(peer.OutgoingPeerParams)

// This function knows how to create new client and encapsulates logic of creating new client inside
type PeerIncomingSpawner func(peer.IncomingPeerParams)

type Retransmitter struct {
	spawner                   PeerOutgoingSpawner
	incomingSpawner           PeerIncomingSpawner
	connectedPeers            *Addr2Peers
	income                    chan peer.ProtoMessage
	outgoing                  chan peer.ProtoMessage
	tl                        *TransactionList
	infoCh                    chan peer.InfoMessage
	ctx                       context.Context
	knownPeers                *KnownPeers
	declAddr                  proto.PeerInfo
	receiveFromRemoteCallback peer.ReceiveFromRemoteCallback
	pool                      conn.Pool
	spawnedPeers              *SpawnedPeers
}

func NewRetransmitter(ctx context.Context, knownPeers *KnownPeers, outgoingSpawner PeerOutgoingSpawner, incomingSpawner PeerIncomingSpawner, ReceiveFromRemoteCallback peer.ReceiveFromRemoteCallback, pool conn.Pool) *Retransmitter {
	return &Retransmitter{
		ctx:                       ctx,
		knownPeers:                knownPeers,
		spawner:                   outgoingSpawner,
		incomingSpawner:           incomingSpawner,
		receiveFromRemoteCallback: ReceiveFromRemoteCallback,
		pool:                      pool,

		income:         make(chan peer.ProtoMessage, 100),
		connectedPeers: NewAddr2Peers(),
		outgoing:       make(chan peer.ProtoMessage, 10),
		infoCh:         make(chan peer.InfoMessage, 100),
		tl:             NewTransactionList(6000),
		spawnedPeers:   NewSpawnedPeers(),
	}
}

func (a *Retransmitter) Run() {

	go a.serveSendAllMyKnownPeers()
	go a.askPeersAboutKnownPeers()
	go a.periodicallySpawnPeers()

	for {
		select {
		case <-a.ctx.Done():
			a.knownPeers.Stop()
			a.connectedPeers.Each(func(id string, p *PeerInfo) {
				p.Peer.Close()
			})
			return
		case incomeMessage := <-a.income:

			//fmt.Println("retransmitter got income message ", incomeMessage)

			switch t := incomeMessage.Message.(type) {
			case *proto.TransactionMessage:
				transaction, err := getTransaction(t)
				if err != nil {
					fmt.Println(err)
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
				fmt.Println("got *proto.PeersMessage")
				for _, p := range t.Peers {
					a.knownPeers.Add(p.String(), proto.Version{})
				}
			default:
				zap.S().Warnf("got unknown incomeMessage.Message of type %T\n", incomeMessage.Message)
			}
		case out := <-a.outgoing:
			a.connectedPeers.Each(func(id string, c *PeerInfo) {
				if id != out.ID {
					c.Peer.SendMessage(out.Message)
				}
			})
		case info := <-a.infoCh:
			switch t := info.Value.(type) {
			case error:
				zap.S().Infof("got error message %s from %s", t, info.ID)
				a.errorHandler(info.ID, t)
			case proto.Version:
				a.connectedPeers.Get(info.ID).Version = t
			case *peer.Connected:
				a.connectedPeers.Add(info.ID, &PeerInfo{
					Version: t.Version,
					Peer:    t.Peer,
				})
				a.knownPeers.Add(info.ID, t.Version)
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
		a.knownPeers.Add(id, p.Version)
	}
}

func (a *Retransmitter) AddAddress(addr string) {
	zap.S().Debug("incomeAddrCh", addr)
	if !a.connectedPeers.Exists(addr) && !a.spawnedPeers.Exists(addr) {
		go a.spawnOutgoingPeer(addr)
		a.spawnedPeers.Add(addr)
	}
}

func (a *Retransmitter) handleGetPeersMess(id string) {
	zap.S().Debug("call retransmitter handleGetPeersMess")
	p := a.connectedPeers.Addresses()
	pm := proto.PeersMessage{
		Peers: p,
	}
	c := a.connectedPeers.Get(id)
	if c != nil {
		c.Peer.SendMessage(&pm)
	}
}

// every 5 minutes we should send known peers list
func (a *Retransmitter) serveSendAllMyKnownPeers() {
	for {
		select {
		case <-time.After(5 * time.Minute):
			addrs := a.connectedPeers.Addresses()
			pm := proto.PeersMessage{
				Peers: addrs,
			}
			a.connectedPeers.Each(func(id string, p *PeerInfo) {
				p.Peer.SendMessage(&pm)
			})
		case <-a.ctx.Done():
			return
		}
	}
}

func (a *Retransmitter) askPeersAboutKnownPeers() {
	for {
		<-time.After(1 * time.Minute)
		zap.S().Debug("ask about peers")
		a.connectedPeers.Each(func(id string, p *PeerInfo) {
			p.Peer.SendMessage(&proto.GetPeersMessage{})
		})
	}
}

// listen incoming connections on provided address
func (a *Retransmitter) Server(listenAddr string) error {
	addr, err := proto.NewPeerInfoFromString(listenAddr)
	if err != nil {
		return err
	}
	a.declAddr = addr
	go a.serve(listenAddr)
	return nil
}

func (a *Retransmitter) spawnOutgoingPeer(addr string) {
	parent := peer.Parent{
		ResendToParentCh: a.income,
		ParentInfoChan:   a.infoCh,
	}

	params := peer.OutgoingPeerParams{
		Ctx:                       a.ctx,
		Address:                   addr,
		Parent:                    parent,
		ReceiveFromRemoteCallback: a.receiveFromRemoteCallback,
		Pool:                      a.pool,
		DeclAddr:                  a.declAddr,
	}

	a.spawner(params)
}

func (a *Retransmitter) serve(listenAddr string) {
	lst, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		c, err := lst.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		parent := peer.Parent{
			ResendToParentCh: a.income,
			ParentInfoChan:   a.infoCh,
		}

		params := peer.IncomingPeerParams{
			Ctx:                       a.ctx,
			Conn:                      c,
			ReceiveFromRemoteCallback: a.receiveFromRemoteCallback,
			Parent:                    parent,
			DeclAddr:                  a.declAddr,
			Pool:                      a.pool,
		}

		go a.incomingSpawner(params)
	}
}

func (a *Retransmitter) ActiveConnections() *Addr2Peers {
	return a.connectedPeers
}

func (a *Retransmitter) KnownPeers() *KnownPeers {
	return a.knownPeers
}

func (a *Retransmitter) periodicallySpawnPeers() {
	for {
		select {
		case <-time.After(10 * time.Minute):
			peers := a.knownPeers.GetAll()
			for _, addr := range peers {
				if !a.spawnedPeers.Exists(addr) {
					a.spawnOutgoingPeer(addr)
				}
			}
		case <-a.ctx.Done():
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
