package server

import (
	"context"
	"testing"

	protobuf "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
)

func TestGetInfo(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/asset_issue_genesis.json")
	assert.NoError(t, err)
	st, stateCloser := stateWithCustomGenesis(t, genesisPath)
	sets, err := st.BlockchainSettings()
	assert.NoError(t, err)
	sch := scheduler.NewScheduler(st, keyPairs, sets)
	err = server.initServer(st, utxpool.New(utxSize), sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	ctx, cancel := context.WithCancel(context.Background())
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
