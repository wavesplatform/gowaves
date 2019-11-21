package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	protobuf "github.com/golang/protobuf/proto"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

var (
	minerSkStr = "6SyE7t2u5HiKP1XJtRubbR9HSUhGGEkVAzHtobHnbGxL"
	minerPkStr = "7SPo26fzFRvFxAd6GiqSP2qBB98qt5hytGxKgq6faiZZ"
)

// wrappedGenesisGetter retrieves block from underlying getter and signs it using `minerSkStr`.
// It also sets block's GenPublicKey to corresponding public key (`minerPkStr`).
type wrappedGenesisGetter struct {
	block *proto.Block
}

func newWrappedGenesisGetter(getter settings.GenesisGetter) (*wrappedGenesisGetter, error) {
	block, err := getter.Get()
	if err != nil {
		return nil, err
	}
	pk := crypto.MustPublicKeyFromBase58(minerPkStr)
	block.GenPublicKey = pk
	sk := crypto.MustSecretKeyFromBase58(minerSkStr)
	if err := block.Sign(sk); err != nil {
		return nil, err
	}
	return &wrappedGenesisGetter{block: block}, nil
}

func (g *wrappedGenesisGetter) Get() (*proto.Block, error) {
	return g.block, nil
}

func stateWithCustomGenesis(t *testing.T, genesisGetter settings.GenesisGetter) (state.State, func()) {
	testGenesisGetter, err := newWrappedGenesisGetter(genesisGetter)
	assert.NoError(t, err)
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	sets := settings.DefaultSettingsForCustomBlockchain(testGenesisGetter)
	st, err := state.NewState(dataDir, state.DefaultTestingStateParams(), sets)
	assert.NoError(t, err)
	return st, func() {
		err = st.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err)
	}
}

func TestGetInfo(t *testing.T) {
	grpcTestAddr := fmt.Sprintf("127.0.0.1:%d", freeport.GetPort())
	genesisGetter := settings.FromCurrentDir("testdata/genesis", "asset_issue_genesis.json")
	st, stateCloser := stateWithCustomGenesis(t, genesisGetter)

	conn := connect(t, grpcTestAddr)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		conn.Close()
		stateCloser()
	}()

	cl := g.NewAssetsApiClient(conn)
	server, err := NewServer(st)
	assert.NoError(t, err)
	go func() {
		if err := server.Run(ctx, grpcTestAddr); err != nil {
			t.Error("server.Run failed")
		}
	}()

	time.Sleep(sleepTime)

	assetId := crypto.MustDigestFromBase58("DHgwrRvVyqJsepd32YbBqUeDH4GJ1N984X8QoekjgH8J")
	correctInfo, err := st.FullAssetInfo(assetId)
	assert.NoError(t, err)
	sets, err := st.BlockchainSettings()
	assert.NoError(t, err)
	correctInfoProto, err := correctInfo.ToProtobuf(sets.AddressSchemeCharacter)
	assert.NoError(t, err)
	req := &g.AssetRequest{AssetId: assetId.Bytes()}
	info, err := cl.GetInfo(ctx, req)
	assert.NoError(t, err)
	assert.True(t, protobuf.Equal(correctInfoProto, info))
}
