package integration_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/itests/net"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"log"
	"os"
	"testing"
	"time"

	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
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
	docker, err := d.NewDocker()
	if err != nil {
		log.Fatalf("couldn't create docker pool %s", err)
	}
	err = docker.RunContainers(paths)
	if err != nil {
		log.Fatalf("couldn't run docker containers %s", err)
	}
	code := m.Run()
	err = docker.Purge()
	if err != nil {
		log.Fatalf("couldn't purge docker containers %s", err)
	}
	os.Exit(code)
}

func TestCheckHeight(t *testing.T) {
	goHeight, err := d.GoNodeClient.GetBlocksHeight()
	assert.NoError(t, err, "failed to get height from go node")

	scalaHeight, err := d.ScalaNodeClient.GetBlocksHeight()
	assert.NoError(t, err, "failed to get height from scala node")

	assert.Equal(t, goHeight, scalaHeight)
}

func TestSendTransaction(t *testing.T) {
	c, err := net.NewConnection(proto.TCPAddr{}, "127.0.0.1"+":"+d.ScalaNodeBindPort, proto.Version{Major: 1, Minor: 4, Patch: 0}, "wavesL")
	assert.NoError(t, err, "failed to create connection to go node")

	a := proto.NewOptionalAssetWaves()
	ts := uint64(time.Now().UnixNano() / 1000000)
	tx := proto.NewUnsignedTransferWithSig(testCfg.Accounts[0].PublicKey, a, a, ts, 10000000, 10000, proto.NewRecipientFromAddress(testCfg.Accounts[1].Address), proto.Attachment{})
	err = tx.Sign('L', testCfg.Accounts[0].SecretKey)
	assert.NoError(t, err, "failed to create proofs frm signature")

	bts, err := tx.MarshalBinary()
	assert.NoError(t, err, "failed to marshal tx")

	txMsg := proto.TransactionMessage{Transaction: bts}
	err = c.SendMessage(&txMsg)
	assert.NoError(t, err, "failed to send message GetPeersMessage")
}
