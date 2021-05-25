package storage

import (
	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"io/ioutil"
	"net"
	"path/filepath"
	"sync"
	"time"

	//"github.com/fxamacker/cbor/v2"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	peersStorageDir = "peers_storage"
)

type IP [net.IPv6len]byte

func (i *IP) String() string {
	return net.IP(i[:]).String()
}

func IPFromString(s string) IP {
	parsed := net.ParseIP(s)
	ip := IP{}
	copy(ip[:], parsed[:net.IPv6len])
	return ip
}

func IpFromIpPort(ipPort proto.IpPort) IP {
	ip := IP{}
	copy(ip[:], ipPort[:net.IPv6len])
	return ip
}

func fromUnixMillis(timestampMillis int64) time.Time {
	sec := timestampMillis / 1_000
	nsec := (timestampMillis % 1_000) * 1_000_000
	return time.Unix(sec, nsec)
}

type SuspendedInfo struct {
	IP                     IP            `cbor:"0,keyasint,omitemtpy"`
	SuspendTimestampMillis int64         `cbor:"1,keyasint,omitemtpy"`
	SuspendDuration        time.Duration `cbor:"2,keyasint,omitemtpy"`
	Reason                 string        `cbor:"3,keyasint,omitemtpy"`
}

func (si *SuspendedInfo) SuspendTime() time.Time {
	return fromUnixMillis(si.SuspendTimestampMillis)
}

func (si *SuspendedInfo) AwakeTime() time.Time {
	return si.SuspendTime().Add(si.SuspendDuration)
}

type BinaryStorageCbor struct {
	mutex     sync.Mutex
	suspended map[IP]SuspendedInfo // nickeskov: suspended peers
	known     Peers                // nickeskov: list of all ever known peers with a publicly available declared address
	//alreadyKnown map[]struct{}
}

func (bs *BinaryStorageCbor) AddSuspended(suspended ...SuspendedInfo) error {
	if len(suspended) == 0 {
		return nil
	}

	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	// TODO(nickeskov): finish it
	return nil
}

func (bs *BinaryStorageCbor) unsafeSyncSuspended() error {
	data, err := cbor.Marshal(bs.suspended)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal suspended peers (type = %T) to CBOR", bs.suspended)
	}

	filePath := suspendedFilePath()
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return errors.Wrapf(err, "failed to write suspended peers in file %q", filePath)
	}
	return nil
}

func (bs *BinaryStorageCbor) unsafeSyncKnown() error {
	data, err := cbor.Marshal(bs.known)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal known peers (type = %T) to CBOR", bs.known)
	}

	filePath := knownFilePath()
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return errors.Wrapf(err, "failed to write known peers in file %q", filePath)
	}
	return nil
}

func knownFilePath() string {
	return filepath.Join(peersStorageDir, "peers_known.cbor")
}

func suspendedFilePath() string {
	return filepath.Join(peersStorageDir, "peers_suspended.cbor")
}
