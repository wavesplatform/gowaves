package utils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/itests/node_client"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

func SendStartMessage(t *testing.T, ctx context.Context) {
	node_client.GoNodeClient(t).PrintMsg(t, ctx, "------------- Start test: "+t.Name()+" -------------")
	node_client.ScalaNodeClient(t).PrintMsg(t, ctx, "------------- Start test: "+t.Name()+" -------------")
}

func SendEndMessage(t *testing.T, ctx context.Context) {
	node_client.GoNodeClient(t).PrintMsg(t, ctx, "------------- End test: "+t.Name()+" -------------")
	node_client.ScalaNodeClient(t).PrintMsg(t, ctx, "------------- End test: "+t.Name()+" -------------")
}

func StateHashCmp(t *testing.T, ctx context.Context, height uint64) {
	goStateHash := node_client.GoNodeClient(t).StateHash(t, ctx, height)
	scalaStateHash := node_client.ScalaNodeClient(t).StateHash(t, ctx, height)

	assert.Equal(t, scalaStateHash, goStateHash)
}

func WaitForNewHeight(t *testing.T, ctx context.Context, beforeHeight client.BlocksHeight) uint64 {
	var scalaHeight, goHeight uint64
	for {
		h := node_client.GoNodeClient(t).GetHeight(t, ctx)
		if h.Height > beforeHeight.Height+1 {
			goHeight = h.Height
			break
		}
		time.Sleep(time.Second * 1)
	}
	for {
		h := node_client.ScalaNodeClient(t).GetHeight(t, ctx)
		if h.Height > beforeHeight.Height+1 {
			scalaHeight = h.Height
			break
		}
		time.Sleep(time.Second * 1)
	}
	if scalaHeight < goHeight {
		return scalaHeight - 1
	} else {
		return goHeight - 1
	}
}

func Retry(timeout time.Duration, f func() error) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Second * 2
	bo.MaxElapsedTime = timeout
	if err := backoff.Retry(f, bo); err != nil {
		if bo.NextBackOff() == backoff.Stop {
			return fmt.Errorf("reached retry deadline")
		}
		return err
	}
	return nil
}

func WaitForTransaction(t *testing.T, ctx context.Context, ID *crypto.Digest, timeout time.Duration) {
	err := Retry(timeout, func() error {
		_, _, err := node_client.GoNodeClient(t).TransactionInfoRaw(ctx, *ID)
		return err
	})
	assert.NoError(t, err, "Failed to get TransactionInfo from go node")
	err = Retry(timeout, func() error {
		_, _, err := node_client.ScalaNodeClient(t).TransactionInfoRaw(ctx, *ID)
		return err
	})
	assert.NoError(t, err, "Failed to get TransactionInfo from scala node")
}
