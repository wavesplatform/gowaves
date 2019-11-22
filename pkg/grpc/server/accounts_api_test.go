package server

import (
	"bytes"
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

/* TODO: uncomment!
func TestGetDataEntries(t *testing.T) {
	genesisGetter := settings.FromCurrentDir("testdata/genesis", "data_entries_genesis.json")
	st, stateCloser := stateWithCustomGenesis(t, genesisGetter)
	err := server.resetState(st)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		conn.Close()
		stateCloser()
	}()

	cl := g.NewAccountsApiClient(conn)

	addr, err := proto.NewAddressFromString("3PPTrTo3AzR56N7ArzbU3Bpq9zYMgcf39Mk")
	assert.NoError(t, err)
	addrBytes := addr.Bytes()
	req := &g.DataRequest{Address: addrBytes}
	stream, err := cl.GetDataEntries(ctx, req)
	assert.NoError(t, err)
	entry0 := &g.DataTransactionData_DataEntry{
		Key:   "waves_usd_2",
		Value: &g.DataTransactionData_DataEntry_IntValue{IntValue: 81},
	}
	entry1 := &g.DataTransactionData_DataEntry{
		Key:   "waves_btc_8",
		Value: &g.DataTransactionData_DataEntry_IntValue{IntValue: 8653},
	}
	correctEntries := []*g.DataEntryResponse{
		&g.DataEntryResponse{Address: addrBytes, Entry: entry0},
		&g.DataEntryResponse{Address: addrBytes, Entry: entry1},
	}
	for _, correct := range correctEntries {
		entry, err := stream.Recv()
		assert.NoError(t, err)
		assert.Equal(t, correct, entry)
	}
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}
*/

func TestResolveAlias(t *testing.T) {
	genesisGetter := settings.FromCurrentDir("testdata/genesis", "alias_genesis.json")
	st, stateCloser := stateWithCustomGenesis(t, genesisGetter)
	err := server.resetState(st)
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
