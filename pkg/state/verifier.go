package state

import (
	"context"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"golang.org/x/sync/errgroup"
)

type verifyTaskType byte

const (
	verifyBlock verifyTaskType = iota + 1
	verifyTx
)

type verifierChans struct {
	errChan   <-chan error
	tasksChan chan<- *verifyTask
}

func (ch *verifierChans) trySend(task *verifyTask) error {
	select {
	case verifyError, ok := <-ch.errChan:
		if !ok {
			return errors.Errorf("sending task with task type (%d) to finished verifier", task.taskType)
		}
		return verifyError
	case ch.tasksChan <- task:
		return nil
	}
}

func (ch *verifierChans) closeAndWait() error {
	close(ch.tasksChan)
	return <-ch.errChan
}

type verifyTask struct {
	taskType    verifyTaskType
	parentID    proto.BlockID
	block       *proto.Block
	tx          proto.Transaction
	checkTxSig  bool
	checkOrder1 bool
	checkOrder2 bool
}

func checkTx(tx proto.Transaction, checkTxSig, checkOrder1, checkOrder2 bool, scheme proto.Scheme) error {
	if _, err := tx.Validate(scheme); err != nil {
		return errs.Extend(err, "invalid tx data")
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
			return errs.NewTxValidationError("transfer tx signature verification failed")
		}
	case *proto.TransferWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errs.NewTxValidationError("transfer tx signature verification failed")
		}
	case *proto.IssueWithSig:
		if ok, err := t.Verify(scheme, t.SenderPK); !ok {
			if err != nil {
				return errs.Extend(err, "issue tx signature verification failed")
			}
			return errors.New("issue tx signature verification failed")
		}
	case *proto.IssueWithProofs:
		if ok, err := t.Verify(scheme, t.SenderPK); !ok {
			if err != nil {
				return errs.Extend(err, "issue tx signature verification failed")
			}
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
		if checkOrder1 {
			if ok, _ := t.Order1.Verify(scheme); !ok {
				return errors.New("first order signature verification failed")
			}
		}
		if checkOrder2 {
			if ok, _ := t.Order2.Verify(scheme); !ok {
				return errors.New("second order signature verification failed")
			}
		}
	case *proto.ExchangeWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("exchange tx signature verification failed")
		}
		if checkOrder1 {
			if ok, _ := t.Order1.Verify(scheme); !ok {
				return errors.New("first order signature verification failed")
			}
		}
		if checkOrder2 {
			if ok, _ := t.Order2.Verify(scheme); !ok {
				return errors.New("second order signature verification failed")
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
			return errs.NewTxValidationError("masstransfer tx signature verification failed")
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
	case *proto.InvokeExpressionTransactionWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("InvokeExpression tx signature verification failed")
		}
	case *proto.UpdateAssetInfoWithProofs:
		if ok, _ := t.Verify(scheme, t.SenderPK); !ok {
			return errors.New("updateassetinfo tx signature verification failed")
		}
	case *proto.EthereumTransaction:
		if _, err := t.Verify(); err != nil {
			return errors.New("ethereumtransaction tx signature verification failed")
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
		if task.parentID != task.block.Parent {
			return errors.Errorf("incorrect parent: want: %s, have: %s", task.parentID.String(), task.block.Parent.String())
		}
		// Check block signature and transactions root hash if applied.
		validSig, err := task.block.VerifySignature(scheme)
		if err != nil {
			return errors.Wrap(err, "State: handleTask: failed to verify block signature")
		}
		if !validSig {
			return errors.Errorf("State: handleTask: invalid block signature (%s) of block '%s'",
				task.block.BlockSignature.String(), task.block.ID.String())
		}
		validRootHash, err := task.block.VerifyTransactionsRoot(scheme)
		if err != nil {
			return errors.Wrap(err, "State: handleTask: failed to verify transactions root hash")
		}
		if !validRootHash {
			return errors.Errorf("State: handleTask: invalid transaction root hash (%s) of block '%s'",
				task.block.TransactionsRoot.String(), task.block.ID.String())
		}
	case verifyTx:
		if err := checkTx(task.tx, task.checkTxSig, task.checkOrder1, task.checkOrder2, scheme); err != nil {
			txID, txIdErr := task.tx.GetID(scheme)
			if txIdErr != nil {
				return errors.Wrap(txIdErr, "failed to get transaction ID")
			}
			return errors.Wrapf(err, "transaction '%s' verification failed", base58.Encode(txID))
		}
	default:
		return errors.Errorf("unknown verify task type (%d)", task.taskType)
	}
	return nil
}

func verify(ctx context.Context, tasks <-chan *verifyTask, scheme proto.Scheme) error {
	for {
		select {
		case task, ok := <-tasks:
			if !ok {
				return nil
			}
			if err := handleTask(task, scheme); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func launchVerifier(ctx context.Context, goroutinesNum int, scheme proto.Scheme) *verifierChans {
	if goroutinesNum <= 0 {
		panic("verifier launched with negative or zero goroutines number")
	}
	errgr, ctx := errgroup.WithContext(ctx)
	// run verifier goroutines
	tasksChan := make(chan *verifyTask)
	for i := 0; i < goroutinesNum; i++ {
		errgr.Go(func() error {
			return verify(ctx, tasksChan, scheme)
		})
	}
	// run waiter goroutine
	errChan := make(chan error, 1)
	go func(ch chan<- error) {
		if err := errgr.Wait(); err != nil {
			ch <- err
		}
		close(ch)
	}(errChan)
	return &verifierChans{tasksChan: tasksChan, errChan: errChan}
}
