package node_client

import (
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	d "github.com/wavesplatform/gowaves/itests/docker"
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

func NewNodesClients(t *testing.T, ports *d.Ports) *NodesClients {
	return &NodesClients{
		GoClients:    NewNodeClient(t, ports.Go.RestApiPort, ports.Go.GrpcPort),
		ScalaClients: NewNodeClient(t, ports.Scala.RestApiPort, ports.Scala.GrpcPort),
	}
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

func (c *NodesClients) WaitForTransaction(id crypto.Digest, timeout time.Duration) (error, error) {
	errGo := retry(timeout, func() error {
		_, _, err := c.GoClients.HttpClient.TransactionInfoRaw(id)
		return err
	})
	errScala := retry(timeout, func() error {
		_, _, err := c.ScalaClients.HttpClient.TransactionInfoRaw(id)
		return err
	})
	return errGo, errScala
}

func (c *NodesClients) ClearBlackList(t *testing.T) {
	c.GoClients.HttpClient.ClearBlackList(t)
	c.ScalaClients.HttpClient.ClearBlackList(t)
}
