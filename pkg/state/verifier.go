package state

import (
	"context"
	"fmt"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

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
	case *proto.ExchangeWithSig:
		return verifyExchangeTransactionWithSig(t, params.Scheme, checkOrder1, checkOrder2)
	case *proto.ExchangeWithProofs:
		return verifyExchangeTransactionWithProofs(t, params.Scheme, checkOrder1, checkOrder2)
	case *proto.EthereumTransaction:
		if _, err := t.Verify(); err != nil {
			return errs.NewTxValidationError("EthereumTransaction transaction signature verification failed")
		}
	default:
		return verifyTransactionsWithProofs(tx, params.Scheme)
	}
	return nil
}

func verifyExchangeTransactionWithSig(tx *proto.ExchangeWithSig, sch proto.Scheme, chOrd1, chOrd2 bool) error {
	if ok, err := tx.Verify(sch, tx.SenderPK); !ok {
		if err != nil {
			return errs.Extend(err, "Exchange transaction signature verification failed")
		}
		return errs.NewTxValidationError("Exchange tx signature verification failed")
	}
	if chOrd1 {
		if ok, _ := tx.Order1.Verify(sch); !ok {
			return errs.NewTxValidationError("first Order signature verification failed")
		}
	}
	if chOrd2 {
		if ok, _ := tx.Order2.Verify(sch); !ok {
			return errs.NewTxValidationError("second Order signature verification failed")
		}
	}
	return nil
}

func verifyExchangeTransactionWithProofs(tx *proto.ExchangeWithProofs, sch proto.Scheme, chOrd1, chOrd2 bool) error {
	if ok, err := tx.Verify(sch, tx.SenderPK); !ok {
		if err != nil {
			return errs.Extend(err, "Exchange transaction signature verification failed")
		}
		return errs.NewTxValidationError("Exchange tx signature verification failed")
	}
	if chOrd1 {
		if ok, _ := tx.Order1.Verify(sch); !ok {
			return errs.NewTxValidationError("first Order signature verification failed")
		}
	}
	if chOrd2 {
		if ok, _ := tx.Order2.Verify(sch); !ok {
			return errs.NewTxValidationError("second Order signature verification failed")
		}
	}
	return nil
}

func verifyTransaction(vf func() (bool, error), name string) error {
	if ok, err := vf(); !ok {
		if err != nil {
			return errs.Extend(err, fmt.Sprintf("%s transaction signature verification failed", name))
		}
		return errs.NewTxValidationError(fmt.Sprintf("%s transaction signature verification failed", name))
	}
	return nil
}

func verifyTransactionsWithProofs(tx proto.Transaction, scheme proto.Scheme) error {
	switch t := tx.(type) {
	case *proto.TransferWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Transfer")
	case *proto.IssueWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Issue")
	case *proto.ReissueWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Reissue")
	case *proto.BurnWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Burn")
	case *proto.LeaseWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Lease")
	case *proto.LeaseCancelWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "LeaseCancel")
	case *proto.CreateAliasWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "CreateAlias")
	case *proto.SponsorshipWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Sponsorship")
	case *proto.MassTransferWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "MassTransfer")
	case *proto.DataWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Data")
	case *proto.SetScriptWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "SetScript")
	case *proto.SetAssetScriptWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "SetAssetScript")
	case *proto.InvokeScriptWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "InvokeScript")
	case *proto.InvokeExpressionTransactionWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "InvokeExpression")
	case *proto.UpdateAssetInfoWithProofs:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "UpdateAssetInfo")
	default:
		return verifyTransactionsWithSignatures(tx, scheme)
	}
}

func verifyTransactionsWithSignatures(tx proto.Transaction, scheme proto.Scheme) error {
	switch t := tx.(type) {
	case *proto.Genesis:
		return nil
	case *proto.Payment:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Payment")
	case *proto.TransferWithSig:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Transfer")
	case *proto.IssueWithSig:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Issue")
	case *proto.ReissueWithSig:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Reissue")
	case *proto.BurnWithSig:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Burn")
	case *proto.LeaseWithSig:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "Lease")
	case *proto.LeaseCancelWithSig:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "LeaseCancel")
	case *proto.CreateAliasWithSig:
		return verifyTransaction(func() (bool, error) { return t.Verify(scheme, t.SenderPK) }, "CreateAlias")
	default:
		return errors.New("unknown transaction type")
	}
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
