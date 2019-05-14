package state

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type leasesTestObjects struct {
	rb      *recentBlocks
	leases  *leases
	stateDB *stateDB
}

func flushLeases(t *testing.T, to *leasesTestObjects) {
	to.rb.flush()
	err := to.leases.flush(false)
	assert.NoError(t, err, "leases.flush() failed")
	to.leases.reset()
	err = to.stateDB.flush()
	assert.NoError(t, err, "stateDB.flush() failed")
	to.stateDB.reset()
}

func createLeases() (*leasesTestObjects, []string, error) {
	dbDir0, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	if err != nil {
		return nil, nil, err
	}
	res := []string{dbDir0}
	db, err := keyvalue.NewKeyVal(dbDir0, defaultTestBloomFilterParams())
	if err != nil {
		return nil, res, err
	}
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, res, err
	}
	stateDB, err := newStateDB(db, dbBatch)
	if err != nil {
		return nil, res, err
	}
	rb, err := newRecentBlocks(rollbackMaxBlocks, nil)
	if err != nil {
		return nil, res, err
	}
	leases, err := newLeases(db, dbBatch, stateDB, rb)
	if err != nil {
		return nil, res, err
	}
	return &leasesTestObjects{rb, leases, stateDB}, res, nil
}

func createLeasingRecord(t *testing.T, blockID crypto.Signature, sender string) *leasingRecord {
	senderAddr, err := proto.NewAddressFromString(sender)
	assert.NoError(t, err, "failed to create address from string")
	recipientAddr, err := proto.NewAddressFromString("3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo")
	assert.NoError(t, err, "failed to create address from string")
	r := &leasingRecord{
		leasing: leasing{
			isActive:    true,
			leaseAmount: 10,
			recipient:   recipientAddr,
			sender:      senderAddr,
		},
		blockID: blockID,
	}
	return r
}

func TestCancelLeases(t *testing.T) {
	to, path, err := createLeases()
	assert.NoError(t, err, "createLeases() failed")

	defer func() {
		err = to.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	addBlock(t, to.stateDB, to.rb, blockID0)
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
		r := createLeasingRecord(t, blockID0, l.sender)
		err = to.leases.addLeasing(leaseID, r)
		assert.NoError(t, err, "failed to add leasing")
	}
	flushLeases(t, to)
	// Cancel one lease by sender and check.
	badSenderStr := leasings[0].sender
	badSender, err := proto.NewAddressFromString(badSenderStr)
	assert.NoError(t, err, "failed to create address from string")
	sendersToCancel := make(map[proto.Address]struct{})
	var empty struct{}
	sendersToCancel[badSender] = empty
	err = to.leases.cancelLeases(sendersToCancel)
	assert.NoError(t, err, "cancelLeases() failed")
	flushLeases(t, to)
	for _, l := range leasings {
		leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{l.leaseIDByte}, crypto.DigestSize))
		assert.NoError(t, err, "failed to create digest from bytes")
		leasing, err := to.leases.leasingInfo(leaseID)
		assert.NoError(t, err, "failed to get leasing")
		if l.sender == badSenderStr {
			assert.Equal(t, leasing.isActive, false, "did not cancel leasing by sender")
		} else {
			assert.Equal(t, leasing.isActive, true, "cancelled leasing with different sender")
		}
	}
	// Cancel all the leases and check.
	err = to.leases.cancelLeases(nil)
	assert.NoError(t, err, "cancelLeases() failed")
	flushLeases(t, to)
	for _, l := range leasings {
		leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{l.leaseIDByte}, crypto.DigestSize))
		assert.NoError(t, err, "failed to create digest from bytes")
		leasing, err := to.leases.leasingInfo(leaseID)
		assert.NoError(t, err, "failed to get leasing")
		assert.Equal(t, leasing.isActive, false, "did not cancel all the leasings")
	}
}

func TestValidLeaseIns(t *testing.T) {
	to, path, err := createLeases()
	assert.NoError(t, err, "createLeases() failed")

	defer func() {
		err = to.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	addBlock(t, to.stateDB, to.rb, blockID0)
	leasings := []struct {
		sender      string
		leaseIDByte byte
	}{
		{"3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp", 0xff},
		{"3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo", 0xaa},
	}
	properLeaseIns := make(map[proto.Address]int64)
	for _, l := range leasings {
		leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{l.leaseIDByte}, crypto.DigestSize))
		assert.NoError(t, err, "failed to create digest from bytes")
		r := createLeasingRecord(t, blockID0, l.sender)
		err = to.leases.addLeasing(leaseID, r)
		assert.NoError(t, err, "failed to add leasing")
		properLeaseIns[r.recipient] = int64(r.leaseAmount)
	}
	flushLeases(t, to)
	leaseIns, err := to.leases.validLeaseIns()
	assert.NoError(t, err, "validLeaseIns() failed")
	assert.Equal(t, len(properLeaseIns), len(leaseIns))
	for k, v := range properLeaseIns {
		v1 := leaseIns[k]
		assert.Equal(t, v, v1)
	}
}

func TestAddLeasing(t *testing.T) {
	to, path, err := createLeases()
	assert.NoError(t, err, "createLeases() failed")

	defer func() {
		err = to.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	addBlock(t, to.stateDB, to.rb, blockID0)
	leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	senderStr := "3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp"
	r := createLeasingRecord(t, blockID0, senderStr)
	err = to.leases.addLeasing(leaseID, r)
	assert.NoError(t, err, "failed to add leasing")
	l, err := to.leases.newestLeasingInfo(leaseID, true)
	assert.NoError(t, err, "failed to get newest leasing info")
	assert.Equal(t, *l, r.leasing, "leasings differ before flushing")
	flushLeases(t, to)
	resLeasing, err := to.leases.leasingInfo(leaseID)
	assert.NoError(t, err, "failed to get leasing info")
	assert.Equal(t, *resLeasing, r.leasing, "leasings differ after flushing")
}

func TestCancelLeasing(t *testing.T) {
	to, path, err := createLeases()
	assert.NoError(t, err, "createLeases() failed")

	defer func() {
		err = to.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	addBlock(t, to.stateDB, to.rb, blockID0)
	leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	senderStr := "3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp"
	r := createLeasingRecord(t, blockID0, senderStr)
	err = to.leases.addLeasing(leaseID, r)
	assert.NoError(t, err, "failed to add leasing")
	err = to.leases.cancelLeasing(leaseID, blockID0, true)
	assert.NoError(t, err, "failed to cancel leasing")
	r.isActive = false
	flushLeases(t, to)
	resLeasing, err := to.leases.leasingInfo(leaseID)
	assert.NoError(t, err, "failed to get leasing info")
	assert.Equal(t, *resLeasing, r.leasing, "invalid leasing record after cancelation")
}
