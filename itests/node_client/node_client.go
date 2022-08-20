package node_client

import (
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type NodeClients struct {
	HttpClient *HttpClient
	GrpcClient *GrpcClient
}

func NewNodeClient(t *testing.T, httpPort string, grpcPort string) *NodeClients {
	return &NodeClients{
		HttpClient: NewHttpClient(t, httpPort),
		GrpcClient: NewGrpcClient(t, grpcPort),
	}
}

type NodesClients struct {
	GoClients    *NodeClients
	ScalaClients *NodeClients
}

func (c *NodesClients) SendStartMessage(t *testing.T) {
	c.GoClients.HttpClient.PrintMsg(t, "------------- Start test: "+t.Name()+" -------------")
	c.ScalaClients.HttpClient.PrintMsg(t, "------------- Start test: "+t.Name()+" -------------")
}

func (c *NodesClients) SendEndMessage(t *testing.T) {
	c.GoClients.HttpClient.PrintMsg(t, "------------- End test: "+t.Name()+" -------------")
	c.ScalaClients.HttpClient.PrintMsg(t, "------------- End test: "+t.Name()+" -------------")
}

func (c *NodesClients) StateHashCmp(t *testing.T, height uint64) {
	goStateHash := c.GoClients.HttpClient.StateHash(t, height)
	scalaStateHash := c.ScalaClients.HttpClient.StateHash(t, height)

	assert.Equal(t, scalaStateHash, goStateHash)
}

func (c *NodesClients) WaitForNewHeight(t *testing.T) uint64 {
	currentHeight := c.ScalaClients.HttpClient.GetHeight(t)
	for {
		h := c.GoClients.HttpClient.GetHeight(t)
		if h.Height >= currentHeight.Height+1 {
			break
		}
		time.Sleep(time.Second * 1)
	}
	for {
		h := c.ScalaClients.HttpClient.GetHeight(t)
		if h.Height >= currentHeight.Height+1 {
			break
		}
		time.Sleep(time.Second * 1)
	}
	return currentHeight.Height
}

func retry(timeout time.Duration, f func() error) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Second * 2
	bo.MaxElapsedTime = timeout
	if err := backoff.Retry(f, bo); err != nil {
		if bo.NextBackOff() == backoff.Stop {
			return errors.Wrap(err, "reached retry deadline")
		}
		return err
	}
	return nil
}

func (c *NodesClients) WaitForTransaction(t *testing.T, ID *crypto.Digest, timeout time.Duration) {
	err := retry(timeout, func() error {
		_, _, err := c.GoClients.HttpClient.TransactionInfoRaw(*ID)
		return err
	})
	assert.NoError(t, err, "Failed to get TransactionInfo from go node")
	err = retry(timeout, func() error {
		_, _, err := c.ScalaClients.HttpClient.TransactionInfoRaw(*ID)
		return err
	})
	assert.NoError(t, err, "Failed to get TransactionInfo from scala node")
}
