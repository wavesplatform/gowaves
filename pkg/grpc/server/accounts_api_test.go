package server

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestGetBalances(t *testing.T) {
	dataDir := t.TempDir()
	params := defaultStateParams()
	st, err := state.NewState(dataDir, true, params, settings.MainNetSettings)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	err = server.initServer(st, nil, nil)
	require.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	t.Cleanup(func() {
		cancel()
		err := conn.Close()
		require.NoError(t, err)
		err = st.Close()
		require.NoError(t, err)
	})

	cl := g.NewAccountsApiClient(conn)
	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	require.NoError(t, err)
	addrBody := addr.Body()
	req := &g.BalancesRequest{
		Address: addrBody,
		Assets:  [][]byte{{}},
	}
	stream, err := cl.GetBalances(ctx, req)
	require.NoError(t, err)
	res, err := stream.Recv()
	require.NoError(t, err)
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
	require.NoError(t, err)
	st := stateWithCustomGenesis(t, genesisPath)
	sets, err := st.BlockchainSettings()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createWallet(ctx, st, sets)
	err = server.initServer(st, nil, sch)
	require.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	t.Cleanup(func() {
		cancel()
		err = conn.Close()
		require.NoError(t, err)
	})

	cl := g.NewAccountsApiClient(conn)
	addr, err := proto.NewAddressFromString("3Fv3jiLvLS4c4N1ZvSLac3HBGUzaHDMvjN1")
	require.NoError(t, err)
	req := &g.AccountRequest{
		Address: addr.Body(),
	}
	stream, err := cl.GetActiveLeases(ctx, req)
	require.NoError(t, err)
	res, err := stream.Recv()
	require.NoError(t, err)
	txId, err := crypto.NewDigestFromBase58("ADXuoPsKMJ59HyLMGzLBbNQD8p2eJ93dciuBPJp3Qhx")
	require.NoError(t, err)
	tx, err := st.TransactionByID(txId.Bytes())
	require.NoError(t, err)

	ltx, ok := tx.(*proto.LeaseWithSig)
	require.True(t, ok)
	assert.Equal(t, ltx.ID.Bytes(), res.LeaseId)
	assert.Equal(t, ltx.ID.Bytes(), res.OriginTransactionId)
	assert.Equal(t, int(ltx.Amount), int(res.Amount))
	expRecipient, err := ltx.Recipient.ToProtobuf()
	require.NoError(t, err)
	assert.Equal(t, expRecipient, res.Recipient)
	expSender := proto.MustAddressFromPublicKey(sets.AddressSchemeCharacter, ltx.SenderPK)
	assert.Equal(t, expSender.Body(), res.Sender)
	expHeight, err := st.TransactionHeightByID(txId.Bytes())
	require.NoError(t, err)
	assert.Equal(t, int(expHeight), int(res.Height))
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestResolveAlias(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/alias_genesis.json")
	require.NoError(t, err)
	st := stateWithCustomGenesis(t, genesisPath)
	sets, err := st.BlockchainSettings()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createWallet(ctx, st, sets)
	err = server.initServer(st, nil, sch)
	require.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	t.Cleanup(func() {
		cancel()
		err := conn.Close()
		require.NoError(t, err)
	})

	cl := g.NewAccountsApiClient(conn)

	aliasStr := "nodes"
	alias := proto.NewAlias('W', aliasStr)
	correctAddr, err := st.AddrByAlias(*alias)
	require.NoError(t, err)
	correctAddrBody := correctAddr.Body()
	addr, err := cl.ResolveAlias(ctx, &wrapperspb.StringValue{Value: aliasStr})
	require.NoError(t, err)
	assert.True(t, bytes.Equal(correctAddrBody, addr.Value))
}
