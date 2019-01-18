package retransmit

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"time"
)

var invalidTransaction = errors.New("invalid transaction")

type Storage interface {
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
}

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

// This function knows how to create new client, it should take as less as possible arguments
// and encapsulate logic of creating new client inside
type PeerOutgoingSpawner func(addr string, resendToParentCh chan peer.IdentifiedMessage, infoCh chan peer.IdentifiedInfo) peer.Peer

// This function knows how to create new client, it should take as less as possible arguments
// and encapsulate logic of creating new client inside
type PeerIncomingSpawner func(conn net.Conn, resendToParentCh chan peer.IdentifiedMessage, infoCh chan peer.IdentifiedInfo) peer.Peer

// This function should decide what we do when connection closed by error
// It can reconnect, close connection or whatever else
type ErrorHandler func(*PeerInfo, *Retransmitter)

type Retransmitter struct {
	spawner          PeerOutgoingSpawner
	incomingSpawner  PeerIncomingSpawner
	addrToPeer       *Addr2Peers
	income           chan peer.IdentifiedMessage
	outgoing         chan peer.IdentifiedMessage
	tl               *TransactionList
	errorHandlerFunc ErrorHandler
	infoCh           chan peer.IdentifiedInfo
	incomingPeerCh   chan peer.Peer
	ctx              context.Context
}

func NewRetransmitter(ctx context.Context, outgoingSpawner PeerOutgoingSpawner, incomingSpawner PeerIncomingSpawner, policy ErrorHandler) *Retransmitter {
	return &Retransmitter{
		spawner:          outgoingSpawner,
		income:           make(chan peer.IdentifiedMessage, 100),
		addrToPeer:       NewAddr2Peers(),
		errorHandlerFunc: policy,
		outgoing:         make(chan peer.IdentifiedMessage, 10),
		infoCh:           make(chan peer.IdentifiedInfo, 100),
		incomingPeerCh:   make(chan peer.Peer, 10),
		incomingSpawner:  incomingSpawner,
		ctx:              ctx,
	}
}

func (a *Retransmitter) Run() {

	go a.serveSendPeer()

	for {
		select {
		case <-a.ctx.Done():
			return
		case peer := <-a.incomingPeerCh:
			if a.addrToPeer.Exists(peer.ID()) {
				// peer already exists
				peer.Close()
				continue
			}

			a.addrToPeer.Add(peer.ID(), &PeerInfo{
				Peer: peer,
			})

		case incomeMessage := <-a.income:

			fmt.Println("Retransmitter got income message ", incomeMessage)

			switch t := incomeMessage.Message.(type) {
			case *proto.TransactionMessage:
				_, err := getTransaction(t)
				if err != nil {
					fmt.Println(err)
					continue
				}

				select {
				case a.outgoing <- incomeMessage:
				default:
				}
			case *proto.GetPeersMessage:
				a.handleGetPeersMess(incomeMessage.ID)
			}
		case out := <-a.outgoing:

			a.addrToPeer.Each(func(id peer.UniqID, c *PeerInfo) {
				if id != out.ID {
					c.Peer.SendMessage(out.Message)
				}
			})
		case info := <-a.infoCh:
			switch t := info.Value.(type) {
			case error:
				a.errorHandler(a.addrToPeer.Get(info.ID), t)
			case proto.Version:
				a.addrToPeer.Get(info.ID).Version = t
			}
		}
	}
}

func (a *Retransmitter) errorHandler(peer *PeerInfo, e error) {
	a.errorHandlerFunc(peer, a)
}

func (a *Retransmitter) AddAddress(addr string) {
	fmt.Println("incomeAddrCh", addr)
	if !a.addrToPeer.Exists(peer.UniqID(addr)) {
		c := a.spawner(addr, a.income, a.infoCh)
		a.incomingPeerCh <- c
	}
}

func (a *Retransmitter) handleGetPeersMess(id peer.UniqID) {
	p := a.addrToPeer.Addresses()
	pm := proto.PeersMessage{
		Peers: p,
	}
	c := a.addrToPeer.Get(id)
	if c != nil {
		c.Peer.SendMessage(&pm)
	}
}

// every 5 minutes we should send known peers list
func (a *Retransmitter) serveSendPeer() {
	for {
		select {
		case <-time.After(5 * time.Minute):
			addrs := a.addrToPeer.Addresses()
			pm := proto.PeersMessage{
				Peers: addrs,
			}
			a.addrToPeer.Each(func(id peer.UniqID, p *PeerInfo) {
				p.Peer.SendMessage(&pm)
			})
		case <-a.ctx.Done():
			return
		}
	}
}

func (a *Retransmitter) Server(listenAddr string) {
	go a.serve(listenAddr)
}

func (a *Retransmitter) serve(listenAddr string) {
	lst, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		conn, err := lst.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		a.incomingPeerCh <- a.incomingSpawner(conn, a.income, a.infoCh)
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
