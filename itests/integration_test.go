package integration_test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/itests/net"
	"github.com/wavesplatform/gowaves/pkg/client"
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

func StateHashCmp(t *testing.T, height uint64) {
	goStateHash, err := d.GoNodeClient.GetStateHash(height)
	assert.NoError(t, err, "failed to get stateHash from scala node")

	scalaStateHash, err := d.ScalaNodeClient.GetStateHash(height)
	assert.NoError(t, err, "failed to get stateHash from scala node")

	assert.Equal(t, scalaStateHash, goStateHash)
}

func WaitForNewHeight(t *testing.T, beforeHeight client.BlocksHeight) uint64 {
	var scalaHeight, goHeight uint64
	for {
		h, err := d.GoNodeClient.GetBlocksHeight()
		assert.NoError(t, err, "failed to get height from go node")
		if h.Height > beforeHeight.Height+3 {
			goHeight = h.Height
			break
		}
		time.Sleep(time.Second * 1)
	}
	for {
		h, err := d.ScalaNodeClient.GetBlocksHeight()
		assert.NoError(t, err, "failed to get height from scala node")
		if h.Height > beforeHeight.Height+3 {
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

func SendStartMessage(t *testing.T) {
	err := d.ScalaNodeClient.PostDebugPrint("------------- Start test: " + t.Name() + " -------------")
	assert.NoError(t, err, "failed to send StartMessage to go node")

	err = d.GoNodeClient.PostDebugPrint("------------- Start test: " + t.Name() + " -------------")
	assert.NoError(t, err, "failed to send StartMessage to scala node")
}

func SendEndMessage(t *testing.T) {
	err := d.ScalaNodeClient.PostDebugPrint("------------- End test: " + t.Name() + " -------------")
	assert.NoError(t, err, "failed to send StartMessage to go node")

	err = d.GoNodeClient.PostDebugPrint("------------- End test: " + t.Name() + " -------------")
	assert.NoError(t, err, "failed to send StartMessage to scala node")
}

func TestSendTransaction(t *testing.T) {
	SendStartMessage(t)
	goCon, err := net.NewConnection(proto.TCPAddr{}, d.Localhost+":"+d.GoNodeBindPort, net.NodeVersion, "wavesL")
	assert.NoError(t, err, "failed to create connection to go node")

	scalaCon, err := net.NewConnection(proto.TCPAddr{}, d.Localhost+":"+d.ScalaNodeBindPort, net.NodeVersion, "wavesL")
	assert.NoError(t, err, "failed to create connection to go node")

	a := proto.NewOptionalAssetWaves()
	ts := uint64(time.Now().UnixNano() / 1000000)
	tx := proto.NewUnsignedTransferWithSig(testCfg.Accounts[0].PublicKey, a, a, ts, 1000000000, 10000000,
		proto.NewRecipientFromAddress(testCfg.Accounts[1].Address), proto.Attachment{})
	err = tx.Sign('L', testCfg.Accounts[0].SecretKey)
	assert.NoError(t, err, "failed to create proofs from signature")

	bts, err := tx.MarshalBinary()
	assert.NoError(t, err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	heightBefore, err := d.GoNodeClient.GetBlocksHeight()
	assert.NoError(t, err, "failed to get height from go node")

	err = goCon.SendMessage(&txMsg)
	assert.NoError(t, err, "failed to send TransactionMessage")
	err = scalaCon.SendMessage(&txMsg)
	assert.NoError(t, err, "failed to send TransactionMessage")

	newHeight := WaitForNewHeight(t, *heightBefore)

	StateHashCmp(t, newHeight)
	SendEndMessage(t)
}
