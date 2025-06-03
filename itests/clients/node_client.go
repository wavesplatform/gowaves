package clients

import (
	"context"
	stderrs "errors"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/wavesplatform/gowaves/pkg/client"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const synchronizedBalancesTimeout = 15 * time.Second

type NodesClients struct {
	GoClient    *NodeUniversalClient
	ScalaClient *NodeUniversalClient
}

func NewNodesClients(ctx context.Context, t *testing.T, goPorts, scalaPorts *d.PortConfig) *NodesClients {
	sp, err := proto.NewPeerInfoFromString(config.DefaultIP + ":" + scalaPorts.BindPort)
	require.NoError(t, err, "failed to create Scala peer info")
	gp, err := proto.NewPeerInfoFromString(config.DefaultIP + ":" + goPorts.BindPort)
	require.NoError(t, err, "failed to create Go peer info")
	peers := []proto.PeerInfo{sp, gp}
	return &NodesClients{
		GoClient: NewNodeUniversalClient(
			ctx, t, NodeGo, goPorts.RESTAPIPort, goPorts.GRPCPort, goPorts.BindPort, peers,
		),
		ScalaClient: NewNodeUniversalClient(
			ctx, t, NodeScala, scalaPorts.RESTAPIPort, scalaPorts.GRPCPort, scalaPorts.BindPort, peers,
		),
	}
}

func (c *NodesClients) SendStartMessage(t *testing.T) {
	c.GoClient.SendStartMessage(t)
	c.ScalaClient.SendStartMessage(t)
}

func (c *NodesClients) SendEndMessage(t *testing.T) {
	c.GoClient.SendEndMessage(t)
	c.ScalaClient.SendEndMessage(t)
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
	)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			hg = c.GoClient.HTTPClient.GetHeight(t).Height
			if hg >= height {
				break
			}
			time.Sleep(time.Second * 1)
		}
		return nil
	})
	g.Go(func() error {
		for {
			hs = c.ScalaClient.HTTPClient.GetHeight(t).Height
			if hs >= height {
				break
			}
			time.Sleep(time.Second * 1)
		}
		return nil
	})
	// Wait for both goroutines to finish.
	if err := g.Wait(); err != nil {
		t.Logf("Error while waiting for height: %v", err)
	}
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

func (c *NodesClients) WaitForConnectedPeers(ctx context.Context, timeout time.Duration) error {
	eg, ctx := errgroup.WithContext(ctx) // context will be canceled when first error occurs
	eg.Go(func() error {
		err := RetryCtx(ctx, timeout, func() error {
			cp, _, err := c.GoClient.HTTPClient.ConnectedPeersCtx(ctx)
			if len(cp) == 0 && err == nil {
				err = errors.New("no connected peers")
			}
			return err
		})
		return errors.Wrap(err, "Go")
	})
	eg.Go(func() error {
		err := RetryCtx(ctx, timeout, func() error {
			cp, _, err := c.ScalaClient.HTTPClient.ConnectedPeersCtx(ctx)
			if len(cp) == 0 && err == nil {
				err = errors.New("no connected peers")
			}
			return err
		})
		return errors.Wrap(err, "Scala")
	})
	return eg.Wait() // Wait for both clients to finish and return first error
}

func (c *NodesClients) reportFirstDivergedHeight(t *testing.T, height uint64) {
	var (
		first         uint64
		goSH, scalaSH *proto.StateHash
	)
	for h := height; h > 0; h-- {
		goSH, scalaSH, _ = c.StateHashCmp(t, h)
		if !goSH.Equal(scalaSH.FieldsHashes) {
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
	var (
		addrBalances = make([]addressedBalanceAtHeight, len(addresses))
		errsForJoin  = make([]error, len(addresses)) // Errors from all goroutines
		wg           = sync.WaitGroup{}
	)
	wg.Add(len(addresses))
	for i, addr := range addresses {
		go func(i int, addr proto.WavesAddress) {
			defer wg.Done()
			ab, err := c.requestNodesAvailableBalances(ctx, addr)
			if err != nil {
				errsForJoin[i] = err // Write error here, we will retry to get synced balances later.
				addrBalances[i] = addressedBalanceAtHeight{address: addr, balance: NodesWavesBalanceAtHeight{}}
				return
			}
			addrBalances[i] = ab
		}(i, addr)
	}
	wg.Wait() // Wait for all goroutines to finish.
	r := make(map[proto.WavesAddress]NodesWavesBalanceAtHeight, len(addresses))
	for _, e := range addrBalances {
		r[e.address] = e.balance
	}
	return r, stderrs.Join(errsForJoin...)
}

func (c *NodesClients) SynchronizedWavesBalances(
	t *testing.T, addresses ...proto.WavesAddress,
) SynchronisedBalances {
	ctx, cancel := context.WithTimeout(context.Background(), synchronizedBalancesTimeout)
	defer cancel()

	t.Logf("Initial balances request")
	sbs, err := c.requestAvailableBalancesForAddresses(ctx, addresses)
	if err != nil {
		t.Logf("Errors while requesting balances: %v", err)
	}
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
		// Update the map with retry results.
		maps.Copy(sbs, rr)
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

func (c *NodesClients) Handshake() {
	c.GoClient.Connection.SendHandshake()
	c.ScalaClient.Connection.SendHandshake()
}

func deduplicateImplementations(s []Implementation) []Implementation {
	c := slices.Clone(s)
	slices.Sort(c)
	return slices.Compact(c)
}

func (c *NodesClients) SendToScalaNode(t *testing.T, m proto.Message) {
	t.Logf("Sending message to Scala node: %T", m)
	c.ScalaClient.Connection.SendMessage(m)
	t.Log("Message sent to Scala node")
}

func (c *NodesClients) SendToGoNode(t *testing.T, m proto.Message) {
	t.Logf("Sending message to Go node: %T", m)
	c.GoClient.Connection.SendMessage(m)
	t.Log("Message sent to Go node")
}

func (c *NodesClients) SendToNodes(t *testing.T, m proto.Message, nodes []Implementation) {
	ns := deduplicateImplementations(nodes)
	for i := range ns {
		switch ns[i] {
		case NodeGo:
			c.SendToGoNode(t, m)
		case NodeScala:
			c.SendToScalaNode(t, m)
		default:
			t.Fatalf("Unexpected node implementation %d", ns[i])
		}
	}
}

func (c *NodesClients) BroadcastToGoNode(t *testing.T, tx proto.Transaction) (*client.Response, error) {
	t.Logf("Broadcasting transaction to Go node: %T", tx)
	respGo, errBrdCstGo := c.GoClient.HTTPClient.TransactionBroadcast(tx)
	if errBrdCstGo != nil {
		t.Logf("Error while broadcasting transaction to Go node: %v", errBrdCstGo)
	} else {
		t.Logf("Transaction was successfully Broadcast to Go node")
	}
	return respGo, errBrdCstGo
}

func (c *NodesClients) BroadcastToScalaNode(t *testing.T, tx proto.Transaction) (*client.Response, error) {
	t.Logf("Broadcasting transaction to Scala node: %T", tx)
	respScala, errBrdCstScala := c.ScalaClient.HTTPClient.TransactionBroadcast(tx)
	if errBrdCstScala != nil {
		t.Logf("Error while broadcasting transaction to Scala node: %v", errBrdCstScala)
	} else {
		t.Logf("Transaction was successfully Broadcast to Scala node")
	}
	return respScala, errBrdCstScala
}

func (c *NodesClients) BroadcastToNodes(t *testing.T, tx proto.Transaction,
	nodes []Implementation) (*client.Response, error, *client.Response, error) {
	var respGo, respScala *client.Response = nil, nil
	var errBrdCstGo, errBrdCstScala error = nil, nil

	ns := deduplicateImplementations(nodes)
	for i := range ns {
		switch ns[i] {
		case NodeGo:
			respGo, errBrdCstGo = c.BroadcastToGoNode(t, tx)
		case NodeScala:
			respScala, errBrdCstScala = c.BroadcastToScalaNode(t, tx)
		default:
			t.Fatalf("Unexpected node implementation %d", ns[i])
		}
	}

	return respGo, errBrdCstGo, respScala, errBrdCstScala
}

func (c *NodesClients) Close(t *testing.T) {
	c.GoClient.GRPCClient.Close(t)
	c.GoClient.Connection.Close()
	c.ScalaClient.GRPCClient.Close(t)
	c.ScalaClient.Connection.Close()
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

func RetryCtx(ctx context.Context, timeout time.Duration, f func() error) error {
	bo := backoff.WithContext(
		backoff.NewExponentialBackOff(
			backoff.WithMaxInterval(time.Second*1),
			backoff.WithMaxElapsedTime(timeout),
		), ctx,
	)
	if err := backoff.Retry(f, bo); err != nil {
		if bo.NextBackOff() == backoff.Stop {
			return errors.Wrap(err, "reached retry deadline")
		}
		return err
	}
	return nil
}

func Retry(timeout time.Duration, f func() error) error {
	return RetryCtx(context.Background(), timeout, f)
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
