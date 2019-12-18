package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"github.com/pkg/errors"
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

func globalPathFromLocal(path string) (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("unable to find current package file")
	}
	dir := filepath.Dir(filename)
	return filepath.Join(dir, path), nil
}

func signBlock(t *testing.T, block *proto.Block) {
	pk := crypto.MustPublicKeyFromBase58(minerPkStr)
	block.GenPublicKey = pk
	sk := crypto.MustSecretKeyFromBase58(minerSkStr)
	err := block.Sign(sk)
	assert.NoError(t, err)
}

func customSettingsWithGenesis(t *testing.T, genesisPath string) *settings.BlockchainSettings {
	genesisFile, err := os.Open(genesisPath)
	assert.NoError(t, err)
	jsonParser := json.NewDecoder(genesisFile)
	genesis := &proto.Block{}
	err = jsonParser.Decode(genesis)
	assert.NoError(t, err)
	err = genesisFile.Close()
	assert.NoError(t, err)
	signBlock(t, genesis)
	sets := settings.DefaultCustomSettings
	sets.Genesis = *genesis
	return sets
}

func stateWithCustomGenesis(t *testing.T, genesisPath string) (state.State, func()) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	sets := customSettingsWithGenesis(t, genesisPath)
	// Activate data transactions.
	sets.PreactivatedFeatures = []int16{5}
	params := state.DefaultTestingStateParams()
	// State should store addl data for gRPC API.
	params.StoreExtendedApiData = true
	st, err := state.NewState(dataDir, params, sets)
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
