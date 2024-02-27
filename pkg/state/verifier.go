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
	case *proto.Genesis: // special case for Genesis
		return nil
	case *proto.EthereumTransaction: // special case for EthereumTransaction
		if _, err := t.Verify(); err != nil {
			return errs.NewTxValidationError("EthereumTransaction transaction signature verification failed")
		}
		return nil
	case proto.Exchange: // special case for ExchangeTransaction
		var ( // compile-time interface checks
			_ selfVerifier   = (proto.Exchange)(nil) // ExchangeTransaction implements selfVerifier interface
			_ proto.Exchange = (*proto.ExchangeWithProofs)(nil)
			_ proto.Exchange = (*proto.ExchangeWithSig)(nil)
		)
		return verifyExchangeTransaction(t, params.Scheme, checkOrder1, checkOrder2)
	case selfVerifier: // general path, but exchange txs are handled separately because they ara also selfVerifiers
		return verifyTransactionSig(t, params.Scheme)
	default: // error path: new waves txs must implement selfVerifier interface.
		return errors.Errorf("unknown transaction type (%T)", t)
	}
}

func verifyExchangeTransaction(tx proto.Exchange, sch proto.Scheme, chOrd1, chOrd2 bool) error {
	if err := verifyTransactionSig(tx, sch); err != nil { // first verify transaction signature
		return err
	}
	// then verify orders signatures
	if chOrd1 {
		o1 := tx.GetOrder1()
		if ok, _ := o1.Verify(sch); !ok {
			return errs.NewTxValidationError("first Order signature verification failed")
		}
	}
	if chOrd2 {
		o2 := tx.GetOrder2()
		if ok, _ := o2.Verify(sch); !ok {
			return errs.NewTxValidationError("second Order signature verification failed")
		}
	}
	return nil
}

// selfVerifier is an interface for transactions that can verify their own signatures.
// It is used to verify WAVES transactions with proofs and transactions with signatures.
type selfVerifier interface {
	GetSenderPK() crypto.PublicKey
	Verify(scheme proto.Scheme, pk crypto.PublicKey) (bool, error)
}

var ( // compile-time interface checks
	_ selfVerifier = (*proto.TransferWithProofs)(nil)
	_ selfVerifier = (*proto.IssueWithProofs)(nil)
	_ selfVerifier = (*proto.ReissueWithProofs)(nil)
	_ selfVerifier = (*proto.BurnWithProofs)(nil)
	_ selfVerifier = (*proto.LeaseWithProofs)(nil)
	_ selfVerifier = (*proto.LeaseCancelWithProofs)(nil)
	_ selfVerifier = (*proto.CreateAliasWithProofs)(nil)
	_ selfVerifier = (*proto.SponsorshipWithProofs)(nil)
	_ selfVerifier = (*proto.MassTransferWithProofs)(nil)
	_ selfVerifier = (*proto.DataWithProofs)(nil)
	_ selfVerifier = (*proto.SetScriptWithProofs)(nil)
	_ selfVerifier = (*proto.SetAssetScriptWithProofs)(nil)
	_ selfVerifier = (*proto.InvokeScriptWithProofs)(nil)
	_ selfVerifier = (*proto.InvokeExpressionTransactionWithProofs)(nil)
	_ selfVerifier = (*proto.UpdateAssetInfoWithProofs)(nil)
	_ selfVerifier = (*proto.ExchangeWithProofs)(nil)
	_ selfVerifier = (*proto.Payment)(nil)
	_ selfVerifier = (*proto.TransferWithSig)(nil)
	_ selfVerifier = (*proto.IssueWithSig)(nil)
	_ selfVerifier = (*proto.ReissueWithSig)(nil)
	_ selfVerifier = (*proto.BurnWithSig)(nil)
	_ selfVerifier = (*proto.LeaseWithSig)(nil)
	_ selfVerifier = (*proto.LeaseCancelWithSig)(nil)
	_ selfVerifier = (*proto.CreateAliasWithSig)(nil)
	_ selfVerifier = (*proto.ExchangeWithSig)(nil)
)

func verifyTransactionSig(sv selfVerifier, scheme proto.Scheme) error {
	if ok, err := sv.Verify(scheme, sv.GetSenderPK()); !ok {
		if err != nil {
			return errs.Extend(err, fmt.Sprintf("(%T) transaction signature verification failed", sv))
		}
		return errs.NewTxValidationError(fmt.Sprintf("(%T) transaction signature verification failed", sv))
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
