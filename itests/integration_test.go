package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xenolf/lego/log"

	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/itests/net"
	"github.com/wavesplatform/gowaves/itests/node_client"
	"github.com/wavesplatform/gowaves/itests/utils"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var (
	testCfg config.TestConfig
)

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
		docker.Finish(cancelFunc)
		log.Fatalf("couldn't run docker containers %s", err)
	}
	code := m.Run()
	docker.Finish(cancelFunc)
	os.Exit(code)
}

func TestSendTransaction(t *testing.T) {
	ctx := context.Background()
	utils.SendStartMessage(t, ctx)

	a := proto.NewOptionalAssetWaves()
	ts := uint64(time.Now().UnixNano() / 1000000)
	tx := proto.NewUnsignedTransferWithSig(testCfg.Accounts[0].PublicKey, a, a, ts, 1000000000, 10000000,
		proto.NewRecipientFromAddress(testCfg.Accounts[1].Address), proto.Attachment{})
	err := tx.Sign('L', testCfg.Accounts[0].SecretKey)
	assert.NoError(t, err, "failed to create proofs from signature")

	bts, err := tx.MarshalBinary()
	assert.NoError(t, err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	heightBefore := node_client.GoNodeClient(t).GetHeight(t, ctx)

	connections := net.NewNodeConnections(t)
	defer connections.Close()
	connections.SendToEachNode(t, &txMsg)

	newHeight := utils.WaitForNewHeight(t, ctx, *heightBefore)
	utils.StateHashCmp(t, ctx, newHeight)

	utils.SendEndMessage(t, ctx)
}
