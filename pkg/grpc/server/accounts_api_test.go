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
