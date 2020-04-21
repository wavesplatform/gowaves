package server

import (
	"context"
	"testing"

	protobuf "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
)

func TestGetInfo(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/asset_issue_genesis.json")
	assert.NoError(t, err)
	st, stateCloser := stateWithCustomGenesis(t, genesisPath)
	sets, err := st.BlockchainSettings()
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createWallet(ctx, st, sets)
	err = server.initServer(st, nil, sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		conn.Close()
		stateCloser()
	}()

	cl := g.NewAssetsApiClient(conn)

	assetId := crypto.MustDigestFromBase58("DHgwrRvVyqJsepd32YbBqUeDH4GJ1N984X8QoekjgH8J")
	correctInfo, err := st.FullAssetInfo(assetId)
	assert.NoError(t, err)
	correctInfoProto, err := correctInfo.ToProtobuf(sets.AddressSchemeCharacter)
	assert.NoError(t, err)
	req := &g.AssetRequest{AssetId: assetId.Bytes()}
	info, err := cl.GetInfo(ctx, req)
	assert.NoError(t, err)
	assert.True(t, protobuf.Equal(correctInfoProto, info))
}
