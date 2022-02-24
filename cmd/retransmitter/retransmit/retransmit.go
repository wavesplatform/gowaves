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

	go a.serveSendAllMyKnownPeers(ctx, 5*time.Minute)
	go a.askPeersAboutKnownPeers(ctx, 1*time.Minute)
	go a.periodicallySpawnPeers(ctx, 1*time.Minute)

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
func (a *Retransmitter) serveSendAllMyKnownPeers(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
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
func (a *Retransmitter) askPeersAboutKnownPeers(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
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

func (a *Retransmitter) periodicallySpawnPeers(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.behaviour.SpawnKnownPeers(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func getTransaction(message proto.Message, scheme proto.Scheme) (proto.Transaction, error) {
	switch t := message.(type) {
	case *proto.TransactionMessage:
		txb := t.Transaction
		switch txb[0] {
		case 0:
			switch txb[1] {
			case byte(proto.IssueTransaction):
				var tx proto.IssueWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.TransferTransaction):
				var tx proto.TransferWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.ReissueTransaction):
				var tx proto.ReissueWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.BurnTransaction):
				var tx proto.BurnWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.ExchangeTransaction):
				var tx proto.ExchangeWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.LeaseTransaction):
				var tx proto.LeaseWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.LeaseCancelTransaction):
				var tx proto.LeaseCancelWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.CreateAliasTransaction):
				var tx proto.CreateAliasWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.DataTransaction):
				var tx proto.DataWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.SetScriptTransaction):
				var tx proto.SetScriptWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.SponsorshipTransaction):
				var tx proto.SponsorshipWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.SetAssetScriptTransaction):
				var tx proto.SetAssetScriptWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			case byte(proto.InvokeScriptTransaction):
				var tx proto.InvokeScriptWithProofs
				err := tx.UnmarshalBinary(txb, scheme)
				if err != nil {
					return nil, err
				}
				return &tx, nil
			default:
				return nil, errors.New("unknown or unsupported transaction")
			}

		case byte(proto.IssueTransaction):
			var tx proto.IssueWithSig
			err := tx.UnmarshalBinary(txb, scheme)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(scheme, tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return &tx, nil

		case byte(proto.TransferTransaction):
			var tx proto.TransferWithSig
			err := tx.UnmarshalBinary(txb, scheme)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(scheme, tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return &tx, nil
		case byte(proto.ReissueTransaction):
			var tx proto.ReissueWithSig
			err := tx.UnmarshalBinary(txb, scheme)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(scheme, tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return &tx, nil
		case byte(proto.BurnTransaction):
			var tx proto.BurnWithSig
			err := tx.UnmarshalBinary(txb, scheme)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(scheme, tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return &tx, nil
		case byte(proto.ExchangeTransaction):
			var tx proto.ExchangeWithSig
			err := tx.UnmarshalBinary(txb, scheme)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(scheme, tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return &tx, nil
		case byte(proto.LeaseTransaction):
			var tx proto.LeaseWithSig
			err := tx.UnmarshalBinary(txb, scheme)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(scheme, tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return &tx, nil
		case byte(proto.LeaseCancelTransaction):
			var tx proto.LeaseCancelWithSig
			err := tx.UnmarshalBinary(txb, scheme)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(scheme, tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return &tx, nil
		case byte(proto.CreateAliasTransaction):
			var tx proto.CreateAliasWithSig
			err := tx.UnmarshalBinary(txb, scheme)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(scheme, tx.SenderPK)
			if err != nil {
				return nil, err
			}

			if !valid {
				return nil, invalidTransaction
			}
			return &tx, nil
		case byte(proto.MassTransferTransaction):
			var tx proto.MassTransferWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			if err != nil {
				return nil, err
			}
			valid, err := tx.Verify(scheme, tx.SenderPK)
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
