package state

import (
	"bytes"
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type verifyTaskType byte

const (
	verifyBlock verifyTaskType = iota
	verifyTx
)

type verifierChans struct {
	errChan   chan error
	tasksChan chan *verifyTask
}

func newVerifierChans() *verifierChans {
	return &verifierChans{make(chan error), make(chan *verifyTask)}
}

type verifyTask struct {
	taskType       verifyTaskType
	parentSig      crypto.Signature
	block          *proto.Block
	tx             proto.Transaction
	checkTxSig     bool
	checkSellOrder bool
	checkBuyOrder  bool
}

func checkTx(tx proto.Transaction, checkTxSig, checkSellOrder, checkBuyOrder bool, scheme proto.Scheme) error {
	if ok, err := tx.Valid(); !ok {
		return errors.Wrap(err, "invalid tx data")
	}
	if !checkTxSig {
		return nil
	}
	switch t := tx.(type) {
	case *proto.Genesis:
	case *proto.Payment:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("payment tx signature verification failed")
		}
	case *proto.TransferWithSig:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("transfer tx signature verification failed")
		}
	case *proto.TransferWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("transfer tx signature verification failed")
		}
	case *proto.IssueWithSig:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("issue tx signature verification failed")
		}
	case *proto.IssueWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("issue tx signature verification failed")
		}
	case *proto.ReissueWithSig:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("reissue tx signature verification failed")
		}
	case *proto.ReissueWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("reissue tx signature verification failed")
		}
	case *proto.BurnWithSig:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("burn tx signature verification failed")
		}
	case *proto.BurnWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("burn tx signature verification failed")
		}
	case *proto.ExchangeWithSig:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("exchange tx signature verification failed")
		}
		if checkSellOrder {
			if ok, _ := t.SellOrder.Verify(scheme, t.SellOrder.SenderPK); !ok {
				return errors.New("sell order signature verification failed")
			}
		}
		if checkBuyOrder {
			if ok, _ := t.BuyOrder.Verify(scheme, t.BuyOrder.SenderPK); !ok {
				return errors.New("buy order signature verification failed")
			}
		}
	case *proto.ExchangeWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("exchange tx signature verification failed")
		}
		if checkSellOrder {
			if ok, _ := t.SellOrder.Verify(scheme, t.SellOrder.GetSenderPK()); !ok {
				return errors.New("sell order signature verification failed")
			}
		}
		if checkBuyOrder {
			if ok, _ := t.BuyOrder.Verify(scheme, t.BuyOrder.GetSenderPK()); !ok {
				return errors.New("buy order signature verification failed")
			}
		}
	case *proto.LeaseWithSig:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("lease tx signature verification failed")
		}
	case *proto.LeaseWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("lease tx signature verification failed")
		}
	case *proto.LeaseCancelWithSig:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("leasecancel tx signature verification failed")
		}
	case *proto.LeaseCancelWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("leasecancel tx signature verification failed")
		}
	case *proto.CreateAliasWithSig:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("createalias tx signature verification failed")
		}
	case *proto.CreateAliasWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("createalias tx signature verification failed")
		}
	case *proto.SponsorshipWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("sponsorship tx signature verification failed")
		}
	case *proto.MassTransferWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("masstransfer tx signature verification failed")
		}
	case *proto.DataWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("data tx signature verification failed")
		}
	case *proto.SetScriptWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("setscript tx signature verification failed")
		}
	case *proto.SetAssetScriptWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("setassetscript tx signature verification failed")
		}
	case *proto.InvokeScriptWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("invokescript tx signature verification failed")
		}
	case *proto.UpdateAssetInfoWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("updateassetinfo tx signature verification failed")
		}
	default:
		return errors.New("unknown transaction type")
	}
	return nil
}

func handleTask(task *verifyTask, scheme proto.Scheme) error {
	switch task.taskType {
	case verifyBlock:
		// Check parent.
		if !bytes.Equal(task.parentSig[:], task.block.Parent[:]) {
			return errors.Errorf("incorrect parent: want: %s, have: %s", task.parentSig.String(), task.block.Parent.String())
		}
		// Check block signature and transactions root hash if applied
		validSig, err := task.block.VerifySignature(scheme)
		if err != nil {
			return errors.Wrap(err, "State: handleTask: failed to verify block signature")
		}
		if !validSig {
			return errors.New("State: handleTask: invalid block signature")
		}
		validRootHash, err := task.block.VerifyTransactionsRoot(scheme)
		if err != nil {
			return errors.Wrap(err, "State: handleTask: failed to verify transactions root hash")
		}
		if !validRootHash {
			return errors.New("State: handleTask: invalid transaction root hash")
		}
	case verifyTx:
		if err := checkTx(task.tx, task.checkTxSig, task.checkSellOrder, task.checkBuyOrder, scheme); err != nil {
			return err
		}
	}
	return nil
}

func verify(ctx context.Context, tasks <-chan *verifyTask, scheme proto.Scheme) error {
	for task := range tasks {
		if err := handleTask(task, scheme); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}
	return nil
}

func launchVerifier(ctx context.Context, chans *verifierChans, goroutinesNum int, scheme proto.Scheme) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < goroutinesNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := verify(ctx, chans.tasksChan, scheme); err != nil {
				chans.errChan <- err
				cancel()
			}
		}()
	}
	wg.Wait()
	close(chans.errChan)
}
