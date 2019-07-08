package retransmit

import (
	"context"
	"net"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

var invalidTransaction = errors.New("invalid transaction")

// Base struct that makes transaction transmit
type Retransmitter struct {
	behaviour *BehaviourImpl
	parent    peer.Parent
}

// creates new Retransmitter
func NewRetransmitter(behaviour *BehaviourImpl, parent peer.Parent) *Retransmitter {
	return &Retransmitter{
		behaviour: behaviour,
		parent:    parent,
	}
}

// this function starts main process of transaction transmitting
func (a *Retransmitter) Run(ctx context.Context) {

	go a.serveSendAllMyKnownPeers(ctx, time.NewTicker(5*time.Minute))
	go a.askPeersAboutKnownPeers(ctx, time.NewTicker(1*time.Minute))
	go a.periodicallySpawnPeers(ctx, time.NewTicker(1*time.Minute))

	// handle messages simultaneously
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					a.behaviour.Stop()
					return
				case incomeMessage := <-a.parent.MessageCh:
					a.behaviour.ProtoMessage(incomeMessage)
				}
			}
		}()
	}

	// handle errors simultaneously
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case info := <-a.parent.InfoCh:
					a.behaviour.InfoMessage(info)
				}
			}
		}()
	}
}

func (a *Retransmitter) Address(ctx context.Context, addr string) {
	a.behaviour.Address(ctx, addr)
}

// send known peers list
func (a *Retransmitter) serveSendAllMyKnownPeers(ctx context.Context, ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			a.behaviour.SendAllMyKnownPeers()
		case <-ctx.Done():
			return
		}
	}
}

// ask peers about knows addresses
func (a *Retransmitter) askPeersAboutKnownPeers(ctx context.Context, ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			a.behaviour.AskAboutKnownPeers()
		case <-ctx.Done():
			return
		}
	}
}

// listen incoming connections on provided address
func (a *Retransmitter) ServeIncomingConnections(ctx context.Context, listenAddr string) error {
	_, err := proto.NewPeerInfoFromString(listenAddr)
	if err != nil {
		return err
	}
	go a.serve(ctx, listenAddr)
	return nil
}

func (a *Retransmitter) serve(ctx context.Context, listenAddr string) {
	lst, err := net.Listen("tcp", listenAddr)
	if err != nil {
		zap.S().Error(err)
		return
	}
	zap.S().Infof("started listen on %s", listenAddr)

	for {
		c, err := lst.Accept()
		if err != nil {
			zap.S().Error(err)
			continue
		}

		go a.behaviour.IncomeConnection(ctx, c)
	}
}

func (a *Retransmitter) periodicallySpawnPeers(ctx context.Context, ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			a.behaviour.SpawnKnownPeers(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func verifySig(transaction proto.Transaction) (bool, error) {
	if !transaction.CanVerifySignatureWithoutState() {
		return true, nil
	}
	switch tx := transaction.(type) {
	case *proto.IssueV1:
		return tx.Verify(tx.SenderPK)
	case *proto.TransferV1:
		return tx.Verify(tx.SenderPK)
	case *proto.ReissueV1:
		return tx.Verify(tx.SenderPK)
	case *proto.BurnV1:
		return tx.Verify(tx.SenderPK)
	case *proto.ExchangeV1:
		return tx.Verify(tx.SenderPK)
	case *proto.LeaseV1:
		return tx.Verify(tx.SenderPK)
	case *proto.LeaseCancelV1:
		return tx.Verify(tx.SenderPK)
	case *proto.CreateAliasV1:
		return tx.Verify(tx.SenderPK)
	case *proto.MassTransferV1:
		return tx.Verify(tx.SenderPK)
	}
	return false, errors.Errorf("invalid transaction %T %+v ", transaction, transaction)
}
