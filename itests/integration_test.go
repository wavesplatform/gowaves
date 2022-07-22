package integration_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xenolf/lego/log"

	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/itests/net"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var (
	testCfg config.TestConfig
)

var (
	GoNodeClient    *client.Client
	ScalaNodeClient *client.Client
)

func initApiClients() error {
	var err error
	GoNodeClient, err = client.NewClient(client.Options{
		BaseUrl: "http://" + d.Localhost + ":" + d.GoNodeRESTApiPort + "/",
		Client:  &http.Client{Timeout: d.DefaultTimeout},
		ApiKey:  "itest-api-key",
	})
	if err != nil {
		return err
	}
	ScalaNodeClient, _ = client.NewClient(client.Options{
		BaseUrl: "http://" + d.Localhost + ":" + d.ScalaNodeRESTApiPort + "/",
		Client:  &http.Client{Timeout: d.DefaultTimeout},
		ApiKey:  "itest-api-key",
	})
	if err != nil {
		return err
	}
	return nil
}

func finishDocker(docker *d.Docker, cancelFunc context.CancelFunc) {
	cancelFunc()
	err := docker.Purge()
	if err != nil {
		log.Warnf("couldn't purge docker containers %s", err)
	}
}

func TestMain(m *testing.M) {
	paths, testConfig, err := config.CreateFileConfigs()
	if err != nil {
		log.Fatalf("couldn't create config %s", err)
	}
	testCfg = testConfig
	ctx, cancelFunc := context.WithCancel(context.Background())
	docker, err := d.NewDocker()
	if err != nil {
		log.Fatalf("couldn't create docker pool %s", err)
	}
	err = docker.RunContainers(ctx, paths)
	if err != nil {
		finishDocker(docker, cancelFunc)
		log.Fatalf("couldn't run docker containers %s", err)
	}
	err = initApiClients()
	if err != nil {
		finishDocker(docker, cancelFunc)
		log.Fatalf("couldn't create api clients %s", err)
	}
	code := m.Run()
	finishDocker(docker, cancelFunc)
	os.Exit(code)
}

func StateHashCmp(t *testing.T, ctx context.Context, height uint64) {
	goStateHash, _, err := GoNodeClient.Debug.StateHash(ctx, height)
	assert.NoError(t, err, "failed to get stateHash from scala node")

	scalaStateHash, _, err := ScalaNodeClient.Debug.StateHash(ctx, height)
	assert.NoError(t, err, "failed to get stateHash from scala node")

	assert.Equal(t, scalaStateHash, goStateHash)
}

func WaitForNewHeight(t *testing.T, ctx context.Context, beforeHeight client.BlocksHeight) uint64 {
	var scalaHeight, goHeight uint64
	for {
		h, _, err := GoNodeClient.Blocks.Height(ctx)
		assert.NoError(t, err, "failed to get height from go node")
		if h.Height > beforeHeight.Height+1 {
			goHeight = h.Height
			break
		}
		time.Sleep(time.Second * 1)
	}
	for {
		h, _, err := ScalaNodeClient.Blocks.Height(ctx)
		assert.NoError(t, err, "failed to get height from scala node")
		if h.Height > beforeHeight.Height+1 {
			scalaHeight = h.Height
			break
		}
		time.Sleep(time.Second * 1)
	}
	if scalaHeight < goHeight {
		return scalaHeight - 1
	} else {
		return goHeight - 1
	}
}

func SendStartMessage(t *testing.T, ctx context.Context) {
	_, err := ScalaNodeClient.Debug.PrintMsg(ctx, "------------- Start test: "+t.Name()+" -------------")
	assert.NoError(t, err, "failed to send StartMessage to go node")

	_, err = GoNodeClient.Debug.PrintMsg(ctx, "------------- Start test: "+t.Name()+" -------------")
	assert.NoError(t, err, "failed to send StartMessage to scala node")
}

func SendEndMessage(t *testing.T, ctx context.Context) {
	_, err := ScalaNodeClient.Debug.PrintMsg(ctx, "------------- End test: "+t.Name()+" -------------")
	assert.NoError(t, err, "failed to send StartMessage to go node")

	_, err = GoNodeClient.Debug.PrintMsg(ctx, "------------- End test: "+t.Name()+" -------------")
	assert.NoError(t, err, "failed to send StartMessage to scala node")
}

func TestSendTransaction(t *testing.T) {
	ctx := context.Background()
	SendStartMessage(t, ctx)
	goCon, err := net.NewConnection(proto.TCPAddr{}, d.Localhost+":"+d.GoNodeBindPort, proto.ProtocolVersion, "wavesL")
	assert.NoError(t, err, "failed to create connection to go node")
	defer func() {
		if err := goCon.Close(); err != nil {
			log.Warnf("Failed to close connection: %s", err)
		}
	}()

	scalaCon, err := net.NewConnection(proto.TCPAddr{}, d.Localhost+":"+d.ScalaNodeBindPort, proto.ProtocolVersion, "wavesL")
	assert.NoError(t, err, "failed to create connection to go node")
	defer func() {
		if err := scalaCon.Close(); err != nil {
			log.Warnf("Failed to close connection: %s", err)
		}
	}()

	a := proto.NewOptionalAssetWaves()
	ts := uint64(time.Now().UnixNano() / 1000000)
	tx := proto.NewUnsignedTransferWithSig(testCfg.Accounts[0].PublicKey, a, a, ts, 1000000000, 10000000,
		proto.NewRecipientFromAddress(testCfg.Accounts[1].Address), proto.Attachment{})
	err = tx.Sign('L', testCfg.Accounts[0].SecretKey)
	assert.NoError(t, err, "failed to create proofs from signature")

	bts, err := tx.MarshalBinary()
	assert.NoError(t, err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	heightBefore, _, err := GoNodeClient.Blocks.Height(ctx)
	assert.NoError(t, err, "failed to get height from go node")

	err = goCon.SendMessage(&txMsg)
	assert.NoError(t, err, "failed to send TransactionMessage")
	err = scalaCon.SendMessage(&txMsg)
	assert.NoError(t, err, "failed to send TransactionMessage")

	newHeight := WaitForNewHeight(t, ctx, *heightBefore)

	StateHashCmp(t, ctx, newHeight)
	SendEndMessage(t, ctx)
}
