package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/wallet"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	sleepTime = 2 * time.Second
	utxSize   = 1000
)

var (
	server       *Server
	keyPairs     []proto.KeyPair
	grpcTestAddr string

	minerSkStr = "6SyE7t2u5HiKP1XJtRubbR9HSUhGGEkVAzHtobHnbGxL"
	minerPkStr = "7SPo26fzFRvFxAd6GiqSP2qBB98qt5hytGxKgq6faiZZ"
	seed       = "4TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk4bc"
)

func globalPathFromLocal(path string) (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("unable to find current package file")
	}
	dir := filepath.Dir(filename)
	return filepath.Join(dir, path), nil
}

func signBlock(t *testing.T, block *proto.Block, scheme proto.Scheme) {
	pk := crypto.MustPublicKeyFromBase58(minerPkStr)
	block.GeneratorPublicKey = pk
	sk := crypto.MustSecretKeyFromBase58(minerSkStr)
	err := block.Sign(scheme, sk)
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
	sets := settings.DefaultCustomSettings
	signBlock(t, genesis, sets.AddressSchemeCharacter)
	sets.Genesis = *genesis
	// For compatibility with MainNet addresses we use the same AddressSchemeCharacter.
	// This is needed because transactions from MainNet blockchain are used in tests' genesis blocks.
	sets.AddressSchemeCharacter = settings.MainNetSettings.AddressSchemeCharacter
	sets.BlockRewardTerm = 100000
	return sets
}

func stateWithCustomGenesis(t *testing.T, genesisPath string) state.State {
	dataDir := t.TempDir()
	sets := customSettingsWithGenesis(t, genesisPath)
	// Activate data transactions.
	sets.PreactivatedFeatures = []int16{5}
	params := defaultStateParams()
	st, err := state.NewState(dataDir, true, params, sets)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, st.Close())
	})
	return st
}

func createTestNetWallet(t *testing.T) types.EmbeddedWallet {
	w := wallet.NewWallet()
	decoded, err := base58.Decode(seed)
	require.NoError(t, err)
	err = w.AddAccountSeed(decoded)
	require.NoError(t, err)
	return wallet.NewEmbeddedWallet(nil, w, proto.TestNetScheme)
}

func connectAutoClose(t *testing.T, addr string) *grpc.ClientConn {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "grpc.Dial() failed")
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	return conn
}

func withAutoCancel(t *testing.T, ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	return ctx
}

func newTestState(t *testing.T, amend bool, params state.StateParams, settings *settings.BlockchainSettings) state.State {
	dataDir := t.TempDir()
	st, err := state.NewState(dataDir, amend, params, settings)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, st.Close())
	})
	return st
}

func TestMain(m *testing.M) {
	var err error
	server, err = NewServer(services.Services{Scheme: proto.MainNetScheme})
	if err != nil {
		log.Fatalf("Failed to create new gRPC server: %v", err)
	}
	grpcTestAddr = fmt.Sprintf("127.0.0.1:%d", freeport.GetPort())
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := server.Run(ctx, grpcTestAddr, nil); err != nil {
			log.Fatalf("server.Run(): %v\n", err)
		}
	}()

	seedBytes, err := base58.Decode(seed)
	if err != nil {
		log.Fatalf("Failed to decode test seed: %v\n", err)
	}
	keyPair, err := proto.NewKeyPair(seedBytes)
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v\n", err)
	}
	keyPairs = []proto.KeyPair{keyPair}

	time.Sleep(sleepTime)
	code := m.Run()
	cancel()
	os.Exit(code)
}

func defaultStateParams() state.StateParams {
	params := state.DefaultTestingStateParams()
	// State should store addrl data for gRPC API.
	params.StoreExtendedApiData = true
	params.ProvideExtendedApi = true
	return params
}

func assertTransactionResponsesEqual(t *testing.T, a, b *g.TransactionResponse) {
	assert.Equal(t, a.Id, b.Id)
	assert.Equal(t, a.ApplicationStatus, b.ApplicationStatus)
	assert.Equal(t, a.Height, b.Height)
	assert.Equal(t, a.Transaction, b.Transaction)
}

func assertTransactionStatusesEqual(t *testing.T, a, b *g.TransactionStatus) {
	assert.Equal(t, a.Id, b.Id)
	assert.Equal(t, a.ApplicationStatus, b.ApplicationStatus)
	assert.Equal(t, a.Height, b.Height)
	assert.Equal(t, a.Status, b.Status)
}
