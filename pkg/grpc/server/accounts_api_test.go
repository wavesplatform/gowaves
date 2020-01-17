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
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func TestGetBalances(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, settings.MainNetSettings)
	err = server.initServer(st, utxpool.New(utxSize), sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
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
	addrBody, err := addr.Body()
	assert.NoError(t, err)
	req := &g.BalancesRequest{
		Address: addrBody,
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

func TestGetActiveLeases(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/lease_genesis.json")
	assert.NoError(t, err)
	st, stateCloser := stateWithCustomGenesis(t, genesisPath)
	sets, err := st.BlockchainSettings()
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, sets)
	err = server.initServer(st, utxpool.New(utxSize), sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		conn.Close()
		stateCloser()
	}()

	cl := g.NewAccountsApiClient(conn)
	addr, err := proto.NewAddressFromString("3Fv3jiLvLS4c4N1ZvSLac3HBGUzaHDMvjN1")
	assert.NoError(t, err)
	addrBody, err := addr.Body()
	assert.NoError(t, err)
	req := &g.AccountRequest{
		Address: addrBody,
	}
	stream, err := cl.GetActiveLeases(ctx, req)
	assert.NoError(t, err)
	res, err := stream.Recv()
	assert.NoError(t, err)
	txId, err := crypto.NewDigestFromBase58("ADXuoPsKMJ59HyLMGzLBbNQD8p2eJ93dciuBPJp3Qhx")
	assert.NoError(t, err)
	tx, err := st.TransactionByID(txId.Bytes())
	assert.NoError(t, err)
	correctRes, err := server.transactionToTransactionResponse(tx, true)
	assert.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestResolveAlias(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/alias_genesis.json")
	assert.NoError(t, err)
	st, stateCloser := stateWithCustomGenesis(t, genesisPath)
	sets, err := st.BlockchainSettings()
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, sets)
	err = server.initServer(st, utxpool.New(utxSize), sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		conn.Close()
		stateCloser()
	}()

	cl := g.NewAccountsApiClient(conn)

	aliasStr := "nodes"
	alias := proto.NewAlias('W', aliasStr)
	correctAddr, err := st.AddrByAlias(*alias)
	assert.NoError(t, err)
	addr, err := cl.ResolveAlias(ctx, &wrappers.StringValue{Value: aliasStr})
	assert.NoError(t, err)
	correctBody, err := correctAddr.Body()
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(correctBody, addr.Value))
}
