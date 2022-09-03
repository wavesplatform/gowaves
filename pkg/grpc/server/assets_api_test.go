package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	protobuf "google.golang.org/protobuf/proto"
)

func TestGetInfo(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/asset_issue_genesis.json")
	assert.NoError(t, err)
	st := stateWithCustomGenesis(t, genesisPath)
	sets, err := st.BlockchainSettings()
	assert.NoError(t, err)
	ctx := withAutoCancel(t, context.Background())
	sch := createTestNetWallet(t)
	err = server.initServer(st, nil, sch)
	assert.NoError(t, err)

	conn := connectAutoClose(t, grpcTestAddr)

	cl := g.NewAssetsApiClient(conn)

	assetId := crypto.MustDigestFromBase58("DHgwrRvVyqJsepd32YbBqUeDH4GJ1N984X8QoekjgH8J")
	correctInfo, err := st.FullAssetInfo(proto.AssetIDFromDigest(assetId))
	assert.NoError(t, err)
	correctInfoProto, err := correctInfo.ToProtobuf(sets.AddressSchemeCharacter)
	assert.NoError(t, err)
	req := &g.AssetRequest{AssetId: assetId.Bytes()}
	info, err := cl.GetInfo(ctx, req)
	assert.NoError(t, err)
	assert.True(t, protobuf.Equal(correctInfoProto, info))
}
