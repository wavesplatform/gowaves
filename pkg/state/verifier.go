package state

import (
	"context"
	"fmt"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	taskType     verifyTaskType
	parentID     proto.BlockID
	block        *proto.Block
	tx           proto.Transaction
	checkTxSig   bool
	checkOrder1  bool
	checkOrder2  bool
	checkVersion bool
}

type selfVerifier interface {
	GetSenderPK() crypto.PublicKey
	GetType() proto.TransactionType
	Verify(scheme proto.Scheme, pk crypto.PublicKey) (bool, error)
}

var ( // compile-time interface checks
	_ proto.Exchange = (*proto.ExchangeWithProofs)(nil)
	_ proto.Exchange = (*proto.ExchangeWithSig)(nil)
	_ selfVerifier   = (*proto.ExchangeWithProofs)(nil)
	_ selfVerifier   = (*proto.ExchangeWithSig)(nil)
	_ selfVerifier   = (*proto.TransferWithProofs)(nil)
	_ selfVerifier   = (*proto.IssueWithProofs)(nil)
	_ selfVerifier   = (*proto.ReissueWithProofs)(nil)
	_ selfVerifier   = (*proto.BurnWithProofs)(nil)
	_ selfVerifier   = (*proto.LeaseWithProofs)(nil)
	_ selfVerifier   = (*proto.LeaseCancelWithProofs)(nil)
	_ selfVerifier   = (*proto.CreateAliasWithProofs)(nil)
	_ selfVerifier   = (*proto.SponsorshipWithProofs)(nil)
	_ selfVerifier   = (*proto.MassTransferWithProofs)(nil)
	_ selfVerifier   = (*proto.DataWithProofs)(nil)
	_ selfVerifier   = (*proto.SetScriptWithProofs)(nil)
	_ selfVerifier   = (*proto.SetAssetScriptWithProofs)(nil)
	_ selfVerifier   = (*proto.InvokeScriptWithProofs)(nil)
	_ selfVerifier   = (*proto.InvokeExpressionTransactionWithProofs)(nil)
	_ selfVerifier   = (*proto.UpdateAssetInfoWithProofs)(nil)
	_ selfVerifier   = (*proto.Payment)(nil)
	_ selfVerifier   = (*proto.TransferWithSig)(nil)
	_ selfVerifier   = (*proto.IssueWithSig)(nil)
	_ selfVerifier   = (*proto.ReissueWithSig)(nil)
	_ selfVerifier   = (*proto.BurnWithSig)(nil)
	_ selfVerifier   = (*proto.LeaseWithSig)(nil)
	_ selfVerifier   = (*proto.LeaseCancelWithSig)(nil)
	_ selfVerifier   = (*proto.CreateAliasWithSig)(nil)
)

func verifyTransactionSignature(sv selfVerifier, scheme proto.Scheme) error {
	if ok, err := sv.Verify(scheme, sv.GetSenderPK()); !ok {
		if err != nil {
			return errs.Extend(err, fmt.Sprintf("%s signature verification failed", sv.GetType().String()))
		}
		return errs.NewTxValidationError(fmt.Sprintf("%s signature verification failed", sv.GetType().String()))
	}
	return nil
}

func checkTx(
	tx proto.Transaction, checkTxSig, checkOrder1, checkOrder2 bool, params proto.TransactionValidationParams,
) error {
	if _, err := tx.Validate(params); err != nil {
		return errs.Extend(err, "invalid tx data")
	}
	if !checkTxSig {
		return nil
	}
	switch t := tx.(type) {
	case *proto.Genesis:
		return nil
	case proto.Exchange: // special case for ExchangeTransaction
		return verifyExchangeTransaction(t, params.Scheme, checkOrder1, checkOrder2)
	case *proto.EthereumTransaction:
		if _, err := t.Verify(); err != nil {
			return errs.NewTxValidationError(fmt.Sprintf(
				"EthereumTransaction transaction signature verification failed: %v", err,
			))
		}
	case selfVerifier:
		return verifyTransactionSignature(t, params.Scheme)
	default:
		return errors.New("unknown transaction type")
	}
	return nil
}

func verifyExchangeTransaction(tx proto.Exchange, sch proto.Scheme, chOrd1, chOrd2 bool) error {
	if ok, err := tx.Verify(sch, tx.GetSenderPK()); err != nil {
		return errs.Extend(err, "Exchange transaction signature verification failed")
	} else if !ok {
		return errs.NewTxValidationError("Exchange tx signature verification failed")
	}
	if chOrd1 {
		if ok, err := tx.GetOrder1().Verify(sch); err != nil {
			return errs.Extend(err, "first Order signature verification failed")
		} else if !ok {
			return errs.NewTxValidationError("first Order signature verification failed")
		}
	}
	if chOrd2 {
		if ok, err := tx.GetOrder2().Verify(sch); err != nil {
			return errs.Extend(err, "second Order signature verification failed")
		} else if !ok {
			return errs.NewTxValidationError("second Order signature verification failed")
		}
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
		// Deny self-challenged blocks.
		if ch, ok := task.block.GetChallengedHeader(); ok {
			if task.block.GeneratorPublicKey == ch.GeneratorPublicKey {
				return errors.Errorf("State: handleTask: block '%s' is self-challenged", task.block.ID.String())
			}
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
		params := proto.TransactionValidationParams{Scheme: scheme, CheckVersion: task.checkVersion}
		if err := checkTx(task.tx, task.checkTxSig, task.checkOrder1, task.checkOrder2, params); err != nil {
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
