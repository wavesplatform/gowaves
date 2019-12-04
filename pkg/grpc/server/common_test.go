package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"google.golang.org/grpc"
)

const (
	sleepTime = 2 * time.Second
	utxSize   = 1000
)

var (
	server       *Server
	grpcTestAddr string

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
	// Activate data transactions.
	sets.PreactivatedFeatures = []int16{5}
	st, err := state.NewState(dataDir, state.DefaultTestingStateParams(), sets)
	assert.NoError(t, err)
	return st, func() {
		err = st.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err)
	}
}

func connect(t *testing.T, addr string) *grpc.ClientConn {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	assert.NoError(t, err, "grpc.Dial() failed")
	return conn
}

func TestMain(m *testing.M) {
	server = &Server{}
	grpcTestAddr = fmt.Sprintf("127.0.0.1:%d", freeport.GetPort())
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := server.Run(ctx, grpcTestAddr); err != nil {
			log.Fatalf("server.Run(): %v\n", err)
		}
	}()

	time.Sleep(sleepTime)
	code := m.Run()
	cancel()
	os.Exit(code)
}
