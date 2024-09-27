package clients

import (
	"context"
	stderrs "errors"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const synchronizedBalancesTimeout = 15 * time.Second

type NodesClients struct {
	GoClient    *NodeUniversalClient
	ScalaClient *NodeUniversalClient
}

func NewNodesClients(t *testing.T, goPorts, scalaPorts *d.PortConfig) *NodesClients {
	return &NodesClients{
		GoClient:    NewNodeUniversalClient(t, NodeGo, goPorts.RESTAPIPort, goPorts.GRPCPort),
		ScalaClient: NewNodeUniversalClient(t, NodeScala, scalaPorts.RESTAPIPort, scalaPorts.GRPCPort),
	}
}

func (c *NodesClients) SendStartMessage(t *testing.T) {
	c.GoClient.HTTPClient.PrintMsg(t, "------------- Start test: "+t.Name()+" -------------")
	c.ScalaClient.HTTPClient.PrintMsg(t, "------------- Start test: "+t.Name()+" -------------")
}

func (c *NodesClients) SendEndMessage(t *testing.T) {
	c.GoClient.HTTPClient.PrintMsg(t, "------------- End test: "+t.Name()+" -------------")
	c.ScalaClient.HTTPClient.PrintMsg(t, "------------- End test: "+t.Name()+" -------------")
}

func (c *NodesClients) StateHashCmp(t *testing.T, height uint64) (*proto.StateHash, *proto.StateHash, bool) {
	goStateHash := c.GoClient.HTTPClient.StateHash(t, height)
	scalaStateHash := c.ScalaClient.HTTPClient.StateHash(t, height)
	return goStateHash, scalaStateHash,
		goStateHash.BlockID == scalaStateHash.BlockID && goStateHash.SumHash == scalaStateHash.SumHash
}

// WaitForNewHeight waits for nodes to generate new block.
// Returns the height that was *before* generation of new block.
func (c *NodesClients) WaitForNewHeight(t *testing.T) uint64 {
	initialHeight := c.ScalaClient.HTTPClient.GetHeight(t).Height
	c.WaitForHeight(t, initialHeight+1)
	return initialHeight
}

// WaitForHeight waits for nodes to get on given height. Exits if nodes' height already equal or greater than requested.
// Function returns actual nodes' height.
func (c *NodesClients) WaitForHeight(t *testing.T, height uint64) uint64 {
	var (
		hg, hs uint64
		wg     sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			hg = c.GoClient.HTTPClient.GetHeight(t).Height
			if hg >= height {
				break
			}
			time.Sleep(time.Second * 1)
		}
	}()
	go func() {
		defer wg.Done()
		for {
			hs = c.ScalaClient.HTTPClient.GetHeight(t).Height
			if hs >= height {
				break
			}
			time.Sleep(time.Second * 1)
		}
	}()
	wg.Wait() // Wait for both clients to finish
	return min(hg, hs)
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
	if !equal && goStateHash != nil && scalaStateHash != nil {
		assert.Failf(t, "Not equal state hashes",
			"Not equal state hash at height %d:\n"+
				"Go:\tBlockID=%s\tStateHash=%s\tFieldHashes=%s\n"+
				"Scala:\tBlockID=%s\tStateHash=%s\tFieldHashes=%s",
			h, goStateHash.BlockID.String(), goStateHash.SumHash.String(),
			mustFieldsHashesToString(goStateHash.FieldsHashes), scalaStateHash.BlockID.String(),
			scalaStateHash.SumHash.String(), mustFieldsHashesToString(scalaStateHash.FieldsHashes),
		)
		c.reportFirstDivergedHeight(t, h)
	}
}

func (c *NodesClients) WaitForTransaction(id crypto.Digest, timeout time.Duration) (error, error) {
	var (
		errGo, errScala error
		wg              sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errGo = Retry(timeout, func() error {
			_, _, err := c.GoClient.HTTPClient.TransactionInfoRaw(id)
			return err
		})
	}()
	go func() {
		defer wg.Done()
		errScala = Retry(timeout, func() error {
			_, _, err := c.ScalaClient.HTTPClient.TransactionInfoRaw(id)
			return err
		})
	}()
	wg.Wait() // Wait for both clients to finish
	return errGo, errScala
}

func (c *NodesClients) WaitForConnectedPeers(timeout time.Duration) (error, error) {
	var (
		errGo, errScala error
		wg              sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errGo = Retry(timeout, func() error {
			cp, _, err := c.GoClient.HTTPClient.ConnectedPeers()
			if len(cp) == 0 && err == nil {
				err = errors.New("no connected peers")
			}
			return err
		})
	}()
	go func() {
		errScala = Retry(timeout, func() error {
			cp, _, err := c.ScalaClient.HTTPClient.ConnectedPeers()
			if len(cp) == 0 && err == nil {
				err = errors.New("no connected peers")
			}
			return err
		})
	}()
	wg.Wait() // Wait for both clients to finish
	return errGo, errScala
}

func (c *NodesClients) reportFirstDivergedHeight(t *testing.T, height uint64) {
	var (
		first         uint64
		goSH, scalaSH *proto.StateHash
	)
	for h := height; h > 0; h-- {
		goSH, scalaSH, _ = c.StateHashCmp(t, h)
		if !goSH.FieldsHashes.Equal(scalaSH.FieldsHashes) {
			first = h
		} else {
			break
		}
	}
	if first == 0 {
		t.Error("couldn't find the height when state hashes diverged. should not happen")
		return
	}

	goSH, scalaSH, _ = c.StateHashCmp(t, first)
	t.Logf("First height when state hashes diverged: %d:\n"+
		"Go:\tBlockID=%s\tStateHash=%s\tFieldHashes=%s\n"+
		"Scala:\tBlockID=%s\tStateHash=%s\tFieldHashes=%s",
		first, goSH.BlockID.String(), goSH.SumHash.String(), mustFieldsHashesToString(goSH.FieldsHashes),
		scalaSH.BlockID.String(), scalaSH.SumHash.String(), mustFieldsHashesToString(scalaSH.FieldsHashes),
	)
}

func (c *NodesClients) requestAvailableBalancesForAddresses(
	ctx context.Context, addresses []proto.WavesAddress,
) (map[proto.WavesAddress]NodesWavesBalanceAtHeight, error) {
	ch := make(chan addressedBalanceAtHeight, len(addresses))

	var jointError error

	g, childCtx := errgroup.WithContext(ctx)
	for _, addr := range addresses {
		g.Go(func() error {
			ab, err := c.requestNodesAvailableBalances(childCtx, addr)
			if err != nil {
				jointError = stderrs.Join(jointError, err)
				// Suppress error here, we will retry to get synced balances later,
				// but throwing error cancels the whole group.
				ch <- addressedBalanceAtHeight{address: addr, balance: NodesWavesBalanceAtHeight{}}
				return nil
			}
			ch <- ab
			return nil
		})
	}
	_ = g.Wait()
	close(ch)

	r := make(map[proto.WavesAddress]NodesWavesBalanceAtHeight, len(addresses))
	for e := range ch {
		r[e.address] = e.balance
	}
	return r, jointError
}

func (c *NodesClients) SynchronizedWavesBalances(
	t *testing.T, addresses ...proto.WavesAddress,
) SynchronisedBalances {
	ctx, cancel := context.WithTimeout(context.Background(), synchronizedBalancesTimeout)
	defer cancel()

	t.Logf("Initial balacnces request")
	sbs, err := c.requestAvailableBalancesForAddresses(ctx, addresses)
	if err != nil {
		t.Logf("Errors while requesting balances: %v", err)
	}
	t.Log("Entering loop")
	for {
		commonHeight := mostCommonHeight(sbs)
		toRetry := make([]proto.WavesAddress, 0, len(addresses))
		for addr, sb := range sbs {
			if sb.Height != commonHeight || sb.Height == 0 { // We are going to retry even if the commonHeight is 0.
				toRetry = append(toRetry, addr)
			}
		}

		if len(toRetry) == 0 {
			break
		}

		t.Logf("Heights differ, retrying for %d addresses", len(toRetry))
		time.Sleep(time.Second)
		rr, rrErr := c.requestAvailableBalancesForAddresses(ctx, toRetry)
		if rrErr != nil {
			t.Logf("Errors while requesting balances: %v", rrErr)
		}
		for k, v := range rr {
			sbs[k] = v // Update the map with retry results.
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			t.Logf("Timeout reached, returning empty result")
			return NewSynchronisedBalances()
		}
	}

	r := NewSynchronisedBalances()
	for address, sb := range sbs {
		r.Put(address, sb.Balance)
		r.Height = sb.Height
	}
	return r
}

func (c *NodesClients) requestNodesAvailableBalances(
	ctx context.Context, address proto.WavesAddress,
) (addressedBalanceAtHeight, error) {
	ch := make(chan balanceAtHeight, 2)
	g, childCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		gbh, err := c.GoClient.GRPCClient.syncedWavesAvailableBalance(childCtx, address)
		if err != nil {
			return err
		}
		ch <- gbh
		return nil
	})
	g.Go(func() error {
		sbh, err := c.ScalaClient.GRPCClient.syncedWavesAvailableBalance(childCtx, address)
		if err != nil {
			return err
		}
		ch <- sbh
		return nil
	})
	err := g.Wait()
	close(ch)
	if err != nil {
		return addressedBalanceAtHeight{}, err
	}
	var gb, sb int64
	var h uint64
	for e := range ch {
		switch e.impl {
		case NodeGo:
			gb = e.balance
			h, err = validateHeights(h, e.height)
			if err != nil {
				return addressedBalanceAtHeight{}, err
			}
		case NodeScala:
			sb = e.balance
			h, err = validateHeights(h, e.height)
			if err != nil {
				return addressedBalanceAtHeight{}, err
			}
		default:
			panic("unexpected implementation or default value")
		}
	}
	r := addressedBalanceAtHeight{
		address: address,
		balance: NodesWavesBalanceAtHeight{
			Balance: NodesWavesBalance{
				GoBalance:    gb,
				ScalaBalance: sb,
			},
			Height: h,
		},
	}
	return r, nil
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

func mustFieldsHashesToString(fieldHashes proto.FieldsHashes) string {
	b, err := fieldHashes.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return string(b)
}

type addressedBalanceAtHeight struct {
	address proto.WavesAddress
	balance NodesWavesBalanceAtHeight
}

func mostCommonHeight(m map[proto.WavesAddress]NodesWavesBalanceAtHeight) proto.Height {
	counts := make(map[proto.Height]int)
	for _, sb := range m {
		counts[sb.Height]++
	}
	var (
		maxHeight proto.Height
		maxCount  int
	)
	for h, c := range counts {
		if c > maxCount {
			maxCount = c
			maxHeight = h
		}
	}
	return maxHeight
}

func validateHeights(known, other uint64) (uint64, error) {
	if known == 0 {
		return other, nil
	}
	if known != other {
		return 0, errors.New("heights differ")
	}
	return known, nil
}
