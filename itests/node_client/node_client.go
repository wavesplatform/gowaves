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

// WaitForNewHeight waits for nodes to generate new block.
// Returns the height that was *before* generation of new block.
func (c *NodesClients) WaitForNewHeight(t *testing.T) uint64 {
	initialHeight := c.ScalaClients.HttpClient.GetHeight(t).Height
	c.WaitForHeight(t, initialHeight+1)
	return initialHeight
}

// WaitForHeight waits for nodes to get on given height. Exits if nodes' height already equal or greater than requested.
// Function returns actual nodes' height.
func (c *NodesClients) WaitForHeight(t *testing.T, height uint64) uint64 {
	var hg, hs uint64
	for {
		hg = c.GoClients.HttpClient.GetHeight(t).Height
		if hg >= height {
			break
		}
		time.Sleep(time.Second * 1)
	}
	for {
		hs = c.ScalaClients.HttpClient.GetHeight(t).Height
		if hs >= height {
			break
		}
		time.Sleep(time.Second * 1)
	}
	if hg < hs {
		return hg
	}
	return hs
}

func Retry(timeout time.Duration, f func() error) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Second * 1
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
	errGo := Retry(timeout, func() error {
		_, _, err := c.GoClients.HttpClient.TransactionInfoRaw(id)
		return err
	})
	errScala := Retry(timeout, func() error {
		_, _, err := c.ScalaClients.HttpClient.TransactionInfoRaw(id)
		return err
	})
	return errGo, errScala
}

func (c *NodesClients) WaitForConnectedPeers(t *testing.T, timeout time.Duration) (error, error) {
	errGo := Retry(timeout, func() error {
		cp, _, err := c.GoClients.HttpClient.ConnectedPeers(t)
		if len(cp) == 0 && err == nil {
			err = errors.New("no connected peers")
		}
		return err
	})
	errScala := Retry(timeout, func() error {
		cp, _, err := c.ScalaClients.HttpClient.ConnectedPeers(t)
		if len(cp) == 0 && err == nil {
			err = errors.New("no connected peers")
		}
		return err
	})
	return errGo, errScala
}
