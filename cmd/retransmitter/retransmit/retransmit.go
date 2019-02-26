package retransmit

import (
	"context"
	"net"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
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
				return &tx, nil
			case byte(proto.TransferTransaction):
				var tx proto.TransferV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.ReissueTransaction):
				var tx proto.ReissueV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.BurnTransaction):
				var tx proto.BurnV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.ExchangeTransaction):
				var tx proto.ExchangeV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.LeaseTransaction):
				var tx proto.LeaseV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.LeaseCancelTransaction):
				var tx proto.LeaseCancelV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.CreateAliasTransaction):
				var tx proto.CreateAliasV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.DataTransaction):
				var tx proto.DataV1
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.SetScriptTransaction):
				var tx proto.SetScriptV1
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.SponsorshipTransaction):
				var tx proto.SponsorshipV1
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, err
				}
				return &tx, nil
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
			return &tx, nil

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
			return &tx, nil
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
			return &tx, nil
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
			return &tx, nil
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
			return &tx, nil
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
			return &tx, nil
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
			return &tx, nil
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
			return &tx, nil
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
			return &tx, nil
		}
	}
	return nil, errors.New("unknown transaction")
}
