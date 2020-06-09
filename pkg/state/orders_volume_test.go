package state

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type ordersVolumesStorageObjects struct {
	stor          *testStorageObjects
	ordersVolumes *ordersVolumes
}

func createOrdersVolumeStorageObjects() (*ordersVolumesStorageObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	ordersVolumes := newOrdersVolumes(stor.hs)
	return &ordersVolumesStorageObjects{stor, ordersVolumes}, path, nil
}

func TestIncreaseFilledFee(t *testing.T) {
	to, path, err := createOrdersVolumeStorageObjects()
	assert.NoError(t, err, "createOrdersVolumeStorageObjects() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	orderId := bytes.Repeat([]byte{0xff}, crypto.DigestSize)
	firstFee := uint64(1)
	secondFee := uint64(100500)
	assert.NoError(t, err)
	err = to.ordersVolumes.increaseFilledFee(orderId, firstFee, blockID0, true)
	assert.NoError(t, err)
	filledFee, err := to.ordersVolumes.newestFilledFee(orderId, true)
	assert.NoError(t, err)
	assert.Equal(t, firstFee, filledFee)

	err = to.ordersVolumes.increaseFilledFee(orderId, secondFee, blockID0, true)
	assert.NoError(t, err)
	filledFee, err = to.ordersVolumes.newestFilledFee(orderId, true)
	assert.NoError(t, err)
	assert.Equal(t, firstFee+secondFee, filledFee)

	to.stor.flush(t)
	filledFee, err = to.ordersVolumes.newestFilledFee(orderId, true)
	assert.NoError(t, err)
	assert.Equal(t, firstFee+secondFee, filledFee)
}

func TestIncreaseFilledAmount(t *testing.T) {
	to, path, err := createOrdersVolumeStorageObjects()
	assert.NoError(t, err, "createOrdersVolumeStorageObjects() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	orderId := bytes.Repeat([]byte{0xff}, crypto.DigestSize)
	firstAmount := uint64(1)
	secondAmount := uint64(100500)
	assert.NoError(t, err)
	err = to.ordersVolumes.increaseFilledAmount(orderId, firstAmount, blockID0, true)
	assert.NoError(t, err)
	filledAmount, err := to.ordersVolumes.newestFilledAmount(orderId, true)
	assert.NoError(t, err)
	assert.Equal(t, firstAmount, filledAmount)

	err = to.ordersVolumes.increaseFilledAmount(orderId, secondAmount, blockID0, true)
	assert.NoError(t, err)
	filledAmount, err = to.ordersVolumes.newestFilledAmount(orderId, true)
	assert.NoError(t, err)
	assert.Equal(t, firstAmount+secondAmount, filledAmount)

	to.stor.flush(t)
	filledAmount, err = to.ordersVolumes.newestFilledAmount(orderId, true)
	assert.NoError(t, err)
	assert.Equal(t, firstAmount+secondAmount, filledAmount)
}
