package storage

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)
import "github.com/fxamacker/cbor/v2"

//var cborEncoder cbor.EncMode
//
//func init() {
//	// nickeskov: cbor encoder initialization
//	opts := cbor.CanonicalEncOptions()
//
//	em, err := opts.EncMode()
//	if err != nil {
//		panic(errors.Wrap(err, "BUG, CREATE REPORT: failed to create CBOR encoder"))
//	}
//	cborEncoder = em
//}

func TestBinaryStorageCbor(t *testing.T) {
	info := map[int]SuspendedInfo{
		2344: {
			IP:                     IPFromString("13.3.4.1"),
			SuspendTimestampMillis: time.Now().UnixNano() / 1_000_000,
			SuspendDuration:        time.Minute * 5,
			Reason:                 "some reason",
		},
	}

	data, err := cbor.Marshal(info)
	assert.NoError(t, err)

	unmarhalled := make(map[int]SuspendedInfo)
	err = cbor.Unmarshal(data, &unmarhalled)
	assert.NoError(t, err)

	assert.Equal(t, info, unmarhalled)

	//assert.Equal(t, info.IP, unmarhalled.IP)
	//assert.Equal(t, info.SuspendTimestampMillis, unmarhalled.SuspendTimestampMillis)
	//assert.Equal(t, info.SuspendDuration, unmarhalled.SuspendDuration)
	//assert.Equal(t, info.Reason, unmarhalled.Reason)
}

func TestFromUnixMillis(t *testing.T) {
	ts := time.Now().Truncate(time.Millisecond)
	tsMillis := ts.UnixNano() / 1_000_000

	assert.Equal(t, ts.String(), fromUnixMillis(tsMillis).String())
}
