package state

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type ordersVolumesStorageObjects struct {
	stor          *testStorageObjects
	ordersVolumes *ordersVolumes
}

func createOrdersVolumeStorageObjects(t *testing.T) *ordersVolumesStorageObjects {
	stor := createStorageObjects(t, true)
	ordersVolumes := newOrdersVolumes(stor.hs)
	return &ordersVolumesStorageObjects{stor, ordersVolumes}
}

func TestIncreaseFilledFee(t *testing.T) {
	to := createOrdersVolumeStorageObjects(t)

	to.stor.addBlock(t, blockID0)
	orderId := bytes.Repeat([]byte{0xff}, crypto.DigestSize)
	firstFee := uint64(1)
	secondFee := uint64(100500)
	err := to.ordersVolumes.increaseFilledFee(orderId, firstFee, blockID0)
	assert.NoError(t, err)
	filledFee, err := to.ordersVolumes.newestFilledFee(orderId)
	assert.NoError(t, err)
	assert.Equal(t, firstFee, filledFee)

	err = to.ordersVolumes.increaseFilledFee(orderId, secondFee, blockID0)
	assert.NoError(t, err)
	filledFee, err = to.ordersVolumes.newestFilledFee(orderId)
	assert.NoError(t, err)
	assert.Equal(t, firstFee+secondFee, filledFee)

	to.stor.flush(t)
	filledFee, err = to.ordersVolumes.newestFilledFee(orderId)
	assert.NoError(t, err)
	assert.Equal(t, firstFee+secondFee, filledFee)
}

func TestIncreaseFilledAmount(t *testing.T) {
	to := createOrdersVolumeStorageObjects(t)

	to.stor.addBlock(t, blockID0)
	orderId := bytes.Repeat([]byte{0xff}, crypto.DigestSize)
	firstAmount := uint64(1)
	secondAmount := uint64(100500)
	err := to.ordersVolumes.increaseFilledAmount(orderId, firstAmount, blockID0)
	assert.NoError(t, err)
	filledAmount, err := to.ordersVolumes.newestFilledAmount(orderId)
	assert.NoError(t, err)
	assert.Equal(t, firstAmount, filledAmount)

	err = to.ordersVolumes.increaseFilledAmount(orderId, secondAmount, blockID0)
	assert.NoError(t, err)
	filledAmount, err = to.ordersVolumes.newestFilledAmount(orderId)
	assert.NoError(t, err)
	assert.Equal(t, firstAmount+secondAmount, filledAmount)

	to.stor.flush(t)
	filledAmount, err = to.ordersVolumes.newestFilledAmount(orderId)
	assert.NoError(t, err)
	assert.Equal(t, firstAmount+secondAmount, filledAmount)
}
