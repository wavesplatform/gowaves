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

func flushLeases(t *testing.T, leases *leases) {
	if err := leases.flush(); err != nil {
		t.Fatalf("flush(): %v\n", err)
	}
	leases.reset()
	if err := leases.db.Flush(leases.dbBatch); err != nil {
		t.Fatalf("db.Flush(): %v\n", err)
	}
}

func createLeases() (*leases, []string, error) {
	res := make([]string, 1)
	dbDir0, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	if err != nil {
		return nil, res, err
	}
	db, err := keyvalue.NewKeyVal(dbDir0)
	if err != nil {
		return nil, res, err
	}
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, res, err
	}
	leases, err := newLeases(db, dbBatch, &mock{}, &mock{})
	if err != nil {
		return nil, res, err
	}
	res = []string{dbDir0}
	return leases, res, nil
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
	leases, path, err := createLeases()
	assert.NoError(t, err, "createLeases() failed")

	defer func() {
		err = leases.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	blockID, err := crypto.NewSignatureFromBytes(bytes.Repeat([]byte{0xff}, crypto.SignatureSize))
	assert.NoError(t, err, "failed to create signature from bytes")
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
		r := createLeasingRecord(t, blockID, l.sender)
		err = leases.addLeasing(leaseID, r)
		assert.NoError(t, err, "failed to add leasing")
	}
	flushLeases(t, leases)
	// Cancel one lease by sender and check.
	badSenderStr := leasings[0].sender
	badSender, err := proto.NewAddressFromString(badSenderStr)
	assert.NoError(t, err, "failed to create address from string")
	sendersToCancel := make(map[proto.Address]struct{})
	var empty struct{}
	sendersToCancel[badSender] = empty
	err = leases.cancelLeases(sendersToCancel)
	assert.NoError(t, err, "cancelLeases() failed")
	flushLeases(t, leases)
	for _, l := range leasings {
		leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{l.leaseIDByte}, crypto.DigestSize))
		assert.NoError(t, err, "failed to create digest from bytes")
		leasing, err := leases.leasingInfo(leaseID)
		assert.NoError(t, err, "failed to get leasing")
		if l.sender == badSenderStr {
			assert.Equal(t, leasing.isActive, false, "did not cancel leasing by sender")
		} else {
			assert.Equal(t, leasing.isActive, true, "cancelled leasing with different sender")
		}
	}
	// Cancel all the leases and check.
	err = leases.cancelLeases(nil)
	assert.NoError(t, err, "cancelLeases() failed")
	flushLeases(t, leases)
	for _, l := range leasings {
		leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{l.leaseIDByte}, crypto.DigestSize))
		assert.NoError(t, err, "failed to create digest from bytes")
		leasing, err := leases.leasingInfo(leaseID)
		assert.NoError(t, err, "failed to get leasing")
		assert.Equal(t, leasing.isActive, false, "did not cancel all the leasings")
	}
}

func TestValidLeaseIns(t *testing.T) {
	leases, path, err := createLeases()
	assert.NoError(t, err, "createLeases() failed")

	defer func() {
		err = leases.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	blockID, err := crypto.NewSignatureFromBytes(bytes.Repeat([]byte{0xff}, crypto.SignatureSize))
	assert.NoError(t, err, "failed to create signature from bytes")
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
		r := createLeasingRecord(t, blockID, l.sender)
		err = leases.addLeasing(leaseID, r)
		assert.NoError(t, err, "failed to add leasing")
		properLeaseIns[r.recipient] = int64(r.leaseAmount)
	}
	flushLeases(t, leases)
	leaseIns, err := leases.validLeaseIns()
	assert.NoError(t, err, "validLeaseIns() failed")
	assert.Equal(t, len(properLeaseIns), len(leaseIns))
	for k, v := range properLeaseIns {
		v1 := leaseIns[k]
		assert.Equal(t, v, v1)
	}
}

func TestAddLeasing(t *testing.T) {
	leases, path, err := createLeases()
	assert.NoError(t, err, "createLeases() failed")

	defer func() {
		err = leases.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	blockID, err := crypto.NewSignatureFromBytes(bytes.Repeat([]byte{0xff}, crypto.SignatureSize))
	assert.NoError(t, err, "failed to create signature from bytes")
	leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	senderStr := "3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp"
	r := createLeasingRecord(t, blockID, senderStr)
	err = leases.addLeasing(leaseID, r)
	assert.NoError(t, err, "failed to add leasing")
	l, err := leases.newestLeasingInfo(leaseID)
	assert.NoError(t, err, "failed to get newest leasing info")
	assert.Equal(t, *l, r.leasing, "leasings differ before flushing")
	flushLeases(t, leases)
	resLeasing, err := leases.leasingInfo(leaseID)
	assert.NoError(t, err, "failed to get leasing info")
	assert.Equal(t, *resLeasing, r.leasing, "leasings differ after flushing")
}

func TestCancelLeasing(t *testing.T) {
	leases, path, err := createLeases()
	assert.NoError(t, err, "createLeases() failed")

	defer func() {
		err = leases.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	blockID, err := crypto.NewSignatureFromBytes(bytes.Repeat([]byte{0xff}, crypto.SignatureSize))
	assert.NoError(t, err, "failed to create signature from bytes")
	leaseID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	senderStr := "3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp"
	r := createLeasingRecord(t, blockID, senderStr)
	err = leases.addLeasing(leaseID, r)
	assert.NoError(t, err, "failed to add leasing")
	err = leases.cancelLeasing(leaseID, blockID)
	assert.NoError(t, err, "failed to cancel leasing")
	r.isActive = false
	flushLeases(t, leases)
	resLeasing, err := leases.leasingInfo(leaseID)
	assert.NoError(t, err, "failed to get leasing info")
	assert.Equal(t, *resLeasing, r.leasing, "invalid leasing record after cancelation")
}
