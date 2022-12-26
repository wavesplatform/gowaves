package trivialdupchecker

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// DuplicateChecker keeps last hash of network message.
type DuplicateChecker struct {
	last crypto.Digest
	lock sync.Mutex
}

// NewDuplicateChecker creates new instance of DuplicateChecker.
func NewDuplicateChecker() *DuplicateChecker {
	return &DuplicateChecker{}
}

// Add compares new bytes with previous, if equal message is now new.
func (a *DuplicateChecker) Add(peerID string, message []byte) (isNew bool) {
	idBytes := []byte(peerID)
	data := make([]byte, len(idBytes)+len(message))
	data = append(data, idBytes...)
	data = append(data, message...)
	digest := crypto.MustFastHash(data)
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.last == digest {
		return false
	}
	a.last = digest
	return true
}
