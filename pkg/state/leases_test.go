package state

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type leasesTestObjects struct {
	stor   *testStorageObjects
	leases *leases
}

func createLeases(t *testing.T) *leasesTestObjects {
	stor := createStorageObjects(t, true)
	leases := newLeases(stor.hs, true)
	return &leasesTestObjects{stor, leases}
}

func createLease(t *testing.T, sender string, id crypto.Digest) *leasing {
	senderAddr, err := proto.NewAddressFromString(sender)
	assert.NoError(t, err, "failed to create address from string")
	recipientAddr, err := proto.NewAddressFromString("3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo")
	assert.NoError(t, err, "failed to create address from string")
	return &leasing{
		OriginTransactionID: &id,
		Recipient:           recipientAddr,
		Sender:              senderAddr,
		Amount:              10,
		Status:              LeaseActive,
	}
}

func TestCancelLeases(t *testing.T) {
	to := createLeases(t)

	to.stor.addBlock(t, blockID0)
	leasings := []struct {
		sender      string
		leaseIDByte byte
	}{
		{"3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp", 0xff},
		{"3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo", 0xaa},
	}
	for _, l := range leasings {
		leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{l.leaseIDByte}, crypto.DigestSize))
		assert.NoError(t, err, "failed to create digest from bytes")
		r := createLease(t, l.sender, leaseID)
		err = to.leases.addLeasing(leaseID, r, blockID0)
		assert.NoError(t, err, "failed to add leasing")
	}
	// Cancel one lease by sender and check.
	badSenderStr := leasings[0].sender
	badSender, err := proto.NewAddressFromString(badSenderStr)
	assert.NoError(t, err, "failed to create address from string")
	sendersToCancel := make(map[proto.WavesAddress]struct{})
	var empty struct{}
	sendersToCancel[badSender] = empty
	err = to.leases.cancelLeases(sendersToCancel, blockID0)
	assert.NoError(t, err, "cancelLeases() failed")
	to.stor.flush(t)
	for _, l := range leasings {
		leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{l.leaseIDByte}, crypto.DigestSize))
		assert.NoError(t, err, "failed to create digest from bytes")
		leasing, err := to.leases.leasingInfo(leaseID)
		assert.NoError(t, err, "failed to get leasing")
		active, err := to.leases.isActive(leaseID)
		assert.NoError(t, err)
		if l.sender == badSenderStr {
			assert.Equal(t, false, active)
			assert.Equal(t, leasing.isActive(), false, "did not cancel leasing by sender")
		} else {
			assert.Equal(t, true, active)
			assert.Equal(t, leasing.isActive(), true, "cancelled leasing with different sender")
		}
	}
	// Cancel all the leases and check.
	err = to.leases.cancelLeases(nil, blockID0)
	assert.NoError(t, err, "cancelLeases() failed")
	to.stor.flush(t)
	for _, l := range leasings {
		leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{l.leaseIDByte}, crypto.DigestSize))
		assert.NoError(t, err, "failed to create digest from bytes")
		leasing, err := to.leases.leasingInfo(leaseID)
		assert.NoError(t, err, "failed to get leasing")
		assert.Equal(t, leasing.isActive(), false, "did not cancel all the leasings")
		active, err := to.leases.isActive(leaseID)
		assert.NoError(t, err)
		assert.Equal(t, false, active)
	}
}

func TestValidLeaseIns(t *testing.T) {
	to := createLeases(t)

	to.stor.addBlock(t, blockID0)
	leasings := []struct {
		sender      string
		leaseIDByte byte
	}{
		{"3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp", 0xff},
		{"3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo", 0xaa},
	}
	properLeaseIns := make(map[proto.WavesAddress]int64)
	for _, l := range leasings {
		leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{l.leaseIDByte}, crypto.DigestSize))
		assert.NoError(t, err, "failed to create digest from bytes")
		r := createLease(t, l.sender, leaseID)
		err = to.leases.addLeasing(leaseID, r, blockID0)
		assert.NoError(t, err, "failed to add leasing")
		properLeaseIns[r.Recipient] += int64(r.Amount)
	}
	leaseIns, err := to.leases.validLeaseIns()
	assert.NoError(t, err, "validLeaseIns() failed")
	assert.Equal(t, len(properLeaseIns), len(leaseIns))
	for k, v := range properLeaseIns {
		v1 := leaseIns[k]
		assert.Equal(t, v, v1)
	}
}

func TestAddLeasing(t *testing.T) {
	to := createLeases(t)

	to.stor.addBlock(t, blockID0)
	leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	senderStr := "3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp"
	r := createLease(t, senderStr, leaseID)
	err = to.leases.addLeasing(leaseID, r, blockID0)
	assert.NoError(t, err, "failed to add leasing")
	l, err := to.leases.newestLeasingInfo(leaseID)
	assert.NoError(t, err, "failed to get newest leasing info")
	assert.Equal(t, l, r, "leasings differ before flushing")
	to.stor.flush(t)
	resLeasing, err := to.leases.leasingInfo(leaseID)
	assert.NoError(t, err, "failed to get leasing info")
	assert.Equal(t, resLeasing, r, "leasings differ after flushing")
}

func TestCancelLeasing(t *testing.T) {
	to := createLeases(t)

	to.stor.addBlock(t, blockID0)
	leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	txID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xfe}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	senderStr := "3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp"
	r := createLease(t, senderStr, leaseID)
	err = to.leases.addLeasing(leaseID, r, blockID0)
	assert.NoError(t, err, "failed to add leasing")
	err = to.leases.cancelLeasing(leaseID, blockID0, to.stor.rw.height, &txID)
	assert.NoError(t, err, "failed to cancel leasing")
	r.Status = LeaseCanceled
	r.CancelHeight = 1
	r.CancelTransactionID = &txID
	to.stor.flush(t)
	resLeasing, err := to.leases.leasingInfo(leaseID)
	assert.NoError(t, err, "failed to get leasing info")
	assert.Equal(t, resLeasing, r, "invalid leasing record after cancellation")
}
