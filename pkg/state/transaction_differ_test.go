package state

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

var (
	defaultTimestamp = uint64(1465742577614)
	defaultAmount    = uint64(100)
	defaultFee       = uint64(1)

	matcherPK     = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa6"
	matcherAddr   = "3P9MUoSW7jfHNVFcq84rurfdWZYZuvVghVi"
	minerPK       = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7"
	minerAddr     = "3PP2ywCpyvC57rN4vUZhJjQrmGMTWnjFKi7"
	senderPK      = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa8"
	senderAddr    = "3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp"
	recipientPK   = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa9"
	recipientAddr = "3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo"

	assetStr = "B2u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"
)

type globalVars struct {
	matcherPK     crypto.PublicKey
	matcherAddr   proto.Address
	senderPK      crypto.PublicKey
	senderAddr    proto.Address
	recipientPK   crypto.PublicKey
	recipientAddr proto.Address
	minerPK       crypto.PublicKey
	minerAddr     proto.Address
}

var global globalVars

func TestMain(m *testing.M) {
	var err error
	global.matcherPK, err = crypto.NewPublicKeyFromBase58(matcherPK)
	if err != nil {
		log.Fatalf("Failed init global test vars: %v\n", err)
	}
	global.matcherAddr, err = proto.NewAddressFromString(matcherAddr)
	if err != nil {
		log.Fatalf("Failed init global test vars: %v\n", err)
	}
	global.senderPK, err = crypto.NewPublicKeyFromBase58(senderPK)
	if err != nil {
		log.Fatalf("Failed init global test vars: %v\n", err)
	}
	global.senderAddr, err = proto.NewAddressFromString(senderAddr)
	if err != nil {
		log.Fatalf("Failed init global test vars: %v\n", err)
	}
	global.recipientPK, err = crypto.NewPublicKeyFromBase58(recipientPK)
	if err != nil {
		log.Fatalf("Failed init global test vars: %v\n", err)
	}
	global.recipientAddr, err = proto.NewAddressFromString(recipientAddr)
	if err != nil {
		log.Fatalf("Failed init global test vars: %v\n", err)
	}
	global.minerPK, err = crypto.NewPublicKeyFromBase58(minerPK)
	if err != nil {
		log.Fatalf("Failed init global test vars: %v\n", err)
	}
	global.minerAddr, err = proto.NewAddressFromString(minerAddr)
	if err != nil {
		log.Fatalf("Failed init global test vars: %v\n", err)
	}
	os.Exit(m.Run())
}

type differTestObjects struct {
	stor *storageObjects
	td   *transactionDiffer
}

func createDifferTestObjects(t *testing.T) (*differTestObjects, []string) {
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	entities, err := newBlockchainEntitiesStorage(stor.hs, settings.MainNetSettings)
	assert.NoError(t, err, "newBlockchainEntitiesStorage() failed")
	td, err := newTransactionDiffer(entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionDiffer() failed")
	return &differTestObjects{stor, td}, path
}

func defaultDifferInfo(t *testing.T) *differInfo {
	return &differInfo{false, global.minerPK}
}

func createGenesis(t *testing.T) *proto.Genesis {
	return proto.NewUnsignedGenesis(global.recipientAddr, defaultAmount, defaultTimestamp)
}

func TestCreateDiffGenesis(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createGenesis(t)
	diff, err := to.td.createDiffGenesis(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffGenesis() failed")
	recipientKey := string((&wavesBalanceKey{address: global.recipientAddr}).bytes())
	correctDiff := txDiff{recipientKey: balanceDiff{balance: int64(tx.Amount)}}
	assert.Equal(t, correctDiff, diff)
}

func createPayment(t *testing.T) *proto.Payment {
	return proto.NewUnsignedPayment(global.senderPK, global.recipientAddr, defaultAmount, defaultFee, defaultTimestamp)
}

func TestCreateDiffPayment(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createPayment(t)
	diff, err := to.td.createDiffPayment(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffPayment() failed")

	minerKey := string((&wavesBalanceKey{address: global.minerAddr}).bytes())
	senderKey := string((&wavesBalanceKey{address: global.senderAddr}).bytes())
	recipientKey := string((&wavesBalanceKey{address: global.recipientAddr}).bytes())
	correctDiff := txDiff{
		senderKey:    balanceDiff{balance: int64(-tx.Amount - tx.Fee)},
		recipientKey: balanceDiff{balance: int64(tx.Amount)},
		minerKey:     balanceDiff{balance: int64(tx.Fee)},
	}
	assert.Equal(t, correctDiff, diff)
}
