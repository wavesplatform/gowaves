package server

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func TestGetBalances(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := state.DefaultTestingStateParams()
	// State should store addl data for gRPC API.
	params.StoreExtendedApiData = true
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	err = server.initServer(st, utxpool.New(utxSize))
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		conn.Close()
		err = st.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err)
	}()

	cl := g.NewAccountsApiClient(conn)
	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	assert.NoError(t, err)
	req := &g.BalancesRequest{
		Address: addr.Bytes(),
		Assets:  [][]byte{{}},
	}
	stream, err := cl.GetBalances(ctx, req)
	assert.NoError(t, err)
	res, err := stream.Recv()
	assert.NoError(t, err)
	correctBalance := &g.BalanceResponse_Waves{Waves: &g.BalanceResponse_WavesBalances{
		Regular:    9999999500000000,
		Generating: 9999999500000000,
		Available:  9999999500000000,
		Effective:  9999999500000000,
		LeaseIn:    0,
		LeaseOut:   0,
	}}
	assert.Equal(t, correctBalance, res.Balance)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestResolveAlias(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/alias_genesis.json")
	assert.NoError(t, err)
	st, stateCloser := stateWithCustomGenesis(t, genesisPath)
	err = server.initServer(st, utxpool.New(utxSize))
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		conn.Close()
		stateCloser()
	}()

	cl := g.NewAccountsApiClient(conn)

	aliasStr := "alias:W:nodes"
	alias, err := proto.NewAliasFromString(aliasStr)
	assert.NoError(t, err)
	correctAddr, err := st.AddrByAlias(*alias)
	assert.NoError(t, err)
	addr, err := cl.ResolveAlias(ctx, &wrappers.StringValue{Value: aliasStr})
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(correctAddr.Bytes(), addr.Value))
}
