package node_client

import (
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

func (c *NodesClients) StateHashCmp(t *testing.T, height uint64) (*proto.StateHash, *proto.StateHash, bool) {
	goStateHash := c.GoClients.HttpClient.StateHash(t, height)
	scalaStateHash := c.ScalaClients.HttpClient.StateHash(t, height)
	return goStateHash, scalaStateHash, goStateHash.BlockID == scalaStateHash.BlockID && goStateHash.SumHash == scalaStateHash.SumHash
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

func (c *NodesClients) WaitForStateHashEquality(t *testing.T) {
	var (
		equal          bool
		goStateHash    *proto.StateHash
		scalaStateHash *proto.StateHash
	)
	h := c.WaitForNewHeight(t)
	for i := 0; i < 3; i++ {
		if goStateHash, scalaStateHash, equal = c.StateHashCmp(t, h); equal {
			break
		}
		c.WaitForNewHeight(t)
	}

	if !equal && goStateHash.FieldsHashes.Equal(scalaStateHash.FieldsHashes) {
		var firstHeight int64 = -1
		for height := h; height > 0; height-- {
			goStateHash, scalaStateHash, equal = c.StateHashCmp(t, height)
			if !goStateHash.FieldsHashes.Equal(scalaStateHash.FieldsHashes) {
				firstHeight = int64(height)
			}
		}
		if firstHeight == -1 {
			t.Errorf("couldn't find the height when state hashes diverged. should not happen")
		}
		goStateHashDiverged, scalaStateHashDiverged, _ := c.StateHashCmp(t, uint64(firstHeight))
		goFieldHashesDiverged, err := goStateHashDiverged.FieldsHashes.MarshalJSON()
		assert.NoError(t, err)
		scalaFieldHashesDiverged, err := scalaStateHashDiverged.FieldsHashes.MarshalJSON()
		assert.NoError(t, err)

		t.Logf("First height when state hashes diverged: "+
			"%d:\nGo:\tBlockID=%s\tStateHash=%s\tFieldHashes=%s\n"+
			"Scala:\tBlockID=%s\tStateHash=%s\tFieldHashes=%s",
			firstHeight, goStateHashDiverged.BlockID.String(), goStateHashDiverged.SumHash.String(), goFieldHashesDiverged,
			scalaStateHashDiverged.BlockID.String(), scalaStateHashDiverged.SumHash.String(), scalaFieldHashesDiverged)
	}

	goFieldHashes, err := goStateHash.FieldsHashes.MarshalJSON()
	assert.NoError(t, err)
	scalaFieldHashes, err := scalaStateHash.FieldsHashes.MarshalJSON()
	assert.NoError(t, err)

	assert.True(t, equal,
		"Not equal state hash at height %d:\nGo:\tBlockID=%s\tStateHash=%s\tFieldHashes=%s\n"+
			"Scala:\tBlockID=%s\tStateHash=%s\tFieldHashes=%s",
		h, goStateHash.BlockID.String(), goStateHash.SumHash.String(), goFieldHashes,
		scalaStateHash.BlockID.String(), scalaStateHash.SumHash.String(), scalaFieldHashes)
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

func (c *NodesClients) WaitForConnectedPeers(timeout time.Duration) (error, error) {
	errGo := Retry(timeout, func() error {
		cp, _, err := c.GoClients.HttpClient.ConnectedPeers()
		if len(cp) == 0 && err == nil {
			err = errors.New("no connected peers")
		}
		return err
	})
	errScala := Retry(timeout, func() error {
		cp, _, err := c.ScalaClients.HttpClient.ConnectedPeers()
		if len(cp) == 0 && err == nil {
			err = errors.New("no connected peers")
		}
		return err
	})
	return errGo, errScala
}
