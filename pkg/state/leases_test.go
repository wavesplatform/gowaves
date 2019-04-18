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

func createLeasingRecord(t *testing.T, leaseID crypto.Digest, blockID crypto.Signature) *leasingRecord {
	addr0, err := proto.NewAddressFromString("3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo")
	assert.NoError(t, err, "failed to create address from string")
	addr1, err := proto.NewAddressFromString("3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp")
	assert.NoError(t, err, "failed to create address from string")
	r := &leasingRecord{
		leasing: leasing{
			isActive:  true,
			leaseIn:   1,
			leaseOut:  10,
			recipient: addr0,
			sender:    addr1,
		},
		blockID: blockID,
	}
	return r
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
	r := createLeasingRecord(t, leaseID, blockID)
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
	r := createLeasingRecord(t, leaseID, blockID)
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
